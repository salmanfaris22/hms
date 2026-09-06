package service

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/salman/hms-backend/internal/modules/inventory/model"
	"github.com/salman/hms-backend/internal/modules/inventory/repository"
)

var (
	ErrTenantUnavail = errors.New("tenant unavailable")
	ErrClinicReq     = errors.New("clinicId required")
	ErrNameReq       = errors.New("name required")
	ErrBadBody       = errors.New("invalid body")
)

type Service struct {
	repo *repository.Repository
}

func New(repo *repository.Repository) *Service { return &Service{repo: repo} }

func (s *Service) pool(ctx context.Context, tenantID string) (*pgxpool.Pool, error) {
	p, err := s.repo.Pool(ctx, tenantID)
	if err != nil {
		return nil, ErrTenantUnavail
	}
	return p, nil
}

// ── drugs ────────────────────────────────────────────────────────────────────

func (s *Service) ListDrugs(ctx context.Context, tenantID string, f model.DrugListFilter) ([]model.DrugDTO, int, error) {
	if f.ClinicID == "" {
		return nil, 0, ErrClinicReq
	}
	if f.Page < 1 {
		f.Page = 1
	}
	pool, err := s.pool(ctx, tenantID)
	if err != nil {
		return nil, 0, err
	}
	return s.repo.ListDrugs(ctx, pool, f)
}

func (s *Service) DrugSummary(ctx context.Context, tenantID, clinicID string) (model.StockSummary, error) {
	if clinicID == "" {
		return model.StockSummary{}, ErrClinicReq
	}
	pool, err := s.pool(ctx, tenantID)
	if err != nil {
		return model.StockSummary{}, err
	}
	return s.repo.DrugSummary(ctx, pool, clinicID)
}

func (s *Service) CreateDrug(ctx context.Context, tenantID, clinicID string, req model.CreateDrugRequest) (string, error) {
	if clinicID == "" {
		return "", ErrClinicReq
	}
	if strings.TrimSpace(req.Name) == "" {
		return "", ErrNameReq
	}
	pool, err := s.pool(ctx, tenantID)
	if err != nil {
		return "", err
	}
	return s.repo.CreateDrug(ctx, pool, clinicID, req)
}

func (s *Service) DeleteDrug(ctx context.Context, tenantID, clinicID, id string) error {
	pool, err := s.pool(ctx, tenantID)
	if err != nil {
		return err
	}
	return s.repo.DeleteDrug(ctx, pool, clinicID, id)
}

// ── categories ───────────────────────────────────────────────────────────────

