package handler

import (
	"github.com/Fazann/notedbook-backend/internal/users/service"
	"github.com/Fazann/notedbook-backend/pkg/response"
	"github.com/Fazann/notedbook-backend/shared/middleware"
	"github.com/gin-gonic/gin"
)

type UserHandler struct {
	userSvc service.UserServiceImpl
}

func NewUserHandler() *UserHandler {
	return &UserHandler{
		userSvc: service.NewUserSvc(),
	}
}

// GetProfile godoc
//
//	@Summary		Get my profile
//	@Description	Returns the signed-in user's profile, with the avatar's full URL when set.
//	@Tags			users
//	@Produce		json
//	@Security		BearerAuth
//	@Param			Accept-Language	header		string	false	"Language of error messages"	Enums(en, km, ms)
//	@Success		200				{object}	response.SuccessRes{data=dto.ProfileRes}
//	@Failure		401				{object}	response.ErrorRes	"UNAUTHORIZED or TOKEN_EXPIRED"
//	@Failure		403				{object}	response.ErrorRes	"ACCOUNT_DISABLED"
//	@Failure		404				{object}	response.ErrorRes	"RECORD_NOT_FOUND"
//	@Failure		500				{object}	response.ErrorRes	"INTERNAL_SERVER_ERROR"
//	@Router			/v1/users/me [get]
func (h *UserHandler) GetProfile(c *gin.Context) {
	res := h.userSvc.GetProfile(c.Request.Context(), middleware.CurrentUserID(c))
	if res.Error != nil {
		response.JsonErrRes(c, res.Error, res.StatusCode)
		return
	}

	response.JsonSuccessRes(c, res.Data, res.StatusCode)
}
