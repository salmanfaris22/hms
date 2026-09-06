package model

import "time"

type PatientPhone struct {
	Type        string `json:"type"`
	CountryCode string `json:"countryCode"`
	Number      string `json:"number"`
}

type FamilyMember struct {
	PatientID string `json:"patientId,omitempty"`
	Name      string `json:"name"`
	Relation  string `json:"relation"`
}

type PatientDTO struct {
	ID                string         `json:"id"`
	ClinicID          string         `json:"clinicId"`
	PatientNumber     string         `json:"patientNumber"`
	FirstName         string         `json:"firstName"`
	MiddleName        string         `json:"middleName"`
	LastName          string         `json:"lastName"`
	DateOfBirth       *time.Time     `json:"dateOfBirth,omitempty"`
	Sex               string         `json:"sex"`
	BloodType         string         `json:"bloodType"`
	MaritalStatus     string         `json:"maritalStatus"`
	PhotoURL          string         `json:"photoUrl"`
	Phones            []PatientPhone `json:"phones"`
	Email             string         `json:"email"`
	AddressLine       string         `json:"addressLine"`
	City              string         `json:"city"`
	State             string         `json:"state"`
	Country           string         `json:"country"`
	ZipCode           string         `json:"zipCode"`
	EmergencyName     string         `json:"emergencyName"`
	EmergencyPhone    string         `json:"emergencyPhone"`
	Allergies         string         `json:"allergies"`
	MedicalConditions string         `json:"medicalConditions"`
	InsuranceProvider string         `json:"insuranceProvider"`
	InsuranceID       string         `json:"insuranceId"`
	Condition         string         `json:"condition"`
	LastVisitAt       *time.Time     `json:"lastVisitAt,omitempty"`
	Status            string         `json:"status"`
	Notes             string         `json:"notes"`
	// Patient group and special status, joined from their catalogues.
	Source            string         `json:"source"`
	InsuranceValidity *time.Time     `json:"insuranceValidity,omitempty"`
	CategoryID        *string        `json:"categoryId,omitempty"`
	SpecialStatusID   *string        `json:"specialStatusId,omitempty"`
	CategoryName      string         `json:"categoryName"`
	SpecialStatusName string         `json:"specialStatusName"`
	FamilyMembers     []FamilyMember `json:"familyMembers"`
	CreatedAt         time.Time      `json:"createdAt"`
	UpdatedAt         time.Time      `json:"updatedAt"`
}

type Stats struct {
	Total        int `json:"total"`
	Active       int `json:"active"`
	Inactive     int `json:"inactive"`
	NewThisMonth int `json:"newThisMonth"`
}

type ListResponse struct {
	Patients []PatientDTO `json:"patients"`
	Stats    Stats        `json:"stats"`
	Page     int          `json:"page"`
	PageSize int          `json:"pageSize"`
	Total    int          `json:"total"`
}

type ListFilter struct {
	TenantID string
	UserID   string
	ClinicID string
	Page     int
	PageSize int
	Search   string
	Status   string
}

type CreateRequest struct {
	ClinicID          string         `json:"clinicId"`
	FirstName         string         `json:"firstName"`
	MiddleName        string         `json:"middleName"`
	LastName          string         `json:"lastName"`
	DateOfBirth       string         `json:"dateOfBirth"`
	Sex               string         `json:"sex"`
	BloodType         string         `json:"bloodType"`
	MaritalStatus     string         `json:"maritalStatus"`
	PhotoURL          string         `json:"photoUrl"`
	Phones            []PatientPhone `json:"phones"`
	Email             string         `json:"email"`
	AddressLine       string         `json:"addressLine"`
	City              string         `json:"city"`
	State             string         `json:"state"`
	Country           string         `json:"country"`
	ZipCode           string         `json:"zipCode"`
	EmergencyName     string         `json:"emergencyName"`
	EmergencyPhone    string         `json:"emergencyPhone"`
	Allergies         string         `json:"allergies"`
	MedicalConditions string         `json:"medicalConditions"`
	InsuranceProvider string         `json:"insuranceProvider"`
	InsuranceID       string         `json:"insuranceId"`
	Condition         string         `json:"condition"`
	Notes             string         `json:"notes"`
	FamilyMembers     []FamilyMember `json:"familyMembers"`
	Source            string         `json:"source"`
	InsuranceValidity string         `json:"insuranceValidity"`
	// set only when the form's UHID radio is on "Manual Entry"; blank means
	// the server allocates the next PA###### itself
	PatientNumber   string `json:"patientNumber"`
	CategoryID      string `json:"categoryId"`
	SpecialStatusID string `json:"specialStatusId"`
}

