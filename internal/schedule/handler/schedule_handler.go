package handler

import (
	"net/http"
	"strconv"

	"github.com/Fazann/notedbook-backend/internal/schedule/dto"
	"github.com/Fazann/notedbook-backend/internal/schedule/service"
	"github.com/Fazann/notedbook-backend/pkg/i18n"
	"github.com/Fazann/notedbook-backend/pkg/response"
	"github.com/Fazann/notedbook-backend/shared/middleware"
	"github.com/gin-gonic/gin"
)

type ScheduleHandler struct {
	scheduleSvc service.ScheduleServiceImpl
}

func NewScheduleHandler() *ScheduleHandler {
	return &ScheduleHandler{
		scheduleSvc: service.NewScheduleSvc(),
	}
}

// ListOccurrences godoc
//
//	@Summary		List occurrences in a date range
//	@Description	Expands every activity (including repeating series) into its concrete occurrences in [from, to], sorted by date then start time. The range is inclusive and at most 42 days.
//	@Tags			schedule
//	@Produce		json
//	@Security		BearerAuth
//	@Param			from			query		string	true	"Start date (YYYY-MM-DD)"
//	@Param			to				query		string	true	"End date (YYYY-MM-DD)"
//	@Param			Accept-Language	header		string	false	"Language of errors"	Enums(en, km, ms)
//	@Success		200				{object}	response.SuccessRes{data=[]dto.OccurrenceRes}
//	@Router			/v1/schedule/occurrences [get]
func (h *ScheduleHandler) ListOccurrences(c *gin.Context) {
	var req dto.OccurrenceListReq
	if err := c.ShouldBindQuery(&req); err != nil {
		response.JsonErrRes(c, err, http.StatusBadRequest)
		return
	}

	res := h.scheduleSvc.ListOccurrences(c.Request.Context(), middleware.CurrentUserID(c), &req)
	if res.Error != nil {
		response.JsonErrRes(c, res.Error, res.StatusCode)
		return
	}

	response.JsonSuccessRes(c, res.Data, res.StatusCode)
}

// GetWeekSummary godoc
//
//	@Summary		Summarize planned time in a date range
//	@Description	Total minutes, minutes by activity type, and occurrence count in [from, to]. The range is inclusive and at most 42 days.
//	@Tags			schedule
//	@Produce		json
//	@Security		BearerAuth
//	@Param			from			query		string	true	"Start date (YYYY-MM-DD)"
//	@Param			to				query		string	true	"End date (YYYY-MM-DD)"
//	@Param			Accept-Language	header		string	false	"Language of errors"	Enums(en, km, ms)
//	@Success		200				{object}	response.SuccessRes{data=dto.WeekSummaryRes}
//	@Router			/v1/schedule/summary [get]
func (h *ScheduleHandler) GetWeekSummary(c *gin.Context) {
	var req dto.OccurrenceListReq
	if err := c.ShouldBindQuery(&req); err != nil {
		response.JsonErrRes(c, err, http.StatusBadRequest)
		return
	}

	res := h.scheduleSvc.GetWeekSummary(c.Request.Context(), middleware.CurrentUserID(c), &req)
	if res.Error != nil {
		response.JsonErrRes(c, res.Error, res.StatusCode)
		return
	}

	response.JsonSuccessRes(c, res.Data, res.StatusCode)
}

// GetActivity godoc
//
//	@Summary		Get an activity
//	@Description	Returns one of the signed-in user's activities (a single event or a repeating series).
//	@Tags			schedule
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id				path		int		true	"Activity ID"			minimum(1)
//	@Param			Accept-Language	header		string	false	"Language of errors"	Enums(en, km, ms)
//	@Success		200				{object}	response.SuccessRes{data=dto.ScheduleActivityRes}
//	@Router			/v1/schedule/activities/{id} [get]
func (h *ScheduleHandler) GetActivity(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	if id <= 0 {
		response.JsonErrRes(c, i18n.InvalidID, http.StatusBadRequest)
		return
	}

	res := h.scheduleSvc.GetActivity(c.Request.Context(), middleware.CurrentUserID(c), uint(id))
	if res.Error != nil {
		response.JsonErrRes(c, res.Error, res.StatusCode)
		return
	}

	response.JsonSuccessRes(c, res.Data, res.StatusCode)
}

