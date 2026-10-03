package dto

import "time"

type PlanStepReq struct {
	Title string `json:"title" binding:"required,min=1,max=255" example:"Open a separate savings account"`
	// Plain date, YYYY-MM-DD; omit for no due date.
	DueDate *string `json:"due_date" binding:"omitempty,datetime=2006-01-02" example:"2026-10-31"`
	// Omit to add the step at the end; set it to put the step back where it was (e.g. undoing a delete).
	Position *float64 `json:"position" example:"1500"`
}

// PlanStepMoveReq moves a step inside its plan: the client sends the new position (between its new neighbours).
type PlanStepMoveReq struct {
	Position *float64 `json:"position" binding:"required" example:"1500"`
}

// PlanStepUpdateReq changes only the fields sent; a field left out (or null) keeps its value.
type PlanStepUpdateReq struct {
	Title *string `json:"title" binding:"omitempty,min=1,max=255" example:"Open a separate savings account"`
	// "" clears the due date.
	DueDate *string `json:"due_date" binding:"omitempty" example:"2026-10-31"`
}

type PlanStepToggleReq struct {
	Done bool `json:"done" example:"true"`
}

type PlanStepRes struct {
	ID     uint   `json:"id" example:"1"`
	PlanID uint   `json:"plan_id" example:"1"`
	Title  string `json:"title" example:"Open a separate savings account"`
	Done   bool   `json:"done" example:"false"`
	// Plain date, YYYY-MM-DD; null when not set.
	DueDate *string `json:"due_date" example:"2026-10-31"`
	// Steps are listed by position, ascending.
	Position  float64   `json:"position" example:"1000"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// PlanStepToggleRes is returned after a step is checked or unchecked, so the client can update both
// the step and the plan's counts/progress/status without a second request.
type PlanStepToggleRes struct {
	Step *PlanStepRes `json:"step"`
	Plan *PlanRes     `json:"plan"`
}

// PlanStepDeleteRes is returned after a step is deleted, with the plan's updated counts/progress.
type PlanStepDeleteRes struct {
	Plan *PlanRes `json:"plan"`
}
