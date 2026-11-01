package handler

import (
	"net/http"

	"github.com/Fazann/notedbook-backend/internal/calendar/dto"
	"github.com/Fazann/notedbook-backend/internal/calendar/service"
	"github.com/Fazann/notedbook-backend/pkg/response"
	"github.com/gin-gonic/gin"
)

type HolidayHandler struct {
	holidaySvc service.HolidayServiceImpl
}

func NewHolidayHandler() *HolidayHandler {
	return &HolidayHandler{
		holidaySvc: service.NewHolidaySvc(),
	}
}

// List godoc
//
//	@Summary		List holidays of a year
//	@Description	Holidays of one year for a country, sorted by date, one entry per day.
//	@Description	Dates are calculated, so a holiday decided by the government or by moon sighting can differ by a day.
//	@Tags			calendar
//	@Produce		json
//	@Security		BearerAuth
//	@Param			year			query		int		true	"Year"					minimum(1900)	maximum(2100)
//	@Param			country			query		string	true	"Country"				Enums(KH, MY)
//	@Param			kind			query		string	true	"Kind of holiday"		Enums(national, islamic)
//	@Param			Accept-Language	header		string	false	"Language of errors"	Enums(en, km, ms)
//	@Success		200				{object}	response.SuccessRes{data=[]dto.HolidayRes}
//	@Failure		400				{object}	response.ErrorRes	"VALIDATION_FAILED"
//	@Router			/v1/holidays [get]
func (h *HolidayHandler) List(c *gin.Context) {
	var req dto.HolidayListReq
	if err := c.ShouldBindQuery(&req); err != nil {
		response.JsonErrRes(c, err, http.StatusBadRequest)
		return
	}
	res := h.holidaySvc.List(c.Request.Context(), &req)
	if res.Error != nil {
		response.JsonErrRes(c, res.Error, res.StatusCode)
		return
	}
	response.JsonSuccessRes(c, res.Data, res.StatusCode)
}
