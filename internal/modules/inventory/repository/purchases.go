package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/salman/hms-backend/internal/modules/inventory/model"
)

func (r *Repository) ListPurchases(ctx context.Context, pool *pgxpool.Pool, f model.PurchaseListFilter) ([]model.PurchaseDTO, int, error) {
	offset := (f.Page - 1) * DefaultLimit
	rows, err := pool.Query(ctx, `
		SELECT p.id, p.purchase_id, p.invoice_no, p.supplier_name, p.stock_point,
		       p.payment_method, p.grand_total, COUNT(pi.id) AS item_count, p.created_at::text
		FROM inv_purchases p
		LEFT JOIN inv_purchase_items pi ON pi.purchase_id=p.id
		WHERE p.clinic_id=$1::uuid
		  AND ($2='' OR p.purchase_id ILIKE '%'||$2||'%' OR p.supplier_name ILIKE '%'||$2||'%')
		GROUP BY p.id ORDER BY p.created_at DESC
		LIMIT $3 OFFSET $4`,
		f.ClinicID, f.Q, DefaultLimit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	purchases := []model.PurchaseDTO{}
	for rows.Next() {
		var p model.PurchaseDTO
		if err := rows.Scan(&p.ID, &p.PurchaseID, &p.InvoiceNo, &p.SupplierName,
			&p.StockPoint, &p.PaymentMethod, &p.GrandTotal, &p.ItemCount, &p.CreatedAt); err == nil {
			purchases = append(purchases, p)
		}
	}
	var total int
	_ = pool.QueryRow(ctx, `SELECT COUNT(*) FROM inv_purchases WHERE clinic_id=$1::uuid AND ($2='' OR purchase_id ILIKE '%'||$2||'%' OR supplier_name ILIKE '%'||$2||'%')`,
		f.ClinicID, f.Q).Scan(&total)
	return purchases, total, nil
}

func (r *Repository) CreatePurchase(ctx context.Context, pool *pgxpool.Pool, clinicID string, req model.CreatePurchaseRequest) (string, string, error) {
	var cnt int
	_ = pool.QueryRow(ctx, `SELECT COUNT(*) FROM inv_purchases WHERE clinic_id=$1::uuid`, clinicID).Scan(&cnt)
	purchaseID := fmt.Sprintf("PUR-%04d", cnt+1)
	if req.StockPoint == "" {
		req.StockPoint = "Main Store"
	}
	if req.PaymentMethod == "" {
		req.PaymentMethod = "Cash"
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return "", "", err
	}
	defer tx.Rollback(ctx)

	var purID string
	err = tx.QueryRow(ctx, `
		INSERT INTO inv_purchases (clinic_id, purchase_id, invoice_no, supplier_name, stock_point,
		    payment_method, subtotal, discount_pct, gst_enabled, round_off, shipping, grand_total, notes)
		VALUES ($1::uuid,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13) RETURNING id`,
		clinicID, purchaseID, req.InvoiceNo, req.SupplierName, req.StockPoint,
		req.PaymentMethod, req.Subtotal, req.DiscountPct, req.GstEnabled,
		req.RoundOff, req.Shipping, req.GrandTotal, req.Notes,
	).Scan(&purID)
	if err != nil {
		return "", "", err
	}

	for _, it := range req.Items {
		_, err = tx.Exec(ctx, `
			INSERT INTO inv_purchase_items (purchase_id, drug_id, drug_name, batch_code, expiry_date,
			    qty_purchased, qty_total, purchase_rate, gst_pct, free_qty, mrp, pack_size, discount_pct, amount)
			VALUES ($1::uuid, NULLIF($2,'')::uuid, $3, $4, $5::date, $6, $7, $8, $9, $10, $11, $12, $13, $14)`,
			purID, it.DrugID, it.DrugName, it.BatchCode, it.ExpiryDate,
			it.QtyPurchased, it.QtyTotal, it.PurchaseRate, it.GstPct,
			it.FreeQty, it.MRP, it.PackSize, it.DiscountPct, it.Amount,
		)
		if err != nil {
			return "", "", err
		}
		if it.DrugID != "" && it.QtyPurchased > 0 {
			_, _ = tx.Exec(ctx, `
				INSERT INTO inv_stock (clinic_id, drug_id, batch_code, expiry_date, qty_available, purchase_rate, mrp)
				VALUES ($1::uuid, $2::uuid, $3, $4::date, $5, $6, $7)
				ON CONFLICT DO NOTHING`,
				clinicID, it.DrugID, it.BatchCode, it.ExpiryDate, it.QtyPurchased, it.PurchaseRate, it.MRP,
			)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return "", "", err
	}
	return purID, purchaseID, nil
}

func (r *Repository) DeletePurchase(ctx context.Context, pool *pgxpool.Pool, clinicID, id string) error {
	_, err := pool.Exec(ctx, `DELETE FROM inv_purchases WHERE id=$1::uuid AND clinic_id=$2::uuid`, id, clinicID)
	return err
}
