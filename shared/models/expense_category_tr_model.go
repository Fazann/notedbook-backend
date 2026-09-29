package model

// ExpenseCategoryTr is a category name in one language. Rows are replaced as a whole when the category is saved.
type ExpenseCategoryTr struct {
	BaseModelID
	CategoryID uint   `gorm:"type:bigint;not null;uniqueIndex:idx_expense_category_tr_lang" json:"category_id"`
	Lang       string `gorm:"type:varchar(10);not null;uniqueIndex:idx_expense_category_tr_lang"`
	Name       string `gorm:"type:varchar(100);not null"`
}
