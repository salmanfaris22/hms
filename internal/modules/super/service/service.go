package service

import (
	"context"
	"errors"
	"log"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/salman/hms-backend/internal/core/audit"
	coreauth "github.com/salman/hms-backend/internal/core/auth"
	"github.com/salman/hms-backend/internal/modules/super/model"
	"github.com/salman/hms-backend/internal/modules/super/repository"
	"github.com/salman/hms-backend/pkg/email"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrInvalidBody        = errors.New("invalid body")
	ErrNothingToUpdate    = errors.New("nothing to update")
	ErrTenantNotFound     = errors.New("tenant not found")
	ErrNoAdminEmail       = errors.New("tenant has no admin email")
	ErrEmailSend          = errors.New("email send failed")
)

type Meta struct {
	IP         string
	UserAgent  string
	ActorID    string
	ActorEmail string
}

type Service struct {
	repo      *repository.Repository
	jwtSecret string
	mailer    email.Sender
	appURL    string
}

func New(repo *repository.Repository, jwtSecret string, mailer email.Sender, appURL string) *Service {
	return &Service{repo: repo, jwtSecret: jwtSecret, mailer: mailer, appURL: appURL}
}

func (s *Service) Login(ctx context.Context, meta Meta, req model.LoginRequest) (model.LoginResponse, error) {
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	if req.Email == "" || req.Password == "" {
		return model.LoginResponse{}, ErrInvalidBody
	}

	row, err := s.repo.FindSuperAdmin(ctx, req.Email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			s.logFail(ctx, meta, "", req.Email, "not_found")
			return model.LoginResponse{}, ErrInvalidCredentials
		}
		log.Printf("super login: %v", err)
		return model.LoginResponse{}, err
	}
	if !coreauth.CheckPassword(row.PasswordHash, req.Password) {
		s.logFail(ctx, meta, row.ID, row.Email, "bad_password")
		return model.LoginResponse{}, ErrInvalidCredentials
	}
	token, err := coreauth.IssueSuperToken(s.jwtSecret, row.ID, row.Email, 12*time.Hour)
	if err != nil {
		return model.LoginResponse{}, err
	}

	audit.LogRaw(ctx, s.repo.Registry(), meta.IP, meta.UserAgent, audit.Event{
		ActorID:    row.ID,
		ActorEmail: row.Email,
		ActorKind:  audit.ActorSuper,
		Action:     "super.login.success",
	})

	return model.LoginResponse{
		Token: token,
		Admin: model.AdminDTO{ID: row.ID, Email: row.Email, FullName: row.FullName},
	}, nil
}

func (s *Service) logFail(ctx context.Context, meta Meta, id, email, reason string) {
	audit.LogRaw(ctx, s.repo.Registry(), meta.IP, meta.UserAgent, audit.Event{
		ActorID:    id,
		ActorEmail: email,
		ActorKind:  audit.ActorSuper,
		Action:     "super.login.failed",
		Metadata:   map[string]any{"reason": reason},
	})
}

func (s *Service) ListTenants(ctx context.Context) ([]model.TenantDTO, error) {
	out, err := s.repo.ListTenants(ctx)
	if err != nil {
		return nil, err
	}
	for i := range out {
		if stats, err := s.repo.TenantStats(ctx, out[i].ID, out[i].DBName); err == nil {
			out[i].Stats = stats
		}
	}
	return out, nil
}

func (s *Service) CreateTenant(ctx context.Context, meta Meta, req model.CreateTenantRequest) (map[string]any, error) {
	req.Slug = strings.ToLower(strings.TrimSpace(req.Slug))
	req.Name = strings.TrimSpace(req.Name)
	req.AdminEmail = strings.ToLower(strings.TrimSpace(req.AdminEmail))
	if req.Slug == "" || req.Name == "" || req.AdminEmail == "" || req.AdminPassword == "" {
		return nil, errors.New("slug, name, adminEmail and adminPassword are required")
	}
	if len(req.AdminPassword) < 8 {
		return nil, errors.New("admin password must be at least 8 characters")
	}
	if req.MaxClinics <= 0 {
		req.MaxClinics = 5
	}
	if req.StorageQuotaGB <= 0 {
		req.StorageQuotaGB = 10
	}
	if len(req.Modules) == 0 {
		req.Modules = []string{"dashboard", "appointments", "patients", "billing"}
	}
	if req.AdminFullName == "" {
		req.AdminFullName = "Hospital Admin"
	}
	if req.PhoneCountry == "" {
		req.PhoneCountry = "+91"
	}
	if req.Phones == nil {
		req.Phones = []model.TenantPhone{}
	}

	info, pool, err := s.repo.Tenants().Provision(ctx, req.Slug, req.Name)
	if err != nil {
		log.Printf("provision tenant: %v", err)
		return nil, errors.New("provision failed")
	}

	if err := s.repo.UpdateTenantSettings(
		ctx, info.ID, req.Phone, req.PhoneCountry, req.Phones,
		req.MaxClinics, req.StorageQuotaGB, req.Modules,
		req.SubscriptionStart, req.SubscriptionEnd,
	); err != nil {
		log.Printf("update tenant settings: %v", err)
		return nil, errors.New("settings update failed")
	}

	hash, err := coreauth.HashPassword(req.AdminPassword)
	if err != nil {
		return nil, errors.New("hash failed")
	}

	adminID, err := s.repo.UpsertAdminUser(ctx, pool,
		req.AdminEmail, hash, req.AdminFullName, req.AdminPhone, req.AdminPhoneCountry)
	if err != nil {
		log.Printf("insert admin user: %v", err)
		return nil, errors.New("admin user failed")
	}
	if err := s.repo.UpsertDirectory(ctx, req.AdminEmail, info.ID, adminID); err != nil {
		log.Printf("register directory: %v", err)
		return nil, errors.New("directory insert failed")
	}

	go s.sendWelcome(info.ID, req.Name, req.AdminEmail, req.AdminPassword)

	audit.LogRaw(ctx, s.repo.Registry(), meta.IP, meta.UserAgent, audit.Event{
		ActorID:    meta.ActorID,
		ActorEmail: meta.ActorEmail,
		ActorKind:  audit.ActorSuper,
		TenantID:   info.ID,
		Action:     "tenant.create",
		Resource:   info.Slug,
		Metadata: map[string]any{
			"name":       info.Name,
			"adminEmail": req.AdminEmail,
			"modules":    req.Modules,
		},
	})

	return map[string]any{
		"id":    info.ID,
		"slug":  info.Slug,
		"name":  info.Name,
		"admin": req.AdminEmail,
	}, nil
}

