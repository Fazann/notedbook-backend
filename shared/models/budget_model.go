package model

import "github.com/Fazann/notedbook-backend/pkg/enums"

// Budget is a monthly limit on a user's total spending that repeats every month.
// Amounts are never added across currencies, so a user has at most one per currency.
type Budget struct {
	BaseModelID
	UserID   uint               `gorm:"type:bigint;not null;uniqueIndex:idx_budget_user_currency,where:deleted_at IS NULL"`
	Currency enums.CurrencyEnum `gorm:"type:varchar(8);not null;uniqueIndex:idx_budget_user_currency,where:deleted_at IS NULL"`
	// In minor units of Currency: USD cents, KHR riel, MYR sen.
	Amount int64 `gorm:"type:bigint;not null"`
	// Percent of Amount at which the budget counts as near its limit.
	AlertPercent int `gorm:"type:smallint;not null;default:80"`
	BaseModelGorm
}
