package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Generic JSON settings table helpers. The table must have columns
// (clinic_id uuid, settings jsonb, updated_at timestamptz).

func (r *Repository) GetJSONSettings(ctx context.Context, pool *pgxpool.Pool, table, clinicID string) ([]byte, error) {
	var raw []byte
	err := pool.QueryRow(ctx,
		"SELECT settings FROM "+table+" WHERE clinic_id = $1::uuid", clinicID,
	).Scan(&raw)
	return raw, err
}

func (r *Repository) PutJSONSettings(ctx context.Context, pool *pgxpool.Pool, table, clinicID string, body []byte) error {
	_, err := pool.Exec(ctx, `
		INSERT INTO `+table+` (clinic_id, settings, updated_at)
		VALUES ($1::uuid, $2::jsonb, now())
		ON CONFLICT (clinic_id) DO UPDATE SET
			settings   = EXCLUDED.settings,
			updated_at = now()`,
		clinicID, string(body),
	)
	return err
}
