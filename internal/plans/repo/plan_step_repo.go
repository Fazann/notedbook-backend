package repo

import (
	"github.com/Fazann/notedbook-backend/pkg/db"
	"github.com/Fazann/notedbook-backend/shared/baserepo"
	model "github.com/Fazann/notedbook-backend/shared/models"
	"gorm.io/gorm"
)

type PlanStepRepoImpl interface {
	baserepo.BaseRepo[model.PlanStep]
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
