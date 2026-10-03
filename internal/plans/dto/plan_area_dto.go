package dto

import "time"

type PlanAreaRes struct {
	ID    uint   `json:"id" example:"1"`
	Title string `json:"title" example:"Finance"`
	// Title in the request language (`?lang=` / Accept-Language), falling back to title.
	DisplayTitle string            `json:"display_title" example:"ហិរញ្ញវត្ថុ"`
	Translations map[string]string `json:"translations"`
	Icon         string            `json:"icon" example:"wallet"`
	CreatedAt    time.Time         `json:"created_at"`
	UpdatedAt    time.Time         `json:"updated_at"`
}
