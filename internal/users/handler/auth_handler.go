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
//	@Description	Limited to 5 attempts per device and IP; after that it is blocked for 5 minutes.
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			body			body		dto.RegisterReq	true	"Account details"
//	@Param			Accept-Language	header		string			false	"Language of error messages"	Enums(en, km, ms)
//	@Success		201				{object}	response.SuccessRes{data=dto.TokenRes}
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
//	@Description	The device is marked as logged in for this user (created on first login) and any other user is signed out of it.
//	@Description	Limited to 5 attempts per device and IP; after that it is blocked for 5 minutes. A successful login resets the count.
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			body			body		dto.LoginReq	true	"Credentials"
//	@Param			Accept-Language	header		string			false	"Language of error messages"	Enums(en, km, ms)
//	@Success		200				{object}	response.SuccessRes{data=dto.TokenRes}
//	@Router			/v1/auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var req dto.LoginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.JsonErrRes(c, err, http.StatusBadRequest)
		return
	}

	req.IP = c.ClientIP()
	req.Device.UserAgent = c.Request.UserAgent()
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
