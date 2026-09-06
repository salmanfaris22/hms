package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/salman/hms-backend/internal/modules/lab/model"
)

type Repository struct{}

func New() *Repository { return &Repository{} }

// ErrNoTests is returned when an order is asked for with nothing on it.
var ErrNoTests = errors.New("an order needs at least one test")

// ErrAlreadyCollected is returned when a sample is collected twice.
var ErrAlreadyCollected = errors.New("this sample has already been collected")

// the worklist columns, shared by the list and the single read so a row cannot
// be drawn two different ways
const testCols = `
	t.id::text, t.order_id::text, coalesce(o.order_number, ''),
	to_char(o.ordered_at, 'YYYY-MM-DD"T"HH24:MI:SSOF:00'),
	coalesce(o.priority, 'routine'), coalesce(o.ordered_by, ''),
	o.patient_id::text,
	coalesce(NULLIF(trim(concat_ws(' ', p.first_name, p.last_name)), ''), 'Unknown patient'),
	coalesce(p.patient_number, ''), coalesce(p.photo_url, ''),
	coalesce(o.patient_age, 0), coalesce(o.patient_sex, ''),
	coalesce(t.source, 'pathology'), coalesce(t.test_id::text, ''),
	coalesce(t.test_name, ''), coalesce(t.test_code, ''), coalesce(t.category, ''),
	coalesce(t.sample_type, ''), t.price_cents, coalesce(t.tax_pct::text, '0'),
	coalesce(t.status, 'pending'),
	coalesce(to_char(t.collected_at, 'YYYY-MM-DD"T"HH24:MI:SSOF:00'), ''),
	coalesce(t.collected_by, ''), coalesce(t.result_notes, ''),
	coalesce(to_char(t.reported_at, 'YYYY-MM-DD"T"HH24:MI:SSOF:00'), ''),
	coalesce(t.reported_by, ''), coalesce(t.report_number, ''), coalesce(t.result_flag, ''),
	CASE WHEN jsonb_typeof(t.result) = 'array' THEN jsonb_array_length(t.result) ELSE 0 END`

const testFrom = `
	FROM lab_order_tests t
	JOIN lab_orders o ON o.id = t.order_id
	LEFT JOIN patients p ON p.id = o.patient_id`

func scanTest(row pgx.Row) (model.OrderTest, error) {
	var v model.OrderTest
	err := row.Scan(&v.ID, &v.OrderID, &v.OrderNumber, &v.OrderedAt, &v.Priority, &v.OrderedBy,
		&v.PatientID, &v.PatientName, &v.PatientNumber, &v.PatientPhoto, &v.PatientAge, &v.PatientSex,
		&v.Source, &v.TestID, &v.TestName, &v.TestCode, &v.Category, &v.SampleType,
		&v.PriceCents, &v.TaxPct, &v.Status, &v.CollectedAt, &v.CollectedBy,
		&v.ResultNotes, &v.ReportedAt, &v.ReportedBy, &v.ReportNumber, &v.ResultFlag,
		&v.ResultCount)
	return v, err
}

// ListTests is the worklist, filtered the way the toolbar filters it.
func (r *Repository) ListTests(
	ctx context.Context, pool *pgxpool.Pool,
	clinicID, q, category, status, priority, sample string, page, pageSize int,
) (model.OrderTestList, error) {
	out := model.OrderTestList{Tests: []model.OrderTest{}, Page: page, PageSize: pageSize}

	where := []string{"t.clinic_id = $1::uuid"}
	args := []any{clinicID}

	if q != "" {
		args = append(args, "%"+strings.ToLower(q)+"%")
		where = append(where, fmt.Sprintf(`(
			lower(concat_ws(' ', p.first_name, p.last_name)) LIKE $%d
			OR lower(coalesce(p.patient_number, '')) LIKE $%d
			OR lower(t.test_name) LIKE $%d
			OR lower(t.test_code) LIKE $%d
			OR lower(o.order_number) LIKE $%d)`,
			len(args), len(args), len(args), len(args), len(args)))
	}
	if category != "" {
		args = append(args, category)
		where = append(where, fmt.Sprintf("t.category = $%d", len(args)))
	}
	if status != "" {
		args = append(args, status)
		where = append(where, fmt.Sprintf("t.status = $%d", len(args)))
	}
	if priority != "" {
		args = append(args, priority)
		where = append(where, fmt.Sprintf("o.priority = $%d", len(args)))
	}
	switch sample {
	case "collected":
		where = append(where, "t.collected_at IS NOT NULL")
	case "pending":
		where = append(where, "t.collected_at IS NULL")
	}
	cond := " WHERE " + strings.Join(where, " AND ")

	if err := pool.QueryRow(ctx, `SELECT count(*)`+testFrom+cond, args...).Scan(&out.Total); err != nil {
		return out, err
	}

	args = append(args, pageSize, (page-1)*pageSize)
	rows, err := pool.Query(ctx,
		`SELECT`+testCols+testFrom+cond+
			fmt.Sprintf(" ORDER BY o.ordered_at DESC, t.created_at DESC LIMIT $%d OFFSET $%d",
				len(args)-1, len(args)), args...)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	for rows.Next() {
		v, err := scanTest(rows)
		if err != nil {
			return out, err
		}
		out.Tests = append(out.Tests, v)
	}
	return out, rows.Err()
}

