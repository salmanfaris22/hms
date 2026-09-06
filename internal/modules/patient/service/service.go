package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/salman/hms-backend/internal/core/audit"
	"github.com/salman/hms-backend/internal/infrastructure/persistence/postgres"
	"github.com/salman/hms-backend/internal/modules/patient/model"
	"github.com/salman/hms-backend/internal/modules/patient/repository"
	"github.com/salman/hms-backend/pkg/email"
	"github.com/salman/hms-backend/pkg/sms"
	"github.com/salman/hms-backend/pkg/whatsapp"
)

var (
	ErrUnauthenticated = errors.New("not authenticated")
	ErrTenantUnavail   = errors.New("tenant unavailable")
	ErrForbidden       = errors.New("not a member of this clinic")
	ErrNumberTaken     = errors.New("that patient ID is already in use")
)

// ValidationError marks a bad request rather than a server fault, so the handler
// can answer 400. Without it every rejected field came back as a 500.
type ValidationError struct{ Msg string }

func (e ValidationError) Error() string { return e.Msg }

func invalid(msg string) error { return ValidationError{Msg: msg} }

var (
	ErrNotFound = errors.New("patient not found")
	ErrDB       = errors.New("db error")
)

type Meta struct {
	IP         string
	UserAgent  string
	ActorID    string
	ActorEmail string
	TenantID   string
	UserID     string
}

type Service struct {
	repo     *repository.Repository
	settings SettingsReader
	mailer   email.Sender
	sms      sms.Sender
	whatsapp whatsapp.Sender
}

// New wires the patient service. settings and the three senders drive the
// Message Configuration triggers; all four may be nil, in which case no patient
// messaging is attempted and everything else behaves as before.
func New(
	repo *repository.Repository,
	settings SettingsReader,
	mailer email.Sender,
	smsSender sms.Sender,
	waSender whatsapp.Sender,
) *Service {
	return &Service{repo: repo, settings: settings, mailer: mailer, sms: smsSender, whatsapp: waSender}
}

func (s *Service) Repo() *repository.Repository      { return s.repo }
func (s *Service) Tenants() *postgres.TenantResolver { return s.repo.Tenants() }

func (s *Service) AssertMember(ctx context.Context, pool *pgxpool.Pool, userID, clinicID string) error {
	return s.repo.AssertMember(ctx, pool, userID, clinicID)
}

func (s *Service) List(ctx context.Context, meta Meta, f model.ListFilter) (model.ListResponse, error) {
	f.TenantID = meta.TenantID
	f.UserID = meta.UserID

	pool, err := s.repo.Pool(ctx, meta.TenantID)
	if err != nil {
		return model.ListResponse{}, ErrTenantUnavail
	}
	if err := s.repo.AssertMember(ctx, pool, meta.UserID, f.ClinicID); err != nil {
		return model.ListResponse{}, ErrForbidden
	}

	patients, stats, total, err := s.repo.List(ctx, pool, f)
	if err != nil {
		log.Printf("patients list: %v", err)
		return model.ListResponse{}, ErrDB
	}

	return model.ListResponse{
		Patients: patients,
		Stats:    stats,
		Page:     f.Page,
		PageSize: f.PageSize,
		Total:    total,
	}, nil
}

func (s *Service) Get(ctx context.Context, meta Meta, id string) (model.PatientDTO, error) {
	pool, err := s.repo.Pool(ctx, meta.TenantID)
	if err != nil {
		return model.PatientDTO{}, ErrTenantUnavail
	}
	p, err := s.repo.Get(ctx, pool, id, meta.UserID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.PatientDTO{}, ErrNotFound
		}
		return model.PatientDTO{}, ErrDB
	}
	return p, nil
}

