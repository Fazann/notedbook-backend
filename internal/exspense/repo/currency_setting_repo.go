package repo

import (
	"context"

	"github.com/Fazann/notedbook-backend/pkg/db"
	model "github.com/Fazann/notedbook-backend/shared/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type CurrencySettingRepoImpl interface {
	FindByUser(ctx context.Context, userID uint) (*model.CurrencySetting, error)
	// Upsert creates the user's settings or replaces them (one row per user).
	Upsert(ctx context.Context, setting *model.CurrencySetting) error
}

type currencySettingRepo struct {
	db *gorm.DB
}

func NewCurrencySettingRepo() *currencySettingRepo {
	return &currencySettingRepo{db: db.DB}
}

func (r *currencySettingRepo) FindByUser(ctx context.Context, userID uint) (*model.CurrencySetting, error) {
	var setting model.CurrencySetting
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).First(&setting).Error; err != nil {
		return nil, err
	}

	return &setting, nil
}

func (r *currencySettingRepo) Upsert(ctx context.Context, setting *model.CurrencySetting) error {
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"enabled_currencies", "default_currency", "updated_at"}),
	}).Create(setting).Error
}
