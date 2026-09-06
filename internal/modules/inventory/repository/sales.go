package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/salman/hms-backend/internal/modules/inventory/model"
)

func (r *Repository) SalesSummary(ctx context.Context, pool *pgxpool.Pool, clinicID string) (model.SaleSummary, error) {
	var s model.SaleSummary
	err := pool.QueryRow(ctx, `
		SELECT
		  COALESCE(SUM(CASE WHEN created_at::date = CURRENT_DATE THEN grand_total END),0),
		  COALESCE(SUM(CASE WHEN created_at >= CURRENT_DATE - INTERVAL '7 days' THEN grand_total END),0),
		  COALESCE(SUM(CASE WHEN created_at >= DATE_TRUNC('month', CURRENT_DATE) THEN grand_total END),0),
		  COUNT(*)
		FROM inv_sales WHERE clinic_id=$1::uuid`, clinicID,
	).Scan(&s.TodaySales, &s.WeeklyRevenue, &s.MonthlyRevenue, &s.TotalTxns)
	return s, err
}

func (r *Repository) ListSales(ctx context.Context, pool *pgxpool.Pool, f model.SaleListFilter) ([]model.SaleDTO, int, error) {
	offset := (f.Page - 1) * DefaultLimit
	rows, err := pool.Query(ctx, `
		SELECT s.id, s.invoice_no, s.customer_name, s.opd_id, s.prescribed_by,
		       s.stock_point, s.payment_method, s.grand_total, s.status,
		       COUNT(si.id) AS item_count,
		       s.created_at::text
		FROM inv_sales s
		LEFT JOIN inv_sale_items si ON si.sale_id=s.id
		WHERE s.clinic_id=$1::uuid
		  AND ($2='' OR s.invoice_no ILIKE '%'||$2||'%' OR s.customer_name ILIKE '%'||$2||'%')
		  AND ($3='' OR s.status=$3)
		GROUP BY s.id ORDER BY s.created_at DESC
		LIMIT $4 OFFSET $5`,
		f.ClinicID, f.Q, f.StatusFilter, DefaultLimit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	sales := []model.SaleDTO{}
	for rows.Next() {
		var d model.SaleDTO
		if err := rows.Scan(&d.ID, &d.InvoiceNo, &d.CustomerName, &d.OpdID, &d.PrescribedBy,
			&d.StockPoint, &d.PaymentMethod, &d.GrandTotal, &d.Status, &d.ItemCount, &d.CreatedAt); err == nil {
			sales = append(sales, d)
		}
	}
	var total int
	_ = pool.QueryRow(ctx, `SELECT COUNT(*) FROM inv_sales WHERE clinic_id=$1::uuid AND ($2='' OR invoice_no ILIKE '%'||$2||'%' OR customer_name ILIKE '%'||$2||'%') AND ($3='' OR status=$3)`,
		f.ClinicID, f.Q, f.StatusFilter).Scan(&total)
	return sales, total, nil
}

func (r *Repository) CreateSale(ctx context.Context, pool *pgxpool.Pool, clinicID string, req model.CreateSaleRequest) (string, string, error) {
	var cnt int
	_ = pool.QueryRow(ctx, `SELECT COUNT(*) FROM inv_sales WHERE clinic_id=$1::uuid`, clinicID).Scan(&cnt)
	invoiceNo := fmt.Sprintf("INV-%d-%04d", time.Now().Year(), cnt+1)

	tx, err := pool.Begin(ctx)
	if err != nil {
		return "", "", err
	}
	defer tx.Rollback(ctx)

	var saleID string
	err = tx.QueryRow(ctx, `
		INSERT INTO inv_sales (clinic_id, invoice_no, customer_name, opd_id, prescribed_by, stock_point, payment_method,
		                       subtotal, discount_pct, gst_enabled, round_off, shipping, grand_total, notes)
		VALUES ($1::uuid,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)
		RETURNING id`,
		clinicID, invoiceNo, req.CustomerName, req.OpdID, req.PrescribedBy,
		req.StockPoint, req.PaymentMethod, req.Subtotal, req.DiscountPct,
		req.GstEnabled, req.RoundOff, req.Shipping, req.GrandTotal, req.Notes,
	).Scan(&saleID)
	if err != nil {
		return "", "", err
	}

	for _, it := range req.Items {
		_, err = tx.Exec(ctx, `
			INSERT INTO inv_sale_items (sale_id, drug_id, drug_name, batch_code, expiry_date, unit, qty, unit_price, discount, tax, total)
			VALUES ($1::uuid, NULLIF($2,'')::uuid, $3, $4, $5::date, $6, $7, $8, $9, $10, $11)`,
			saleID, it.DrugID, it.DrugName, it.BatchCode, it.ExpiryDate,
			it.Unit, it.Qty, it.UnitPrice, it.Discount, it.Tax, it.Total,
		)
		if err != nil {
			return "", "", err
		}
		if it.DrugID != "" && it.Qty > 0 {
			_, _ = tx.Exec(ctx, `
				UPDATE inv_stock SET qty_available = GREATEST(0, qty_available - $1)
				WHERE drug_id=$2::uuid AND clinic_id=$3::uuid AND batch_code=$4`,
				it.Qty, it.DrugID, clinicID, it.BatchCode,
			)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return "", "", err
	}
	return saleID, invoiceNo, nil
}

func (r *Repository) DeleteSale(ctx context.Context, pool *pgxpool.Pool, clinicID, id string) error {
	_, err := pool.Exec(ctx, `DELETE FROM inv_sales WHERE id=$1::uuid AND clinic_id=$2::uuid`, id, clinicID)
	return err
}
