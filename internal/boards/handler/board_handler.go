package handler

import (
	"net/http"

	"github.com/Fazann/notedbook-backend/internal/boards/dto"
	"github.com/Fazann/notedbook-backend/internal/boards/service"
	"github.com/Fazann/notedbook-backend/pkg/response"
	"github.com/Fazann/notedbook-backend/shared/middleware"
	"github.com/gin-gonic/gin"
)

type BoardHandler struct {
	boardSvc service.BoardServiceImpl
}

func NewBoardHandler() *BoardHandler {
	return &BoardHandler{
		boardSvc: service.NewBoardSvc(),
	}
}

// List godoc
//
//	@Summary		List my boards
//	@Description	Returns the signed-in user's boards, oldest first, with the number of cards on each.
//	@Tags			boards
//	@Produce		json
//	@Security		BearerAuth
//	@Param			Accept-Language	header		string	false	"Language of errors"	Enums(en, km, ms)
//	@Success		200				{object}	response.SuccessRes{data=[]dto.BoardListRes}
//	@Failure		429				{object}	response.ErrorRes	"TOO_MANY_REQUESTS"
//	@Router			/v1/boards [get]
func (h *BoardHandler) List(c *gin.Context) {
	res := h.boardSvc.List(c.Request.Context(), middleware.CurrentUserID(c))
	if res.Error != nil {
		response.JsonErrRes(c, res.Error, res.StatusCode)
		return
	}

	response.JsonSuccessRes(c, res.Data, res.StatusCode)
}

// ListDueCards godoc
//
//	@Summary		List cards due soon
//	@Description	Cards from all of the signed-in user's boards due on or before `today` + `days`, overdue ones included.
//	@Description	Cards in the last column of a board ("Done") are left out. Sorted by due date.
//	@Tags			boards
//	@Produce		json
//	@Security		BearerAuth
//	@Param			days			query		int		false	"Days ahead (1-60, default 7)"	minimum(1)	maximum(60)
//	@Param			today			query		string	false	"Current date (YYYY-MM-DD); defaults to today in UTC"
//	@Param			Accept-Language	header		string	false	"Language of errors"	Enums(en, km, ms)
//	@Success		200				{object}	response.SuccessRes{data=[]dto.DueCardRes}
//	@Failure		400				{object}	response.ErrorRes	"VALIDATION_FAILED (days out of range, today not YYYY-MM-DD)"
//	@Failure		429				{object}	response.ErrorRes	"TOO_MANY_REQUESTS"
//	@Router			/v1/cards/due [get]
func (h *BoardHandler) ListDueCards(c *gin.Context) {
	var req dto.DueCardListReq
	if err := c.ShouldBindQuery(&req); err != nil {
		response.JsonErrRes(c, err, http.StatusBadRequest)
		return
	}

	res := h.boardSvc.ListDueCards(c.Request.Context(), middleware.CurrentUserID(c), &req)
	if res.Error != nil {
		response.JsonErrRes(c, res.Error, res.StatusCode)
		return
	}

	response.JsonSuccessRes(c, res.Data, res.StatusCode)
}
