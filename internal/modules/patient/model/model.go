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

type InvoiceTotals struct {
	Total int `json:"total"`
	Paid  int `json:"paid"`
	Due   int `json:"due"`
	Count int `json:"count"`
}
