package postgres

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"net/url"
	"path"
	"sort"
	"strings"
	"sync"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/salman/hms-backend/migrations"
)

// TenantResolver owns the registry pool and a cache of per-tenant pools.
// Tenants live in physically separate databases (`hms_<slug>`); this type
// handles provisioning (create database + run migrations) and lookups.
type TenantResolver struct {
	baseURL  string
	registry *pgxpool.Pool
	mu       sync.Mutex
	cache    map[string]*pgxpool.Pool
}

type TenantInfo struct {
	ID     string
	Slug   string
	Name   string
	DBName string
}

// NewTenantResolver ensures the registry database exists, runs its migrations,
// and returns a resolver plus the registry pool.
func NewTenantResolver(ctx context.Context, baseURL string) (*TenantResolver, *pgxpool.Pool, error) {
	if err := ensureDatabase(ctx, baseURL, "hms_registry"); err != nil {
		return nil, nil, fmt.Errorf("ensure registry db: %w", err)
	}
	regPool, err := pgxpool.New(ctx, replaceDB(baseURL, "hms_registry"))
	if err != nil {
		return nil, nil, fmt.Errorf("connect registry: %w", err)
	}
	if err := regPool.Ping(ctx); err != nil {
		return nil, nil, fmt.Errorf("ping registry: %w", err)
	}
	if err := applyMigrations(ctx, regPool, migrations.RegistryFS, "registry"); err != nil {
		return nil, nil, fmt.Errorf("migrate registry: %w", err)
	}
	return &TenantResolver{
		baseURL:  baseURL,
		registry: regPool,
		cache:    map[string]*pgxpool.Pool{},
	}, regPool, nil
}

func (r *TenantResolver) Registry() *pgxpool.Pool { return r.registry }

// Provision is idempotent: if a tenant with the given slug already exists it
// returns the existing info and a ready-to-use pool; otherwise it creates the
// database, runs tenant migrations, records the tenant, and caches the pool.
func (r *TenantResolver) Provision(ctx context.Context, slug, name string) (TenantInfo, *pgxpool.Pool, error) {
	slug = strings.ToLower(strings.TrimSpace(slug))
	if slug == "" {
		return TenantInfo{}, nil, errors.New("slug required")
	}

	var info TenantInfo
	err := r.registry.QueryRow(ctx,
		`SELECT id::text, slug, name, db_name FROM tenants WHERE slug = $1`, slug,
	).Scan(&info.ID, &info.Slug, &info.Name, &info.DBName)
	if err == nil {
		pool, pErr := r.Pool(ctx, info.ID)
		return info, pool, pErr
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return TenantInfo{}, nil, err
	}

	dbName := "hms_" + sanitizeSlug(slug)
	if err := ensureDatabase(ctx, r.baseURL, dbName); err != nil {
		return TenantInfo{}, nil, fmt.Errorf("create tenant db: %w", err)
	}
	pool, err := pgxpool.New(ctx, replaceDB(r.baseURL, dbName))
	if err != nil {
		return TenantInfo{}, nil, err
	}
	if err := applyMigrations(ctx, pool, migrations.TenantFS, "tenant"); err != nil {
		pool.Close()
		return TenantInfo{}, nil, fmt.Errorf("migrate tenant db: %w", err)
	}

	err = r.registry.QueryRow(ctx,
		`INSERT INTO tenants (slug, name, db_name) VALUES ($1, $2, $3) RETURNING id::text`,
		slug, name, dbName,
	).Scan(&info.ID)
	if err != nil {
		pool.Close()
		return TenantInfo{}, nil, err
	}
	info.Slug = slug
	info.Name = name
	info.DBName = dbName

	r.mu.Lock()
	r.cache[info.ID] = pool
	r.mu.Unlock()
	return info, pool, nil
}

// MigrateAllTenants iterates over every tenant in the registry and applies the
// embedded tenant migrations to each of their databases. This is how we keep
// existing tenants in sync when schema/tenant/*.sql files are added.
func (r *TenantResolver) MigrateAllTenants(ctx context.Context) error {
	rows, err := r.registry.Query(ctx, `SELECT id::text, db_name FROM tenants`)
	if err != nil {
		return err
	}
	defer rows.Close()

	type t struct{ id, dbName string }
	var list []t
	for rows.Next() {
		var x t
		if err := rows.Scan(&x.id, &x.dbName); err != nil {
			return err
		}
		list = append(list, x)
	}
	for _, x := range list {
		pool, err := pgxpool.New(ctx, replaceDB(r.baseURL, x.dbName))
		if err != nil {
			return fmt.Errorf("connect %s: %w", x.dbName, err)
		}
		if err := applyMigrations(ctx, pool, migrations.TenantFS, "tenant"); err != nil {
			pool.Close()
			return fmt.Errorf("migrate %s: %w", x.dbName, err)
		}
		r.mu.Lock()
		if existing, ok := r.cache[x.id]; ok {
			existing.Close()
		}
		r.cache[x.id] = pool
		r.mu.Unlock()
	}
	return nil
}

