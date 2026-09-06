package service

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/salman/hms-backend/internal/core/audit"
	"github.com/salman/hms-backend/internal/modules/clinic/model"
	"github.com/salman/hms-backend/internal/modules/clinic/repository"
)

var (
	ErrTenantUnavail  = errors.New("tenant unavailable")
	ErrForbidden      = errors.New("not a member of this clinic")
	ErrSourceForbid   = errors.New("not a member of source clinic")
	ErrNotFound       = errors.New("clinic not found")
	ErrGenericNotFound = errors.New("not found")
	ErrBadBody        = errors.New("invalid body")
	ErrNameReq        = errors.New("name is required")
	ErrNothingUpdate  = errors.New("nothing to update")
	ErrBadInvite      = errors.New("invite code must be 6 characters")
	ErrInviteNotFound = errors.New("invite code not found")
	ErrBadKind        = errors.New("invalid kind")
	ErrBadTable       = errors.New("bad table")
	ErrTargetsReq     = errors.New("targetClinicIds is required")
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

func (s *Service) pool(ctx context.Context, tenantID string) (*pgxpool.Pool, error) {
	pool, err := s.repo.Pool(ctx, tenantID)
	if err != nil {
		return nil, ErrTenantUnavail
	}
	return pool, nil
}

func (s *Service) poolAndMember(ctx context.Context, tenantID, clinicID, userID string) (*pgxpool.Pool, error) {
	pool, err := s.pool(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	if !s.repo.IsMember(ctx, pool, clinicID, userID) {
		return nil, ErrForbidden
	}
	return pool, nil
}

func (s *Service) audit(ctx context.Context, m Meta, action, resource string) {
	audit.LogRaw(ctx, s.repo.Tenants().Registry(), m.IP, m.UserAgent, audit.Event{
		ActorID: m.ActorID, ActorEmail: m.ActorEmail, ActorKind: audit.ActorUser,
		TenantID: m.TenantID, Action: action, Resource: resource,
	})
}

// ── clinics CRUD ─────────────────────────────────────────────────────────────

func (s *Service) List(ctx context.Context, m Meta, archived bool) ([]model.ClinicDTO, error) {
	pool, err := s.pool(ctx, m.TenantID)
	if err != nil {
		return nil, err
	}
	return s.repo.ListClinics(ctx, pool, m.UserID, archived)
}

func (s *Service) Get(ctx context.Context, m Meta, id string) (model.ClinicDTO, error) {
	pool, err := s.pool(ctx, m.TenantID)
	if err != nil {
		return model.ClinicDTO{}, err
	}
	c, err := s.repo.GetClinic(ctx, pool, id, m.UserID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.ClinicDTO{}, ErrNotFound
		}
		return model.ClinicDTO{}, err
	}
	return c, nil
}

func (s *Service) Create(ctx context.Context, m Meta, req model.CreateClinicRequest) (string, error) {
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		return "", ErrNameReq
	}
	if req.Color == "" {
		req.Color = "purple"
	}
	if req.TimeFormat == "" {
		req.TimeFormat = "12h"
	}
	if req.SystemLanguage == "" {
		req.SystemLanguage = "en"
	}
	if req.TimeZone == "" {
		req.TimeZone = "UTC"
	}
	if req.DateFormat == "" {
		req.DateFormat = "MM/DD/YYYY"
	}
	if req.Phones == nil {
		req.Phones = []model.Phone{}
	}
	if req.Timings == nil {
		req.Timings = model.Timings{}
	}
	code, err := repository.RandomInviteCode(6)
	if err != nil {
		return "", errors.New("code gen failed")
	}
	pool, err := s.pool(ctx, m.TenantID)
	if err != nil {
		return "", err
	}
	id, err := s.repo.CreateClinic(ctx, pool, m.UserID, code, req)
	if err != nil {
		log.Printf("create clinic: %v", err)
		return "", errors.New("insert failed")
	}
	s.audit(ctx, m, "clinic.create", req.Name)
	return id, nil
}

func (s *Service) Update(ctx context.Context, m Meta, id string, req model.UpdateClinicRequest) error {
	pool, err := s.pool(ctx, m.TenantID)
	if err != nil {
		return err
	}
	ok, err := s.repo.UpdateClinic(ctx, pool, id, m.UserID, req)
	if err != nil {
		log.Printf("update clinic: %v", err)
		return errors.New("update failed")
	}
	if !ok {
		return ErrNothingUpdate
	}
	s.audit(ctx, m, "clinic.update", id)
	return nil
}

func (s *Service) Archive(ctx context.Context, m Meta, id string) error {
	return s.flag(ctx, m, id, `
		UPDATE clinics
		SET is_archived = true, is_primary = false, archived_at = now(), updated_at = now()
		WHERE id = $1::uuid`, "clinic.archive")
}

