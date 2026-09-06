package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/salman/hms-backend/internal/modules/billing/model"
)

// ErrPatientNotFound means the number on the form matches nobody in this clinic.
var ErrPatientNotFound = errors.New("patient not found")

type Repository struct{}

func New() *Repository { return &Repository{} }

/*
 * "Overdue" is not a stored status — it is a pending invoice whose due date has
 * passed. Deriving it here means a bill becomes overdue by the calendar rather
 * than by someone remembering to run a job.
 */
const statusExpr = `
	CASE
		WHEN i.status = 'draft' THEN 'draft'
		-- a proforma is a quotation, not money owed: it never becomes overdue
		-- and never counts as pending payment
		WHEN i.status = 'proforma' THEN 'proforma'
		WHEN i.amount_due_cents <= 0 THEN 'paid'
		WHEN i.due_at IS NOT NULL AND i.due_at < current_date THEN 'overdue'
		ELSE 'pending'
	END`

// ListInvoices returns one page of a clinic's invoices, newest first.
func (r *Repository) ListInvoices(
	ctx context.Context, pool *pgxpool.Pool,
	clinicID, q, status string, page, pageSize int,
) (model.InvoiceList, error) {
	out := model.InvoiceList{Invoices: []model.Invoice{}, Page: page, PageSize: pageSize}

	where := []string{"i.clinic_id = $1::uuid"}
	args := []any{clinicID}
	if q != "" {
		args = append(args, "%"+strings.ToLower(q)+"%")
		where = append(where, fmt.Sprintf(
			`(lower(i.number) LIKE $%d
			  OR lower(concat_ws(' ', p.first_name, p.last_name)) LIKE $%d
			  OR lower(p.patient_number) LIKE $%d)`, len(args), len(args), len(args)))
	}
	if status != "" {
		args = append(args, status)
		where = append(where, fmt.Sprintf("(%s) = $%d", statusExpr, len(args)))
	}
	clause := strings.Join(where, " AND ")

	from := `FROM invoices i LEFT JOIN patients p ON p.id = i.patient_id WHERE ` + clause
	if err := pool.QueryRow(ctx, `SELECT count(*) `+from, args...).Scan(&out.Total); err != nil {
		return out, err
	}

	args = append(args, pageSize, (page-1)*pageSize)
	rows, err := pool.Query(ctx, `
		SELECT i.id::text, coalesce(i.number, ''), coalesce(i.patient_id::text, ''),
		       coalesce(NULLIF(trim(concat_ws(' ', p.first_name, p.last_name)), ''), 'Unknown patient'),
		       coalesce(p.patient_number, ''), coalesce(p.photo_url, ''),
		       coalesce(to_char(i.issued_at, 'YYYY-MM-DD'), ''),
		       coalesce(to_char(i.due_at, 'YYYY-MM-DD'), ''),
		       i.amount_total_cents, i.amount_paid_cents, i.amount_due_cents,
		       `+statusExpr+`
		`+from+`
		ORDER BY i.issued_at DESC NULLS LAST, i.created_at DESC
		LIMIT $`+fmt.Sprint(len(args)-1)+` OFFSET $`+fmt.Sprint(len(args)), args...)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	for rows.Next() {
		var v model.Invoice
		if err := rows.Scan(&v.ID, &v.Number, &v.PatientID, &v.PatientName, &v.PatientNumber,
			&v.PatientPhoto, &v.IssuedAt, &v.DueAt, &v.TotalCents, &v.PaidCents, &v.DueCents,
			&v.Status); err != nil {
			return out, err
		}
		out.Invoices = append(out.Invoices, v)
	}
	return out, rows.Err()
}

