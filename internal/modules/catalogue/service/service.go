package service

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/salman/hms-backend/internal/modules/catalogue/model"
	"github.com/salman/hms-backend/internal/modules/catalogue/repository"
	patientService "github.com/salman/hms-backend/internal/modules/patient/service"
)

var (
	ErrTenantUnavail = errors.New("tenant unavailable")
	ErrForbidden     = errors.New("not a member of this clinic")
	ErrClinicReq     = errors.New("clinic is required")
	ErrBadBody       = errors.New("invalid body")
	ErrNameReq       = errors.New("name is required")
	ErrClinicNameReq = errors.New("clinicId and name are required")
	ErrClinicDrugReq = errors.New("clinicId and drugName are required")
	ErrDrugNameReq   = errors.New("drugName is required")
	ErrClinicTestReq = errors.New("clinicId and testName are required")
	ErrTestNameReq   = errors.New("testName is required")
	ErrClinicTreatReq = errors.New("clinicId and treatmentName are required")
	ErrTreatNameReq  = errors.New("treatmentName is required")
	ErrParamNameReq  = errors.New("parameterName is required")
	ErrParamIDsReq   = errors.New("testId or clinicId is required")
	ErrGroupReq      = errors.New("clinicId and groupName required")
	ErrGroupNameReq  = errors.New("groupName required")
	ErrCopyReq       = errors.New("sourceClinicId and targetClinicIds are required")
	ErrNotFound      = errors.New("not found")
	ErrTestNotFound  = errors.New("test not found")
)

type Service struct {
	repo     *repository.Repository
	patients *patientService.Service
}

func New(repo *repository.Repository, patients *patientService.Service) *Service {
	return &Service{repo: repo, patients: patients}
}

func (s *Service) Patients() *patientService.Service { return s.patients }

func (s *Service) poolAndMember(ctx context.Context, tenantID, userID, clinicID string) (*pgxpool.Pool, error) {
	pool, err := s.repo.Pool(ctx, tenantID)
	if err != nil {
		return nil, ErrTenantUnavail
	}
	if err := s.repo.AssertMember(ctx, pool, userID, clinicID); err != nil {
		return nil, ErrForbidden
	}
	return pool, nil
}

func (s *Service) pool(ctx context.Context, tenantID string) (*pgxpool.Pool, error) {
	pool, err := s.repo.Pool(ctx, tenantID)
	if err != nil {
		return nil, ErrTenantUnavail
	}
	return pool, nil
}

// ── drugs ────────────────────────────────────────────────────────────────────

func (s *Service) ListDrugs(ctx context.Context, tenantID, userID string, f model.DrugListFilter) ([]model.DrugDTO, int, error) {
	if f.ClinicID == "" {
		return nil, 0, ErrClinicReq
	}
	pool, err := s.poolAndMember(ctx, tenantID, userID, f.ClinicID)
	if err != nil {
		return nil, 0, err
	}
	return s.repo.ListDrugs(ctx, pool, f)
}

func (s *Service) CreateDrug(ctx context.Context, tenantID, userID string, req model.CreateDrugRequest) (string, error) {
	if req.ClinicID == "" || strings.TrimSpace(req.DrugName) == "" {
		return "", ErrClinicDrugReq
	}
	pool, err := s.poolAndMember(ctx, tenantID, userID, req.ClinicID)
	if err != nil {
		return "", err
	}
	return s.repo.CreateDrug(ctx, pool, req)
}