func (s *Service) Create(ctx context.Context, meta Meta, req model.CreateRequest) (id, patientNumber string, err error) {
	req.FirstName = strings.TrimSpace(req.FirstName)
	if req.ClinicID == "" || req.FirstName == "" {
		return "", "", errors.New("clinicId and firstName are required")
	}
	if req.Phones == nil {
		req.Phones = []model.PatientPhone{}
	}
	if req.FamilyMembers == nil {
		req.FamilyMembers = []model.FamilyMember{}
	}

	pool, err := s.repo.Pool(ctx, meta.TenantID)
	if err != nil {
		return "", "", ErrTenantUnavail
	}
	if err := s.repo.AssertMember(ctx, pool, meta.UserID, req.ClinicID); err != nil {
		return "", "", ErrForbidden
	}

	if n := strings.TrimSpace(req.PatientNumber); n != "" {
		// Manual Entry on the form; the clinic owns the numbering scheme
		taken, err := s.repo.PatientNumberExists(ctx, pool, req.ClinicID, n)
		if err != nil {
			return "", "", errors.New("create failed")
		}
		if taken {
			return "", "", ErrNumberTaken
		}
		patientNumber = n
	} else {
		seq, _ := s.repo.NextPatientNumber(ctx, pool)
		patientNumber = fmt.Sprintf("PA%06d", seq)
	}

	id, err = s.repo.Create(ctx, pool, patientNumber, req)
	if err != nil {
		log.Printf("patient create: %v", err)
		return "", "", errors.New("create failed")
	}

	audit.LogRaw(ctx, s.repo.Tenants().Registry(), meta.IP, meta.UserAgent, audit.Event{
		ActorID: meta.ActorID, ActorEmail: meta.ActorEmail, ActorKind: audit.ActorUser,
		TenantID: meta.TenantID, Action: "patient.create", Resource: patientNumber,
	})

	// Fire-and-forget: a slow or failing provider must never affect registration.
	go s.notifyWelcome(meta.TenantID, req.ClinicID, patientNumber, req)

	return id, patientNumber, nil
}

func (s *Service) Update(ctx context.Context, meta Meta, id string, req model.CreateRequest) error {
	if strings.TrimSpace(req.FirstName) == "" {
		return invalid("firstName is required")
	}
	pool, err := s.repo.Pool(ctx, meta.TenantID)
	if err != nil {
		return ErrTenantUnavail
	}
	if err := s.repo.AssertPatientMember(ctx, pool, meta.UserID, id); err != nil {
		return ErrForbidden
	}
	if err := s.repo.Update(ctx, pool, id, meta.UserID, req); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		log.Printf("patient update: %v", err)
		return errors.New("update failed")
	}
	audit.LogRaw(ctx, s.repo.Tenants().Registry(), meta.IP, meta.UserAgent, audit.Event{
		ActorID: meta.ActorID, ActorEmail: meta.ActorEmail, ActorKind: audit.ActorUser,
		TenantID: meta.TenantID, Action: "patient.update", Resource: id,
	})
	return nil
}

func (s *Service) Delete(ctx context.Context, meta Meta, id string) error {
	pool, err := s.repo.Pool(ctx, meta.TenantID)
	if err != nil {
		return ErrTenantUnavail
	}
	if err := s.repo.Delete(ctx, pool, id, meta.UserID); err != nil {
		return errors.New("delete failed")
	}
	audit.LogRaw(ctx, s.repo.Tenants().Registry(), meta.IP, meta.UserAgent, audit.Event{
		ActorID: meta.ActorID, ActorEmail: meta.ActorEmail, ActorKind: audit.ActorUser,
		TenantID: meta.TenantID, Action: "patient.delete", Resource: id,
	})
	return nil
}

func (s *Service) ListDocuments(ctx context.Context, meta Meta, patientID string) ([]model.PatientDocument, error) {
	pool, err := s.repo.Pool(ctx, meta.TenantID)
	if err != nil {
		return nil, ErrTenantUnavail
	}
	if err := s.repo.AssertPatientMember(ctx, pool, meta.UserID, patientID); err != nil {
		return nil, ErrForbidden
	}
	docs, err := s.repo.ListDocuments(ctx, pool, patientID)
	if err != nil {
		log.Printf("list documents: %v", err)
		return nil, ErrDB
	}
	return docs, nil
}

