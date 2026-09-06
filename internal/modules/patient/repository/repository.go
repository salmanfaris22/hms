package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/salman/hms-backend/internal/infrastructure/persistence/postgres"
	"github.com/salman/hms-backend/internal/modules/patient/model"
)

type Repository struct {
	tenants *postgres.TenantResolver
}

func New(tenants *postgres.TenantResolver) *Repository { return &Repository{tenants: tenants} }

func (r *Repository) Tenants() *postgres.TenantResolver { return r.tenants }
func (r *Repository) Pool(ctx context.Context, tenantID string) (*pgxpool.Pool, error) {
	return r.tenants.Pool(ctx, tenantID)
}

const Columns = `
	id::text, clinic_id::text, patient_number,
	first_name, middle_name, last_name, date_of_birth, sex,
	blood_type, marital_status, photo_url,
	phones, email, address_line, city, state, country, zip_code,
	emergency_name, emergency_phone,
	allergies, medical_conditions,
	insurance_provider, insurance_id,
	condition, last_visit_at, status, notes, family_members,
	created_at, updated_at,
	source, insurance_validity,
	category_id::text, special_status_id::text,
	-- sub-selects rather than joins: Columns is unqualified and shared with the
	-- single-row read, and both catalogues have id/clinic_id/created_at of their own
	COALESCE((SELECT c.name FROM patient_categories c WHERE c.id = category_id), ''),
	COALESCE((SELECT ss.name FROM special_statuses ss WHERE ss.id = special_status_id), '')
`

type RowScanner interface{ Scan(dest ...any) error }

func ScanPatient(row RowScanner) (model.PatientDTO, error) {
	var p model.PatientDTO
	var phonesRaw, familyRaw []byte
	err := row.Scan(
		&p.ID, &p.ClinicID, &p.PatientNumber,
		&p.FirstName, &p.MiddleName, &p.LastName, &p.DateOfBirth, &p.Sex,
		&p.BloodType, &p.MaritalStatus, &p.PhotoURL,
		&phonesRaw, &p.Email, &p.AddressLine, &p.City, &p.State, &p.Country, &p.ZipCode,
		&p.EmergencyName, &p.EmergencyPhone,
		&p.Allergies, &p.MedicalConditions,
		&p.InsuranceProvider, &p.InsuranceID,
		&p.Condition, &p.LastVisitAt, &p.Status, &p.Notes, &familyRaw,
		&p.CreatedAt, &p.UpdatedAt,
		&p.Source, &p.InsuranceValidity,
		&p.CategoryID, &p.SpecialStatusID, &p.CategoryName, &p.SpecialStatusName,
	)
	if err != nil {
		return p, err
	}
	if len(phonesRaw) > 0 {
		_ = json.Unmarshal(phonesRaw, &p.Phones)
	}
	if p.Phones == nil {
		p.Phones = []model.PatientPhone{}
	}
	if len(familyRaw) > 0 {
		_ = json.Unmarshal(familyRaw, &p.FamilyMembers)
	}
	if p.FamilyMembers == nil {
		p.FamilyMembers = []model.FamilyMember{}
	}
	return p, nil
}

func (r *Repository) AssertMember(ctx context.Context, pool *pgxpool.Pool, userID, clinicID string) error {
	var one int
	err := pool.QueryRow(ctx,
		`SELECT 1 FROM clinic_members WHERE clinic_id = $1::uuid AND user_id = $2::uuid`,
		clinicID, userID,
	).Scan(&one)
	return err
}

func (r *Repository) AssertPatientMember(ctx context.Context, pool *pgxpool.Pool, userID, patientID string) error {
	var one int
	return pool.QueryRow(ctx, `
		SELECT 1 FROM patients p
		JOIN clinic_members m ON m.clinic_id = p.clinic_id
		WHERE p.id = $1::uuid AND m.user_id = $2::uuid`,
		patientID, userID,
	).Scan(&one)
}