func (s *Service) sendWelcome(tenantID, name, adminEmail, password string) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	subject, html := email.WelcomeHTML(name, adminEmail, password, s.appURL)
	if err := s.mailer.Send(ctx, adminEmail, subject, html); err != nil {
		log.Printf("welcome email to %s failed: %v", adminEmail, err)
		audit.LogRaw(ctx, s.repo.Registry(), "", "", audit.Event{
			ActorKind: audit.ActorSystem,
			TenantID:  tenantID,
			Action:    "email.welcome.failed",
			Metadata:  map[string]any{"to": adminEmail, "error": err.Error()},
		})
		return
	}
	s.repo.MarkWelcomeSent(ctx, tenantID)
	audit.LogRaw(ctx, s.repo.Registry(), "", "", audit.Event{
		ActorKind: audit.ActorSystem,
		TenantID:  tenantID,
		Action:    "email.welcome.sent",
		Metadata:  map[string]any{"to": adminEmail},
	})
}

func (s *Service) UpdateTenant(ctx context.Context, meta Meta, id string, req model.UpdateTenantRequest) error {
	ok, err := s.repo.UpdateTenant(ctx, id, req)
	if err != nil {
		log.Printf("update tenant: %v", err)
		return errors.New("update failed")
	}
	if !ok {
		return ErrNothingToUpdate
	}
	audit.LogRaw(ctx, s.repo.Registry(), meta.IP, meta.UserAgent, audit.Event{
		ActorID: meta.ActorID, ActorEmail: meta.ActorEmail, ActorKind: audit.ActorSuper,
		TenantID: id, Action: "tenant.update",
		Metadata: map[string]any{"fields": SetNames(req)},
	})
	return nil
}

func (s *Service) DeleteTenant(ctx context.Context, meta Meta, id string) error {
	if err := s.repo.DeleteTenant(ctx, id); err != nil {
		log.Printf("delete tenant: %v", err)
		return errors.New("delete failed")
	}
	audit.LogRaw(ctx, s.repo.Registry(), meta.IP, meta.UserAgent, audit.Event{
		ActorID: meta.ActorID, ActorEmail: meta.ActorEmail, ActorKind: audit.ActorSuper,
		TenantID: id, Action: "tenant.delete",
	})
	return nil
}

func (s *Service) SendSubscriptionReminder(ctx context.Context, meta Meta, id string) error {
	info, err := s.repo.TenantForReminder(ctx, id)
	if err != nil {
		return ErrTenantNotFound
	}
	if info.AdminEmail == "" {
		return ErrNoAdminEmail
	}
	when := time.Now().AddDate(0, 0, 7)
	if info.SubEnd != nil {
		when = *info.SubEnd
	}
	subject, html := email.SubscriptionEndingHTML(info.Name, when, s.appURL)
	if err := s.mailer.Send(ctx, info.AdminEmail, subject, html); err != nil {
		return err
	}
	audit.LogRaw(ctx, s.repo.Registry(), meta.IP, meta.UserAgent, audit.Event{
		ActorKind: audit.ActorSystem, TenantID: id,
		Action: "email.subscription_reminder.sent",
		Metadata: map[string]any{"to": info.AdminEmail},
	})
	return nil
}

func SetNames(r model.UpdateTenantRequest) []string {
	out := []string{}
	if r.Name != nil {
		out = append(out, "name")
	}
	if r.Phone != nil {
		out = append(out, "phone")
	}
	if r.Phones != nil {
		out = append(out, "phones")
	}
	if r.MaxClinics != nil {
		out = append(out, "maxClinics")
	}
	if r.StorageQuotaGB != nil {
		out = append(out, "storageQuotaGB")
	}
	if r.Modules != nil {
		out = append(out, "modules")
	}
	if r.SubscriptionStart != nil {
		out = append(out, "subscriptionStart")
	}
	if r.SubscriptionEnd != nil {
		out = append(out, "subscriptionEnd")
	}
	if r.IsActive != nil {
		out = append(out, "isActive")
	}
	return out
}
