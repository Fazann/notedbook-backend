package handler

import (
	"net/http"

	"github.com/Fazann/notedbook-backend/internal/users/dto"
	"github.com/Fazann/notedbook-backend/internal/users/service"
	"github.com/Fazann/notedbook-backend/pkg/response"
	"github.com/Fazann/notedbook-backend/shared/middleware"
	"github.com/gin-gonic/gin"
)

type UserHandler struct {
	userSvc service.UserServiceImpl
}

func NewUserHandler(attachments service.AttachmentUsage) *UserHandler {
	return &UserHandler{
		userSvc: service.NewUserSvc(attachments),
	}
}

// GetProfile godoc
//
//	@Summary		Get my profile
//	@Description	Returns the signed-in user's profile, with the avatar's full URL when set.
//	@Tags			auth
//	@Produce		json
//	@Security		BearerAuth
//	@Param			Accept-Language	header		string	false	"Language of error messages"	Enums(en, km, ms)
//	@Success		200				{object}	response.SuccessRes{data=dto.ProfileRes}
//	@Failure		429				{object}	response.ErrorRes	"TOO_MANY_REQUESTS"
//	@Router			/v1/auth/profile [get]
func (h *UserHandler) GetProfile(c *gin.Context) {
	res := h.userSvc.GetProfile(c.Request.Context(), middleware.CurrentUserID(c))
	if res.Error != nil {
		response.JsonErrRes(c, res.Error, res.StatusCode)
		return
	}

	response.JsonSuccessRes(c, res.Data, res.StatusCode)
}

// UpdateProfile godoc
//
//	@Summary		Update my profile
//	@Description	Changes only the fields that are sent. Username and email must not belong to another user.
//	@Description	avatar_id must be an uploaded image that is not used yet; it is then marked as used and the
//	@Description	previous avatar, if any, is deleted.
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			body			body		dto.UpdateProfileReq	true	"Fields to change"
//	@Param			Accept-Language	header		string					false	"Language of error messages"	Enums(en, km, ms)
//	@Success		200				{object}	response.SuccessRes{data=dto.ProfileRes}
//	@Failure		413				{object}	response.ErrorRes	"REQUEST_TOO_LARGE"
//	@Failure		429				{object}	response.ErrorRes	"TOO_MANY_REQUESTS"
//	@Router			/v1/auth/profile [patch]
func (h *UserHandler) UpdateProfile(c *gin.Context) {
	var req dto.UpdateProfileReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.JsonErrRes(c, err, http.StatusBadRequest)
		return
	}

	res := h.userSvc.UpdateProfile(c.Request.Context(), middleware.CurrentUserID(c), &req)
	if res.Error != nil {
		response.JsonErrRes(c, res.Error, res.StatusCode)
		return
	}

	response.JsonSuccessRes(c, res.Data, res.StatusCode)
}

// ChangePassword godoc
//
//	@Summary		Change my password
//	@Description	Checks the current password, then replaces it with the new one. Returns the updated profile.
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			body			body		dto.ChangePasswordReq	true	"Current and new password"
//	@Param			Accept-Language	header		string					false	"Language of error messages"	Enums(en, km, ms)
//	@Success		200				{object}	response.SuccessRes{data=dto.ProfileRes}
//	@Failure		400				{object}	response.ErrorRes	"validation error or INCORRECT_PASSWORD"
//	@Failure		401				{object}	response.ErrorRes	"UNAUTHORIZED or TOKEN_EXPIRED"
//	@Failure		403				{object}	response.ErrorRes	"ACCOUNT_DISABLED"
//	@Failure		404				{object}	response.ErrorRes	"RECORD_NOT_FOUND"
//	@Failure		413				{object}	response.ErrorRes	"REQUEST_TOO_LARGE"
//	@Failure		429				{object}	response.ErrorRes	"TOO_MANY_REQUESTS"
//	@Failure		500				{object}	response.ErrorRes	"INTERNAL_SERVER_ERROR"
//	@Router			/v1/auth/change-password [put]
func (h *UserHandler) ChangePassword(c *gin.Context) {
	var req dto.ChangePasswordReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.JsonErrRes(c, err, http.StatusBadRequest)
		return
	}
	req.UserID = middleware.CurrentUserID(c)

	res := h.userSvc.ChangePassword(c.Request.Context(), &req)
	if res.Error != nil {
		response.JsonErrRes(c, res.Error, res.StatusCode)
		return
	}

	response.JsonSuccessRes(c, res.Data, res.StatusCode)
}
