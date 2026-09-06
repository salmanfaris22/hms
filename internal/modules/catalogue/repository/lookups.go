package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/salman/hms-backend/internal/modules/catalogue/model"
)

func (r *Repository) ListLookup(ctx context.Context, pool *pgxpool.Pool, table, clinicID, kind string) ([]model.LookupDTO, error) {
	args := []any{clinicID}
	where := "clinic_id = $1::uuid"
	if kind != "" {
		args = append(args, kind)
		where += " AND kind = $2"
	}
	rows, err := pool.Query(ctx,
		"SELECT id::text, name FROM "+table+" WHERE "+where+" ORDER BY lower(name)",
		args...,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.LookupDTO{}
	for rows.Next() {
		var l model.LookupDTO
		if err := rows.Scan(&l.ID, &l.Name); err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, nil
}

func (r *Repository) CreateLookup(ctx context.Context, pool *pgxpool.Pool, table, clinicID, name, kind string) (string, error) {
	var id string
	var err error
	if kind != "" {
		err = pool.QueryRow(ctx,
			"INSERT INTO "+table+" (clinic_id, name, kind) VALUES ($1::uuid, $2, $3) "+
				"ON CONFLICT (clinic_id, kind, lower(name)) DO UPDATE SET name = EXCLUDED.name "+
				"RETURNING id::text",
			clinicID, name, kind,
		).Scan(&id)
	} else {
		err = pool.QueryRow(ctx,
			"INSERT INTO "+table+" (clinic_id, name) VALUES ($1::uuid, $2) "+
				"ON CONFLICT (clinic_id, lower(name)) DO UPDATE SET name = EXCLUDED.name "+
				"RETURNING id::text",
			clinicID, name,
		).Scan(&id)
	}
	return id, err
}

func (r *Repository) DeleteLookup(ctx context.Context, pool *pgxpool.Pool, table, id, userID string) (int64, error) {
	ct, err := pool.Exec(ctx,
		"DELETE FROM "+table+" WHERE id = $1::uuid "+
			"AND clinic_id IN (SELECT clinic_id FROM clinic_members WHERE user_id = $2::uuid)",
		id, userID,
	)
	if err != nil {
		return 0, err
	}
	return ct.RowsAffected(), nil
}