func (s *Service) ListCategories(ctx context.Context, tenantID, clinicID string) ([]model.CategoryDTO, error) {
	if clinicID == "" {
		return nil, ErrClinicReq
	}
	pool, err := s.pool(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	return s.repo.ListCategories(ctx, pool, clinicID)
}

func (s *Service) CreateCategory(ctx context.Context, tenantID, clinicID, name string) (string, error) {
	if strings.TrimSpace(name) == "" {
		return "", ErrNameReq
	}
	pool, err := s.pool(ctx, tenantID)
	if err != nil {
		return "", err
	}
	return s.repo.CreateCategory(ctx, pool, clinicID, name)
}

func (s *Service) DeleteCategory(ctx context.Context, tenantID, clinicID, id string) error {
	pool, err := s.pool(ctx, tenantID)
	if err != nil {
		return err
	}
	return s.repo.DeleteCategory(ctx, pool, clinicID, id)
}

// ── manufacturers ────────────────────────────────────────────────────────────

func (s *Service) ListManufacturers(ctx context.Context, tenantID, clinicID string) ([]model.ManufacturerDTO, error) {
	if clinicID == "" {
		return nil, ErrClinicReq
	}
	pool, err := s.pool(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	return s.repo.ListManufacturers(ctx, pool, clinicID)
}

func (s *Service) CreateManufacturer(ctx context.Context, tenantID, clinicID, name string) (string, error) {
	if strings.TrimSpace(name) == "" {
		return "", ErrNameReq
	}
	pool, err := s.pool(ctx, tenantID)
	if err != nil {
		return "", err
	}
	return s.repo.CreateManufacturer(ctx, pool, clinicID, name)
}

func (s *Service) DeleteManufacturer(ctx context.Context, tenantID, clinicID, id string) error {
	pool, err := s.pool(ctx, tenantID)
	if err != nil {
		return err
	}
	return s.repo.DeleteManufacturer(ctx, pool, clinicID, id)
}

// ── stock ────────────────────────────────────────────────────────────────────

func (s *Service) ListStock(ctx context.Context, tenantID, clinicID, drugID string) ([]model.StockDTO, error) {
	if clinicID == "" {
		return nil, ErrClinicReq
	}
	pool, err := s.pool(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	return s.repo.ListStock(ctx, pool, clinicID, drugID)
}

func (s *Service) AddStock(ctx context.Context, tenantID, clinicID string, req model.CreateStockRequest) (string, error) {
	pool, err := s.pool(ctx, tenantID)
	if err != nil {
		return "", err
	}
	return s.repo.AddStock(ctx, pool, clinicID, req)
}

// ── POS search ───────────────────────────────────────────────────────────────

func (s *Service) SearchPOS(ctx context.Context, tenantID, clinicID, q, catID string) ([]model.POSItem, error) {
	pool, err := s.pool(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	return s.repo.SearchPOS(ctx, pool, clinicID, q, catID)
}

// ── sales ────────────────────────────────────────────────────────────────────

func (s *Service) SalesSummary(ctx context.Context, tenantID, clinicID string) (model.SaleSummary, error) {
	if clinicID == "" {
		return model.SaleSummary{}, ErrClinicReq
	}
	pool, err := s.pool(ctx, tenantID)
	if err != nil {
		return model.SaleSummary{}, err
	}
	return s.repo.SalesSummary(ctx, pool, clinicID)
}

func (s *Service) ListSales(ctx context.Context, tenantID string, f model.SaleListFilter) ([]model.SaleDTO, int, error) {
	if f.ClinicID == "" {
		return nil, 0, ErrClinicReq
	}
	if f.Page < 1 {
		f.Page = 1
	}
	pool, err := s.pool(ctx, tenantID)
	if err != nil {
		return nil, 0, err
	}
	return s.repo.ListSales(ctx, pool, f)
}

func (s *Service) CreateSale(ctx context.Context, tenantID, clinicID string, req model.CreateSaleRequest) (string, string, error) {
	if clinicID == "" {
		return "", "", ErrClinicReq
	}
	pool, err := s.pool(ctx, tenantID)
	if err != nil {
		return "", "", err
	}
	return s.repo.CreateSale(ctx, pool, clinicID, req)
}

func (s *Service) DeleteSale(ctx context.Context, tenantID, clinicID, id string) error {
	pool, err := s.pool(ctx, tenantID)
	if err != nil {
		return err
	}
	return s.repo.DeleteSale(ctx, pool, clinicID, id)
}

// ── purchases ────────────────────────────────────────────────────────────────

func (s *Service) ListPurchases(ctx context.Context, tenantID string, f model.PurchaseListFilter) ([]model.PurchaseDTO, int, error) {
	if f.ClinicID == "" {
		return nil, 0, ErrClinicReq
	}
	if f.Page < 1 {
		f.Page = 1
	}
	pool, err := s.pool(ctx, tenantID)
	if err != nil {
		return nil, 0, err
	}
	return s.repo.ListPurchases(ctx, pool, f)
}

func (s *Service) CreatePurchase(ctx context.Context, tenantID, clinicID string, req model.CreatePurchaseRequest) (string, string, error) {
	if clinicID == "" {
		return "", "", ErrClinicReq
	}
	pool, err := s.pool(ctx, tenantID)
	if err != nil {
		return "", "", err
	}
	return s.repo.CreatePurchase(ctx, pool, clinicID, req)
}

func (s *Service) DeletePurchase(ctx context.Context, tenantID, clinicID, id string) error {
	pool, err := s.pool(ctx, tenantID)
	if err != nil {
		return err
	}
	return s.repo.DeletePurchase(ctx, pool, clinicID, id)
}

// ── dispense ─────────────────────────────────────────────────────────────────

func (s *Service) DispenseSummary(ctx context.Context, tenantID, clinicID string) (model.DispenseSummary, error) {
	if clinicID == "" {
		return model.DispenseSummary{}, ErrClinicReq
	}
	pool, err := s.pool(ctx, tenantID)
	if err != nil {
		return model.DispenseSummary{}, err
	}
	return s.repo.DispenseSummary(ctx, pool, clinicID)
}

func (s *Service) ListDispenses(ctx context.Context, tenantID string, f model.DispenseListFilter) ([]model.DispenseDTO, int, error) {
	if f.ClinicID == "" {
		return nil, 0, ErrClinicReq
	}
	if f.Page < 1 {
		f.Page = 1
	}
	pool, err := s.pool(ctx, tenantID)
	if err != nil {
		return nil, 0, err
	}
	return s.repo.ListDispenses(ctx, pool, f)
}

func (s *Service) CreateDispense(ctx context.Context, tenantID, clinicID string, req model.CreateDispenseRequest) (string, string, error) {
	if clinicID == "" {
		return "", "", ErrClinicReq
	}
	pool, err := s.pool(ctx, tenantID)
	if err != nil {
		return "", "", err
	}
	return s.repo.CreateDispense(ctx, pool, clinicID, req)
}

func (s *Service) UpdateDispenseStatus(ctx context.Context, tenantID, clinicID, id, status string) error {
	pool, err := s.pool(ctx, tenantID)
	if err != nil {
		return err
	}
	return s.repo.UpdateDispenseStatus(ctx, pool, clinicID, id, status)
}

// ── assets ───────────────────────────────────────────────────────────────────

func (s *Service) AssetSummary(ctx context.Context, tenantID, clinicID string) (model.AssetSummary, error) {
	if clinicID == "" {
		return model.AssetSummary{}, ErrClinicReq
	}
	pool, err := s.pool(ctx, tenantID)
	if err != nil {
		return model.AssetSummary{}, err
	}
	return s.repo.AssetSummary(ctx, pool, clinicID)
}

func (s *Service) ListAssets(ctx context.Context, tenantID string, f model.AssetListFilter) ([]model.AssetDTO, int, error) {
	if f.ClinicID == "" {
		return nil, 0, ErrClinicReq
	}
	if f.Page < 1 {
		f.Page = 1
	}
	pool, err := s.pool(ctx, tenantID)
	if err != nil {
		return nil, 0, err
	}
	return s.repo.ListAssets(ctx, pool, f)
}

func (s *Service) CreateAsset(ctx context.Context, tenantID, clinicID string, req model.CreateAssetRequest) (string, string, error) {
	if clinicID == "" {
		return "", "", ErrClinicReq
	}
	if strings.TrimSpace(req.Name) == "" {
		return "", "", ErrNameReq
	}
	pool, err := s.pool(ctx, tenantID)
	if err != nil {
		return "", "", err
	}
	return s.repo.CreateAsset(ctx, pool, clinicID, req)
}

func (s *Service) UpdateAsset(ctx context.Context, tenantID, clinicID, id string, req model.CreateAssetRequest) error {
	pool, err := s.pool(ctx, tenantID)
	if err != nil {
		return err
	}
	return s.repo.UpdateAsset(ctx, pool, clinicID, id, req)
}

func (s *Service) DeleteAsset(ctx context.Context, tenantID, clinicID, id string) error {
	pool, err := s.pool(ctx, tenantID)
	if err != nil {
		return err
	}
	return s.repo.DeleteAsset(ctx, pool, clinicID, id)
}
