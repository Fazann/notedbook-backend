package model

import "time"

type PlanStep struct {
	BaseModelID
	PlanID  uint       `gorm:"type:bigint;not null;index"`
	Title   string     `gorm:"type:varchar(255);not null"`
	Done    bool       `gorm:"not null;default:false"`
	DueDate *time.Time `gorm:"column:due_date;type:date"`
	// Float ordering inside the plan: 1000, 2000…; a move between two steps takes the midpoint.
	Position float64 `gorm:"type:double precision;not null;default:0;index"`
	BaseModelDateColumns
}
