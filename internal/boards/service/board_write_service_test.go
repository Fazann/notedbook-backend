package service

import (
	"context"
	"math"
	"net/http"
	"testing"
	"time"

	"github.com/Fazann/notedbook-backend/internal/boards/dto"
	"github.com/Fazann/notedbook-backend/internal/boards/repo"
	"github.com/Fazann/notedbook-backend/pkg/enums"
	"github.com/Fazann/notedbook-backend/pkg/i18n"
	"github.com/Fazann/notedbook-backend/pkg/response"
	model "github.com/Fazann/notedbook-backend/shared/models"
	"gorm.io/gorm"
)

// memBoardRepo keeps boards, columns and cards in memory; user 1 owns board 1, user 2 owns board 2.
type memBoardRepo struct {
	fakeBoardRepo
	boards  map[uint]*model.Board
	columns map[uint]*model.BoardColumn
	cards   map[uint]*model.BoardCard
	nextID  uint
	updates map[string]any
}

func newMemRepo() *memBoardRepo {
	r := &memBoardRepo{
		boards:  map[uint]*model.Board{1: {UserID: 1, Name: "Mine"}, 2: {UserID: 2, Name: "Theirs"}},
		columns: map[uint]*model.BoardColumn{10: {BoardID: 1, Name: "To do", Position: 1000}, 20: {BoardID: 2, Name: "Other", Position: 1000}},
		cards:   map[uint]*model.BoardCard{100: {ColumnID: 10, Title: "Card", Position: 1000}},
		nextID:  1000,
	}
	for id, b := range r.boards {
		b.ID = id
	}
	for id, c := range r.columns {
		c.ID = id
	}
	for id, c := range r.cards {
		c.ID = id
	}
	return r
}

func (r *memBoardRepo) Transaction(_ context.Context, fn func(repo.BoardRepoImpl) error) error {
	return fn(r)
}

func (r *memBoardRepo) FindOwnBoard(_ context.Context, userID, id uint) (*model.Board, error) {
	if b, ok := r.boards[id]; ok && b.UserID == userID {
		return b, nil
	}
	return nil, gorm.ErrRecordNotFound
}

func (r *memBoardRepo) CreateBoard(_ context.Context, b *model.Board) error {
	r.nextID++
	b.ID = r.nextID
	r.boards[b.ID] = b
	return nil
}

func (r *memBoardRepo) FindOwnColumn(_ context.Context, userID, id uint) (*model.BoardColumn, error) {
	c, ok := r.columns[id]
	if !ok || r.boards[c.BoardID].UserID != userID {
		return nil, gorm.ErrRecordNotFound
	}
	copied := *c
	return &copied, nil
}

func (r *memBoardRepo) LastColumnPosition(_ context.Context, boardID uint) (float64, error) {
	last := 0.0
	for _, c := range r.columns {
		if c.BoardID == boardID {
			last = math.Max(last, c.Position)
		}
	}
	return last, nil
}

func (r *memBoardRepo) CreateColumn(_ context.Context, c *model.BoardColumn) error {
	r.nextID++
	c.ID = r.nextID
	r.columns[c.ID] = c
	return nil
}

func (r *memBoardRepo) FindOwnCard(ctx context.Context, userID, id uint) (*model.BoardCard, uint, error) {
	card, ok := r.cards[id]
	if !ok {
		return nil, 0, gorm.ErrRecordNotFound
	}
	column, err := r.FindOwnColumn(ctx, userID, card.ColumnID)
	if err != nil {
		return nil, 0, err
	}
	return card, column.BoardID, nil
}

func (r *memBoardRepo) FindCard(_ context.Context, id uint) (*model.BoardCard, error) {
	return r.cards[id], nil
}

func (r *memBoardRepo) LastCardPosition(_ context.Context, columnID uint) (float64, error) {
	last := 0.0
	for _, c := range r.cards {
		if c.ColumnID == columnID {
			last = math.Max(last, c.Position)
		}
	}
	return last, nil
}

func (r *memBoardRepo) CreateCard(_ context.Context, c *model.BoardCard) error {
	r.nextID++
	c.ID = r.nextID
	r.cards[c.ID] = c
	return nil
}

func (r *memBoardRepo) UpdateCard(_ context.Context, id uint, updates map[string]any) error {
	r.updates = updates
	card := r.cards[id]
	if v, ok := updates["column_id"]; ok {
		card.ColumnID = v.(uint)
	}
	if v, ok := updates["position"]; ok {
		card.Position = v.(float64)
	}
	return nil
}

func (r *memBoardRepo) DeleteCard(_ context.Context, id uint) error {
	delete(r.cards, id)
	return nil
}

func ptr[T any](v T) *T { return &v }

func memSvc(r *memBoardRepo) *boardSvc {
	return &boardSvc{boardRepo: r, now: time.Now}
}

type result struct {
	status int
	err    error
}

func statusErr(res response.SvcRes) result { return result{res.StatusCode, res.Error} }

func keyOf(err error) i18n.Key {
	key, _, _ := i18n.FromError(err)
	return key
}

