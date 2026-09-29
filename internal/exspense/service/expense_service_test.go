package service

import (
	"context"
	"net/http"
	"reflect"
	"testing"
	"time"

	"github.com/Fazann/notedbook-backend/internal/exspense/dto"
	"github.com/Fazann/notedbook-backend/internal/exspense/repo"
	"github.com/Fazann/notedbook-backend/pkg/enums"
	"github.com/Fazann/notedbook-backend/pkg/i18n"
	model "github.com/Fazann/notedbook-backend/shared/models"
	"gorm.io/gorm"
)

// fakeExpenseRepo keeps expenses in memory; methods the service does not call panic via the nil embed.
type fakeExpenseRepo struct {
	repo.ExpenseRepoImpl
	expenses map[uint]*model.Expense
	created  *model.Expense
	updates  map[string]any
	deleted  []uint
	filter   *repo.ExpenseFilter
}

func newFakeExpenseRepo() *fakeExpenseRepo {
	return &fakeExpenseRepo{
		expenses: map[uint]*model.Expense{
			1: {BaseModelID: model.BaseModelID{ID: 1}, UserID: 1, CategoryID: 1, Amount: 1250, Currency: "USD",
				SpentAt: time.Date(2026, 9, 29, 0, 0, 0, 0, time.UTC), Note: "Lunch"},
			2: {BaseModelID: model.BaseModelID{ID: 2}, UserID: 2, CategoryID: 3, Amount: 40000, Currency: "KHR",
				SpentAt: time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)},
			3: {BaseModelID: model.BaseModelID{ID: 3}, UserID: 1, CategoryID: 1, Amount: 5000, Currency: "USD",
				SpentAt: time.Date(2026, 9, 2, 0, 0, 0, 0, time.UTC)},
		},
	}
}

func (r *fakeExpenseRepo) FindOneByID(_ context.Context, userID, id uint) (*model.Expense, error) {
	if e, ok := r.expenses[id]; ok && e.UserID == userID {
		return e, nil
	}
	return nil, gorm.ErrRecordNotFound
}

func (r *fakeExpenseRepo) List(_ context.Context, _ uint, filter *repo.ExpenseFilter, _, _ int) ([]model.Expense, int64, error) {
	r.filter = filter
	return []model.Expense{*r.expenses[1]}, 21, nil
}

func (r *fakeExpenseRepo) Create(_ context.Context, e *model.Expense) (*model.Expense, error) {
	e.ID = 10
	r.created = e
	return e, nil
}

func (r *fakeExpenseRepo) Update(_ context.Context, _ any, updates map[string]any) error {
	r.updates = updates
	return nil
}

func (r *fakeExpenseRepo) SoftDelete(_ context.Context, _, id, _ uint) error {
	r.deleted = append(r.deleted, id)
	return nil
}

func newTestExpenseSvc(r *fakeExpenseRepo) *expenseSvc {
	return &expenseSvc{expenseRepo: r, categoryRepo: newFakeCategoryRepo()}
}

func validExpenseReq() dto.ExpenseReq {
	return dto.ExpenseReq{Amount: "10.00", Currency: "USD", CategoryID: 1, SpentAt: "2026-09-29", Note: " Coffee "}
}

