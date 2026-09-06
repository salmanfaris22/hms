package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/salman/hms-backend/internal/modules/inventory/model"
)

func (r *Repository) AssetSummary(ctx context.Context, pool *pgxpool.Pool, clinicID string) (model.AssetSummary, error) {
	var s model.AssetSummary
	err := pool.QueryRow(ctx, `
		SELECT COUNT(*),
		       COUNT(*) FILTER (WHERE status='Active'),
		       COUNT(*) FILTER (WHERE status='Maintenance'),
		       COUNT(*) FILTER (WHERE status='Decommissioned'),
		       COALESCE(SUM(current_value),0)
		FROM inv_assets WHERE clinic_id=$1::uuid`, clinicID,
	).Scan(&s.TotalAssets, &s.ActiveAssets, &s.Maintenance, &s.Decommissioned, &s.TotalBookValue)
	return s, err
}

func (r *Repository) ListAssets(ctx context.Context, pool *pgxpool.Pool, f model.AssetListFilter) ([]model.AssetDTO, int, error) {
	offset := (f.Page - 1) * DefaultLimit
	rows, err := pool.Query(ctx, `
		SELECT id, asset_no, name, manufacturer, serial_number, model, location,
		       purchase_date::text, current_value, purchase_cost, warranty_expiry::text,
		       category, department, condition, status, notes, created_at::text
		FROM inv_assets
		WHERE clinic_id=$1::uuid
		  AND ($2='' OR name ILIKE '%'||$2||'%' OR asset_no ILIKE '%'||$2||'%')
		  AND ($3='' OR status=$3)
		  AND ($4='' OR condition=$4)
		  AND ($5='' OR category=$5)
		ORDER BY created_at DESC
		LIMIT $6 OFFSET $7`,
		f.ClinicID, f.Q, f.StatusFilter, f.CondFilter, f.CatFilter, DefaultLimit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	assets := []model.AssetDTO{}
	for rows.Next() {
		var a model.AssetDTO
		if err := rows.Scan(&a.ID, &a.AssetNo, &a.Name, &a.Manufacturer, &a.SerialNumber,
			&a.Model, &a.Location, &a.PurchaseDate, &a.CurrentValue, &a.PurchaseCost,
			&a.WarrantyExpiry, &a.Category, &a.Department, &a.Condition, &a.Status,
			&a.Notes, &a.CreatedAt); err == nil {
			assets = append(assets, a)
		}
	}
	var total int
	_ = pool.QueryRow(ctx, `SELECT COUNT(*) FROM inv_assets WHERE clinic_id=$1::uuid AND ($2='' OR name ILIKE '%'||$2||'%') AND ($3='' OR status=$3) AND ($4='' OR condition=$4) AND ($5='' OR category=$5)`,
		f.ClinicID, f.Q, f.StatusFilter, f.CondFilter, f.CatFilter).Scan(&total)
	return assets, total, nil
}

func (r *Repository) CreateAsset(ctx context.Context, pool *pgxpool.Pool, clinicID string, req model.CreateAssetRequest) (string, string, error) {
	var cnt int
	_ = pool.QueryRow(ctx, `SELECT COUNT(*) FROM inv_assets WHERE clinic_id=$1::uuid`, clinicID).Scan(&cnt)
	assetNo := fmt.Sprintf("AST-PUR-%03d", cnt+1)
	if req.Status == "" {
		req.Status = "Active"
	}
	if req.Condition == "" {
		req.Condition = "Good"
	}

	var id string
	err := pool.QueryRow(ctx, `
		INSERT INTO inv_assets (clinic_id, asset_no, name, manufacturer, serial_number, model, location,
		    purchase_date, current_value, purchase_cost, warranty_expiry, category, department, condition, status, notes)
		VALUES ($1::uuid,$2,$3,$4,$5,$6,$7,$8::date,$9,$10,$11::date,$12,$13,$14,$15,$16)
		RETURNING id`,
		clinicID, assetNo, req.Name, req.Manufacturer, req.SerialNumber, req.Model,
		req.Location, req.PurchaseDate, req.CurrentValue, req.PurchaseCost,
		req.WarrantyExpiry, req.Category, req.Department, req.Condition, req.Status, req.Notes,
	).Scan(&id)
	return id, assetNo, err
}

func (r *Repository) UpdateAsset(ctx context.Context, pool *pgxpool.Pool, clinicID, id string, req model.CreateAssetRequest) error {
	_, err := pool.Exec(ctx, `
		UPDATE inv_assets SET name=$1, manufacturer=$2, serial_number=$3, model=$4, location=$5,
		    purchase_date=$6::date, current_value=$7, purchase_cost=$8, warranty_expiry=$9::date,
		    category=$10, department=$11, condition=$12, status=$13, notes=$14
		WHERE id=$15::uuid AND clinic_id=$16::uuid`,
		req.Name, req.Manufacturer, req.SerialNumber, req.Model, req.Location,
		req.PurchaseDate, req.CurrentValue, req.PurchaseCost, req.WarrantyExpiry,
		req.Category, req.Department, req.Condition, req.Status, req.Notes,
		id, clinicID,
	)
	return err
}

func (r *Repository) DeleteAsset(ctx context.Context, pool *pgxpool.Pool, clinicID, id string) error {
	_, err := pool.Exec(ctx, `DELETE FROM inv_assets WHERE id=$1::uuid AND clinic_id=$2::uuid`, id, clinicID)
	return err
}
