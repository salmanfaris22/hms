package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/salman/hms-backend/internal/infrastructure/persistence/postgres"
)

const DefaultLimit = 8

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
