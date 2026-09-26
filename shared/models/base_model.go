package model

import (
	"time"

	"gorm.io/gorm"
)

type BaseModelID struct {
	ID uint `gorm:"primarykey" json:"id"`
}

type BaseModelGorm struct {
	CreatedAt time.Time      `json:"created_at"`
	CreatedBy uint           `json:"created_by"`
	UpdatedAt time.Time      `json:"updated_at"`
	UpdatedBy uint           `json:"updated_by"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at"`
	DeletedBy uint           `json:"deleted_by"`
}
