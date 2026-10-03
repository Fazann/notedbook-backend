package dto

import "time"

type ExpenseReq struct {
	// Decimal string in the currency's major unit, at most its decimal places (2 for USD / MYR, 0 for KHR).
	Amount     string `json:"amount" binding:"required,max=20" example:"12.50"`
	Currency   string `json:"currency" binding:"required" example:"USD"`
	CategoryID uint   `json:"category_id" binding:"required,min=1" example:"1"`
	// Plain date, YYYY-MM-DD.
	SpentAt string `json:"spent_at" binding:"required,datetime=2006-01-02" example:"2026-09-29"`
	Note    string `json:"note" binding:"omitempty,max=200" example:"Lunch with team"`
}

// ExpenseUpdateReq changes only the fields sent; a field left out (or null) keeps its value.
type ExpenseUpdateReq struct {
	// Decimal string, as in ExpenseReq. Changing only currency keeps the same decimal value ("10.00" USD → "10" KHR).
	Amount     *string `json:"amount" binding:"omitempty,max=20" example:"12.50"`
	Currency   *string `json:"currency" example:"USD"`
	CategoryID *uint   `json:"category_id" binding:"omitempty,min=1" example:"1"`
	SpentAt    *string `json:"spent_at" binding:"omitempty,datetime=2006-01-02" example:"2026-09-29"`
	// "" clears the note.
	Note *string `json:"note" binding:"omitempty,max=200" example:"Lunch with team"`
}

type ExpenseListReq struct {
	Page  int `form:"page" binding:"omitempty,min=1" example:"1"`
	Limit int `form:"limit" binding:"omitempty,min=1,max=1000" example:"10"`
	// YYYY-MM; all months when empty.
	Month      string `form:"month" binding:"omitempty,datetime=2006-01" example:"2026-09"`
	CategoryID uint   `form:"category_id" binding:"omitempty,min=1" example:"1"`
	Currency   string `form:"currency" binding:"omitempty,oneof=USD KHR MYR" example:"USD"`
	// Matches the note.
	Q    string `form:"q" binding:"omitempty,max=100" example:"lunch"`
	Sort string `form:"sort" binding:"omitempty,oneof=spent_at -spent_at amount -amount" example:"-spent_at"`
}

type ExpenseAllReq struct {
	// YYYY-MM; all months when empty.
	Month string `form:"month" binding:"omitempty,datetime=2006-01" example:"2026-09"`
}

type ExpenseSummaryReq struct {
	// YYYY-MM
	Month string `form:"month" binding:"required,datetime=2006-01" example:"2026-09"`
}

// ExpenseSummaryRes holds a month's totals; amounts are never added across currencies.
type ExpenseSummaryRes struct {
	Month      string             `json:"month" example:"2026-09"`
	Totals     []CurrencyTotalRes `json:"totals"`
	ByCategory []CategoryTotalRes `json:"by_category"`
}

type CurrencyTotalRes struct {
	Currency string `json:"currency" example:"USD"`
	// Decimal string in the currency's major unit, as in ExpenseRes.
	Amount string `json:"amount" example:"125.50"`
	Count  int64  `json:"count" example:"12"`
}

type CategoryTotalRes struct {
	CategoryID uint   `json:"category_id" example:"1"`
	Currency   string `json:"currency" example:"USD"`
	Amount     string `json:"amount" example:"42.00"`
	Count      int64  `json:"count" example:"5"`
}

type ExpenseRes struct {
	ID uint `json:"id" example:"1"`
	// Decimal string in the currency's major unit, e.g. "12.50" USD, "40000" KHR.
	Amount     string `json:"amount" example:"12.50"`
	Currency   string `json:"currency" example:"USD"`
	CategoryID uint   `json:"category_id" example:"1"`
	// Plain date, YYYY-MM-DD.
	SpentAt   string    `json:"spent_at" example:"2026-09-29"`
	Note      string    `json:"note" example:"Lunch with team"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
