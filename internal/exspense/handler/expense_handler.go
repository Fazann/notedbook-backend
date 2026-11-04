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

type ExpenseHandler struct {
	expenseSvc service.ExpenseServiceImpl
}

func NewExpenseHandler() *ExpenseHandler {
	return &ExpenseHandler{
		expenseSvc: service.NewExpenseSvc(),
	}
}

// List godoc
//
//	@Summary		List my expenses
//	@Description	One page of the signed-in user's expenses, optionally for one month, category and currency. `q` matches the note.
//	@Description	Sorting by amount groups by currency first. amount is a decimal string in the currency's major unit.
//	@Tags			expenses
//	@Produce		json
//	@Security		BearerAuth
//	@Param			page			query		int		false	"Page (1-based)"	minimum(1)	default(1)
//	@Param			limit			query		int		false	"Rows per page"		minimum(1)	maximum(1000)	default(10)
//	@Param			month			query		string	false	"Month (YYYY-MM)"
//	@Param			category_id		query		int		false	"Category ID"				minimum(1)
//	@Param			currency		query		string	false	"Currency"					Enums(USD, KHR, MYR)
//	@Param			q				query		string	false	"Search in the note"		maxlength(100)
//	@Param			sort			query		string	false	"Sort (\"-\" = descending)"	Enums(spent_at, -spent_at, amount, -amount)	default(-spent_at)
//	@Param			Accept-Language	header		string	false	"Language of errors"		Enums(en, km, ms)
//	@Success		200				{object}	response.Paginate{data=[]dto.ExpenseRes}
//	@Failure		429				{object}	response.ErrorRes	"TOO_MANY_REQUESTS"
//	@Router			/v1/expenses [get]
func (h *ExpenseHandler) List(c *gin.Context) {
	var req dto.ExpenseListReq
	if err := c.ShouldBindQuery(&req); err != nil {
		response.JsonErrRes(c, err, http.StatusBadRequest)
		return
	}

	res := h.expenseSvc.List(c.Request.Context(), middleware.CurrentUserID(c), &req)
	if res.Error != nil {
		response.JsonErrRes(c, res.Error, res.StatusCode)
		return
	}

	response.JsonSuccessResWithPage(c, res)
}

// Summary godoc
//
//	@Summary		Summarize my expenses of a month
//	@Description	Totals of the signed-in user's expenses in one month, per currency and per category and currency.
//	@Description	Amounts in different currencies are never added together. Amounts are decimal strings, as in the list.
//	@Tags			expenses
//	@Produce		json
//	@Security		BearerAuth
//	@Param			month			query		string	true	"Month (YYYY-MM)"
//	@Param			Accept-Language	header		string	false	"Language of errors"	Enums(en, km, ms)
//	@Success		200				{object}	response.SuccessRes{data=dto.ExpenseSummaryRes}
//	@Failure		400				{object}	response.ErrorRes	"VALIDATION_FAILED (month missing or not YYYY-MM)"
//	@Failure		429				{object}	response.ErrorRes	"TOO_MANY_REQUESTS"
//	@Router			/v1/expenses/summary [get]
func (h *ExpenseHandler) Summary(c *gin.Context) {
	var req dto.ExpenseSummaryReq
	if err := c.ShouldBindQuery(&req); err != nil {
		response.JsonErrRes(c, err, http.StatusBadRequest)
		return
	}

	res := h.expenseSvc.Summary(c.Request.Context(), middleware.CurrentUserID(c), &req)
	if res.Error != nil {
		response.JsonErrRes(c, res.Error, res.StatusCode)
		return
	}

	response.JsonSuccessRes(c, res.Data, res.StatusCode)
}

