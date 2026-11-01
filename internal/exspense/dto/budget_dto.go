package dto

import "time"

// BudgetReq is the whole budget; an update replaces every field.
type BudgetReq struct {
	// Decimal string in the currency's major unit, at most its decimal places (2 for USD / MYR, 0 for KHR).
	Amount   string `json:"amount" binding:"required,max=20" example:"200.00"`
	Currency string `json:"currency" binding:"required" example:"USD"`
	// Percent of amount at which the budget counts as near its limit.
	AlertPercent int `json:"alert_percent" binding:"required,min=1,max=100" example:"80"`
}

type BudgetListReq struct {
	// YYYY-MM: the month whose spending is compared with the budgets.
	Month string `form:"month" binding:"required,datetime=2006-01" example:"2026-11"`
}

type BudgetRes struct {
	ID uint `json:"id" example:"1"`
	// Decimal string in the currency's major unit, as in ExpenseRes.
	Amount       string    `json:"amount" example:"200.00"`
	Currency     string    `json:"currency" example:"USD"`
	AlertPercent int       `json:"alert_percent" example:"80"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// BudgetStatusRes is a budget compared with one month's spending.
type BudgetStatusRes struct {
	BudgetRes
	// Total spent in the month in the budget's currency, as a decimal string.
	Spent string `json:"spent" example:"170.00"`
	// Share of amount spent, rounded down; above 100 when over budget.
	Percent int64 `json:"percent" example:"85"`
	// ok, near (reached alert_percent) or over (spent more than amount).
	Status string `json:"status" example:"near" enums:"ok,near,over"`
}
