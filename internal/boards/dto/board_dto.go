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

type BoardReq struct {
	Name string `json:"name" binding:"required,min=1,max=100" example:"Side project"`
	// Names of the columns to start with, left to right (the client sends them in the user's language).
	// The last column counts as "Done".
	Columns []string `json:"columns" binding:"omitempty,max=10,dive,min=1,max=50" example:"To do,In progress,Done"`
}

type BoardUpdateReq struct {
	Name string `json:"name" binding:"required,min=1,max=100" example:"Side project"`
}

// BoardRes is a board with its columns (by position) and each column's cards (by position).
type BoardRes struct {
	ID      uint         `json:"id" example:"1"`
	Name    string       `json:"name" example:"Side project"`
	Columns []*ColumnRes `json:"columns"`
}

type ColumnReq struct {
	Name string `json:"name" binding:"required,min=1,max=50" example:"In review"`
}

type ColumnRes struct {
	ID       uint       `json:"id" example:"2"`
	BoardID  uint       `json:"board_id" example:"1"`
	Name     string     `json:"name" example:"In progress"`
	Position float64    `json:"position" example:"2000"`
	Cards    []*CardRes `json:"cards"`
}

// CardReq creates a card, or replaces all of its fields (PUT): send null to clear due_date or label.
type CardReq struct {
	Title       string `json:"title" binding:"required,min=1,max=255" example:"Write release notes"`
	Description string `json:"description" binding:"max=5000" example:""`
	// Plain date, YYYY-MM-DD, or null.
	DueDate *string `json:"due_date" binding:"omitempty,datetime=2006-01-02" example:"2026-10-05"`
	// red, yellow, green, blue, purple, or null.
	Label *string `json:"label" binding:"omitempty,oneof=red yellow green blue purple" example:"red"`
}

// CardMoveReq moves a card to a column of the same board: the client sends the new position between its
// new neighbours (see calcPosition in the frontend).
type CardMoveReq struct {
	ColumnID uint     `json:"column_id" binding:"required,min=1" example:"3"`
	Position *float64 `json:"position" binding:"required" example:"1500"`
}

type CardRes struct {
	ID          uint   `json:"id" example:"1"`
	ColumnID    uint   `json:"column_id" example:"2"`
	Title       string `json:"title" example:"Write release notes"`
	Description string `json:"description" example:""`
	// Plain date, YYYY-MM-DD; null when not set.
	DueDate *string `json:"due_date" example:"2026-10-05"`
	// null when the card has no label.
	Label    *enums.BoardLabelEnum `json:"label" example:"red"`
	Position float64               `json:"position" example:"1000"`
}
