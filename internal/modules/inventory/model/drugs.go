package model

type DrugDTO struct {
	ID             string  `json:"id"`
	Name           string  `json:"name"`
	GenericName    string  `json:"genericName"`
	CategoryID     *string `json:"categoryId"`
	CategoryName   string  `json:"categoryName"`
	Strength       string  `json:"strength"`
	ItemCode       string  `json:"itemCode"`
	ManufacturerID *string `json:"manufacturerId"`
	Manufacturer   string  `json:"manufacturer"`
	Instruction    string  `json:"instruction"`
	StockLevel     int     `json:"stockLevel"`
	Status         string  `json:"status"`
	Price          float64 `json:"price"`
	ExpiryDate     *string `json:"expiryDate"`
}

type StockSummary struct {
	TotalItems   int     `json:"totalItems"`
	TotalValue   float64 `json:"totalValue"`
	LowStock     int     `json:"lowStock"`
	ExpiringSoon int     `json:"expiringSoon"`
	Categories   int     `json:"categories"`
}

type DrugListFilter struct {
	ClinicID     string
	Page         int
	Q            string
	CategoryID   string
	StatusFilter string
	Sort         string
}

type CreateDrugRequest struct {
	Name           string  `json:"name"`
	GenericName    string  `json:"genericName"`
	CategoryID     *string `json:"categoryId"`
	Strength       string  `json:"strength"`
	ItemCode       string  `json:"itemCode"`
	ManufacturerID *string `json:"manufacturerId"`
	Instruction    string  `json:"instruction"`
}

type CategoryDTO struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type ManufacturerDTO struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type StockDTO struct {
	ID           string  `json:"id"`
	DrugID       string  `json:"drugId"`
	BatchCode    string  `json:"batchCode"`
	ExpiryDate   *string `json:"expiryDate"`
	QtyAvailable int     `json:"qtyAvailable"`
	PurchaseRate float64 `json:"purchaseRate"`
	MRP          float64 `json:"mrp"`
}

type CreateStockRequest struct {
	DrugID       string  `json:"drugId"`
	BatchCode    string  `json:"batchCode"`
	ExpiryDate   *string `json:"expiryDate"`
	QtyAvailable int     `json:"qtyAvailable"`
	PurchaseRate float64 `json:"purchaseRate"`
	MRP          float64 `json:"mrp"`
}

type POSItem struct {
	ID           string  `json:"id"`
	Name         string  `json:"name"`
	Manufacturer string  `json:"manufacturer"`
	CategoryName string  `json:"categoryName"`
	Price        float64 `json:"price"`
	Stock        int     `json:"stock"`
	BatchCode    string  `json:"batchCode"`
	ExpiryDate   *string `json:"expiryDate"`
}
