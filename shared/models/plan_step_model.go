package model

import "time"

type PlanStep struct {
	BaseModelID
	PlanID  uint       `gorm:"index"`
	Title   string     `gorm:"type:varchar(255)"`
	Done    bool       `gorm:"default:0"`
	DueDate *time.Time `gorm:"column:due_date;type:date"`
	BaseModelDateColumns
}