func (s *Service) Restore(ctx context.Context, m Meta, id string) error {
	return s.flag(ctx, m, id, `
		UPDATE clinics
		SET is_archived = false, archived_at = NULL, updated_at = now()
		WHERE id = $1::uuid`, "clinic.restore")
}

func (s *Service) Delete(ctx context.Context, m Meta, id string) error {
	return s.flag(ctx, m, id, `DELETE FROM clinics WHERE id = $1::uuid`, "clinic.delete")
}

func (s *Service) flag(ctx context.Context, m Meta, id, sql, action string) error {
	pool, err := s.pool(ctx, m.TenantID)
	if err != nil {
		return err
	}
	if err := s.repo.FlagClinic(ctx, pool, sql, id); err != nil {
		log.Printf("flag clinic: %v", err)
		return errors.New("update failed")
	}
	s.audit(ctx, m, action, id)
	return nil
}

func (s *Service) MakePrimary(ctx context.Context, m Meta, id string) error {
	pool, err := s.pool(ctx, m.TenantID)
	if err != nil {
		return err
	}
	if err := s.repo.MakePrimary(ctx, pool, id); err != nil {
		return errors.New("set primary failed")
	}
	s.audit(ctx, m, "clinic.set_primary", id)
	return nil
}

func (s *Service) Join(ctx context.Context, m Meta, inviteCode string) (string, error) {
	inviteCode = strings.ToUpper(strings.TrimSpace(inviteCode))
	if len(inviteCode) != 6 {
		return "", ErrBadInvite
	}
	pool, err := s.pool(ctx, m.TenantID)
	if err != nil {
		return "", err
	}
	id, err := s.repo.FindClinicByInvite(ctx, pool, inviteCode)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", ErrInviteNotFound
		}
		return "", errors.New("db error")
	}
	if err := s.repo.JoinClinic(ctx, pool, id, m.UserID); err != nil {
		return "", errors.New("join failed")
	}
	s.audit(ctx, m, "clinic.join", id)
	return id, nil
}

// ── settings (communication, integration, print, numbering) ───────────────────

func (s *Service) GetJSONSettings(ctx context.Context, m Meta, table, clinicID string) ([]byte, error) {
	pool, err := s.poolAndMember(ctx, m.TenantID, clinicID, m.UserID)
	if err != nil {
		return nil, err
	}
	raw, err := s.repo.GetJSONSettings(ctx, pool, table, clinicID)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		log.Printf("get %s: %v", table, err)
		return nil, errors.New("db error")
	}
	if len(raw) == 0 {
		raw = []byte("{}")
	}
	return raw, nil
}

func (s *Service) PutJSONSettings(ctx context.Context, m Meta, table, clinicID string, body []byte) error {
	pool, err := s.poolAndMember(ctx, m.TenantID, clinicID, m.UserID)
	if err != nil {
		return err
	}
	if len(body) == 0 {
		body = []byte("{}")
	}
	if err := s.repo.PutJSONSettings(ctx, pool, table, clinicID, body); err != nil {
		log.Printf("put %s: %v", table, err)
		return errors.New("save failed")
	}
	return nil
}

// ── tax & payment ────────────────────────────────────────────────────────────

const paginationLimit = 8

func (s *Service) ListTaxes(ctx context.Context, m Meta, clinicID string, page int) ([]model.TaxDTO, int, error) {
	if page < 1 {
		page = 1
	}
	pool, err := s.pool(ctx, m.TenantID)
	if err != nil {
		return nil, 0, err
	}
	total, _ := s.repo.CountTaxes(ctx, pool, clinicID)
	out, err := s.repo.ListTaxes(ctx, pool, clinicID, paginationLimit, (page-1)*paginationLimit)
	return out, total, err
}

func (s *Service) CreateTax(ctx context.Context, m Meta, clinicID string, req model.CreateTaxRequest) (string, error) {
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		return "", ErrNameReq
	}
	pool, err := s.pool(ctx, m.TenantID)
	if err != nil {
		return "", err
	}
	return s.repo.CreateTax(ctx, pool, clinicID, req.Name, req.Rate)
}

func (s *Service) UpdateTax(ctx context.Context, m Meta, clinicID, taxID string, req model.UpdateTaxRequest) error {
	pool, err := s.pool(ctx, m.TenantID)
	if err != nil {
		return err
	}
	ok, err := s.repo.UpdateTax(ctx, pool, clinicID, taxID, req)
	if err != nil {
		return errors.New("update failed")
	}
	if !ok {
		return ErrNothingUpdate
	}
	return nil
}

