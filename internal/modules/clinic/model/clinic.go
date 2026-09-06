package model

import "time"

type Phone struct {
	Type        string `json:"type"`
	CountryCode string `json:"countryCode"`
	Number      string `json:"number"`
	Label       string `json:"label,omitempty"`
}

type TimeSlot struct {
	Start string `json:"start"`
	End   string `json:"end"`
}

type DayTimings struct {
	Enabled bool       `json:"enabled"`
	Is24h   bool       `json:"is24h"`
	Slots   []TimeSlot `json:"slots"`
}

type Timings map[string]DayTimings

type ClinicDTO struct {
	ID             string     `json:"id"`
	Name           string     `json:"name"`
	ClinicType     string     `json:"clinicType"`
	Location       string     `json:"location"`
	Color          string     `json:"color"`
	InviteCode     string     `json:"inviteCode"`
	Role           string     `json:"role,omitempty"`
	IsDefault      bool       `json:"isDefault,omitempty"`
	LastAccessedAt *time.Time `json:"lastAccessedAt,omitempty"`
	CreatedAt      time.Time  `json:"createdAt"`
	LogoURL        string     `json:"logoUrl"`
	Address        string     `json:"address"`
	Locality       string     `json:"locality"`
	PinCode        string     `json:"pinCode"`
	State          string     `json:"state"`
	Country        string     `json:"country"`
	Phones         []Phone    `json:"phones"`
	Email          string     `json:"email"`
	Website        string     `json:"website"`
	GSTIN          string     `json:"gstin"`
	FacilityID     string     `json:"facilityId"`
	TimeFormat     string     `json:"timeFormat"`
	SystemLanguage string     `json:"systemLanguage"`
	TimeZone       string     `json:"timeZone"`
	DateFormat     string     `json:"dateFormat"`
	Currency       string     `json:"currency"`
	Timings        Timings    `json:"timings"`
	IsPrimary      bool       `json:"isPrimary"`
	IsArchived     bool       `json:"isArchived"`
}

type CreateClinicRequest struct {
	Name           string  `json:"name"`
	ClinicType     string  `json:"clinicType"`
	Location       string  `json:"location"`
	Color          string  `json:"color"`
	LogoURL        string  `json:"logoUrl"`
	Address        string  `json:"address"`
	Locality       string  `json:"locality"`
	PinCode        string  `json:"pinCode"`
	State          string  `json:"state"`
	Country        string  `json:"country"`
	Phones         []Phone `json:"phones"`
	Email          string  `json:"email"`
	Website        string  `json:"website"`
	GSTIN          string  `json:"gstin"`
	FacilityID     string  `json:"facilityId"`
	TimeFormat     string  `json:"timeFormat"`
	SystemLanguage string  `json:"systemLanguage"`
	TimeZone       string  `json:"timeZone"`
	DateFormat     string  `json:"dateFormat"`
	Currency       string  `json:"currency"`
	Timings        Timings `json:"timings"`
	MakePrimary    bool    `json:"makePrimary"`
}

type UpdateClinicRequest struct {
	Name           *string  `json:"name,omitempty"`
	ClinicType     *string  `json:"clinicType,omitempty"`
	Location       *string  `json:"location,omitempty"`
	Color          *string  `json:"color,omitempty"`
	LogoURL        *string  `json:"logoUrl,omitempty"`
	Address        *string  `json:"address,omitempty"`
	Locality       *string  `json:"locality,omitempty"`
	PinCode        *string  `json:"pinCode,omitempty"`
	State          *string  `json:"state,omitempty"`
	Country        *string  `json:"country,omitempty"`
	Phones         *[]Phone `json:"phones,omitempty"`
	Email          *string  `json:"email,omitempty"`
	Website        *string  `json:"website,omitempty"`
	GSTIN          *string  `json:"gstin,omitempty"`
	FacilityID     *string  `json:"facilityId,omitempty"`
	TimeFormat     *string  `json:"timeFormat,omitempty"`
	SystemLanguage *string  `json:"systemLanguage,omitempty"`
	TimeZone       *string  `json:"timeZone,omitempty"`
	DateFormat     *string  `json:"dateFormat,omitempty"`
	Currency       *string  `json:"currency,omitempty"`
	Timings        *Timings `json:"timings,omitempty"`
}

type JoinRequest struct {
	InviteCode string `json:"inviteCode"`
}
