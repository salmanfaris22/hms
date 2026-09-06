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
		INSERT INTO appointments
			(clinic_id, patient_id, provider_id, provider_name, kind, scheduled_at,
			 duration_min, notes, is_emergency, booking_mode, marked_by)
		VALUES ($1::uuid, NULLIF($2,'')::uuid, NULLIF($3,'')::uuid, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id::text`,
		req.ClinicID, req.PatientID, req.ProviderID, req.ProviderName,
		req.Kind, scheduled, req.DurationMin, req.Notes,
		req.IsEmergency, req.BookingMode, req.MarkedBy,
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
		// The queue reads these to show how long someone has waited and how long
		// the session has run (3453:44305), so the moments are stamped here rather
		// than guessed later. Each is written once; re-entering a status keeps the
		// original time.
		n := utils.Itoa(len(args))
		sets = append(sets,
			"checked_in_at = CASE WHEN $"+n+" = 'checked_in' AND checked_in_at IS NULL THEN now() ELSE checked_in_at END",
			"visit_started_at = CASE WHEN $"+n+" = 'in_progress' AND visit_started_at IS NULL THEN now() ELSE visit_started_at END",
			"visit_ended_at = CASE WHEN $"+n+" = 'completed' AND visit_ended_at IS NULL THEN now() ELSE visit_ended_at END",
			// the token is the patient's place in that clinic's queue for the day
			"token = CASE WHEN $"+n+" = 'checked_in' AND token = '' THEN 'T-' || lpad(("+
				"SELECT count(*) + 1 FROM appointments q"+
				" WHERE q.clinic_id = appointments.clinic_id AND q.token <> ''"+
				" AND q.scheduled_at::date = appointments.scheduled_at::date)::text, 3, '0')"+
				" ELSE token END",
		)
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
	if req.ProviderID != nil {
		// reassigning to nobody is allowed; NULLIF keeps the column clean
		args = append(args, *req.ProviderID)
		sets = append(sets, "provider_id = NULLIF($"+utils.Itoa(len(args))+",'')::uuid")
	}
	if req.IsEmergency != nil {
		push("is_emergency", *req.IsEmergency)
	}
	if req.Kind != nil {
		push("kind", *req.Kind)
	}
	if req.DurationMin != nil {
		push("duration_min", *req.DurationMin)
	}
	if req.BookingMode != nil {
		push("booking_mode", *req.BookingMode)
	}
	if req.MarkedBy != nil {
		push("marked_by", *req.MarkedBy)
	}
	if req.PatientID != nil {
		// an event slot has no patient, so an empty string clears it
		args = append(args, *req.PatientID)
		sets = append(sets, "patient_id = NULLIF($"+utils.Itoa(len(args))+",'')::uuid")
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
		       followup_set_by, default_followup_count, default_followup_days, followup_rules,
		       consultation_fee_cents
		FROM appointment_settings WHERE clinic_id = $1::uuid`, clinicID,
	).Scan(
		&out.SlotDurationMin,
		&out.OnlineBookingConfirmation,
		&out.FreeFollowupEnabled,
		&out.FollowupSetBy,
		&out.DefaultFollowupCount,
		&out.DefaultFollowupDays,
		&rulesRaw,
		&out.ConsultationFeeCents,
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

func (r *Repository) PutSettings(ctx context.Context, pool *pgxpool.Pool, clinicID string, slot int, onlineConf, followupOn bool, setBy string, defaultCount, defaultDays int, rules []model.FollowupRule, feeCents int) error {
	rulesJSON, _ := json.Marshal(rules)
	_, err := pool.Exec(ctx, `
		INSERT INTO appointment_settings (
			clinic_id, slot_duration_min, online_booking_confirmation,
			free_followup_enabled, followup_set_by,
			default_followup_count, default_followup_days,
			followup_rules, consultation_fee_cents, updated_at
		)
		VALUES ($1::uuid, $2, $3, $4, $5, $6, $7, $8::jsonb, $9, now())
		ON CONFLICT (clinic_id) DO UPDATE SET
			slot_duration_min = EXCLUDED.slot_duration_min,
			online_booking_confirmation = EXCLUDED.online_booking_confirmation,
			free_followup_enabled = EXCLUDED.free_followup_enabled,
			followup_set_by = EXCLUDED.followup_set_by,
			default_followup_count = EXCLUDED.default_followup_count,
			default_followup_days = EXCLUDED.default_followup_days,
			followup_rules = EXCLUDED.followup_rules,
			consultation_fee_cents = EXCLUDED.consultation_fee_cents,
			updated_at = now()`,
		clinicID, slot, onlineConf, followupOn, setBy,
		defaultCount, defaultDays, string(rulesJSON), feeCents,
	)
	return err
}

// Calendar — one clinic's day (3424:46141), joined to the patient so the chip can
// name them and to the staff profile so a column knows whose it is.

func (r *Repository) Calendar(
	ctx context.Context, pool *pgxpool.Pool, clinicID string, from, to time.Time,
) ([]model.CalendarAppointment, error) {
	rows, err := pool.Query(ctx, `
		SELECT coalesce(a.id::text, ''), coalesce(a.patient_id::text, ''),
		       btrim(coalesce(p.first_name,'') || ' ' || coalesce(p.last_name,'')),
		       -- the queue prints age, sex and a phone under the name
		       CASE WHEN p.date_of_birth IS NULL THEN 0
		            ELSE date_part('year', age(p.date_of_birth))::int END,
		       coalesce(p.sex,''),
		       coalesce(
		         btrim(coalesce(p.phones->0->>'countryCode','') || ' ' || coalesce(p.phones->0->>'number','')),
		         ''),
		       coalesce(NULLIF(ss.name,''), NULLIF(pc.name,''), ''),
		       coalesce(p.photo_url,''),
		       coalesce(c.name,''),
		       coalesce(a.provider_id::text, ''),
		       -- the staff profile names them best, the user row next, and the
		       -- free-text provider_name last, for bookings against a non-user
		       coalesce(
		         NULLIF(btrim(coalesce(sp.first_name,'') || ' ' || coalesce(sp.last_name,'')), ''),
		         NULLIF(u.full_name, ''),
		         a.provider_name
		       ),
		       a.kind, a.scheduled_at, a.duration_min, a.status, a.is_emergency, a.notes,
		       a.booking_mode, a.marked_by,
		       a.checked_in_at, a.visit_started_at, a.visit_ended_at, a.token,
		       EXISTS (
		         SELECT 1 FROM patient_prescriptions pr
		         WHERE pr.appointment_id = a.id AND pr.doc <> '{}'::jsonb
		       )
		FROM appointments a
		LEFT JOIN patients p ON p.id = a.patient_id
		LEFT JOIN staff_profiles sp ON sp.user_id = a.provider_id
		LEFT JOIN users u ON u.id = a.provider_id
		LEFT JOIN clinics c ON c.id = a.clinic_id
		LEFT JOIN special_statuses ss ON ss.id = p.special_status_id
		LEFT JOIN patient_categories pc ON pc.id = p.category_id
		WHERE a.clinic_id = $1::uuid
		  AND a.scheduled_at >= $2 AND a.scheduled_at < $3
		ORDER BY a.scheduled_at`, clinicID, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.CalendarAppointment{}
	for rows.Next() {
		var a model.CalendarAppointment
		var at time.Time
		var checkedIn, started, ended *time.Time
		if err := rows.Scan(&a.ID, &a.PatientID, &a.PatientName,
			&a.PatientAge, &a.PatientSex, &a.PatientPhone, &a.PatientTag, &a.PatientPhotoURL,
			&a.ClinicName,
			&a.ProviderID, &a.ProviderName,
			&a.Kind, &at, &a.DurationMin, &a.Status, &a.IsEmergency, &a.Notes,
			&a.BookingMode, &a.MarkedBy, &checkedIn, &started, &ended, &a.Token,
			&a.HasPrescription); err != nil {
			return nil, err
		}
		a.ScheduledAt = at.Format(time.RFC3339)
		if checkedIn != nil {
			a.CheckedInAt = checkedIn.Format(time.RFC3339)
		}
		if started != nil {
			a.VisitStartedAt = started.Format(time.RFC3339)
		}
		if ended != nil {
			a.VisitEndedAt = ended.Format(time.RFC3339)
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// Blocks overlapping the window, not merely starting inside it — a block that
// began yesterday still greys out this morning.
func (r *Repository) Blocks(
	ctx context.Context, pool *pgxpool.Pool, clinicID string, from, to time.Time,
) ([]model.Block, error) {
	rows, err := pool.Query(ctx, `
		SELECT id::text, coalesce(provider_id::text, ''), starts_at, ends_at, reason
		FROM appointment_blocks
		WHERE clinic_id = $1::uuid AND starts_at < $3 AND ends_at > $2
		ORDER BY starts_at`, clinicID, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.Block{}
	for rows.Next() {
		var b model.Block
		var s, e time.Time
		if err := rows.Scan(&b.ID, &b.ProviderID, &s, &e, &b.Reason); err != nil {
			return nil, err
		}
		b.StartsAt, b.EndsAt = s.Format(time.RFC3339), e.Format(time.RFC3339)
		out = append(out, b)
	}
	return out, rows.Err()
}

func (r *Repository) CreateBlock(
	ctx context.Context, pool *pgxpool.Pool, clinicID, providerID string, starts, ends time.Time, reason string,
) (string, error) {
	var id string
	err := pool.QueryRow(ctx, `
		INSERT INTO appointment_blocks (clinic_id, provider_id, starts_at, ends_at, reason)
		VALUES ($1::uuid, NULLIF($2,'')::uuid, $3, $4, $5)
		RETURNING id::text`, clinicID, providerID, starts, ends, reason).Scan(&id)
	return id, err
}

func (r *Repository) DeleteBlock(ctx context.Context, pool *pgxpool.Pool, clinicID, id string) error {
	_, err := pool.Exec(ctx,
		`DELETE FROM appointment_blocks WHERE id = $1::uuid AND clinic_id = $2::uuid`, id, clinicID)
	return err
}

// Delete removes an appointment, scoped to a clinic the caller belongs to.
func (r *Repository) Delete(ctx context.Context, pool *pgxpool.Pool, id, userID string) (bool, error) {
	tag, err := pool.Exec(ctx, `
		DELETE FROM appointments
		WHERE id = $1::uuid
		  AND clinic_id IN (SELECT clinic_id FROM clinic_members WHERE user_id = $2::uuid)`,
		id, userID,
	)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

// The prescription pad's document, one per appointment (3467:53677).

func (r *Repository) Prescription(
	ctx context.Context, pool *pgxpool.Pool, clinicID, appointmentID string,
) (model.Prescription, error) {
	var p model.Prescription
	var updated time.Time
	err := pool.QueryRow(ctx, `
		SELECT id::text, patient_id::text, coalesce(appointment_id::text,''), doc, updated_at
		FROM patient_prescriptions
		WHERE clinic_id = $1::uuid AND appointment_id = $2::uuid`, clinicID, appointmentID,
	).Scan(&p.ID, &p.PatientID, &p.AppointmentID, &p.Doc, &updated)
	if err != nil {
		return p, err
	}
	p.UpdatedAt = updated.Format(time.RFC3339)
	return p, nil
}

// PastVisits lists this patient's earlier appointments, newest first, each with
// whatever the pad recorded at it. Appointments drive the list, not
// prescriptions: a visit where the pad was never opened still belongs in the
// history, it simply has an empty document.
func (r *Repository) PastVisits(
	ctx context.Context, pool *pgxpool.Pool, clinicID, patientID string,
) ([]model.PastVisit, error) {
	rows, err := pool.Query(ctx, `
		SELECT coalesce(p.id::text, ''),
		       a.id::text,
		       a.scheduled_at,
		       coalesce(a.kind, ''),
		       coalesce(a.status, ''),
		       coalesce(a.notes, ''),
		       coalesce(NULLIF(trim(concat_ws(' ', sp.first_name, sp.last_name)), ''),
		                NULLIF(u.full_name, ''), coalesce(a.provider_name, '')),
		       coalesce(p.doc, '{}'::jsonb),
		       coalesce(p.updated_at, a.scheduled_at)
		FROM appointments a
		LEFT JOIN patient_prescriptions p
		       ON p.appointment_id = a.id AND p.clinic_id = a.clinic_id
		LEFT JOIN users u ON u.id = a.provider_id
		LEFT JOIN staff_profiles sp ON sp.user_id = a.provider_id
		WHERE a.clinic_id = $1::uuid AND a.patient_id = $2::uuid
		ORDER BY a.scheduled_at DESC`, clinicID, patientID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.PastVisit{}
	for rows.Next() {
		var v model.PastVisit
		var when, updated time.Time
		if err := rows.Scan(&v.ID, &v.AppointmentID, &when, &v.Kind, &v.Status, &v.Notes,
			&v.ProviderName, &v.Doc, &updated); err != nil {
			return nil, err
		}
		v.ScheduledAt = when.Format(time.RFC3339)
		v.UpdatedAt = updated.Format(time.RFC3339)
		out = append(out, v)
	}
	return out, rows.Err()
}

// SavePrescription writes the pad, replacing whatever was there for that
// appointment — the pad always sends the whole document.
func (r *Repository) SavePrescription(
	ctx context.Context, pool *pgxpool.Pool, clinicID, appointmentID, patientID string, doc []byte,
) (string, error) {
	var id string
	err := pool.QueryRow(ctx, `
		INSERT INTO patient_prescriptions (clinic_id, patient_id, appointment_id, doc)
		VALUES ($1::uuid, $2::uuid, $3::uuid, $4::jsonb)
		ON CONFLICT (appointment_id) WHERE appointment_id IS NOT NULL
		DO UPDATE SET doc = EXCLUDED.doc, updated_at = now()
		RETURNING id::text`, clinicID, patientID, appointmentID, string(doc)).Scan(&id)
	return id, err
}

// RecordConsultationFee writes the money taken at booking: one invoice for the
// appointment, and a payment row per split. Both live in the ledger the billing
// screens already read, so a fee collected here shows up there without any
// special case.
func (r *Repository) RecordConsultationFee(
	ctx context.Context, pool *pgxpool.Pool,
	clinicID, patientID, appointmentID string, splits []model.FeeSplit,
) error {
	total := 0
	for _, sp := range splits {
		total += sp.AmountCents
	}
	if total <= 0 {
		return nil
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var invoiceID string
	// paid in full at the counter, so it opens and closes in one step
	err = tx.QueryRow(ctx, `
		INSERT INTO invoices (
			clinic_id, patient_id, appointment_id, number,
			amount_total_cents, amount_paid_cents, amount_due_cents,
			status, issued_at, line_items
		)
		VALUES ($1::uuid, $2::uuid, $3::uuid,
		        'CF-' || to_char(now(), 'YYYYMMDD') || '-' || substr(md5(random()::text), 1, 6),
		        $4, $4, 0, 'paid', current_date,
		        jsonb_build_array(jsonb_build_object('name', 'Consultation Fee', 'amount', $4)))
		RETURNING id::text`, clinicID, patientID, appointmentID, total,
	).Scan(&invoiceID)
	if err != nil {
		return err
	}
	for _, sp := range splits {
		if sp.AmountCents <= 0 {
			continue
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO invoice_payments (clinic_id, invoice_id, patient_id, amount_cents, method)
			VALUES ($1::uuid, $2::uuid, $3::uuid, $4, $5)`,
			clinicID, invoiceID, patientID, sp.AmountCents, sp.Method,
		); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}
