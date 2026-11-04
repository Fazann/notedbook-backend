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
//	@Description	One page of the signed-in user's plans. `q` matches the title. `status` and `area_id` take several
//	@Description	comma-separated values. Plans without a target date come last when sorting by target_date.
//	@Tags			plans
//	@Produce		json
//	@Security		BearerAuth
//	@Param			page			query		int		false	"Page (1-based)"	minimum(1)	default(1)
//	@Param			limit			query		int		false	"Rows per page"		minimum(1)	maximum(1000)	default(10)
//	@Param			status			query		string	false	"Statuses, comma separated (NOT_START, IN_PROGRESS, DONE)"
//	@Param			area_id			query		string	false	"Plan area IDs, comma separated"
//	@Param			q				query		string	false	"Search in the title"					maxlength(100)
//	@Param			sort			query		string	false	"Sort (\"-\" = descending)"				Enums(target_date, -target_date, created_at, -created_at, title, -title, progress, -progress)	default(target_date)
//	@Param			Accept-Language	header		string	false	"Language of the area title and errors"	Enums(en, km, ms)
//	@Success		200				{object}	response.Paginate{data=[]dto.PlanRes}
//	@Failure		400				{object}	response.ErrorRes	"VALIDATION_FAILED, FIELD_INVALID (status or area_id)"
//	@Failure		429				{object}	response.ErrorRes	"TOO_MANY_REQUESTS"
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

// Stats godoc
//
//	@Summary		Summarize my plans
//	@Description	Active (NOT_START + IN_PROGRESS) plans, plans done this year, overdue active plans and the average
//	@Description	progress of the active plans. `today` is the user's date; it defaults to today in UTC.
//	@Tags			plans
//	@Produce		json
//	@Security		BearerAuth
//	@Param			today			query		string	false	"Current date (YYYY-MM-DD)"
//	@Param			Accept-Language	header		string	false	"Language of errors"	Enums(en, km, ms)
//	@Success		200				{object}	response.SuccessRes{data=dto.PlanStatsRes}
//	@Failure		400				{object}	response.ErrorRes	"VALIDATION_FAILED (today not YYYY-MM-DD)"
//	@Failure		429				{object}	response.ErrorRes	"TOO_MANY_REQUESTS"
//	@Router			/v1/plans/stats [get]
func (h *PlanHandler) Stats(c *gin.Context) {
	var req dto.PlanStatsReq
	if err := c.ShouldBindQuery(&req); err != nil {
		response.JsonErrRes(c, err, http.StatusBadRequest)
		return
	}

	res := h.planSvc.Stats(c.Request.Context(), middleware.CurrentUserID(c), &req)
	if res.Error != nil {
		response.JsonErrRes(c, res.Error, res.StatusCode)
		return
	}

	response.JsonSuccessRes(c, res.Data, res.StatusCode)
}

// GetDetail godoc
//
//	@Summary		Get a plan
//	@Description	Returns one of the signed-in user's plans, with its steps by position.
//	@Tags			plans
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id				path		int		true	"Plan ID"								minimum(1)
//	@Param			Accept-Language	header		string	false	"Language of the area title and errors"	Enums(en, km, ms)
//	@Success		200				{object}	response.SuccessRes{data=dto.PlanDetailRes}
//	@Failure		429				{object}	response.ErrorRes	"TOO_MANY_REQUESTS"
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
//	@Failure		413				{object}	response.ErrorRes	"REQUEST_TOO_LARGE"
//	@Failure		429				{object}	response.ErrorRes	"TOO_MANY_REQUESTS"
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
//	@Failure		413				{object}	response.ErrorRes	"REQUEST_TOO_LARGE"
//	@Failure		429				{object}	response.ErrorRes	"TOO_MANY_REQUESTS"
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
//	@Failure		413				{object}	response.ErrorRes	"REQUEST_TOO_LARGE"
//	@Failure		429				{object}	response.ErrorRes	"TOO_MANY_REQUESTS"
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
//	@Failure		429				{object}	response.ErrorRes	"TOO_MANY_REQUESTS"
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