func TestCreateBoardPositionsColumns(t *testing.T) {
	r := newMemRepo()
	res := memSvc(r).Create(context.Background(), 1, &dto.BoardReq{
		Name:    "  Trip  ",
		Columns: []string{"To do", " Doing ", "Done"},
	})
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("status = %d (%v)", res.StatusCode, res.Error)
	}
	board := res.Data.(*dto.BoardRes)
	if board.Name != "Trip" || len(board.Columns) != 3 {
		t.Fatalf("board = %+v", board)
	}
	for i, want := range []struct {
		name string
		pos  float64
	}{{"To do", 1000}, {"Doing", 2000}, {"Done", 3000}} {
		if board.Columns[i].Name != want.name || board.Columns[i].Position != want.pos {
			t.Errorf("column %d = %+v, want %s at %v", i, board.Columns[i], want.name, want.pos)
		}
	}
}

func TestBlankNamesAreRejected(t *testing.T) {
	r := newMemRepo()
	svc := memSvc(r)
	ctx := context.Background()
	for name, res := range map[string]result{
		"board":        statusErr(svc.Create(ctx, 1, &dto.BoardReq{Name: "   "})),
		"column":       statusErr(svc.CreateColumn(ctx, 1, 1, &dto.ColumnReq{Name: " "})),
		"card":         statusErr(svc.CreateCard(ctx, 1, 10, &dto.CardReq{Title: "\t"})),
		"board column": statusErr(svc.Create(ctx, 1, &dto.BoardReq{Name: "Ok", Columns: []string{" "}})),
	} {
		if res.status != http.StatusBadRequest || keyOf(res.err) != i18n.FieldRequired {
			t.Errorf("%s: status %d, err %v", name, res.status, res.err)
		}
	}
}

func TestOtherUsersRecordsAreNotFound(t *testing.T) {
	r := newMemRepo()
	svc := memSvc(r)
	ctx := context.Background()
	// User 2 tries board 1, column 10 and card 100, which belong to user 1.
	for name, res := range map[string]result{
		"rename board":  statusErr(svc.Update(ctx, 2, 1, &dto.BoardUpdateReq{Name: "x"})),
		"delete board":  statusErr(svc.Delete(ctx, 2, 1)),
		"add column":    statusErr(svc.CreateColumn(ctx, 2, 1, &dto.ColumnReq{Name: "x"})),
		"delete column": statusErr(svc.DeleteColumn(ctx, 2, 10)),
		"add card":      statusErr(svc.CreateCard(ctx, 2, 10, &dto.CardReq{Title: "x"})),
		"edit card":     statusErr(svc.UpdateCard(ctx, 2, 100, &dto.CardReq{Title: "x"})),
		"move card":     statusErr(svc.MoveCard(ctx, 2, 100, &dto.CardMoveReq{ColumnID: 20, Position: ptr(1.0)})),
		"delete card":   statusErr(svc.DeleteCard(ctx, 2, 100)),
	} {
		if res.status != http.StatusNotFound || keyOf(res.err) != i18n.RecordNotFound {
			t.Errorf("%s: status %d, err %v", name, res.status, res.err)
		}
	}
	if _, ok := r.cards[100]; !ok {
		t.Error("card 100 was deleted by another user")
	}
}

func TestCreateCardGoesToTheBottom(t *testing.T) {
	r := newMemRepo()
	res := memSvc(r).CreateCard(context.Background(), 1, 10, &dto.CardReq{
		Title:   "Book flights",
		DueDate: ptr("2026-12-01"),
		Label:   ptr("blue"),
	})
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("status = %d (%v)", res.StatusCode, res.Error)
	}
	card := res.Data.(*dto.CardRes)
	if card.Position != 2000 || card.DueDate == nil || *card.DueDate != "2026-12-01" || card.Label == nil || *card.Label != "blue" {
		t.Errorf("card = %+v", card)
	}
}

func TestUpdateCardClearsDueDateAndLabel(t *testing.T) {
	r := newMemRepo()
	res := memSvc(r).UpdateCard(context.Background(), 1, 100, &dto.CardReq{Title: "Card"})
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status = %d (%v)", res.StatusCode, res.Error)
	}
	if due, ok := r.updates["due_date"].(*time.Time); !ok || due != nil {
		t.Errorf("due_date = %v, want nil", r.updates["due_date"])
	}
	if label, ok := r.updates["label"].(*enums.BoardLabelEnum); !ok || label != nil {
		t.Errorf("label = %v, want nil", r.updates["label"])
	}
}

func TestMoveCard(t *testing.T) {
	r := newMemRepo()
	r.columns[11] = &model.BoardColumn{BaseModelID: model.BaseModelID{ID: 11}, BoardID: 1, Name: "Done", Position: 2000}
	svc := memSvc(r)
	ctx := context.Background()

	res := svc.MoveCard(ctx, 1, 100, &dto.CardMoveReq{ColumnID: 11, Position: ptr(500.0)})
	if res.StatusCode != http.StatusOK || r.cards[100].ColumnID != 11 || r.cards[100].Position != 500 {
		t.Fatalf("status %d (%v), card %+v", res.StatusCode, res.Error, r.cards[100])
	}

	for name, req := range map[string]*dto.CardMoveReq{
		"another user's column": {ColumnID: 20, Position: ptr(1.0)},
		"missing column":        {ColumnID: 999, Position: ptr(1.0)},
		"NaN position":          {ColumnID: 10, Position: ptr(math.NaN())},
		"infinite position":     {ColumnID: 10, Position: ptr(math.Inf(1))},
	} {
		res := svc.MoveCard(ctx, 1, 100, req)
		if res.StatusCode != http.StatusBadRequest || keyOf(res.Error) != i18n.FieldInvalid {
			t.Errorf("%s: status %d, err %v", name, res.StatusCode, res.Error)
		}
	}
}
