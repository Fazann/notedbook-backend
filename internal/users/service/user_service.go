package service

import (
	"context"
	"errors"
	"net/http"

	"github.com/Fazann/notedbook-backend/internal/users/dto"
	"github.com/Fazann/notedbook-backend/internal/users/repo"
	"github.com/Fazann/notedbook-backend/pkg/i18n"
	"github.com/Fazann/notedbook-backend/pkg/response"
	"gorm.io/gorm"
)

type UserServiceImpl interface {
	GetProfile(ctx context.Context, userID uint) response.SvcRes
}

type userSvc struct {
	userRepo repo.UserRepoImpl
}

func NewUserSvc() *userSvc {
	return &userSvc{
		userRepo: repo.NewUserRepo(),
	}
}

func (s *userSvc) GetProfile(ctx context.Context, userID uint) response.SvcRes {
	user, err := s.userRepo.FindByIDWithAvatar(ctx, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return response.SvcRespErr(i18n.RecordNotFound, http.StatusNotFound)
		}
		return response.SvcRespErr(err, http.StatusInternalServerError)
	}
	if user.IsDisabled {
		return response.SvcRespErr(i18n.AccountDisabled, http.StatusForbidden)
	}

	res := &dto.ProfileRes{
		ID:        user.ID,
		Username:  user.Username,
		Fullname:  user.Fullname,
		Email:     user.Email,
		CreatedAt: user.CreatedAt,
	}
	if user.Avatar != nil {
		res.Avatar = &dto.AvatarRes{
			ID:           user.Avatar.ID,
			Url:          user.Avatar.Path,
			ThumbnailUrl: user.Avatar.ThumbnailUrl,
		}
	}

	return response.SvcRespSuccess(res, http.StatusOK)
}
