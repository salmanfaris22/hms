package model

import "time"

type TreatmentConsumable struct {
	DrugID         string  `json:"drugId"`
	Name           string  `json:"name"`
	Quantity       float64 `json:"quantity"`
	DispenseMethod string  `json:"dispenseMethod"`
}

type TreatmentDTO struct {
	ID            string                `json:"id"`
	ClinicID      string                `json:"clinicId"`
	TreatmentName string                `json:"treatmentName"`
	TreatmentCode string                `json:"treatmentCode"`
	Description   string                `json:"description"`
	Price         float64               `json:"price"`
	Discount      float64               `json:"discount"`
	Tax           string                `json:"tax"`
	Consumables   []TreatmentConsumable `json:"consumables"`
	CreatedAt     time.Time             `json:"createdAt"`
	UpdatedAt     time.Time             `json:"updatedAt"`
}

type CreateTreatmentRequest struct {
	ClinicID      string                `json:"clinicId"`
	TreatmentName string                `json:"treatmentName"`
	TreatmentCode string                `json:"treatmentCode"`
	Description   string                `json:"description"`
	Price         float64               `json:"price"`
	Discount      float64               `json:"discount"`
	Tax           string                `json:"tax"`
	Consumables   []TreatmentConsumable `json:"consumables"`
}

type TreatmentListFilter struct {
	ClinicID string
	Page     int
	PageSize int
	Search   string
	Sort     string
}