func (r *Repository) List(ctx context.Context, pool *pgxpool.Pool, f model.ListFilter) ([]model.PatientDTO, model.Stats, int, error) {
	args := []any{f.ClinicID}
	where := "p.clinic_id = $1::uuid"
	if f.Status == "active" || f.Status == "inactive" {
		args = append(args, f.Status)
		where += fmt.Sprintf(" AND status = $%d", len(args))
	}
	if f.Search != "" {
		args = append(args, "%"+strings.ToLower(f.Search)+"%")
		i := len(args)
		where += fmt.Sprintf(
			" AND (lower(first_name) LIKE $%d OR lower(last_name) LIKE $%d OR lower(email) LIKE $%d OR lower(patient_number) LIKE $%d)",
			i, i, i, i,
		)
	}

	args = append(args, f.PageSize, (f.Page-1)*f.PageSize)
	limitIdx := len(args) - 1
	offsetIdx := len(args)

	query := fmt.Sprintf(`
		WITH stats AS (
			SELECT
				count(*) FILTER (WHERE TRUE) as total,
				count(*) FILTER (WHERE status = 'active') as active,
				count(*) FILTER (WHERE status = 'inactive') as inactive,
				count(*) FILTER (WHERE created_at >= date_trunc('month', now())) as new_this_month
			FROM patients WHERE clinic_id = $1::uuid
		)
		SELECT
			s.total, s.active, s.inactive, s.new_this_month,
			(SELECT count(*) FROM patients WHERE %s) as filtered_count,
			%s
		FROM stats s
		CROSS JOIN patients p
		WHERE %s
		ORDER BY p.created_at DESC
		LIMIT $%d OFFSET $%d`,
		where, Columns, where, limitIdx, offsetIdx,
	)

	rows, err := pool.Query(ctx, query, args...)
	if err != nil {
		return nil, model.Stats{}, 0, err
	}
	defer rows.Close()

	var st model.Stats
	var total int
	out := []model.PatientDTO{}
	for rows.Next() {
		var p model.PatientDTO
		var phonesRaw, familyRaw []byte
		err := rows.Scan(
			&st.Total, &st.Active, &st.Inactive, &st.NewThisMonth, &total,
			&p.ID, &p.ClinicID, &p.PatientNumber,
			&p.FirstName, &p.MiddleName, &p.LastName, &p.DateOfBirth, &p.Sex,
			&p.BloodType, &p.MaritalStatus, &p.PhotoURL,
			&phonesRaw, &p.Email, &p.AddressLine, &p.City, &p.State, &p.Country, &p.ZipCode,
			&p.EmergencyName, &p.EmergencyPhone,
			&p.Allergies, &p.MedicalConditions,
			&p.InsuranceProvider, &p.InsuranceID,
			&p.Condition, &p.LastVisitAt, &p.Status, &p.Notes, &familyRaw,
			&p.CreatedAt, &p.UpdatedAt,
			&p.Source, &p.InsuranceValidity,
			&p.CategoryID, &p.SpecialStatusID, &p.CategoryName, &p.SpecialStatusName,
		)
		if err != nil {
			return nil, model.Stats{}, 0, err
		}
		if len(phonesRaw) > 0 {
			_ = json.Unmarshal(phonesRaw, &p.Phones)
		}
		if p.Phones == nil {
			p.Phones = []model.PatientPhone{}
		}
		if len(familyRaw) > 0 {
			_ = json.Unmarshal(familyRaw, &p.FamilyMembers)
		}
		if p.FamilyMembers == nil {
			p.FamilyMembers = []model.FamilyMember{}
		}
		out = append(out, p)
	}
	return out, st, total, nil
}

