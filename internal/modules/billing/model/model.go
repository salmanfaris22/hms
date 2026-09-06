package model

// Invoice is one row of the billing list (3564:55048): who owes what, when it
// was raised and when it falls due.
type Invoice struct {
	ID            string `json:"id"`
	Number        string `json:"number"`
	PatientID     string `json:"patientId"`
	PatientName   string `json:"patientName"`
	PatientNumber string `json:"patientNumber"`
	PatientPhoto  string `json:"patientPhotoUrl"`
	IssuedAt      string `json:"issuedAt"`
	DueAt         string `json:"dueAt"`
	TotalCents    int    `json:"amountTotalCents"`
	PaidCents     int    `json:"amountPaidCents"`
	DueCents      int    `json:"amountDueCents"`
	// paid | pending | overdue | draft — overdue is derived, not stored
	Status string `json:"status"`
}

type InvoiceList struct {
	Invoices []Invoice `json:"invoices"`
	Total    int       `json:"total"`
	Page     int       `json:"page"`
	PageSize int       `json:"pageSize"`
}

// Summary feeds the four cards above the table.
type Summary struct {
	CollectedCents int `json:"collectedCents"`
	PaidCount      int `json:"paidCount"`
	PendingCents   int `json:"pendingCents"`
	PendingCount   int `json:"pendingCount"`
	OverdueCents   int `json:"overdueCents"`
	OverdueCount   int `json:"overdueCount"`
	DraftCount     int `json:"draftCount"`
	ProformaCount  int `json:"proformaCount"`
}

// Payment is one line of the Payment History tab.
type Payment struct {
	ID            string `json:"id"`
	InvoiceID     string `json:"invoiceId"`
	InvoiceNumber string `json:"invoiceNumber"`
	PatientName   string `json:"patientName"`
	AmountCents   int    `json:"amountCents"`
	Method        string `json:"method"`
	CollectedAt   string `json:"collectedAt"`
}

type PaymentList struct {
	Payments []Payment `json:"payments"`
	Total    int       `json:"total"`
	Page     int       `json:"page"`
	PageSize int       `json:"pageSize"`
}

// ── Creating an invoice (3564:58640) ────────────────────────────────────────

// InvoiceItemInput is one line as the form sends it. Money is never taken from
// the client: the server recomputes every figure from qty, rate and the two
// percentages, and the totals it stores are its own.
type InvoiceItemInput struct {
	// medicines | treatments | labTests | other
	Group       string  `json:"group"`
	Name        string  `json:"name"`
	Qty         float64 `json:"qty"`
	RateCents   int     `json:"rateCents"`
	DiscountPct float64 `json:"discountPct"`
	TaxPct      float64 `json:"taxPct"`
}

type PaymentInput struct {
	Method      string `json:"method"`
	AmountCents int    `json:"amountCents"`
}

type CreateInvoiceRequest struct {
	ClinicID string `json:"clinicId"`
	// one of the two identifies the patient; the number is used when the bill is
	// typed rather than raised from a visit
	PatientID     string `json:"patientId"`
	PatientNumber string `json:"patientNumber"`
	AppointmentID string `json:"appointmentId"`

	IssuedAt string `json:"issuedAt"`
	DueAt    string `json:"dueAt"`
	// pending | paid | draft | proforma
	Status string `json:"status"`

	Items []InvoiceItemInput `json:"items"`

	ExtraDiscountCents int    `json:"extraDiscountCents"`
	ExtraTaxCents      int    `json:"extraTaxCents"`
	RoundOff           string `json:"roundOff"`

	CollectedBy string `json:"collectedBy"`
	// money taken at the same time, if any
	Payments    []PaymentInput `json:"payments"`
	PaymentNote string         `json:"paymentNote"`
}

type CreateInvoiceResult struct {
	ID               string `json:"id"`
	Number           string `json:"number"`
	AmountTotalCents int    `json:"amountTotalCents"`
	AmountPaidCents  int    `json:"amountPaidCents"`
	AmountDueCents   int    `json:"amountDueCents"`
	Status           string `json:"status"`
}

// ── One invoice, opened (3889:55439 / 55599 / 55760) ────────────────────────

// InvoiceLine is a stored line, read back as it was billed.
type InvoiceLine struct {
	Group       string  `json:"group"`
	Name        string  `json:"name"`
	Qty         float64 `json:"qty"`
	RateCents   int     `json:"rateCents"`
	DiscountPct float64 `json:"discountPct"`
	TaxPct      float64 `json:"taxPct"`
	// the key the invoice was stored under, kept as-is so an old bill reads back
	AmountCents int `json:"amount"`
}

type InvoiceDetail struct {
	Invoice
	Lines    []InvoiceLine `json:"lines"`
	Payments []Payment     `json:"payments"`
	Refunds  []Refund      `json:"refunds"`
	// what has been handed back, so the card can state it separately from
	// what was taken
	RefundedCents int `json:"refundedCents"`
	// the figures the card shows, recomputed from the stored lines so an old
	// invoice reads exactly as it was billed
	SubtotalCents      int `json:"subtotalCents"`
	TotalDiscountCents int `json:"totalDiscountCents"`
	TaxableCents       int `json:"taxableCents"`
	TaxCents           int `json:"taxCents"`
	// applied on top of the lines by the form, and stored since 046
	ExtraDiscountCents int `json:"extraDiscountCents"`
	ExtraTaxCents      int `json:"extraTaxCents"`
	RoundOffCents      int `json:"roundOffCents"`
	// the rounding rule itself, so a reopened draft rounds the way it did
	RoundOffMode string `json:"roundOffMode"`
	// the visit this bill was raised from, if it was
	AppointmentID string `json:"appointmentId"`
}

// PaymentRequest takes money against an invoice that already exists
// (3614:77115, opened from the invoice card).
type PaymentRequest struct {
	ClinicID    string `json:"clinicId"`
	CollectedBy string `json:"collectedBy"`
	Note        string `json:"note"`
	Payments    []struct {
		Method      string `json:"method"`
		AmountCents int    `json:"amountCents"`
	} `json:"payments"`
}

// Refund is money handed back against an invoice (4565:73203).
type Refund struct {
	ID          string `json:"id"`
	InvoiceID   string `json:"invoiceId"`
	AmountCents int    `json:"amountCents"`
	Method      string `json:"method"`
	Note        string `json:"note"`
	CreatedAt   string `json:"createdAt"`
}

type RefundRequest struct {
	ClinicID    string `json:"clinicId"`
	AmountCents int    `json:"amountCents"`
	Method      string `json:"method"`
	Note        string `json:"note"`
	RefundedBy  string `json:"refundedBy"`
}
