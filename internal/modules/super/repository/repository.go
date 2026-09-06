package repository

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/salman/hms-backend/internal/core/utils"
	"github.com/salman/hms-backend/internal/infrastructure/persistence/postgres"
	"github.com/salman/hms-backend/internal/modules/super/model"
)

type Repository struct {
	registry *pgxpool.Pool
	tenants  *postgres.TenantResolver
}

func New(registry *pgxpool.Pool, tenants *postgres.TenantResolver) *Repository {
	return &Repository{registry: registry, tenants: tenants}
}

func (r *Repository) Registry() *pgxpool.Pool              { return r.registry }
func (r *Repository) Tenants() *postgres.TenantResolver    { return r.tenants }

func (r *Repository) FindSuperAdmin(ctx context.Context, email string) (model.SuperAdminRow, error) {
	var row model.SuperAdminRow
	err := r.registry.QueryRow(ctx, `
		SELECT id::text, email, password_hash, full_name
		FROM super_admins
		WHERE email = $1`, email,
	).Scan(&row.ID, &row.Email, &row.PasswordHash, &row.FullName)
	return row, err
}

func (r *Repository) ListTenants(ctx context.Context) ([]model.TenantDTO, error) {
	rows, err := r.registry.Query(ctx, `
		SELECT id::text, slug, name, db_name, phone, phone_country, phones,
		       max_clinics, storage_quota_gb, modules,
		       subscription_start, subscription_end, is_active,
		       created_at, welcome_sent_at
		FROM tenants
		ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []model.TenantDTO{}
	for rows.Next() {
		var t model.TenantDTO
		var phonesRaw []byte
		if err := rows.Scan(
			&t.ID, &t.Slug, &t.Name, &t.DBName, &t.Phone, &t.PhoneCountry, &phonesRaw,
			&t.MaxClinics, &t.StorageQuotaGB, &t.Modules,
			&t.SubscriptionStart, &t.SubscriptionEnd, &t.IsActive,
			&t.CreatedAt, &t.WelcomeSentAt,
		); err != nil {
			return nil, err
		}
		if len(phonesRaw) > 0 {
			_ = json.Unmarshal(phonesRaw, &t.Phones)
		}
		if t.Phones == nil {
			t.Phones = []model.TenantPhone{}
		}
		out = append(out, t)
	}
	return out, nil
}

func (r *Repository) TenantStats(ctx context.Context, tenantID, dbName string) (model.TenantStats, error) {
	var s model.TenantStats
	pool, err := r.tenants.Pool(ctx, tenantID)
	if err != nil {
		return s, err
	}
	_ = pool.QueryRow(ctx, `SELECT count(*) FROM users`).Scan(&s.UsersCount)
	_ = pool.QueryRow(ctx, `SELECT count(*) FROM clinics WHERE is_archived = false`).Scan(&s.ClinicsCount)
	_ = r.registry.QueryRow(ctx, `SELECT pg_database_size($1)`, dbName).Scan(&s.StorageBytes)
	return s, nil
}

func (r *Repository) UpdateTenantSettings(
	ctx context.Context, tenantID, phone, phoneCountry string, phones []model.TenantPhone,
	maxClinics, storageQuotaGB int, modules []string, subStart, subEnd string,
) error {
	phonesJSON, _ := json.Marshal(phones)
	_, err := r.registry.Exec(ctx, `
		UPDATE tenants
		SET phone = $1, phone_country = $2, phones = $3::jsonb,
		    max_clinics = $4, storage_quota_gb = $5, modules = $6,
		    subscription_start = NULLIF($7,'')::date,
		    subscription_end   = NULLIF($8,'')::date,
		    updated_at = now()
		WHERE id = $9::uuid`,
		phone, phoneCountry, string(phonesJSON),
		maxClinics, storageQuotaGB, modules,
		subStart, subEnd, tenantID,
	)
	return err
}

func (r *Repository) UpsertAdminUser(
	ctx context.Context, pool *pgxpool.Pool,
	email, passwordHash, fullName, phone, phoneCountry string,
) (string, error) {
	var id string
	err := pool.QueryRow(ctx, `
		INSERT INTO users (email, password_hash, full_name, role, phone, phone_country, is_super_admin)
		VALUES ($1, $2, $3, 'admin', $4, $5, true)
		ON CONFLICT (email) DO UPDATE
		SET password_hash = EXCLUDED.password_hash,
		    full_name = EXCLUDED.full_name,
		    phone = EXCLUDED.phone,
		    phone_country = EXCLUDED.phone_country,
		    is_super_admin = true,
		    updated_at = now()
		RETURNING id::text`,
		email, passwordHash, fullName, phone, phoneCountry,
	).Scan(&id)
	return id, err
}

func (r *Repository) UpsertDirectory(ctx context.Context, email, tenantID, userID string) error {
	_, err := r.registry.Exec(ctx, `
		INSERT INTO user_directory (email, tenant_id, user_id)
		VALUES ($1, $2::uuid, $3::uuid)
		ON CONFLICT (email) DO UPDATE
		SET tenant_id = EXCLUDED.tenant_id, user_id = EXCLUDED.user_id`,
		email, tenantID, userID)
	return err
}

func (r *Repository) MarkWelcomeSent(ctx context.Context, tenantID string) {
	_, _ = r.registry.Exec(ctx,
		`UPDATE tenants SET welcome_sent_at = now() WHERE id = $1::uuid`, tenantID)
}

func (r *Repository) UpdateTenant(ctx context.Context, id string, req model.UpdateTenantRequest) (bool, error) {
	sets := []string{}
	args := []any{}
	arg := func(v any) string {
		args = append(args, v)
		return "$" + utils.Itoa(len(args))
	}
	if req.Name != nil {
		sets = append(sets, "name = "+arg(*req.Name))
	}
	if req.Phone != nil {
		sets = append(sets, "phone = "+arg(*req.Phone))
	}
	if req.PhoneCountry != nil {
		sets = append(sets, "phone_country = "+arg(*req.PhoneCountry))
	}
	if req.Phones != nil {
		b, _ := json.Marshal(*req.Phones)
		sets = append(sets, "phones = "+arg(string(b))+"::jsonb")
	}
	if req.MaxClinics != nil {
		sets = append(sets, "max_clinics = "+arg(*req.MaxClinics))
	}
	if req.StorageQuotaGB != nil {
		sets = append(sets, "storage_quota_gb = "+arg(*req.StorageQuotaGB))
	}
	if req.Modules != nil {
		sets = append(sets, "modules = "+arg(*req.Modules))
	}
	if req.SubscriptionStart != nil {
		sets = append(sets, "subscription_start = NULLIF("+arg(*req.SubscriptionStart)+",'')::date")
	}
	if req.SubscriptionEnd != nil {
		sets = append(sets, "subscription_end = NULLIF("+arg(*req.SubscriptionEnd)+",'')::date")
	}
	if req.IsActive != nil {
		sets = append(sets, "is_active = "+arg(*req.IsActive))
	}
	if len(sets) == 0 {
		return false, nil
	}
	sets = append(sets, "updated_at = now()")

	query := "UPDATE tenants SET " + strings.Join(sets, ", ") + " WHERE id = " + arg(id) + "::uuid"
	_, err := r.registry.Exec(ctx, query, args...)
	return true, err
}

func (r *Repository) DeleteTenant(ctx context.Context, id string) error {
	return r.tenants.DeleteTenant(ctx, id)
}

type ReminderInfo struct {
	Name       string
	AdminEmail string
	SubEnd     *time.Time
}

func (r *Repository) TenantForReminder(ctx context.Context, id string) (ReminderInfo, error) {
	var info ReminderInfo
	err := r.registry.QueryRow(ctx, `
		SELECT t.name, COALESCE(ud.email, ''), t.subscription_end
		FROM tenants t
		LEFT JOIN user_directory ud ON ud.tenant_id = t.id
		WHERE t.id = $1::uuid
		LIMIT 1`, id,
	).Scan(&info.Name, &info.AdminEmail, &info.SubEnd)
	return info, err
}
