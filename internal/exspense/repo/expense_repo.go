package repo

import (
	"context"
	"time"

	"github.com/Fazann/notedbook-backend/pkg/db"
	"github.com/Fazann/notedbook-backend/pkg/enums"
	"github.com/Fazann/notedbook-backend/shared/baserepo"
	model "github.com/Fazann/notedbook-backend/shared/models"
	"gorm.io/gorm"
)

// ExpenseFilter is the expense list query. Zero filters and nil sort fields are not applied;
// sorts apply in field order.
type ExpenseFilter struct {
	SpentFrom  time.Time `filter:"column=spent_at;op=gte"`
	SpentTo    time.Time `filter:"column=spent_at;op=lt"`
	CategoryID uint      `filter:"column=category_id"`
	Currency   string    `filter:"column=currency"`
	// Lowercased, with LIKE wildcards escaped.
	Search string `filter:"column=LOWER(note);op=like"`
	// Amounts in different currencies cannot be compared, so an amount sort groups by currency first.
	SortCurrency *string `sort:"currency"`
	SortAmount   *string `sort:"amount"`
	SortSpentAt  *string `sort:"spent_at"`
	// Tie-breaker, so pages do not overlap when the sort value repeats.
	SortID *string `sort:"id"`
}

// CategoryTotal is the sum (minor units) and count of a user's expenses in one category and currency.
type CategoryTotal struct {
	CategoryID uint
	Currency   enums.CurrencyEnum
	Amount     int64
	Count      int64
}

type ExpenseRepoImpl interface {
	baserepo.BaseRepo[model.Expense]
	WithTx(tx *gorm.DB) ExpenseRepoImpl
	FindOneByID(ctx context.Context, userID, id uint) (*model.Expense, error)
	List(ctx context.Context, userID uint, filter *ExpenseFilter, page, limit int) ([]model.Expense, int64, error)
	SumByCategory(ctx context.Context, userID uint, from, to time.Time) ([]CategoryTotal, error)
	CountByCategory(ctx context.Context, userID uint, categoryIDs []uint) (map[uint]int64, error)
	ReassignCategory(ctx context.Context, userID, fromID, toID uint) error
	SoftDelete(ctx context.Context, userID, id, deletedBy uint) error
}

type expenseRepo struct {
	db *gorm.DB
	baserepo.BaseRepo[model.Expense]
}

func NewExpenseRepo() *expenseRepo {
	return &expenseRepo{
		db:       db.DB,
		BaseRepo: baserepo.NewBaseRepo[model.Expense](db.DB),
	}
}

func (r *expenseRepo) WithTx(tx *gorm.DB) ExpenseRepoImpl {
	return &expenseRepo{
		db:       tx,
		BaseRepo: baserepo.NewBaseRepo[model.Expense](tx),
	}
}

func (r *expenseRepo) FindOneByID(ctx context.Context, userID, id uint) (*model.Expense, error) {
	var expense model.Expense
	if err := r.db.WithContext(ctx).Where("id = ? AND user_id = ?", id, userID).First(&expense).Error; err != nil {
		return nil, err
	}

	return &expense, nil
}

func (r *expenseRepo) List(ctx context.Context, userID uint, filter *ExpenseFilter, page, limit int) ([]model.Expense, int64, error) {
	return baserepo.NewQueryBuilder[model.Expense](r.db.Where("user_id = ?", userID)).
		ApplyFilters(filter).
		Paginate(ctx, page, limit)
}

// SumByCategory totals the user's expenses spent in [from, to) per category and currency.
func (r *expenseRepo) SumByCategory(ctx context.Context, userID uint, from, to time.Time) ([]CategoryTotal, error) {
	rows := []CategoryTotal{}
	if err := r.db.WithContext(ctx).Model(&model.Expense{}).
		Select("category_id, currency, SUM(amount) AS amount, COUNT(*) AS count").
		Where("user_id = ? AND spent_at >= ? AND spent_at < ?", userID, from, to).
		Group("category_id, currency").Order("category_id, currency").
		Scan(&rows).Error; err != nil {
		return nil, err
	}

	return rows, nil
}

// CountByCategory returns how many of the user's expenses use each category; categories without any are left out.
func (r *expenseRepo) CountByCategory(ctx context.Context, userID uint, categoryIDs []uint) (map[uint]int64, error) {
	counts := make(map[uint]int64, len(categoryIDs))
	if len(categoryIDs) == 0 {
		return counts, nil
	}

	var rows []struct {
		CategoryID uint
		Count      int64
	}
	if err := r.db.WithContext(ctx).Model(&model.Expense{}).
		Select("category_id, COUNT(*) AS count").
		Where("user_id = ? AND category_id IN ?", userID, categoryIDs).
		Group("category_id").Scan(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		counts[row.CategoryID] = row.Count
	}
	return counts, nil
}

// ReassignCategory moves the user's expenses from one category to another.
func (r *expenseRepo) ReassignCategory(ctx context.Context, userID, fromID, toID uint) error {
	return r.db.WithContext(ctx).Model(&model.Expense{}).
		Where("user_id = ? AND category_id = ?", userID, fromID).
		Updates(map[string]any{"category_id": toID, "updated_by": userID}).Error
}

// SoftDelete marks the user's expense as deleted by deletedBy.
func (r *expenseRepo) SoftDelete(ctx context.Context, userID, id, deletedBy uint) error {
	return r.db.WithContext(ctx).Model(&model.Expense{}).
		Where("id = ? AND user_id = ?", id, userID).
		Updates(map[string]any{"deleted_at": time.Now(), "deleted_by": deletedBy}).Error
}
