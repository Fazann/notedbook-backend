package service

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/Fazann/notedbook-backend/internal/exspense/dto"
	"github.com/Fazann/notedbook-backend/internal/exspense/repo"
	"github.com/Fazann/notedbook-backend/pkg/enums"
	"github.com/Fazann/notedbook-backend/pkg/i18n"
	"github.com/Fazann/notedbook-backend/pkg/money"
	"github.com/Fazann/notedbook-backend/pkg/response"
	model "github.com/Fazann/notedbook-backend/shared/models"
	"gorm.io/gorm"
)

type BudgetServiceImpl interface {
	List(ctx context.Context, userID uint, req *dto.BudgetListReq) response.SvcRes
	Create(ctx context.Context, userID uint, req *dto.BudgetReq) response.SvcRes
	Update(ctx context.Context, userID, id uint, req *dto.BudgetReq) response.SvcRes
	Delete(ctx context.Context, userID, id uint) response.SvcRes
}

type budgetSvc struct {
	budgetRepo  repo.BudgetRepoImpl
	expenseRepo repo.ExpenseRepoImpl
}

func NewBudgetSvc() *budgetSvc {
	return &budgetSvc{
		budgetRepo:  repo.NewBudgetRepo(),
		expenseRepo: repo.NewExpenseRepo(),
	}
}

// List returns every budget of the user, compared with the spending of req.Month.
func (s *budgetSvc) List(ctx context.Context, userID uint, req *dto.BudgetListReq) response.SvcRes {
	from, to, err := monthRange(req.Month)
	if err != nil {
		return response.SvcRespErr(err, http.StatusBadRequest)
	}

	budgets, err := s.budgetRepo.ListByUser(ctx, userID)
	if err != nil {
		return response.SvcRespErr(fmt.Errorf("list budgets: %w", err), http.StatusInternalServerError)
	}
	rows, err := s.expenseRepo.SumByCategory(ctx, userID, from, to)
	if err != nil {
		return response.SvcRespErr(fmt.Errorf("sum expenses: %w", err), http.StatusInternalServerError)
	}

	res := make([]dto.BudgetStatusRes, len(budgets))
	for i := range budgets {
		res[i] = toBudgetStatusRes(&budgets[i], rows)
	}
	return response.SvcRespSuccess(res, http.StatusOK)
}

func (s *budgetSvc) Create(ctx context.Context, userID uint, req *dto.BudgetReq) response.SvcRes {
	currency, amount, res, ok := s.parseInput(ctx, userID, 0, req)
	if !ok {
		return res
	}

	budget := &model.Budget{
		UserID:        userID,
		Amount:        amount,
		Currency:      currency,
		AlertPercent:  req.AlertPercent,
		BaseModelGorm: model.BaseModelGorm{CreatedBy: userID, UpdatedBy: userID},
	}
	if _, err := s.budgetRepo.Create(ctx, budget); err != nil {
		return response.SvcRespErr(fmt.Errorf("create budget: %w", err), http.StatusInternalServerError)
	}

	return response.SvcRespSuccess(toBudgetRes(budget), http.StatusCreated)
}

func (s *budgetSvc) Update(ctx context.Context, userID, id uint, req *dto.BudgetReq) response.SvcRes {
	if _, res, ok := s.findOwn(ctx, userID, id); !ok {
		return res
	}
	currency, amount, res, ok := s.parseInput(ctx, userID, id, req)
	if !ok {
		return res
	}

	updates := map[string]any{
		"amount":        amount,
		"currency":      currency,
		"alert_percent": req.AlertPercent,
		"updated_by":    userID,
	}
	if err := s.budgetRepo.Update(ctx, map[string]any{"id": id, "user_id": userID}, updates); err != nil {
		return response.SvcRespErr(fmt.Errorf("update budget: %w", err), http.StatusInternalServerError)
	}

	budget, res, ok := s.findOwn(ctx, userID, id)
	if !ok {
		return res
	}
	return response.SvcRespSuccess(toBudgetRes(budget), http.StatusOK)
}

func (s *budgetSvc) Delete(ctx context.Context, userID, id uint) response.SvcRes {
	if _, res, ok := s.findOwn(ctx, userID, id); !ok {
		return res
	}

	if err := s.budgetRepo.SoftDelete(ctx, userID, id, userID); err != nil {
		return response.SvcRespErr(fmt.Errorf("delete budget: %w", err), http.StatusInternalServerError)
	}

	return response.SvcRespSuccess(nil, http.StatusOK)
}

// findOwn returns the user's budget; ok is false with the error response otherwise.
func (s *budgetSvc) findOwn(ctx context.Context, userID, id uint) (*model.Budget, response.SvcRes, bool) {
	budget, err := s.budgetRepo.FindOneByID(ctx, userID, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, response.SvcRespErr(i18n.RecordNotFound, http.StatusNotFound), false
		}
		return nil, response.SvcRespErr(err, http.StatusInternalServerError), false
	}
	return budget, response.SvcRes{}, true
}

// parseInput validates a request for budget id (0 when creating); ok is false with the error response otherwise.
func (s *budgetSvc) parseInput(ctx context.Context, userID, id uint, req *dto.BudgetReq) (enums.CurrencyEnum, int64, response.SvcRes, bool) {
	if err := checkCurrency(req.Currency); err != nil {
		return "", 0, response.SvcRespErr(err, http.StatusBadRequest), false
	}
	currency := enums.CurrencyEnum(req.Currency)
	amount, err := parseAmount(req.Amount, currency)
	if err != nil {
		return "", 0, response.SvcRespErr(err, http.StatusBadRequest), false
	}

	other, err := s.budgetRepo.FindOneByCurrency(ctx, userID, currency)
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		return currency, amount, response.SvcRes{}, true
	case err != nil:
		return "", 0, response.SvcRespErr(err, http.StatusInternalServerError), false
	case other.ID != id:
		return "", 0, response.SvcRespErr(i18n.BudgetExists, http.StatusConflict), false
	}
	return currency, amount, response.SvcRes{}, true
}

// totalSpent adds up the month's spending in one currency.
func totalSpent(rows []repo.CategoryTotal, currency enums.CurrencyEnum) int64 {
	var spent int64
	for _, row := range rows {
		if row.Currency == currency {
			spent += row.Amount
		}
	}
	return spent
}

// budgetStatus uses whole-number math: amounts stay well below the int64 limit even when multiplied by 100.
func budgetStatus(spent, amount int64, alertPercent int) enums.BudgetStatusEnum {
	switch {
	case spent > amount:
		return enums.BudgetStatusOver
	case spent*100 >= amount*int64(alertPercent):
		return enums.BudgetStatusNear
	default:
		return enums.BudgetStatusOK
	}
}

func toBudgetRes(budget *model.Budget) *dto.BudgetRes {
	return &dto.BudgetRes{
		ID:           budget.ID,
		Amount:       money.ToDecimalString(budget.Amount, budget.Currency),
		Currency:     string(budget.Currency),
		AlertPercent: budget.AlertPercent,
		CreatedAt:    budget.CreatedAt,
		UpdatedAt:    budget.UpdatedAt,
	}
}

func toBudgetStatusRes(budget *model.Budget, rows []repo.CategoryTotal) dto.BudgetStatusRes {
	spent := totalSpent(rows, budget.Currency)
	return dto.BudgetStatusRes{
		BudgetRes: *toBudgetRes(budget),
		Spent:     money.ToDecimalString(spent, budget.Currency),
		Percent:   spent * 100 / budget.Amount,
		Status:    string(budgetStatus(spent, budget.Amount, budget.AlertPercent)),
	}
}
