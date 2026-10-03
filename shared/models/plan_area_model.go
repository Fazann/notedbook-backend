package model

type PlanArea struct {
	BaseModelID
	Icon         string       `gorm:"type:varchar(100)"`
	Title        string       `gorm:"type:varchar(50)"`
	Translations []PlanAreaTr `gorm:"foreignKey:PlanAreaID;constraint:OnDelete:CASCADE"`
	BaseModelGorm
}