func (s *Service) AddDocument(ctx context.Context, meta Meta, patientID string, req model.AddDocumentRequest) (string, error) {
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" || strings.TrimSpace(req.URL) == "" {
		return "", invalid("name and url are required")
	}
	if req.Category == "" {
		req.Category = "Other"
	}
	pool, err := s.repo.Pool(ctx, meta.TenantID)
	if err != nil {
		return "", ErrTenantUnavail
	}
	if err := s.repo.AssertPatientMember(ctx, pool, meta.UserID, patientID); err != nil {
		return "", ErrForbidden
	}
	id, err := s.repo.AddDocument(ctx, pool, patientID, req)
	if err != nil {
		log.Printf("add document: %v", err)
		return "", errors.New("insert failed")
	}
	audit.LogRaw(ctx, s.repo.Tenants().Registry(), meta.IP, meta.UserAgent, audit.Event{
		ActorID: meta.ActorID, ActorEmail: meta.ActorEmail, ActorKind: audit.ActorUser,
		TenantID: meta.TenantID, Action: "patient.document.add", Resource: patientID,
		Metadata: map[string]any{"document": req.Name},
	})
	return id, nil
}

func (s *Service) DeleteDocument(ctx context.Context, meta Meta, patientID, id string) error {
	pool, err := s.repo.Pool(ctx, meta.TenantID)
	if err != nil {
		return ErrTenantUnavail
	}
	if err := s.repo.AssertPatientMember(ctx, pool, meta.UserID, patientID); err != nil {
		return ErrForbidden
	}
	if err := s.repo.DeleteDocument(ctx, pool, id, patientID); err != nil {
		return errors.New("delete failed")
	}
	audit.LogRaw(ctx, s.repo.Tenants().Registry(), meta.IP, meta.UserAgent, audit.Event{
		ActorID: meta.ActorID, ActorEmail: meta.ActorEmail, ActorKind: audit.ActorUser,
		TenantID: meta.TenantID, Action: "patient.document.delete", Resource: id,
	})
	return nil
}

func (s *Service) AddMedication(ctx context.Context, meta Meta, patientID string, req model.AddMedicationRequest) (string, error) {
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		return "", invalid("name is required")
	}
	if req.Status == "" {
		req.Status = "active"
	}
	pool, err := s.repo.Pool(ctx, meta.TenantID)
	if err != nil {
		return "", ErrTenantUnavail
	}
	if err := s.repo.AssertPatientMember(ctx, pool, meta.UserID, patientID); err != nil {
		return "", ErrForbidden
	}
	id, err := s.repo.AddMedication(ctx, pool, patientID, req)
	if err != nil {
		log.Printf("add medication: %v", err)
		return "", errors.New("insert failed")
	}
	audit.LogRaw(ctx, s.repo.Tenants().Registry(), meta.IP, meta.UserAgent, audit.Event{
		ActorID: meta.ActorID, ActorEmail: meta.ActorEmail, ActorKind: audit.ActorUser,
		TenantID: meta.TenantID, Action: "patient.medication.add", Resource: patientID,
		Metadata: map[string]any{"medication": req.Name},
	})
	return id, nil
}

func (s *Service) AddAlert(ctx context.Context, meta Meta, patientID string, req model.AddAlertRequest) (string, error) {
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		return "", invalid("name is required")
	}
	if req.Severity == "" {
		req.Severity = "medium"
	}
	pool, err := s.repo.Pool(ctx, meta.TenantID)
	if err != nil {
		return "", ErrTenantUnavail
	}
	if err := s.repo.AssertPatientMember(ctx, pool, meta.UserID, patientID); err != nil {
		return "", errors.New("forbidden")
	}
	id, err := s.repo.AddAlert(ctx, pool, patientID, req.Name, req.Severity)
	if err != nil {
		return "", errors.New("insert failed")
	}
	audit.LogRaw(ctx, s.repo.Tenants().Registry(), meta.IP, meta.UserAgent, audit.Event{
		ActorID: meta.ActorID, ActorEmail: meta.ActorEmail, ActorKind: audit.ActorUser,
		TenantID: meta.TenantID, Action: "patient.alert.add", Resource: patientID,
		Metadata: map[string]any{"alert": req.Name, "severity": req.Severity},
	})
	return id, nil
}

