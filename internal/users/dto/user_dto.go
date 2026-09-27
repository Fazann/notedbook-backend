package dto

import "time"

type ProfileRes struct {
	ID        uint       `json:"id" example:"1"`
	Username  string     `json:"username" example:"john"`
	Fullname  string     `json:"fullname" example:"John Doe"`
	Email     string     `json:"email" example:"john@example.com"`
	Avatar    *AvatarRes `json:"avatar"`
	CreatedAt time.Time  `json:"created_at"`
}

type AvatarRes struct {
	ID           uint   `json:"id" example:"10"`
	Url          string `json:"url"`
	ThumbnailUrl string `json:"thumbnail_url,omitempty"`
}

// UpdateProfileReq changes only the fields that are sent.
type UpdateProfileReq struct {
	Fullname *string `json:"fullname" binding:"omitempty,min=2,max=50" example:"John Doe"`
	Username *string `json:"username" binding:"omitempty,min=2,max=50" example:"john"`
	Email    *string `json:"email" binding:"omitempty,max=100" example:"john@example.com"`
	// ID of an uploaded image that is not used by another record yet.
	AvatarID *uint `json:"avatar_id" binding:"omitempty,min=1" example:"10"`
}

type ChangePasswordReq struct {
	UserID          uint   `json:"-"`
	CurrentPassword string `json:"current_password" binding:"required,min=6,max=20"`
	NewPassword     string `json:"new_password" binding:"required,min=6,max=20"`
}
