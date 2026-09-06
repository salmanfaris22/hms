package model

import "time"

type DrugDTO struct {
	ID            string    `json:"id"`
	ClinicID      string    `json:"clinicId"`
	DrugName      string    `json:"drugName"`
	GenericName   string    `json:"genericName"`
	Category      string    `json:"category"`
	Strength      string    `json:"strength"`
	ItemCode      string    `json:"itemCode"`
	Manufacturer  string    `json:"manufacturer"`
	Instruction   string    `json:"instruction"`
	PrimaryUnit   string    `json:"primaryUnit"`
	SecondaryUnit string    `json:"secondaryUnit"`
	ReorderLevel  string    `json:"reorderLevel"`
	HSNCode       string    `json:"hsnCode"`
	Tax           string    `json:"tax"`
	Discount      string    `json:"discount"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

type CreateDrugRequest struct {
	ClinicID      string `json:"clinicId"`
	DrugName      string `json:"drugName"`
	GenericName   string `json:"genericName"`
	Category      string `json:"category"`
	Strength      string `json:"strength"`
	ItemCode      string `json:"itemCode"`
	Manufacturer  string `json:"manufacturer"`
	Instruction   string `json:"instruction"`
	PrimaryUnit   string `json:"primaryUnit"`
	SecondaryUnit string `json:"secondaryUnit"`
	ReorderLevel  string `json:"reorderLevel"`
	HSNCode       string `json:"hsnCode"`
	Tax           string `json:"tax"`
	Discount      string `json:"discount"`
}

type DrugListFilter struct {
	ClinicID string
	Page     int
	PageSize int
	Search   string
	Category string
}

type LookupDTO struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type LookupCreateRequest struct {
	ClinicID string `json:"clinicId"`
	Name     string `json:"name"`
}
