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
	WithTx(tx *gorm.DB) DeviceRepoImpl
	FindByUserWithUID(ctx context.Context, userID uint, uid string) (*model.Device, error)
	LogoutOtherUsers(ctx context.Context, uid string, userID uint) error
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

func (r *deviceRepo) WithTx(tx *gorm.DB) DeviceRepoImpl {
	return &deviceRepo{
		db:       tx,
		BaseRepo: baserepo.NewBaseRepo[model.Device](tx),
	}
}

func (r *deviceRepo) FindByUserWithUID(ctx context.Context, userID uint, uid string) (*model.Device, error) {
	var device model.Device
	if err := r.db.WithContext(ctx).Model(&model.Device{}).
		Where("uid = ? AND user_id = ?", uid, userID).First(&device).Select([]string{
		"id",
		"uid",
		"user_id",
	}).Error; err != nil {
		return nil, err
	}

	return &device, nil
}

// LogoutOtherUsers marks the device with this uid as logged out for every user except userID.
func (r *deviceRepo) LogoutOtherUsers(ctx context.Context, uid string, userID uint) error {
	return r.db.WithContext(ctx).Model(&model.Device{}).
		Where("uid = ? AND user_id <> ? AND is_login = ?", uid, userID, true).
		Update("is_login", false).Error
}