// Summary totals the four cards. Counted the same way the list derives status,
// so the cards and the table can never disagree.
func (r *Repository) Summary(ctx context.Context, pool *pgxpool.Pool, clinicID string) (model.Summary, error) {
	var s model.Summary
	err := pool.QueryRow(ctx, `
		WITH scoped AS (
			SELECT i.amount_paid_cents, i.amount_due_cents, `+statusExpr+` AS derived
			FROM invoices i WHERE i.clinic_id = $1::uuid
		)
		SELECT
			coalesce(sum(amount_paid_cents), 0),
			count(*) FILTER (WHERE derived = 'paid'),
			coalesce(sum(amount_due_cents) FILTER (WHERE derived = 'pending'), 0),
			count(*) FILTER (WHERE derived = 'pending'),
			coalesce(sum(amount_due_cents) FILTER (WHERE derived = 'overdue'), 0),
			count(*) FILTER (WHERE derived = 'overdue'),
			count(*) FILTER (WHERE derived = 'draft'),
			count(*) FILTER (WHERE derived = 'proforma')
		FROM scoped`, clinicID,
	).Scan(&s.CollectedCents, &s.PaidCount, &s.PendingCents, &s.PendingCount,
		&s.OverdueCents, &s.OverdueCount, &s.DraftCount, &s.ProformaCount)
	return s, err
}

// ListPayments is the Payment History tab: what was actually taken, newest first.
func (r *Repository) ListPayments(
	ctx context.Context, pool *pgxpool.Pool, clinicID, q string, page, pageSize int,
) (model.PaymentList, error) {
	out := model.PaymentList{Payments: []model.Payment{}, Page: page, PageSize: pageSize}

	where := []string{"pay.clinic_id = $1::uuid"}
	args := []any{clinicID}
	if q != "" {
		args = append(args, "%"+strings.ToLower(q)+"%")
		where = append(where, fmt.Sprintf(
			`(lower(i.number) LIKE $%d OR lower(concat_ws(' ', p.first_name, p.last_name)) LIKE $%d)`,
			len(args), len(args)))
	}
	from := `FROM invoice_payments pay
		LEFT JOIN invoices i ON i.id = pay.invoice_id
		LEFT JOIN patients p ON p.id = pay.patient_id
		WHERE ` + strings.Join(where, " AND ")

	if err := pool.QueryRow(ctx, `SELECT count(*) `+from, args...).Scan(&out.Total); err != nil {
		return out, err
	}
	args = append(args, pageSize, (page-1)*pageSize)
	rows, err := pool.Query(ctx, `
		SELECT pay.id::text, coalesce(pay.invoice_id::text, ''), coalesce(i.number, ''),
		       coalesce(NULLIF(trim(concat_ws(' ', p.first_name, p.last_name)), ''), 'Unknown patient'),
		       pay.amount_cents, coalesce(pay.method, ''),
		       to_char(pay.created_at, 'YYYY-MM-DD"T"HH24:MI:SSOF:00')
		`+from+`
		ORDER BY pay.created_at DESC
		LIMIT $`+fmt.Sprint(len(args)-1)+` OFFSET $`+fmt.Sprint(len(args)), args...)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	for rows.Next() {
		var v model.Payment
		if err := rows.Scan(&v.ID, &v.InvoiceID, &v.InvoiceNumber, &v.PatientName,
			&v.AmountCents, &v.Method, &v.CollectedAt); err != nil {
			return out, err
		}
		out.Payments = append(out.Payments, v)
	}
	return out, rows.Err()
}

