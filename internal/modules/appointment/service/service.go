package service

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/salman/hms-backend/internal/core/audit"
	"github.com/salman/hms-backend/internal/modules/appointment/model"
	"github.com/salman/hms-backend/internal/modules/appointment/repository"
	patientService "github.com/salman/hms-backend/internal/modules/patient/service"
)

// ValidationError is a caller mistake, not a server fault — mapErr turns it into
// a 400 so a bad range or a reversed span does not read as a crash.
type ValidationError struct{ Msg string }

func (e ValidationError) Error() string { return e.Msg }

func invalid(msg string) error { return ValidationError{Msg: msg} }

var (
	ErrTenantUnavail = errors.New("tenant unavailable")
	ErrForbidden     = errors.New("not a member of this clinic")
	ErrBadBody       = errors.New("invalid body")
	ErrNoUpdate      = errors.New("nothing to update")
	ErrBadTime       = errors.New("scheduledAt must be RFC3339")
)

type Meta struct {
	IP         string
	UserAgent  string
	ActorID    string
	ActorEmail string
	TenantID   string
	UserID     string
}

type Service struct {
	repo     *repository.Repository
	patients *patientService.Service
}

func New(repo *repository.Repository, patients *patientService.Service) *Service {
	return &Service{repo: repo, patients: patients}
}

func (s *Service) Book(ctx context.Context, meta Meta, req model.BookRequest) (string, error) {
	if req.ClinicID == "" || req.ScheduledAt == "" {
		return "", invalid("clinicId and scheduledAt are required")
	}
	if req.DurationMin <= 0 {
		req.DurationMin = 30
	}
	switch req.BookingMode {
	// the booking form's five tabs (3691:53076)
	case "offline", "online", "temporary", "event", "walkin":
	case "":
		req.BookingMode = "offline"
	default:
		return "", invalid("bookingMode must be offline, online, temporary, event or walkin")
	}
	// an event slot reserves the doctor's time and has no patient field at all
	// (4691:62738); every other tab must name one
	if req.BookingMode != "event" && req.PatientID == "" {
		return "", invalid("patientId is required")
	}
	scheduled, err := time.Parse(time.RFC3339, req.ScheduledAt)
	if err != nil {
		return "", ErrBadTime
	}

	pool, err := s.patients.Tenants().Pool(ctx, meta.TenantID)
	if err != nil {
		return "", ErrTenantUnavail
	}
	if err := s.patients.AssertMember(ctx, pool, meta.UserID, req.ClinicID); err != nil {
		return "", ErrForbidden
	}

	id, err := s.repo.Book(ctx, pool, req, scheduled)
	if err != nil {
		log.Printf("book appointment: %v", err)
		return "", errors.New("create failed")
	}
	// an event slot has no patient, so there is no record to write it onto
	if req.PatientID != "" {
		s.repo.RecordVisitEvent(ctx, pool, req.PatientID, req.ProviderName, req.Kind, scheduled)
	}

	audit.LogRaw(ctx, s.patients.Tenants().Registry(), meta.IP, meta.UserAgent, audit.Event{
		ActorID: meta.ActorID, ActorEmail: meta.ActorEmail, ActorKind: audit.ActorUser,
		TenantID: meta.TenantID, Action: "appointment.book", Resource: id,
		Metadata: map[string]any{"patientId": req.PatientID, "provider": req.ProviderName, "kind": req.Kind},
	})

	// "Collect Consultation Fee" — an Event Slot has no patient to charge, so it
	// never carries a split. A failure here must not lose the appointment, which
	// is already booked: it is reported in the trail and the booking stands.
	if len(req.FeeSplits) > 0 && req.PatientID != "" {
		if err := s.repo.RecordConsultationFee(ctx, pool, req.ClinicID, req.PatientID, id, req.FeeSplits); err != nil {
			log.Printf("record consultation fee for %s: %v", id, err)
		} else {
			audit.LogRaw(ctx, s.patients.Tenants().Registry(), meta.IP, meta.UserAgent, audit.Event{
				ActorID: meta.ActorID, ActorEmail: meta.ActorEmail, ActorKind: audit.ActorUser,
				TenantID: meta.TenantID, Action: "appointment.fee_collected", Resource: id,
				Metadata: map[string]any{"splits": len(req.FeeSplits)},
			})
		}
	}
	return id, nil
}

