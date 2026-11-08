package handler

import (
	"net/http"
	"strconv"

	"github.com/Fazann/notedbook-backend/internal/boards/dto"
	"github.com/Fazann/notedbook-backend/pkg/i18n"
	"github.com/Fazann/notedbook-backend/pkg/response"
	"github.com/Fazann/notedbook-backend/shared/middleware"
	"github.com/gin-gonic/gin"
)

// pathID reads the :id path parameter; on a bad value it writes 400 INVALID_ID and returns ok = false.
func pathID(c *gin.Context) (uint, bool) {
	id, _ := strconv.Atoi(c.Param("id"))
	if id <= 0 {
		response.JsonErrRes(c, i18n.InvalidID, http.StatusBadRequest)
		return 0, false
	}
	return uint(id), true
}

// bindJSON binds the body; on a bad body it writes 400 and returns false.
func bindJSON(c *gin.Context, req any) bool {
	if err := c.ShouldBindJSON(req); err != nil {
		response.JsonErrRes(c, err, http.StatusBadRequest)
		return false
	}
	return true
}

func write(c *gin.Context, res response.SvcRes) {
	if res.Error != nil {
		response.JsonErrRes(c, res.Error, res.StatusCode)
		return
	}
	response.JsonSuccessRes(c, res.Data, res.StatusCode)
}

// Get godoc
//
//	@Summary		Get a board
//	@Description	The board with its columns and cards, each sorted by position. The last column counts as "Done".
//	@Tags			boards
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id				path		int		true	"Board ID"				minimum(1)
//	@Param			Accept-Language	header		string	false	"Language of errors"	Enums(en, km, ms)
//	@Success		200				{object}	response.SuccessRes{data=dto.BoardRes}
//	@Failure		400				{object}	response.ErrorRes	"INVALID_ID"
//	@Failure		404				{object}	response.ErrorRes	"RECORD_NOT_FOUND"
//	@Failure		429				{object}	response.ErrorRes	"TOO_MANY_REQUESTS"
//	@Router			/v1/boards/{id} [get]
func (h *BoardHandler) Get(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	write(c, h.boardSvc.Get(c.Request.Context(), middleware.CurrentUserID(c), id))
}

// Create godoc
//
//	@Summary		Create a board
//	@Description	Creates a board with the given columns (left to right). Send the column names in the user's language.
//	@Tags			boards
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			body			body		dto.BoardReq	true	"Board"
//	@Param			Accept-Language	header		string			false	"Language of errors"	Enums(en, km, ms)
//	@Success		201				{object}	response.SuccessRes{data=dto.BoardRes}
//	@Failure		400				{object}	response.ErrorRes	"VALIDATION_FAILED, FIELD_REQUIRED"
//	@Failure		413				{object}	response.ErrorRes	"REQUEST_TOO_LARGE"
//	@Failure		429				{object}	response.ErrorRes	"TOO_MANY_REQUESTS"
//	@Router			/v1/boards [post]
func (h *BoardHandler) Create(c *gin.Context) {
	var req dto.BoardReq
	if !bindJSON(c, &req) {
		return
	}
	write(c, h.boardSvc.Create(c.Request.Context(), middleware.CurrentUserID(c), &req))
}

// Update godoc
//
//	@Summary	Rename a board
//	@Tags		boards
//	@Accept		json
//	@Produce	json
//	@Security	BearerAuth
//	@Param		id				path		int					true	"Board ID"	minimum(1)
//	@Param		body			body		dto.BoardUpdateReq	true	"New name"
//	@Param		Accept-Language	header		string				false	"Language of errors"	Enums(en, km, ms)
//	@Success	200				{object}	response.SuccessRes{data=dto.BoardRes}
//	@Failure	400				{object}	response.ErrorRes	"INVALID_ID, VALIDATION_FAILED, FIELD_REQUIRED"
//	@Failure	404				{object}	response.ErrorRes	"RECORD_NOT_FOUND"
//	@Failure	413				{object}	response.ErrorRes	"REQUEST_TOO_LARGE"
//	@Failure	429				{object}	response.ErrorRes	"TOO_MANY_REQUESTS"
//	@Router		/v1/boards/{id} [put]
func (h *BoardHandler) Update(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	var req dto.BoardUpdateReq
	if !bindJSON(c, &req) {
		return
	}
	write(c, h.boardSvc.Update(c.Request.Context(), middleware.CurrentUserID(c), id, &req))
}

