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
//	@Router			/v1/plan-areas [get]
func (h *PlanAreaHandler) List(c *gin.Context) {
	res := h.planAreaSvc.List(c.Request.Context(), i18n.FromRequest(c.Request))
	if res.Error != nil {
		response.JsonErrRes(c, res.Error, res.StatusCode)
		return
	}

	response.JsonSuccessRes(c, res.Data, res.StatusCode)
}