func (s *Service) UpdateDrug(ctx context.Context, tenantID, userID, id string, req model.CreateDrugRequest) error {
	if strings.TrimSpace(req.DrugName) == "" {
		return ErrDrugNameReq
	}
	pool, err := s.pool(ctx, tenantID)
	if err != nil {
		return err
	}
	n, err := s.repo.UpdateDrug(ctx, pool, id, userID, req)
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Service) DeleteDrug(ctx context.Context, tenantID, userID, id string) error {
	pool, err := s.pool(ctx, tenantID)
	if err != nil {
		return err
	}
	n, err := s.repo.DeleteDrug(ctx, pool, id, userID)
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// ── lookups ──────────────────────────────────────────────────────────────────

func (s *Service) ListLookup(ctx context.Context, tenantID, userID, table, clinicID, kind string) ([]model.LookupDTO, error) {
	if clinicID == "" {
		return nil, ErrClinicReq
	}
	pool, err := s.poolAndMember(ctx, tenantID, userID, clinicID)
	if err != nil {
		return nil, err
	}
	return s.repo.ListLookup(ctx, pool, table, clinicID, kind)
}

func (s *Service) CreateLookup(ctx context.Context, tenantID, userID, table, kind string, req model.LookupCreateRequest) (string, string, error) {
	name := strings.TrimSpace(req.Name)
	if req.ClinicID == "" || name == "" {
		return "", "", ErrClinicNameReq
	}
	pool, err := s.poolAndMember(ctx, tenantID, userID, req.ClinicID)
	if err != nil {
		return "", "", err
	}
	id, err := s.repo.CreateLookup(ctx, pool, table, req.ClinicID, name, kind)
	return id, name, err
}

func (s *Service) DeleteLookup(ctx context.Context, tenantID, userID, table, id string) error {
	pool, err := s.pool(ctx, tenantID)
	if err != nil {
		return err
	}
	n, err := s.repo.DeleteLookup(ctx, pool, table, id, userID)
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// ── treatments ───────────────────────────────────────────────────────────────

func (s *Service) ListTreatments(ctx context.Context, tenantID, userID string, f model.TreatmentListFilter) ([]model.TreatmentDTO, int, error) {
	if f.ClinicID == "" {
		return nil, 0, ErrClinicReq
	}
	pool, err := s.poolAndMember(ctx, tenantID, userID, f.ClinicID)
	if err != nil {
		return nil, 0, err
	}
	return s.repo.ListTreatments(ctx, pool, f)
}

func (s *Service) CreateTreatment(ctx context.Context, tenantID, userID string, req model.CreateTreatmentRequest) (string, error) {
	if req.ClinicID == "" || strings.TrimSpace(req.TreatmentName) == "" {
		return "", ErrClinicTreatReq
	}
	if req.Consumables == nil {
		req.Consumables = []model.TreatmentConsumable{}
	}
	pool, err := s.poolAndMember(ctx, tenantID, userID, req.ClinicID)
	if err != nil {
		return "", err
	}
	return s.repo.CreateTreatment(ctx, pool, req)
}

func (s *Service) UpdateTreatment(ctx context.Context, tenantID, userID, id string, req model.CreateTreatmentRequest) error {
	if strings.TrimSpace(req.TreatmentName) == "" {
		return ErrTreatNameReq
	}
	if req.Consumables == nil {
		req.Consumables = []model.TreatmentConsumable{}
	}
	pool, err := s.pool(ctx, tenantID)
	if err != nil {
		return err
	}
	n, err := s.repo.UpdateTreatment(ctx, pool, id, userID, req)
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Service) DeleteTreatment(ctx context.Context, tenantID, userID, id string) error {
	pool, err := s.pool(ctx, tenantID)
	if err != nil {
		return err
	}
	n, err := s.repo.DeleteTreatment(ctx, pool, id, userID)
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// ── radiology ────────────────────────────────────────────────────────────────

func (s *Service) ListRadiologyTests(ctx context.Context, tenantID, userID string, f model.RadiologyListFilter) ([]model.RadiologyTestDTO, int, error) {
	if f.ClinicID == "" {
		return nil, 0, ErrClinicReq
	}
	pool, err := s.poolAndMember(ctx, tenantID, userID, f.ClinicID)
	if err != nil {
		return nil, 0, err
	}
	return s.repo.ListRadiologyTests(ctx, pool, f)
}

func (s *Service) CreateRadiologyTest(ctx context.Context, tenantID, userID string, req model.CreateRadiologyTestRequest) (string, error) {
	if req.ClinicID == "" || strings.TrimSpace(req.TestName) == "" {
		return "", ErrClinicTestReq
	}
	pool, err := s.poolAndMember(ctx, tenantID, userID, req.ClinicID)
	if err != nil {
		return "", err
	}
	return s.repo.CreateRadiologyTest(ctx, pool, req)
}

func (s *Service) UpdateRadiologyTest(ctx context.Context, tenantID, userID, id string, req model.CreateRadiologyTestRequest) error {
	if strings.TrimSpace(req.TestName) == "" {
		return ErrTestNameReq
	}
	pool, err := s.pool(ctx, tenantID)
	if err != nil {
		return err
	}
	n, err := s.repo.UpdateRadiologyTest(ctx, pool, id, userID, req)
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Service) DeleteRadiologyTest(ctx context.Context, tenantID, userID, id string) error {
	pool, err := s.pool(ctx, tenantID)
	if err != nil {
		return err
	}
	n, err := s.repo.DeleteRadiologyTest(ctx, pool, id, userID)
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// ── pathology tests ──────────────────────────────────────────────────────────

func (s *Service) ListPathologyTests(ctx context.Context, tenantID, userID string, f model.PathologyListFilter) ([]model.PathologyTestDTO, int, error) {
	if f.ClinicID == "" {
		return nil, 0, ErrClinicReq
	}
	pool, err := s.poolAndMember(ctx, tenantID, userID, f.ClinicID)
	if err != nil {
		return nil, 0, err
	}
	return s.repo.ListPathologyTests(ctx, pool, f)
}

func (s *Service) CreatePathologyTest(ctx context.Context, tenantID, userID string, req model.CreatePathologyTestRequest) (string, error) {
	if req.ClinicID == "" || strings.TrimSpace(req.TestName) == "" {
		return "", ErrClinicTestReq
	}
	pool, err := s.poolAndMember(ctx, tenantID, userID, req.ClinicID)
	if err != nil {
		return "", err
	}
	return s.repo.CreatePathologyTest(ctx, pool, req)
}

func (s *Service) UpdatePathologyTest(ctx context.Context, tenantID, userID, id string, req model.CreatePathologyTestRequest) error {
	if strings.TrimSpace(req.TestName) == "" {
		return ErrTestNameReq
	}
	pool, err := s.pool(ctx, tenantID)
	if err != nil {
		return err
	}
	n, err := s.repo.UpdatePathologyTest(ctx, pool, id, userID, req)
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Service) SyncTestParameters(ctx context.Context, tenantID, userID, testID string, paramIDs []string) (int, error) {
	pool, err := s.pool(ctx, tenantID)
	if err != nil {
		return 0, err
	}
	ok, _ := s.repo.TestBelongsToUserClinic(ctx, pool, testID, userID)
	if !ok {
		return 0, ErrForbidden
	}
	if err := s.repo.SyncTestParameters(ctx, pool, testID, paramIDs); err != nil {
		return 0, err
	}
	return len(paramIDs), nil
}

func (s *Service) DeletePathologyTest(ctx context.Context, tenantID, userID, id string) error {
	pool, err := s.pool(ctx, tenantID)
	if err != nil {
		return err
	}
	n, err := s.repo.DeletePathologyTest(ctx, pool, id, userID)
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// ── pathology parameters ─────────────────────────────────────────────────────

func (s *Service) ListPathologyParameters(ctx context.Context, tenantID, testID, clinicID string) ([]model.PathologyParameterDTO, error) {
	if testID == "" && clinicID == "" {
		return nil, errors.New("test or clinic is required")
	}
	pool, err := s.pool(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	return s.repo.ListPathologyParameters(ctx, pool, testID, clinicID)
}

func (s *Service) CreatePathologyParameter(ctx context.Context, tenantID string, req model.CreatePathologyParamRequest) (string, error) {
	if strings.TrimSpace(req.ParameterName) == "" {
		return "", ErrParamNameReq
	}
	if req.TestID == "" && req.ClinicID == "" {
		return "", ErrParamIDsReq
	}
	if req.PatientGroups == nil {
		req.PatientGroups = []model.PatientGroupRange{}
	}
	pool, err := s.pool(ctx, tenantID)
	if err != nil {
		return "", err
	}
	clinicID := req.ClinicID
	if req.TestID != "" {
		derived, err := s.repo.DeriveClinicFromTest(ctx, pool, req.TestID)
		if err != nil {
			return "", ErrTestNotFound
		}
		clinicID = derived
	}
	paramID, err := s.repo.UpsertPathologyParameter(ctx, pool, clinicID, req)
	if err != nil {
		return "", err
	}
	if req.TestID != "" {
		if err := s.repo.LinkTestParameter(ctx, pool, req.TestID, paramID); err != nil {
			return "", err
		}
	}
	return paramID, nil
}

func (s *Service) UpdatePathologyParameter(ctx context.Context, tenantID, userID, id string, req model.CreatePathologyParamRequest) error {
	if strings.TrimSpace(req.ParameterName) == "" {
		return ErrParamNameReq
	}
	if req.PatientGroups == nil {
		req.PatientGroups = []model.PatientGroupRange{}
	}
	pool, err := s.pool(ctx, tenantID)
	if err != nil {
		return err
	}
	n, err := s.repo.UpdatePathologyParameter(ctx, pool, id, userID, req)
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Service) DeletePathologyParameter(ctx context.Context, tenantID, userID, id string) error {
	pool, err := s.pool(ctx, tenantID)
	if err != nil {
		return err
	}
	n, err := s.repo.DeletePathologyParameter(ctx, pool, id, userID)
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// ── pathology groups ─────────────────────────────────────────────────────────

func (s *Service) ListPathologyGroups(ctx context.Context, tenantID, clinicID string) ([]model.ParamGroupDTO, error) {
	if clinicID == "" {
		return nil, ErrClinicReq
	}
	pool, err := s.pool(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	return s.repo.ListPathologyGroups(ctx, pool, clinicID)
}

func (s *Service) CreatePathologyGroup(ctx context.Context, tenantID string, req model.SaveGroupRequest) (string, error) {
	if req.ClinicID == "" || strings.TrimSpace(req.GroupName) == "" {
		return "", ErrGroupReq
	}
	pool, err := s.pool(ctx, tenantID)
	if err != nil {
		return "", err
	}
	return s.repo.UpsertPathologyGroup(ctx, pool, req.ClinicID, req.GroupName, req.ParamIDs)
}

func (s *Service) UpdatePathologyGroup(ctx context.Context, tenantID, userID, id string, req model.SaveGroupRequest) error {
	if strings.TrimSpace(req.GroupName) == "" {
		return ErrGroupNameReq
	}
	pool, err := s.pool(ctx, tenantID)
	if err != nil {
		return err
	}
	n, err := s.repo.UpdatePathologyGroup(ctx, pool, id, userID, req.GroupName, req.ParamIDs)
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Service) DeletePathologyGroup(ctx context.Context, tenantID, userID, id string) error {
	pool, err := s.pool(ctx, tenantID)
	if err != nil {
		return err
	}
	n, err := s.repo.DeletePathologyGroup(ctx, pool, id, userID)
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Service) RemoveGroupParam(ctx context.Context, tenantID, userID, groupID, paramID string) error {
	pool, err := s.pool(ctx, tenantID)
	if err != nil {
		return err
	}
	return s.repo.RemoveGroupParam(ctx, pool, groupID, paramID, userID)
}

// ── copy ─────────────────────────────────────────────────────────────────────

func (s *Service) CopyCatalogue(ctx context.Context, tenantID, userID string, req model.CopyCatalogueRequest) ([]model.CopyCatalogueResult, error) {
	if req.SourceClinicID == "" || len(req.TargetClinicIDs) == 0 {
		return nil, ErrCopyReq
	}
	pool, err := s.poolAndMember(ctx, tenantID, userID, req.SourceClinicID)
	if err != nil {
		return nil, err
	}
	results := make([]model.CopyCatalogueResult, 0, len(req.TargetClinicIDs))
	for _, target := range req.TargetClinicIDs {
		if target == req.SourceClinicID {
			continue
		}
		if err := s.repo.AssertMember(ctx, pool, userID, target); err != nil {
			return nil, errors.New("not a member of target clinic " + target)
		}
		res, err := s.repo.CopyCatalogue(ctx, pool, req.SourceClinicID, target)
		if err != nil {
			return nil, err
		}
		results = append(results, res)
	}
	return results, nil
}
