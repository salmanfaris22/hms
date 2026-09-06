package repository

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/salman/hms-backend/internal/infrastructure/persistence/postgres"
	"github.com/salman/hms-backend/internal/modules/dashboard/model"
)

type Repository struct {
	tenants *postgres.TenantResolver
}

func New(tenants *postgres.TenantResolver) *Repository { return &Repository{tenants: tenants} }

func (r *Repository) Tenants() *postgres.TenantResolver { return r.tenants }
func (r *Repository) Pool(ctx context.Context, tenantID string) (*pgxpool.Pool, error) {
	return r.tenants.Pool(ctx, tenantID)
}
func (r *Repository) IsMember(ctx context.Context, pool *pgxpool.Pool, clinicID, userID string) bool {
	var ok bool
	_ = pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM clinic_members WHERE clinic_id = $1::uuid AND user_id = $2::uuid)`,
		clinicID, userID,
	).Scan(&ok)
	return ok
}

func (r *Repository) CountScalar(ctx context.Context, pool *pgxpool.Pool, q string, args ...any) int {
	var n int
	_ = pool.QueryRow(ctx, q, args...).Scan(&n)
	return n
}

func (r *Repository) SumScalar(ctx context.Context, pool *pgxpool.Pool, q string, args ...any) float64 {
	var v float64
	_ = pool.QueryRow(ctx, q, args...).Scan(&v)
	return v
}

// Daily time-series for a given sum query over N days back.
// The query must return (day date, value numeric) for the clinic.
func (r *Repository) DailyTrend(ctx context.Context, pool *pgxpool.Pool, q string, clinicID string, days int) []model.TrendPoint {
	rows, err := pool.Query(ctx, q, clinicID, days)
	if err != nil {
		return []model.TrendPoint{}
	}
	defer rows.Close()

	labelFor := func(d time.Time) string { return d.Format("Mon") }

	// build full [today-days+1 .. today] series zero-filled
	out := make([]model.TrendPoint, 0, days)
	today := time.Now().Truncate(24 * time.Hour)
	byDay := map[string]float64{}
	for rows.Next() {
		var day time.Time
		var val float64
		if err := rows.Scan(&day, &val); err == nil {
			byDay[day.Format("2006-01-02")] = val
		}
	}
	for i := days - 1; i >= 0; i-- {
		d := today.AddDate(0, 0, -i)
		out = append(out, model.TrendPoint{
			Label: labelFor(d),
			Value: byDay[d.Format("2006-01-02")],
		})
	}
	return out
}

func (r *Repository) TopDepartments(ctx context.Context, pool *pgxpool.Pool, clinicID string, limit int) []model.DeptRow {
	rows, err := pool.Query(ctx, `
		SELECT d.name, d.staff_count,
		  (SELECT COUNT(*) FROM staff_profiles sp WHERE sp.department_id = d.id) AS actual_staff
		FROM departments d
		WHERE d.clinic_id = $1::uuid
		ORDER BY actual_staff DESC, d.staff_count DESC, d.name ASC
		LIMIT $2`,
		clinicID, limit,
	)
	if err != nil {
		return []model.DeptRow{}
	}
	defer rows.Close()
	out := []model.DeptRow{}
	for rows.Next() {
		var r model.DeptRow
		var actual int
		if err := rows.Scan(&r.Name, &r.StaffCount, &actual); err == nil {
			r.PatientCount = 0
			if actual > r.StaffCount {
				r.StaffCount = actual
			}
			out = append(out, r)
		}
	}
	return out
}

func (r *Repository) IncomeByCategory(ctx context.Context, pool *pgxpool.Pool, clinicID string) []model.CategorySlice {
	// Approximation: break revenue into Pharmacy (from sales) vs OPD / other.
	// With no billing module yet, we use what's recorded.
	var pharmacy float64
	_ = pool.QueryRow(ctx,
		`SELECT COALESCE(SUM(grand_total), 0) FROM inv_sales WHERE clinic_id = $1::uuid`,
		clinicID,
	).Scan(&pharmacy)

	out := []model.CategorySlice{
		{Label: "Pharmacy", Value: pharmacy, Color: "#10b981"},
	}
	return out
}

func (r *Repository) RecentActivity(ctx context.Context, pool *pgxpool.Pool, clinicID string, limit int) []model.ActivityRow {
	// Surface: newest patients, staff, sales. Keep it simple for now.
	out := []model.ActivityRow{}

	if rows, err := pool.Query(ctx, `
		SELECT 'patient', 'New patient registered', COALESCE(first_name||' '||last_name, full_name), created_at::text
		FROM patients WHERE clinic_id = $1::uuid ORDER BY created_at DESC LIMIT $2`,
		clinicID, limit,
	); err == nil {
		defer rows.Close()
		for rows.Next() {
			var a model.ActivityRow
			if err := rows.Scan(&a.Kind, &a.Title, &a.Subject, &a.CreatedAt); err == nil {
				out = append(out, a)
			}
		}
	}

	if rows, err := pool.Query(ctx, `
		SELECT 'sale', 'Sale recorded', invoice_no || ' · ' || COALESCE(customer_name, ''), created_at::text
		FROM inv_sales WHERE clinic_id = $1::uuid ORDER BY created_at DESC LIMIT $2`,
		clinicID, limit,
	); err == nil {
		defer rows.Close()
		for rows.Next() {
			var a model.ActivityRow
			if err := rows.Scan(&a.Kind, &a.Title, &a.Subject, &a.CreatedAt); err == nil {
				out = append(out, a)
			}
		}
	}

	if rows, err := pool.Query(ctx, `
		SELECT 'staff', 'Staff added', u.full_name, cm.joined_at::text
		FROM clinic_members cm JOIN users u ON u.id = cm.user_id
		WHERE cm.clinic_id = $1::uuid ORDER BY cm.joined_at DESC LIMIT $2`,
		clinicID, limit,
	); err == nil {
		defer rows.Close()
		for rows.Next() {
			var a model.ActivityRow
			if err := rows.Scan(&a.Kind, &a.Title, &a.Subject, &a.CreatedAt); err == nil {
				out = append(out, a)
			}
		}
	}

	return out
}

// CountClinics returns the total clinics in the tenant DB.
func (r *Repository) CountClinics(ctx context.Context, pool *pgxpool.Pool) int {
	var n int
	_ = pool.QueryRow(ctx, `SELECT COUNT(*) FROM clinics`).Scan(&n)
	return n
}

// DatabaseSizeGB returns the on-disk size of the tenant DB in gigabytes.
func (r *Repository) DatabaseSizeGB(ctx context.Context, pool *pgxpool.Pool) float64 {
	var bytes int64
	_ = pool.QueryRow(ctx,
		`SELECT pg_database_size(current_database())`,
	).Scan(&bytes)
	return float64(bytes) / (1024.0 * 1024.0 * 1024.0)
}
