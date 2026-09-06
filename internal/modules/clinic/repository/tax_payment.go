package repository

import (
	"context"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/salman/hms-backend/internal/modules/clinic/model"
)

func (r *Repository) CountTaxes(ctx context.Context, pool *pgxpool.Pool, clinicID string) (int, error) {
	var total int
	err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM tax_types WHERE clinic_id = $1::uuid`, clinicID).Scan(&total)
	return total, err
}

func (r *Repository) ListTaxes(ctx context.Context, pool *pgxpool.Pool, clinicID string, limit, offset int) ([]model.TaxDTO, error) {
	rows, err := pool.Query(ctx, `
		SELECT id::text, name, rate, enabled
		FROM tax_types
		WHERE clinic_id = $1::uuid
		ORDER BY created_at ASC
		LIMIT $2 OFFSET $3`,
		clinicID, limit, offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.TaxDTO{}
	for rows.Next() {
		var t model.TaxDTO
		if err := rows.Scan(&t.ID, &t.Name, &t.Rate, &t.Enabled); err == nil {
			out = append(out, t)
		}
	}
	return out, nil
}

func (r *Repository) CreateTax(ctx context.Context, pool *pgxpool.Pool, clinicID, name string, rate float64) (string, error) {
	var id string
	err := pool.QueryRow(ctx, `
		INSERT INTO tax_types (clinic_id, name, rate, enabled)
		VALUES ($1::uuid, $2, $3, true)
		RETURNING id::text`,
		clinicID, name, rate,
	).Scan(&id)
	return id, err
}

func (r *Repository) UpdateTax(ctx context.Context, pool *pgxpool.Pool, clinicID, taxID string, req model.UpdateTaxRequest) (bool, error) {
	sets := []string{}
	args := []any{}
	if req.Enabled != nil {
		args = append(args, *req.Enabled)
		sets = append(sets, "enabled = $"+strconv.Itoa(len(args)))
	}
	if req.Name != nil {
		args = append(args, strings.TrimSpace(*req.Name))
		sets = append(sets, "name = $"+strconv.Itoa(len(args)))
	}
	if req.Rate != nil {
		args = append(args, *req.Rate)
		sets = append(sets, "rate = $"+strconv.Itoa(len(args)))
	}
	if len(sets) == 0 {
		return false, nil
	}
	args = append(args, taxID, clinicID)
	q := "UPDATE tax_types SET " + strings.Join(sets, ", ") +
		" WHERE id = $" + strconv.Itoa(len(args)-1) + "::uuid" +
		" AND clinic_id = $" + strconv.Itoa(len(args)) + "::uuid"
	_, err := pool.Exec(ctx, q, args...)
	return true, err
}

func (r *Repository) DeleteTax(ctx context.Context, pool *pgxpool.Pool, clinicID, taxID string) error {
	_, err := pool.Exec(ctx,
		`DELETE FROM tax_types WHERE id = $1::uuid AND clinic_id = $2::uuid`,
		taxID, clinicID,
	)
	return err
}

func (r *Repository) CountPaymentModes(ctx context.Context, pool *pgxpool.Pool, clinicID string) (int, error) {
	var total int
	err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM payment_modes WHERE clinic_id = $1::uuid`, clinicID).Scan(&total)
	return total, err
}

func (r *Repository) ListPaymentModes(ctx context.Context, pool *pgxpool.Pool, clinicID string, limit, offset int) ([]model.PaymentModeDTO, error) {
	rows, err := pool.Query(ctx, `
		SELECT id::text, name
		FROM payment_modes
		WHERE clinic_id = $1::uuid
		ORDER BY created_at ASC
		LIMIT $2 OFFSET $3`,
		clinicID, limit, offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.PaymentModeDTO{}
	for rows.Next() {
		var m model.PaymentModeDTO
		if err := rows.Scan(&m.ID, &m.Name); err == nil {
			out = append(out, m)
		}
	}
	return out, nil
}

func (r *Repository) CreatePaymentMode(ctx context.Context, pool *pgxpool.Pool, clinicID, name string) (string, error) {
	var id string
	err := pool.QueryRow(ctx, `
		INSERT INTO payment_modes (clinic_id, name)
		VALUES ($1::uuid, $2)
		RETURNING id::text`,
		clinicID, name,
	).Scan(&id)
	return id, err
}

func (r *Repository) DeletePaymentMode(ctx context.Context, pool *pgxpool.Pool, clinicID, modeID string) error {
	_, err := pool.Exec(ctx,
		`DELETE FROM payment_modes WHERE id = $1::uuid AND clinic_id = $2::uuid`,
		modeID, clinicID,
	)
	return err
}
