package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/salman/hms-backend/internal/modules/inventory/model"
)

func (r *Repository) ListDrugs(ctx context.Context, pool *pgxpool.Pool, f model.DrugListFilter) ([]model.DrugDTO, int, error) {
	orderCol := "d.name"
	if f.Sort == "expiry" {
		orderCol = "s.min_expiry"
	} else if f.Sort == "stock" {
		orderCol = "s.total_qty"
	}
	offset := (f.Page - 1) * DefaultLimit

	rows, err := pool.Query(ctx, `
		SELECT d.id, d.name, d.generic_name,
		       d.category_id, COALESCE(c.name,'') AS category_name,
		       d.strength, d.item_code,
		       d.manufacturer_id, COALESCE(m.name,'') AS manufacturer_name,
		       d.instruction,
		       COALESCE(s.total_qty,0) AS stock_level,
		       COALESCE(s.min_mrp,0) AS price,
		       s.min_expiry::text AS expiry_date
		FROM inv_drugs d
		LEFT JOIN inv_categories c ON c.id = d.category_id
		LEFT JOIN inv_manufacturers m ON m.id = d.manufacturer_id
		LEFT JOIN (
		    SELECT drug_id,
		           SUM(qty_available) AS total_qty,
		           MIN(mrp) AS min_mrp,
		           MIN(expiry_date) AS min_expiry
		    FROM inv_stock
		    GROUP BY drug_id
		) s ON s.drug_id = d.id
		WHERE d.clinic_id = $1::uuid
		  AND ($2 = '' OR d.name ILIKE '%'||$2||'%' OR d.item_code ILIKE '%'||$2||'%')
		  AND ($3 = '' OR d.category_id::text = $3)
		  AND ($4 = '' OR CASE
		        WHEN COALESCE(s.total_qty,0) = 0 THEN 'Out of Stock'
		        WHEN COALESCE(s.total_qty,0) <= 10 THEN 'Low Stock'
		        ELSE 'In Stock'
		       END = $4)
		ORDER BY `+orderCol+`
		LIMIT $5 OFFSET $6`,
		f.ClinicID, f.Q, f.CategoryID, f.StatusFilter, DefaultLimit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	drugs := []model.DrugDTO{}
	for rows.Next() {
		var d model.DrugDTO
		var expiryDate *string
		err := rows.Scan(&d.ID, &d.Name, &d.GenericName,
			&d.CategoryID, &d.CategoryName,
			&d.Strength, &d.ItemCode,
			&d.ManufacturerID, &d.Manufacturer,
			&d.Instruction,
			&d.StockLevel, &d.Price, &expiryDate,
		)
		if err != nil {
			continue
		}
		d.ExpiryDate = expiryDate
		switch {
		case d.StockLevel == 0:
			d.Status = "Out of Stock"
		case d.StockLevel <= 10:
			d.Status = "Low Stock"
		default:
			d.Status = "In Stock"
		}
		drugs = append(drugs, d)
	}

	var total int
	_ = pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM inv_drugs d
		WHERE d.clinic_id = $1::uuid
		  AND ($2 = '' OR d.name ILIKE '%'||$2||'%' OR d.item_code ILIKE '%'||$2||'%')
		  AND ($3 = '' OR d.category_id::text = $3)`,
		f.ClinicID, f.Q, f.CategoryID,
	).Scan(&total)

	return drugs, total, nil
}

func (r *Repository) DrugSummary(ctx context.Context, pool *pgxpool.Pool, clinicID string) (model.StockSummary, error) {
	var s model.StockSummary
	err := pool.QueryRow(ctx, `
		SELECT
		  COUNT(DISTINCT d.id),
		  COALESCE(SUM(s.qty_available * s.mrp), 0),
		  COUNT(DISTINCT CASE WHEN s2.total_qty <= 10 AND s2.total_qty > 0 THEN d.id END),
		  COUNT(DISTINCT CASE WHEN s.expiry_date <= CURRENT_DATE + INTERVAL '30 days' AND s.expiry_date >= CURRENT_DATE THEN d.id END),
		  COUNT(DISTINCT d.category_id)
		FROM inv_drugs d
		LEFT JOIN inv_stock s ON s.drug_id = d.id
		LEFT JOIN (SELECT drug_id, SUM(qty_available) AS total_qty FROM inv_stock GROUP BY drug_id) s2 ON s2.drug_id = d.id
		WHERE d.clinic_id = $1::uuid`, clinicID,
	).Scan(&s.TotalItems, &s.TotalValue, &s.LowStock, &s.ExpiringSoon, &s.Categories)
	return s, err
}

func (r *Repository) CreateDrug(ctx context.Context, pool *pgxpool.Pool, clinicID string, req model.CreateDrugRequest) (string, error) {
	var id string
	err := pool.QueryRow(ctx, `
		INSERT INTO inv_drugs (clinic_id, name, generic_name, category_id, strength, item_code, manufacturer_id, instruction)
		VALUES ($1::uuid, $2, $3, $4::uuid, $5, $6, $7::uuid, $8)
		RETURNING id`,
		clinicID, req.Name, req.GenericName, req.CategoryID, req.Strength, req.ItemCode, req.ManufacturerID, req.Instruction,
	).Scan(&id)
	return id, err
}

func (r *Repository) DeleteDrug(ctx context.Context, pool *pgxpool.Pool, clinicID, id string) error {
	_, err := pool.Exec(ctx, `DELETE FROM inv_drugs WHERE id=$1::uuid AND clinic_id=$2::uuid`, id, clinicID)
	return err
}

func (r *Repository) ListCategories(ctx context.Context, pool *pgxpool.Pool, clinicID string) ([]model.CategoryDTO, error) {
	rows, err := pool.Query(ctx, `SELECT id, name FROM inv_categories WHERE clinic_id=$1::uuid ORDER BY name`, clinicID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	cats := []model.CategoryDTO{}
	for rows.Next() {
		var cat model.CategoryDTO
		if err := rows.Scan(&cat.ID, &cat.Name); err == nil {
			cats = append(cats, cat)
		}
	}
	return cats, nil
}

func (r *Repository) CreateCategory(ctx context.Context, pool *pgxpool.Pool, clinicID, name string) (string, error) {
	var id string
	err := pool.QueryRow(ctx, `INSERT INTO inv_categories (clinic_id, name) VALUES ($1::uuid,$2) RETURNING id`, clinicID, name).Scan(&id)
	return id, err
}

func (r *Repository) DeleteCategory(ctx context.Context, pool *pgxpool.Pool, clinicID, id string) error {
	_, err := pool.Exec(ctx, `DELETE FROM inv_categories WHERE id=$1::uuid AND clinic_id=$2::uuid`, id, clinicID)
	return err
}

func (r *Repository) ListManufacturers(ctx context.Context, pool *pgxpool.Pool, clinicID string) ([]model.ManufacturerDTO, error) {
	rows, err := pool.Query(ctx, `SELECT id, name FROM inv_manufacturers WHERE clinic_id=$1::uuid ORDER BY name`, clinicID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	mfrs := []model.ManufacturerDTO{}
	for rows.Next() {
		var m model.ManufacturerDTO
		if err := rows.Scan(&m.ID, &m.Name); err == nil {
			mfrs = append(mfrs, m)
		}
	}
	return mfrs, nil
}

func (r *Repository) CreateManufacturer(ctx context.Context, pool *pgxpool.Pool, clinicID, name string) (string, error) {
	var id string
	err := pool.QueryRow(ctx, `INSERT INTO inv_manufacturers (clinic_id, name) VALUES ($1::uuid,$2) RETURNING id`, clinicID, name).Scan(&id)
	return id, err
}

func (r *Repository) DeleteManufacturer(ctx context.Context, pool *pgxpool.Pool, clinicID, id string) error {
	_, err := pool.Exec(ctx, `DELETE FROM inv_manufacturers WHERE id=$1::uuid AND clinic_id=$2::uuid`, id, clinicID)
	return err
}

func (r *Repository) ListStock(ctx context.Context, pool *pgxpool.Pool, clinicID, drugID string) ([]model.StockDTO, error) {
	rows, err := pool.Query(ctx, `
		SELECT id, drug_id, batch_code, expiry_date::text, qty_available, purchase_rate, mrp
		FROM inv_stock
		WHERE clinic_id=$1::uuid AND ($2='' OR drug_id=$2::uuid)
		ORDER BY expiry_date ASC NULLS LAST`,
		clinicID, drugID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	batches := []model.StockDTO{}
	for rows.Next() {
		var s model.StockDTO
		if err := rows.Scan(&s.ID, &s.DrugID, &s.BatchCode, &s.ExpiryDate, &s.QtyAvailable, &s.PurchaseRate, &s.MRP); err == nil {
			batches = append(batches, s)
		}
	}
	return batches, nil
}

func (r *Repository) AddStock(ctx context.Context, pool *pgxpool.Pool, clinicID string, req model.CreateStockRequest) (string, error) {
	var id string
	err := pool.QueryRow(ctx, `
		INSERT INTO inv_stock (clinic_id, drug_id, batch_code, expiry_date, qty_available, purchase_rate, mrp)
		VALUES ($1::uuid,$2::uuid,$3,$4::date,$5,$6,$7) RETURNING id`,
		clinicID, req.DrugID, req.BatchCode, req.ExpiryDate, req.QtyAvailable, req.PurchaseRate, req.MRP,
	).Scan(&id)
	return id, err
}

func (r *Repository) SearchPOS(ctx context.Context, pool *pgxpool.Pool, clinicID, q, catID string) ([]model.POSItem, error) {
	rows, err := pool.Query(ctx, `
		SELECT d.id, d.name,
		       COALESCE(m.name,'') AS manufacturer,
		       COALESCE(cat.name,'') AS category_name,
		       COALESCE(s.min_mrp,0),
		       COALESCE(s.total_qty,0),
		       COALESCE(s.batch_code,''),
		       s.expiry_date::text
		FROM inv_drugs d
		LEFT JOIN inv_manufacturers m ON m.id=d.manufacturer_id
		LEFT JOIN inv_categories cat ON cat.id=d.category_id
		LEFT JOIN (
		    SELECT drug_id, MIN(mrp) AS min_mrp, SUM(qty_available) AS total_qty,
		           (SELECT batch_code FROM inv_stock WHERE drug_id=s2.drug_id ORDER BY expiry_date ASC NULLS LAST LIMIT 1) AS batch_code,
		           (SELECT expiry_date FROM inv_stock WHERE drug_id=s2.drug_id ORDER BY expiry_date ASC NULLS LAST LIMIT 1) AS expiry_date
		    FROM inv_stock s2 GROUP BY drug_id
		) s ON s.drug_id=d.id
		WHERE d.clinic_id=$1::uuid
		  AND ($2='' OR d.name ILIKE '%'||$2||'%')
		  AND ($3='' OR d.category_id::text=$3)
		  AND COALESCE(s.total_qty,0) > 0
		ORDER BY d.name
		LIMIT 50`,
		clinicID, q, catID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []model.POSItem{}
	for rows.Next() {
		var it model.POSItem
		if err := rows.Scan(&it.ID, &it.Name, &it.Manufacturer, &it.CategoryName, &it.Price, &it.Stock, &it.BatchCode, &it.ExpiryDate); err == nil {
			items = append(items, it)
		}
	}
	return items, nil
}