// Delete godoc
//
//	@Summary		Delete a board
//	@Description	Deletes the board with all of its columns and cards.
//	@Tags			boards
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id				path		int		true	"Board ID"				minimum(1)
//	@Param			Accept-Language	header		string	false	"Language of errors"	Enums(en, km, ms)
//	@Success		200				{object}	response.SuccessRes
//	@Failure		400				{object}	response.ErrorRes	"INVALID_ID"
//	@Failure		404				{object}	response.ErrorRes	"RECORD_NOT_FOUND"
//	@Failure		429				{object}	response.ErrorRes	"TOO_MANY_REQUESTS"
//	@Router			/v1/boards/{id} [delete]
func (h *BoardHandler) Delete(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	write(c, h.boardSvc.Delete(c.Request.Context(), middleware.CurrentUserID(c), id))
}

// CreateColumn godoc
//
//	@Summary		Add a column
//	@Description	Adds a column at the right end of the board (it becomes the new "Done" column).
//	@Tags			boards
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id				path		int				true	"Board ID"	minimum(1)
//	@Param			body			body		dto.ColumnReq	true	"Column"
//	@Param			Accept-Language	header		string			false	"Language of errors"	Enums(en, km, ms)
//	@Success		201				{object}	response.SuccessRes{data=dto.ColumnRes}
//	@Failure		400				{object}	response.ErrorRes	"INVALID_ID, VALIDATION_FAILED, FIELD_REQUIRED"
//	@Failure		404				{object}	response.ErrorRes	"RECORD_NOT_FOUND"
//	@Failure		413				{object}	response.ErrorRes	"REQUEST_TOO_LARGE"
//	@Failure		429				{object}	response.ErrorRes	"TOO_MANY_REQUESTS"
//	@Router			/v1/boards/{id}/columns [post]
func (h *BoardHandler) CreateColumn(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	var req dto.ColumnReq
	if !bindJSON(c, &req) {
		return
	}
	write(c, h.boardSvc.CreateColumn(c.Request.Context(), middleware.CurrentUserID(c), id, &req))
}

// UpdateColumn godoc
//
//	@Summary	Rename a column
//	@Tags		boards
//	@Accept		json
//	@Produce	json
//	@Security	BearerAuth
//	@Param		id				path		int				true	"Column ID"	minimum(1)
//	@Param		body			body		dto.ColumnReq	true	"New name"
//	@Param		Accept-Language	header		string			false	"Language of errors"	Enums(en, km, ms)
//	@Success	200				{object}	response.SuccessRes{data=dto.ColumnRes}
//	@Failure	400				{object}	response.ErrorRes	"INVALID_ID, VALIDATION_FAILED, FIELD_REQUIRED"
//	@Failure	404				{object}	response.ErrorRes	"RECORD_NOT_FOUND"
//	@Failure	413				{object}	response.ErrorRes	"REQUEST_TOO_LARGE"
//	@Failure	429				{object}	response.ErrorRes	"TOO_MANY_REQUESTS"
//	@Router		/v1/columns/{id} [put]
func (h *BoardHandler) UpdateColumn(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	var req dto.ColumnReq
	if !bindJSON(c, &req) {
		return
	}
	write(c, h.boardSvc.UpdateColumn(c.Request.Context(), middleware.CurrentUserID(c), id, &req))
}

// DeleteColumn godoc
//
//	@Summary		Delete a column
//	@Description	Deletes the column with all of its cards.
//	@Tags			boards
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id				path		int		true	"Column ID"				minimum(1)
//	@Param			Accept-Language	header		string	false	"Language of errors"	Enums(en, km, ms)
//	@Success		200				{object}	response.SuccessRes
//	@Failure		400				{object}	response.ErrorRes	"INVALID_ID"
//	@Failure		404				{object}	response.ErrorRes	"RECORD_NOT_FOUND"
//	@Failure		429				{object}	response.ErrorRes	"TOO_MANY_REQUESTS"
//	@Router			/v1/columns/{id} [delete]
func (h *BoardHandler) DeleteColumn(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	write(c, h.boardSvc.DeleteColumn(c.Request.Context(), middleware.CurrentUserID(c), id))
}

