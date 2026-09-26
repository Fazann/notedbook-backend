package model

import (
	"time"
)

type RefreshToken struct {
	BaseModelID
	UserID    uint      `gorm:"column:user_id;not null;index"`
	SessionID string    `gorm:"column:session_id;type:varchar(100);index"`
	ExpiredAt int64     `gorm:"column:expired_at;not null"`
	IsRevoked bool      `gorm:"column:is_revoked"`
	CreatedAt time.Time `json:"created_at"`
}
