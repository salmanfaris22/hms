// Command seedusers provisions a tenant and fills it with a known set of
// demo accounts — one platform super admin, one tenant owner, and a spread of
// role-specific staff logins — then prints the credentials.
//
// It is idempotent: every account is upserted, so re-running it resets the
// demo passwords rather than failing on duplicates. Intended for development
// and demo environments only; the passwords here are deliberately public.
//
//	go run ./cmd/seedusers                  # seed the "demo" tenant
//	go run ./cmd/seedusers -tenant acme     # seed a different tenant
//	go run ./cmd/seedusers -password 'X1!'  # override the shared password
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/salman/hms-backend/internal/core/config"
	"github.com/salman/hms-backend/internal/core/utils"
	"github.com/salman/hms-backend/internal/infrastructure/persistence/postgres"
)

// account is one demo login to create in the tenant database.
type account struct {
	Email      string
	FullName   string
	Role       string // users.role and clinic_members.role
	SuperAdmin bool   // tenant-level owner, not the platform super admin
}

// demoAccounts is the roster seeded into every tenant this command touches.
// The first entry is the tenant owner; the rest exercise the role-based
// screens (staff list, appointment assignment, lab queue, billing).
var demoAccounts = []account{
	{Email: "admin@%s.clinic", FullName: "Demo Admin", Role: "admin", SuperAdmin: true},
	{Email: "doctor@%s.clinic", FullName: "Dr. Aisha Rahman", Role: "doctor"},
	{Email: "nurse@%s.clinic", FullName: "Nurse Joseph Mathew", Role: "nurse"},
	{Email: "reception@%s.clinic", FullName: "Reception Desk", Role: "receptionist"},
	{Email: "lab@%s.clinic", FullName: "Lab Technician", Role: "lab"},
	{Email: "pharmacy@%s.clinic", FullName: "Pharmacy Counter", Role: "pharmacist"},
	{Email: "accounts@%s.clinic", FullName: "Accounts Desk", Role: "accountant"},
	{Email: "staff@%s.clinic", FullName: "General Staff", Role: "staff"},
}

func main() {
	var (
		tenantSlug = flag.String("tenant", "demo", "tenant slug to seed")
		tenantName = flag.String("tenant-name", "", "display name for the tenant (default: derived from slug)")
		password   = flag.String("password", "Passw0rd!", "shared password for every seeded tenant account")
		timeout    = flag.Duration("timeout", 90*time.Second, "overall timeout")
	)
	flag.Parse()

	if *tenantName == "" {
		*tenantName = titleCase(*tenantSlug) + " Hospital"
	}

	cfg := config.Load()
	dbURL := cfg.DatabaseURL
	if cfg.PgBouncerURL != "" {
		dbURL = cfg.PgBouncerURL
	}

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	resolver, registry, err := postgres.NewTenantResolver(ctx, dbURL)
	if err != nil {
		log.Fatalf("db bootstrap: %v", err)
	}
	defer resolver.Close()

	// 1. Platform super admin — lives in the registry, logs into the
	//    superadmin app rather than a tenant.
	if err := upsertPlatformSuperAdmin(ctx, registry, cfg.SuperAdminEmail, cfg.SuperAdminPassword); err != nil {
		log.Fatalf("super admin: %v", err)
	}

	// 2. Tenant. Provision is idempotent and applies tenant migrations.
	info, pool, err := resolver.Provision(ctx, *tenantSlug, *tenantName)
	if err != nil {
		log.Fatalf("provision tenant %q: %v", *tenantSlug, err)
	}
	if err := resolver.MigrateAllTenants(ctx); err != nil {
		log.Fatalf("migrate tenants: %v", err)
	}

	// 3. A clinic to attach members to — the staff screens are all
	//    clinic-scoped, so an account with no membership sees nothing.
	clinicIDs, err := ensureClinic(ctx, pool, *tenantName)
	if err != nil {
		log.Fatalf("ensure clinic: %v", err)
	}

	// 4. The accounts themselves.
	hash, err := utils.HashPassword(*password)
	if err != nil {
		log.Fatalf("hash password: %v", err)
	}

	seeded := make([]account, 0, len(demoAccounts))
	for _, a := range demoAccounts {
		a.Email = fmt.Sprintf(a.Email, *tenantSlug)
		if err := upsertTenantUser(ctx, resolver, pool, info.ID, a, hash, clinicIDs); err != nil {
			log.Fatalf("seed %s: %v", a.Email, err)
		}
		seeded = append(seeded, a)
	}

	report(os.Stdout, *tenantSlug, cfg, *password, seeded, len(clinicIDs))
}

