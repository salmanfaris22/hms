package repository

import (
	"context"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/salman/hms-backend/internal/modules/clinic/model"
)

func IsOrgTable(name string) bool {
	switch name {
	case "departments", "specializations", "designations":
		return true
	}
	return false
}

func (r *Repository) CountOrgItems(ctx context.Context, pool *pgxpool.Pool, table, clinicID, search string) (int, error) {
	var total int
	var err error
	if search != "" {
		err = pool.QueryRow(ctx,
			`SELECT COUNT(*) FROM `+table+` WHERE clinic_id = $1::uuid AND name ILIKE $2`,
			clinicID, "%"+search+"%",
		).Scan(&total)
	} else {
		err = pool.QueryRow(ctx,
			`SELECT COUNT(*) FROM `+table+` WHERE clinic_id = $1::uuid`, clinicID,
		).Scan(&total)
	}
	return total, err
}

func (r *Repository) ListOrgItems(ctx context.Context, pool *pgxpool.Pool, table string, f model.OrgListFilter) ([]model.OrgItemDTO, error) {
	offset := (f.Page - 1) * f.Limit
	var rows interface {
		Next() bool
		Scan(...any) error
		Close()
	}
	var err error
	if f.Search != "" {
		rows, err = pool.Query(ctx,
			`SELECT id::text, name, description, staff_count
			 FROM `+table+`
			 WHERE clinic_id = $1::uuid AND name ILIKE $2
			 ORDER BY created_at ASC LIMIT $3 OFFSET $4`,
			f.ClinicID, "%"+f.Search+"%", f.Limit, offset,
		)
	} else {
		rows, err = pool.Query(ctx,
			`SELECT id::text, name, description, staff_count
			 FROM `+table+`
			 WHERE clinic_id = $1::uuid
			 ORDER BY created_at ASC LIMIT $2 OFFSET $3`,
			f.ClinicID, f.Limit, offset,
		)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []model.OrgItemDTO{}
	for rows.Next() {
		var it model.OrgItemDTO
		if err := rows.Scan(&it.ID, &it.Name, &it.Description, &it.StaffCount); err == nil {
			items = append(items, it)
		}
	}
	return items, nil
}

func (r *Repository) CreateOrgItem(ctx context.Context, pool *pgxpool.Pool, table, clinicID, name, description string) (string, error) {
	var id string
	err := pool.QueryRow(ctx,
		`INSERT INTO `+table+` (clinic_id, name, description)
		 VALUES ($1::uuid, $2, $3)
		 RETURNING id::text`,
		clinicID, name, description,
	).Scan(&id)
	return id, err
}

func (r *Repository) UpdateOrgItem(ctx context.Context, pool *pgxpool.Pool, table, clinicID, itemID string, req model.UpdateOrgItemRequest) error {
	sets := []string{}
	args := []any{}
	if req.Name != nil {
		n := strings.TrimSpace(*req.Name)
		args = append(args, n)
		sets = append(sets, "name = $"+strconv.Itoa(len(args)))
	}
	if req.Description != nil {
		args = append(args, strings.TrimSpace(*req.Description))
		sets = append(sets, "description = $"+strconv.Itoa(len(args)))
	}
	if len(sets) == 0 {
		return nil
	}
	args = append(args, itemID, clinicID)
	q := "UPDATE " + table + " SET " + strings.Join(sets, ", ") +
		" WHERE id = $" + strconv.Itoa(len(args)-1) + "::uuid" +
		" AND clinic_id = $" + strconv.Itoa(len(args)) + "::uuid"
	_, err := pool.Exec(ctx, q, args...)
	return err
}

func (r *Repository) DeleteOrgItem(ctx context.Context, pool *pgxpool.Pool, table, clinicID, itemID string) error {
	_, err := pool.Exec(ctx,
		`DELETE FROM `+table+` WHERE id = $1::uuid AND clinic_id = $2::uuid`,
		itemID, clinicID,
	)
	return err
}

func (r *Repository) CopyOrgTable(ctx context.Context, pool *pgxpool.Pool, table, sourceID, targetID string) (int, error) {
	ct, err := pool.Exec(ctx,
		`INSERT INTO `+table+` (clinic_id, name, description)
		 SELECT $2::uuid, name, description FROM `+table+`
		 WHERE clinic_id = $1::uuid
		 ON CONFLICT (clinic_id, name) DO NOTHING`,
		sourceID, targetID,
	)
	if err != nil {
		return 0, err
	}
	return int(ct.RowsAffected()), nil
}
