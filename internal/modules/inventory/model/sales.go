package model

type SaleItemRequest struct {
	DrugID     string  `json:"drugId"`
	DrugName   string  `json:"drugName"`
	BatchCode  string  `json:"batchCode"`
	ExpiryDate *string `json:"expiryDate"`
	Unit       string  `json:"unit"`
	Qty        int     `json:"qty"`
	UnitPrice  float64 `json:"unitPrice"`
	Discount   float64 `json:"discount"`
	Tax        float64 `json:"tax"`
	Total      float64 `json:"total"`
}

type CreateSaleRequest struct {
	ClinicID      string            `json:"clinicId"`
	CustomerName  string            `json:"customerName"`
	OpdID         string            `json:"opdId"`
	PrescribedBy  string            `json:"prescribedBy"`
	StockPoint    string            `json:"stockPoint"`
	PaymentMethod string            `json:"paymentMethod"`
	Subtotal      float64           `json:"subtotal"`
	DiscountPct   float64           `json:"discountPct"`
	GstEnabled    bool              `json:"gstEnabled"`
	RoundOff      float64           `json:"roundOff"`
	Shipping      float64           `json:"shipping"`
	GrandTotal    float64           `json:"grandTotal"`
	Notes         string            `json:"notes"`
	Items         []SaleItemRequest `json:"items"`
}

type SaleDTO struct {
	ID            string  `json:"id"`
	InvoiceNo     string  `json:"invoiceNo"`
	CustomerName  string  `json:"customerName"`
	OpdID         string  `json:"opdId"`
	PrescribedBy  string  `json:"prescribedBy"`
	StockPoint    string  `json:"stockPoint"`
	PaymentMethod string  `json:"paymentMethod"`
	GrandTotal    float64 `json:"grandTotal"`
	Status        string  `json:"status"`
	ItemCount     int     `json:"itemCount"`
	CreatedAt     string  `json:"createdAt"`
}

type SaleSummary struct {
	TodaySales     float64 `json:"todaySales"`
	TodayGrowth    float64 `json:"todayGrowth"`
	WeeklyRevenue  float64 `json:"weeklyRevenue"`
	WeeklyGrowth   float64 `json:"weeklyGrowth"`
	MonthlyRevenue float64 `json:"monthlyRevenue"`
	MonthlyGrowth  float64 `json:"monthlyGrowth"`
	TotalTxns      int     `json:"totalTxns"`
}

type SaleListFilter struct {
	ClinicID     string
	Page         int
	Q            string
	StatusFilter string
}