func TestCreateExpense(t *testing.T) {
	tests := []struct {
		name       string
		edit       func(*dto.ExpenseReq)
		wantStatus int
		wantErr    error
	}{
		{name: "created", wantStatus: http.StatusCreated},
		{name: "default category", edit: func(r *dto.ExpenseReq) { r.CategoryID = 4 }, wantStatus: http.StatusCreated},
		{name: "unknown currency", edit: func(r *dto.ExpenseReq) { r.Currency = "EUR" },
			wantStatus: http.StatusBadRequest, wantErr: i18n.InvalidCurrency},
		{name: "zero amount", edit: func(r *dto.ExpenseReq) { r.Amount = "0.00" },
			wantStatus: http.StatusBadRequest, wantErr: i18n.InvalidAmount},
		{name: "too many decimals", edit: func(r *dto.ExpenseReq) { r.Amount = "10.005" },
			wantStatus: http.StatusBadRequest, wantErr: i18n.InvalidAmount},
		{name: "riel minimum", edit: func(r *dto.ExpenseReq) { r.Amount, r.Currency = "100", "KHR" },
			wantStatus: http.StatusCreated},
		{name: "riel below minimum", edit: func(r *dto.ExpenseReq) { r.Amount, r.Currency = "99", "KHR" },
			wantStatus: http.StatusBadRequest, wantErr: i18n.InvalidAmount},
		{name: "one cent", edit: func(r *dto.ExpenseReq) { r.Amount = "0.01" }, wantStatus: http.StatusCreated},
		{name: "riel has no decimals", edit: func(r *dto.ExpenseReq) { r.Amount, r.Currency = "10.5", "KHR" },
			wantStatus: http.StatusBadRequest, wantErr: i18n.InvalidAmount},
		{name: "not a number", edit: func(r *dto.ExpenseReq) { r.Amount = "ten" },
			wantStatus: http.StatusBadRequest, wantErr: i18n.InvalidAmount},
		{name: "invalid date", edit: func(r *dto.ExpenseReq) { r.SpentAt = "2026-02-30" },
			wantStatus: http.StatusBadRequest, wantErr: i18n.FieldInvalid},
		{name: "another user's category", edit: func(r *dto.ExpenseReq) { r.CategoryID = 3 },
			wantStatus: http.StatusBadRequest, wantErr: i18n.InvalidExpenseCategory},
		{name: "missing category", edit: func(r *dto.ExpenseReq) { r.CategoryID = 99 },
			wantStatus: http.StatusBadRequest, wantErr: i18n.InvalidExpenseCategory},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := newFakeExpenseRepo()
			req := validExpenseReq()
			if tt.edit != nil {
				tt.edit(&req)
			}

			res := newTestExpenseSvc(r).Create(context.Background(), 1, &req)

			if res.StatusCode != tt.wantStatus {
				t.Fatalf("status = %d, want %d (err %v)", res.StatusCode, tt.wantStatus, res.Error)
			}
			if tt.wantErr != nil {
				if key, _, _ := i18n.FromError(res.Error); key != tt.wantErr {
					t.Fatalf("err = %v, want %v", res.Error, tt.wantErr)
				}
				if r.created != nil {
					t.Fatal("expense created on error")
				}
				return
			}
			if r.created.UserID != 1 || r.created.CreatedBy != 1 || r.created.Note != "Coffee" {
				t.Errorf("created = %+v", r.created)
			}
		})
	}
}

func TestCreateExpenseResult(t *testing.T) {
	r := newFakeExpenseRepo()
	req := validExpenseReq()

	res := newTestExpenseSvc(r).Create(context.Background(), 1, &req)

	got := res.Data.(*dto.ExpenseRes)
	if got.ID != 10 || got.Amount != "10.00" || got.SpentAt != "2026-09-29" || got.Note != "Coffee" {
		t.Errorf("response = %+v", got)
	}
	if r.created.Amount != 1000 || !r.created.SpentAt.Equal(time.Date(2026, 9, 29, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("stored amount = %d, spent_at = %v", r.created.Amount, r.created.SpentAt)
	}
}

func TestUpdateExpense(t *testing.T) {
	tests := []struct {
		name        string
		userID, id  uint
		req         dto.ExpenseUpdateReq
		wantStatus  int
		wantErr     error
		wantUpdates map[string]any
	}{
		{name: "amount only", userID: 1, id: 1,
			req:         dto.ExpenseUpdateReq{Amount: new("5.00")},
			wantStatus:  http.StatusOK,
			wantUpdates: map[string]any{"amount": int64(500), "updated_by": uint(1)}},
		{name: "every field", userID: 1, id: 1,
			req: dto.ExpenseUpdateReq{Amount: new("40000"), Currency: new("KHR"), CategoryID: new(uint(4)),
				SpentAt: new("2026-09-01"), Note: new(" ")},
			wantStatus: http.StatusOK,
			wantUpdates: map[string]any{"amount": int64(40000), "currency": enums.CurrencyEnumKHRiel, "category_id": uint(4),
				"spent_at": time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC), "note": "", "updated_by": uint(1)}},
		{name: "currency only keeps the decimal value", userID: 1, id: 1,
			req:        dto.ExpenseUpdateReq{Currency: new("MYR")},
			wantStatus: http.StatusOK,
			wantUpdates: map[string]any{"amount": int64(1250), "currency": enums.CurrencyEnumMYR,
				"updated_by": uint(1)}},
		{name: "currency only, value does not fit", userID: 1, id: 1,
			req:        dto.ExpenseUpdateReq{Currency: new("KHR")},
			wantStatus: http.StatusBadRequest, wantErr: i18n.InvalidAmount},
		{name: "currency only, below riel minimum", userID: 1, id: 3,
			req:        dto.ExpenseUpdateReq{Currency: new("KHR")},
			wantStatus: http.StatusBadRequest, wantErr: i18n.InvalidAmount},
		{name: "amount uses stored currency", userID: 1, id: 1,
			req:        dto.ExpenseUpdateReq{Amount: new("1.234")},
			wantStatus: http.StatusBadRequest, wantErr: i18n.InvalidAmount},
		{name: "nothing sent", userID: 1, id: 1, wantStatus: http.StatusOK},
		{name: "another user's expense", userID: 1, id: 2,
			req:        dto.ExpenseUpdateReq{Amount: new("1.00")},
			wantStatus: http.StatusNotFound, wantErr: i18n.RecordNotFound},
		{name: "unknown currency", userID: 1, id: 1,
			req:        dto.ExpenseUpdateReq{Currency: new("EUR")},
			wantStatus: http.StatusBadRequest, wantErr: i18n.InvalidCurrency},
		{name: "another user's category", userID: 1, id: 1,
			req:        dto.ExpenseUpdateReq{CategoryID: new(uint(3))},
			wantStatus: http.StatusBadRequest, wantErr: i18n.InvalidExpenseCategory},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := newFakeExpenseRepo()

			res := newTestExpenseSvc(r).Update(context.Background(), tt.userID, tt.id, &tt.req)

			if res.StatusCode != tt.wantStatus {
				t.Fatalf("status = %d, want %d (err %v)", res.StatusCode, tt.wantStatus, res.Error)
			}
			if tt.wantErr != nil {
				if key, _, _ := i18n.FromError(res.Error); key != tt.wantErr {
					t.Fatalf("err = %v, want %v", res.Error, tt.wantErr)
				}
				if r.updates != nil {
					t.Fatal("expense updated on error")
				}
				return
			}
			if !reflect.DeepEqual(r.updates, tt.wantUpdates) {
				t.Errorf("updates = %v, want %v", r.updates, tt.wantUpdates)
			}
		})
	}
}