// CreateCard godoc
//
//	@Summary		Add a card
//	@Description	Adds a card at the bottom of the column.
//	@Tags			boards
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id				path		int			true	"Column ID"	minimum(1)
//	@Param			body			body		dto.CardReq	true	"Card"
//	@Param			Accept-Language	header		string		false	"Language of errors"	Enums(en, km, ms)
//	@Success		201				{object}	response.SuccessRes{data=dto.CardRes}
//	@Failure		400				{object}	response.ErrorRes	"INVALID_ID, VALIDATION_FAILED, FIELD_REQUIRED, FIELD_INVALID"
//	@Failure		404				{object}	response.ErrorRes	"RECORD_NOT_FOUND"
//	@Failure		413				{object}	response.ErrorRes	"REQUEST_TOO_LARGE"
//	@Failure		429				{object}	response.ErrorRes	"TOO_MANY_REQUESTS"
//	@Router			/v1/columns/{id}/cards [post]
func (h *BoardHandler) CreateCard(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	var req dto.CardReq
	if !bindJSON(c, &req) {
		return
	}
	write(c, h.boardSvc.CreateCard(c.Request.Context(), middleware.CurrentUserID(c), id, &req))
}

// UpdateCard godoc
//
//	@Summary		Update a card
//	@Description	Replaces the title, description, due date and label (send null to clear due_date or label).
//	@Tags			boards
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id				path		int			true	"Card ID"	minimum(1)
//	@Param			body			body		dto.CardReq	true	"Card"
//	@Param			Accept-Language	header		string		false	"Language of errors"	Enums(en, km, ms)
//	@Success		200				{object}	response.SuccessRes{data=dto.CardRes}
//	@Failure		400				{object}	response.ErrorRes	"INVALID_ID, VALIDATION_FAILED, FIELD_REQUIRED, FIELD_INVALID"
//	@Failure		404				{object}	response.ErrorRes	"RECORD_NOT_FOUND"
//	@Failure		413				{object}	response.ErrorRes	"REQUEST_TOO_LARGE"
//	@Failure		429				{object}	response.ErrorRes	"TOO_MANY_REQUESTS"
//	@Router			/v1/cards/{id} [put]
func (h *BoardHandler) UpdateCard(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	var req dto.CardReq
	if !bindJSON(c, &req) {
		return
	}
	write(c, h.boardSvc.UpdateCard(c.Request.Context(), middleware.CurrentUserID(c), id, &req))
}

// MoveCard godoc
//
//	@Summary		Move a card
//	@Description	Moves the card to a column of the same board at the given position (between its new neighbours).
//	@Tags			boards
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id				path		int				true	"Card ID"	minimum(1)
//	@Param			body			body		dto.CardMoveReq	true	"Target column and position"
//	@Param			Accept-Language	header		string			false	"Language of errors"	Enums(en, km, ms)
//	@Success		200				{object}	response.SuccessRes{data=dto.CardRes}
//	@Failure		400				{object}	response.ErrorRes	"INVALID_ID, VALIDATION_FAILED, FIELD_INVALID (position, column_id)"
//	@Failure		404				{object}	response.ErrorRes	"RECORD_NOT_FOUND"
//	@Failure		413				{object}	response.ErrorRes	"REQUEST_TOO_LARGE"
//	@Failure		429				{object}	response.ErrorRes	"TOO_MANY_REQUESTS"
//	@Router			/v1/cards/{id}/move [patch]
func (h *BoardHandler) MoveCard(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	var req dto.CardMoveReq
	if !bindJSON(c, &req) {
		return
	}
	write(c, h.boardSvc.MoveCard(c.Request.Context(), middleware.CurrentUserID(c), id, &req))
}

// DeleteCard godoc
//
//	@Summary	Delete a card
//	@Tags		boards
//	@Produce	json
//	@Security	BearerAuth
//	@Param		id				path		int		true	"Card ID"				minimum(1)
//	@Param		Accept-Language	header		string	false	"Language of errors"	Enums(en, km, ms)
//	@Success	200				{object}	response.SuccessRes
//	@Failure	400				{object}	response.ErrorRes	"INVALID_ID"
//	@Failure	404				{object}	response.ErrorRes	"RECORD_NOT_FOUND"
//	@Failure	429				{object}	response.ErrorRes	"TOO_MANY_REQUESTS"
//	@Router		/v1/cards/{id} [delete]
func (h *BoardHandler) DeleteCard(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	write(c, h.boardSvc.DeleteCard(c.Request.Context(), middleware.CurrentUserID(c), id))
}
