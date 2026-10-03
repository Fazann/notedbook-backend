package repo

import (
	"context"

	"github.com/Fazann/notedbook-backend/pkg/db"
	"github.com/Fazann/notedbook-backend/shared/baserepo"
	model "github.com/Fazann/notedbook-backend/shared/models"
	"gorm.io/gorm"
)

type PlanAreaRepoImpl interface {
	baserepo.BaseRepo[model.PlanArea]
	WithTx(tx *gorm.DB) PlanAreaRepoImpl
	FindAll(ctx context.Context) ([]model.PlanArea, error)
	FindByTitle(ctx context.Context, title string) (*model.PlanArea, error)
	ReplaceTranslations(ctx context.Context, planAreaID uint, trs []model.PlanAreaTr) error
}

type planAreaRepo struct {
	db *gorm.DB
	baserepo.BaseRepo[model.PlanArea]
}

func NewPlanAreaRepo() *planAreaRepo {
	return &planAreaRepo{
		db:       db.DB,
		BaseRepo: baserepo.NewBaseRepo[model.PlanArea](db.DB),
	}
}

func (r *planAreaRepo) WithTx(tx *gorm.DB) PlanAreaRepoImpl {
	return &planAreaRepo{
		db:       tx,
		BaseRepo: baserepo.NewBaseRepo[model.PlanArea](tx),
	}
}

// FindAll returns every plan area with its translations, oldest first.
func (r *planAreaRepo) FindAll(ctx context.Context) ([]model.PlanArea, error) {
	areas := []model.PlanArea{}
	if err := r.db.WithContext(ctx).Preload("Translations").Order("id ASC").Find(&areas).Error; err != nil {
		return nil, err
	}
	return areas, nil
}

// FindByTitle matches a plan area by title, including deleted ones, so a seed does not bring back
// an area an admin removed.
func (r *planAreaRepo) FindByTitle(ctx context.Context, title string) (*model.PlanArea, error) {
	var area model.PlanArea
	if err := r.db.WithContext(ctx).Unscoped().
		Where("LOWER(title) = LOWER(?)", title).First(&area).Error; err != nil {
		return nil, err
	}
	return &area, nil
}

// ReplaceTranslations deletes the area's translations and inserts trs instead.
func (r *planAreaRepo) ReplaceTranslations(ctx context.Context, planAreaID uint, trs []model.PlanAreaTr) error {
	if err := r.db.WithContext(ctx).Where("plan_area_id = ?", planAreaID).
		Delete(&model.PlanAreaTr{}).Error; err != nil {
		return err
	}
	if len(trs) == 0 {
		return nil
	}
	for i := range trs {
		trs[i].PlanAreaID = planAreaID
	}
	return r.db.WithContext(ctx).Create(&trs).Error
}
