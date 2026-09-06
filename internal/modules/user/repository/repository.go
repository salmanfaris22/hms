package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/salman/hms-backend/internal/modules/user/model"
)

type Repository struct {
	registry *pgxpool.Pool
}

func New(registry *pgxpool.Pool) *Repository {
	return &Repository{registry: registry}
}

func (r *Repository) Registry() *pgxpool.Pool { return r.registry }

func (r *Repository) FindDirectory(ctx context.Context, email string) (model.DirectoryEntry, error) {
	var e model.DirectoryEntry
	err := r.registry.QueryRow(ctx,
		`SELECT tenant_id::text, user_id::text FROM user_directory WHERE email = $1`,
		email,
	).Scan(&e.TenantID, &e.UserID)
	return e, err
}

func (r *Repository) GetTenantStatus(ctx context.Context, tenantID string) (model.TenantStatus, error) {
	var s model.TenantStatus
	err := r.registry.QueryRow(ctx, `
		SELECT is_active, subscription_end, modules
		FROM tenants WHERE id = $1::uuid`, tenantID,
	).Scan(&s.IsActive, &s.SubEnd, &s.Modules)
	return s, err
}

func (r *Repository) FindUser(ctx context.Context, pool *pgxpool.Pool, userID string) (model.DBUser, error) {
	var u model.DBUser
	err := pool.QueryRow(ctx, `
		SELECT email, password_hash, full_name, role
		FROM users WHERE id = $1::uuid AND is_active = true`, userID,
	).Scan(&u.Email, &u.PasswordHash, &u.FullName, &u.Role)
	return u, err
}

func (r *Repository) TouchLastLogin(ctx context.Context, pool *pgxpool.Pool, userID string) {
	_, _ = pool.Exec(ctx,
		`UPDATE users SET last_login_at = now() WHERE id = $1::uuid`, userID,
	)
}
