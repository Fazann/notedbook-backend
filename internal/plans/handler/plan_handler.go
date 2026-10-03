package handler

import (
	"net/http"
	"strconv"

	"github.com/Fazann/notedbook-backend/internal/plans/dto"
	"github.com/Fazann/notedbook-backend/internal/plans/service"
	"github.com/Fazann/notedbook-backend/pkg/i18n"
	"github.com/Fazann/notedbook-backend/pkg/response"
	"github.com/Fazann/notedbook-backend/shared/middleware"
	"github.com/gin-gonic/gin"
)

type PlanHandler struct {
	planSvc service.PlanServiceImpl
}

func NewPlanHandler() *PlanHandler {
	return &PlanHandler{
		planSvc: service.NewPlanSvc(),
	}
}

// List godoc
//
//	@Summary		List my plans
//	@Description	One page of the signed-in user's plans. `q` matches the title.
//	@Tags			plans
//	@Produce		json
//	@Security		BearerAuth
//	@Param			page			query		int		false	"Page (1-based)"						minimum(1)	default(1)
//	@Param			limit			query		int		false	"Rows per page"							minimum(1)	maximum(1000)	default(10)
//	@Param			status			query		string	false	"Status"								Enums(NOT_START, IN_PROGRESS, DONE)
//	@Param			area_id			query		int		false	"Plan area ID"							minimum(1)
//	@Param			q				query		string	false	"Search in the title"					maxlength(100)
//	@Param			sort			query		string	false	"Sort (\"-\" = descending)"				Enums(target_date, -target_date, created_at, -created_at, title, -title)	default(target_date)
//	@Param			Accept-Language	header		string	false	"Language of the area title and errors"	Enums(en, km, ms)
//	@Success		200				{object}	response.Paginate{data=[]dto.PlanRes}
//	@Router			/v1/plans [get]
func (h *PlanHandler) List(c *gin.Context) {
	var req dto.PlanListReq
	if err := c.ShouldBindQuery(&req); err != nil {
		response.JsonErrRes(c, err, http.StatusBadRequest)
		return
	}

	res := h.planSvc.List(c.Request.Context(), middleware.CurrentUserID(c), i18n.FromRequest(c.Request), &req)
	if res.Error != nil {
		response.JsonErrRes(c, res.Error, res.StatusCode)
		return
	}

	response.JsonSuccessResWithPage(c, res)
}

// GetDetail godoc
//
//	@Summary		Get a plan
//	@Description	Returns one of the signed-in user's plans, with its steps.
//	@Tags			plans
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id				path		int		true	"Plan ID"								minimum(1)
//	@Param			Accept-Language	header		string	false	"Language of the area title and errors"	Enums(en, km, ms)
//	@Success		200				{object}	response.SuccessRes{data=dto.PlanDetailRes}
//	@Router			/v1/plans/{id} [get]
func (h *PlanHandler) GetDetail(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	if id <= 0 {
		response.JsonErrRes(c, i18n.InvalidID, http.StatusBadRequest)
		return
	}

	res := h.planSvc.GetDetail(c.Request.Context(), middleware.CurrentUserID(c), uint(id), i18n.FromRequest(c.Request))
	if res.Error != nil {
		response.JsonErrRes(c, res.Error, res.StatusCode)
		return
	}

	response.JsonSuccessRes(c, res.Data, res.StatusCode)
}

// Create godoc
//
//	@Summary		Create a plan
//	@Description	area_id must be one of the plan areas (see GET /plan-areas). Status starts at NOT_START.
//	@Tags			plans
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			body			body		dto.PlanReq	true	"Plan"
//	@Param			Accept-Language	header		string		false	"Language of the area title and errors"	Enums(en, km, ms)
//	@Success		201				{object}	response.SuccessRes{data=dto.PlanRes}
//	@Router			/v1/plans [post]
func (h *PlanHandler) Create(c *gin.Context) {
	var req dto.PlanReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.JsonErrRes(c, err, http.StatusBadRequest)
		return
	}

	res := h.planSvc.Create(c.Request.Context(), middleware.CurrentUserID(c), i18n.FromRequest(c.Request), &req)
	if res.Error != nil {
		response.JsonErrRes(c, res.Error, res.StatusCode)
		return
	}

	response.JsonSuccessRes(c, res.Data, res.StatusCode)
}

// Update godoc
//
//	@Summary		Update a plan
//	@Description	Partial update: only the fields sent are changed.
//	@Tags			plans
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id				path		int					true	"Plan ID"	minimum(1)
//	@Param			body			body		dto.PlanUpdateReq	true	"Fields to change"
//	@Param			Accept-Language	header		string				false	"Language of the area title and errors"	Enums(en, km, ms)
//	@Success		200				{object}	response.SuccessRes{data=dto.PlanDetailRes}
//	@Router			/v1/plans/{id} [put]
func (h *PlanHandler) Update(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	if id <= 0 {
		response.JsonErrRes(c, i18n.InvalidID, http.StatusBadRequest)
		return
	}

	var req dto.PlanUpdateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.JsonErrRes(c, err, http.StatusBadRequest)
		return
	}

	res := h.planSvc.Update(c.Request.Context(), middleware.CurrentUserID(c), uint(id), i18n.FromRequest(c.Request), &req)
	if res.Error != nil {
		response.JsonErrRes(c, res.Error, res.StatusCode)
		return
	}

	response.JsonSuccessRes(c, res.Data, res.StatusCode)
}

// UpdateStatus godoc
//
//	@Summary		Change a plan's status
//	@Description	Sets the status directly; it does not check or uncheck steps. Setting DONE sets done_at; leaving DONE clears it.
//	@Tags			plans
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id				path		int					true	"Plan ID"	minimum(1)
//	@Param			body			body		dto.PlanStatusReq	true	"New status"
//	@Param			Accept-Language	header		string				false	"Language of the area title and errors"	Enums(en, km, ms)
//	@Success		200				{object}	response.SuccessRes{data=dto.PlanDetailRes}
//	@Router			/v1/plans/{id}/status [patch]
func (h *PlanHandler) UpdateStatus(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	if id <= 0 {
		response.JsonErrRes(c, i18n.InvalidID, http.StatusBadRequest)
		return
	}

	var req dto.PlanStatusReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.JsonErrRes(c, err, http.StatusBadRequest)
		return
	}

	res := h.planSvc.UpdateStatus(c.Request.Context(), middleware.CurrentUserID(c), uint(id), i18n.FromRequest(c.Request), &req)
	if res.Error != nil {
		response.JsonErrRes(c, res.Error, res.StatusCode)
		return
	}

	response.JsonSuccessRes(c, res.Data, res.StatusCode)
}

// Delete godoc
//
//	@Summary		Delete a plan
//	@Description	Soft-deletes one of the signed-in user's plans; its steps are deleted too.
//	@Tags			plans
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id				path		int		true	"Plan ID"				minimum(1)
//	@Param			Accept-Language	header		string	false	"Language of errors"	Enums(en, km, ms)
//	@Success		200				{object}	response.SuccessRes
//	@Router			/v1/plans/{id} [delete]
func (h *PlanHandler) Delete(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	if id <= 0 {
		response.JsonErrRes(c, i18n.InvalidID, http.StatusBadRequest)
		return
	}

	res := h.planSvc.Delete(c.Request.Context(), middleware.CurrentUserID(c), uint(id))
	if res.Error != nil {
		response.JsonErrRes(c, res.Error, res.StatusCode)
		return
	}

	response.JsonSuccessRes(c, res.Data, res.StatusCode)
}
