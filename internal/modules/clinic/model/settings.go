package model

type NumberingEntry struct {
	Enabled   bool   `json:"enabled"`
	Prefix    string `json:"prefix"`
	StartFrom int    `json:"startFrom"`
	Digits    int    `json:"digits"`
}

type NumberingSettings struct {
	PatientID       NumberingEntry `json:"patientId"`
	OpID            NumberingEntry `json:"opId"`
	InvoiceNo       NumberingEntry `json:"invoiceNo"`
	OpdBillNo       NumberingEntry `json:"opdBillNo"`
	PurchaseID      NumberingEntry `json:"purchaseId"`
	StaffID         NumberingEntry `json:"staffId"`
	PharmacyBillNo  NumberingEntry `json:"pharmacyBillNo"`
	PathologyBillNo NumberingEntry `json:"pathologyBillNo"`
	PathologyTestID NumberingEntry `json:"pathologyTestId"`
}

func DefaultNumbering() NumberingSettings {
	d := NumberingEntry{Enabled: true, Prefix: "PAT", StartFrom: 1001, Digits: 4}
	return NumberingSettings{
		PatientID: d, OpID: d, InvoiceNo: d, OpdBillNo: d,
		PurchaseID: d, StaffID: d, PharmacyBillNo: d,
		PathologyBillNo: d, PathologyTestID: d,
	}
}
