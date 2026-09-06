package seeder

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/salman/hms-backend/internal/core/utils"
	"github.com/salman/hms-backend/internal/infrastructure/persistence/postgres"
)

// DemoPassword is the shared password given to every seeded demo login.
const DemoPassword = "Passw0rd!"

// Account is one demo login seeded into a tenant.
type Account struct {
	Email      string
	FullName   string
	Role       string // users.role and clinic_members.role
	SuperAdmin bool   // tenant owner, not the platform super admin
}

// demoAccounts is the roster seeded into every tenant. The first entry is the
// tenant owner; the rest exercise the role-based screens (staff list,
// appointment assignment, lab queue, billing). %s is the tenant slug.
var demoAccounts = []Account{
	{Email: "admin@%s.clinic", FullName: "Demo Admin", Role: "admin", SuperAdmin: true},
	{Email: "doctor@%s.clinic", FullName: "Dr. Aisha Rahman", Role: "doctor"},
	{Email: "nurse@%s.clinic", FullName: "Nurse Joseph Mathew", Role: "nurse"},
	{Email: "reception@%s.clinic", FullName: "Reception Desk", Role: "receptionist"},
	{Email: "lab@%s.clinic", FullName: "Lab Technician", Role: "lab"},
	{Email: "pharmacy@%s.clinic", FullName: "Pharmacy Counter", Role: "pharmacist"},
	{Email: "accounts@%s.clinic", FullName: "Accounts Desk", Role: "accountant"},
	{Email: "staff@%s.clinic", FullName: "General Staff", Role: "staff"},
}

// AccountOptions controls one EnsureDemoAccounts run.
type AccountOptions struct {
	TenantSlug string // defaults to "demo"
	TenantName string // defaults to "<Slug> Hospital"
	Password   string // defaults to DemoPassword

	// ResetPasswords rewrites the password of accounts that already exist.
	// The standalone seedusers command sets this so the credentials it prints
	// are guaranteed to work. Server startup leaves it false, so restarting
	// the server never clobbers a password somebody changed in the app.
	ResetPasswords bool

	// SkipMigrate suppresses the MigrateAllTenants call. applyMigrations has
	// no ledger — it re-executes every .sql file each time — so the server,
	// which already migrates during startup, sets this to avoid paying for a
	// second full pass on every boot.
	SkipMigrate bool
}

func (o *AccountOptions) applyDefaults() {
	if o.TenantSlug == "" {
		o.TenantSlug = "demo"
	}
	if o.TenantName == "" {
		o.TenantName = titleCase(o.TenantSlug) + " Hospital"
	}
	if o.Password == "" {
		o.Password = DemoPassword
	}
}

// titleCase upper-cases the first byte — slugs are ASCII, so this is enough
// and avoids the deprecated strings.Title.
func titleCase(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// AccountReport describes what a seeding run produced, for printing.
type AccountReport struct {
	TenantSlug string
	Password   string
	Clinics    int
	Accounts   []Account
}

// EnsureDemoAccounts provisions the tenant if missing and seeds the demo
// roster into it. Idempotent — safe to call on every server start.
//
// Note the pool lifetime: MigrateAllTenants closes and replaces the cached
// per-tenant pools, so callers must not reuse a pool taken before it ran. This
// function re-fetches its own.
func EnsureDemoAccounts(
	ctx context.Context,
	resolver *postgres.TenantResolver,
	registry *pgxpool.Pool,
	opts AccountOptions,
) (AccountReport, error) {
	opts.applyDefaults()

	info, _, err := resolver.Provision(ctx, opts.TenantSlug, opts.TenantName)
	if err != nil {
		return AccountReport{}, fmt.Errorf("provision tenant %q: %w", opts.TenantSlug, err)
	}
	if !opts.SkipMigrate {
		if err := resolver.MigrateAllTenants(ctx); err != nil {
			return AccountReport{}, fmt.Errorf("migrate tenants: %w", err)
		}
	}
	pool, err := resolver.Pool(ctx, info.ID)
	if err != nil {
		return AccountReport{}, fmt.Errorf("tenant pool: %w", err)
	}

	// The staff screens are all clinic-scoped, so an account with no
	// membership would log in and see nothing.
	clinicIDs, err := ensureClinic(ctx, pool, opts.TenantName)
	if err != nil {
		return AccountReport{}, fmt.Errorf("ensure clinic: %w", err)
	}

	hash, err := utils.HashPassword(opts.Password)
	if err != nil {
		return AccountReport{}, err
	}

	report := AccountReport{
		TenantSlug: opts.TenantSlug,
		Password:   opts.Password,
		Clinics:    len(clinicIDs),
		Accounts:   make([]Account, 0, len(demoAccounts)),
	}
	for _, a := range demoAccounts {
		a.Email = fmt.Sprintf(a.Email, opts.TenantSlug)
		if err := upsertTenantUser(ctx, resolver, pool, info.ID, a, hash, clinicIDs, opts.ResetPasswords); err != nil {
			return AccountReport{}, fmt.Errorf("seed %s: %w", a.Email, err)
		}
		report.Accounts = append(report.Accounts, a)
	}
	return report, nil
}

// UpsertPlatformSuperAdmin writes the registry-level super admin, resetting the
// password so printed credentials are always the working ones. Unlike
// EnsureSuperAdmin it does not skip when the table is already populated.
func UpsertPlatformSuperAdmin(ctx context.Context, registry *pgxpool.Pool, email, password string) error {
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

// upsertTenantUser creates one tenant login, registers it in the cross-tenant
// email directory, and joins it to every clinic. When resetPassword is false an
// account that already exists keeps its current password and name.
func upsertTenantUser(
	ctx context.Context,
	resolver *postgres.TenantResolver,
	pool *pgxpool.Pool,
	tenantID string,
	a Account,
	passwordHash string,
	clinicIDs []string,
	resetPassword bool,
) error {
	// DO UPDATE rather than DO NOTHING even in the non-reset case: the row has
	// to come back so we can wire up membership, and DO NOTHING returns none.
	const resetQ = `
		INSERT INTO users (email, password_hash, full_name, role, is_super_admin, is_active)
		VALUES ($1, $2, $3, $4, $5, true)
		ON CONFLICT (email) DO UPDATE
		   SET password_hash  = EXCLUDED.password_hash,
		       full_name      = EXCLUDED.full_name,
		       role           = EXCLUDED.role,
		       is_super_admin = EXCLUDED.is_super_admin,
		       is_active      = true,
		       updated_at     = now()
		RETURNING id::text`
	const keepQ = `
		INSERT INTO users (email, password_hash, full_name, role, is_super_admin, is_active)
		VALUES ($1, $2, $3, $4, $5, true)
		ON CONFLICT (email) DO UPDATE
		   SET email = users.email
		RETURNING id::text`

	q := keepQ
	if resetPassword {
		q = resetQ
	}

	var userID string
	if err := pool.QueryRow(ctx, q,
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