// CreateActivity godoc
//
//	@Summary		Create an activity
//	@Description	Creates a single event, or the start of a repeating series when recurrence.kind is not NONE.
//	@Tags			schedule
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			body			body		dto.ScheduleActivityReq	true	"Activity"
//	@Param			Accept-Language	header		string					false	"Language of errors"	Enums(en, km, ms)
//	@Success		201				{object}	response.SuccessRes{data=dto.ScheduleActivityRes}
//	@Router			/v1/schedule/activities [post]
func (h *ScheduleHandler) CreateActivity(c *gin.Context) {
	var req dto.ScheduleActivityReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.JsonErrRes(c, err, http.StatusBadRequest)
		return
	}

	res := h.scheduleSvc.CreateActivity(c.Request.Context(), middleware.CurrentUserID(c), &req)
	if res.Error != nil {
		response.JsonErrRes(c, res.Error, res.StatusCode)
		return
	}

	response.JsonSuccessRes(c, res.Data, res.StatusCode)
}

// UpdateActivity godoc
//
//	@Summary		Update an activity
//	@Description	scope=all replaces the whole series and returns it. scope=this (with date) keeps the series, adds date to its
//	@Description	exceptions, and creates a new single activity with the changes for that one occurrence; returns { series, created }.
//	@Tags			schedule
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id				path		int						true	"Activity ID"	minimum(1)
//	@Param			scope			query		string					true	"all or this"	Enums(all, this)
//	@Param			date			query		string					false	"Occurrence date (YYYY-MM-DD); required when scope is this"
//	@Param			body			body		dto.ScheduleActivityReq	true	"Activity"
//	@Param			Accept-Language	header		string					false	"Language of errors"	Enums(en, km, ms)
//	@Success		200				{object}	response.SuccessRes{data=dto.ScheduleActivityRes}
//	@Router			/v1/schedule/activities/{id} [put]
func (h *ScheduleHandler) UpdateActivity(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	if id <= 0 {
		response.JsonErrRes(c, i18n.InvalidID, http.StatusBadRequest)
		return
	}

	var scope dto.ScheduleActivityScopeReq
	if err := c.ShouldBindQuery(&scope); err != nil {
		response.JsonErrRes(c, err, http.StatusBadRequest)
		return
	}

	var req dto.ScheduleActivityReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.JsonErrRes(c, err, http.StatusBadRequest)
		return
	}

	res := h.scheduleSvc.UpdateActivity(c.Request.Context(), middleware.CurrentUserID(c), uint(id), &scope, &req)
	if res.Error != nil {
		response.JsonErrRes(c, res.Error, res.StatusCode)
		return
	}

	response.JsonSuccessRes(c, res.Data, res.StatusCode)
}

// DeleteActivity godoc
//
//	@Summary		Delete an activity
//	@Description	scope=all soft-deletes the whole series. scope=this (with date) keeps the series and only adds date to its exceptions.
//	@Tags			schedule
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id				path		int		true	"Activity ID"	minimum(1)
//	@Param			scope			query		string	true	"all or this"	Enums(all, this)
//	@Param			date			query		string	false	"Occurrence date (YYYY-MM-DD); required when scope is this"
//	@Param			Accept-Language	header		string	false	"Language of errors"	Enums(en, km, ms)
//	@Success		200				{object}	response.SuccessRes
//	@Router			/v1/schedule/activities/{id} [delete]
func (h *ScheduleHandler) DeleteActivity(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	if id <= 0 {
		response.JsonErrRes(c, i18n.InvalidID, http.StatusBadRequest)
		return
	}

	var scope dto.ScheduleActivityScopeReq
	if err := c.ShouldBindQuery(&scope); err != nil {
		response.JsonErrRes(c, err, http.StatusBadRequest)
		return
	}

	res := h.scheduleSvc.DeleteActivity(c.Request.Context(), middleware.CurrentUserID(c), uint(id), &scope)
	if res.Error != nil {
		response.JsonErrRes(c, res.Error, res.StatusCode)
		return
	}

	response.JsonSuccessRes(c, res.Data, res.StatusCode)
}
