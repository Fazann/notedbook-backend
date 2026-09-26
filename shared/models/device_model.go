package model

import (
	"time"

	"github.com/Fazann/notedbook-backend/pkg/enums"
)

type Device struct {
	BaseModelID
	UserID    uint               `gorm:"index" json:"user_id"`
	Name      string             `gorm:"type:varchar(100);not null"`
	OS        enums.DeviceOSEnum `gorm:"type:varchar(20);not null"`
	Version   string             `gorm:"type:varchar(100)"`
	UID       string             `gorm:"column:uid;type:varchar(80);index; not null"`
	UserAgent string             `gorm:"column:useragent;type:varchar(300)"`
	LoginAt   time.Time          `gorm:"column:login_at" json:"login_at"`
	IsLogin   bool               `gorm:"column:is_login"`
	BaseModelGorm
}
