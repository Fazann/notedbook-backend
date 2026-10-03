package service

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/Fazann/notedbook-backend/internal/boards/dto"
	"github.com/Fazann/notedbook-backend/internal/boards/repo"
	"github.com/Fazann/notedbook-backend/pkg/enums"
)

// fakeBoardRepo records the arguments it is called with; methods the service does not call panic via the nil embed.
type fakeBoardRepo struct {
	repo.BoardRepoImpl
	boards    []repo.BoardSummary
	cards     []repo.DueCard
	err       error
	gotUserID uint
	gotUntil  time.Time
}

func (r *fakeBoardRepo) ListWithCardCount(_ context.Context, userID uint) ([]repo.BoardSummary, error) {
	r.gotUserID = userID
	return r.boards, r.err
}

func (r *fakeBoardRepo) FindDueCards(_ context.Context, userID uint, until time.Time) ([]repo.DueCard, error) {
	r.gotUserID = userID
	r.gotUntil = until
	return r.cards, r.err
}

func newTestSvc(r *fakeBoardRepo) *boardSvc {
	return &boardSvc{
		boardRepo: r,
		now:       func() time.Time { return time.Date(2026, 10, 3, 22, 30, 0, 0, time.UTC) },
	}
}

func TestList(t *testing.T) {
	r := &fakeBoardRepo{boards: []repo.BoardSummary{{ID: 1, Name: "Side project", CardCount: 3}}}
	res := newTestSvc(r).List(context.Background(), 7)

	if res.Error != nil || res.StatusCode != http.StatusOK {
		t.Fatalf("got err %v, status %d", res.Error, res.StatusCode)
	}
	if r.gotUserID != 7 {
		t.Errorf("user id = %d, want 7", r.gotUserID)
	}
	got := res.Data.([]*dto.BoardListRes)
	if len(got) != 1 || got[0].ID != 1 || got[0].Name != "Side project" || got[0].CardCount != 3 {
		t.Errorf("got %+v", got)
	}
}

func TestListDueCardsUntil(t *testing.T) {
	tests := []struct {
		name       string
		req        dto.DueCardListReq
		wantUntil  string
		wantStatus int
	}{
		{name: "defaults to 7 days from today in UTC", req: dto.DueCardListReq{}, wantUntil: "2026-10-10", wantStatus: http.StatusOK},
		{name: "custom days", req: dto.DueCardListReq{Days: 1}, wantUntil: "2026-10-04", wantStatus: http.StatusOK},
		{name: "user's today", req: dto.DueCardListReq{Days: 3, Today: "2026-10-04"}, wantUntil: "2026-10-07", wantStatus: http.StatusOK},
		{name: "invalid today", req: dto.DueCardListReq{Today: "2026-13-01"}, wantStatus: http.StatusBadRequest},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &fakeBoardRepo{}
			res := newTestSvc(r).ListDueCards(context.Background(), 1, &tt.req)
			if res.StatusCode != tt.wantStatus {
				t.Fatalf("status = %d, want %d (err %v)", res.StatusCode, tt.wantStatus, res.Error)
			}
			if tt.wantUntil != "" && r.gotUntil.Format(dateLayout) != tt.wantUntil {
				t.Errorf("until = %s, want %s", r.gotUntil.Format(dateLayout), tt.wantUntil)
			}
		})
	}
}

func TestListDueCardsMapping(t *testing.T) {
	red := enums.BoardLabelRed
	r := &fakeBoardRepo{cards: []repo.DueCard{{
		ID: 5, ColumnID: 2, Title: "Ship", DueDate: time.Date(2026, 10, 5, 0, 0, 0, 0, time.UTC),
		Label: &red, Position: 1000, BoardID: 1, BoardName: "Work", ColumnName: "Doing",
	}}}
	res := newTestSvc(r).ListDueCards(context.Background(), 1, &dto.DueCardListReq{})

	got := res.Data.([]*dto.DueCardRes)
	if len(got) != 1 {
		t.Fatalf("got %d cards", len(got))
	}
	c := got[0]
	if c.DueDate == nil || *c.DueDate != "2026-10-05" || c.Label == nil || *c.Label != red ||
		c.BoardName != "Work" || c.ColumnName != "Doing" || c.ColumnID != 2 {
		t.Errorf("got %+v", c)
	}
}

func TestListDueCardsRepoError(t *testing.T) {
	r := &fakeBoardRepo{err: errors.New("db down")}
	res := newTestSvc(r).ListDueCards(context.Background(), 1, &dto.DueCardListReq{})
	if res.StatusCode != http.StatusInternalServerError || res.Error == nil {
		t.Errorf("got status %d, err %v", res.StatusCode, res.Error)
	}
}
