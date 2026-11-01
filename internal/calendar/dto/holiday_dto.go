package dto

type HolidayListReq struct {
	Year int `form:"year" binding:"required,min=1900,max=2100" example:"2026"`
	// KH or MY.
	Country string `form:"country" binding:"required,oneof=KH MY" example:"KH"`
	// national: public holidays of the country. islamic: Islamic holidays observed there.
	Kind string `form:"kind" binding:"required,oneof=national islamic" example:"national"`
}

// HolidayRes is one day of a holiday; a holiday over several days has one entry per day.
type HolidayRes struct {
	// Plain date, YYYY-MM-DD.
	Date string `json:"date" example:"2026-11-09"`
	// Stable key the client translates.
	Key string `json:"key" example:"independenceDay"`
	// English name, for clients without a translation of key.
	Name string `json:"name" example:"Independence Day"`
}
