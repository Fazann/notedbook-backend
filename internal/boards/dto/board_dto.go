package dto

import "github.com/Fazann/notedbook-backend/pkg/enums"

type BoardListRes struct {
	ID   uint   `json:"id" example:"1"`
	Name string `json:"name" example:"Side project"`
	// Cards in every column of the board.
	CardCount int64 `json:"card_count" example:"12"`
}

type DueCardListReq struct {
	// Cards due up to this many days after `today` are returned (overdue ones too). Defaults to 7.
	Days int `form:"days" binding:"omitempty,min=1,max=60" example:"7"`
	// The user's current date, YYYY-MM-DD; defaults to today in UTC.
	Today string `form:"today" binding:"omitempty,datetime=2006-01-02" example:"2026-10-03"`
}

type DueCardRes struct {
	ID          uint   `json:"id" example:"1"`
	ColumnID    uint   `json:"column_id" example:"2"`
	Title       string `json:"title" example:"Write release notes"`
	Description string `json:"description" example:""`
	// Plain date, YYYY-MM-DD.
	DueDate *string `json:"due_date" example:"2026-10-05"`
	// null when the card has no label.
	Label      *enums.BoardLabelEnum `json:"label" example:"red"`
	Position   float64               `json:"position" example:"1000"`
	BoardID    uint                  `json:"board_id" example:"1"`
	BoardName  string                `json:"board_name" example:"Side project"`
	ColumnName string                `json:"column_name" example:"In progress"`
}
