package repository

import (
	"context"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/salman/hms-backend/internal/modules/clinic/model"
)

func (r *Repository) CountPatientCategories(ctx context.Context, pool *pgxpool.Pool, clinicID, search string) (int, error) {
	var total int
	var err error
	if search != "" {
		err = pool.QueryRow(ctx,
			`SELECT COUNT(*) FROM patient_categories WHERE clinic_id = $1::uuid AND name ILIKE $2`,
			clinicID, "%"+search+"%",
		).Scan(&total)
	} else {
		err = pool.QueryRow(ctx,
			`SELECT COUNT(*) FROM patient_categories WHERE clinic_id = $1::uuid`, clinicID,
		).Scan(&total)
	}
	return total, err
}

func (r *Repository) ListPatientCategories(ctx context.Context, pool *pgxpool.Pool, f model.PCListFilter) ([]model.PatientCategoryDTO, error) {
	offset := (f.Page - 1) * f.Limit
	var rows interface {
		Next() bool
		Scan(...any) error
		Close()
	}
	var err error
	if f.Search != "" {
		rows, err = pool.Query(ctx, `
			SELECT id::text, name, patient_count
			FROM patient_categories
			WHERE clinic_id = $1::uuid AND name ILIKE $2
			ORDER BY created_at ASC LIMIT $3 OFFSET $4`,
			f.ClinicID, "%"+f.Search+"%", f.Limit, offset,
		)
	} else {
		rows, err = pool.Query(ctx, `
			SELECT id::text, name, patient_count
			FROM patient_categories
			WHERE clinic_id = $1::uuid
			ORDER BY created_at ASC LIMIT $2 OFFSET $3`,
			f.ClinicID, f.Limit, offset,
		)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	cats := []model.PatientCategoryDTO{}
	for rows.Next() {
		var cat model.PatientCategoryDTO
		if err := rows.Scan(&cat.ID, &cat.Name, &cat.PatientCount); err == nil {
			cats = append(cats, cat)
		}
	}
	return cats, nil
}

func (r *Repository) CreatePatientCategory(ctx context.Context, pool *pgxpool.Pool, clinicID, name string) (string, error) {
	var id string
	err := pool.QueryRow(ctx,
		`INSERT INTO patient_categories (clinic_id, name) VALUES ($1::uuid, $2) RETURNING id::text`,
		clinicID, name,
	).Scan(&id)
	return id, err
}

func (r *Repository) UpdatePatientCategory(ctx context.Context, pool *pgxpool.Pool, clinicID, catID, name string) (int64, error) {
	ct, err := pool.Exec(ctx,
		`UPDATE patient_categories SET name = $3 WHERE id = $1::uuid AND clinic_id = $2::uuid`,
		catID, clinicID, name,
	)
	if err != nil {
		return 0, err
	}
	return ct.RowsAffected(), nil
}

func (r *Repository) DeletePatientCategory(ctx context.Context, pool *pgxpool.Pool, clinicID, catID string) error {
	_, err := pool.Exec(ctx,
		`DELETE FROM patient_categories WHERE id = $1::uuid AND clinic_id = $2::uuid`,
		catID, clinicID,
	)
	return err
}

func (r *Repository) CountSpecialStatuses(ctx context.Context, pool *pgxpool.Pool, clinicID, search string) (int, error) {
	var total int
	var err error
	if search != "" {
		err = pool.QueryRow(ctx,
			`SELECT COUNT(*) FROM special_statuses WHERE clinic_id = $1::uuid AND name ILIKE $2`,
			clinicID, "%"+search+"%",
		).Scan(&total)
	} else {
		err = pool.QueryRow(ctx,
			`SELECT COUNT(*) FROM special_statuses WHERE clinic_id = $1::uuid`, clinicID,
		).Scan(&total)
	}
	return total, err
}

func (r *Repository) ListSpecialStatuses(ctx context.Context, pool *pgxpool.Pool, f model.PCListFilter) ([]model.SpecialStatusDTO, error) {
	offset := (f.Page - 1) * f.Limit
	var rows interface {
		Next() bool
		Scan(...any) error
		Close()
	}
	var err error
	if f.Search != "" {
		rows, err = pool.Query(ctx, `
			SELECT id::text, name, color, patient_count
			FROM special_statuses
			WHERE clinic_id = $1::uuid AND name ILIKE $2
			ORDER BY created_at ASC LIMIT $3 OFFSET $4`,
			f.ClinicID, "%"+f.Search+"%", f.Limit, offset,
		)
	} else {
		rows, err = pool.Query(ctx, `
			SELECT id::text, name, color, patient_count
			FROM special_statuses
			WHERE clinic_id = $1::uuid
			ORDER BY created_at ASC LIMIT $2 OFFSET $3`,
			f.ClinicID, f.Limit, offset,
		)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	statuses := []model.SpecialStatusDTO{}
	for rows.Next() {
		var s model.SpecialStatusDTO
		if err := rows.Scan(&s.ID, &s.Name, &s.Color, &s.PatientCount); err == nil {
			statuses = append(statuses, s)
		}
	}
	return statuses, nil
}

func (r *Repository) CreateSpecialStatus(ctx context.Context, pool *pgxpool.Pool, clinicID, name, color string) (string, error) {
	var id string
	err := pool.QueryRow(ctx,
		`INSERT INTO special_statuses (clinic_id, name, color) VALUES ($1::uuid, $2, $3) RETURNING id::text`,
		clinicID, name, color,
	).Scan(&id)
	return id, err
}

func (r *Repository) UpdateSpecialStatus(ctx context.Context, pool *pgxpool.Pool, clinicID, statusID string, req model.UpdateSpecialStatusRequest) error {
	sets := []string{}
	args := []any{}
	if req.Name != nil {
		n := strings.TrimSpace(*req.Name)
		args = append(args, n)
		sets = append(sets, "name = $"+strconv.Itoa(len(args)))
	}
	if req.Color != nil {
		args = append(args, strings.TrimSpace(*req.Color))
		sets = append(sets, "color = $"+strconv.Itoa(len(args)))
	}
	if len(sets) == 0 {
		return nil
	}
	args = append(args, statusID, clinicID)
	q := "UPDATE special_statuses SET " + strings.Join(sets, ", ") +
		" WHERE id = $" + strconv.Itoa(len(args)-1) + "::uuid" +
		" AND clinic_id = $" + strconv.Itoa(len(args)) + "::uuid"
	_, err := pool.Exec(ctx, q, args...)
	return err
}

func (r *Repository) DeleteSpecialStatus(ctx context.Context, pool *pgxpool.Pool, clinicID, statusID string) error {
	_, err := pool.Exec(ctx,
		`DELETE FROM special_statuses WHERE id = $1::uuid AND clinic_id = $2::uuid`,
		statusID, clinicID,
	)
	return err
}
