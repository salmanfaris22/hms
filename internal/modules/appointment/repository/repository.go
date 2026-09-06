package repository

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/salman/hms-backend/internal/core/utils"
	"github.com/salman/hms-backend/internal/modules/appointment/model"
)

type Repository struct{}

func New() *Repository { return &Repository{} }

func (r *Repository) Book(ctx context.Context, pool *pgxpool.Pool, req model.BookRequest, scheduled time.Time) (string, error) {
	var id string
	err := pool.QueryRow(ctx, `
		INSERT INTO appointments (clinic_id, patient_id, provider_name, kind, scheduled_at, duration_min, notes)
		VALUES ($1::uuid, $2::uuid, $3, $4, $5, $6, $7)
		RETURNING id::text`,
		req.ClinicID, req.PatientID, req.ProviderName, req.Kind, scheduled, req.DurationMin, req.Notes,
	).Scan(&id)
	return id, err
}

func (r *Repository) RecordVisitEvent(ctx context.Context, pool *pgxpool.Pool, patientID, providerName, kind string, scheduled time.Time) {
	_, _ = pool.Exec(ctx, `
		INSERT INTO patient_events (patient_id, kind, title, description, occurred_at)
		VALUES ($1::uuid, 'visit', $2, $3, $4)`,
		patientID, "Appointment scheduled with "+providerName, kind, scheduled,
	)
}

func (r *Repository) Update(ctx context.Context, pool *pgxpool.Pool, id, userID string, req model.UpdateRequest) (bool, error) {
	sets := []string{"updated_at = now()"}
	args := []any{}
	push := func(col string, v any) {
		args = append(args, v)
		sets = append(sets, col+" = $"+utils.Itoa(len(args)))
	}
	if req.Status != nil {
		push("status", *req.Status)
	}
	if req.ScheduledAt != nil {
		t, err := time.Parse(time.RFC3339, *req.ScheduledAt)
		if err != nil {
			return false, err
		}
		push("scheduled_at", t)
	}
	if req.Notes != nil {
		push("notes", *req.Notes)
	}
	if len(sets) == 1 {
		return false, nil
	}

	args = append(args, id, userID)
	query := "UPDATE appointments SET " + strings.Join(sets, ", ") +
		" WHERE id = $" + utils.Itoa(len(args)-1) + "::uuid" +
		" AND clinic_id IN (SELECT clinic_id FROM clinic_members WHERE user_id = $" + utils.Itoa(len(args)) + "::uuid)"
	_, err := pool.Exec(ctx, query, args...)
	return true, err
}

func (r *Repository) GetSettings(ctx context.Context, pool *pgxpool.Pool, clinicID string) (model.SettingsDTO, error) {
	out := model.SettingsDTO{
		SlotDurationMin:      30,
		FollowupSetBy:        "default",
		DefaultFollowupCount: 2,
		DefaultFollowupDays:  30,
		FollowupRules:        []model.FollowupRule{},
	}
	var rulesRaw []byte
	err := pool.QueryRow(ctx, `
		SELECT slot_duration_min, online_booking_confirmation, free_followup_enabled,
		       followup_set_by, default_followup_count, default_followup_days, followup_rules
		FROM appointment_settings WHERE clinic_id = $1::uuid`, clinicID,
	).Scan(
		&out.SlotDurationMin,
		&out.OnlineBookingConfirmation,
		&out.FreeFollowupEnabled,
		&out.FollowupSetBy,
		&out.DefaultFollowupCount,
		&out.DefaultFollowupDays,
		&rulesRaw,
	)
	if err != nil {
		return out, err
	}
	if len(rulesRaw) > 0 {
		_ = json.Unmarshal(rulesRaw, &out.FollowupRules)
	}
	if out.FollowupRules == nil {
		out.FollowupRules = []model.FollowupRule{}
	}
	return out, nil
}

func (r *Repository) PutSettings(ctx context.Context, pool *pgxpool.Pool, clinicID string, slot int, onlineConf, followupOn bool, setBy string, defaultCount, defaultDays int, rules []model.FollowupRule) error {
	rulesJSON, _ := json.Marshal(rules)
	_, err := pool.Exec(ctx, `
		INSERT INTO appointment_settings (
			clinic_id, slot_duration_min, online_booking_confirmation,
			free_followup_enabled, followup_set_by,
			default_followup_count, default_followup_days,
			followup_rules, updated_at
		)
		VALUES ($1::uuid, $2, $3, $4, $5, $6, $7, $8::jsonb, now())
		ON CONFLICT (clinic_id) DO UPDATE SET
			slot_duration_min = EXCLUDED.slot_duration_min,
			online_booking_confirmation = EXCLUDED.online_booking_confirmation,
			free_followup_enabled = EXCLUDED.free_followup_enabled,
			followup_set_by = EXCLUDED.followup_set_by,
			default_followup_count = EXCLUDED.default_followup_count,
			default_followup_days = EXCLUDED.default_followup_days,
			followup_rules = EXCLUDED.followup_rules,
			updated_at = now()`,
		clinicID, slot, onlineConf, followupOn, setBy,
		defaultCount, defaultDays, string(rulesJSON),
	)
	return err
}
