package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	patientService "github.com/salman/hms-backend/internal/modules/patient/service"
)

type Repository struct {
	patients *patientService.Service
}

func New(patients *patientService.Service) *Repository {
	return &Repository{patients: patients}
}

func (r *Repository) Patients() *patientService.Service { return r.patients }

func (r *Repository) Pool(ctx context.Context, tenantID string) (*pgxpool.Pool, error) {
	return r.patients.Tenants().Pool(ctx, tenantID)
}

func (r *Repository) AssertMember(ctx context.Context, pool *pgxpool.Pool, userID, clinicID string) error {
	return r.patients.AssertMember(ctx, pool, userID, clinicID)
}
