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
