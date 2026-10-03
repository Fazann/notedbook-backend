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

type CategoryHandler struct {
	categorySvc service.CategoryServiceImpl
}

func NewCategoryHandler() *CategoryHandler {
	return &CategoryHandler{
		categorySvc: service.NewCategorySvc(),
	}
}

// List godoc
//
//	@Summary		List my expense categories
//	@Description	One page of the default (admin) categories plus the signed-in user's own. `q` matches the name in every language
//	@Description	(base name and translations). display_name follows the request language.
//	@Tags			expense-categories
//	@Produce		json
//	@Security		BearerAuth
//	@Param			page			query		int		false	"Page (1-based)"				minimum(1)	default(1)
//	@Param			limit			query		int		false	"Rows per page"					minimum(1)	maximum(1000)	default(10)
//	@Param			q				query		string	false	"Search in any language"		maxlength(100)
//	@Param			sort			query		string	false	"Sort (\"-\" = descending)"		Enums(name, -name, created_at, -created_at, expense_count, -expense_count)	default(name)
//	@Param			Accept-Language	header		string	false	"Language of names and errors"	Enums(en, km, ms)
//	@Success		200				{object}	response.Paginate{data=[]dto.CategoryRes}
//	@Router			/v1/categories [get]
func (h *CategoryHandler) List(c *gin.Context) {
	var req dto.CategoryListReq
	if err := c.ShouldBindQuery(&req); err != nil {
		response.JsonErrRes(c, err, http.StatusBadRequest)
		return
	}

	res := h.categorySvc.List(c.Request.Context(), middleware.CurrentUserID(c), i18n.FromRequest(c.Request), &req)
	if res.Error != nil {
		response.JsonErrRes(c, res.Error, res.StatusCode)
		return
	}

	response.JsonSuccessResWithPage(c, res)
}

// ListAll godoc
//
//	@Summary		List all my expense categories
//	@Description	The default (admin) categories, then the signed-in user's own, without paging (for pickers), oldest first.
//	@Tags			expense-categories
//	@Produce		json
//	@Security		BearerAuth
//	@Param			Accept-Language	header		string	false	"Language of names and errors"	Enums(en, km, ms)
//	@Success		200				{object}	response.SuccessRes{data=[]dto.CategoryRes}
//	@Router			/v1/categories/all [get]
func (h *CategoryHandler) ListAll(c *gin.Context) {
	res := h.categorySvc.ListAll(c.Request.Context(), middleware.CurrentUserID(c), i18n.FromRequest(c.Request))
	if res.Error != nil {
		response.JsonErrRes(c, res.Error, res.StatusCode)
		return
	}

	response.JsonSuccessRes(c, res.Data, res.StatusCode)
}

// GetDetail godoc
//
//	@Summary		Get an expense category
//	@Description	Returns a default category or one of the signed-in user's, with all its translations.
//	@Tags			expense-categories
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id				path		int		true	"Category ID"					minimum(1)
//	@Param			Accept-Language	header		string	false	"Language of names and errors"	Enums(en, km, ms)
//	@Success		200				{object}	response.SuccessRes{data=dto.CategoryRes}
//	@Router			/v1/categories/{id} [get]
func (h *CategoryHandler) GetDetail(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	if id <= 0 {
		response.JsonErrRes(c, i18n.InvalidID, http.StatusBadRequest)
		return
	}

	res := h.categorySvc.GetDetail(c.Request.Context(), middleware.CurrentUserID(c), uint(id), i18n.FromRequest(c.Request))
	if res.Error != nil {
		response.JsonErrRes(c, res.Error, res.StatusCode)
		return
	}

	response.JsonSuccessRes(c, res.Data, res.StatusCode)
}

