package handler

import (
	"net/http"
	"strconv"

	"github.com/Fazann/notedbook-backend/internal/exspense/dto"
	"github.com/Fazann/notedbook-backend/internal/exspense/service"
	"github.com/Fazann/notedbook-backend/pkg/i18n"
	"github.com/Fazann/notedbook-backend/pkg/response"
	"github.com/Fazann/notedbook-backend/shared/middleware"
	"github.com/gin-gonic/gin"
)

type BudgetHandler struct {
	budgetSvc service.BudgetServiceImpl
}

func NewBudgetHandler() *BudgetHandler {
	return &BudgetHandler{
		budgetSvc: service.NewBudgetSvc(),
	}
}

// List godoc
//
//	@Summary		List my budgets for a month
//	@Description	Every budget of the signed-in user (at most one per currency), compared with that month's total spending in that currency.
//	@Description	Budgets repeat every month. status is ok, near (spent reached alert_percent of amount) or over (spent more than amount).
//	@Tags			budgets
//	@Produce		json
//	@Security		BearerAuth
//	@Param			month			query		string	true	"Month (YYYY-MM)"
//	@Param			Accept-Language	header		string	false	"Language of errors"	Enums(en, km, ms)
//	@Success		200				{object}	response.SuccessRes{data=[]dto.BudgetStatusRes}
//	@Failure		400				{object}	response.ErrorRes	"VALIDATION_FAILED (month missing or not YYYY-MM)"
//	@Failure		429				{object}	response.ErrorRes	"TOO_MANY_REQUESTS"
//	@Router			/v1/budgets [get]
func (h *BudgetHandler) List(c *gin.Context) {
	var req dto.BudgetListReq
	if err := c.ShouldBindQuery(&req); err != nil {
		response.JsonErrRes(c, err, http.StatusBadRequest)
		return
	}
	res := h.budgetSvc.List(c.Request.Context(), middleware.CurrentUserID(c), &req)
	if res.Error != nil {
		response.JsonErrRes(c, res.Error, res.StatusCode)
		return
	}
	response.JsonSuccessRes(c, res.Data, res.StatusCode)
}

// Create godoc
//
//	@Summary		Create a budget
//	@Description	A monthly limit on total spending in one currency. amount is a decimal string, as for expenses.
//	@Tags			budgets
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			body			body		dto.BudgetReq	true	"Budget"
//	@Param			Accept-Language	header		string			false	"Language of errors"	Enums(en, km, ms)
//	@Success		201				{object}	response.SuccessRes{data=dto.BudgetRes}
//	@Failure		400				{object}	response.ErrorRes	"VALIDATION_FAILED, INVALID_CURRENCY or INVALID_AMOUNT"
//	@Failure		409				{object}	response.ErrorRes	"BUDGET_EXISTS (this currency already has a budget)"
//	@Failure		413				{object}	response.ErrorRes	"REQUEST_TOO_LARGE"
//	@Failure		429				{object}	response.ErrorRes	"TOO_MANY_REQUESTS"
//	@Router			/v1/budgets [post]
func (h *BudgetHandler) Create(c *gin.Context) {
	var req dto.BudgetReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.JsonErrRes(c, err, http.StatusBadRequest)
		return
	}
	res := h.budgetSvc.Create(c.Request.Context(), middleware.CurrentUserID(c), &req)
	if res.Error != nil {
		response.JsonErrRes(c, res.Error, res.StatusCode)
		return
	}
	response.JsonSuccessRes(c, res.Data, res.StatusCode)
}

// Update godoc
//
//	@Summary		Update a budget
//	@Description	Replaces every field of one of the signed-in user's budgets.
//	@Tags			budgets
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id				path		int				true	"Budget ID"	minimum(1)
//	@Param			body			body		dto.BudgetReq	true	"Budget"
//	@Param			Accept-Language	header		string			false	"Language of errors"	Enums(en, km, ms)
//	@Success		200				{object}	response.SuccessRes{data=dto.BudgetRes}
//	@Failure		400				{object}	response.ErrorRes	"VALIDATION_FAILED, INVALID_ID, INVALID_CURRENCY or INVALID_AMOUNT"
//	@Failure		404				{object}	response.ErrorRes	"RECORD_NOT_FOUND"
//	@Failure		409				{object}	response.ErrorRes	"BUDGET_EXISTS (this currency already has a budget)"
//	@Failure		413				{object}	response.ErrorRes	"REQUEST_TOO_LARGE"
//	@Failure		429				{object}	response.ErrorRes	"TOO_MANY_REQUESTS"
//	@Router			/v1/budgets/{id} [put]
func (h *BudgetHandler) Update(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	if id <= 0 {
		response.JsonErrRes(c, i18n.InvalidID, http.StatusBadRequest)
		return
	}
	var req dto.BudgetReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.JsonErrRes(c, err, http.StatusBadRequest)
		return
	}
	res := h.budgetSvc.Update(c.Request.Context(), middleware.CurrentUserID(c), uint(id), &req)
	if res.Error != nil {
		response.JsonErrRes(c, res.Error, res.StatusCode)
		return
	}
	response.JsonSuccessRes(c, res.Data, res.StatusCode)
}

// Delete godoc
//
//	@Summary		Delete a budget
//	@Description	Soft-deletes one of the signed-in user's budgets.
//	@Tags			budgets
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id				path		int		true	"Budget ID"				minimum(1)
//	@Param			Accept-Language	header		string	false	"Language of errors"	Enums(en, km, ms)
//	@Success		200				{object}	response.SuccessRes
//	@Failure		400				{object}	response.ErrorRes	"INVALID_ID"
//	@Failure		404				{object}	response.ErrorRes	"RECORD_NOT_FOUND"
//	@Failure		429				{object}	response.ErrorRes	"TOO_MANY_REQUESTS"
//	@Router			/v1/budgets/{id} [delete]
func (h *BudgetHandler) Delete(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	if id <= 0 {
		response.JsonErrRes(c, i18n.InvalidID, http.StatusBadRequest)
		return
	}
	res := h.budgetSvc.Delete(c.Request.Context(), middleware.CurrentUserID(c), uint(id))
	if res.Error != nil {
		response.JsonErrRes(c, res.Error, res.StatusCode)
		return
	}
	response.JsonSuccessRes(c, res.Data, res.StatusCode)
}
