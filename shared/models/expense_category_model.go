package model

type ExpenseCategory struct {
	BaseModelID
	UserID       uint   `gorm:"type:bigint;index;default:null"`
	SystemUserID uint   `gorm:"type:bigint;index;default:null"` // for future api create by admin user
	Name         string `gorm:"type:varchar(100);not null"`
	// Lowercased base name and every translation, so a search matches the category in any language.
	NameSearch   string              `gorm:"column:name_search;type:varchar(300);not null;index"`
	Icon         string              `gorm:"type:varchar(50);not null"`
	Color        string              `gorm:"type:varchar(50);not null"`
	Translations []ExpenseCategoryTr `gorm:"foreignKey:CategoryID;constraint:OnDelete:CASCADE"`
	BaseModelGorm
}
