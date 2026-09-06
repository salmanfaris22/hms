package model

type TaxDTO struct {
	ID      string  `json:"id"`
	Name    string  `json:"name"`
	Rate    float64 `json:"rate"`
	Enabled bool    `json:"enabled"`
}

type CreateTaxRequest struct {
	Name string  `json:"name"`
	Rate float64 `json:"rate"`
}

type UpdateTaxRequest struct {
	Enabled *bool    `json:"enabled"`
	Name    *string  `json:"name"`
	Rate    *float64 `json:"rate"`
}

type PaymentModeDTO struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type CreatePaymentModeRequest struct {
	Name string `json:"name"`
}
