package repo

import (
	"context"

	"github.com/Fazann/notedbook-backend/pkg/db"
	"github.com/Fazann/notedbook-backend/shared/baserepo"
	model "github.com/Fazann/notedbook-backend/shared/models"
	"gorm.io/gorm"
)

type DeviceRepoImpl interface {
	baserepo.BaseRepo[model.Device]
	FindByUserWithUID(ctx context.Context, userID uint, uid string) (*model.Device, error)
}

type deviceRepo struct {
	db *gorm.DB
	baserepo.BaseRepo[model.Device]
}

func NewDeviceRepo() *deviceRepo {
	return &deviceRepo{
		db:       db.DB,
		BaseRepo: baserepo.NewBaseRepo[model.Device](db.DB),
	}
}

func (r *deviceRepo) FindByUserWithUID(ctx context.Context, userID uint, uid string) (*model.Device, error) {
	var device model.Device
	if err := r.db.WithContext(ctx).Model(&model.Device{}).
		Where("uid = ? AND user_id = ?", uid, userID).First(&device).Error; err != nil {
		return nil, err
	}

	return &device, nil
}
