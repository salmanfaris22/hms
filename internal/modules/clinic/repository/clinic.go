package repository

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/salman/hms-backend/internal/core/utils"
	"github.com/salman/hms-backend/internal/modules/clinic/model"
)

type rowScanner interface {
	Scan(dest ...any) error
}

func scanClinic(row rowScanner) (model.ClinicDTO, error) {
	var c model.ClinicDTO
	var phonesRaw, timingsRaw []byte
	err := row.Scan(
		&c.ID, &c.Name, &c.ClinicType, &c.Location, &c.Color,
		&c.InviteCode, &c.Role, &c.IsDefault, &c.LastAccessedAt, &c.CreatedAt,
		&c.LogoURL, &c.Address, &c.Locality, &c.PinCode, &c.State, &c.Country,
		&phonesRaw, &c.Email, &c.Website, &c.GSTIN, &c.FacilityID,
		&c.TimeFormat, &c.SystemLanguage, &c.TimeZone, &c.DateFormat, &c.Currency,
		&timingsRaw, &c.IsPrimary, &c.IsArchived,
	)
	if err != nil {
		return c, err
	}
	if len(phonesRaw) > 0 {
		_ = json.Unmarshal(phonesRaw, &c.Phones)
	}
	if c.Phones == nil {
		c.Phones = []model.Phone{}
	}
	if len(timingsRaw) > 0 {
		_ = json.Unmarshal(timingsRaw, &c.Timings)
	}
	if c.Timings == nil {
		c.Timings = model.Timings{}
	}
	return c, nil
}

func RandomInviteCode(n int) (string, error) {
	const alphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	out := make([]byte, n)
	for i := range buf {
		out[i] = alphabet[int(buf[i])%len(alphabet)]
	}
	return string(out), nil
}

