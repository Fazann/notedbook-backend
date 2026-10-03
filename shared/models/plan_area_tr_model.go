package model

// PlanAreaTr is a plan area title in one language. Rows are replaced as a whole when the area is saved.
type PlanAreaTr struct {
	BaseModelID
	PlanAreaID uint   `gorm:"type:bigint;not null;uniqueIndex:idx_plan_area_tr_lang" json:"plan_area_id"`
	Lang       string `gorm:"type:varchar(10);not null;uniqueIndex:idx_plan_area_tr_lang"`
	Title      string `gorm:"type:varchar(100);not null"`
}
