package model

import (
	"time"

	"github.com/Fazann/notedbook-backend/pkg/enums"
)

type Plan struct {
	BaseModelID
	UserID        uint                   `gorm:"index"`
	Title         string                 `gorm:"type:varchar(255);not null"`
	Description   string                 `gorm:"type:text"`
	Area          string                 `gorm:"type:varchar(100);not null"`
	Priority      enums.PlanPriorityEnum `gorm:"type:varchar(15);not null"`
	TargetDate    *time.Time             `gorm:"type:date;not null"`
	StepCount     int                    `gorm:"type:int;default:0"`
	DoneStepCount int                    `gorm:"column:done_step_count;type:int;default:0"`
	Status        enums.PlanPriorityEnum `gorm:"type:varchar(15);default:NOT_START"`
	DoneAt        *time.Time             `gorm:"column:done_at;type:date"`
	BaseModelDateColumns
}
