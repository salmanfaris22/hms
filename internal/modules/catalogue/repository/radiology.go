package repository

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/salman/hms-backend/internal/modules/catalogue/model"
)

func (r *Repository) ListRadiologyTests(ctx context.Context, pool *pgxpool.Pool, f model.RadiologyListFilter) ([]model.RadiologyTestDTO, int, error) {
	args := []any{f.ClinicID}
	where := "clinic_id = $1::uuid"
	if f.Category != "" {
		args = append(args, f.Category)
		where += fmt.Sprintf(" AND category = $%d", len(args))
	}
	if f.Search != "" {
		args = append(args, "%"+strings.ToLower(f.Search)+"%")
		i := len(args)
		where += fmt.Sprintf(" AND (lower(test_name) LIKE $%d OR lower(test_code) LIKE $%d OR lower(description) LIKE $%d)", i, i, i)
	}

	var total int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM radiology_tests WHERE "+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	args = append(args, f.PageSize, (f.Page-1)*f.PageSize)
	query := `
		SELECT id::text, clinic_id::text, test_name, test_code, category, body_part,
		       description, price, tax, created_at, updated_at
		FROM radiology_tests
		WHERE ` + where + `
		ORDER BY created_at DESC
		LIMIT $` + strconv.Itoa(len(args)-1) + ` OFFSET $` + strconv.Itoa(len(args))
	rows, err := pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	out := []model.RadiologyTestDTO{}
	for rows.Next() {
		var t model.RadiologyTestDTO
		if err := rows.Scan(
			&t.ID, &t.ClinicID, &t.TestName, &t.TestCode, &t.Category, &t.BodyPart,
			&t.Description, &t.Price, &t.Tax, &t.CreatedAt, &t.UpdatedAt,
		); err != nil {
			return nil, 0, err
		}
		out = append(out, t)
	}
	return out, total, nil
}

func (r *Repository) CreateRadiologyTest(ctx context.Context, pool *pgxpool.Pool, req model.CreateRadiologyTestRequest) (string, error) {
	var id string
	err := pool.QueryRow(ctx, `
		INSERT INTO radiology_tests (clinic_id, test_name, test_code, category, body_part,
		                             description, price, tax)
		VALUES ($1::uuid, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id::text`,
		req.ClinicID, req.TestName, req.TestCode, req.Category, req.BodyPart,
		req.Description, req.Price, req.Tax,
	).Scan(&id)
	return id, err
}

func (r *Repository) UpdateRadiologyTest(ctx context.Context, pool *pgxpool.Pool, id, userID string, req model.CreateRadiologyTestRequest) (int64, error) {
	ct, err := pool.Exec(ctx, `
		UPDATE radiology_tests SET
			test_name = $3, test_code = $4, category = $5, body_part = $6,
			description = $7, price = $8, tax = $9, updated_at = now()
		WHERE id = $1::uuid
		  AND clinic_id IN (SELECT clinic_id FROM clinic_members WHERE user_id = $2::uuid)`,
		id, userID,
		req.TestName, req.TestCode, req.Category, req.BodyPart,
		req.Description, req.Price, req.Tax,
	)
	if err != nil {
		return 0, err
	}
	return ct.RowsAffected(), nil
}

func (r *Repository) DeleteRadiologyTest(ctx context.Context, pool *pgxpool.Pool, id, userID string) (int64, error) {
	ct, err := pool.Exec(ctx, `
		DELETE FROM radiology_tests
		WHERE id = $1::uuid
		  AND clinic_id IN (SELECT clinic_id FROM clinic_members WHERE user_id = $2::uuid)`,
		id, userID,
	)
	if err != nil {
		return 0, err
	}
	return ct.RowsAffected(), nil
}
