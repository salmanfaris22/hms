// Command seedusers provisions a tenant and fills it with a known set of demo
// accounts — one platform super admin, one tenant owner, and a spread of
// role-specific staff logins — then prints the credentials.
//
// The server seeds the same roster on startup (see cmd/server). The difference
// is that this command also RESETS the password of accounts that already
// exist, so the credentials it prints are guaranteed to work. Intended for
// development and demo environments; the passwords are deliberately public.
//
//	go run ./cmd/seedusers                  # seed the "demo" tenant
//	go run ./cmd/seedusers -tenant acme     # seed a different tenant
//	go run ./cmd/seedusers -password 'X1!'  # override the shared password
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	"github.com/salman/hms-backend/internal/core/config"
	"github.com/salman/hms-backend/internal/core/seeder"
	"github.com/salman/hms-backend/internal/infrastructure/persistence/postgres"
)

func main() {
	var (
		tenantSlug = flag.String("tenant", "demo", "tenant slug to seed")
		tenantName = flag.String("tenant-name", "", "display name for the tenant (default: derived from slug)")
		password   = flag.String("password", seeder.DemoPassword, "shared password for every seeded tenant account")
		timeout    = flag.Duration("timeout", 90*time.Second, "overall timeout")
	)
	flag.Parse()

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

	// Platform super admin lives in the registry and logs into the superadmin
	// app rather than a tenant.
	if err := seeder.UpsertPlatformSuperAdmin(ctx, registry, cfg.SuperAdminEmail, cfg.SuperAdminPassword); err != nil {
		log.Fatalf("super admin: %v", err)
	}

	report, err := seeder.EnsureDemoAccounts(ctx, resolver, registry, seeder.AccountOptions{
		TenantSlug:     *tenantSlug,
		TenantName:     *tenantName,
		Password:       *password,
		ResetPasswords: true,
	})
	if err != nil {
		log.Fatalf("seed accounts: %v", err)
	}

	printReport(os.Stdout, cfg, report)
}

func printReport(w io.Writer, cfg config.Config, r seeder.AccountReport) {
	p := func(format string, args ...any) { fmt.Fprintf(w, format+"\n", args...) }

	p("")
	p("=== HMS demo accounts seeded ===")
	p("")
	p("Tenant: %s   (%d clinic(s))", r.TenantSlug, r.Clinics)
	p("")
	p("Platform super admin — superadmin app (%s):", cfg.AppURL)
	p("  %-28s  %s", cfg.SuperAdminEmail, cfg.SuperAdminPassword)
	p("")
	p("Tenant logins — main app. Shared password: %s", r.Password)
	p("")
	p("  %-28s  %-22s  %s", "EMAIL", "NAME", "ROLE")
	p("  %-28s  %-22s  %s", strings.Repeat("-", 28), strings.Repeat("-", 22), strings.Repeat("-", 12))
	for _, a := range r.Accounts {
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
