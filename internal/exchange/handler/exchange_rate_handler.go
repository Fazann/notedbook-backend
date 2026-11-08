package handler

import (
	"net/http"

	"github.com/Fazann/notedbook-backend/internal/exchange/dto"
	"github.com/Fazann/notedbook-backend/internal/exchange/service"
	"github.com/Fazann/notedbook-backend/pkg/response"
	"github.com/gin-gonic/gin"
)

type ExchangeRateHandler struct {
	exchangeRateSvc service.ExchangeRateServiceImpl
}

func NewExchangeRateHandler() *ExchangeRateHandler {
	return &ExchangeRateHandler{
		exchangeRateSvc: service.NewExchangeRateSvc(),
	}
}

// List godoc
//
//	@Summary		Latest exchange rates
//	@Description	Rates of every supported currency against base (1 base = rates[code] code), from ExchangeRate-API.
//	@Description	Updated once a day and cached in Redis, never stored in the database. While the source is down,
//	@Description	the last fetched rates are returned; updated_at tells how old they are.
//	@Tags			exchange
//	@Produce		json
//	@Security		BearerAuth
//	@Param			base			query		string	false	"ISO 4217 code, default USD"	minLength(3)	maxLength(3)
//	@Param			Accept-Language	header		string	false	"Language of errors"			Enums(en, km, ms)
//	@Success		200				{object}	response.SuccessRes{data=dto.ExchangeRateRes}
//	@Failure		400				{object}	response.ErrorRes	"VALIDATION_FAILED, UNSUPPORTED_CURRENCY"
//	@Failure		429				{object}	response.ErrorRes	"TOO_MANY_REQUESTS"
//	@Failure		503				{object}	response.ErrorRes	"EXCHANGE_RATES_UNAVAILABLE"
//	@Router			/v1/exchange-rates [get]
func (h *ExchangeRateHandler) List(c *gin.Context) {
	var req dto.ExchangeRateListReq
	if err := c.ShouldBindQuery(&req); err != nil {
		response.JsonErrRes(c, err, http.StatusBadRequest)
		return
	}
	res := h.exchangeRateSvc.List(c.Request.Context(), &req)
	if res.Error != nil {
		response.JsonErrRes(c, res.Error, res.StatusCode)
		return
	}
	response.JsonSuccessRes(c, res.Data, res.StatusCode)
}