// titleCase upper-cases the first byte — slugs are ASCII, so this is enough
// and avoids the deprecated strings.Title.
func titleCase(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// upsertPlatformSuperAdmin writes the registry-level super admin, resetting the
// password so the printed credentials are always the working ones.
func upsertPlatformSuperAdmin(ctx context.Context, registry *pgxpool.Pool, email, password string) error {
	hash, err := utils.HashPassword(password)
	if err != nil {
		return err
	}
	_, err = registry.Exec(ctx, `
		INSERT INTO super_admins (email, password_hash, full_name)
		VALUES ($1, $2, 'Platform Super Admin')
		ON CONFLICT (email) DO UPDATE
		   SET password_hash = EXCLUDED.password_hash,
		       updated_at    = now()`,
		email, hash,
	)
	return err
}

// ensureClinic returns the tenant's clinic ids, creating a default clinic when
// the tenant is brand new.
func ensureClinic(ctx context.Context, pool *pgxpool.Pool, tenantName string) ([]string, error) {
	rows, err := pool.Query(ctx, `SELECT id::text FROM clinics ORDER BY created_at LIMIT 20`)
	if err != nil {
		return nil, err
	}
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return nil, err
		}
		ids = append(ids, id)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(ids) > 0 {
		return ids, nil
	}

	code, err := utils.RandomInviteCode(6)
	if err != nil {
		return nil, err
	}
	var id string
	if err := pool.QueryRow(ctx, `
		INSERT INTO clinics (name, clinic_type, location, color, invite_code)
		VALUES ($1, 'Multi-Specialty Clinic', 'Main Campus', 'purple', $2)
		RETURNING id::text`,
		tenantName+" — Main Clinic", code,
	).Scan(&id); err != nil {
		return nil, err
	}
	return []string{id}, nil
}

// upsertTenantUser creates or resets one tenant login, registers it in the
// cross-tenant email directory, and joins it to every clinic.
func upsertTenantUser(
	ctx context.Context,
	resolver *postgres.TenantResolver,
	pool *pgxpool.Pool,
	tenantID string,
	a account,
	passwordHash string,
	clinicIDs []string,
) error {
	var userID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO users (email, password_hash, full_name, role, is_super_admin, is_active)
		VALUES ($1, $2, $3, $4, $5, true)
		ON CONFLICT (email) DO UPDATE
		   SET password_hash  = EXCLUDED.password_hash,
		       full_name      = EXCLUDED.full_name,
		       role           = EXCLUDED.role,
		       is_super_admin = EXCLUDED.is_super_admin,
		       is_active      = true,
		       updated_at     = now()
		RETURNING id::text`,
		a.Email, passwordHash, a.FullName, a.Role, a.SuperAdmin,
	).Scan(&userID); err != nil {
		return fmt.Errorf("upsert user: %w", err)
	}

	// The directory is keyed by email across all tenants, so a clash means the
	// address already belongs to another hospital — surface it rather than
	// silently seeding an account nobody can log into.
	var ownerTenant string
	if err := resolver.Registry().QueryRow(ctx, `
		INSERT INTO user_directory (email, tenant_id, user_id)
		VALUES ($1, $2::uuid, $3::uuid)
		ON CONFLICT (email) DO UPDATE
		   SET tenant_id = EXCLUDED.tenant_id,
		       user_id   = EXCLUDED.user_id
		 WHERE user_directory.tenant_id = EXCLUDED.tenant_id
		RETURNING tenant_id::text`,
		a.Email, tenantID, userID,
	).Scan(&ownerTenant); err != nil {
		return fmt.Errorf("%s is already registered to a different tenant", a.Email)
	}

	for i, cID := range clinicIDs {
		if _, err := pool.Exec(ctx, `
			INSERT INTO clinic_members (clinic_id, user_id, role, is_default)
			VALUES ($1::uuid, $2::uuid, $3, $4)
			ON CONFLICT (clinic_id, user_id) DO UPDATE
			   SET role = EXCLUDED.role`,
			cID, userID, a.Role, i == 0,
		); err != nil {
			return fmt.Errorf("join clinic: %w", err)
		}
	}
	return nil
}

func report(w *os.File, slug string, cfg config.Config, password string, accounts []account, clinics int) {
	p := func(format string, args ...any) { fmt.Fprintf(w, format+"\n", args...) }

	p("")
	p("=== HMS demo accounts seeded ===")
	p("")
	p("Tenant: %s   (%d clinic(s))", slug, clinics)
	p("")
	p("Platform super admin — superadmin app (%s):", cfg.AppURL)
	p("  %-28s  %s", cfg.SuperAdminEmail, cfg.SuperAdminPassword)
	p("")
	p("Tenant logins — main app. Shared password: %s", password)
	p("")
	p("  %-28s  %-22s  %s", "EMAIL", "NAME", "ROLE")
	p("  %-28s  %-22s  %s", strings.Repeat("-", 28), strings.Repeat("-", 22), strings.Repeat("-", 12))
	for _, a := range accounts {
		role := a.Role
		if a.SuperAdmin {
			role += " (owner)"
		}
		p("  %-28s  %-22s  %s", a.Email, a.FullName, role)
	}
	p("")
	p("Re-run this command any time to reset these passwords.")
	p("")
}