func (r *Repository) ListClinics(ctx context.Context, pool *pgxpool.Pool, userID string, archived bool) ([]model.ClinicDTO, error) {
	rows, err := pool.Query(ctx, `
		SELECT c.id::text, c.name, c.clinic_type, c.location, c.color,
		       c.invite_code, m.role, m.is_default, m.last_accessed_at, c.created_at,
		       c.logo_url, c.address, c.locality, c.pin_code, c.state, c.country,
		       c.phones, c.email, c.website, c.gstin, c.facility_id,
		       c.time_format, c.system_language, c.time_zone, c.date_format, c.currency,
		       c.timings, c.is_primary, c.is_archived
		FROM clinics c
		JOIN clinic_members m ON m.clinic_id = c.id
		WHERE m.user_id = $1::uuid AND c.is_archived = $2
		ORDER BY c.is_primary DESC, m.is_default DESC, c.created_at ASC`,
		userID, archived,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.ClinicDTO{}
	for rows.Next() {
		clinic, err := scanClinic(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, clinic)
	}
	return out, nil
}

func (r *Repository) GetClinic(ctx context.Context, pool *pgxpool.Pool, id, userID string) (model.ClinicDTO, error) {
	row := pool.QueryRow(ctx, `
		SELECT c.id::text, c.name, c.clinic_type, c.location, c.color,
		       c.invite_code, m.role, m.is_default, m.last_accessed_at, c.created_at,
		       c.logo_url, c.address, c.locality, c.pin_code, c.state, c.country,
		       c.phones, c.email, c.website, c.gstin, c.facility_id,
		       c.time_format, c.system_language, c.time_zone, c.date_format, c.currency,
		       c.timings, c.is_primary, c.is_archived
		FROM clinics c
		JOIN clinic_members m ON m.clinic_id = c.id
		WHERE c.id = $1::uuid AND m.user_id = $2::uuid`,
		id, userID,
	)
	return scanClinic(row)
}

func (r *Repository) CreateClinic(ctx context.Context, pool *pgxpool.Pool, userID, inviteCode string, req model.CreateClinicRequest) (string, error) {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer tx.Rollback(ctx)

	if req.MakePrimary {
		if _, err := tx.Exec(ctx, `UPDATE clinics SET is_primary = false WHERE is_primary = true`); err != nil {
			return "", err
		}
	}

	phonesJSON, _ := json.Marshal(req.Phones)
	timingsJSON, _ := json.Marshal(req.Timings)

	var id string
	err = tx.QueryRow(ctx, `
		INSERT INTO clinics (
			name, clinic_type, location, color, invite_code,
			logo_url, address, locality, pin_code, state, country,
			phones, email, website, gstin, facility_id,
			time_format, system_language, time_zone, date_format, currency,
			timings, is_primary
		)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12::jsonb,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22::jsonb,$23)
		RETURNING id::text`,
		req.Name, req.ClinicType, req.Location, req.Color, inviteCode,
		req.LogoURL, req.Address, req.Locality, req.PinCode, req.State, req.Country,
		string(phonesJSON), req.Email, req.Website, req.GSTIN, req.FacilityID,
		req.TimeFormat, req.SystemLanguage, req.TimeZone, req.DateFormat, req.Currency,
		string(timingsJSON), req.MakePrimary,
	).Scan(&id)
	if err != nil {
		return "", err
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO clinic_members (clinic_id, user_id, role, is_default)
		VALUES ($1::uuid, $2::uuid, 'admin', false)`,
		id, userID,
	); err != nil {
		return "", err
	}

	if err := tx.Commit(ctx); err != nil {
		return "", err
	}
	return id, nil
}

func (r *Repository) UpdateClinic(ctx context.Context, pool *pgxpool.Pool, id, userID string, req model.UpdateClinicRequest) (bool, error) {
	sets := []string{}
	args := []any{}
	push := func(col string, v any, cast string) {
		args = append(args, v)
		if cast != "" {
			sets = append(sets, col+" = $"+utils.Itoa(len(args))+"::"+cast)
		} else {
			sets = append(sets, col+" = $"+utils.Itoa(len(args)))
		}
	}
	if req.Name != nil {
		push("name", *req.Name, "")
	}
	if req.ClinicType != nil {
		push("clinic_type", *req.ClinicType, "")
	}
	if req.Location != nil {
		push("location", *req.Location, "")
	}
	if req.Color != nil {
		push("color", *req.Color, "")
	}
	if req.LogoURL != nil {
		push("logo_url", *req.LogoURL, "")
	}
	if req.Address != nil {
		push("address", *req.Address, "")
	}
	if req.Locality != nil {
		push("locality", *req.Locality, "")
	}
	if req.PinCode != nil {
		push("pin_code", *req.PinCode, "")
	}
	if req.State != nil {
		push("state", *req.State, "")
	}
	if req.Country != nil {
		push("country", *req.Country, "")
	}
	if req.Phones != nil {
		b, _ := json.Marshal(*req.Phones)
		push("phones", string(b), "jsonb")
	}
	if req.Email != nil {
		push("email", *req.Email, "")
	}
	if req.Website != nil {
		push("website", *req.Website, "")
	}
	if req.GSTIN != nil {
		push("gstin", *req.GSTIN, "")
	}
	if req.FacilityID != nil {
		push("facility_id", *req.FacilityID, "")
	}
	if req.TimeFormat != nil {
		push("time_format", *req.TimeFormat, "")
	}
	if req.SystemLanguage != nil {
		push("system_language", *req.SystemLanguage, "")
	}
	if req.TimeZone != nil {
		push("time_zone", *req.TimeZone, "")
	}
	if req.DateFormat != nil {
		push("date_format", *req.DateFormat, "")
	}
	if req.Currency != nil {
		push("currency", *req.Currency, "")
	}
	if req.Timings != nil {
		b, _ := json.Marshal(*req.Timings)
		push("timings", string(b), "jsonb")
	}
	if len(sets) == 0 {
		return false, nil
	}
	sets = append(sets, "updated_at = now()")

	args = append(args, id, userID)
	query := "UPDATE clinics SET " + strings.Join(sets, ", ") +
		" WHERE id = $" + utils.Itoa(len(args)-1) + "::uuid " +
		"AND EXISTS (SELECT 1 FROM clinic_members m WHERE m.clinic_id = clinics.id AND m.user_id = $" + utils.Itoa(len(args)) + "::uuid)"
	_, err := pool.Exec(ctx, query, args...)
	return true, err
}

func (r *Repository) FlagClinic(ctx context.Context, pool *pgxpool.Pool, sql, id string) error {
	_, err := pool.Exec(ctx, sql, id)
	return err
}

func (r *Repository) MakePrimary(ctx context.Context, pool *pgxpool.Pool, id string) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `UPDATE clinics SET is_primary = false WHERE is_primary = true`); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE clinics SET is_primary = true, is_archived = false, updated_at = now()
		WHERE id = $1::uuid`, id,
	); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *Repository) FindClinicByInvite(ctx context.Context, pool *pgxpool.Pool, code string) (string, error) {
	var id string
	err := pool.QueryRow(ctx,
		`SELECT id::text FROM clinics WHERE invite_code = $1 AND is_archived = false`, code,
	).Scan(&id)
	return id, err
}

func (r *Repository) JoinClinic(ctx context.Context, pool *pgxpool.Pool, clinicID, userID string) error {
	_, err := pool.Exec(ctx, `
		INSERT INTO clinic_members (clinic_id, user_id, role, is_default)
		VALUES ($1::uuid, $2::uuid, 'member', false)
		ON CONFLICT (clinic_id, user_id) DO NOTHING`,
		clinicID, userID,
	)
	return err
}