type AddAlertRequest struct {
	Name     string `json:"name"`
	Severity string `json:"severity"`
}

type AddAllergyRequest struct {
	Name     string `json:"name"`
	Severity string `json:"severity"`
	Reaction string `json:"reaction"`
}

// CollectPaymentRequest is one collection against a pending invoice
// (4723:64491): how much was taken, and how.
type CollectPaymentRequest struct {
	ClinicID    string `json:"clinicId"`
	AmountCents int    `json:"amountCents"`
	Method      string `json:"method"`
}

// CollectPaymentResult is the invoice as it stands after the money landed.
type CollectPaymentResult struct {
	InvoiceID  string `json:"invoiceId"`
	PaidCents  int    `json:"amountPaidCents"`
	DueCents   int    `json:"amountDueCents"`
	Status     string `json:"status"`
}

type InvoiceTotals struct {
	Total int `json:"total"`
	Paid  int `json:"paid"`
	Due   int `json:"due"`
	Count int `json:"count"`
}

// AddVitalsRequest — one batch from the Add Vitals modal (2862:39200).
type AddVitalsRequest struct {
	RecordedAt string        `json:"recordedAt"`
	Notes      string        `json:"notes"`
	Readings   []VitalIntake `json:"readings"`
}

type VitalIntake struct {
	Category       string `json:"category"`
	Kind           string `json:"kind"`
	ValueText      string `json:"valueText"`
	Unit           string `json:"unit"`
	Status         string `json:"status"`
	ReferenceRange string `json:"referenceRange"`
}

// AddMedicationRequest — one row from the Add Medication form (3820:66132).
type AddMedicationRequest struct {
	Name      string `json:"name"`
	Dose      string `json:"dose"`
	Schedule  string `json:"schedule"`
	Purpose   string `json:"purpose"`
	Doctor    string `json:"doctor"`
	StartedAt string `json:"startedAt"`
	Status    string `json:"status"`
}

// PatientDocument — one row of the Documents tab (3151:46605).
type PatientDocument struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	URL          string `json:"url"`
	MimeType     string `json:"mimeType"`
	SizeBytes    int64  `json:"sizeBytes"`
	Category     string `json:"category"`
	Doctor       string `json:"doctor"`
	DocumentDate string `json:"documentDate"`
	CreatedAt    string `json:"createdAt"`
}

// DocumentCategory is one clinic-defined document category ("Create New
// Category", 1336:20803). The built-in six live in the frontend.
type DocumentCategory struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type AddDocumentRequest struct {
	Name         string `json:"name"`
	URL          string `json:"url"`
	MimeType     string `json:"mimeType"`
	SizeBytes    int64  `json:"sizeBytes"`
	Category     string `json:"category"`
	Doctor       string `json:"doctor"`
	DocumentDate string `json:"documentDate"`
}

// FamilyLink is one link from "Add Family Member" (3841:57713), resolved to the
// relative's own record so the list can name them. Distinct from the patient's
// free-text FamilyMember list above, which the Add Patient form collects as next
// of kin and need not be a patient of this clinic.
type FamilyLink struct {
	ID            string `json:"id"`
	RelativeID    string `json:"relativeId"`
	Name          string `json:"name"`
	PatientNumber string `json:"patientNumber"`
	Relation      string `json:"relation"`
}

type AddFamilyLinkRequest struct {
	RelativeID string `json:"relativeId"`
	Relation   string `json:"relation"`
}