func (r *Repository) Get(ctx context.Context, pool *pgxpool.Pool, id, userID string) (model.PatientDTO, error) {
	row := pool.QueryRow(ctx, `
		SELECT `+Columns+` FROM patients
		WHERE id = $1::uuid
		  AND clinic_id IN (SELECT clinic_id FROM clinic_members WHERE user_id = $2::uuid)`,
		id, userID,
	)
	return ScanPatient(row)
}

func (r *Repository) NextPatientNumber(ctx context.Context, pool *pgxpool.Pool) (int, error) {
	var seq int
	err := pool.QueryRow(ctx, `SELECT nextval('patient_number_seq')`).Scan(&seq)
	return seq, err
}

// PatientNumberExists reports whether a clinic already uses this UHID. Only
// the form's Manual Entry path can collide; the generated PA###### cannot.
// There is no unique index to lean on — adding one would abort boot for any
// tenant whose existing rows already collide, since migrations re-run on start.
func (r *Repository) PatientNumberExists(ctx context.Context, pool *pgxpool.Pool, clinicID, number string) (bool, error) {
	var exists bool
	err := pool.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM patients WHERE clinic_id = $1::uuid AND patient_number = $2)`,
		clinicID, number).Scan(&exists)
	return exists, err
}

func (r *Repository) Create(ctx context.Context, pool *pgxpool.Pool, patientNumber string, req model.CreateRequest) (string, error) {
	phones, _ := json.Marshal(req.Phones)
	family, _ := json.Marshal(req.FamilyMembers)
	var id string
	err := pool.QueryRow(ctx, `
		INSERT INTO patients (
			clinic_id, patient_number,
			first_name, middle_name, last_name, date_of_birth, sex,
			blood_type, marital_status, photo_url,
			phones, email, address_line, city, state, country, zip_code,
			emergency_name, emergency_phone,
			allergies, medical_conditions,
			insurance_provider, insurance_id,
			condition, notes, family_members,
			source, insurance_validity,
			category_id, special_status_id
		) VALUES (
			$1::uuid, $2,
			$3, $4, $5, NULLIF($6,'')::date, $7,
			$8, $9, $10,
			$11::jsonb, $12, $13, $14, $15, $16, $17,
			$18, $19,
			$20, $21,
			$22, $23,
			$24, $25, $26::jsonb,
			$27, NULLIF($28,'')::date,
			NULLIF($29,'')::uuid, NULLIF($30,'')::uuid
		)
		RETURNING id::text`,
		req.ClinicID, patientNumber,
		req.FirstName, req.MiddleName, req.LastName, req.DateOfBirth, req.Sex,
		req.BloodType, req.MaritalStatus, req.PhotoURL,
		string(phones), req.Email, req.AddressLine, req.City, req.State, req.Country, req.ZipCode,
		req.EmergencyName, req.EmergencyPhone,
		req.Allergies, req.MedicalConditions,
		req.InsuranceProvider, req.InsuranceID,
		req.Condition, req.Notes, string(family),
		req.Source, req.InsuranceValidity,
		req.CategoryID, req.SpecialStatusID,
	).Scan(&id)
	return id, err
}

// Update rewrites the editable columns of one patient. The edit form loads
// every field from the existing record before submitting, so a whole-row write
// is safe here and avoids assembling dynamic SET clauses.
func (r *Repository) Update(ctx context.Context, pool *pgxpool.Pool, id, userID string, req model.CreateRequest) error {
	phones, _ := json.Marshal(req.Phones)
	family, _ := json.Marshal(req.FamilyMembers)
	tag, err := pool.Exec(ctx, `
		UPDATE patients SET
			first_name = $2, middle_name = $3, last_name = $4,
			date_of_birth = NULLIF($5,'')::date, sex = $6,
			blood_type = $7, marital_status = $8, photo_url = $9,
			phones = $10::jsonb, email = $11,
			address_line = $12, city = $13, state = $14, country = $15, zip_code = $16,
			emergency_name = $17, emergency_phone = $18,
			allergies = $19, medical_conditions = $20,
			insurance_provider = $21, insurance_id = $22,
			notes = $23, family_members = $24::jsonb,
			source = $25, insurance_validity = NULLIF($26,'')::date,
			category_id = NULLIF($27,'')::uuid, special_status_id = NULLIF($28,'')::uuid,
			updated_at = now()
		WHERE id = $1::uuid
		  AND clinic_id IN (SELECT clinic_id FROM clinic_members WHERE user_id = $29::uuid)`,
		id,
		req.FirstName, req.MiddleName, req.LastName, req.DateOfBirth, req.Sex,
		req.BloodType, req.MaritalStatus, req.PhotoURL,
		string(phones), req.Email,
		req.AddressLine, req.City, req.State, req.Country, req.ZipCode,
		req.EmergencyName, req.EmergencyPhone,
		req.Allergies, req.MedicalConditions,
		req.InsuranceProvider, req.InsuranceID,
		req.Notes, string(family),
		req.Source, req.InsuranceValidity,
		req.CategoryID, req.SpecialStatusID,
		userID,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *Repository) Delete(ctx context.Context, pool *pgxpool.Pool, id, userID string) error {
	_, err := pool.Exec(ctx, `
		DELETE FROM patients
		WHERE id = $1::uuid
		  AND clinic_id IN (SELECT clinic_id FROM clinic_members WHERE user_id = $2::uuid)`,
		id, userID,
	)
	return err
}

func (r *Repository) ListDocuments(ctx context.Context, pool *pgxpool.Pool, patientID string) ([]model.PatientDocument, error) {
	rows, err := pool.Query(ctx, `
		SELECT id::text, name, url, mime_type, size_bytes, category, doctor,
		       COALESCE(document_date::text, ''), created_at::text
		FROM patient_documents WHERE patient_id = $1::uuid ORDER BY created_at DESC`, patientID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.PatientDocument{}
	for rows.Next() {
		var d model.PatientDocument
		if err := rows.Scan(&d.ID, &d.Name, &d.URL, &d.MimeType, &d.SizeBytes,
			&d.Category, &d.Doctor, &d.DocumentDate, &d.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

func (r *Repository) AddDocument(ctx context.Context, pool *pgxpool.Pool, patientID string, req model.AddDocumentRequest) (string, error) {
	var id string
	err := pool.QueryRow(ctx, `
		INSERT INTO patient_documents
			(patient_id, name, url, mime_type, size_bytes, category, doctor, document_date)
		VALUES ($1::uuid, $2, $3, $4, $5, $6, $7, NULLIF($8,'')::date)
		RETURNING id::text`,
		patientID, req.Name, req.URL, req.MimeType, req.SizeBytes, req.Category, req.Doctor, req.DocumentDate,
	).Scan(&id)
	return id, err
}

func (r *Repository) DeleteDocument(ctx context.Context, pool *pgxpool.Pool, id, patientID string) error {
	_, err := pool.Exec(ctx,
		`DELETE FROM patient_documents WHERE id = $1::uuid AND patient_id = $2::uuid`, id, patientID)
	return err
}

func (r *Repository) AddMedication(ctx context.Context, pool *pgxpool.Pool, patientID string, req model.AddMedicationRequest) (string, error) {
	var id string
	err := pool.QueryRow(ctx, `
		INSERT INTO patient_medications
			(patient_id, name, dose, schedule, purpose, doctor, started_at, status)
		VALUES ($1::uuid, $2, $3, $4, $5, $6, NULLIF($7,'')::date, $8)
		RETURNING id::text`,
		patientID, req.Name, req.Dose, req.Schedule, req.Purpose, req.Doctor, req.StartedAt, req.Status,
	).Scan(&id)
	return id, err
}

func (r *Repository) AddAlert(ctx context.Context, pool *pgxpool.Pool, patientID, name, severity string) (string, error) {
	var id string
	err := pool.QueryRow(ctx,
		`INSERT INTO patient_alerts (patient_id, name, severity)
		 VALUES ($1::uuid, $2, $3) RETURNING id::text`,
		patientID, name, severity,
	).Scan(&id)
	return id, err
}

func (r *Repository) DeleteAlert(ctx context.Context, pool *pgxpool.Pool, id, userID string) error {
	_, err := pool.Exec(ctx, `
		DELETE FROM patient_alerts
		WHERE id = $1::uuid
		  AND patient_id IN (
		    SELECT p.id FROM patients p
		    JOIN clinic_members m ON m.clinic_id = p.clinic_id
		    WHERE m.user_id = $2::uuid
		  )`,
		id, userID,
	)
	return err
}

func (r *Repository) AddAllergy(ctx context.Context, pool *pgxpool.Pool, patientID, name, severity, reaction string) (string, error) {
	var id string
	err := pool.QueryRow(ctx,
		`INSERT INTO patient_allergy_entries (patient_id, name, severity, reaction)
		 VALUES ($1::uuid, $2, $3, $4) RETURNING id::text`,
		patientID, name, severity, reaction,
	).Scan(&id)
	return id, err
}

func (r *Repository) DeleteAllergy(ctx context.Context, pool *pgxpool.Pool, id, userID string) error {
	_, err := pool.Exec(ctx, `
		DELETE FROM patient_allergy_entries
		WHERE id = $1::uuid
		  AND patient_id IN (
		    SELECT p.id FROM patients p
		    JOIN clinic_members m ON m.clinic_id = p.clinic_id
		    WHERE m.user_id = $2::uuid
		  )`,
		id, userID,
	)
	return err
}

func (r *Repository) ClinicForPatient(ctx context.Context, pool *pgxpool.Pool, patientID, userID string) (string, error) {
	var clinicID string
	err := pool.QueryRow(ctx, `
		SELECT p.clinic_id::text
		FROM patients p
		WHERE p.id = $1::uuid
		  AND p.clinic_id IN (SELECT clinic_id FROM clinic_members WHERE user_id = $2::uuid)`,
		patientID, userID,
	).Scan(&clinicID)
	return clinicID, err
}

func (r *Repository) FieldConfig(ctx context.Context, pool *pgxpool.Pool, clinicID string) ([]byte, error) {
	var cfg []byte
	err := pool.QueryRow(ctx,
		`SELECT config FROM patient_field_config WHERE clinic_id = $1::uuid`, clinicID,
	).Scan(&cfg)
	return cfg, err
}

func (r *Repository) PutFieldConfig(ctx context.Context, pool *pgxpool.Pool, clinicID string, body []byte) error {
	_, err := pool.Exec(ctx, `
		INSERT INTO patient_field_config (clinic_id, config, updated_at)
		VALUES ($1::uuid, $2::jsonb, now())
		ON CONFLICT (clinic_id) DO UPDATE
			SET config = EXCLUDED.config, updated_at = now()`,
		clinicID, string(body),
	)
	return err
}

// VitalsConfig / PutVitalsConfig — the clinic's chosen vitals and calculators
// (3055:43827). Mirrors the field-config pair above.
func (r *Repository) VitalsConfig(ctx context.Context, pool *pgxpool.Pool, clinicID string) ([]byte, error) {
	var cfg []byte
	err := pool.QueryRow(ctx,
		`SELECT config FROM patient_vitals_config WHERE clinic_id = $1::uuid`, clinicID,
	).Scan(&cfg)
	return cfg, err
}

func (r *Repository) PutVitalsConfig(ctx context.Context, pool *pgxpool.Pool, clinicID string, body []byte) error {
	_, err := pool.Exec(ctx, `
		INSERT INTO patient_vitals_config (clinic_id, config, updated_at)
		VALUES ($1::uuid, $2::jsonb, now())
		ON CONFLICT (clinic_id) DO UPDATE
			SET config = EXCLUDED.config, updated_at = now()`,
		clinicID, string(body),
	)
	return err
}

// AddVitals writes one batch of readings. The whole batch shares a recorded_at
// and note, so a reading is always attributable to the visit that produced it.
func (r *Repository) AddVitals(ctx context.Context, pool *pgxpool.Pool, patientID string, req model.AddVitalsRequest) (int, error) {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	n := 0
	for _, v := range req.Readings {
		if strings.TrimSpace(v.ValueText) == "" {
			continue // a blank box is "not measured", not a zero reading
		}
		_, err := tx.Exec(ctx, `
			INSERT INTO patient_vitals
				(patient_id, recorded_at, category, kind, value_text, unit, status, reference_range, notes)
			VALUES ($1::uuid, COALESCE(NULLIF($2,'')::timestamptz, now()), $3, $4, $5, $6, $7, $8, $9)`,
			patientID, req.RecordedAt, v.Category, v.Kind, v.ValueText, v.Unit, v.Status, v.ReferenceRange, req.Notes)
		if err != nil {
			return 0, err
		}
		n++
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, err
	}
	return n, nil
}

// ErrOverpay guards the one arithmetic mistake this panel can make: taking
// more than the invoice still owes.
var ErrOverpay = errors.New("amount is more than the invoice owes")

// CollectPayment records money taken against an invoice and moves the
// invoice's own totals with it, in one transaction so the ledger and the
// invoice can never disagree.
func (r *Repository) CollectPayment(
	ctx context.Context, pool *pgxpool.Pool, patientID, invoiceID, userID string,
	req model.CollectPaymentRequest,
) (model.CollectPaymentResult, error) {
	var out model.CollectPaymentResult
	tx, err := pool.Begin(ctx)
	if err != nil {
		return out, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// lock the invoice so two clinicians collecting at once cannot overpay it
	var due, paid int
	var clinicID string
	err = tx.QueryRow(ctx, `
		SELECT amount_due_cents, amount_paid_cents, clinic_id::text
		FROM invoices WHERE id = $1::uuid AND patient_id = $2::uuid FOR UPDATE`,
		invoiceID, patientID).Scan(&due, &paid, &clinicID)
	if err != nil {
		return out, err
	}
	if req.AmountCents > due {
		return out, ErrOverpay
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO invoice_payments (clinic_id, invoice_id, patient_id, amount_cents, method, collected_by)
		VALUES ($1::uuid, $2::uuid, $3::uuid, $4, $5, NULLIF($6,'')::uuid)`,
		clinicID, invoiceID, patientID, req.AmountCents, req.Method, userID); err != nil {
		return out, err
	}

	err = tx.QueryRow(ctx, `
		UPDATE invoices
		   SET amount_paid_cents = amount_paid_cents + $2,
		       amount_due_cents  = amount_due_cents  - $2,
		       status = CASE WHEN amount_due_cents - $2 <= 0 THEN 'paid' ELSE status END
		 WHERE id = $1::uuid
		RETURNING id::text, amount_paid_cents, amount_due_cents, status`,
		invoiceID, req.AmountCents,
	).Scan(&out.InvoiceID, &out.PaidCents, &out.DueCents, &out.Status)
	if err != nil {
		return out, err
	}
	return out, tx.Commit(ctx)
}