// GetTest reads one row of the worklist.
func (r *Repository) GetTest(
	ctx context.Context, pool *pgxpool.Pool, clinicID, id string,
) (model.OrderTest, error) {
	return scanTest(pool.QueryRow(ctx,
		`SELECT`+testCols+testFrom+` WHERE t.id = $1::uuid AND t.clinic_id = $2::uuid`, id, clinicID))
}

// Summary counts the four cards. Counted from the same rows the list shows, so
// the cards and the table can never disagree.
func (r *Repository) Summary(
	ctx context.Context, pool *pgxpool.Pool, clinicID string,
) (model.Summary, error) {
	var s model.Summary
	err := pool.QueryRow(ctx, `
		SELECT
			count(*) FILTER (WHERE o.ordered_at::date = current_date),
			count(*) FILTER (WHERE t.status = 'pending'),
			count(*) FILTER (WHERE t.status = 'in_progress'),
			count(*) FILTER (WHERE t.status = 'completed')
		FROM lab_order_tests t
		JOIN lab_orders o ON o.id = t.order_id
		WHERE t.clinic_id = $1::uuid`, clinicID,
	).Scan(&s.TodayCount, &s.PendingCount, &s.InProgressCount, &s.CompletedCount)
	return s, err
}

// CreateOrder writes the order and its tests together, so an order can never
// exist with half its tests recorded.
func (r *Repository) CreateOrder(
	ctx context.Context, pool *pgxpool.Pool, req model.CreateOrderRequest,
) (model.CreateOrderResult, error) {
	var out model.CreateOrderResult
	if len(req.Tests) == 0 {
		return out, ErrNoTests
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return out, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	err = tx.QueryRow(ctx, `
		INSERT INTO lab_orders (
			clinic_id, patient_id, order_number, priority, ordered_by, ordered_by_id,
			patient_age, patient_sex, patient_category, appointment_id, notes
		)
		VALUES (
			$1::uuid, $2::uuid,
			'LAB-' || to_char(now(), 'YYYY') || '-' || lpad(
				(coalesce((SELECT count(*) FROM lab_orders WHERE clinic_id = $1::uuid), 0) + 1)::text, 4, '0'),
			$3, $4, NULLIF($5, '')::uuid,
			NULLIF($6, 0), $7, $8, NULLIF($9, '')::uuid, $10
		)
		RETURNING id::text, order_number`,
		req.ClinicID, req.PatientID, req.Priority, req.OrderedBy, req.OrderedByID,
		req.PatientAge, req.PatientSex, req.PatientCategory, req.AppointmentID, req.Notes,
	).Scan(&out.ID, &out.Number)
	if err != nil {
		return out, err
	}

	for _, t := range req.Tests {
		if _, err := tx.Exec(ctx, `
			INSERT INTO lab_order_tests (
				clinic_id, order_id, source, test_id, test_name, test_code,
				category, sample_type, price_cents, tax_pct
			)
			VALUES ($1::uuid, $2::uuid, $3, NULLIF($4, '')::uuid, $5, $6, $7, $8, $9,
			        coalesce(NULLIF($10, '')::numeric, 0))`,
			req.ClinicID, out.ID, t.Source, t.TestID, t.TestName, t.TestCode,
			t.Category, t.SampleType, t.PriceCents, t.TaxPct,
		); err != nil {
			return out, err
		}
		out.Tests++
	}

	return out, tx.Commit(ctx)
}

// Collect records who took the sample and starts the test.
func (r *Repository) Collect(
	ctx context.Context, pool *pgxpool.Pool, clinicID, testID, collectedBy string,
) (model.OrderTest, error) {
	var out model.OrderTest

	tx, err := pool.Begin(ctx)
	if err != nil {
		return out, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// locked while the state is read, so two counters cannot both take it
	var collected *string
	if err := tx.QueryRow(ctx,
		`SELECT to_char(collected_at, 'YYYY-MM-DD') FROM lab_order_tests
		 WHERE id = $1::uuid AND clinic_id = $2::uuid FOR UPDATE`, testID, clinicID,
	).Scan(&collected); err != nil {
		return out, err
	}
	if collected != nil {
		return out, ErrAlreadyCollected
	}

	if _, err := tx.Exec(ctx, `
		UPDATE lab_order_tests SET
			collected_at = now(), collected_by = $3, updated_at = now(),
			-- a sample in hand is work in progress; a result is what completes it
			status = CASE WHEN status = 'pending' THEN 'in_progress' ELSE status END
		WHERE id = $1::uuid AND clinic_id = $2::uuid`, testID, clinicID, collectedBy,
	); err != nil {
		return out, err
	}
	if err := tx.Commit(ctx); err != nil {
		return out, err
	}
	return r.GetTest(ctx, pool, clinicID, testID)
}

// ErrNotCollected is returned when a result is entered before the sample is.
var ErrNotCollected = errors.New("collect the sample before entering a result")

// SaveResult writes the values, completes the test and gives it a report
// number. The number is only assigned once, so re-reporting a corrected result
// does not renumber the report a patient already holds.
func (r *Repository) SaveResult(
	ctx context.Context, pool *pgxpool.Pool, clinicID, testID string,
	req model.SaveResultRequest, flag string,
) (model.OrderTest, error) {
	var out model.OrderTest

	// a single test reports only a flag; store an empty list rather than the
	// `null` a nil slice would marshal to
	values := req.Values
	if values == nil {
		values = []model.ResultValue{}
	}
	valueJSON, err := json.Marshal(values)
	if err != nil {
		return out, err
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return out, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var collected *string
	var number string
	if err := tx.QueryRow(ctx, `
		SELECT to_char(collected_at, 'YYYY-MM-DD'), coalesce(report_number, '')
		FROM lab_order_tests WHERE id = $1::uuid AND clinic_id = $2::uuid FOR UPDATE`,
		testID, clinicID,
	).Scan(&collected, &number); err != nil {
		return out, err
	}
	if collected == nil {
		return out, ErrNotCollected
	}

	if _, err := tx.Exec(ctx, `
		UPDATE lab_order_tests SET
			result = $3::jsonb, result_notes = $4, result_flag = $5,
			reported_by = $6, reported_at = now(), status = 'completed',
			report_number = CASE WHEN coalesce(report_number, '') = ''
				THEN 'RPT-' || to_char(now(), 'YYYY') || '-' || lpad(
					(coalesce((SELECT count(*) FROM lab_order_tests
					           WHERE clinic_id = $2::uuid AND coalesce(report_number, '') <> ''), 0) + 1)::text,
					4, '0')
				ELSE report_number END,
			updated_at = now()
		WHERE id = $1::uuid AND clinic_id = $2::uuid`,
		testID, clinicID, string(valueJSON), req.Remarks, flag, req.ReportedBy,
	); err != nil {
		return out, err
	}
	if err := tx.Commit(ctx); err != nil {
		return out, err
	}
	return r.GetTest(ctx, pool, clinicID, testID)
}

// Values reads back the parameters as they were reported.
func (r *Repository) Values(
	ctx context.Context, pool *pgxpool.Pool, clinicID, testID string,
) ([]model.ResultValue, error) {
	var raw []byte
	if err := pool.QueryRow(ctx,
		`SELECT coalesce(result, '[]'::jsonb) FROM lab_order_tests
		 WHERE id = $1::uuid AND clinic_id = $2::uuid`, testID, clinicID,
	).Scan(&raw); err != nil {
		return nil, err
	}
	out := []model.ResultValue{}
	if len(raw) > 0 {
		// an older row may hold `null`; that is no values, not an error
		_ = json.Unmarshal(raw, &out)
		if out == nil {
			out = []model.ResultValue{}
		}
	}
	return out, nil
}

// ListReports is the Reports tab (3765:52465) — the tests that have come back.
func (r *Repository) ListReports(
	ctx context.Context, pool *pgxpool.Pool, clinicID, q, source, flag string, page, pageSize int,
) (model.ReportList, error) {
	out := model.ReportList{Reports: []model.OrderTest{}, Page: page, PageSize: pageSize}

	where := []string{"t.clinic_id = $1::uuid", "t.status = 'completed'"}
	args := []any{clinicID}

	if q != "" {
		args = append(args, "%"+strings.ToLower(q)+"%")
		where = append(where, fmt.Sprintf(`(
			lower(concat_ws(' ', p.first_name, p.last_name)) LIKE $%d
			OR lower(t.test_name) LIKE $%d
			OR lower(coalesce(t.report_number, '')) LIKE $%d
			OR lower(t.category) LIKE $%d)`, len(args), len(args), len(args), len(args)))
	}
	if source != "" {
		args = append(args, source)
		where = append(where, fmt.Sprintf("t.source = $%d", len(args)))
	}
	if flag != "" {
		args = append(args, flag)
		where = append(where, fmt.Sprintf("t.result_flag = $%d", len(args)))
	}
	cond := " WHERE " + strings.Join(where, " AND ")

	if err := pool.QueryRow(ctx, `SELECT count(*)`+testFrom+cond, args...).Scan(&out.Total); err != nil {
		return out, err
	}

	args = append(args, pageSize, (page-1)*pageSize)
	rows, err := pool.Query(ctx,
		`SELECT`+testCols+testFrom+cond+
			fmt.Sprintf(" ORDER BY t.reported_at DESC NULLS LAST LIMIT $%d OFFSET $%d",
				len(args)-1, len(args)), args...)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	for rows.Next() {
		v, err := scanTest(rows)
		if err != nil {
			return out, err
		}
		out.Reports = append(out.Reports, v)
	}
	return out, rows.Err()
}
