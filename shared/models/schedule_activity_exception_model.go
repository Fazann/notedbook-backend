package model

import "time"

// ScheduleActivityException is one date removed from a repeating activity's series — "this
// occurrence only" was edited (a new single ScheduleActivity is created for that date) or deleted.
type ScheduleActivityException struct {
	BaseModelID
	ActivityID uint      `gorm:"type:bigint;not null;uniqueIndex:idx_schedule_activity_exception"`
	Date       time.Time `gorm:"type:date;not null;uniqueIndex:idx_schedule_activity_exception"`
}