// Create godoc
//
//	@Summary		Create an expense category
//	@Description	The base name must differ from the user's and the default categories (case-insensitive). translations holds an optional
//	@Description	name per language (en, km, ms). icon and color must be one of the values the app supports.
//	@Tags			expense-categories
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			body			body		dto.CategoryReq	true	"Category"
//	@Param			Accept-Language	header		string			false	"Language of names and errors"	Enums(en, km, ms)
//	@Success		201				{object}	response.SuccessRes{data=dto.CategoryRes}
//	@Router			/v1/categories [post]
func (h *CategoryHandler) Create(c *gin.Context) {
	var req dto.CategoryReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.JsonErrRes(c, err, http.StatusBadRequest)
		return
	}

	res := h.categorySvc.Create(c.Request.Context(), middleware.CurrentUserID(c), i18n.FromRequest(c.Request), &req)
	if res.Error != nil {
		response.JsonErrRes(c, res.Error, res.StatusCode)
		return
	}

	response.JsonSuccessRes(c, res.Data, res.StatusCode)
}

// Update godoc
//
//	@Summary		Update an expense category
//	@Description	Partial update: only the fields sent are changed. translations, when sent, replaces all translations (languages
//	@Description	left out are removed). Default categories are read-only.
//	@Tags			expense-categories
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id				path		int						true	"Category ID"	minimum(1)
//	@Param			body			body		dto.CategoryUpdateReq	true	"Fields to change"
//	@Param			Accept-Language	header		string					false	"Language of names and errors"	Enums(en, km, ms)
//	@Success		200				{object}	response.SuccessRes{data=dto.CategoryRes}
//	@Router			/v1/categories/{id} [put]
func (h *CategoryHandler) Update(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	if id <= 0 {
		response.JsonErrRes(c, i18n.InvalidID, http.StatusBadRequest)
		return
	}

	var req dto.CategoryUpdateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.JsonErrRes(c, err, http.StatusBadRequest)
		return
	}

	res := h.categorySvc.Update(c.Request.Context(), middleware.CurrentUserID(c), uint(id), i18n.FromRequest(c.Request), &req)
	if res.Error != nil {
		response.JsonErrRes(c, res.Error, res.StatusCode)
		return
	}

	response.JsonSuccessRes(c, res.Data, res.StatusCode)
}

// Delete godoc
//
//	@Summary		Delete an expense category
//	@Description	Soft-deletes one of the signed-in user's categories. Default categories are read-only.
//	@Description	A category that still has expenses needs `reassign_to`: its expenses are moved to that category first.
//	@Tags			expense-categories
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id				path		int		true	"Category ID"						minimum(1)
//	@Param			reassign_to		query		int		false	"Category to move the expenses to"	minimum(1)
//	@Param			Accept-Language	header		string	false	"Language of errors"				Enums(en, km, ms)
//	@Success		200				{object}	response.SuccessRes
//	@Failure		400				{object}	response.ErrorRes	"INVALID_ID or INVALID_REASSIGN_CATEGORY"
//	@Failure		403				{object}	response.ErrorRes	"CATEGORY_IS_DEFAULT"
//	@Failure		404				{object}	response.ErrorRes	"RECORD_NOT_FOUND"
//	@Failure		409				{object}	response.ErrorRes	"CATEGORY_IN_USE (has expenses, no reassign_to)"
//	@Router			/v1/categories/{id} [delete]
func (h *CategoryHandler) Delete(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	if id <= 0 {
		response.JsonErrRes(c, i18n.InvalidID, http.StatusBadRequest)
		return
	}
	var req dto.CategoryDeleteReq
	if err := c.ShouldBindQuery(&req); err != nil {
		response.JsonErrRes(c, err, http.StatusBadRequest)
		return
	}

	res := h.categorySvc.Delete(c.Request.Context(), middleware.CurrentUserID(c), uint(id), req.ReassignTo)
	if res.Error != nil {
		response.JsonErrRes(c, res.Error, res.StatusCode)
		return
	}

	response.JsonSuccessRes(c, res.Data, res.StatusCode)
}
