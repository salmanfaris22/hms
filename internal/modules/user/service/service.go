package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/salman/hms-backend/internal/core/audit"
	coreauth "github.com/salman/hms-backend/internal/core/auth"
	"github.com/salman/hms-backend/internal/infrastructure/persistence/postgres"
	"github.com/salman/hms-backend/internal/modules/user/model"
	"github.com/salman/hms-backend/internal/modules/user/repository"
)

var (
	ErrInvalidCredentials  = errors.New("invalid credentials")
	ErrOrgInactive         = errors.New("organisation is inactive — contact your administrator")
	ErrSubscriptionExpired = errors.New("subscription has ended — please renew to regain access")
	ErrTenantUnavailable   = errors.New("tenant unavailable")
)

type Service struct {
	repo      *repository.Repository
	tenants   *postgres.TenantResolver
	jwtSecret string
}

func New(repo *repository.Repository, tenants *postgres.TenantResolver, jwtSecret string) *Service {
	return &Service{repo: repo, tenants: tenants, jwtSecret: jwtSecret}
}

func (s *Service) Login(ctx context.Context, meta model.RequestMeta, email, password string) (model.LoginResponse, error) {
	email = strings.ToLower(strings.TrimSpace(email))

	entry, err := s.repo.FindDirectory(ctx, email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			s.logFailed(ctx, meta, email, "", "user_not_found")
			return model.LoginResponse{}, ErrInvalidCredentials
		}
		return model.LoginResponse{}, err
	}

	status, err := s.repo.GetTenantStatus(ctx, entry.TenantID)
	if err != nil {
		return model.LoginResponse{}, err
	}
	if !status.IsActive {
		s.logFailed(ctx, meta, email, entry.TenantID, "org_inactive")
		return model.LoginResponse{}, ErrOrgInactive
	}
	if status.SubEnd != nil && status.SubEnd.Before(time.Now()) {
		s.logFailed(ctx, meta, email, entry.TenantID, "subscription_expired")
		return model.LoginResponse{}, ErrSubscriptionExpired
	}

	pool, err := s.tenants.Pool(ctx, entry.TenantID)
	if err != nil {
		return model.LoginResponse{}, ErrTenantUnavailable
	}

	u, err := s.repo.FindUser(ctx, pool, entry.UserID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			s.logFailed(ctx, meta, email, entry.TenantID, "user_inactive")
			return model.LoginResponse{}, ErrInvalidCredentials
		}
		return model.LoginResponse{}, err
	}

	if !coreauth.CheckPassword(u.PasswordHash, password) {
		s.logFailed(ctx, meta, email, entry.TenantID, "bad_password")
		return model.LoginResponse{}, ErrInvalidCredentials
	}

	s.repo.TouchLastLogin(ctx, pool, entry.UserID)

	token, err := coreauth.IssueToken(s.jwtSecret, entry.UserID, entry.TenantID, u.Email, u.Role, 12*time.Hour)
	if err != nil {
		return model.LoginResponse{}, err
	}

	audit.LogRaw(ctx, s.repo.Registry(), meta.IP, meta.UserAgent, audit.Event{
		ActorID:    entry.UserID,
		ActorEmail: u.Email,
		ActorKind:  audit.ActorUser,
		TenantID:   entry.TenantID,
		Action:     "login.success",
	})

	return model.LoginResponse{
		Token: token,
		User: model.UserDTO{
			ID:       entry.UserID,
			Email:    u.Email,
			FullName: u.FullName,
			TenantID: entry.TenantID,
			Role:     u.Role,
			Modules:  status.Modules,
		},
	}, nil
}

func (s *Service) Me(ctx context.Context, claims *coreauth.Claims) (model.UserDTO, error) {
	status, err := s.repo.GetTenantStatus(ctx, claims.TenantID)
	if err != nil {
		return model.UserDTO{}, err
	}
	if !status.IsActive {
		return model.UserDTO{}, ErrOrgInactive
	}
	if status.SubEnd != nil && status.SubEnd.Before(time.Now()) {
		return model.UserDTO{}, ErrSubscriptionExpired
	}

	pool, err := s.tenants.Pool(ctx, claims.TenantID)
	if err != nil {
		return model.UserDTO{}, ErrTenantUnavailable
	}

	u, err := s.repo.FindUser(ctx, pool, claims.UserID)
	if err != nil {
		return model.UserDTO{}, ErrInvalidCredentials
	}

	return model.UserDTO{
		ID:       claims.UserID,
		Email:    u.Email,
		FullName: u.FullName,
		TenantID: claims.TenantID,
		Role:     u.Role,
		Modules:  status.Modules,
	}, nil
}

func (s *Service) logFailed(ctx context.Context, meta model.RequestMeta, email, tenantID, reason string) {
	audit.LogRaw(ctx, s.repo.Registry(), meta.IP, meta.UserAgent, audit.Event{
		ActorEmail: email,
		ActorKind:  audit.ActorUser,
		TenantID:   tenantID,
		Action:     "login.failed",
		Metadata:   map[string]any{"reason": reason},
	})
}
