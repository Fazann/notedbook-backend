package handler

import (
	"net/http"

	"github.com/Fazann/notedbook-backend/internal/users/dto"
	"github.com/Fazann/notedbook-backend/internal/users/service"
	"github.com/Fazann/notedbook-backend/pkg/response"
	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	authSvc service.AuthServiceImpl
}

func NewAuthHandler() *AuthHandler {
	return &AuthHandler{
		authSvc: service.NewAuthSvc(),
	}
}

// Register godoc
//
//	@Summary		Register a new user
//	@Description	Creates an account and signs the user in, returning an access and a refresh token.
//	@Description	Limited to 5 attempts per email and IP; after that it is blocked for 5 minutes.
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			body			body		dto.RegisterReq	true	"Account details"
//	@Param			Accept-Language	header		string			false	"Language of error messages"	Enums(en, km, ms)
//	@Success		201				{object}	response.SuccessRes{data=dto.TokenRes}
//	@Failure		400				{object}	response.ErrorRes	"BAD_REQUEST, INVALID_USERNAME or INVALID_EMAIL"
//	@Failure		409				{object}	response.ErrorRes	"USERNAME_TAKEN or EMAIL_TAKEN"
//	@Failure		429				{object}	response.ErrorRes	"TOO_MANY_ATTEMPTS or TOO_MANY_REQUESTS"
//	@Failure		500				{object}	response.ErrorRes	"INTERNAL_SERVER_ERROR"
//	@Router			/v1/auth/register [post]
func (h *AuthHandler) Register(c *gin.Context) {
	var req dto.RegisterReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.JsonErrRes(c, err, http.StatusBadRequest)
		return
	}
	req.Device.UserAgent = c.Request.UserAgent()

	res := h.authSvc.Register(c.Request.Context(), &req, c.ClientIP())
	if res.Error != nil {
		response.JsonErrRes(c, res.Error, res.StatusCode)
		return
	}

	response.JsonSuccessRes(c, res.Data, res.StatusCode)
}

// Login godoc
//
//	@Summary		Sign in
//	@Description	Checks the username and password and starts a new session, returning an access and a refresh token.
//	@Description	Limited to 5 attempts per username; after that it is blocked for 5 minutes. A successful login resets the count.
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			body			body		dto.LoginReq	true	"Credentials"
//	@Param			Accept-Language	header		string			false	"Language of error messages"	Enums(en, km, ms)
//	@Success		200				{object}	response.SuccessRes{data=dto.TokenRes}
//	@Failure		400				{object}	response.ErrorRes	"BAD_REQUEST"
//	@Failure		401				{object}	response.ErrorRes	"INVALID_CREDENTIALS"
//	@Failure		403				{object}	response.ErrorRes	"ACCOUNT_DISABLED"
//	@Failure		429				{object}	response.ErrorRes	"TOO_MANY_ATTEMPTS or TOO_MANY_REQUESTS"
//	@Failure		500				{object}	response.ErrorRes	"INTERNAL_SERVER_ERROR"
//	@Router			/v1/auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var req dto.LoginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.JsonErrRes(c, err, http.StatusBadRequest)
		return
	}

	res := h.authSvc.Login(c.Request.Context(), &req)
	if res.Error != nil {
		response.JsonErrRes(c, res.Error, res.StatusCode)
		return
	}

	response.JsonSuccessRes(c, res.Data, res.StatusCode)
}

// RefreshToken godoc
//
//	@Summary		Refresh tokens
//	@Description	Exchanges a valid refresh token for a new access and refresh token. The old refresh token is
//	@Description	revoked; using a revoked one again signs the user out of every session.
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			body			body		dto.RefreshTokenReq	true	"Refresh token"
//	@Param			Accept-Language	header		string				false	"Language of error messages"	Enums(en, km, ms)
//	@Success		200				{object}	response.SuccessRes{data=dto.TokenRes}
//	@Failure		400				{object}	response.ErrorRes	"BAD_REQUEST"
//	@Failure		401				{object}	response.ErrorRes	"UNAUTHORIZED or TOKEN_EXPIRED"
//	@Failure		403				{object}	response.ErrorRes	"ACCOUNT_DISABLED"
//	@Failure		429				{object}	response.ErrorRes	"TOO_MANY_REQUESTS"
//	@Failure		500				{object}	response.ErrorRes	"INTERNAL_SERVER_ERROR"
//	@Router			/v1/auth/refresh-token [post]
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	var req dto.RefreshTokenReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.JsonErrRes(c, err, http.StatusBadRequest)
		return
	}

	res := h.authSvc.RefreshToken(c.Request.Context(), &req)
	if res.Error != nil {
		response.JsonErrRes(c, res.Error, res.StatusCode)
		return
	}

	response.JsonSuccessRes(c, res.Data, res.StatusCode)
}
