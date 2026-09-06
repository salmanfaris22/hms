package model

type BookRequest struct {
	ClinicID     string `json:"clinicId"`
	PatientID    string `json:"patientId"`
	ProviderName string `json:"providerName"`
	Kind         string `json:"kind"`
	ScheduledAt  string `json:"scheduledAt"`
	DurationMin  int    `json:"durationMin"`
	Notes        string `json:"notes"`
}

type UpdateRequest struct {
	Status      *string `json:"status,omitempty"`
	ScheduledAt *string `json:"scheduledAt,omitempty"`
	Notes       *string `json:"notes,omitempty"`
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
}

type PutSettingsRequest struct {
	SlotDurationMin           *int            `json:"slotDurationMin,omitempty"`
	OnlineBookingConfirmation *bool           `json:"onlineBookingConfirmation,omitempty"`
	FreeFollowupEnabled       *bool           `json:"freeFollowupEnabled,omitempty"`
	FollowupSetBy             *string         `json:"followupSetBy,omitempty"`
	DefaultFollowupCount      *int            `json:"defaultFollowupCount,omitempty"`
	DefaultFollowupDays       *int            `json:"defaultFollowupDays,omitempty"`
	FollowupRules             *[]FollowupRule `json:"followupRules,omitempty"`
}
