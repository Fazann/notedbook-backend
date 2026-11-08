package service

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/Fazann/notedbook-backend/internal/boards/dto"
	"github.com/Fazann/notedbook-backend/internal/boards/repo"
	"github.com/Fazann/notedbook-backend/pkg/enums"
	"github.com/Fazann/notedbook-backend/pkg/i18n"
	"github.com/Fazann/notedbook-backend/pkg/response"
	model "github.com/Fazann/notedbook-backend/shared/models"
	"gorm.io/gorm"
)

// positionGap is the gap between columns / cards added at the end (the first one is at 1000).
const positionGap = 1000

func (s *boardSvc) Get(ctx context.Context, userID, id uint) response.SvcRes {
	board, err := s.boardRepo.FindBoardDetail(ctx, userID, id)
	if err != nil {
		return notFoundOr(err, "find board")
	}
	return response.SvcRespSuccess(toBoardRes(board), http.StatusOK)
}

// Create adds a board with the given columns, left to right at positions 1000, 2000…
func (s *boardSvc) Create(ctx context.Context, userID uint, req *dto.BoardReq) response.SvcRes {
	name, err := cleanName(req.Name, "name")
	if err != nil {
		return response.SvcRespErr(err, http.StatusBadRequest)
	}
	board := &model.Board{UserID: userID, Name: name}
	for i, columnName := range req.Columns {
		columnName, err := cleanName(columnName, fmt.Sprintf("columns[%d]", i))
		if err != nil {
			return response.SvcRespErr(err, http.StatusBadRequest)
		}
		board.Columns = append(board.Columns, model.BoardColumn{Name: columnName, Position: float64((i + 1) * positionGap)})
	}
	if err := s.boardRepo.CreateBoard(ctx, board); err != nil {
		return response.SvcRespErr(fmt.Errorf("create board: %w", err), http.StatusInternalServerError)
	}
	return response.SvcRespSuccess(toBoardRes(board), http.StatusCreated)
}

func (s *boardSvc) Update(ctx context.Context, userID, id uint, req *dto.BoardUpdateReq) response.SvcRes {
	if _, err := s.boardRepo.FindOwnBoard(ctx, userID, id); err != nil {
		return notFoundOr(err, "find board")
	}
	name, err := cleanName(req.Name, "name")
	if err != nil {
		return response.SvcRespErr(err, http.StatusBadRequest)
	}
	if err := s.boardRepo.UpdateBoardName(ctx, id, name); err != nil {
		return response.SvcRespErr(fmt.Errorf("rename board: %w", err), http.StatusInternalServerError)
	}
	return s.Get(ctx, userID, id)
}

func (s *boardSvc) Delete(ctx context.Context, userID, id uint) response.SvcRes {
	if _, err := s.boardRepo.FindOwnBoard(ctx, userID, id); err != nil {
		return notFoundOr(err, "find board")
	}
	if err := s.boardRepo.Transaction(ctx, func(r repo.BoardRepoImpl) error { return r.DeleteBoard(ctx, id) }); err != nil {
		return response.SvcRespErr(fmt.Errorf("delete board: %w", err), http.StatusInternalServerError)
	}
	return response.SvcRespSuccess(nil, http.StatusOK)
}

// CreateColumn adds a column at the right end of the board.
func (s *boardSvc) CreateColumn(ctx context.Context, userID, boardID uint, req *dto.ColumnReq) response.SvcRes {
	if _, err := s.boardRepo.FindOwnBoard(ctx, userID, boardID); err != nil {
		return notFoundOr(err, "find board")
	}
	name, err := cleanName(req.Name, "name")
	if err != nil {
		return response.SvcRespErr(err, http.StatusBadRequest)
	}
	column := &model.BoardColumn{BoardID: boardID, Name: name}
	err = s.boardRepo.Transaction(ctx, func(r repo.BoardRepoImpl) error {
		last, err := r.LastColumnPosition(ctx, boardID)
		if err != nil {
			return fmt.Errorf("find last column position: %w", err)
		}
		column.Position = last + positionGap
		return r.CreateColumn(ctx, column)
	})
	if err != nil {
		return response.SvcRespErr(fmt.Errorf("create column: %w", err), http.StatusInternalServerError)
	}
	return response.SvcRespSuccess(toColumnRes(column), http.StatusCreated)
}

