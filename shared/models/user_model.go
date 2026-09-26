package model

type User struct {
	BaseModelID
	Username   string      `gorm:"type:varchar(50);not null;index"`
	Fullname   string      `gorm:"type:varchar(100);not null"`
	Email      string      `gorm:"type:varchar(100)"`
	Password   string      `gorm:"type:varchar(255); not null"`
	AvatarID   *uint       `gorm:"type:bigint;index" json:"avatar_id"`
	Avatar     *Attachment `gorm:"foreignKey:AvatarID;constraint:OnDelete:SET NULL" json:"avatar,omitempty"`
	IsDisabled bool        `gorm:"type:bool;default:0" json:"is_disabled"`
	BaseModelGorm
}