func (s *Service) DeleteTax(ctx context.Context, m Meta, clinicID, taxID string) error {
	pool, err := s.poolAndMember(ctx, m.TenantID, clinicID, m.UserID)
	if err != nil {
		return err
	}
	return s.repo.DeleteTax(ctx, pool, clinicID, taxID)
}

func (s *Service) ListPaymentModes(ctx context.Context, m Meta, clinicID string, page int) ([]model.PaymentModeDTO, int, error) {
	if page < 1 {
		page = 1
	}
	pool, err := s.pool(ctx, m.TenantID)
	if err != nil {
		return nil, 0, err
	}
	total, _ := s.repo.CountPaymentModes(ctx, pool, clinicID)
	out, err := s.repo.ListPaymentModes(ctx, pool, clinicID, paginationLimit, (page-1)*paginationLimit)
	return out, total, err
}

func (s *Service) CreatePaymentMode(ctx context.Context, m Meta, clinicID string, req model.CreatePaymentModeRequest) (string, error) {
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		return "", ErrNameReq
	}
	pool, err := s.pool(ctx, m.TenantID)
	if err != nil {
		return "", err
	}
	return s.repo.CreatePaymentMode(ctx, pool, clinicID, req.Name)
}

func (s *Service) DeletePaymentMode(ctx context.Context, m Meta, clinicID, modeID string) error {
	pool, err := s.poolAndMember(ctx, m.TenantID, clinicID, m.UserID)
	if err != nil {
		return err
	}
	return s.repo.DeletePaymentMode(ctx, pool, clinicID, modeID)
}

// ── org setup (departments, specializations, designations) ────────────────────

func (s *Service) ListOrgItems(ctx context.Context, m Meta, table string, f model.OrgListFilter) ([]model.OrgItemDTO, int, error) {
	if !repository.IsOrgTable(table) {
		return nil, 0, ErrBadTable
	}
	pool, err := s.pool(ctx, m.TenantID)
	if err != nil {
		return nil, 0, err
	}
	total, _ := s.repo.CountOrgItems(ctx, pool, table, f.ClinicID, f.Search)
	out, err := s.repo.ListOrgItems(ctx, pool, table, f)
	return out, total, err
}

func (s *Service) CreateOrgItem(ctx context.Context, m Meta, table, clinicID string, req model.CreateOrgItemRequest) (string, error) {
	if !repository.IsOrgTable(table) {
		return "", ErrBadTable
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return "", ErrNameReq
	}
	pool, err := s.pool(ctx, m.TenantID)
	if err != nil {
		return "", err
	}
	return s.repo.CreateOrgItem(ctx, pool, table, clinicID, name, strings.TrimSpace(req.Description))
}

func (s *Service) UpdateOrgItem(ctx context.Context, m Meta, table, clinicID, itemID string, req model.UpdateOrgItemRequest) error {
	if !repository.IsOrgTable(table) {
		return ErrBadTable
	}
	if req.Name != nil && strings.TrimSpace(*req.Name) == "" {
		return ErrNameReq
	}
	pool, err := s.pool(ctx, m.TenantID)
	if err != nil {
		return err
	}
	if err := s.repo.UpdateOrgItem(ctx, pool, table, clinicID, itemID, req); err != nil {
		return errors.New("update failed")
	}
	return nil
}

func (s *Service) DeleteOrgItem(ctx context.Context, m Meta, table, clinicID, itemID string) error {
	if !repository.IsOrgTable(table) {
		return ErrBadTable
	}
	pool, err := s.pool(ctx, m.TenantID)
	if err != nil {
		return err
	}
	return s.repo.DeleteOrgItem(ctx, pool, table, clinicID, itemID)
}

func (s *Service) CopyOrgSetup(ctx context.Context, m Meta, sourceClinicID string, req model.CopyOrgRequest) ([]model.CopyOrgResult, error) {
	if !repository.IsOrgTable(req.Kind) {
		return nil, ErrBadKind
	}
	if len(req.TargetClinicIDs) == 0 {
		return nil, ErrTargetsReq
	}
	pool, err := s.poolAndMember(ctx, m.TenantID, sourceClinicID, m.UserID)
	if err != nil {
		if errors.Is(err, ErrForbidden) {
			return nil, ErrSourceForbid
		}
		return nil, err
	}
	results := make([]model.CopyOrgResult, 0, len(req.TargetClinicIDs))
	for _, target := range req.TargetClinicIDs {
		if target == sourceClinicID {
			continue
		}
		if !s.repo.IsMember(ctx, pool, target, m.UserID) {
			return nil, errors.New("not a member of target clinic " + target)
		}
		count, err := s.repo.CopyOrgTable(ctx, pool, req.Kind, sourceClinicID, target)
		if err != nil {
			return nil, err
		}
		results = append(results, model.CopyOrgResult{ClinicID: target, Inserted: count})
	}
	return results, nil
}