// The statuses an appointment can hold, in the order the queue walks them.
var validStatuses = map[string]bool{
	"scheduled": true, "confirmed": true, "checked_in": true,
	"in_progress": true, "completed": true, "cancelled": true, "rescheduled": true,
}

func (s *Service) Update(ctx context.Context, meta Meta, id string, req model.UpdateRequest) error {
	if req.Status != nil && !validStatuses[*req.Status] {
		return invalid("unknown status")
	}
	pool, err := s.patients.Tenants().Pool(ctx, meta.TenantID)
	if err != nil {
		return ErrTenantUnavail
	}
	ok, err := s.repo.Update(ctx, pool, id, meta.UserID, req)
	if err != nil {
		if err == ErrBadTime || errors.Is(err, ErrBadTime) {
			return ErrBadTime
		}
		if _, perr := time.Parse(time.RFC3339, ""); perr != nil && req.ScheduledAt != nil {
			// do nothing, err path covered
		}
		log.Printf("update appointment: %v", err)
		return errors.New("update failed")
	}
	if !ok {
		return ErrNoUpdate
	}
	audit.LogRaw(ctx, s.patients.Tenants().Registry(), meta.IP, meta.UserAgent, audit.Event{
		ActorID: meta.ActorID, ActorEmail: meta.ActorEmail, ActorKind: audit.ActorUser,
		TenantID: meta.TenantID, Action: "appointment.update", Resource: id,
	})
	return nil
}

func (s *Service) GetSettings(ctx context.Context, meta Meta, clinicID string) (model.SettingsDTO, error) {
	pool, err := s.patients.Tenants().Pool(ctx, meta.TenantID)
	if err != nil {
		return model.SettingsDTO{}, ErrTenantUnavail
	}
	if err := s.patients.AssertMember(ctx, pool, meta.UserID, clinicID); err != nil {
		return model.SettingsDTO{}, ErrForbidden
	}
	out, err := s.repo.GetSettings(ctx, pool, clinicID)
	if err != nil && !errSkipNoRows(err) {
		log.Printf("get appt settings: %v", err)
		return out, errors.New("db error")
	}
	return out, nil
}

func (s *Service) PutSettings(ctx context.Context, meta Meta, clinicID string, req model.PutSettingsRequest) error {
	pool, err := s.patients.Tenants().Pool(ctx, meta.TenantID)
	if err != nil {
		return ErrTenantUnavail
	}
	if err := s.patients.AssertMember(ctx, pool, meta.UserID, clinicID); err != nil {
		return ErrForbidden
	}
	slot := 30
	onlineConf := false
	followupOn := false
	setBy := "default"
	defaultCount := 2
	defaultDays := 30
	rules := []model.FollowupRule{}
	feeCents := 0

	if req.SlotDurationMin != nil {
		slot = *req.SlotDurationMin
	}
	if req.OnlineBookingConfirmation != nil {
		onlineConf = *req.OnlineBookingConfirmation
	}
	if req.FreeFollowupEnabled != nil {
		followupOn = *req.FreeFollowupEnabled
	}
	if req.FollowupSetBy != nil {
		setBy = *req.FollowupSetBy
	}
	if req.DefaultFollowupCount != nil {
		defaultCount = *req.DefaultFollowupCount
	}
	if req.DefaultFollowupDays != nil {
		defaultDays = *req.DefaultFollowupDays
	}
	if req.FollowupRules != nil {
		rules = *req.FollowupRules
	}
	if req.ConsultationFeeCents != nil && *req.ConsultationFeeCents >= 0 {
		feeCents = *req.ConsultationFeeCents
	}

	if err := s.repo.PutSettings(ctx, pool, clinicID, slot, onlineConf, followupOn, setBy, defaultCount, defaultDays, rules, feeCents); err != nil {
		log.Printf("put appt settings: %v", err)
		return errors.New("save failed")
	}
	return nil
}