// CreateInvoice writes the bill and, where money was taken at the same time,
// the payments against it — in one transaction, so an invoice can never exist
// with half its payments recorded.
func (r *Repository) CreateInvoice(
	ctx context.Context, pool *pgxpool.Pool,
	req model.CreateInvoiceRequest, totalCents, paidCents int, lines []map[string]any,
	extraDiscount, extraTax, roundOff int,
) (model.CreateInvoiceResult, error) {
	var out model.CreateInvoiceResult

	patientID := req.PatientID
	if patientID == "" && req.PatientNumber != "" {
		// a typed bill identifies the patient by their number
		if err := pool.QueryRow(ctx,
			`SELECT id::text FROM patients WHERE clinic_id = $1::uuid AND patient_number = $2`,
			req.ClinicID, req.PatientNumber,
		).Scan(&patientID); err != nil {
			return out, fmt.Errorf("no patient with number %q: %w", req.PatientNumber, ErrPatientNotFound)
		}
	}
	if patientID == "" {
		return out, ErrPatientNotFound
	}

	lineJSON, err := json.Marshal(lines)
	if err != nil {
		return out, err
	}

	due := totalCents - paidCents
	// a bill settled in full is paid, whatever the form asked for
	status := req.Status
	if status == "pending" && due <= 0 {
		status = "paid"
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return out, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	err = tx.QueryRow(ctx, `
		INSERT INTO invoices (
			clinic_id, patient_id, appointment_id, number,
			amount_total_cents, amount_paid_cents, amount_due_cents,
			status, issued_at, due_at, line_items,
			extra_discount_cents, extra_tax_cents, round_off_cents, round_off_mode
		)
		VALUES (
			$1::uuid, $2::uuid, NULLIF($3, '')::uuid,
			'INV-' || to_char(now(), 'YYYY') || '-' || lpad(
				(coalesce((SELECT count(*) FROM invoices WHERE clinic_id = $1::uuid), 0) + 1)::text, 4, '0'),
			$4, $5, $6, $7,
			coalesce(NULLIF($8, '')::date, current_date),
			NULLIF($9, '')::date,
			$10::jsonb, $11, $12, $13, $14
		)
		RETURNING id::text, number, amount_total_cents, amount_paid_cents, amount_due_cents, status`,
		req.ClinicID, patientID, req.AppointmentID,
		totalCents, paidCents, due, status,
		req.IssuedAt, req.DueAt, string(lineJSON),
		extraDiscount, extraTax, roundOff, req.RoundOff,
	).Scan(&out.ID, &out.Number, &out.AmountTotalCents, &out.AmountPaidCents, &out.AmountDueCents, &out.Status)
	if err != nil {
		return out, err
	}

	for _, p := range req.Payments {
		if p.AmountCents <= 0 {
			continue
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO invoice_payments (clinic_id, invoice_id, patient_id, amount_cents, method, collected_by)
			VALUES ($1::uuid, $2::uuid, $3::uuid, $4, $5, NULLIF($6, '')::uuid)`,
			req.ClinicID, out.ID, patientID, p.AmountCents, p.Method, req.CollectedBy,
		); err != nil {
			return out, err
		}
	}

	return out, tx.Commit(ctx)
}

// GetInvoice reads one invoice with its lines and the payments against it.
func (r *Repository) GetInvoice(
	ctx context.Context, pool *pgxpool.Pool, clinicID, id string,
) (model.InvoiceDetail, error) {
	var out model.InvoiceDetail
	var lineRaw []byte

	err := pool.QueryRow(ctx, `
		SELECT i.id::text, coalesce(i.number, ''), coalesce(i.patient_id::text, ''),
		       coalesce(NULLIF(trim(concat_ws(' ', p.first_name, p.last_name)), ''), 'Unknown patient'),
		       coalesce(p.patient_number, ''), coalesce(p.photo_url, ''),
		       coalesce(to_char(i.issued_at, 'YYYY-MM-DD'), ''),
		       coalesce(to_char(i.due_at, 'YYYY-MM-DD'), ''),
		       i.amount_total_cents, i.amount_paid_cents, i.amount_due_cents,
		       `+statusExpr+`, coalesce(i.line_items, '[]'::jsonb),
		       i.extra_discount_cents, i.extra_tax_cents, i.round_off_cents,
		       coalesce(i.round_off_mode, 'none'), coalesce(i.appointment_id::text, '')
		FROM invoices i
		LEFT JOIN patients p ON p.id = i.patient_id
		WHERE i.id = $1::uuid AND i.clinic_id = $2::uuid`, id, clinicID,
	).Scan(&out.ID, &out.Number, &out.PatientID, &out.PatientName, &out.PatientNumber,
		&out.PatientPhoto, &out.IssuedAt, &out.DueAt, &out.TotalCents, &out.PaidCents,
		&out.DueCents, &out.Status, &lineRaw,
		&out.ExtraDiscountCents, &out.ExtraTaxCents, &out.RoundOffCents,
		&out.RoundOffMode, &out.AppointmentID)
	if err != nil {
		return out, err
	}

	out.Lines = []model.InvoiceLine{}
	if len(lineRaw) > 0 {
		_ = json.Unmarshal(lineRaw, &out.Lines)
	}
	// the totals the card breaks out, from the lines as billed
	for _, l := range out.Lines {
		gross := l.Qty * float64(l.RateCents)
		discount := gross * (l.DiscountPct / 100)
		net := gross - discount
		out.SubtotalCents += int(gross + 0.5)
		out.TotalDiscountCents += int(discount + 0.5)
		out.TaxableCents += int(net + 0.5)
		out.TaxCents += int(net*(l.TaxPct/100) + 0.5)
	}
	// the whole-bill extras belong to the same two lines on the card
	out.TotalDiscountCents += out.ExtraDiscountCents
	out.TaxCents += out.ExtraTaxCents

	out.Payments = []model.Payment{}
	rows, err := pool.Query(ctx, `
		SELECT pay.id::text, coalesce(pay.invoice_id::text, ''), $2,
		       coalesce(NULLIF(trim(concat_ws(' ', p.first_name, p.last_name)), ''), ''),
		       pay.amount_cents, coalesce(pay.method, ''),
		       to_char(pay.created_at, 'YYYY-MM-DD"T"HH24:MI:SSOF:00')
		FROM invoice_payments pay
		LEFT JOIN patients p ON p.id = pay.patient_id
		WHERE pay.invoice_id = $1::uuid
		ORDER BY pay.created_at DESC`, id, out.Number)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	for rows.Next() {
		var v model.Payment
		if err := rows.Scan(&v.ID, &v.InvoiceID, &v.InvoiceNumber, &v.PatientName,
			&v.AmountCents, &v.Method, &v.CollectedAt); err != nil {
			return out, err
		}
		out.Payments = append(out.Payments, v)
	}
	if err := rows.Err(); err != nil {
		return out, err
	}

	out.Refunds = []model.Refund{}
	rRows, err := pool.Query(ctx, `
		SELECT id::text, invoice_id::text, amount_cents, coalesce(method, ''), coalesce(note, ''),
		       to_char(created_at, 'YYYY-MM-DD"T"HH24:MI:SSOF:00')
		FROM invoice_refunds WHERE invoice_id = $1::uuid
		ORDER BY created_at DESC`, id)
	if err != nil {
		return out, err
	}
	defer rRows.Close()
	for rRows.Next() {
		var v model.Refund
		if err := rRows.Scan(&v.ID, &v.InvoiceID, &v.AmountCents, &v.Method, &v.Note, &v.CreatedAt); err != nil {
			return out, err
		}
		out.Refunds = append(out.Refunds, v)
		out.RefundedCents += v.AmountCents
	}
	return out, rRows.Err()
}

// ErrNothingDue is returned when a settled invoice is paid again.
var ErrNothingDue = errors.New("this invoice is already settled")

// ErrPaymentTooLarge is returned when the splits exceed the balance.
var ErrPaymentTooLarge = errors.New("payment is more than the balance due")

// RecordPayment books money against an existing invoice and moves the
// invoice's own totals with it, in one transaction so the two can never drift.
func (r *Repository) RecordPayment(
	ctx context.Context, pool *pgxpool.Pool, clinicID, invoiceID string, req model.PaymentRequest,
) (model.CreateInvoiceResult, error) {
	var out model.CreateInvoiceResult

	total := 0
	for _, p := range req.Payments {
		if p.AmountCents > 0 {
			total += p.AmountCents
		}
	}
	if total <= 0 {
		return out, ErrNothingDue
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return out, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// the row is locked while the balance is read, so two counters cannot both
	// take the last rupee
	var patientID string
	var due int
	if err := tx.QueryRow(ctx, `
		SELECT patient_id::text, amount_due_cents FROM invoices
		WHERE id = $1::uuid AND clinic_id = $2::uuid FOR UPDATE`,
		invoiceID, clinicID,
	).Scan(&patientID, &due); err != nil {
		return out, err
	}
	if due <= 0 {
		return out, ErrNothingDue
	}
	if total > due {
		return out, ErrPaymentTooLarge
	}

	for _, p := range req.Payments {
		if p.AmountCents <= 0 {
			continue
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO invoice_payments (clinic_id, invoice_id, patient_id, amount_cents, method, collected_by)
			VALUES ($1::uuid, $2::uuid, $3::uuid, $4, $5, NULLIF($6, '')::uuid)`,
			clinicID, invoiceID, patientID, p.AmountCents, p.Method, req.CollectedBy,
		); err != nil {
			return out, err
		}
	}

	err = tx.QueryRow(ctx, `
		UPDATE invoices SET
			amount_paid_cents = amount_paid_cents + $3,
			amount_due_cents  = amount_due_cents - $3,
			-- a bill settled in full is paid; a draft or proforma keeps its kind
			status = CASE WHEN status = 'pending' AND amount_due_cents - $3 <= 0
			              THEN 'paid' ELSE status END
		WHERE id = $1::uuid AND clinic_id = $2::uuid
		RETURNING id::text, number, amount_total_cents, amount_paid_cents, amount_due_cents, status`,
		invoiceID, clinicID, total,
	).Scan(&out.ID, &out.Number, &out.AmountTotalCents, &out.AmountPaidCents, &out.AmountDueCents, &out.Status)
	if err != nil {
		return out, err
	}
	return out, tx.Commit(ctx)
}

// ErrNothingPaid is returned when a refund is asked for on an unpaid invoice.
var ErrNothingPaid = errors.New("there is nothing paid on this invoice to refund")

// ErrRefundTooLarge is returned when the refund exceeds what was taken.
var ErrRefundTooLarge = errors.New("refund is more than the net paid")

// RecordRefund hands money back and reverses it on the invoice in one
// transaction, so `total = paid + due` still holds afterwards.
func (r *Repository) RecordRefund(
	ctx context.Context, pool *pgxpool.Pool, invoiceID string, req model.RefundRequest,
) (model.CreateInvoiceResult, error) {
	var out model.CreateInvoiceResult
	if req.AmountCents <= 0 {
		return out, ErrRefundTooLarge
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return out, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var patientID string
	var paid int
	if err := tx.QueryRow(ctx, `
		SELECT patient_id::text, amount_paid_cents FROM invoices
		WHERE id = $1::uuid AND clinic_id = $2::uuid FOR UPDATE`,
		invoiceID, req.ClinicID,
	).Scan(&patientID, &paid); err != nil {
		return out, err
	}
	if paid <= 0 {
		return out, ErrNothingPaid
	}
	if req.AmountCents > paid {
		return out, ErrRefundTooLarge
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO invoice_refunds (clinic_id, invoice_id, patient_id, amount_cents, method, note, refunded_by)
		VALUES ($1::uuid, $2::uuid, $3::uuid, $4, $5, $6, NULLIF($7, '')::uuid)`,
		req.ClinicID, invoiceID, patientID, req.AmountCents, req.Method, req.Note, req.RefundedBy,
	); err != nil {
		return out, err
	}

	err = tx.QueryRow(ctx, `
		UPDATE invoices SET
			amount_paid_cents = amount_paid_cents - $3,
			amount_due_cents  = amount_due_cents + $3,
			-- money given back is money owed again; a settled bill reopens
			status = CASE WHEN status = 'paid' THEN 'pending' ELSE status END
		WHERE id = $1::uuid AND clinic_id = $2::uuid
		RETURNING id::text, number, amount_total_cents, amount_paid_cents, amount_due_cents, status`,
		invoiceID, req.ClinicID, req.AmountCents,
	).Scan(&out.ID, &out.Number, &out.AmountTotalCents, &out.AmountPaidCents, &out.AmountDueCents, &out.Status)
	if err != nil {
		return out, err
	}
	return out, tx.Commit(ctx)
}

// ErrNoDraft is returned when a visit has no draft bill waiting.
var ErrNoDraft = errors.New("no draft bill for this appointment")

// ErrNotDraft is returned when an invoice that has already been issued is
// edited. A draft is a work in progress; a raised bill is a record.
var ErrNotDraft = errors.New("only a draft can be edited")

// DraftForAppointment finds the unfinished bill a visit already has, so
// Generate Bill reopens it instead of starting a second one.
func (r *Repository) DraftForAppointment(
	ctx context.Context, pool *pgxpool.Pool, clinicID, appointmentID string,
) (string, error) {
	var id string
	err := pool.QueryRow(ctx, `
		SELECT id::text FROM invoices
		WHERE clinic_id = $1::uuid AND appointment_id = $2::uuid AND status = 'draft'
		ORDER BY created_at DESC
		LIMIT 1`, clinicID, appointmentID).Scan(&id)
	if err != nil {
		return "", ErrNoDraft
	}
	return id, nil
}

// UpdateInvoice rewrites a draft in place — same invoice, same number, the
// figures the form now says.
func (r *Repository) UpdateInvoice(
	ctx context.Context, pool *pgxpool.Pool, id string,
	req model.CreateInvoiceRequest, totalCents, paidCents int, lines []map[string]any,
	extraDiscount, extraTax, roundOff int,
) (model.CreateInvoiceResult, error) {
	var out model.CreateInvoiceResult

	lineJSON, err := json.Marshal(lines)
	if err != nil {
		return out, err
	}
	due := totalCents - paidCents
	status := req.Status
	if status == "pending" && due <= 0 {
		status = "paid"
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return out, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	err = tx.QueryRow(ctx, `
		UPDATE invoices SET
			amount_total_cents = $3, amount_paid_cents = $4, amount_due_cents = $5,
			status = $6,
			issued_at = coalesce(NULLIF($7, '')::date, issued_at),
			due_at = NULLIF($8, '')::date,
			line_items = $9::jsonb,
			extra_discount_cents = $10, extra_tax_cents = $11,
			round_off_cents = $12, round_off_mode = $13
		WHERE id = $1::uuid AND clinic_id = $2::uuid AND status = 'draft'
		RETURNING id::text, number, amount_total_cents, amount_paid_cents, amount_due_cents, status`,
		id, req.ClinicID, totalCents, paidCents, due, status,
		req.IssuedAt, req.DueAt, string(lineJSON),
		extraDiscount, extraTax, roundOff, req.RoundOff,
	).Scan(&out.ID, &out.Number, &out.AmountTotalCents, &out.AmountPaidCents, &out.AmountDueCents, &out.Status)
	if errors.Is(err, pgx.ErrNoRows) {
		return out, ErrNotDraft
	}
	if err != nil {
		return out, err
	}

	// money taken as the draft is finalised is a payment like any other, and
	// belongs in the ledger rather than only in the invoice's own total
	if paidCents > 0 {
		var patientID string
		if err := tx.QueryRow(ctx,
			`SELECT patient_id::text FROM invoices WHERE id = $1::uuid`, id,
		).Scan(&patientID); err != nil {
			return out, err
		}
		for _, p := range req.Payments {
			if p.AmountCents <= 0 {
				continue
			}
			if _, err := tx.Exec(ctx, `
				INSERT INTO invoice_payments (clinic_id, invoice_id, patient_id, amount_cents, method, collected_by)
				VALUES ($1::uuid, $2::uuid, $3::uuid, $4, $5, NULLIF($6, '')::uuid)`,
				req.ClinicID, id, patientID, p.AmountCents, p.Method, req.CollectedBy,
			); err != nil {
				return out, err
			}
		}
	}
	return out, tx.Commit(ctx)
}
