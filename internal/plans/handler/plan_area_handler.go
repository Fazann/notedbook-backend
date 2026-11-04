package handler

import (
	"github.com/Fazann/notedbook-backend/internal/plans/service"
	"github.com/Fazann/notedbook-backend/pkg/i18n"
	"github.com/Fazann/notedbook-backend/pkg/response"
	"github.com/gin-gonic/gin"
)

type PlanAreaHandler struct {
	planAreaSvc service.PlanAreaServiceImpl
}

func NewPlanAreaHandler() *PlanAreaHandler {
	return &PlanAreaHandler{
		planAreaSvc: service.NewPlanAreaSvc(),
	}
}

// List godoc
//
//	@Summary		List plan areas
//	@Description	Returns every plan area (e.g. Personal, Career, Finance), with its icon and translations. display_title follows the request language.
//	@Tags			plan-areas
//	@Produce		json
//	@Security		BearerAuth
//	@Param			Accept-Language	header		string	false	"Language of titles and errors"	Enums(en, km, ms)
//	@Success		200				{object}	response.SuccessRes{data=[]dto.PlanAreaRes}
//	@Failure		429				{object}	response.ErrorRes	"TOO_MANY_REQUESTS"
//	@Router			/v1/plan-areas [get]
func (h *PlanAreaHandler) List(c *gin.Context) {
	res := h.planAreaSvc.List(c.Request.Context(), i18n.FromRequest(c.Request))
	if res.Error != nil {
		response.JsonErrRes(c, res.Error, res.StatusCode)
		return
	}

	response.JsonSuccessRes(c, res.Data, res.StatusCode)
}

// ResetCache godoc
//
//	@Summary		Reset the plan area cache
//	@Description	Plan areas are cached for a day. This drops the cache so the next list reads them from the database.
//	@Tags			plan-areas
//	@Produce		json
//	@Security		BearerAuth
//	@Param			Accept-Language	header		string	false	"Language of errors"	Enums(en, km, ms)
//	@Success		200				{object}	response.SuccessRes
//	@Failure		429				{object}	response.ErrorRes	"TOO_MANY_REQUESTS"
//	@Failure		500				{object}	response.ErrorRes	"INTERNAL_SERVER_ERROR"
//	@Router			/v1/plan-areas/cache [delete]
func (h *PlanAreaHandler) ResetCache(c *gin.Context) {
	res := h.planAreaSvc.ResetCache(c.Request.Context())
	if res.Error != nil {
		response.JsonErrRes(c, res.Error, res.StatusCode)
		return
	}

	response.JsonSuccessRes(c, res.Data, res.StatusCode)
}
