package service

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/Fazann/notedbook-backend/internal/boards/dto"
	"github.com/Fazann/notedbook-backend/internal/boards/repo"
	"github.com/Fazann/notedbook-backend/pkg/i18n"
	"github.com/Fazann/notedbook-backend/pkg/response"
)

const (
	dateLayout     = "2006-01-02"
	defaultDueDays = 7
)

type BoardServiceImpl interface {
	List(ctx context.Context, userID uint) response.SvcRes
	ListDueCards(ctx context.Context, userID uint, req *dto.DueCardListReq) response.SvcRes

	Get(ctx context.Context, userID, id uint) response.SvcRes
	Create(ctx context.Context, userID uint, req *dto.BoardReq) response.SvcRes
	Update(ctx context.Context, userID, id uint, req *dto.BoardUpdateReq) response.SvcRes
	Delete(ctx context.Context, userID, id uint) response.SvcRes

	CreateColumn(ctx context.Context, userID, boardID uint, req *dto.ColumnReq) response.SvcRes
	UpdateColumn(ctx context.Context, userID, id uint, req *dto.ColumnReq) response.SvcRes
	DeleteColumn(ctx context.Context, userID, id uint) response.SvcRes

	CreateCard(ctx context.Context, userID, columnID uint, req *dto.CardReq) response.SvcRes
	UpdateCard(ctx context.Context, userID, id uint, req *dto.CardReq) response.SvcRes
	MoveCard(ctx context.Context, userID, id uint, req *dto.CardMoveReq) response.SvcRes
	DeleteCard(ctx context.Context, userID, id uint) response.SvcRes
}

type boardSvc struct {
	boardRepo repo.BoardRepoImpl
	// Returns the current time; replaced in tests.
	now func() time.Time
}

func NewBoardSvc() *boardSvc {
	return &boardSvc{
		boardRepo: repo.NewBoardRepo(),
		now:       time.Now,
	}
}

func (s *boardSvc) List(ctx context.Context, userID uint) response.SvcRes {
	boards, err := s.boardRepo.ListWithCardCount(ctx, userID)
	if err != nil {
		return response.SvcRespErr(fmt.Errorf("list boards: %w", err), http.StatusInternalServerError)
	}

	res := make([]*dto.BoardListRes, len(boards))
	for i, b := range boards {
		res[i] = &dto.BoardListRes{ID: b.ID, Name: b.Name, CardCount: b.CardCount}
	}
	return response.SvcRespSuccess(res, http.StatusOK)
}

// ListDueCards returns the cards due within req.Days of req.Today (default: today in UTC), overdue included.
func (s *boardSvc) ListDueCards(ctx context.Context, userID uint, req *dto.DueCardListReq) response.SvcRes {
	today := s.now().UTC().Truncate(24 * time.Hour)
	if req.Today != "" {
		date, err := time.Parse(dateLayout, req.Today)
		if err != nil {
			return response.SvcRespErr(i18n.Err(i18n.FieldInvalid, i18n.Params{"field": "today"}), http.StatusBadRequest)
		}
		today = date
	}
	days := req.Days
	if days == 0 {
		days = defaultDueDays
	}

	cards, err := s.boardRepo.FindDueCards(ctx, userID, today.AddDate(0, 0, days))
	if err != nil {
		return response.SvcRespErr(fmt.Errorf("list due cards: %w", err), http.StatusInternalServerError)
	}

	res := make([]*dto.DueCardRes, len(cards))
	for i, c := range cards {
		due := c.DueDate.Format(dateLayout)
		res[i] = &dto.DueCardRes{
			ID:          c.ID,
			ColumnID:    c.ColumnID,
			Title:       c.Title,
			Description: c.Description,
			DueDate:     &due,
			Label:       c.Label,
			Position:    c.Position,
			BoardID:     c.BoardID,
			BoardName:   c.BoardName,
			ColumnName:  c.ColumnName,
		}
	}
	return response.SvcRespSuccess(res, http.StatusOK)
}