func (s *Service) DeleteAlert(ctx context.Context, meta Meta, id string) error {
	pool, err := s.repo.Pool(ctx, meta.TenantID)
	if err != nil {
		return ErrTenantUnavail
	}
	if err := s.repo.DeleteAlert(ctx, pool, id, meta.UserID); err != nil {
		return errors.New("delete failed")
	}
	audit.LogRaw(ctx, s.repo.Tenants().Registry(), meta.IP, meta.UserAgent, audit.Event{
		ActorID: meta.ActorID, ActorEmail: meta.ActorEmail, ActorKind: audit.ActorUser,
		TenantID: meta.TenantID, Action: "patient.alert.delete", Resource: id,
	})
	return nil
}

func (s *Service) AddAllergy(ctx context.Context, meta Meta, patientID string, req model.AddAllergyRequest) (string, error) {
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		return "", invalid("name is required")
	}
	if req.Severity == "" {
		req.Severity = "medium"
	}
	pool, err := s.repo.Pool(ctx, meta.TenantID)
	if err != nil {
		return "", ErrTenantUnavail
	}
	if err := s.repo.AssertPatientMember(ctx, pool, meta.UserID, patientID); err != nil {
		return "", errors.New("forbidden")
	}
	id, err := s.repo.AddAllergy(ctx, pool, patientID, req.Name, req.Severity, req.Reaction)
	if err != nil {
		return "", errors.New("insert failed")
	}
	audit.LogRaw(ctx, s.repo.Tenants().Registry(), meta.IP, meta.UserAgent, audit.Event{
		ActorID: meta.ActorID, ActorEmail: meta.ActorEmail, ActorKind: audit.ActorUser,
		TenantID: meta.TenantID, Action: "patient.allergy.add", Resource: patientID,
		Metadata: map[string]any{"allergy": req.Name, "severity": req.Severity},
	})
	return id, nil
}

func (s *Service) DeleteAllergy(ctx context.Context, meta Meta, id string) error {
	pool, err := s.repo.Pool(ctx, meta.TenantID)
	if err != nil {
		return ErrTenantUnavail
	}
	if err := s.repo.DeleteAllergy(ctx, pool, id, meta.UserID); err != nil {
		return errors.New("delete failed")
	}
	audit.LogRaw(ctx, s.repo.Tenants().Registry(), meta.IP, meta.UserAgent, audit.Event{
		ActorID: meta.ActorID, ActorEmail: meta.ActorEmail, ActorKind: audit.ActorUser,
		TenantID: meta.TenantID, Action: "patient.allergy.delete", Resource: id,
	})
	return nil
}

func (s *Service) SeedDemo(ctx context.Context, meta Meta, patientID string) error {
	pool, err := s.repo.Pool(ctx, meta.TenantID)
	if err != nil {
		return ErrTenantUnavail
	}
	clinicID, err := s.repo.ClinicForPatient(ctx, pool, patientID, meta.UserID)
	if err != nil {
		return errors.New("forbidden or not found")
	}
	if err := s.repo.SeedDemo(ctx, pool, patientID, clinicID); err != nil {
		return err
	}
	// this replaces the patient's alerts, allergies, vitals, appointments,
	// invoices and timeline, so it belongs in the trail like any other deletion
	audit.LogRaw(ctx, s.repo.Tenants().Registry(), meta.IP, meta.UserAgent, audit.Event{
		ActorID: meta.ActorID, ActorEmail: meta.ActorEmail, ActorKind: audit.ActorUser,
		TenantID: meta.TenantID, Action: "patient.demo.seed", Resource: patientID,
	})
	return nil
}

