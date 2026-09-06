package model

type PatientCategoryDTO struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	PatientCount int    `json:"patientCount"`
}

type SpecialStatusDTO struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Color        string `json:"color"`
	PatientCount int    `json:"patientCount"`
}

type CreateSpecialStatusRequest struct {
	Name  string `json:"name"`
	Color string `json:"color"`
}

type UpdateSpecialStatusRequest struct {
	Name  *string `json:"name"`
	Color *string `json:"color"`
}

type PCListFilter struct {
	ClinicID string
	Page     int
	Limit    int
	Search   string
}