func (r *Repository) InvoiceTotals(ctx context.Context, pool *pgxpool.Pool, patientID string) (model.InvoiceTotals, error) {
	var t model.InvoiceTotals
	err := pool.QueryRow(ctx, `
		SELECT COALESCE(sum(amount_total_cents),0),
		       COALESCE(sum(amount_paid_cents),0),
		       COALESCE(sum(amount_due_cents),0),
		       count(*)
		FROM invoices WHERE patient_id = $1::uuid`, patientID,
	).Scan(&t.Total, &t.Paid, &t.Due, &t.Count)
	return t, err
}

func (r *Repository) FetchJSONRows(ctx context.Context, pool *pgxpool.Pool, query string, args ...any) []map[string]any {
	rows, err := pool.Query(ctx, `SELECT row_to_json(t) FROM (`+query+`) t`, args...)
	if err != nil {
		log.Printf("profile subquery: %v", err)
		return []map[string]any{}
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var raw []byte
		if err := rows.Scan(&raw); err != nil {
			continue
		}
		var m map[string]any
		if err := json.Unmarshal(raw, &m); err == nil {
			out = append(out, m)
		}
	}
	return out
}

// Document categories — the clinic's own additions from "Create New Category"
// (1336:20803). The six built-ins are frontend constants and never stored.