func (s *Service) GetProfile(ctx context.Context, meta Meta, patientID string) (map[string]any, error) {
	pool, err := s.repo.Pool(ctx, meta.TenantID)
	if err != nil {
		return nil, ErrTenantUnavail
	}
	if _, err := s.repo.ClinicForPatient(ctx, pool, patientID, meta.UserID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		log.Printf("profile authz: %v", err)
		return nil, ErrDB
	}

	patRow := pool.QueryRow(ctx, `SELECT `+repository.Columns+` FROM patients WHERE id = $1::uuid`, patientID)
	pt, err := repository.ScanPatient(patRow)
	if err != nil {
		return nil, errors.New("scan error")
	}

	alerts := s.repo.FetchJSONRows(ctx, pool, `
		SELECT id::text, name, severity, created_at
		FROM patient_alerts WHERE patient_id = $1::uuid ORDER BY created_at DESC`, patientID)

	allergies := s.repo.FetchJSONRows(ctx, pool, `
		SELECT id::text, name, severity, reaction, created_at
		FROM patient_allergy_entries WHERE patient_id = $1::uuid ORDER BY created_at DESC`, patientID)

	medications := s.repo.FetchJSONRows(ctx, pool, `
		SELECT id::text, name, dose, schedule, started_at, status, purpose, doctor, created_at
		FROM patient_medications WHERE patient_id = $1::uuid ORDER BY started_at DESC NULLS LAST`, patientID)

	// The Vital History modal is a full-history surface, and one capture writes a
	// dozen rows, so a small cap silently drops whole visits: at LIMIT 50 a patient
	// with 69 readings lost every reading older than the third-most-recent visit.
	vitals := s.repo.FetchJSONRows(ctx, pool, `
		SELECT id::text, recorded_at, category, kind, value_text, unit, status,
		       reference_range AS "referenceRange", notes
		FROM patient_vitals WHERE patient_id = $1::uuid ORDER BY recorded_at DESC LIMIT 2000`, patientID)

	// the provider's identity travels with the visit: free follow-ups can be
	// counted per doctor or per department, and a name alone cannot carry that
	appointments := s.repo.FetchJSONRows(ctx, pool, `
		SELECT a.id::text, a.clinic_id::text, a.patient_id::text,
		       coalesce(a.provider_id::text, '') AS provider_id,
		       a.provider_name, a.kind,
		       a.scheduled_at, a.duration_min, a.status, a.notes,
		       coalesce(dep.name, '')  AS provider_department,
		       coalesce(spec.name, '') AS provider_specialization,
		       a.created_at, a.updated_at
		FROM appointments a
		LEFT JOIN staff_profiles sp    ON sp.user_id = a.provider_id
		LEFT JOIN departments dep      ON dep.id = sp.department_id
		LEFT JOIN specializations spec ON spec.id = sp.specialization_id
		WHERE a.patient_id = $1::uuid ORDER BY a.scheduled_at DESC LIMIT 50`, patientID)

	treatments := s.repo.FetchJSONRows(ctx, pool, `
		SELECT id::text, name, category, amount_cents, performed_at, created_at
		FROM patient_treatments WHERE patient_id = $1::uuid ORDER BY performed_at DESC NULLS LAST LIMIT 20`, patientID)

	invoices := s.repo.FetchJSONRows(ctx, pool, `
		SELECT id::text, number, amount_total_cents, amount_paid_cents, amount_due_cents,
		       status, issued_at, due_at, line_items, created_at
		FROM invoices WHERE patient_id = $1::uuid ORDER BY issued_at DESC NULLS LAST LIMIT 20`, patientID)

	totals, _ := s.repo.InvoiceTotals(ctx, pool, patientID)

	insurance := s.repo.FetchJSONRows(ctx, pool, `
		SELECT id::text, provider, policy_number, claim_number,
		       amount_claimed_cents, amount_approved_cents, amount_due_cents,
		       status, submitted_at, resolved_at, created_at
		FROM insurance_claims WHERE patient_id = $1::uuid ORDER BY submitted_at DESC NULLS LAST`, patientID)

	timeline := s.repo.FetchJSONRows(ctx, pool, `
		SELECT id::text, kind, title, description, amount_cents, metadata, occurred_at, created_at
		FROM patient_events WHERE patient_id = $1::uuid ORDER BY occurred_at DESC LIMIT 100`, patientID)

	return map[string]any{
		"patient":       pt,
		"alerts":        alerts,
		"allergies":     allergies,
		"medications":   medications,
		"vitals":        vitals,
		"appointments":  appointments,
		"treatments":    treatments,
		"invoices":      invoices,
		"invoiceTotals": totals,
		"insurance":     insurance,
		"timeline":      timeline,
	}, nil
}

