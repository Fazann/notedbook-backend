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

type BudgetRepoImpl interface {
	baserepo.BaseRepo[model.Budget]
	WithTx(tx *gorm.DB) BudgetRepoImpl
	FindOneByID(ctx context.Context, userID, id uint) (*model.Budget, error)
	FindOneByCurrency(ctx context.Context, userID uint, currency enums.CurrencyEnum) (*model.Budget, error)
	ListByUser(ctx context.Context, userID uint) ([]model.Budget, error)
	SoftDelete(ctx context.Context, userID, id, deletedBy uint) error
}

type budgetRepo struct {
	db *gorm.DB
	baserepo.BaseRepo[model.Budget]
}

func NewBudgetRepo() *budgetRepo {
	return &budgetRepo{
		db:       db.DB,
		BaseRepo: baserepo.NewBaseRepo[model.Budget](db.DB),
	}
}

func (r *budgetRepo) WithTx(tx *gorm.DB) BudgetRepoImpl {
	return &budgetRepo{
		db:       tx,
		BaseRepo: baserepo.NewBaseRepo[model.Budget](tx),
	}
}

func (r *budgetRepo) FindOneByID(ctx context.Context, userID, id uint) (*model.Budget, error) {
	var budget model.Budget
	if err := r.db.WithContext(ctx).Where("id = ? AND user_id = ?", id, userID).First(&budget).Error; err != nil {
		return nil, err
	}

	return &budget, nil
}

func (r *budgetRepo) FindOneByCurrency(ctx context.Context, userID uint, currency enums.CurrencyEnum) (*model.Budget, error) {
	var budget model.Budget
	if err := r.db.WithContext(ctx).Where("user_id = ? AND currency = ?", userID, currency).First(&budget).Error; err != nil {
		return nil, err
	}

	return &budget, nil
}

func (r *budgetRepo) ListByUser(ctx context.Context, userID uint) ([]model.Budget, error) {
	budgets := []model.Budget{}
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).Order("id").Find(&budgets).Error; err != nil {
		return nil, err
	}

	return budgets, nil
}

// SoftDelete marks the user's budget as deleted by deletedBy.
func (r *budgetRepo) SoftDelete(ctx context.Context, userID, id, deletedBy uint) error {
	return r.db.WithContext(ctx).Model(&model.Budget{}).
		Where("id = ? AND user_id = ?", id, userID).
		Updates(map[string]any{"deleted_at": time.Now(), "deleted_by": deletedBy}).Error
}