// GetDetail godoc
//
//	@Summary		Get an expense
//	@Description	Returns one of the signed-in user's expenses.
//	@Tags			expenses
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id				path		int		true	"Expense ID"			minimum(1)
//	@Param			Accept-Language	header		string	false	"Language of errors"	Enums(en, km, ms)
//	@Success		200				{object}	response.SuccessRes{data=dto.ExpenseRes}
//	@Failure		429				{object}	response.ErrorRes	"TOO_MANY_REQUESTS"
//	@Router			/v1/expenses/{id} [get]
func (h *ExpenseHandler) GetDetail(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	if id <= 0 {
		response.JsonErrRes(c, i18n.InvalidID, http.StatusBadRequest)
		return
	}

	res := h.expenseSvc.GetDetail(c.Request.Context(), middleware.CurrentUserID(c), uint(id))
	if res.Error != nil {
		response.JsonErrRes(c, res.Error, res.StatusCode)
		return
	}

	response.JsonSuccessRes(c, res.Data, res.StatusCode)
}

// Create godoc
//
//	@Summary		Create an expense
//	@Description	amount is a decimal string with at most the currency's decimal places (2 for USD / MYR, 0 for KHR), e.g. "12.50".
//	@Description	category_id must be a default category or one of the user's own.
//	@Tags			expenses
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			body			body		dto.ExpenseReq	true	"Expense"
//	@Param			Accept-Language	header		string			false	"Language of errors"	Enums(en, km, ms)
//	@Success		201				{object}	response.SuccessRes{data=dto.ExpenseRes}
//	@Failure		429				{object}	response.ErrorRes	"TOO_MANY_REQUESTS"
//	@Router			/v1/expenses [post]
func (h *ExpenseHandler) Create(c *gin.Context) {
	var req dto.ExpenseReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.JsonErrRes(c, err, http.StatusBadRequest)
		return
	}

	res := h.expenseSvc.Create(c.Request.Context(), middleware.CurrentUserID(c), &req)
	if res.Error != nil {
		response.JsonErrRes(c, res.Error, res.StatusCode)
		return
	}

	response.JsonSuccessRes(c, res.Data, res.StatusCode)
}

// Update godoc
//
//	@Summary		Update an expense
//	@Description	Partial update: only the fields sent are changed. amount is a decimal string; changing only currency keeps the decimal value.
//	@Tags			expenses
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id				path		int						true	"Expense ID"	minimum(1)
//	@Param			body			body		dto.ExpenseUpdateReq	true	"Fields to change"
//	@Param			Accept-Language	header		string					false	"Language of errors"	Enums(en, km, ms)
//	@Success		200				{object}	response.SuccessRes{data=dto.ExpenseRes}
//	@Failure		429				{object}	response.ErrorRes	"TOO_MANY_REQUESTS"
//	@Router			/v1/expenses/{id} [put]
func (h *ExpenseHandler) Update(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	if id <= 0 {
		response.JsonErrRes(c, i18n.InvalidID, http.StatusBadRequest)
		return
	}

	var req dto.ExpenseUpdateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.JsonErrRes(c, err, http.StatusBadRequest)
		return
	}

	res := h.expenseSvc.Update(c.Request.Context(), middleware.CurrentUserID(c), uint(id), &req)
	if res.Error != nil {
		response.JsonErrRes(c, res.Error, res.StatusCode)
		return
	}

	response.JsonSuccessRes(c, res.Data, res.StatusCode)
}

// Delete godoc
//
//	@Summary		Delete an expense
//	@Description	Soft-deletes one of the signed-in user's expenses.
//	@Tags			expenses
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id				path		int		true	"Expense ID"			minimum(1)
//	@Param			Accept-Language	header		string	false	"Language of errors"	Enums(en, km, ms)
//	@Success		200				{object}	response.SuccessRes
//	@Failure		429				{object}	response.ErrorRes	"TOO_MANY_REQUESTS"
//	@Router			/v1/expenses/{id} [delete]
func (h *ExpenseHandler) Delete(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	if id <= 0 {
		response.JsonErrRes(c, i18n.InvalidID, http.StatusBadRequest)
		return
	}

	res := h.expenseSvc.Delete(c.Request.Context(), middleware.CurrentUserID(c), uint(id))
	if res.Error != nil {
		response.JsonErrRes(c, res.Error, res.StatusCode)
		return
	}

	response.JsonSuccessRes(c, res.Data, res.StatusCode)
}
