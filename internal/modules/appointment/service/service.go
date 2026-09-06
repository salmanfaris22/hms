package service

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/salman/hms-backend/internal/core/audit"
	"github.com/salman/hms-backend/internal/modules/appointment/model"
	"github.com/salman/hms-backend/internal/modules/appointment/repository"
	patientService "github.com/salman/hms-backend/internal/modules/patient/service"
)

var (
	ErrTenantUnavail = errors.New("tenant unavailable")
	ErrForbidden     = errors.New("not a member of this clinic")
	ErrBadBody       = errors.New("invalid body")
	ErrNoUpdate      = errors.New("nothing to update")
	ErrBadTime       = errors.New("scheduledAt must be RFC3339")
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
	patients *patientService.Service
}

func New(repo *repository.Repository, patients *patientService.Service) *Service {
	return &Service{repo: repo, patients: patients}
}

func (s *Service) Book(ctx context.Context, meta Meta, req model.BookRequest) (string, error) {
	if req.ClinicID == "" || req.PatientID == "" || req.ScheduledAt == "" {
		return "", errors.New("clinicId, patientId, scheduledAt are required")
	}
	if req.DurationMin <= 0 {
		req.DurationMin = 30
	}
	scheduled, err := time.Parse(time.RFC3339, req.ScheduledAt)
	if err != nil {
		return "", ErrBadTime
	}

	pool, err := s.patients.Tenants().Pool(ctx, meta.TenantID)
	if err != nil {
		return "", ErrTenantUnavail
	}
	if err := s.patients.AssertMember(ctx, pool, meta.UserID, req.ClinicID); err != nil {
		return "", ErrForbidden
	}

	id, err := s.repo.Book(ctx, pool, req, scheduled)
	if err != nil {
		log.Printf("book appointment: %v", err)
		return "", errors.New("create failed")
	}
	s.repo.RecordVisitEvent(ctx, pool, req.PatientID, req.ProviderName, req.Kind, scheduled)

	audit.LogRaw(ctx, s.patients.Tenants().Registry(), meta.IP, meta.UserAgent, audit.Event{
		ActorID: meta.ActorID, ActorEmail: meta.ActorEmail, ActorKind: audit.ActorUser,
		TenantID: meta.TenantID, Action: "appointment.book", Resource: id,
		Metadata: map[string]any{"patientId": req.PatientID, "provider": req.ProviderName, "kind": req.Kind},
	})
	return id, nil
}

func (s *Service) Update(ctx context.Context, meta Meta, id string, req model.UpdateRequest) error {
	pool, err := s.patients.Tenants().Pool(ctx, meta.TenantID)
	if err != nil {
		return ErrTenantUnavail
	}
	ok, err := s.repo.Update(ctx, pool, id, meta.UserID, req)
	if err != nil {
		if err == ErrBadTime || errors.Is(err, ErrBadTime) {
			return ErrBadTime
		}
		if _, perr := time.Parse(time.RFC3339, ""); perr != nil && req.ScheduledAt != nil {
			// do nothing, err path covered
		}
		log.Printf("update appointment: %v", err)
		return errors.New("update failed")
	}
	if !ok {
		return ErrNoUpdate
	}
	audit.LogRaw(ctx, s.patients.Tenants().Registry(), meta.IP, meta.UserAgent, audit.Event{
		ActorID: meta.ActorID, ActorEmail: meta.ActorEmail, ActorKind: audit.ActorUser,
		TenantID: meta.TenantID, Action: "appointment.update", Resource: id,
	})
	return nil
}

func (s *Service) GetSettings(ctx context.Context, meta Meta, clinicID string) (model.SettingsDTO, error) {
	pool, err := s.patients.Tenants().Pool(ctx, meta.TenantID)
	if err != nil {
		return model.SettingsDTO{}, ErrTenantUnavail
	}
	if err := s.patients.AssertMember(ctx, pool, meta.UserID, clinicID); err != nil {
		return model.SettingsDTO{}, ErrForbidden
	}
	out, err := s.repo.GetSettings(ctx, pool, clinicID)
	if err != nil && !errSkipNoRows(err) {
		log.Printf("get appt settings: %v", err)
		return out, errors.New("db error")
	}
	return out, nil
}

func (s *Service) PutSettings(ctx context.Context, meta Meta, clinicID string, req model.PutSettingsRequest) error {
	pool, err := s.patients.Tenants().Pool(ctx, meta.TenantID)
	if err != nil {
		return ErrTenantUnavail
	}
	if err := s.patients.AssertMember(ctx, pool, meta.UserID, clinicID); err != nil {
		return ErrForbidden
	}
	slot := 30
	onlineConf := false
	followupOn := false
	setBy := "default"
	defaultCount := 2
	defaultDays := 30
	rules := []model.FollowupRule{}

	if req.SlotDurationMin != nil {
		slot = *req.SlotDurationMin
	}
	if req.OnlineBookingConfirmation != nil {
		onlineConf = *req.OnlineBookingConfirmation
	}
	if req.FreeFollowupEnabled != nil {
		followupOn = *req.FreeFollowupEnabled
	}
	if req.FollowupSetBy != nil {
		setBy = *req.FollowupSetBy
	}
	if req.DefaultFollowupCount != nil {
		defaultCount = *req.DefaultFollowupCount
	}
	if req.DefaultFollowupDays != nil {
		defaultDays = *req.DefaultFollowupDays
	}
	if req.FollowupRules != nil {
		rules = *req.FollowupRules
	}

	if err := s.repo.PutSettings(ctx, pool, clinicID, slot, onlineConf, followupOn, setBy, defaultCount, defaultDays, rules); err != nil {
		log.Printf("put appt settings: %v", err)
		return errors.New("save failed")
	}
	return nil
}

func errSkipNoRows(err error) bool {
	return err != nil && errors.Is(err, pgx.ErrNoRows)
}
