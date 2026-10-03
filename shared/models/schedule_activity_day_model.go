package model

// ScheduleActivityDay is one weekday (ISO: 1 = Monday … 7 = Sunday) a "weekly" recurring activity
// repeats on. Rows are replaced as a whole when the activity's recurrence is saved.
type ScheduleActivityDay struct {
	BaseModelID
	ActivityID uint `gorm:"type:bigint;not null;uniqueIndex:idx_schedule_activity_day"`
	Weekday    int  `gorm:"type:smallint;not null;uniqueIndex:idx_schedule_activity_day"`
}
