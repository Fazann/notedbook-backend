package dto

import (
	"time"

	"github.com/Fazann/notedbook-backend/pkg/enums"
)

type RecurrenceReq struct {
	Kind enums.RecurrenceKindEnum `json:"kind" binding:"required,oneof=NONE DAILY WEEKDAYS WEEKLY" example:"WEEKLY"`
	// ISO weekday, 1 (Monday) to 7 (Sunday); required when kind is WEEKLY, ignored otherwise.
	Days []int `json:"days" binding:"omitempty,dive,min=1,max=7" example:"2,4"`
	// Plain date, YYYY-MM-DD; inclusive end of the series. Omit for no end date. Ignored when kind is NONE.
	Until *string `json:"until" binding:"omitempty,datetime=2006-01-02" example:"2026-12-31"`
}

type ScheduleActivityReq struct {
	Title string                         `json:"title" binding:"required,min=1,max=100" example:"Team meeting"`
	Type  enums.ScheduleActivityTypeEnum `json:"type" binding:"required,oneof=MEETING LEARNING WORK EXERCISE PERSONAL FAMILY OTHER" example:"MEETING"`
	// Plain date, YYYY-MM-DD; the first (or only) day.
	Date string `json:"date" binding:"required,datetime=2006-01-02" example:"2026-09-30"`
	// HH:mm, 24h.
	StartTime  string        `json:"start_time" binding:"required,datetime=15:04" example:"10:10"`
	EndTime    string        `json:"end_time" binding:"required,datetime=15:04" example:"10:40"`
	Location   string        `json:"location" binding:"omitempty,max=100" example:"Zoom"`
	Note       string        `json:"note" binding:"omitempty,max=500" example:"Bring the roadmap doc"`
	Recurrence RecurrenceReq `json:"recurrence" binding:"required"`
}

// ScheduleActivityScopeReq is bound from the query string on an update or delete; it is required
// because editing or deleting a repeating activity must say whether it applies to this occurrence
// only or to the whole series.
type ScheduleActivityScopeReq struct {
	Scope string `form:"scope" binding:"required,oneof=all this" example:"all"`
	// Required when scope is "this": which occurrence is being changed. Plain date, YYYY-MM-DD.
	Date string `form:"date" binding:"omitempty,datetime=2006-01-02" example:"2026-09-30"`
}

type OccurrenceListReq struct {
	// Inclusive dates, YYYY-MM-DD; the range cannot be more than 42 days.
	From string `form:"from" binding:"required,datetime=2006-01-02" example:"2026-09-28"`
	To   string `form:"to" binding:"required,datetime=2006-01-02" example:"2026-10-04"`
}

type RecurrenceRes struct {
	Kind enums.RecurrenceKindEnum `json:"kind" example:"WEEKLY"`
	// Only set when kind is WEEKLY.
	Days  []int   `json:"days,omitempty" example:"2,4"`
	Until *string `json:"until" example:"2026-12-31"`
}

type ScheduleActivityRes struct {
	ID         uint                           `json:"id" example:"1"`
	Title      string                         `json:"title" example:"Team meeting"`
	Type       enums.ScheduleActivityTypeEnum `json:"type" example:"MEETING"`
	Date       string                         `json:"date" example:"2026-09-30"`
	StartTime  string                         `json:"start_time" example:"10:10"`
	EndTime    string                         `json:"end_time" example:"10:40"`
	Location   string                         `json:"location" example:"Zoom"`
	Note       string                         `json:"note" example:"Bring the roadmap doc"`
	Recurrence RecurrenceRes                  `json:"recurrence"`
	// Dates (YYYY-MM-DD) removed from the series by a "this occurrence only" edit or delete.
	Exceptions []string  `json:"exceptions"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// ScheduleActivityUpdateRes is returned from a scope=this edit: the series (with the new exception
// date added) and the new standalone activity created for that one occurrence.
type ScheduleActivityUpdateRes struct {
	Series  *ScheduleActivityRes `json:"series"`
	Created *ScheduleActivityRes `json:"created"`
}

// OccurrenceRes is one concrete appearance of an activity on the calendar; a repeating activity
// expands into one of these per matching day in the requested range.
type OccurrenceRes struct {
	// `{activity_id}:{date}`, stable across requests.
	Key         string                         `json:"key" example:"1:2026-09-30"`
	ActivityID  uint                           `json:"activity_id" example:"1"`
	Date        string                         `json:"date" example:"2026-09-30"`
	Title       string                         `json:"title" example:"Team meeting"`
	Type        enums.ScheduleActivityTypeEnum `json:"type" example:"MEETING"`
	StartTime   string                         `json:"start_time" example:"10:10"`
	EndTime     string                         `json:"end_time" example:"10:40"`
	Location    string                         `json:"location" example:"Zoom"`
	Note        string                         `json:"note" example:"Bring the roadmap doc"`
	IsRecurring bool                           `json:"is_recurring" example:"true"`
}

type WeekSummaryRes struct {
	TotalMinutes int `json:"total_minutes" example:"2280"`
	// Minutes planned per enums.ScheduleActivityTypeEnum value.
	ByType map[string]int `json:"by_type"`
	Count  int            `json:"count" example:"32"`
}
