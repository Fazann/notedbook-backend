package repo

import (
	"context"

	"github.com/Fazann/notedbook-backend/pkg/db"
	model "github.com/Fazann/notedbook-backend/shared/models"
	"gorm.io/gorm"
)

type RefreshTokenRepoImpl interface {
	WithTx(tx *gorm.DB) RefreshTokenRepoImpl
	FindUserTokenWithSessionID(ctx context.Context, userID uint, sessionID string) (*model.RefreshToken, error)
	IssueNewUserToken(ctx context.Context, data model.RefreshToken) (*model.RefreshToken, error)
	RevokeUserToken(ctx context.Context, userID uint) error
	RevokeByID(ctx context.Context, id uint) (bool, error)
}

type refreshTokenRepo struct {
	db *gorm.DB
}

func NewRefreshTokenRepo() *refreshTokenRepo {
	return &refreshTokenRepo{
		db: db.DB,
	}
}

func (r *refreshTokenRepo) WithTx(tx *gorm.DB) RefreshTokenRepoImpl {
	return &refreshTokenRepo{
		db: tx,
	}
}

func (r *refreshTokenRepo) FindUserTokenWithSessionID(ctx context.Context, userID uint, sessionID string) (
	*model.RefreshToken,
	error,
) {
	var data model.RefreshToken
	if err := r.db.WithContext(ctx).Model(&model.RefreshToken{}).
		Where("user_id = ? AND session_id = ?", userID, sessionID).
		First(&data).Error; err != nil {
		return nil, err
	}

	return &data, nil
}

func (r *refreshTokenRepo) IssueNewUserToken(ctx context.Context, data model.RefreshToken) (*model.RefreshToken, error) {
	if err := r.db.WithContext(ctx).Create(&data).Error; err != nil {
		return nil, err
	}

	return &data, nil
}

func (r *refreshTokenRepo) RevokeUserToken(ctx context.Context, userID uint) error {
	return r.db.WithContext(ctx).Model(&model.RefreshToken{}).
		Where("user_id = ?", userID).Update("is_revoked", true).Error
}

// RevokeByID revokes one token. It returns false when the token was already revoked, so two requests
// racing to use the same token cannot both succeed.
func (r *refreshTokenRepo) RevokeByID(ctx context.Context, id uint) (bool, error) {
	res := r.db.WithContext(ctx).Model(&model.RefreshToken{}).
		Where("id = ? AND is_revoked = ?", id, false).Update("is_revoked", true)
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected == 1, nil
}
