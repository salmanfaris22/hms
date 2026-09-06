package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/salman/hms-backend/internal/core/auth"
	"github.com/salman/hms-backend/internal/modules/staff/model"
	"github.com/salman/hms-backend/internal/modules/staff/repository"
)

var (
	ErrTenantUnavail = errors.New("tenant unavailable")
	ErrForbidden     = errors.New("not a member of this clinic")
	ErrBadBody       = errors.New("invalid body")
	ErrNameReq       = errors.New("name is required")
	ErrEmailReq      = errors.New("email is required")
	ErrEmailUsed     = errors.New("email already in use")
	ErrCodeUsed      = errors.New("staff code already in use")
	ErrNotFound      = errors.New("staff not found")
)

type Meta struct {
	IP, UserAgent, ActorID, ActorEmail, TenantID, UserID string
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

// ── Staff ───────────────────────────────────────────────────────────────────

func (s *Service) ListStaff(ctx context.Context, m Meta, clinicID string, f model.ListFilter) ([]model.StaffDTO, int, error) {
	pool, err := s.poolAndMember(ctx, m.TenantID, clinicID, m.UserID)
	if err != nil {
		return nil, 0, err
	}
	f.ClinicID = clinicID
	return s.repo.ListStaff(ctx, pool, f)
}

func (s *Service) GetStaff(ctx context.Context, m Meta, clinicID, userID string) (*model.StaffDTO, error) {
	pool, err := s.poolAndMember(ctx, m.TenantID, clinicID, m.UserID)
	if err != nil {
		return nil, err
	}
	dto, err := s.repo.GetStaff(ctx, pool, clinicID, userID)
	if err != nil {
		return nil, ErrNotFound
	}
	return dto, nil
}

func (s *Service) CreateStaff(ctx context.Context, m Meta, clinicID string, req model.CreateStaffRequest) (string, error) {
	pool, err := s.poolAndMember(ctx, m.TenantID, clinicID, m.UserID)
	if err != nil {
		return "", err
	}

	email := strings.ToLower(strings.TrimSpace(req.Email))
	if email == "" {
		return "", ErrEmailReq
	}
	first := strings.TrimSpace(req.FirstName)
	if first == "" {
		return "", ErrNameReq
	}

	// staff code
	code := strings.TrimSpace(req.StaffCode)
	if req.AutoGenerateCode || code == "" {
		c, err := s.repo.NextStaffCode(ctx, pool)
		if err != nil {
			return "", err
		}
		code = c
	} else if s.repo.StaffCodeExists(ctx, pool, code) {
		return "", ErrCodeUsed
	}

	// existing user lookup
	tenantID, existingUserID, _ := s.repo.DirectoryLookup(ctx, email)
	if existingUserID != "" && tenantID != m.TenantID {
		return "", ErrEmailUsed
	}

	password := req.Password
	if password == "" {
		password = "Welcome@123"
	}
	hash, err := auth.HashPassword(password)
	if err != nil {
		return "", err
	}

	full := strings.TrimSpace(strings.Join([]string{req.FirstName, req.MiddleName, req.LastName}, " "))
	full = strings.Join(strings.Fields(full), " ")

	id, err := s.repo.CreateStaff(ctx, pool, repository.CreateStaffParams{
		UserID:           existingUserID,
		Email:            email,
		PasswordHash:     hash,
		FullName:         full,
		StaffCode:        code,
		FirstName:        req.FirstName,
		MiddleName:       req.MiddleName,
		LastName:         req.LastName,
		ProfilePhoto:     req.ProfilePhoto,
		DepartmentID:     req.DepartmentID,
		DesignationID:    req.DesignationID,
		SpecializationID: req.SpecializationID,
		MobileCountry:    req.MobileCountry,
		MobileNumber:     req.MobileNumber,
		AdditionalMobile: req.AdditionalMobile,
		LandlineNumber:   req.LandlineNumber,
		ViewInEMR:        req.ViewInEMR,
		WorkLocations:    req.WorkLocations,
		PrimaryClinicID:  clinicID,
	})
	if err != nil {
		return "", fmt.Errorf("create staff: %w", err)
	}

	if err := s.repo.UpsertDirectory(ctx, email, m.TenantID, id); err != nil {
		return "", fmt.Errorf("upsert directory: %w", err)
	}
	return id, nil
}

func (s *Service) UpdateStaff(ctx context.Context, m Meta, clinicID, userID string, req model.UpdateStaffRequest) error {
	pool, err := s.poolAndMember(ctx, m.TenantID, clinicID, m.UserID)
	if err != nil {
		return err
	}
	return s.repo.UpdateStaff(ctx, pool, userID, req)
}

func (s *Service) DeactivateStaff(ctx context.Context, m Meta, clinicID, userID string) error {
	pool, err := s.poolAndMember(ctx, m.TenantID, clinicID, m.UserID)
	if err != nil {
		return err
	}
	return s.repo.DeactivateStaff(ctx, pool, userID)
}

func (s *Service) Stats(ctx context.Context, m Meta, clinicID string) (model.StatsDTO, error) {
	pool, err := s.poolAndMember(ctx, m.TenantID, clinicID, m.UserID)
	if err != nil {
		return model.StatsDTO{}, err
	}
	return s.repo.Stats(ctx, pool, clinicID)
}

// ── Roles ───────────────────────────────────────────────────────────────────

func (s *Service) ListRoles(ctx context.Context, m Meta, clinicID string) ([]model.RoleDTO, error) {
	pool, err := s.poolAndMember(ctx, m.TenantID, clinicID, m.UserID)
	if err != nil {
		return nil, err
	}
	return s.repo.ListRoles(ctx, pool, clinicID)
}

func (s *Service) CreateRole(ctx context.Context, m Meta, clinicID string, req model.CreateRoleRequest) (string, error) {
	if strings.TrimSpace(req.Name) == "" {
		return "", ErrNameReq
	}
	pool, err := s.poolAndMember(ctx, m.TenantID, clinicID, m.UserID)
	if err != nil {
		return "", err
	}
	return s.repo.CreateRole(ctx, pool, clinicID, req)
}

func (s *Service) UpdateRole(ctx context.Context, m Meta, clinicID, roleID string, req model.UpdateRoleRequest) error {
	pool, err := s.poolAndMember(ctx, m.TenantID, clinicID, m.UserID)
	if err != nil {
		return err
	}
	return s.repo.UpdateRole(ctx, pool, clinicID, roleID, req)
}

func (s *Service) DeleteRole(ctx context.Context, m Meta, clinicID, roleID string) error {
	pool, err := s.poolAndMember(ctx, m.TenantID, clinicID, m.UserID)
	if err != nil {
		return err
	}
	return s.repo.DeleteRole(ctx, pool, clinicID, roleID)
}

func (s *Service) AssignRoles(ctx context.Context, m Meta, clinicID string, req model.AssignRolesRequest) error {
	pool, err := s.poolAndMember(ctx, m.TenantID, clinicID, m.UserID)
	if err != nil {
		return err
	}
	return s.repo.AssignRoles(ctx, pool, clinicID, req.UserIDs, req.RoleIDs)
}

// ── Documents ───────────────────────────────────────────────────────────────

func (s *Service) ListDocuments(ctx context.Context, m Meta, clinicID, userID string) ([]model.DocumentDTO, error) {
	pool, err := s.poolAndMember(ctx, m.TenantID, clinicID, m.UserID)
	if err != nil {
		return nil, err
	}
	return s.repo.ListDocuments(ctx, pool, userID)
}

func (s *Service) CreateDocument(ctx context.Context, m Meta, clinicID, userID string, req model.CreateDocumentRequest) (string, error) {
	pool, err := s.poolAndMember(ctx, m.TenantID, clinicID, m.UserID)
	if err != nil {
		return "", err
	}
	return s.repo.CreateDocument(ctx, pool, userID, req)
}

func (s *Service) DeleteDocument(ctx context.Context, m Meta, clinicID, userID, docID string) error {
	pool, err := s.poolAndMember(ctx, m.TenantID, clinicID, m.UserID)
	if err != nil {
		return err
	}
	return s.repo.DeleteDocument(ctx, pool, userID, docID)
}
