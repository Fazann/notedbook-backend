package model

import (
	"time"

	"github.com/Fazann/notedbook-backend/pkg/enums"
)

type Expense struct {
	BaseModelID
	UserID     uint `gorm:"type:bigint;not null;index:idx_expense_user_spent_at"`
	CategoryID uint `gorm:"type:bigint;not null;index"`
	// In minor units of Currency: USD cents, KHR riel, MYR sen.
	Amount   int64              `gorm:"type:bigint;not null"`
	Currency enums.CurrencyEnum `gorm:"type:varchar(8);not null"`
	// Plain date, no time of day.
	SpentAt time.Time `gorm:"type:date;not null;index:idx_expense_user_spent_at"`
	Note    string    `gorm:"type:varchar(200);not null;default:''"`
	BaseModelGorm
}