func TestDeleteExpense(t *testing.T) {
	tests := []struct {
		name        string
		userID, id  uint
		wantStatus  int
		wantDeleted []uint
	}{
		{name: "own expense", userID: 1, id: 1, wantStatus: http.StatusOK, wantDeleted: []uint{1}},
		{name: "another user's expense", userID: 1, id: 2, wantStatus: http.StatusNotFound},
		{name: "missing", userID: 1, id: 99, wantStatus: http.StatusNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := newFakeExpenseRepo()

			res := newTestExpenseSvc(r).Delete(context.Background(), tt.userID, tt.id)

			if res.StatusCode != tt.wantStatus {
				t.Fatalf("status = %d, want %d", res.StatusCode, tt.wantStatus)
			}
			if !reflect.DeepEqual(r.deleted, tt.wantDeleted) {
				t.Errorf("deleted = %v, want %v", r.deleted, tt.wantDeleted)
			}
		})
	}
}

func TestListExpenses(t *testing.T) {
	r := newFakeExpenseRepo()

	res := newTestExpenseSvc(r).List(context.Background(), 1, &dto.ExpenseListReq{
		Page: 2, Limit: 10, Month: "2026-12", CategoryID: 4, Currency: "USD", Q: " 50%_OFF ", Sort: "amount",
	})

	if res.Error != nil || res.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, err = %v", res.StatusCode, res.Error)
	}
	if res.Total != 21 || res.TotalPage != 3 || !res.IsNext || !res.IsPrev {
		t.Errorf("paging = total %d, pages %d, next %v, prev %v", res.Total, res.TotalPage, res.IsNext, res.IsPrev)
	}
	f := r.filter
	if !f.SpentFrom.Equal(time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC)) || !f.SpentTo.Equal(time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("month range = %v – %v", f.SpentFrom, f.SpentTo)
	}
	if f.CategoryID != 4 || f.Currency != "USD" || f.Search != `50\%\_off` {
		t.Errorf("filter = %+v", f)
	}
	if *f.SortCurrency != "asc" || *f.SortAmount != "asc" || f.SortSpentAt != nil || *f.SortID != "asc" {
		t.Errorf("sort = currency %v, amount %v, spent_at %v, id %v", f.SortCurrency, f.SortAmount, f.SortSpentAt, f.SortID)
	}
	if items := res.Data.([]*dto.ExpenseRes); items[0].Amount != "12.50" {
		t.Errorf("amount = %q, want 12.50", items[0].Amount)
	}
}

func TestExpenseListFilterDefaults(t *testing.T) {
	filter, err := expenseListFilter(&dto.ExpenseListReq{})
	if err != nil {
		t.Fatal(err)
	}
	if !filter.SpentFrom.IsZero() || filter.SortAmount != nil || *filter.SortSpentAt != "desc" || *filter.SortID != "desc" {
		t.Errorf("filter = %+v", filter)
	}
}
