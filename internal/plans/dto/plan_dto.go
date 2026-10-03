package dto

import (
	"time"

	"github.com/Fazann/notedbook-backend/pkg/enums"
)

type CreatePlanReq struct {
	UserID      uint                   `json:"-"`
	Title       string                 `json:"title" binding:"required,min=2max=255" example:"Complete Golang module lesson"`
	Description string                 `json:"description"`
	Area        string                 `json:"area" binding:"required"`
	Priority    enums.PlanPriorityEnum `json:"priority" binding:"required"`
	TargetDate  *time.Time             `json:"target_date"`
}

type CreatePlanRes struct {
	ID          uint                   `json:"id"`
	Title       string                 `json:"title"`
	Description string                 `json:"description"`
	Area        string                 `json:"area"`
	Priority    enums.PlanPriorityEnum `json:"priority"`
	TargetDate  *time.Time             `json:"target_date"`
	CreatedAt   time.Time              `json:"created_at"`
}
