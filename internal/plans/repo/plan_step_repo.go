package repo

import (
	"context"

	"github.com/Fazann/notedbook-backend/pkg/db"
	"github.com/Fazann/notedbook-backend/shared/baserepo"
	model "github.com/Fazann/notedbook-backend/shared/models"
	"gorm.io/gorm"
)

type PlanStepRepoImpl interface {
	baserepo.BaseRepo[model.PlanStep]
	WithTx(tx *gorm.DB) PlanStepRepoImpl
	FindOneByID(ctx context.Context, id uint) (*model.PlanStep, error)
	FindByPlan(ctx context.Context, planID uint) ([]model.PlanStep, error)
	// LastPosition is the highest position in the plan, or 0 when it has no steps.
	LastPosition(ctx context.Context, planID uint) (float64, error)
	SoftDelete(ctx context.Context, id uint) error
}

type planStepRepo struct {
	db *gorm.DB
	baserepo.BaseRepo[model.PlanStep]
}

func NewPlanStepRepo() *planStepRepo {
	return &planStepRepo{
		db:       db.DB,
		BaseRepo: baserepo.NewBaseRepo[model.PlanStep](db.DB),
	}
}

func (r *planStepRepo) WithTx(tx *gorm.DB) PlanStepRepoImpl {
	return &planStepRepo{
		db:       tx,
		BaseRepo: baserepo.NewBaseRepo[model.PlanStep](tx),
	}
}

func (r *planStepRepo) FindOneByID(ctx context.Context, id uint) (*model.PlanStep, error) {
	var step model.PlanStep
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&step).Error; err != nil {
		return nil, err
	}

	return &step, nil
}

// FindByPlan returns every step of planID by position (oldest first for equal positions).
func (r *planStepRepo) FindByPlan(ctx context.Context, planID uint) ([]model.PlanStep, error) {
	steps := []model.PlanStep{}
	err := r.db.WithContext(ctx).Where("plan_id = ?", planID).Order("position ASC, id ASC").Find(&steps).Error
	if err != nil {
		return nil, err
	}

	return steps, nil
}

func (r *planStepRepo) LastPosition(ctx context.Context, planID uint) (float64, error) {
	var last float64
	err := r.db.WithContext(ctx).Model(&model.PlanStep{}).Where("plan_id = ?", planID).
		Select("COALESCE(MAX(position), 0)").Scan(&last).Error
	return last, err
}

// BackfillPositions gives steps created before positions existed (position 0) the positions 1000, 2000…
// per plan, in creation order. Plans that already have positioned steps are left alone, so it is safe to run
// again. Returns the number of steps updated.
func BackfillPositions(ctx context.Context, tx *gorm.DB) (int64, error) {
	res := tx.WithContext(ctx).Exec(`
		UPDATE plan_steps AS s SET position = r.rn * 1000
		FROM (
			SELECT id, ROW_NUMBER() OVER (PARTITION BY plan_id ORDER BY id) AS rn
			FROM plan_steps
			WHERE plan_id NOT IN (SELECT plan_id FROM plan_steps WHERE position <> 0)
		) AS r
		WHERE s.id = r.id`)
	return res.RowsAffected, res.Error
}

// SoftDelete deletes the step; ownership of its plan is checked by the caller first.
func (r *planStepRepo) SoftDelete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Where("id = ?", id).Delete(&model.PlanStep{}).Error
}
