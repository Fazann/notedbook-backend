package dto

import "time"

type ExchangeRateListReq struct {
	// ISO 4217 code the rates are relative to; USD when empty.
	Base string `form:"base" binding:"omitempty,len=3,alpha,uppercase" example:"USD"`
}

// ExchangeRateRes is the rate of every supported currency against base: 1 base = rates[code] code.
type ExchangeRateRes struct {
	Base  string             `json:"base" example:"USD"`
	Rates map[string]float64 `json:"rates" swaggertype:"object,number" example:"KHR:4094,MYR:4.06,USD:1"`
	// When the source last published the rates, and when it publishes the next ones (RFC 3339).
	UpdatedAt    time.Time `json:"updated_at" example:"2026-11-08T00:02:31Z"`
	NextUpdateAt time.Time `json:"next_update_at" example:"2026-11-09T00:14:21Z"`
	// Name and link of the source, which its terms ask to show next to the rates.
	Provider    string `json:"provider" example:"ExchangeRate-API"`
	ProviderURL string `json:"provider_url" example:"https://www.exchangerate-api.com"`
}
