package dto

import (
	"time"

	"github.com/Fazann/notedbook-backend/pkg/enums"
)

type PlanReq struct {
	Title       string                 `json:"title" binding:"required,min=2,max=255" example:"Save $500 for a new laptop"`
	Description string                 `json:"description" binding:"omitempty,max=2000" example:"I need a laptop for my Golang course"`
	AreaID      uint                   `json:"area_id" binding:"required,min=1" example:"1"`
	Priority    enums.PlanPriorityEnum `json:"priority" binding:"required,oneof=LOW MEDIUM HIGH" example:"HIGH"`
	// Plain date, YYYY-MM-DD; omit for no target date.
	TargetDate *string `json:"target_date" binding:"omitempty,datetime=2006-01-02" example:"2026-12-01"`
}

// PlanUpdateReq changes only the fields sent; a field left out (or null) keeps its value.
type PlanUpdateReq struct {
	Title       *string                 `json:"title" binding:"omitempty,min=2,max=255" example:"Save $500 for a new laptop"`
	Description *string                 `json:"description" binding:"omitempty,max=2000" example:"I need a laptop for my Golang course"`
	AreaID      *uint                   `json:"area_id" binding:"omitempty,min=1" example:"1"`
	Priority    *enums.PlanPriorityEnum `json:"priority" binding:"omitempty,oneof=LOW MEDIUM HIGH" example:"HIGH"`
	// "" clears the target date.
	TargetDate *string `json:"target_date" binding:"omitempty" example:"2026-12-01"`
}

// PlanStatusReq sets the plan's status directly; it does not check or uncheck steps.
type PlanStatusReq struct {
	Status enums.PlanStatusEnum `json:"status" binding:"required,oneof=NOT_START IN_PROGRESS DONE" example:"DONE"`
}

type PlanListReq struct {
	Page  int `form:"page" binding:"omitempty,min=1" example:"1"`
	Limit int `form:"limit" binding:"omitempty,min=1,max=1000" example:"10"`
	// One or more statuses, comma separated (NOT_START, IN_PROGRESS, DONE).
	Status string `form:"status" binding:"omitempty,max=50" example:"NOT_START,IN_PROGRESS"`
	// One or more plan area IDs, comma separated.
	AreaID string `form:"area_id" binding:"omitempty,max=100" example:"1,4"`
	// Matches the title.
	Q string `form:"q" binding:"omitempty,max=100" example:"laptop"`
	// Plans without a target date always come last when sorting by target_date.
	Sort string `form:"sort" binding:"omitempty,oneof=target_date -target_date created_at -created_at title -title progress -progress" example:"target_date"`
}

type PlanStatsReq struct {
	// The user's current date, YYYY-MM-DD (decides "overdue" and "this year"); defaults to today in UTC.
	Today string `form:"today" binding:"omitempty,datetime=2006-01-02" example:"2026-10-03"`
}

// PlanStatsRes summarizes the signed-in user's plans. Active plans are NOT_START and IN_PROGRESS.
type PlanStatsRes struct {
	Active int `json:"active" example:"5"`
	// DONE plans whose done_at is in the year of `today`.
	DoneThisYear int `json:"done_this_year" example:"3"`
	// Active plans with a target date before `today`.
	Overdue int `json:"overdue" example:"1"`
	// Average progress (0-100) of the active plans; 0 without active plans.
	AverageProgress int `json:"average_progress" example:"42"`
}

// AreaPlanRes is the plan area shown on a plan, with the title resolved in the request language.
type AreaPlanRes struct {
	ID    uint   `json:"id" example:"1"`
	Title string `json:"title" example:"Finance"`
}

type PlanRes struct {
	ID          uint                   `json:"id" example:"1"`
	Title       string                 `json:"title" example:"Save $500 for a new laptop"`
	Description string                 `json:"description" example:"I need a laptop for my Golang course"`
	Area        AreaPlanRes            `json:"area"`
	Priority    enums.PlanPriorityEnum `json:"priority" example:"HIGH"`
	Status      enums.PlanStatusEnum   `json:"status" example:"IN_PROGRESS"`
	// Plain date, YYYY-MM-DD; null when not set.
	TargetDate    *string `json:"target_date" example:"2026-12-01"`
	StepCount     int     `json:"step_count" example:"5"`
	DoneStepCount int     `json:"done_step_count" example:"2"`
	// 0-100, round(done_step_count / step_count * 100); 0 with no steps, 100 once status is DONE.
	Progress  int        `json:"progress" example:"40"`
	DoneAt    *time.Time `json:"done_at"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

// PlanDetailRes is a plan with its steps, by position.
type PlanDetailRes struct {
	PlanRes
	Steps []*PlanStepRes `json:"steps"`
}
