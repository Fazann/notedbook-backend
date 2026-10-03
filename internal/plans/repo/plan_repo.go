package repo

import (
	"context"

	"github.com/Fazann/notedbook-backend/pkg/db"
	"github.com/Fazann/notedbook-backend/shared/baserepo"
	model "github.com/Fazann/notedbook-backend/shared/models"
	"gorm.io/gorm"
)

// PlanFilter is the plan list query. Zero filters and nil sort fields are not applied.
type PlanFilter struct {
	Status string `filter:"column=status"`
	AreaID uint   `filter:"column=area_id"`
	// Lowercased, with LIKE wildcards escaped.
	Search         string  `filter:"column=LOWER(title);op=like"`
	SortTargetDate *string `sort:"target_date"`
	SortCreatedAt  *string `sort:"created_at"`
	SortTitle      *string `sort:"LOWER(title)"`
	// Tie-breaker, so pages do not overlap when the sort value repeats.
	SortID *string `sort:"id"`
}

type PlanRepoImpl interface {
	baserepo.BaseRepo[model.Plan]
	WithTx(tx *gorm.DB) PlanRepoImpl
	FindOneByID(ctx context.Context, userID, id uint) (*model.Plan, error)
	List(ctx context.Context, userID uint, filter *PlanFilter, page, limit int) ([]model.Plan, int64, error)
	// AdjustStepCounts atomically changes step_count and done_step_count by the given deltas.
	AdjustStepCounts(ctx context.Context, planID uint, stepDelta, doneDelta int) error
	SoftDelete(ctx context.Context, userID, id uint) error
}

type planRepo struct {
	db *gorm.DB
	baserepo.BaseRepo[model.Plan]
}

func NewPlanRepo() *planRepo {
	return &planRepo{
		db:       db.DB,
		BaseRepo: baserepo.NewBaseRepo[model.Plan](db.DB),
	}
}

func (r *planRepo) WithTx(tx *gorm.DB) PlanRepoImpl {
	return &planRepo{
		db:       tx,
		BaseRepo: baserepo.NewBaseRepo[model.Plan](tx),
	}
}

// FindOneByID returns the user's plan.
func (r *planRepo) FindOneByID(ctx context.Context, userID, id uint) (*model.Plan, error) {
	var plan model.Plan
	if err := r.db.WithContext(ctx).Where("id = ? AND user_id = ?", id, userID).First(&plan).Error; err != nil {
		return nil, err
	}

	return &plan, nil
}

func (r *planRepo) List(ctx context.Context, userID uint, filter *PlanFilter, page, limit int) ([]model.Plan, int64, error) {
	return baserepo.NewQueryBuilder[model.Plan](r.db.Where("user_id = ?", userID)).
		ApplyFilters(filter).
		Paginate(ctx, page, limit)
}

func (r *planRepo) AdjustStepCounts(ctx context.Context, planID uint, stepDelta, doneDelta int) error {
	return r.db.WithContext(ctx).Model(&model.Plan{}).Where("id = ?", planID).
		Updates(map[string]any{
			"step_count":      gorm.Expr("step_count + ?", stepDelta),
			"done_step_count": gorm.Expr("done_step_count + ?", doneDelta),
		}).Error
}

// SoftDelete marks the user's plan, and its steps, as deleted.
func (r *planRepo) SoftDelete(ctx context.Context, userID, id uint) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		res := tx.Where("id = ? AND user_id = ?", id, userID).Delete(&model.Plan{})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		return tx.Where("plan_id = ?", id).Delete(&model.PlanStep{}).Error
	})
}
