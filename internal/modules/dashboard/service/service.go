package service

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/salman/hms-backend/internal/modules/dashboard/model"
	"github.com/salman/hms-backend/internal/modules/dashboard/repository"
)

var (
	ErrTenantUnavail = errors.New("tenant unavailable")
	ErrForbidden     = errors.New("not a member of this clinic")
)

type Meta struct {
	TenantID, UserID string
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

func (s *Service) Overview(ctx context.Context, m Meta, clinicID, period string) (*model.Overview, error) {
	pool, err := s.poolAndMember(ctx, m.TenantID, clinicID, m.UserID)
	if err != nil {
		return nil, err
	}

	// window boundary in days
	days := 7
	switch period {
	case "today":
		days = 1
	case "week":
		days = 7
	case "month":
		days = 30
	case "year":
		days = 12 // 12 months → still returned as daily points for simplicity
	}

	ov := &model.Overview{Period: period}

	ov.TotalPatients = s.repo.CountScalar(ctx, pool,
		`SELECT COUNT(*) FROM patients WHERE clinic_id = $1::uuid`, clinicID)
	ov.TotalStaff = s.repo.CountScalar(ctx, pool,
		`SELECT COUNT(*) FROM clinic_members cm JOIN users u ON u.id = cm.user_id
		 WHERE cm.clinic_id = $1::uuid AND u.is_active = true`, clinicID)
	ov.ActiveToday = s.repo.CountScalar(ctx, pool,
		`SELECT COUNT(*) FROM clinic_members cm JOIN users u ON u.id = cm.user_id
		 WHERE cm.clinic_id = $1::uuid AND u.last_login_at::date = CURRENT_DATE`, clinicID)
	ov.TotalClinics = s.repo.CountClinics(ctx, pool)

	ov.TotalRevenue = s.repo.SumScalar(ctx, pool,
		`SELECT COALESCE(SUM(grand_total), 0) FROM inv_sales WHERE clinic_id = $1::uuid`, clinicID)
	ov.TotalExpenses = s.repo.SumScalar(ctx, pool,
		`SELECT COALESCE(SUM(grand_total), 0) FROM inv_purchases WHERE clinic_id = $1::uuid`, clinicID)
	ov.NetProfit = ov.TotalRevenue - ov.TotalExpenses

	ov.OutstandingAR = s.repo.SumScalar(ctx, pool,
		`SELECT COALESCE(SUM(grand_total), 0) FROM inv_sales
		 WHERE clinic_id = $1::uuid AND COALESCE(status, '') = 'Pending'`, clinicID)
	ov.CashInHand = s.repo.SumScalar(ctx, pool,
		`SELECT COALESCE(SUM(grand_total), 0) FROM inv_sales
		 WHERE clinic_id = $1::uuid AND COALESCE(payment_method, 'Cash') = 'Cash'`, clinicID)

	ov.LowStockCount = s.repo.CountScalar(ctx, pool, `
		SELECT COUNT(*) FROM inv_drugs d
		WHERE d.clinic_id = $1::uuid
		  AND COALESCE((SELECT SUM(qty_available) FROM inv_stock WHERE drug_id = d.id), 0) <= COALESCE(d.reorder_level, 10)
		  AND COALESCE((SELECT SUM(qty_available) FROM inv_stock WHERE drug_id = d.id), 0) > 0`,
		clinicID,
	)
	ov.ExpiringSoonCount = s.repo.CountScalar(ctx, pool, `
		SELECT COUNT(DISTINCT drug_id) FROM inv_stock
		WHERE expiry_date IS NOT NULL AND expiry_date <= CURRENT_DATE + INTERVAL '30 days'
		  AND drug_id IN (SELECT id FROM inv_drugs WHERE clinic_id = $1::uuid)`,
		clinicID,
	)

	// Storage
	ov.StorageUsedGB = s.repo.DatabaseSizeGB(ctx, pool)
	ov.StorageQuotaGB = 50.0 // TODO: pull from tenant limit

	// Appointments — table for actual bookings not yet in schema; use 0s for now.
	ov.AppointmentsToday = 0
	ov.AppointmentsWeek = 0

	// Rating placeholder (reviews module pending)
	ov.Rating = 0
	ov.ReviewsCount = 0

	// Time series: revenue & expenses last N days (cap at 30 for the chart)
	span := days
	if span > 30 {
		span = 30
	}
	if span < 7 {
		span = 7
	}
	ov.RevenueTrend = s.repo.DailyTrend(ctx, pool, `
		SELECT d::date, COALESCE(SUM(s.grand_total), 0)
		FROM generate_series(CURRENT_DATE - ($2::int - 1), CURRENT_DATE, '1 day') d
		LEFT JOIN inv_sales s ON s.created_at::date = d::date AND s.clinic_id = $1::uuid
		GROUP BY d ORDER BY d ASC`, clinicID, span)

	ov.ExpenseTrend = s.repo.DailyTrend(ctx, pool, `
		SELECT d::date, COALESCE(SUM(p.grand_total), 0)
		FROM generate_series(CURRENT_DATE - ($2::int - 1), CURRENT_DATE, '1 day') d
		LEFT JOIN inv_purchases p ON p.created_at::date = d::date AND p.clinic_id = $1::uuid
		GROUP BY d ORDER BY d ASC`, clinicID, span)

	ov.IncomeByCategory = s.repo.IncomeByCategory(ctx, pool, clinicID)
	ov.TopDepartments = s.repo.TopDepartments(ctx, pool, clinicID, 6)
	ov.RecentActivity = s.repo.RecentActivity(ctx, pool, clinicID, 3)

	return ov, nil
}
