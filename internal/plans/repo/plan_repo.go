package repo

import (
	"github.com/Fazann/notedbook-backend/pkg/db"
	"github.com/Fazann/notedbook-backend/shared/baserepo"
	model "github.com/Fazann/notedbook-backend/shared/models"
	"gorm.io/gorm"
)

type PlanRepoImpl interface {
	baserepo.BaseRepo[model.Plan]
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
