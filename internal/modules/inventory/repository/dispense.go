package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/salman/hms-backend/internal/modules/inventory/model"
)

func (r *Repository) DispenseSummary(ctx context.Context, pool *pgxpool.Pool, clinicID string) (model.DispenseSummary, error) {
	var s model.DispenseSummary
	err := pool.QueryRow(ctx, `
		SELECT
		  COUNT(*),
		  COUNT(*) FILTER (WHERE created_at::date = CURRENT_DATE),
		  COUNT(*) FILTER (WHERE status='Pending'),
		  COALESCE((SELECT SUM(qty_total) FROM inv_dispense_items di
		            JOIN inv_dispenses d2 ON d2.id=di.dispense_id
		            WHERE d2.clinic_id=$1::uuid),0)
		FROM inv_dispenses WHERE clinic_id=$1::uuid`, clinicID,
	).Scan(&s.TotalDispensed, &s.TodayDispensed, &s.Pending, &s.TotalUnitsOut)
	return s, err
}

func (r *Repository) ListDispenses(ctx context.Context, pool *pgxpool.Pool, f model.DispenseListFilter) ([]model.DispenseDTO, int, error) {
	offset := (f.Page - 1) * DefaultLimit
	rows, err := pool.Query(ctx, `
		SELECT d.id, d.dispense_no, d.patient_name, d.issued_to_dept, d.prescribed_by,
		       d.issued_date::text, d.status, COUNT(di.id) AS item_count, d.created_at::text
		FROM inv_dispenses d
		LEFT JOIN inv_dispense_items di ON di.dispense_id=d.id
		WHERE d.clinic_id=$1::uuid
		  AND ($2='' OR d.dispense_no ILIKE '%'||$2||'%' OR d.patient_name ILIKE '%'||$2||'%')
		  AND ($3='' OR d.status=$3)
		GROUP BY d.id ORDER BY d.created_at DESC
		LIMIT $4 OFFSET $5`,
		f.ClinicID, f.Q, f.StatusFilter, DefaultLimit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	dispenses := []model.DispenseDTO{}
	for rows.Next() {
		var d model.DispenseDTO
		if err := rows.Scan(&d.ID, &d.DispenseNo, &d.PatientName, &d.IssuedToDept,
			&d.PrescribedBy, &d.IssuedDate, &d.Status, &d.ItemCount, &d.CreatedAt); err == nil {
			dispenses = append(dispenses, d)
		}
	}
	var total int
	_ = pool.QueryRow(ctx, `SELECT COUNT(*) FROM inv_dispenses WHERE clinic_id=$1::uuid AND ($2='' OR patient_name ILIKE '%'||$2||'%') AND ($3='' OR status=$3)`,
		f.ClinicID, f.Q, f.StatusFilter).Scan(&total)
	return dispenses, total, nil
}

func (r *Repository) CreateDispense(ctx context.Context, pool *pgxpool.Pool, clinicID string, req model.CreateDispenseRequest) (string, string, error) {
	var cnt int
	_ = pool.QueryRow(ctx, `SELECT COUNT(*) FROM inv_dispenses WHERE clinic_id=$1::uuid`, clinicID).Scan(&cnt)
	dispenseNo := fmt.Sprintf("DSP-%d-%03d", time.Now().Year(), cnt+1)

	issuedDate := req.IssuedDate
	if issuedDate == "" {
		issuedDate = time.Now().Format("2006-01-02")
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return "", "", err
	}
	defer tx.Rollback(ctx)

	var dispID string
	err = tx.QueryRow(ctx, `
		INSERT INTO inv_dispenses (clinic_id, dispense_no, patient_name, issued_to_dept, prescribed_by, issued_date, notes)
		VALUES ($1::uuid,$2,$3,$4,$5,$6::date,$7) RETURNING id`,
		clinicID, dispenseNo, req.PatientName, req.IssuedToDept, req.PrescribedBy, issuedDate, req.Notes,
	).Scan(&dispID)
	if err != nil {
		return "", "", err
	}

	for _, it := range req.Items {
		_, err = tx.Exec(ctx, `
			INSERT INTO inv_dispense_items (dispense_id, drug_id, drug_name, unit, qty_prescribed, qty_total)
			VALUES ($1::uuid, NULLIF($2,'')::uuid, $3, $4, $5, $6)`,
			dispID, it.DrugID, it.DrugName, it.Unit, it.QtyPrescribed, it.QtyTotal,
		)
		if err != nil {
			return "", "", err
		}
		if it.DrugID != "" && it.QtyTotal > 0 {
			_, _ = tx.Exec(ctx, `
				UPDATE inv_stock SET qty_available = GREATEST(0, qty_available - $1)
				WHERE drug_id=$2::uuid AND clinic_id=$3::uuid
				  AND qty_available > 0 LIMIT 1`,
				it.QtyTotal, it.DrugID, clinicID,
			)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return "", "", err
	}
	return dispID, dispenseNo, nil
}

func (r *Repository) UpdateDispenseStatus(ctx context.Context, pool *pgxpool.Pool, clinicID, id, status string) error {
	_, err := pool.Exec(ctx, `UPDATE inv_dispenses SET status=$1 WHERE id=$2::uuid AND clinic_id=$3::uuid`, status, id, clinicID)
	return err
}
