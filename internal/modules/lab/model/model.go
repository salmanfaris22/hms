package model

// One row of the lab worklist (3765:51814): a single test on an order, with
// enough of the order and the patient to draw the row without a second call.
type OrderTest struct {
	ID            string `json:"id"`
	OrderID       string `json:"orderId"`
	OrderNumber   string `json:"orderNumber"`
	OrderedAt     string `json:"orderedAt"`
	Priority      string `json:"priority"`
	OrderedBy     string `json:"orderedBy"`
	PatientID     string `json:"patientId"`
	PatientName   string `json:"patientName"`
	PatientNumber string `json:"patientNumber"`
	PatientPhoto  string `json:"patientPhotoUrl"`
	PatientAge    int    `json:"patientAge"`
	PatientSex    string `json:"patientSex"`
	Source        string `json:"source"`
	TestID        string `json:"testId"`
	TestName      string `json:"testName"`
	TestCode      string `json:"testCode"`
	Category      string `json:"category"`
	SampleType    string `json:"sampleType"`
	PriceCents    int    `json:"priceCents"`
	TaxPct        string `json:"taxPct"`
	Status        string `json:"status"`
	CollectedAt   string `json:"collectedAt"`
	CollectedBy   string `json:"collectedBy"`
	ResultNotes   string `json:"resultNotes"`
	ReportedAt    string `json:"reportedAt"`
	ReportedBy    string `json:"reportedBy"`
	ReportNumber  string `json:"reportNumber"`
	// '' until reported, then 'normal' | 'abnormal'
	ResultFlag string `json:"resultFlag"`
	// how many parameters have been entered; the rows themselves come with the
	// single-test read, not with the list
	ResultCount int `json:"resultCount"`
}

type OrderTestList struct {
	Tests    []OrderTest `json:"tests"`
	Total    int         `json:"total"`
	Page     int         `json:"page"`
	PageSize int         `json:"pageSize"`
}

// Summary is the four cards above the worklist.
type Summary struct {
	TodayCount      int `json:"todayCount"`
	PendingCount    int `json:"pendingCount"`
	InProgressCount int `json:"inProgressCount"`
	CompletedCount  int `json:"completedCount"`
}

// CreateOrderRequest — the New Test Order form (3765:51732).
type CreateOrderRequest struct {
	ClinicID        string           `json:"clinicId"`
	PatientID       string           `json:"patientId"`
	PatientAge      int              `json:"patientAge"`
	PatientSex      string           `json:"patientSex"`
	PatientCategory string           `json:"patientCategory"`
	Priority        string           `json:"priority"`
	OrderedBy       string           `json:"orderedBy"`
	OrderedByID     string           `json:"orderedById"`
	AppointmentID   string           `json:"appointmentId"`
	Notes           string           `json:"notes"`
	Tests           []OrderTestInput `json:"tests"`
}

// OrderTestInput is one test chosen from the catalogue.
type OrderTestInput struct {
	Source     string `json:"source"`
	TestID     string `json:"testId"`
	TestName   string `json:"testName"`
	TestCode   string `json:"testCode"`
	Category   string `json:"category"`
	SampleType string `json:"sampleType"`
	PriceCents int    `json:"priceCents"`
	TaxPct     string `json:"taxPct"`
}

type CreateOrderResult struct {
	ID     string `json:"id"`
	Number string `json:"number"`
	Tests  int    `json:"tests"`
}

// CollectRequest — Collect Sample (4565:70829).
type CollectRequest struct {
	ClinicID    string `json:"clinicId"`
	CollectedBy string `json:"collectedBy"`
}

// ResultValue is one parameter as it was measured (3911:56691).
type ResultValue struct {
	Parameter string `json:"parameter"`
	Value     string `json:"value"`
	Unit      string `json:"unit"`
	// the band it was read against, kept with the value so an old report still
	// says what "normal" meant on the day
	RefLow  string `json:"refLow"`
	RefHigh string `json:"refHigh"`
	Flag    string `json:"flag"`
}

// SaveResultRequest — Enter Test Result, in both its shapes: a panel sends
// values, a single test sends only its flag.
type SaveResultRequest struct {
	ClinicID   string        `json:"clinicId"`
	Values     []ResultValue `json:"values"`
	Remarks    string        `json:"remarks"`
	ReportedBy string        `json:"reportedBy"`
	// the overall flag for a test with no parameters of its own
	Flag string `json:"flag"`
}

// Report is one row of the Reports tab (3765:52465).
type Report struct {
	OrderTest
	// the values as reported, so a report opens without a second call
	Values []ResultValue `json:"values"`
}

type ReportList struct {
	Reports  []OrderTest `json:"reports"`
	Total    int         `json:"total"`
	Page     int         `json:"page"`
	PageSize int         `json:"pageSize"`
}
