package service

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/Fazann/notedbook-backend/internal/users/dto"
	"github.com/Fazann/notedbook-backend/internal/users/repo"
	"github.com/Fazann/notedbook-backend/pkg/enums"
	"github.com/Fazann/notedbook-backend/pkg/i18n"
	"github.com/Fazann/notedbook-backend/pkg/logger"
	"github.com/Fazann/notedbook-backend/pkg/response"
	"github.com/Fazann/notedbook-backend/pkg/utils"
	model "github.com/Fazann/notedbook-backend/shared/models"
	"gorm.io/gorm"
)

type UserServiceImpl interface {
	GetProfile(ctx context.Context, userID uint) response.SvcRes
	UpdateProfile(ctx context.Context, userID uint, req *dto.UpdateProfileReq) response.SvcRes
	ChangePassword(ctx context.Context, req *dto.ChangePasswordReq) response.SvcRes
}

// AttachmentUsage is the part of the attachment service this module uses. It is passed in by
// routes, so users does not import the attachments module.
type AttachmentUsage interface {
	MarkUsed(ctx context.Context, id uint, userID uint, types ...enums.AttachmentTypeEnum) error
	Delete(ctx context.Context, id uint, deletedBy uint) error
}

type userSvc struct {
	userRepo    repo.UserRepoImpl
	attachments AttachmentUsage
}

func NewUserSvc(attachments AttachmentUsage) *userSvc {
	return &userSvc{
		userRepo:    repo.NewUserRepo(),
		attachments: attachments,
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

func (s *userSvc) UpdateProfile(ctx context.Context, userID uint, req *dto.UpdateProfileReq) response.SvcRes {
	user, err := s.userRepo.FindOne(ctx, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return response.SvcRespErr(i18n.RecordNotFound, http.StatusNotFound)
		}
		return response.SvcRespErr(err, http.StatusInternalServerError)
	}
	if user.IsDisabled {
		return response.SvcRespErr(i18n.AccountDisabled, http.StatusForbidden)
	}

	updates := map[string]any{}

	if req.Fullname != nil {
		fullname := strings.TrimSpace(*req.Fullname)
		if n := utf8.RuneCountInString(fullname); n < 2 || n > 50 {
			return response.SvcRespErr(i18n.InvalidFullname, http.StatusBadRequest)
		}
		updates["fullname"] = fullname
	}

	if req.Username != nil {
		username, err := cleanUsername(*req.Username)
		if err != nil {
			return response.SvcRespErr(err, http.StatusBadRequest)
		}
		if username != user.Username {
			taken, err := s.isTakenByOther(userID, func() (*model.User, error) { return s.userRepo.FindByUsername(ctx, username) })
			if err != nil {
				return response.SvcRespErr(err, http.StatusInternalServerError)
			}
			if taken {
				return response.SvcRespErr(i18n.UsernameTaken, http.StatusConflict)
			}
			updates["username"] = username
		}
	}

	if req.Email != nil {
		email, err := cleanEmail(*req.Email)
		if err != nil {
			return response.SvcRespErr(err, http.StatusBadRequest)
		}
		if email != user.Email {
			taken, err := s.isTakenByOther(userID, func() (*model.User, error) { return s.userRepo.FindByEmail(ctx, email) })
			if err != nil {
				return response.SvcRespErr(err, http.StatusInternalServerError)
			}
			if taken {
				return response.SvcRespErr(i18n.EmailTaken, http.StatusConflict)
			}
			updates["email"] = email
		}
	}

	// Claim the avatar last, so a rejected request does not leave the file marked as used.
	if req.AvatarID != nil && (user.AvatarID == nil || *user.AvatarID != *req.AvatarID) {
		if err := s.attachments.MarkUsed(ctx, *req.AvatarID, userID, enums.AttachmentTypeImage); err != nil {
			return response.SvcRespErr(err, avatarErrStatus(err))
		}
		updates["avatar_id"] = *req.AvatarID
	}

	if len(updates) > 0 {
		updates["updated_by"] = userID
		if err := s.userRepo.Update(ctx, map[string]any{"id": userID}, updates); err != nil {
			return response.SvcRespErr(fmt.Errorf("update profile: %w", err), http.StatusInternalServerError)
		}
	}

	// The profile is saved; a failure here only leaves an unused file behind, so it is logged, not returned.
	if _, changed := updates["avatar_id"]; changed && user.AvatarID != nil {
		if err := s.attachments.Delete(ctx, *user.AvatarID, userID); err != nil {
			logger.Error("delete old avatar failed", "err", err, "attachment_id", *user.AvatarID, "user_id", userID)
		}
	}

	return s.GetProfile(ctx, userID)
}

func (s *userSvc) ChangePassword(ctx context.Context, req *dto.ChangePasswordReq) response.SvcRes {
	user, err := s.userRepo.FindOne(ctx, req.UserID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return response.SvcRespErr(i18n.RecordNotFound, http.StatusNotFound)
		}
		return response.SvcRespErr(err, http.StatusInternalServerError)
	}
	if user.IsDisabled {
		return response.SvcRespErr(i18n.AccountDisabled, http.StatusForbidden)
	}

	if err := utils.CompareHash(user.Password, req.CurrentPassword); err != nil {
		return response.SvcRespErr(i18n.IncorrectPassword, http.StatusBadRequest)
	}

	newHash, err := utils.GenerateHash(req.NewPassword)
	if err != nil {
		return response.SvcRespErr(fmt.Errorf("hash password: %w", err), http.StatusInternalServerError)
	}

	updates := map[string]any{"password": newHash, "updated_by": req.UserID}
	if err := s.userRepo.Update(ctx, map[string]any{"id": req.UserID}, updates); err != nil {
		return response.SvcRespErr(fmt.Errorf("change password: %w", err), http.StatusInternalServerError)
	}

	return s.GetProfile(ctx, req.UserID)
}

// isTakenByOther reports whether find returns a user other than userID.
func (s *userSvc) isTakenByOther(userID uint, find func() (*model.User, error)) (bool, error) {
	other, err := find()
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return other.ID != userID, nil
}

func avatarErrStatus(err error) int {
	key, _, ok := i18n.FromError(err)
	if !ok {
		return http.StatusInternalServerError
	}
	switch key {
	case i18n.RecordNotFound:
		return http.StatusNotFound
	case i18n.AttachmentInUse:
		return http.StatusConflict
	case i18n.InvalidAttachmentType:
		return http.StatusBadRequest
	}
	return http.StatusInternalServerError
}
