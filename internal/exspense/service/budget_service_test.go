package service

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/Fazann/notedbook-backend/internal/exspense/dto"
	"github.com/Fazann/notedbook-backend/internal/exspense/repo"
	"github.com/Fazann/notedbook-backend/pkg/enums"
	"github.com/Fazann/notedbook-backend/pkg/i18n"
	model "github.com/Fazann/notedbook-backend/shared/models"
	"gorm.io/gorm"
)

// fakeBudgetRepo keeps budgets in memory; methods the service does not call panic via the nil embed.
type fakeBudgetRepo struct {
	repo.BudgetRepoImpl
	budgets map[uint]*model.Budget
	created *model.Budget
	updates map[string]any
	deleted []uint
}

func newFakeBudgetRepo() *fakeBudgetRepo {
	return &fakeBudgetRepo{budgets: map[uint]*model.Budget{
		1: {BaseModelID: model.BaseModelID{ID: 1}, UserID: 1, Currency: "USD", Amount: 20000, AlertPercent: 80},
		2: {BaseModelID: model.BaseModelID{ID: 2}, UserID: 1, Currency: "KHR", Amount: 400000, AlertPercent: 50},
		3: {BaseModelID: model.BaseModelID{ID: 3}, UserID: 2, Currency: "MYR", Amount: 10000, AlertPercent: 80},
	}}
}

func (r *fakeBudgetRepo) FindOneByID(_ context.Context, userID, id uint) (*model.Budget, error) {
	if b, ok := r.budgets[id]; ok && b.UserID == userID {
		return b, nil
	}
	return nil, gorm.ErrRecordNotFound
}