func (s *boardSvc) UpdateColumn(ctx context.Context, userID, id uint, req *dto.ColumnReq) response.SvcRes {
	column, err := s.boardRepo.FindOwnColumn(ctx, userID, id)
	if err != nil {
		return notFoundOr(err, "find column")
	}
	name, err := cleanName(req.Name, "name")
	if err != nil {
		return response.SvcRespErr(err, http.StatusBadRequest)
	}
	if err := s.boardRepo.UpdateColumnName(ctx, id, name); err != nil {
		return response.SvcRespErr(fmt.Errorf("rename column: %w", err), http.StatusInternalServerError)
	}
	column.Name = name
	return response.SvcRespSuccess(toColumnRes(column), http.StatusOK)
}

func (s *boardSvc) DeleteColumn(ctx context.Context, userID, id uint) response.SvcRes {
	if _, err := s.boardRepo.FindOwnColumn(ctx, userID, id); err != nil {
		return notFoundOr(err, "find column")
	}
	if err := s.boardRepo.Transaction(ctx, func(r repo.BoardRepoImpl) error { return r.DeleteColumn(ctx, id) }); err != nil {
		return response.SvcRespErr(fmt.Errorf("delete column: %w", err), http.StatusInternalServerError)
	}
	return response.SvcRespSuccess(nil, http.StatusOK)
}

// CreateCard adds a card at the bottom of the column.
func (s *boardSvc) CreateCard(ctx context.Context, userID, columnID uint, req *dto.CardReq) response.SvcRes {
	if _, err := s.boardRepo.FindOwnColumn(ctx, userID, columnID); err != nil {
		return notFoundOr(err, "find column")
	}
	fields, err := cleanCard(req)
	if err != nil {
		return response.SvcRespErr(err, http.StatusBadRequest)
	}
	card := &model.BoardCard{
		ColumnID:    columnID,
		Title:       fields.title,
		Description: fields.description,
		DueDate:     fields.dueDate,
		Label:       fields.label,
	}
	err = s.boardRepo.Transaction(ctx, func(r repo.BoardRepoImpl) error {
		last, err := r.LastCardPosition(ctx, columnID)
		if err != nil {
			return fmt.Errorf("find last card position: %w", err)
		}
		card.Position = last + positionGap
		return r.CreateCard(ctx, card)
	})
	if err != nil {
		return response.SvcRespErr(fmt.Errorf("create card: %w", err), http.StatusInternalServerError)
	}
	return response.SvcRespSuccess(toCardRes(card), http.StatusCreated)
}

// UpdateCard replaces the card's title, description, due date and label.
func (s *boardSvc) UpdateCard(ctx context.Context, userID, id uint, req *dto.CardReq) response.SvcRes {
	if _, _, err := s.boardRepo.FindOwnCard(ctx, userID, id); err != nil {
		return notFoundOr(err, "find card")
	}
	fields, err := cleanCard(req)
	if err != nil {
		return response.SvcRespErr(err, http.StatusBadRequest)
	}
	updates := map[string]any{
		"title":       fields.title,
		"description": fields.description,
		"due_date":    fields.dueDate,
		"label":       fields.label,
	}
	return s.updateCard(ctx, id, updates)
}

// MoveCard puts the card at a new position, in the same or another column of its board.
func (s *boardSvc) MoveCard(ctx context.Context, userID, id uint, req *dto.CardMoveReq) response.SvcRes {
	_, boardID, err := s.boardRepo.FindOwnCard(ctx, userID, id)
	if err != nil {
		return notFoundOr(err, "find card")
	}
	if req.Position == nil || math.IsNaN(*req.Position) || math.IsInf(*req.Position, 0) {
		return response.SvcRespErr(i18n.Err(i18n.FieldInvalid, i18n.Params{"field": "position"}), http.StatusBadRequest)
	}
	column, err := s.boardRepo.FindOwnColumn(ctx, userID, req.ColumnID)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return response.SvcRespErr(fmt.Errorf("find column: %w", err), http.StatusInternalServerError)
	}
	if column == nil || column.BoardID != boardID {
		return response.SvcRespErr(i18n.Err(i18n.FieldInvalid, i18n.Params{"field": "column_id"}), http.StatusBadRequest)
	}
	return s.updateCard(ctx, id, map[string]any{"column_id": column.ID, "position": *req.Position})
}