func (s *Service) GetFieldConfig(ctx context.Context, meta Meta, clinicID string) ([]byte, error) {
	pool, err := s.repo.Pool(ctx, meta.TenantID)
	if err != nil {
		return nil, ErrTenantUnavail
	}
	if err := s.repo.AssertMember(ctx, pool, meta.UserID, clinicID); err != nil {
		return nil, ErrForbidden
	}
	cfg, err := s.repo.FieldConfig(ctx, pool, clinicID)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrDB
	}
	if len(cfg) == 0 {
		cfg = []byte("{}")
	}
	return cfg, nil
}

func (s *Service) PutFieldConfig(ctx context.Context, meta Meta, clinicID string, body json.RawMessage) error {
	pool, err := s.repo.Pool(ctx, meta.TenantID)
	if err != nil {
		return ErrTenantUnavail
	}
	if err := s.repo.AssertMember(ctx, pool, meta.UserID, clinicID); err != nil {
		return ErrForbidden
	}
	if err := s.repo.PutFieldConfig(ctx, pool, clinicID, body); err != nil {
		log.Printf("put field config: %v", err)
		return errors.New("save failed")
	}
	audit.LogRaw(ctx, s.repo.Tenants().Registry(), meta.IP, meta.UserAgent, audit.Event{
		ActorID: meta.ActorID, ActorEmail: meta.ActorEmail, ActorKind: audit.ActorUser,
		TenantID: meta.TenantID, Action: "patient.field-config.update", Resource: clinicID,
	})
	return nil
}

// AddVitals records one batch of readings from the Add Vitals modal.
func (s *Service) AddVitals(ctx context.Context, meta Meta, patientID string, req model.AddVitalsRequest) (int, error) {
	pool, err := s.repo.Pool(ctx, meta.TenantID)
	if err != nil {
		return 0, ErrTenantUnavail
	}
	if err := s.repo.AssertPatientMember(ctx, pool, meta.UserID, patientID); err != nil {
		return 0, ErrForbidden
	}
	n, err := s.repo.AddVitals(ctx, pool, patientID, req)
	if err != nil {
		log.Printf("add vitals: %v", err)
		return 0, errors.New("save failed")
	}
	if n == 0 {
		return 0, invalid("no readings to save")
	}
	audit.LogRaw(ctx, s.repo.Tenants().Registry(), meta.IP, meta.UserAgent, audit.Event{
		ActorID: meta.ActorID, ActorEmail: meta.ActorEmail, ActorKind: audit.ActorUser,
		TenantID: meta.TenantID, Action: "patient.vitals.add", Resource: patientID,
		Metadata: map[string]any{"readings": n},
	})
	return n, nil
}

// CollectPayment takes money against one of the patient's pending invoices.
func (s *Service) CollectPayment(
	ctx context.Context, meta Meta, patientID, invoiceID string, req model.CollectPaymentRequest,
) (model.CollectPaymentResult, error) {
	var out model.CollectPaymentResult
	if patientID == "" || invoiceID == "" {
		return out, invalid("patient and invoice are required")
	}
	if req.AmountCents <= 0 {
		return out, invalid("enter an amount to collect")
	}
	if strings.TrimSpace(req.Method) == "" {
		return out, invalid("choose a payment method")
	}
	pool, err := s.repo.Pool(ctx, meta.TenantID)
	if err != nil {
		return out, ErrTenantUnavail
	}
	if err := s.repo.AssertPatientMember(ctx, pool, meta.UserID, patientID); err != nil {
		return out, ErrForbidden
	}
	out, err = s.repo.CollectPayment(ctx, pool, patientID, invoiceID, meta.UserID, req)
	if err != nil {
		if errors.Is(err, repository.ErrOverpay) {
			return out, invalid("that is more than the invoice owes")
		}
		if errors.Is(err, pgx.ErrNoRows) {
			return out, ErrNotFound
		}
		log.Printf("collect payment: %v", err)
		return out, errors.New("save failed")
	}
	audit.LogRaw(ctx, s.repo.Tenants().Registry(), meta.IP, meta.UserAgent, audit.Event{
		ActorID: meta.ActorID, ActorEmail: meta.ActorEmail, ActorKind: audit.ActorUser,
		TenantID: meta.TenantID, Action: "patient.payment.collect", Resource: invoiceID,
		Metadata: map[string]any{"amountCents": req.AmountCents, "method": req.Method},
	})
	return out, nil
}

