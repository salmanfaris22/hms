package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/salman/hms-backend/internal/core/audit"
	"github.com/salman/hms-backend/internal/infrastructure/persistence/postgres"
	"github.com/salman/hms-backend/internal/modules/patient/model"
	"github.com/salman/hms-backend/internal/modules/patient/repository"
	"github.com/salman/hms-backend/pkg/cache"
)

var (
	ErrUnauthenticated = errors.New("not authenticated")
	ErrTenantUnavail   = errors.New("tenant unavailable")
	ErrForbidden       = errors.New("not a member of this clinic")
	ErrNotFound        = errors.New("patient not found")
	ErrDB              = errors.New("db error")
)

const (
	cacheTTLList   = 5 * time.Minute
	cacheTTLGet    = 5 * time.Minute
	cacheTTLConfig = 10 * time.Minute
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
	repo *repository.Repository
}

func New(repo *repository.Repository) *Service { return &Service{repo: repo} }

func (s *Service) Repo() *repository.Repository       { return s.repo }
func (s *Service) Tenants() *postgres.TenantResolver  { return s.repo.Tenants() }

func (s *Service) AssertMember(ctx context.Context, pool *pgxpool.Pool, userID, clinicID string) error {
	return s.repo.AssertMember(ctx, pool, userID, clinicID)
}

func (s *Service) List(ctx context.Context, meta Meta, f model.ListFilter) (model.ListResponse, error) {
	f.TenantID = meta.TenantID
	f.UserID = meta.UserID

	cacheKey := fmt.Sprintf("patients:%s:%s:%d:%d:%s:%s", f.TenantID, f.ClinicID, f.Page, f.PageSize, f.Status, f.Search)
	if cache.Client != nil {
		var cached model.ListResponse
		if err := cache.Get(ctx, cacheKey, &cached); err == nil {
			return cached, nil
		}
	}

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

	resp := model.ListResponse{
		Patients: patients,
		Stats:    stats,
		Page:     f.Page,
		PageSize: f.PageSize,
		Total:    total,
	}
	if cache.Client != nil {
		if err := cache.Set(ctx, cacheKey, resp, cacheTTLList); err != nil {
			log.Printf("cache set: %v", err)
		}
	}
	return resp, nil
}

func (s *Service) Get(ctx context.Context, meta Meta, id string) (model.PatientDTO, error) {
	cacheKey := cache.PatientGetKey(meta.TenantID, id)
	if cache.Client != nil {
		var cached model.PatientDTO
		if err := cache.Get(ctx, cacheKey, &cached); err == nil {
			return cached, nil
		}
	}

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
	if cache.Client != nil {
		if err := cache.Set(ctx, cacheKey, p, cacheTTLGet); err != nil {
			log.Printf("cache set: %v", err)
		}
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

	seq, _ := s.repo.NextPatientNumber(ctx, pool)
	patientNumber = fmt.Sprintf("PA%06d", seq)

	id, err = s.repo.Create(ctx, pool, patientNumber, req)
	if err != nil {
		log.Printf("patient create: %v", err)
		return "", "", errors.New("create failed")
	}

	if cache.Client != nil {
		_ = cache.InvalidatePatientList(ctx, meta.TenantID)
		_ = cache.InvalidateClinic(ctx, meta.TenantID)
	}

	audit.LogRaw(ctx, s.repo.Tenants().Registry(), meta.IP, meta.UserAgent, audit.Event{
		ActorID: meta.ActorID, ActorEmail: meta.ActorEmail, ActorKind: audit.ActorUser,
		TenantID: meta.TenantID, Action: "patient.create", Resource: patientNumber,
	})
	return id, patientNumber, nil
}

func (s *Service) Delete(ctx context.Context, meta Meta, id string) error {
	pool, err := s.repo.Pool(ctx, meta.TenantID)
	if err != nil {
		return ErrTenantUnavail
	}
	if err := s.repo.Delete(ctx, pool, id, meta.UserID); err != nil {
		return errors.New("delete failed")
	}
	if cache.Client != nil {
		_ = cache.InvalidatePatient(ctx, meta.TenantID, id)
		_ = cache.InvalidatePatientList(ctx, meta.TenantID)
	}
	audit.LogRaw(ctx, s.repo.Tenants().Registry(), meta.IP, meta.UserAgent, audit.Event{
		ActorID: meta.ActorID, ActorEmail: meta.ActorEmail, ActorKind: audit.ActorUser,
		TenantID: meta.TenantID, Action: "patient.delete", Resource: id,
	})
	return nil
}

func (s *Service) AddAlert(ctx context.Context, meta Meta, patientID string, req model.AddAlertRequest) (string, error) {
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		return "", errors.New("name is required")
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
	return nil
}

func (s *Service) AddAllergy(ctx context.Context, meta Meta, patientID string, req model.AddAllergyRequest) (string, error) {
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		return "", errors.New("name is required")
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
	return s.repo.SeedDemo(ctx, pool, patientID, clinicID)
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
		SELECT id::text, name, dose, schedule, started_at, status, created_at
		FROM patient_medications WHERE patient_id = $1::uuid ORDER BY started_at DESC NULLS LAST`, patientID)

	vitals := s.repo.FetchJSONRows(ctx, pool, `
		SELECT id::text, recorded_at, category, kind, value_text, unit, status
		FROM patient_vitals WHERE patient_id = $1::uuid ORDER BY recorded_at DESC LIMIT 50`, patientID)

	appointments := s.repo.FetchJSONRows(ctx, pool, `
		SELECT id::text, clinic_id::text, patient_id::text, provider_name, kind,
		       scheduled_at, duration_min, status, notes, created_at, updated_at
		FROM appointments WHERE patient_id = $1::uuid ORDER BY scheduled_at DESC LIMIT 50`, patientID)

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
	return nil
}
