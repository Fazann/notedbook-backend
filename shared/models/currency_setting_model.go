package model

import "time"

// CurrencySetting holds which currencies a user records expenses in and which one new expenses start in.
// A user without a row uses every currency, with USD as the default.
type CurrencySetting struct {
	BaseModelID
	UserID uint `gorm:"type:bigint;not null;uniqueIndex"`
	// Comma-separated codes in the order of enums.CurrencyValues, e.g. "USD,KHR". Never empty.
	EnabledCurrencies string `gorm:"type:varchar(64);not null"`
	// One of EnabledCurrencies.
	DefaultCurrency string    `gorm:"type:varchar(8);not null"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}
