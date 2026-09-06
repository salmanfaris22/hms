package model

type DispenseItemRequest struct {
	DrugID        string `json:"drugId"`
	DrugName      string `json:"drugName"`
	Unit          string `json:"unit"`
	QtyPrescribed int    `json:"qtyPrescribed"`
	QtyTotal      int    `json:"qtyTotal"`
}

type CreateDispenseRequest struct {
	PatientName  string                `json:"patientName"`
	IssuedToDept string                `json:"issuedToDept"`
	PrescribedBy string                `json:"prescribedBy"`
	IssuedDate   string                `json:"issuedDate"`
	Notes        string                `json:"notes"`
	Items        []DispenseItemRequest `json:"items"`
}

type DispenseDTO struct {
	ID           string `json:"id"`
	DispenseNo   string `json:"dispenseNo"`
	PatientName  string `json:"patientName"`
	IssuedToDept string `json:"issuedToDept"`
	PrescribedBy string `json:"prescribedBy"`
	IssuedDate   string `json:"issuedDate"`
	Status       string `json:"status"`
	ItemCount    int    `json:"itemCount"`
	CreatedAt    string `json:"createdAt"`
}

type DispenseSummary struct {
	TotalDispensed int `json:"totalDispensed"`
	TodayDispensed int `json:"todayDispensed"`
	Pending        int `json:"pending"`
	TotalUnitsOut  int `json:"totalUnitsOut"`
}

type DispenseListFilter struct {
	ClinicID     string
	Page         int
	Q            string
	StatusFilter string
}