// VitalsConfig / PutVitalsConfig — which vitals and calculators the clinic uses.
func (s *Service) VitalsConfig(ctx context.Context, meta Meta, clinicID string) (json.RawMessage, error) {
	pool, err := s.repo.Pool(ctx, meta.TenantID)
	if err != nil {
		return nil, ErrTenantUnavail
	}
	if err := s.repo.AssertMember(ctx, pool, meta.UserID, clinicID); err != nil {
		return nil, ErrForbidden
	}
	cfg, err := s.repo.VitalsConfig(ctx, pool, clinicID)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrDB
	}
	if len(cfg) == 0 {
		cfg = []byte("{}")
	}
	return cfg, nil
}

func (s *Service) PutVitalsConfig(ctx context.Context, meta Meta, clinicID string, body json.RawMessage) error {
	pool, err := s.repo.Pool(ctx, meta.TenantID)
	if err != nil {
		return ErrTenantUnavail
	}
	if err := s.repo.AssertMember(ctx, pool, meta.UserID, clinicID); err != nil {
		return ErrForbidden
	}
	if err := s.repo.PutVitalsConfig(ctx, pool, clinicID, body); err != nil {
		log.Printf("put vitals config: %v", err)
		return errors.New("save failed")
	}
	audit.LogRaw(ctx, s.repo.Tenants().Registry(), meta.IP, meta.UserAgent, audit.Event{
		ActorID: meta.ActorID, ActorEmail: meta.ActorEmail, ActorKind: audit.ActorUser,
		TenantID: meta.TenantID, Action: "patient.vitals-config.update", Resource: clinicID,
	})
	return nil
}

// Document categories. Clinic-scoped like the vitals config, not patient-scoped:
// a category the clinic invents is offered on every patient's upload dialog.

func (s *Service) ListDocumentCategories(ctx context.Context, meta Meta, clinicID string) ([]model.DocumentCategory, error) {
	pool, err := s.repo.Pool(ctx, meta.TenantID)
	if err != nil {
		return nil, ErrTenantUnavail
	}
	if err := s.repo.AssertMember(ctx, pool, meta.UserID, clinicID); err != nil {
		return nil, ErrForbidden
	}
	cats, err := s.repo.ListDocumentCategories(ctx, pool, clinicID)
	if err != nil {
		log.Printf("list document categories: %v", err)
		return nil, ErrDB
	}
	return cats, nil
}

func (s *Service) AddDocumentCategory(ctx context.Context, meta Meta, clinicID, name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", invalid("name is required")
	}
	if len([]rune(name)) > 60 {
		return "", invalid("name is too long")
	}
	pool, err := s.repo.Pool(ctx, meta.TenantID)
	if err != nil {
		return "", ErrTenantUnavail
	}
	if err := s.repo.AssertMember(ctx, pool, meta.UserID, clinicID); err != nil {
		return "", ErrForbidden
	}
	id, err := s.repo.AddDocumentCategory(ctx, pool, clinicID, name)
	if err != nil {
		log.Printf("add document category: %v", err)
		return "", errors.New("insert failed")
	}
	audit.LogRaw(ctx, s.repo.Tenants().Registry(), meta.IP, meta.UserAgent, audit.Event{
		ActorID: meta.ActorID, ActorEmail: meta.ActorEmail, ActorKind: audit.ActorUser,
		TenantID: meta.TenantID, Action: "patient.document.category.add", Resource: clinicID,
		Metadata: map[string]any{"category": name},
	})
	return id, nil
}