func errSkipNoRows(err error) bool {
	return err != nil && errors.Is(err, pgx.ErrNoRows)
}

// Calendar returns one window of the clinic's schedule — the appointments and the
// blocked spans the grid draws over them (3424:46141).
func (s *Service) Calendar(ctx context.Context, meta Meta, clinicID, fromISO, toISO string) (model.CalendarDay, error) {
	var out model.CalendarDay
	if clinicID == "" {
		return out, invalid("clinic is required")
	}
	from, err := time.Parse(time.RFC3339, fromISO)
	if err != nil {
		return out, ErrBadTime
	}
	to, err := time.Parse(time.RFC3339, toISO)
	if err != nil {
		return out, ErrBadTime
	}
	if !to.After(from) {
		return out, invalid("to must be after from")
	}
	// a month view is the widest the frame offers; anything larger is a mistake
	if to.Sub(from) > 62*24*time.Hour {
		return out, invalid("range must be 62 days or less")
	}

	pool, err := s.patients.Tenants().Pool(ctx, meta.TenantID)
	if err != nil {
		return out, ErrTenantUnavail
	}
	if err := s.patients.AssertMember(ctx, pool, meta.UserID, clinicID); err != nil {
		return out, ErrForbidden
	}
	appts, err := s.repo.Calendar(ctx, pool, clinicID, from, to)
	if err != nil {
		log.Printf("calendar: %v", err)
		return out, errors.New("db error")
	}
	blocks, err := s.repo.Blocks(ctx, pool, clinicID, from, to)
	if err != nil {
		log.Printf("calendar blocks: %v", err)
		return out, errors.New("db error")
	}
	out.Appointments, out.Blocks = appts, blocks
	return out, nil
}

func (s *Service) CreateBlock(ctx context.Context, meta Meta, clinicID string, req model.CreateBlockRequest) (string, error) {
	if clinicID == "" {
		return "", invalid("clinic is required")
	}
	starts, err := time.Parse(time.RFC3339, req.StartsAt)
	if err != nil {
		return "", ErrBadTime
	}
	ends, err := time.Parse(time.RFC3339, req.EndsAt)
	if err != nil {
		return "", ErrBadTime
	}
	if !ends.After(starts) {
		return "", invalid("endsAt must be after startsAt")
	}

	pool, err := s.patients.Tenants().Pool(ctx, meta.TenantID)
	if err != nil {
		return "", ErrTenantUnavail
	}
	if err := s.patients.AssertMember(ctx, pool, meta.UserID, clinicID); err != nil {
		return "", ErrForbidden
	}
	id, err := s.repo.CreateBlock(ctx, pool, clinicID, req.ProviderID, starts, ends, req.Reason)
	if err != nil {
		log.Printf("create block: %v", err)
		return "", errors.New("insert failed")
	}
	audit.LogRaw(ctx, s.patients.Tenants().Registry(), meta.IP, meta.UserAgent, audit.Event{
		ActorID: meta.ActorID, ActorEmail: meta.ActorEmail, ActorKind: audit.ActorUser,
		TenantID: meta.TenantID, Action: "appointment.block.create", Resource: clinicID,
		Metadata: map[string]any{"from": req.StartsAt, "to": req.EndsAt, "provider": req.ProviderID},
	})
	return id, nil
}

func (s *Service) DeleteBlock(ctx context.Context, meta Meta, clinicID, id string) error {
	pool, err := s.patients.Tenants().Pool(ctx, meta.TenantID)
	if err != nil {
		return ErrTenantUnavail
	}
	if err := s.patients.AssertMember(ctx, pool, meta.UserID, clinicID); err != nil {
		return ErrForbidden
	}
	if err := s.repo.DeleteBlock(ctx, pool, clinicID, id); err != nil {
		log.Printf("delete block: %v", err)
		return errors.New("delete failed")
	}
	audit.LogRaw(ctx, s.patients.Tenants().Registry(), meta.IP, meta.UserAgent, audit.Event{
		ActorID: meta.ActorID, ActorEmail: meta.ActorEmail, ActorKind: audit.ActorUser,
		TenantID: meta.TenantID, Action: "appointment.block.delete", Resource: id,
	})
	return nil
}

