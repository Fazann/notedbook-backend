package handler

import (
	"net/http"

	"github.com/Fazann/notedbook-backend/internal/exspense/dto"
	"github.com/Fazann/notedbook-backend/internal/exspense/service"
	"github.com/Fazann/notedbook-backend/pkg/response"
	"github.com/Fazann/notedbook-backend/shared/middleware"
	"github.com/gin-gonic/gin"
)

type CurrencySettingHandler struct {
	currencySettingSvc service.CurrencySettingServiceImpl
}

func NewCurrencySettingHandler() *CurrencySettingHandler {
	return &CurrencySettingHandler{
		currencySettingSvc: service.NewCurrencySettingSvc(),
	}
}

// Get godoc
//
//	@Summary		Get my currency settings
//	@Description	The currencies the signed-in user records expenses in and the default one for new expenses.
//	@Description	A user who never saved settings gets every currency with USD as the default. Served from a cache.
//	@Tags			currency-settings
//	@Produce		json
//	@Security		BearerAuth
//	@Param			Accept-Language	header		string	false	"Language of errors"	Enums(en, km, ms)
//	@Success		200				{object}	response.SuccessRes{data=dto.CurrencySettingRes}
//	@Router			/v1/currency-settings [get]
func (h *CurrencySettingHandler) Get(c *gin.Context) {
	res := h.currencySettingSvc.Get(c.Request.Context(), middleware.CurrentUserID(c))
	if res.Error != nil {
		response.JsonErrRes(c, res.Error, res.StatusCode)
		return
	}
	response.JsonSuccessRes(c, res.Data, res.StatusCode)
}

// Update godoc
//
//	@Summary		Save my currency settings
//	@Description	Replaces the enabled currencies and the default one, and clears the cached copy.
//	@Description	Existing expenses and budgets in a currency that is turned off are kept.
//	@Tags			currency-settings
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			body			body		dto.CurrencySettingReq	true	"Currency settings"
//	@Param			Accept-Language	header		string					false	"Language of errors"	Enums(en, km, ms)
//	@Success		200				{object}	response.SuccessRes{data=dto.CurrencySettingRes}
//	@Failure		400				{object}	response.ErrorRes	"VALIDATION_FAILED, INVALID_CURRENCY, NO_CURRENCY_ENABLED or DEFAULT_CURRENCY_NOT_ENABLED"
//	@Router			/v1/currency-settings [put]
func (h *CurrencySettingHandler) Update(c *gin.Context) {
	var req dto.CurrencySettingReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.JsonErrRes(c, err, http.StatusBadRequest)
		return
	}
	res := h.currencySettingSvc.Update(c.Request.Context(), middleware.CurrentUserID(c), &req)
	if res.Error != nil {
		response.JsonErrRes(c, res.Error, res.StatusCode)
		return
	}
	response.JsonSuccessRes(c, res.Data, res.StatusCode)
}
