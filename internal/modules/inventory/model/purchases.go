package model

type PurchaseItemRequest struct {
	DrugID       string  `json:"drugId"`
	DrugName     string  `json:"drugName"`
	BatchCode    string  `json:"batchCode"`
	ExpiryDate   *string `json:"expiryDate"`
	QtyPurchased int     `json:"qtyPurchased"`
	QtyTotal     int     `json:"qtyTotal"`
	PurchaseRate float64 `json:"purchaseRate"`
	GstPct       float64 `json:"gstPct"`
	FreeQty      int     `json:"freeQty"`
	MRP          float64 `json:"mrp"`
	PackSize     int     `json:"packSize"`
	DiscountPct  float64 `json:"discountPct"`
	Amount       float64 `json:"amount"`
}

type CreatePurchaseRequest struct {
	InvoiceNo     string                `json:"invoiceNo"`
	SupplierName  string                `json:"supplierName"`
	StockPoint    string                `json:"stockPoint"`
	PaymentMethod string                `json:"paymentMethod"`
	Subtotal      float64               `json:"subtotal"`
	DiscountPct   float64               `json:"discountPct"`
	GstEnabled    bool                  `json:"gstEnabled"`
	RoundOff      float64               `json:"roundOff"`
	Shipping      float64               `json:"shipping"`
	GrandTotal    float64               `json:"grandTotal"`
	Notes         string                `json:"notes"`
	Items         []PurchaseItemRequest `json:"items"`
}

type PurchaseDTO struct {
	ID            string  `json:"id"`
	PurchaseID    string  `json:"purchaseId"`
	InvoiceNo     string  `json:"invoiceNo"`
	SupplierName  string  `json:"supplierName"`
	StockPoint    string  `json:"stockPoint"`
	PaymentMethod string  `json:"paymentMethod"`
	GrandTotal    float64 `json:"grandTotal"`
	ItemCount     int     `json:"itemCount"`
	CreatedAt     string  `json:"createdAt"`
}

type PurchaseListFilter struct {
	ClinicID string
	Page     int
	Q        string
}
