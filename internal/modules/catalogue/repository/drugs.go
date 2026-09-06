package repository

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/salman/hms-backend/internal/modules/catalogue/model"
)

func (r *Repository) ListDrugs(ctx context.Context, pool *pgxpool.Pool, f model.DrugListFilter) ([]model.DrugDTO, int, error) {
	args := []any{f.ClinicID}
	where := "clinic_id = $1::uuid"
	if f.Category != "" {
		args = append(args, f.Category)
		where += fmt.Sprintf(" AND category = $%d", len(args))
	}
	if f.Search != "" {
		args = append(args, "%"+strings.ToLower(f.Search)+"%")
		i := len(args)
		where += fmt.Sprintf(" AND (lower(drug_name) LIKE $%d OR lower(generic_name) LIKE $%d OR lower(category) LIKE $%d)", i, i, i)
	}

	var total int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM drugs WHERE "+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	args = append(args, f.PageSize, (f.Page-1)*f.PageSize)
	query := `
		SELECT id::text, clinic_id::text, drug_name, generic_name, category, strength,
		       item_code, manufacturer, instruction, primary_unit, secondary_unit,
		       reorder_level, hsn_code, tax, discount, created_at, updated_at
		FROM drugs
		WHERE ` + where + `
		ORDER BY created_at DESC
		LIMIT $` + strconv.Itoa(len(args)-1) + ` OFFSET $` + strconv.Itoa(len(args))
	rows, err := pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	out := []model.DrugDTO{}
	for rows.Next() {
		var d model.DrugDTO
		if err := rows.Scan(
			&d.ID, &d.ClinicID, &d.DrugName, &d.GenericName, &d.Category, &d.Strength,
			&d.ItemCode, &d.Manufacturer, &d.Instruction, &d.PrimaryUnit, &d.SecondaryUnit,
			&d.ReorderLevel, &d.HSNCode, &d.Tax, &d.Discount, &d.CreatedAt, &d.UpdatedAt,
		); err != nil {
			return nil, 0, err
		}
		out = append(out, d)
	}
	return out, total, nil
}

func (r *Repository) CreateDrug(ctx context.Context, pool *pgxpool.Pool, req model.CreateDrugRequest) (string, error) {
	var id string
	err := pool.QueryRow(ctx, `
		INSERT INTO drugs (clinic_id, drug_name, generic_name, category, strength, item_code,
		                   manufacturer, instruction, primary_unit, secondary_unit,
		                   reorder_level, hsn_code, tax, discount)
		VALUES ($1::uuid,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)
		RETURNING id::text`,
		req.ClinicID, req.DrugName, req.GenericName, req.Category, req.Strength, req.ItemCode,
		req.Manufacturer, req.Instruction, req.PrimaryUnit, req.SecondaryUnit,
		req.ReorderLevel, req.HSNCode, req.Tax, req.Discount,
	).Scan(&id)
	return id, err
}

func (r *Repository) UpdateDrug(ctx context.Context, pool *pgxpool.Pool, id, userID string, req model.CreateDrugRequest) (int64, error) {
	ct, err := pool.Exec(ctx, `
		UPDATE drugs SET
			drug_name = $3, generic_name = $4, category = $5, strength = $6, item_code = $7,
			manufacturer = $8, instruction = $9, primary_unit = $10, secondary_unit = $11,
			reorder_level = $12, hsn_code = $13, tax = $14, discount = $15, updated_at = now()
		WHERE id = $1::uuid
		  AND clinic_id IN (SELECT clinic_id FROM clinic_members WHERE user_id = $2::uuid)`,
		id, userID,
		req.DrugName, req.GenericName, req.Category, req.Strength, req.ItemCode,
		req.Manufacturer, req.Instruction, req.PrimaryUnit, req.SecondaryUnit,
		req.ReorderLevel, req.HSNCode, req.Tax, req.Discount,
	)
	if err != nil {
		return 0, err
	}
	return ct.RowsAffected(), nil
}

func (r *Repository) DeleteDrug(ctx context.Context, pool *pgxpool.Pool, id, userID string) (int64, error) {
	ct, err := pool.Exec(ctx, `
		DELETE FROM drugs
		WHERE id = $1::uuid
		  AND clinic_id IN (SELECT clinic_id FROM clinic_members WHERE user_id = $2::uuid)`,
		id, userID,
	)
	if err != nil {
		return 0, err
	}
	return ct.RowsAffected(), nil
}
