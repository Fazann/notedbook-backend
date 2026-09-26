package repo

import (
	"context"

	"github.com/Fazann/notedbook-backend/pkg/db"
	"github.com/Fazann/notedbook-backend/shared/baserepo"
	model "github.com/Fazann/notedbook-backend/shared/models"
	"gorm.io/gorm"
)

type UserRepoImpl interface {
	baserepo.BaseRepo[model.User]
	WithTx(tx *gorm.DB) UserRepoImpl
	FindByUsername(ctx context.Context, username string) (*model.User, error)
	FindByUsernameOrEmail(ctx context.Context, username, email string) (*model.User, error)
	FindByIDWithAvatar(ctx context.Context, id uint) (*model.User, error)
}

type userRepo struct {
	baserepo.BaseRepo[model.User]
	db *gorm.DB
}

func NewUserRepo() *userRepo {
	return &userRepo{
		db:       db.DB,
		BaseRepo: baserepo.NewBaseRepo[model.User](db.DB),
	}
}

func (r *userRepo) WithTx(tx *gorm.DB) UserRepoImpl {
	return &userRepo{
		db:       tx,
		BaseRepo: baserepo.NewBaseRepo[model.User](tx),
	}
}

func (r *userRepo) FindByUsername(ctx context.Context, username string) (*model.User, error) {
	var user model.User
	if err := r.db.WithContext(ctx).Model(&model.User{}).Where("username = ?", username).First(&user).Error; err != nil {
		return nil, err
	}

	return &user, nil
}

func (r *userRepo) FindByEmail(ctx context.Context, email string) (*model.User, error) {
	var user model.User
	if err := r.db.WithContext(ctx).Model(&model.User{}).Where("email = ?", email).First(&user).Error; err != nil {
		return nil, err
	}

	return &user, nil
}

func (r *userRepo) FindByUsernameOrEmail(ctx context.Context, username, email string) (*model.User, error) {
	var user model.User
	if err := r.db.WithContext(ctx).Model(&model.User{}).
		Where("username = ? OR email = ?", username, email).First(&user).Error; err != nil {
		return nil, err
	}

	return &user, nil
}

func (r *userRepo) FindByIDWithAvatar(ctx context.Context, id uint) (*model.User, error) {
	var user model.User
	if err := r.db.WithContext(ctx).Preload("Avatar").First(&user, id).Error; err != nil {
		return nil, err
	}

	return &user, nil
}
