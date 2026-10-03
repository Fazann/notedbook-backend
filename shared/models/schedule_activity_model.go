package model

import (
	"time"

	"github.com/Fazann/notedbook-backend/pkg/enums"
)

// ScheduleActivity is a single event or a repeating series on the user's weekly schedule.
// Date and the times are wall-clock values in the user's timezone, never combined into a UTC
// timestamp: date math is done on them as plain values, not with time zone conversions.
type ScheduleActivity struct {
	BaseModelID
	UserID uint                           `gorm:"type:bigint;not null;index"`
	Title  string                         `gorm:"type:varchar(100);not null"`
	Type   enums.ScheduleActivityTypeEnum `gorm:"type:varchar(20);not null"`
	// The first (or only) day; YYYY-MM-DD.
	Date time.Time `gorm:"type:date;not null;index"`
	// HH:mm, 24h.
	StartTime string `gorm:"column:start_time;type:varchar(5);not null"`
	EndTime   string `gorm:"column:end_time;type:varchar(5);not null"`
	Location  string `gorm:"type:varchar(100)"`
	Note      string `gorm:"type:varchar(500)"`
	// NONE for a single occurrence; WEEKLY uses Days for which weekdays it repeats on.
	Recurrence enums.RecurrenceKindEnum `gorm:"type:varchar(10);not null;default:NONE"`
	// Inclusive end date of the series; nil means it never ends.
	RecurrenceUntil *time.Time                  `gorm:"column:recurrence_until;type:date"`
	Days            []ScheduleActivityDay       `gorm:"foreignKey:ActivityID;constraint:OnDelete:CASCADE"`
	Exceptions      []ScheduleActivityException `gorm:"foreignKey:ActivityID;constraint:OnDelete:CASCADE"`
	BaseModelGorm
}