func (s *boardSvc) DeleteCard(ctx context.Context, userID, id uint) response.SvcRes {
	if _, _, err := s.boardRepo.FindOwnCard(ctx, userID, id); err != nil {
		return notFoundOr(err, "find card")
	}
	if err := s.boardRepo.DeleteCard(ctx, id); err != nil {
		return response.SvcRespErr(fmt.Errorf("delete card: %w", err), http.StatusInternalServerError)
	}
	return response.SvcRespSuccess(nil, http.StatusOK)
}

func (s *boardSvc) updateCard(ctx context.Context, id uint, updates map[string]any) response.SvcRes {
	if err := s.boardRepo.UpdateCard(ctx, id, updates); err != nil {
		return response.SvcRespErr(fmt.Errorf("update card: %w", err), http.StatusInternalServerError)
	}
	card, err := s.boardRepo.FindCard(ctx, id)
	if err != nil {
		return response.SvcRespErr(fmt.Errorf("reload card: %w", err), http.StatusInternalServerError)
	}
	return response.SvcRespSuccess(toCardRes(card), http.StatusOK)
}

// notFoundOr turns a missing (or another user's) record into 404 RECORD_NOT_FOUND, anything else into a 500.
func notFoundOr(err error, action string) response.SvcRes {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return response.SvcRespErr(i18n.RecordNotFound, http.StatusNotFound)
	}
	return response.SvcRespErr(fmt.Errorf("%s: %w", action, err), http.StatusInternalServerError)
}

// cleanName trims a board / column name; binding already checked the length, but not blank-only names.
func cleanName(name, field string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", i18n.Err(i18n.FieldRequired, i18n.Params{"field": field})
	}
	return name, nil
}

type cardFields struct {
	title       string
	description string
	dueDate     *time.Time
	label       *enums.BoardLabelEnum
}

func cleanCard(req *dto.CardReq) (*cardFields, error) {
	title, err := cleanName(req.Title, "title")
	if err != nil {
		return nil, err
	}
	fields := &cardFields{title: title, description: strings.TrimSpace(req.Description)}
	if req.DueDate != nil && *req.DueDate != "" {
		date, err := time.Parse(dateLayout, *req.DueDate)
		if err != nil {
			return nil, i18n.Err(i18n.FieldInvalid, i18n.Params{"field": "due_date"})
		}
		fields.dueDate = &date
	}
	if req.Label != nil && *req.Label != "" {
		label := enums.BoardLabelEnum(*req.Label)
		valid := false
		for _, v := range enums.BoardLabelValues {
			valid = valid || v == *req.Label
		}
		if !valid {
			return nil, i18n.Err(i18n.FieldInvalid, i18n.Params{"field": "label"})
		}
		fields.label = &label
	}
	return fields, nil
}

func toBoardRes(board *model.Board) *dto.BoardRes {
	res := &dto.BoardRes{ID: board.ID, Name: board.Name, Columns: make([]*dto.ColumnRes, len(board.Columns))}
	for i := range board.Columns {
		res.Columns[i] = toColumnRes(&board.Columns[i])
	}
	return res
}

func toColumnRes(column *model.BoardColumn) *dto.ColumnRes {
	res := &dto.ColumnRes{
		ID:       column.ID,
		BoardID:  column.BoardID,
		Name:     column.Name,
		Position: column.Position,
		Cards:    make([]*dto.CardRes, len(column.Cards)),
	}
	for i := range column.Cards {
		res.Cards[i] = toCardRes(&column.Cards[i])
	}
	return res
}

func toCardRes(card *model.BoardCard) *dto.CardRes {
	var due *string
	if card.DueDate != nil {
		formatted := card.DueDate.Format(dateLayout)
		due = &formatted
	}
	return &dto.CardRes{
		ID:          card.ID,
		ColumnID:    card.ColumnID,
		Title:       card.Title,
		Description: card.Description,
		DueDate:     due,
		Label:       card.Label,
		Position:    card.Position,
	}
}