// AllTenantIDs returns every tenant ID from the registry.
func (r *TenantResolver) AllTenantIDs(ctx context.Context) ([]string, error) {
	rows, err := r.registry.Query(ctx, `SELECT id::text FROM tenants`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err == nil {
			ids = append(ids, id)
		}
	}
	return ids, nil
}

// Pool returns the per-tenant pool, lazily opening a connection if needed.
func (r *TenantResolver) Pool(ctx context.Context, tenantID string) (*pgxpool.Pool, error) {
	r.mu.Lock()
	if p, ok := r.cache[tenantID]; ok {
		r.mu.Unlock()
		return p, nil
	}
	r.mu.Unlock()

	var dbName string
	err := r.registry.QueryRow(ctx,
		`SELECT db_name FROM tenants WHERE id = $1::uuid`, tenantID,
	).Scan(&dbName)
	if err != nil {
		return nil, err
	}

	pool, err := pgxpool.New(ctx, replaceDB(r.baseURL, dbName))
	if err != nil {
		return nil, err
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	if existing, ok := r.cache[tenantID]; ok {
		pool.Close()
		return existing, nil
	}
	r.cache[tenantID] = pool
	return pool, nil
}

func (r *TenantResolver) Close() {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, p := range r.cache {
		p.Close()
	}
	r.registry.Close()
}

// DeleteTenant drops the tenant's database and removes its registry row.
// Any cached connection pool for that tenant is closed first, then sessions
// on the target DB are terminated so DROP can proceed.
func (r *TenantResolver) DeleteTenant(ctx context.Context, tenantID string) error {
	var dbName string
	err := r.registry.QueryRow(ctx,
		`SELECT db_name FROM tenants WHERE id = $1::uuid`, tenantID,
	).Scan(&dbName)
	if err != nil {
		return err
	}

	r.mu.Lock()
	if p, ok := r.cache[tenantID]; ok {
		p.Close()
		delete(r.cache, tenantID)
	}
	r.mu.Unlock()

	adminURL := replaceDB(r.baseURL, "postgres")
	conn, err := pgx.Connect(ctx, adminURL)
	if err != nil {
		return fmt.Errorf("connect admin: %w", err)
	}
	defer conn.Close(ctx)

	if _, err := conn.Exec(ctx, `
		SELECT pg_terminate_backend(pid)
		FROM pg_stat_activity
		WHERE datname = $1 AND pid <> pg_backend_pid()`, dbName); err != nil {
		return fmt.Errorf("terminate sessions: %w", err)
	}
	if _, err := conn.Exec(ctx, `DROP DATABASE IF EXISTS "`+dbName+`"`); err != nil {
		return fmt.Errorf("drop database: %w", err)
	}

	if _, err := r.registry.Exec(ctx,
		`DELETE FROM tenants WHERE id = $1::uuid`, tenantID,
	); err != nil {
		return fmt.Errorf("delete tenant row: %w", err)
	}
	return nil
}

// ensureDatabase creates `dbName` on the same server as baseURL if missing.
// CREATE DATABASE can't run inside a transaction or via pgxpool, so we use a
// direct pgx.Conn against the default `postgres` database.
func ensureDatabase(ctx context.Context, baseURL, dbName string) error {
	adminURL := replaceDB(baseURL, "postgres")
	conn, err := pgx.Connect(ctx, adminURL)
	if err != nil {
		return fmt.Errorf("connect admin: %w", err)
	}
	defer conn.Close(ctx)

	var exists bool
	if err := conn.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM pg_database WHERE datname = $1)`, dbName,
	).Scan(&exists); err != nil {
		return err
	}
	if exists {
		return nil
	}
	_, err = conn.Exec(ctx, `CREATE DATABASE "`+dbName+`"`)
	return err
}

func replaceDB(baseURL, db string) string {
	u, err := url.Parse(baseURL)
	if err != nil {
		return baseURL
	}
	u.Path = "/" + db
	return u.String()
}

func sanitizeSlug(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '_' || r == '-':
			b.WriteByte('_')
		}
	}
	if b.Len() == 0 {
		return "tenant"
	}
	return b.String()
}

func applyMigrations(ctx context.Context, pool *pgxpool.Pool, fsys embed.FS, dir string) error {
	entries, err := fs.ReadDir(fsys, dir)
	if err != nil {
		return err
	}
	var names []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".sql") {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	for _, n := range names {
		data, err := fs.ReadFile(fsys, path.Join(dir, n))
		if err != nil {
			return err
		}
		if _, err := pool.Exec(ctx, string(data)); err != nil {
			return fmt.Errorf("migration %s: %w", n, err)
		}
	}
	return nil
}
