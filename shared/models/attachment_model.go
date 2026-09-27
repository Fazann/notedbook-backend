package model

import (
	"time"

	"github.com/Fazann/notedbook-backend/pkg/enums"
	"github.com/Fazann/notedbook-backend/pkg/utils"
	"gorm.io/gorm"
)

type Attachment struct {
	BaseModelID
	MetaData      any                      `gorm:"type:json"`
	Type          enums.AttachmentTypeEnum `gorm:"type:varchar(50);not null"`
	Filename      string                   `gorm:"type:varchar(255);not null"`
	Size          int64                    `gorm:"type:int;not null"`
	Width         int16                    `gorm:"type:smallint;not null"`
	Height        int16                    `gorm:"type:smallint;not null"`
	OriginalName  string                   `gorm:"type:varchar(255);not null"`
	Mimetype      string                   `gorm:"type:varchar(255);not null"`
	Path          string                   `gorm:"type:varchar(255);not null"`
	ThumbnailPath string                   `gorm:"type:varchar(255);not null"`
	ThumbnailUrl  string                   `gorm:"type:varchar(255);not null"`
	Duration      *float64
	UsedAt        *time.Time `gorm:"type:timestamptz"` // nil until a record uses the file
	BaseModelGorm
}

func (a *Attachment) AfterFind(tx *gorm.DB) error {
	a.Path = utils.GetFullFileUrl(a.Path)

	if a.ThumbnailPath != "" {
		a.ThumbnailUrl = utils.GetFullFileUrl(a.ThumbnailPath)
	}

	return nil
}