func (r *Repository) ListDocumentCategories(ctx context.Context, pool *pgxpool.Pool, clinicID string) ([]model.DocumentCategory, error) {
	rows, err := pool.Query(ctx, `
		SELECT id::text, name FROM patient_document_categories
		WHERE clinic_id = $1::uuid ORDER BY created_at`, clinicID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.DocumentCategory{}
	for rows.Next() {
		var c model.DocumentCategory
		if err := rows.Scan(&c.ID, &c.Name); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// AddDocumentCategory returns the existing row when the name is already taken,
// so adding a duplicate is a no-op rather than an error the dialog has to explain.
func (r *Repository) AddDocumentCategory(ctx context.Context, pool *pgxpool.Pool, clinicID, name string) (string, error) {
	var id string
	err := pool.QueryRow(ctx, `
		INSERT INTO patient_document_categories (clinic_id, name)
		VALUES ($1::uuid, $2)
		ON CONFLICT (clinic_id, lower(name)) DO UPDATE SET name = patient_document_categories.name
		RETURNING id::text`, clinicID, name).Scan(&id)
	return id, err
}

func (r *Repository) DeleteDocumentCategory(ctx context.Context, pool *pgxpool.Pool, clinicID, id string) error {
	_, err := pool.Exec(ctx,
		`DELETE FROM patient_document_categories WHERE id = $1::uuid AND clinic_id = $2::uuid`, id, clinicID)
	return err
}

// Family members — the links behind "Add Family Member" (3841:57713).

func (r *Repository) ListFamily(ctx context.Context, pool *pgxpool.Pool, patientID string) ([]model.FamilyLink, error) {
	rows, err := pool.Query(ctx, `
		SELECT f.id::text, f.relative_id::text,
		       btrim(coalesce(p.first_name,'') || ' ' || coalesce(p.last_name,'')),
		       coalesce(p.patient_number,''), f.relation
		FROM patient_family_members f
		JOIN patients p ON p.id = f.relative_id
		WHERE f.patient_id = $1::uuid
		ORDER BY f.created_at`, patientID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.FamilyLink{}
	for rows.Next() {
		var m model.FamilyLink
		if err := rows.Scan(&m.ID, &m.RelativeID, &m.Name, &m.PatientNumber, &m.Relation); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// AddFamilyMember re-links an existing pair rather than failing, so saving the
// dialog twice just corrects the relation.
func (r *Repository) AddFamilyMember(ctx context.Context, pool *pgxpool.Pool, patientID string, req model.AddFamilyLinkRequest) (string, error) {
	var id string
	err := pool.QueryRow(ctx, `
		INSERT INTO patient_family_members (patient_id, relative_id, relation)
		VALUES ($1::uuid, $2::uuid, $3)
		ON CONFLICT (patient_id, relative_id) DO UPDATE SET relation = EXCLUDED.relation
		RETURNING id::text`, patientID, req.RelativeID, req.Relation).Scan(&id)
	return id, err
}

func (r *Repository) DeleteFamilyMember(ctx context.Context, pool *pgxpool.Pool, patientID, id string) error {
	_, err := pool.Exec(ctx,
		`DELETE FROM patient_family_members WHERE id = $1::uuid AND patient_id = $2::uuid`, id, patientID)
	return err
}
