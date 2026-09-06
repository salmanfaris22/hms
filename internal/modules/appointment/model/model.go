package model

import "encoding/json"

type BookRequest struct {
	ClinicID  string `json:"clinicId"`
	PatientID string `json:"patientId"`
	// the staff user the appointment belongs to; ProviderName remains for
	// bookings made against someone who is not a staff user
	ProviderID   string `json:"providerId"`
	ProviderName string `json:"providerName"`
	Kind         string `json:"kind"`
	ScheduledAt  string `json:"scheduledAt"`
	DurationMin  int    `json:"durationMin"`
	Notes        string `json:"notes"`
	IsEmergency  bool   `json:"isEmergency"`
	// the form's booking tabs: offline | online | temporary | event
	BookingMode string `json:"bookingMode"`
	// the form's "Marked by" footer, e.g. "Front Desk"
	MarkedBy string `json:"markedBy"`
	// "Collect Consultation Fee" (5332:76270). The split says how the money was
	// taken; an empty list means nothing was collected at booking. An Event Slot
	// never carries one — it reserves time without a patient to charge.
	FeeSplits []FeeSplit `json:"feeSplits,omitempty"`
}

type UpdateRequest struct {
	Status      *string `json:"status,omitempty"`
	ProviderID  *string `json:"providerId,omitempty"`
	IsEmergency *bool   `json:"isEmergency,omitempty"`
	ScheduledAt *string `json:"scheduledAt,omitempty"`
	Notes       *string `json:"notes,omitempty"`
	// the booking form edits these too (4599:60437's Edit button)
	PatientID   *string `json:"patientId,omitempty"`
	Kind        *string `json:"kind,omitempty"`
	DurationMin *int    `json:"durationMin,omitempty"`
	BookingMode *string `json:"bookingMode,omitempty"`
	MarkedBy    *string `json:"markedBy,omitempty"`
}

type FollowupRule struct {
	Name      string `json:"name"`
	Count     int    `json:"count"`
	Days      int    `json:"days"`
	Specialty string `json:"specialty,omitempty"`
}

type SettingsDTO struct {
	SlotDurationMin           int            `json:"slotDurationMin"`
	OnlineBookingConfirmation bool           `json:"onlineBookingConfirmation"`
	FreeFollowupEnabled       bool           `json:"freeFollowupEnabled"`
	FollowupSetBy             string         `json:"followupSetBy"`
	DefaultFollowupCount      int            `json:"defaultFollowupCount"`
	DefaultFollowupDays       int            `json:"defaultFollowupDays"`
	FollowupRules             []FollowupRule `json:"followupRules"`
	// What the clinic collects at the time of booking, in cents. 0 means it
	// does not collect one, and the booking dialog hides the block.
	ConsultationFeeCents int `json:"consultationFeeCents"`
}

type PutSettingsRequest struct {
	SlotDurationMin           *int            `json:"slotDurationMin,omitempty"`
	OnlineBookingConfirmation *bool           `json:"onlineBookingConfirmation,omitempty"`
	FreeFollowupEnabled       *bool           `json:"freeFollowupEnabled,omitempty"`
	FollowupSetBy             *string         `json:"followupSetBy,omitempty"`
	DefaultFollowupCount      *int            `json:"defaultFollowupCount,omitempty"`
	DefaultFollowupDays       *int            `json:"defaultFollowupDays,omitempty"`
	FollowupRules             *[]FollowupRule `json:"followupRules,omitempty"`
	ConsultationFeeCents      *int            `json:"consultationFeeCents,omitempty"`
}

// FeeSplit is one way the consultation fee was paid. A fee may be split across
// several methods — part cash, part card — so this is a list on the booking.
type FeeSplit struct {
	Method      string `json:"method"`
	AmountCents int    `json:"amountCents"`
}

// CalendarAppointment is one chip on the calendar (3424:46141), resolved past the
// foreign keys so the grid can draw it without a second round trip.
type CalendarAppointment struct {
	ID          string `json:"id"`
	PatientID   string `json:"patientId"`
	PatientName string `json:"patientName"`
	// what the queue prints under the name (3453:44305)
	PatientAge   int    `json:"patientAge"`
	PatientSex   string `json:"patientSex"`
	PatientPhone string `json:"patientPhone"`
	// the patient's special status, else their category — the tag beside the
	// name in the checked-in view (3733:53966)
	PatientTag string `json:"patientTag"`
	// their photograph, so the queue shows a face rather than initials
	PatientPhotoURL string `json:"patientPhotoUrl"`
	ClinicName      string `json:"clinicName"`
	ProviderID      string `json:"providerId"`
	ProviderName    string `json:"providerName"`
	Kind            string `json:"kind"`
	ScheduledAt     string `json:"scheduledAt"`
	DurationMin     int    `json:"durationMin"`
	Status          string `json:"status"`
	IsEmergency     bool   `json:"isEmergency"`
	Notes           string `json:"notes"`
	BookingMode     string `json:"bookingMode"`
	MarkedBy        string `json:"markedBy"`
	// stamped as the appointment moves through the queue; blank until it does
	CheckedInAt    string `json:"checkedInAt"`
	VisitStartedAt string `json:"visitStartedAt"`
	VisitEndedAt   string `json:"visitEndedAt"`
	Token          string `json:"token"`
	// whether a prescription pad has already been written for this appointment,
	// which decides how the Rx button reads (4022:56208 vs 3823:58812)
	HasPrescription bool `json:"hasPrescription"`
}

// Block is one "Blocked" span. A blank ProviderID blocks every column.
type Block struct {
	ID         string `json:"id"`
	ProviderID string `json:"providerId"`
	StartsAt   string `json:"startsAt"`
	EndsAt     string `json:"endsAt"`
	Reason     string `json:"reason"`
}

type CreateBlockRequest struct {
	ProviderID string `json:"providerId"`
	StartsAt   string `json:"startsAt"`
	EndsAt     string `json:"endsAt"`
	Reason     string `json:"reason"`
}

// CalendarDay is everything one screenful of the calendar needs.
type CalendarDay struct {
	Appointments []CalendarAppointment `json:"appointments"`
	Blocks       []Block               `json:"blocks"`
}

// Prescription is the pad's whole document (3467:53677). Its shape is the
// frontend's; the server stores and returns it untouched.
type Prescription struct {
	ID            string          `json:"id"`
	PatientID     string          `json:"patientId"`
	AppointmentID string          `json:"appointmentId"`
	Doc           json.RawMessage `json:"doc"`
	UpdatedAt     string          `json:"updatedAt"`
}

// PastVisit is one earlier consultation as the Visit History panel shows it:
// the prescription document plus the appointment it belongs to.
type PastVisit struct {
	ID            string `json:"id"`
	AppointmentID string `json:"appointmentId"`
	ScheduledAt   string `json:"scheduledAt"`
	Kind          string `json:"kind"`
	Status        string `json:"status"`
	// what the appointment was booked for, which heads the timeline entry
	Notes        string          `json:"notes"`
	ProviderName string          `json:"providerName"`
	Doc          json.RawMessage `json:"doc"`
	UpdatedAt    string          `json:"updatedAt"`
}

type SavePrescriptionRequest struct {
	PatientID string          `json:"patientId"`
	Doc       json.RawMessage `json:"doc"`
}
