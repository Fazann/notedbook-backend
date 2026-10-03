package model

import (
	"time"

	"github.com/Fazann/notedbook-backend/pkg/enums"
)

type Board struct {
	BaseModelID
	UserID  uint          `gorm:"type:bigint;not null;index"`
	Name    string        `gorm:"type:varchar(100);not null"`
	Columns []BoardColumn `gorm:"foreignKey:BoardID;constraint:OnDelete:CASCADE"`
	BaseModelDateColumns
}

// BoardColumn is a kanban column; the last one by position is treated as "Done".
type BoardColumn struct {
	BaseModelID
	BoardID uint   `gorm:"type:bigint;not null;index"`
	Name    string `gorm:"type:varchar(50);not null"`
	// Float ordering inside the board: 1000, 2000…; a move between two columns takes the midpoint.
	Position float64     `gorm:"type:double precision;not null;default:0"`
	Cards    []BoardCard `gorm:"foreignKey:ColumnID;constraint:OnDelete:CASCADE"`
	BaseModelDateColumns
}

type BoardCard struct {
	BaseModelID
	ColumnID    uint                  `gorm:"type:bigint;not null;index"`
	Title       string                `gorm:"type:varchar(255);not null"`
	Description string                `gorm:"type:text"`
	DueDate     *time.Time            `gorm:"column:due_date;type:date;index"`
	Label       *enums.BoardLabelEnum `gorm:"type:varchar(10)"`
	// Float ordering inside the column, like BoardColumn.Position.
	Position float64 `gorm:"type:double precision;not null;default:0"`
	BaseModelDateColumns
}
