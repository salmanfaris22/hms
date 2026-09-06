package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

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
	created_at, updated_at
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
			condition, notes, family_members
		) VALUES (
			$1::uuid, $2,
			$3, $4, $5, NULLIF($6,'')::date, $7,
			$8, $9, $10,
			$11::jsonb, $12, $13, $14, $15, $16, $17,
			$18, $19,
			$20, $21,
			$22, $23,
			$24, $25, $26::jsonb
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
	).Scan(&id)
	return id, err
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