// Delete removes an appointment outright — the detail dialog's Delete (4599:60437).
func (s *Service) Delete(ctx context.Context, meta Meta, id string) error {
	pool, err := s.patients.Tenants().Pool(ctx, meta.TenantID)
	if err != nil {
		return ErrTenantUnavail
	}
	found, err := s.repo.Delete(ctx, pool, id, meta.UserID)
	if err != nil {
		log.Printf("delete appointment: %v", err)
		return errors.New("delete failed")
	}
	if !found {
		return ErrForbidden
	}
	audit.LogRaw(ctx, s.patients.Tenants().Registry(), meta.IP, meta.UserAgent, audit.Event{
		ActorID: meta.ActorID, ActorEmail: meta.ActorEmail, ActorKind: audit.ActorUser,
		TenantID: meta.TenantID, Action: "appointment.delete", Resource: id,
	})
	return nil
}

// Prescription returns the pad for one appointment, or an empty document when
// it has never been written.
// PastVisits is the patient's earlier consultations, for the history panel.
func (s *Service) PastVisits(ctx context.Context, meta Meta, clinicID, patientID string) ([]model.PastVisit, error) {
	if clinicID == "" || patientID == "" {
		return nil, invalid("clinic and patient are required")
	}
	pool, err := s.patients.Tenants().Pool(ctx, meta.TenantID)
	if err != nil {
		return nil, ErrTenantUnavail
	}
	if err := s.patients.AssertMember(ctx, pool, meta.UserID, clinicID); err != nil {
		return nil, ErrForbidden
	}
	out, err := s.repo.PastVisits(ctx, pool, clinicID, patientID)
	if err != nil {
		log.Printf("past visits: %v", err)
		return nil, errors.New("db error")
	}
	return out, nil
}

func (s *Service) Prescription(ctx context.Context, meta Meta, clinicID, appointmentID string) (model.Prescription, error) {
	var out model.Prescription
	if clinicID == "" || appointmentID == "" {
		return out, invalid("clinic and appointment are required")
	}
	pool, err := s.patients.Tenants().Pool(ctx, meta.TenantID)
	if err != nil {
		return out, ErrTenantUnavail
	}
	if err := s.patients.AssertMember(ctx, pool, meta.UserID, clinicID); err != nil {
		return out, ErrForbidden
	}
	p, err := s.repo.Prescription(ctx, pool, clinicID, appointmentID)
	if err != nil {
		if errSkipNoRows(err) {
			return model.Prescription{AppointmentID: appointmentID, Doc: []byte("{}")}, nil
		}
		log.Printf("prescription: %v", err)
		return out, errors.New("db error")
	}
	return p, nil
}

func (s *Service) SavePrescription(
	ctx context.Context, meta Meta, clinicID, appointmentID string, req model.SavePrescriptionRequest,
) (string, error) {
	if clinicID == "" || appointmentID == "" || req.PatientID == "" {
		return "", invalid("clinic, appointment and patientId are required")
	}
	if len(req.Doc) == 0 {
		return "", invalid("doc is required")
	}
	pool, err := s.patients.Tenants().Pool(ctx, meta.TenantID)
	if err != nil {
		return "", ErrTenantUnavail
	}
	if err := s.patients.AssertMember(ctx, pool, meta.UserID, clinicID); err != nil {
		return "", ErrForbidden
	}
	id, err := s.repo.SavePrescription(ctx, pool, clinicID, appointmentID, req.PatientID, req.Doc)
	if err != nil {
		log.Printf("save prescription: %v", err)
		return "", errors.New("save failed")
	}
	audit.LogRaw(ctx, s.patients.Tenants().Registry(), meta.IP, meta.UserAgent, audit.Event{
		ActorID: meta.ActorID, ActorEmail: meta.ActorEmail, ActorKind: audit.ActorUser,
		TenantID: meta.TenantID, Action: "appointment.prescription.save", Resource: appointmentID,
		Metadata: map[string]any{"patientId": req.PatientID},
	})
	return id, nil
}