// ── patient categories & special statuses ────────────────────────────────────

func (s *Service) ListPatientCategories(ctx context.Context, m Meta, f model.PCListFilter) ([]model.PatientCategoryDTO, int, error) {
	pool, err := s.pool(ctx, m.TenantID)
	if err != nil {
		return nil, 0, err
	}
	total, _ := s.repo.CountPatientCategories(ctx, pool, f.ClinicID, f.Search)
	out, err := s.repo.ListPatientCategories(ctx, pool, f)
	return out, total, err
}

func (s *Service) CreatePatientCategory(ctx context.Context, m Meta, clinicID, name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", ErrNameReq
	}
	pool, err := s.pool(ctx, m.TenantID)
	if err != nil {
		return "", err
	}
	return s.repo.CreatePatientCategory(ctx, pool, clinicID, name)
}

func (s *Service) UpdatePatientCategory(ctx context.Context, m Meta, clinicID, catID, name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return ErrNameReq
	}
	pool, err := s.pool(ctx, m.TenantID)
	if err != nil {
		return err
	}
	n, err := s.repo.UpdatePatientCategory(ctx, pool, clinicID, catID, name)
	if err != nil {
		return errors.New("update failed")
	}
	if n == 0 {
		return ErrGenericNotFound
	}
	s.audit(ctx, m, "patient_category.update", name)
	return nil
}

func (s *Service) DeletePatientCategory(ctx context.Context, m Meta, clinicID, catID string) error {
	pool, err := s.pool(ctx, m.TenantID)
	if err != nil {
		return err
	}
	return s.repo.DeletePatientCategory(ctx, pool, clinicID, catID)
}

func (s *Service) ListSpecialStatuses(ctx context.Context, m Meta, f model.PCListFilter) ([]model.SpecialStatusDTO, int, error) {
	pool, err := s.pool(ctx, m.TenantID)
	if err != nil {
		return nil, 0, err
	}
	total, _ := s.repo.CountSpecialStatuses(ctx, pool, f.ClinicID, f.Search)
	out, err := s.repo.ListSpecialStatuses(ctx, pool, f)
	return out, total, err
}

func (s *Service) CreateSpecialStatus(ctx context.Context, m Meta, clinicID string, req model.CreateSpecialStatusRequest) (string, error) {
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		return "", ErrNameReq
	}
	if req.Color == "" {
		req.Color = "#BA8936"
	}
	pool, err := s.pool(ctx, m.TenantID)
	if err != nil {
		return "", err
	}
	return s.repo.CreateSpecialStatus(ctx, pool, clinicID, req.Name, req.Color)
}

func (s *Service) UpdateSpecialStatus(ctx context.Context, m Meta, clinicID, statusID string, req model.UpdateSpecialStatusRequest) error {
	if req.Name != nil && strings.TrimSpace(*req.Name) == "" {
		return ErrNameReq
	}
	pool, err := s.pool(ctx, m.TenantID)
	if err != nil {
		return err
	}
	if err := s.repo.UpdateSpecialStatus(ctx, pool, clinicID, statusID, req); err != nil {
		return errors.New("update failed")
	}
	name := ""
	if req.Name != nil {
		name = strings.TrimSpace(*req.Name)
	}
	s.audit(ctx, m, "special_status.update", name)
	return nil
}

func (s *Service) DeleteSpecialStatus(ctx context.Context, m Meta, clinicID, statusID string) error {
	pool, err := s.pool(ctx, m.TenantID)
	if err != nil {
		return err
	}
	return s.repo.DeleteSpecialStatus(ctx, pool, clinicID, statusID)
}

// ── numbering (typed helpers) ────────────────────────────────────────────────

func (s *Service) GetNumbering(ctx context.Context, m Meta, clinicID string) (model.NumberingSettings, error) {
	raw, err := s.GetJSONSettings(ctx, m, "numbering_settings", clinicID)
	if err != nil {
		return model.NumberingSettings{}, err
	}
	out := model.DefaultNumbering()
	if len(raw) > 0 {
		_ = jsonUnmarshal(raw, &out)
	}
	return out, nil
}

func (s *Service) PutNumbering(ctx context.Context, m Meta, clinicID string, settings model.NumberingSettings) error {
	b, _ := json.Marshal(settings)
	return s.PutJSONSettings(ctx, m, "numbering_settings", clinicID, b)
}

func jsonUnmarshal(data []byte, v any) error { return json.Unmarshal(data, v) }
