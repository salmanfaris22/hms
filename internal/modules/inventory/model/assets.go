package model

type AssetDTO struct {
	ID             string  `json:"id"`
	AssetNo        string  `json:"assetNo"`
	Name           string  `json:"name"`
	Manufacturer   string  `json:"manufacturer"`
	SerialNumber   string  `json:"serialNumber"`
	Model          string  `json:"model"`
	Location       string  `json:"location"`
	PurchaseDate   *string `json:"purchaseDate"`
	CurrentValue   float64 `json:"currentValue"`
	PurchaseCost   float64 `json:"purchaseCost"`
	WarrantyExpiry *string `json:"warrantyExpiry"`
	Category       string  `json:"category"`
	Department     string  `json:"department"`
	Condition      string  `json:"condition"`
	Status         string  `json:"status"`
	Notes          string  `json:"notes"`
	CreatedAt      string  `json:"createdAt"`
}

type AssetSummary struct {
	TotalAssets    int     `json:"totalAssets"`
	ActiveAssets   int     `json:"activeAssets"`
	Maintenance    int     `json:"maintenance"`
	Decommissioned int     `json:"decommissioned"`
	TotalBookValue float64 `json:"totalBookValue"`
}

type CreateAssetRequest struct {
	Name           string  `json:"name"`
	Manufacturer   string  `json:"manufacturer"`
	SerialNumber   string  `json:"serialNumber"`
	Model          string  `json:"model"`
	Location       string  `json:"location"`
	PurchaseDate   *string `json:"purchaseDate"`
	CurrentValue   float64 `json:"currentValue"`
	PurchaseCost   float64 `json:"purchaseCost"`
	WarrantyExpiry *string `json:"warrantyExpiry"`
	Category       string  `json:"category"`
	Department     string  `json:"department"`
	Condition      string  `json:"condition"`
	Status         string  `json:"status"`
	Notes          string  `json:"notes"`
}

type AssetListFilter struct {
	ClinicID     string
	Page         int
	Q            string
	StatusFilter string
	CondFilter   string
	CatFilter    string
}
