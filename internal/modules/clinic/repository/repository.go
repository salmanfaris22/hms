package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/salman/hms-backend/internal/infrastructure/persistence/postgres"
)

type Repository struct {
	tenants *postgres.TenantResolver
}

func New(tenants *postgres.TenantResolver) *Repository {
	return &Repository{tenants: tenants}
}

func (r *Repository) Tenants() *postgres.TenantResolver { return r.tenants }
func (r *Repository) Pool(ctx context.Context, tenantID string) (*pgxpool.Pool, error) {
	return r.tenants.Pool(ctx, tenantID)
}

func (r *Repository) IsMember(ctx context.Context, pool *pgxpool.Pool, clinicID, userID string) bool {
	var ok bool
	_ = pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM clinic_members WHERE clinic_id = $1::uuid AND user_id = $2::uuid)`,
		clinicID, userID,
	).Scan(&ok)
	return ok
}