func (r *fakeBudgetRepo) FindOneByCurrency(_ context.Context, userID uint, currency enums.CurrencyEnum) (*model.Budget, error) {
	for _, b := range r.budgets {
		if b.UserID == userID && b.Currency == currency {
			return b, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (r *fakeBudgetRepo) ListByUser(_ context.Context, userID uint) ([]model.Budget, error) {
	budgets := []model.Budget{}
	for _, id := range []uint{1, 2, 3} {
		if b, ok := r.budgets[id]; ok && b.UserID == userID {
			budgets = append(budgets, *b)
		}
	}
	return budgets, nil
}

func (r *fakeBudgetRepo) Create(_ context.Context, b *model.Budget) (*model.Budget, error) {
	b.ID = 10
	r.created = b
	return b, nil
}

func (r *fakeBudgetRepo) Update(_ context.Context, _ any, updates map[string]any) error {
	r.updates = updates
	return nil
}

func (r *fakeBudgetRepo) SoftDelete(_ context.Context, _, id, _ uint) error {
	r.deleted = append(r.deleted, id)
	return nil
}

func TestBudgetStatus(t *testing.T) {
	tests := []struct {
		name         string
		spent        int64
		amount       int64
		alertPercent int
		want         enums.BudgetStatusEnum
	}{
		{"nothing spent", 0, 20000, 80, enums.BudgetStatusOK},
		{"just below the alert", 15999, 20000, 80, enums.BudgetStatusOK},
		{"at the alert", 16000, 20000, 80, enums.BudgetStatusNear},
		{"exactly the limit is not over", 20000, 20000, 80, enums.BudgetStatusNear},
		{"one minor unit over", 20001, 20000, 80, enums.BudgetStatusOver},
		{"alert at 100 only fires at the limit", 19999, 20000, 100, enums.BudgetStatusOK},
		{"largest amounts do not overflow", 999_999_999_999_999, 999_999_999_999_999, 100, enums.BudgetStatusNear},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := budgetStatus(tt.spent, tt.amount, tt.alertPercent); got != tt.want {
				t.Errorf("budgetStatus = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestListBudgets(t *testing.T) {
	e := newFakeExpenseRepo()
	e.sums = []repo.CategoryTotal{
		{CategoryID: 1, Currency: "USD", Amount: 10000},
		{CategoryID: 2, Currency: "USD", Amount: 7050},
		{CategoryID: 1, Currency: "KHR", Amount: 40000},
		{CategoryID: 1, Currency: "MYR", Amount: 99999},
	}
	svc := &budgetSvc{budgetRepo: newFakeBudgetRepo(), expenseRepo: e}

	res := svc.List(context.Background(), 1, &dto.BudgetListReq{Month: "2026-11"})
	if res.Error != nil {
		t.Fatal(res.Error)
	}

	got := res.Data.([]dto.BudgetStatusRes)
	if len(got) != 2 {
		t.Fatalf("got %d budgets, want the user's 2", len(got))
	}
	// Categories add up within a currency; other currencies are left out.
	if got[0].Spent != "170.50" || got[0].Percent != 85 || got[0].Status != "near" {
		t.Errorf("USD = %+v", got[0])
	}
	if got[1].Spent != "40000" || got[1].Percent != 10 || got[1].Status != "ok" {
		t.Errorf("KHR = %+v", got[1])
	}
	if want := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC); !e.sumFrom.Equal(want) {
		t.Errorf("sumFrom = %v, want %v", e.sumFrom, want)
	}
	if want := time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC); !e.sumTo.Equal(want) {
		t.Errorf("sumTo = %v, want %v", e.sumTo, want)
	}
}

func TestCreateBudget(t *testing.T) {
	tests := []struct {
		name       string
		req        dto.BudgetReq
		wantStatus int
		wantErr    error
	}{
		{"valid", dto.BudgetReq{Amount: "100.00", Currency: "MYR", AlertPercent: 80}, http.StatusCreated, nil},
		{"currency already has a budget", dto.BudgetReq{Amount: "100.00", Currency: "USD", AlertPercent: 80},
			http.StatusConflict, i18n.BudgetExists},
		{"unknown currency", dto.BudgetReq{Amount: "100.00", Currency: "EUR", AlertPercent: 80},
			http.StatusBadRequest, i18n.InvalidCurrency},
		{"zero amount", dto.BudgetReq{Amount: "0", Currency: "MYR", AlertPercent: 80},
			http.StatusBadRequest, i18n.InvalidAmount},
		{"decimals in riel", dto.BudgetReq{Amount: "10.5", Currency: "KHR", AlertPercent: 80},
			http.StatusBadRequest, i18n.InvalidAmount},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := newFakeBudgetRepo()
			if tt.name == "decimals in riel" {
				delete(r.budgets, 2)
			}
			res := (&budgetSvc{budgetRepo: r}).Create(context.Background(), 1, &tt.req)

			if res.StatusCode != tt.wantStatus {
				t.Fatalf("status = %d, want %d (err %v)", res.StatusCode, tt.wantStatus, res.Error)
			}
			if tt.wantErr != nil {
				if key, _, _ := i18n.FromError(res.Error); key != tt.wantErr {
					t.Errorf("error = %v, want %v", res.Error, tt.wantErr)
				}
				if r.created != nil {
					t.Error("budget created despite the error")
				}
				return
			}
			if r.created.UserID != 1 || r.created.Amount != 10000 || r.created.AlertPercent != 80 {
				t.Errorf("created = %+v", r.created)
			}
		})
	}
}

func TestUpdateBudget(t *testing.T) {
	tests := []struct {
		name       string
		userID, id uint
		req        dto.BudgetReq
		wantStatus int
		wantErr    error
	}{
		{"keeps its own currency", 1, 1, dto.BudgetReq{Amount: "300.00", Currency: "USD", AlertPercent: 90}, http.StatusOK, nil},
		{"moves to a free currency", 1, 1, dto.BudgetReq{Amount: "300.00", Currency: "MYR", AlertPercent: 90}, http.StatusOK, nil},
		{"moves to a taken currency", 1, 1, dto.BudgetReq{Amount: "300", Currency: "KHR", AlertPercent: 90},
			http.StatusConflict, i18n.BudgetExists},
		{"another user's budget", 1, 3, dto.BudgetReq{Amount: "300.00", Currency: "MYR", AlertPercent: 90},
			http.StatusNotFound, i18n.RecordNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := newFakeBudgetRepo()
			res := (&budgetSvc{budgetRepo: r}).Update(context.Background(), tt.userID, tt.id, &tt.req)

			if res.StatusCode != tt.wantStatus {
				t.Fatalf("status = %d, want %d (err %v)", res.StatusCode, tt.wantStatus, res.Error)
			}
			if tt.wantErr != nil {
				if key, _, _ := i18n.FromError(res.Error); key != tt.wantErr {
					t.Errorf("error = %v, want %v", res.Error, tt.wantErr)
				}
				if r.updates != nil {
					t.Errorf("updated despite the error: %v", r.updates)
				}
				return
			}
			if r.updates["amount"] != int64(30000) || r.updates["alert_percent"] != 90 {
				t.Errorf("updates = %v", r.updates)
			}
		})
	}
}

func TestDeleteBudget(t *testing.T) {
	r := newFakeBudgetRepo()
	svc := &budgetSvc{budgetRepo: r}

	if res := svc.Delete(context.Background(), 1, 3); res.StatusCode != http.StatusNotFound {
		t.Errorf("another user's budget: status = %d", res.StatusCode)
	}
	if res := svc.Delete(context.Background(), 1, 1); res.StatusCode != http.StatusOK {
		t.Errorf("own budget: status = %d (err %v)", res.StatusCode, res.Error)
	}
	if len(r.deleted) != 1 || r.deleted[0] != 1 {
		t.Errorf("deleted = %v, want [1]", r.deleted)
	}
}
