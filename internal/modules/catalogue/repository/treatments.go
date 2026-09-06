package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/salman/hms-backend/internal/modules/catalogue/model"
)

func (r *Repository) ListTreatments(ctx context.Context, pool *pgxpool.Pool, f model.TreatmentListFilter) ([]model.TreatmentDTO, int, error) {
	orderBy := "created_at DESC"
	if strings.ToLower(f.Sort) == "fifo" {
		orderBy = "created_at ASC"
	}

	args := []any{f.ClinicID}
	where := "clinic_id = $1::uuid"
	if f.Search != "" {
		args = append(args, "%"+strings.ToLower(f.Search)+"%")
		i := len(args)
		where += fmt.Sprintf(" AND (lower(treatment_name) LIKE $%d OR lower(treatment_code) LIKE $%d OR lower(description) LIKE $%d)", i, i, i)
	}

	var total int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM treatments WHERE "+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	args = append(args, f.PageSize, (f.Page-1)*f.PageSize)
	query := `
		SELECT id::text, clinic_id::text, treatment_name, treatment_code, description,
		       price, discount, tax, consumables, created_at, updated_at
		FROM treatments
		WHERE ` + where + `
		ORDER BY ` + orderBy + `
		LIMIT $` + strconv.Itoa(len(args)-1) + ` OFFSET $` + strconv.Itoa(len(args))
	rows, err := pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	out := []model.TreatmentDTO{}
	for rows.Next() {
		var t model.TreatmentDTO
		var raw []byte
		if err := rows.Scan(
			&t.ID, &t.ClinicID, &t.TreatmentName, &t.TreatmentCode, &t.Description,
			&t.Price, &t.Discount, &t.Tax, &raw, &t.CreatedAt, &t.UpdatedAt,
		); err != nil {
			return nil, 0, err
		}
		if len(raw) > 0 {
			_ = json.Unmarshal(raw, &t.Consumables)
		}
		if t.Consumables == nil {
			t.Consumables = []model.TreatmentConsumable{}
		}
		out = append(out, t)
	}
	return out, total, nil
}

func (r *Repository) CreateTreatment(ctx context.Context, pool *pgxpool.Pool, req model.CreateTreatmentRequest) (string, error) {
	consumablesJSON, _ := json.Marshal(req.Consumables)
	var id string
	err := pool.QueryRow(ctx, `
		INSERT INTO treatments (clinic_id, treatment_name, treatment_code, description,
		                        price, discount, tax, consumables)
		VALUES ($1::uuid, $2, $3, $4, $5, $6, $7, $8::jsonb)
		RETURNING id::text`,
		req.ClinicID, req.TreatmentName, req.TreatmentCode, req.Description,
		req.Price, req.Discount, req.Tax, string(consumablesJSON),
	).Scan(&id)
	return id, err
}

func (r *Repository) UpdateTreatment(ctx context.Context, pool *pgxpool.Pool, id, userID string, req model.CreateTreatmentRequest) (int64, error) {
	consumablesJSON, _ := json.Marshal(req.Consumables)
	ct, err := pool.Exec(ctx, `
		UPDATE treatments SET
			treatment_name = $3, treatment_code = $4, description = $5,
			price = $6, discount = $7, tax = $8, consumables = $9::jsonb, updated_at = now()
		WHERE id = $1::uuid
		  AND clinic_id IN (SELECT clinic_id FROM clinic_members WHERE user_id = $2::uuid)`,
		id, userID,
		req.TreatmentName, req.TreatmentCode, req.Description,
		req.Price, req.Discount, req.Tax, string(consumablesJSON),
	)
	if err != nil {
		return 0, err
	}
	return ct.RowsAffected(), nil
}

func (r *Repository) DeleteTreatment(ctx context.Context, pool *pgxpool.Pool, id, userID string) (int64, error) {
	ct, err := pool.Exec(ctx, `
		DELETE FROM treatments
		WHERE id = $1::uuid
		  AND clinic_id IN (SELECT clinic_id FROM clinic_members WHERE user_id = $2::uuid)`,
		id, userID,
	)
	if err != nil {
		return 0, err
	}
	return ct.RowsAffected(), nil
}
