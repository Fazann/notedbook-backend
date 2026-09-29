package repo

import (
	"context"
	"time"

	"github.com/Fazann/notedbook-backend/pkg/db"
	"github.com/Fazann/notedbook-backend/shared/baserepo"
	model "github.com/Fazann/notedbook-backend/shared/models"
	"gorm.io/gorm"
)

// visibleTo matches the categories a user can see: their own and the admin (default) ones.
const visibleTo = "(user_id = ? OR user_id IS NULL)"

// CategoryFilter is the category list query. Sort fields left nil are not applied.
type CategoryFilter struct {
	// Lowercased, with LIKE wildcards escaped.
	Search        string  `filter:"column=name_search;op=like"`
	SortName      *string `sort:"LOWER(name)"`
	SortCreatedAt *string `sort:"created_at"`
	// Tie-breaker, so pages do not overlap when the sort value repeats.
	SortID *string `sort:"id"`
}

type CategoryRepoImpl interface {
	baserepo.BaseRepo[model.ExpenseCategory]
	WithTx(tx *gorm.DB) CategoryRepoImpl
	FindOneByID(ctx context.Context, userID, id uint) (*model.ExpenseCategory, error)
	FindByName(ctx context.Context, userID uint, name string) (*model.ExpenseCategory, error)
	FindDefaultByName(ctx context.Context, name string) (*model.ExpenseCategory, error)
	FindAllByUser(ctx context.Context, userID uint) ([]model.ExpenseCategory, error)
	List(ctx context.Context, userID uint, filter *CategoryFilter, page, limit int) ([]model.ExpenseCategory, int64, error)
	ReplaceTranslations(ctx context.Context, categoryID uint, trs []model.ExpenseCategoryTr) error
	SoftDelete(ctx context.Context, userID, id, deletedBy uint) error
}

type categoryRepo struct {
	baserepo.BaseRepo[model.ExpenseCategory]
	db *gorm.DB
}

func NewCategoryRepo() *categoryRepo {
	return &categoryRepo{
		db:       db.DB,
		BaseRepo: baserepo.NewBaseRepo[model.ExpenseCategory](db.DB),
	}
}

func (r *categoryRepo) WithTx(tx *gorm.DB) CategoryRepoImpl {
	return &categoryRepo{
		db:       tx,
		BaseRepo: baserepo.NewBaseRepo[model.ExpenseCategory](tx),
	}
}

// FindOneByID returns the category with its translations when userID can see it (own or default).
func (r *categoryRepo) FindOneByID(ctx context.Context, userID, id uint) (*model.ExpenseCategory, error) {
	var category model.ExpenseCategory
	if err := r.db.WithContext(ctx).Preload("Translations").
		Where("id = ? AND "+visibleTo, id, userID).First(&category).Error; err != nil {
		return nil, err
	}

	return &category, nil
}

// FindByName matches the base name case-insensitively among the categories userID can see.
func (r *categoryRepo) FindByName(ctx context.Context, userID uint, name string) (*model.ExpenseCategory, error) {
	var category model.ExpenseCategory
	if err := r.db.WithContext(ctx).
		Where(visibleTo+" AND LOWER(name) = LOWER(?)", userID, name).First(&category).Error; err != nil {
		return nil, err
	}

	return &category, nil
}

// FindDefaultByName matches a default category by base name, including deleted ones, so a seed does not
// bring back a category an admin removed.
func (r *categoryRepo) FindDefaultByName(ctx context.Context, name string) (*model.ExpenseCategory, error) {
	var category model.ExpenseCategory
	if err := r.db.WithContext(ctx).Unscoped().
		Where("user_id IS NULL AND LOWER(name) = LOWER(?)", name).First(&category).Error; err != nil {
		return nil, err
	}

	return &category, nil
}

// FindAllByUser returns the default categories first, then the user's own, in creation order.

func (r *categoryRepo) FindAllByUser(ctx context.Context, userID uint) ([]model.ExpenseCategory, error) {
	categories := []model.ExpenseCategory{}
	if err := r.db.WithContext(ctx).Preload("Translations").
		Where(visibleTo, userID).Order("user_id IS NULL DESC, id ASC").Find(&categories).Error; err != nil {
		return nil, err
	}

	return categories, nil
}

func (r *categoryRepo) List(ctx context.Context, userID uint, filter *CategoryFilter, page, limit int) ([]model.ExpenseCategory, int64, error) {
	return baserepo.NewQueryBuilder[model.ExpenseCategory](r.db.Preload("Translations").Where(visibleTo, userID)).
		ApplyFilters(filter).
		Paginate(ctx, page, limit)
}

// ReplaceTranslations deletes the category's translations and inserts trs instead.
func (r *categoryRepo) ReplaceTranslations(ctx context.Context, categoryID uint, trs []model.ExpenseCategoryTr) error {
	if err := r.db.WithContext(ctx).Where("category_id = ?", categoryID).
		Delete(&model.ExpenseCategoryTr{}).Error; err != nil {
		return err
	}
	if len(trs) == 0 {
		return nil
	}
	for i := range trs {
		trs[i].CategoryID = categoryID
	}
	return r.db.WithContext(ctx).Create(&trs).Error
}

// SoftDelete marks the user's category as deleted by deletedBy; its translations are kept.
func (r *categoryRepo) SoftDelete(ctx context.Context, userID, id, deletedBy uint) error {
	return r.db.WithContext(ctx).Model(&model.ExpenseCategory{}).
		Where("id = ? AND user_id = ?", id, userID).
		Updates(map[string]any{"deleted_at": time.Now(), "deleted_by": deletedBy}).Error
}
