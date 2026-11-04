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
//	@Failure		413				{object}	response.ErrorRes	"REQUEST_TOO_LARGE"
//	@Failure		429				{object}	response.ErrorRes	"TOO_MANY_REQUESTS"
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
//	@Failure		413				{object}	response.ErrorRes	"REQUEST_TOO_LARGE"
//	@Failure		429				{object}	response.ErrorRes	"TOO_MANY_REQUESTS"
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
//	@Failure		413				{object}	response.ErrorRes	"REQUEST_TOO_LARGE"
//	@Failure		429				{object}	response.ErrorRes	"TOO_MANY_REQUESTS"
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

// RequestMailOTP godoc
//
//	@Summary		Request a password reset code
//	@Description	Emails a 6-digit code to the address if it belongs to an active account. The response is the same
//	@Description	whether the email is registered or not. One request per email per minute; a new code replaces the old one.
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			body			body		dto.MailOTPRequestReq	true	"Account email"
//	@Param			Accept-Language	header		string					false	"Language of error messages"	Enums(en, km, ms)
//	@Success		200				{object}	response.SuccessRes{data=dto.MailOTPRequestRes}
//	@Failure		413				{object}	response.ErrorRes	"REQUEST_TOO_LARGE"
//	@Failure		429				{object}	response.ErrorRes	"TOO_MANY_REQUESTS"
//	@Router			/v1/auth/mail-otp/request [post]
func (h *AuthHandler) RequestMailOTP(c *gin.Context) {
	var req dto.MailOTPRequestReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.JsonErrRes(c, err, http.StatusBadRequest)
		return
	}

	res := h.authSvc.RequestMailOTP(c.Request.Context(), &req)
	if res.Error != nil {
		response.JsonErrRes(c, res.Error, res.StatusCode)
		return
	}

	response.JsonSuccessRes(c, res.Data, res.StatusCode)
}

// VerifyMailOTP godoc
//
//	@Summary		Verify a password reset code
//	@Description	Checks the emailed code and returns a single-use key for /auth/reset-password, valid for 10 minutes.
//	@Description	After 5 wrong codes the code is cancelled and a new one must be requested.
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			body			body		dto.MailOTPVerifyReq	true	"Email and code"
//	@Param			Accept-Language	header		string					false	"Language of error messages"	Enums(en, km, ms)
//	@Success		200				{object}	response.SuccessRes{data=dto.MailOTPVerifyRes}
//	@Failure		413				{object}	response.ErrorRes	"REQUEST_TOO_LARGE"
//	@Failure		429				{object}	response.ErrorRes	"TOO_MANY_REQUESTS"
//	@Router			/v1/auth/mail-otp/verify [post]
func (h *AuthHandler) VerifyMailOTP(c *gin.Context) {
	var req dto.MailOTPVerifyReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.JsonErrRes(c, err, http.StatusBadRequest)
		return
	}

	res := h.authSvc.VerifyMailOTP(c.Request.Context(), &req)
	if res.Error != nil {
		response.JsonErrRes(c, res.Error, res.StatusCode)
		return
	}

	response.JsonSuccessRes(c, res.Data, res.StatusCode)
}

// ResetPassword godoc
//
//	@Summary		Reset the password
//	@Description	Sets a new password with the key from /auth/mail-otp/verify. The key works once.
//	@Description	Every session of the user is signed out.
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			body			body		dto.ResetPasswordReq	true	"Key and new password"
//	@Param			Accept-Language	header		string					false	"Language of error messages"	Enums(en, km, ms)
//	@Success		200				{object}	response.SuccessRes
//	@Failure		413				{object}	response.ErrorRes	"REQUEST_TOO_LARGE"
//	@Failure		429				{object}	response.ErrorRes	"TOO_MANY_REQUESTS"
//	@Router			/v1/auth/reset-password [post]
func (h *AuthHandler) ResetPassword(c *gin.Context) {
	var req dto.ResetPasswordReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.JsonErrRes(c, err, http.StatusBadRequest)
		return
	}

	res := h.authSvc.ResetPassword(c.Request.Context(), &req)
	if res.Error != nil {
		response.JsonErrRes(c, res.Error, res.StatusCode)
		return
	}

	response.JsonSuccessRes(c, res.Data, res.StatusCode)
}
