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

// FindByPlan returns every step of planID, oldest first.
func (r *planStepRepo) FindByPlan(ctx context.Context, planID uint) ([]model.PlanStep, error) {
	steps := []model.PlanStep{}
	if err := r.db.WithContext(ctx).Where("plan_id = ?", planID).Order("id ASC").Find(&steps).Error; err != nil {
		return nil, err
	}

	return steps, nil
}

// SoftDelete deletes the step; ownership of its plan is checked by the caller first.
func (r *planStepRepo) SoftDelete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Where("id = ?", id).Delete(&model.PlanStep{}).Error
}