func (s *Service) DeleteDocumentCategory(ctx context.Context, meta Meta, clinicID, id string) error {
	pool, err := s.repo.Pool(ctx, meta.TenantID)
	if err != nil {
		return ErrTenantUnavail
	}
	if err := s.repo.AssertMember(ctx, pool, meta.UserID, clinicID); err != nil {
		return ErrForbidden
	}
	if err := s.repo.DeleteDocumentCategory(ctx, pool, clinicID, id); err != nil {
		log.Printf("delete document category: %v", err)
		return errors.New("delete failed")
	}
	audit.LogRaw(ctx, s.repo.Tenants().Registry(), meta.IP, meta.UserAgent, audit.Event{
		ActorID: meta.ActorID, ActorEmail: meta.ActorEmail, ActorKind: audit.ActorUser,
		TenantID: meta.TenantID, Action: "patient.document.category.delete", Resource: id,
	})
	return nil
}

// Family members. Patient-scoped, so membership is asserted against the patient
// on both sides of the link — a relative in another clinic cannot be attached.

func (s *Service) ListFamily(ctx context.Context, meta Meta, patientID string) ([]model.FamilyLink, error) {
	pool, err := s.repo.Pool(ctx, meta.TenantID)
	if err != nil {
		return nil, ErrTenantUnavail
	}
	if err := s.repo.AssertPatientMember(ctx, pool, meta.UserID, patientID); err != nil {
		return nil, ErrForbidden
	}
	items, err := s.repo.ListFamily(ctx, pool, patientID)
	if err != nil {
		log.Printf("list family: %v", err)
		return nil, ErrDB
	}
	return items, nil
}

func (s *Service) AddFamilyMember(ctx context.Context, meta Meta, patientID string, req model.AddFamilyLinkRequest) (string, error) {
	req.Relation = strings.TrimSpace(req.Relation)
	if req.RelativeID == "" || req.Relation == "" {
		return "", invalid("relativeId and relation are required")
	}
	if req.RelativeID == patientID {
		return "", invalid("a patient cannot be their own family member")
	}
	pool, err := s.repo.Pool(ctx, meta.TenantID)
	if err != nil {
		return "", ErrTenantUnavail
	}
	if err := s.repo.AssertPatientMember(ctx, pool, meta.UserID, patientID); err != nil {
		return "", ErrForbidden
	}
	// the relative has to be reachable by this user too, or the link would leak
	// a name across clinics
	if err := s.repo.AssertPatientMember(ctx, pool, meta.UserID, req.RelativeID); err != nil {
		return "", ErrForbidden
	}
	id, err := s.repo.AddFamilyMember(ctx, pool, patientID, req)
	if err != nil {
		log.Printf("add family member: %v", err)
		return "", errors.New("insert failed")
	}
	audit.LogRaw(ctx, s.repo.Tenants().Registry(), meta.IP, meta.UserAgent, audit.Event{
		ActorID: meta.ActorID, ActorEmail: meta.ActorEmail, ActorKind: audit.ActorUser,
		TenantID: meta.TenantID, Action: "patient.family.add", Resource: patientID,
		Metadata: map[string]any{"relative": req.RelativeID, "relation": req.Relation},
	})
	return id, nil
}

func (s *Service) DeleteFamilyMember(ctx context.Context, meta Meta, patientID, id string) error {
	pool, err := s.repo.Pool(ctx, meta.TenantID)
	if err != nil {
		return ErrTenantUnavail
	}
	if err := s.repo.AssertPatientMember(ctx, pool, meta.UserID, patientID); err != nil {
		return ErrForbidden
	}
	if err := s.repo.DeleteFamilyMember(ctx, pool, patientID, id); err != nil {
		log.Printf("delete family member: %v", err)
		return errors.New("delete failed")
	}
	audit.LogRaw(ctx, s.repo.Tenants().Registry(), meta.IP, meta.UserAgent, audit.Event{
		ActorID: meta.ActorID, ActorEmail: meta.ActorEmail, ActorKind: audit.ActorUser,
		TenantID: meta.TenantID, Action: "patient.family.delete", Resource: id,
	})
	return nil
}
