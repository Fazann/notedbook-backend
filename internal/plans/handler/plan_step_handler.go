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

type PlanStepHandler struct {
	planStepSvc service.PlanStepServiceImpl
}

func NewPlanStepHandler() *PlanStepHandler {
	return &PlanStepHandler{
		planStepSvc: service.NewPlanStepSvc(),
	}
}

// Create godoc
//
//	@Summary		Add a step to a plan
//	@Description	Adds a step at the end of the plan and increments its step_count.
//	@Tags			plan-steps
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id				path		int				true	"Plan ID"	minimum(1)
//	@Param			body			body		dto.PlanStepReq	true	"Step"
//	@Param			Accept-Language	header		string			false	"Language of errors"	Enums(en, km, ms)
//	@Success		201				{object}	response.SuccessRes{data=dto.PlanStepRes}
//	@Router			/v1/plans/{id}/steps [post]
func (h *PlanStepHandler) Create(c *gin.Context) {
	planID, _ := strconv.Atoi(c.Param("id"))
	if planID <= 0 {
		response.JsonErrRes(c, i18n.InvalidID, http.StatusBadRequest)
		return
	}

	var req dto.PlanStepReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.JsonErrRes(c, err, http.StatusBadRequest)
		return
	}

	res := h.planStepSvc.Create(c.Request.Context(), middleware.CurrentUserID(c), uint(planID), &req)
	if res.Error != nil {
		response.JsonErrRes(c, res.Error, res.StatusCode)
		return
	}

	response.JsonSuccessRes(c, res.Data, res.StatusCode)
}

// Update godoc
//
//	@Summary		Update a plan step
//	@Description	Partial update: only the fields sent are changed. Does not affect step_count or done_step_count.
//	@Tags			plan-steps
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id				path		int						true	"Step ID"	minimum(1)
//	@Param			body			body		dto.PlanStepUpdateReq	true	"Fields to change"
//	@Param			Accept-Language	header		string					false	"Language of errors"	Enums(en, km, ms)
//	@Success		200				{object}	response.SuccessRes{data=dto.PlanStepRes}
//	@Router			/v1/steps/{id} [put]
func (h *PlanStepHandler) Update(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	if id <= 0 {
		response.JsonErrRes(c, i18n.InvalidID, http.StatusBadRequest)
		return
	}

	var req dto.PlanStepUpdateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.JsonErrRes(c, err, http.StatusBadRequest)
		return
	}

	res := h.planStepSvc.Update(c.Request.Context(), middleware.CurrentUserID(c), uint(id), &req)
	if res.Error != nil {
		response.JsonErrRes(c, res.Error, res.StatusCode)
		return
	}

	response.JsonSuccessRes(c, res.Data, res.StatusCode)
}

// Toggle godoc
//
//	@Summary		Check or uncheck a plan step
//	@Description	Keeps the plan's done_step_count in sync. Checking the first step of a NOT_START plan moves it to
//	@Description	IN_PROGRESS; unchecking a step on a DONE plan moves it back to IN_PROGRESS. Returns the step and the plan.
//	@Tags			plan-steps
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id				path		int						true	"Step ID"	minimum(1)
//	@Param			body			body		dto.PlanStepToggleReq	true	"Done state"
//	@Param			Accept-Language	header		string					false	"Language of the area title and errors"	Enums(en, km, ms)
//	@Success		200				{object}	response.SuccessRes{data=dto.PlanStepToggleRes}
//	@Router			/v1/steps/{id}/toggle [patch]
func (h *PlanStepHandler) Toggle(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	if id <= 0 {
		response.JsonErrRes(c, i18n.InvalidID, http.StatusBadRequest)
		return
	}

	var req dto.PlanStepToggleReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.JsonErrRes(c, err, http.StatusBadRequest)
		return
	}

	res := h.planStepSvc.Toggle(c.Request.Context(), middleware.CurrentUserID(c), uint(id), i18n.FromRequest(c.Request), &req)
	if res.Error != nil {
		response.JsonErrRes(c, res.Error, res.StatusCode)
		return
	}

	response.JsonSuccessRes(c, res.Data, res.StatusCode)
}

// Delete godoc
//
//	@Summary		Delete a plan step
//	@Description	Decrements the plan's step_count (and done_step_count if it was done). Returns the updated plan.
//	@Tags			plan-steps
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id				path		int		true	"Step ID"								minimum(1)
//	@Param			Accept-Language	header		string	false	"Language of the area title and errors"	Enums(en, km, ms)
//	@Success		200				{object}	response.SuccessRes{data=dto.PlanStepDeleteRes}
//	@Router			/v1/steps/{id} [delete]
func (h *PlanStepHandler) Delete(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	if id <= 0 {
		response.JsonErrRes(c, i18n.InvalidID, http.StatusBadRequest)
		return
	}

	res := h.planStepSvc.Delete(c.Request.Context(), middleware.CurrentUserID(c), uint(id), i18n.FromRequest(c.Request))
	if res.Error != nil {
		response.JsonErrRes(c, res.Error, res.StatusCode)
		return
	}

	response.JsonSuccessRes(c, res.Data, res.StatusCode)
}
