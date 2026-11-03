package dto

// CurrencySettingReq replaces the user's currency settings.
type CurrencySettingReq struct {
	// Currencies the user records expenses in; at least one. Duplicates are ignored.
	EnabledCurrencies []string `json:"enabled_currencies" binding:"required,max=10" example:"USD,KHR"`
	// One of enabled_currencies.
	DefaultCurrency string `json:"default_currency" binding:"required" example:"USD"`
}

type CurrencySettingRes struct {
	// In the fixed order USD, KHR, MYR. Every currency when the user never saved settings.
	EnabledCurrencies []string `json:"enabled_currencies" example:"USD,KHR"`
	DefaultCurrency   string   `json:"default_currency" example:"USD"`
}
