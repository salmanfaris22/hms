package model

import "time"

type RadiologyTestDTO struct {
	ID          string    `json:"id"`
	ClinicID    string    `json:"clinicId"`
	TestName    string    `json:"testName"`
	TestCode    string    `json:"testCode"`
	Category    string    `json:"category"`
	BodyPart    string    `json:"bodyPart"`
	Description string    `json:"description"`
	Price       float64   `json:"price"`
	Tax         string    `json:"tax"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type CreateRadiologyTestRequest struct {
	ClinicID    string  `json:"clinicId"`
	TestName    string  `json:"testName"`
	TestCode    string  `json:"testCode"`
	Category    string  `json:"category"`
	BodyPart    string  `json:"bodyPart"`
	Description string  `json:"description"`
	Price       float64 `json:"price"`
	Tax         string  `json:"tax"`
}

type RadiologyListFilter struct {
	ClinicID string
	Page     int
	PageSize int
	Search   string
	Category string
}
