package service

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/Fazann/notedbook-backend/internal/exspense/dto"
	"github.com/Fazann/notedbook-backend/internal/exspense/repo"
	"github.com/Fazann/notedbook-backend/pkg/constants"
	"github.com/Fazann/notedbook-backend/pkg/enums"
	"github.com/Fazann/notedbook-backend/pkg/i18n"
	"github.com/Fazann/notedbook-backend/pkg/money"
	"github.com/Fazann/notedbook-backend/pkg/response"
	model "github.com/Fazann/notedbook-backend/shared/models"
	"gorm.io/gorm"
)

const (
	dateLayout  = "2006-01-02"
	monthLayout = "2006-01"
)

type ExpenseServiceImpl interface {
	List(ctx context.Context, userID uint, req *dto.ExpenseListReq) response.SvcResWithPaginate
	GetDetail(ctx context.Context, userID, id uint) response.SvcRes
	Create(ctx context.Context, userID uint, req *dto.ExpenseReq) response.SvcRes
	Update(ctx context.Context, userID, id uint, req *dto.ExpenseUpdateReq) response.SvcRes
	Delete(ctx context.Context, userID, id uint) response.SvcRes
}

type expenseSvc struct {
	expenseRepo  repo.ExpenseRepoImpl
	categoryRepo repo.CategoryRepoImpl
}

func NewExpenseSvc() *expenseSvc {
	return &expenseSvc{
		expenseRepo:  repo.NewExpenseRepo(),
		categoryRepo: repo.NewCategoryRepo(),
	}
}

func (s *expenseSvc) List(ctx context.Context, userID uint, req *dto.ExpenseListReq) response.SvcResWithPaginate {
	page, limit := req.Page, req.Limit
	if page <= 0 {
		page = constants.DefaultPage
	}
	if limit <= 0 {
		limit = constants.DefaultLimit
	}

	filter, err := expenseListFilter(req)
	if err != nil {
		return response.SvcResWithPaginate{Error: err, StatusCode: http.StatusBadRequest}
	}

	expenses, total, err := s.expenseRepo.List(ctx, userID, filter, page, limit)
	if err != nil {
		return response.SvcResWithPaginate{Error: fmt.Errorf("list expenses: %w", err), StatusCode: http.StatusInternalServerError}
	}

	totalPage := int(math.Ceil(float64(total) / float64(limit)))
	return response.SvcResSucWithPage(response.Paginate{
		Data:       toExpenseResList(expenses),
		StatusCode: http.StatusOK,
		IsNext:     page < totalPage,
		IsPrev:     page > 1,
		Total:      total,
		TotalPage:  totalPage,
	})
}

func (s *expenseSvc) GetDetail(ctx context.Context, userID, id uint) response.SvcRes {
	expense, res, ok := s.findOwn(ctx, userID, id)
	if !ok {
		return res
	}

	return response.SvcRespSuccess(toExpenseRes(expense), http.StatusOK)
}

func (s *expenseSvc) Create(ctx context.Context, userID uint, req *dto.ExpenseReq) response.SvcRes {
	if err := checkCurrency(req.Currency); err != nil {
		return response.SvcRespErr(err, http.StatusBadRequest)
	}
	amount, err := parseAmount(req.Amount, enums.CurrencyEnum(req.Currency))
	if err != nil {
		return response.SvcRespErr(err, http.StatusBadRequest)
	}
	spentAt, err := parseDate(req.SpentAt)
	if err != nil {
		return response.SvcRespErr(err, http.StatusBadRequest)
	}
	if res, ok := s.checkCategory(ctx, userID, req.CategoryID); !ok {
		return res
	}

	expense := &model.Expense{
		UserID:        userID,
		CategoryID:    req.CategoryID,
		Amount:        amount,
		Currency:      enums.CurrencyEnum(req.Currency),
		SpentAt:       spentAt,
		Note:          strings.TrimSpace(req.Note),
		BaseModelGorm: model.BaseModelGorm{CreatedBy: userID, UpdatedBy: userID},
	}
	if _, err := s.expenseRepo.Create(ctx, expense); err != nil {
		return response.SvcRespErr(fmt.Errorf("create expense: %w", err), http.StatusInternalServerError)
	}

	return response.SvcRespSuccess(toExpenseRes(expense), http.StatusCreated)
}

func (s *expenseSvc) Update(ctx context.Context, userID, id uint, req *dto.ExpenseUpdateReq) response.SvcRes {
	expense, res, ok := s.findOwn(ctx, userID, id)
	if !ok {
		return res
	}

	updates := map[string]any{}
	currency := expense.Currency
	if req.Currency != nil {
		if err := checkCurrency(*req.Currency); err != nil {
			return response.SvcRespErr(err, http.StatusBadRequest)
		}
		currency = enums.CurrencyEnum(*req.Currency)
		updates["currency"] = currency
	}
	// Minor units depend on the currency, so a new currency re-reads the current decimal value.
	if req.Amount != nil || req.Currency != nil {
		value := money.ToDecimalString(expense.Amount, expense.Currency)
		if req.Amount != nil {
			value = *req.Amount
		}
		amount, err := parseAmount(value, currency)
		if err != nil {
			return response.SvcRespErr(err, http.StatusBadRequest)
		}
		updates["amount"] = amount
	}
	if req.SpentAt != nil {
		spentAt, err := parseDate(*req.SpentAt)
		if err != nil {
			return response.SvcRespErr(err, http.StatusBadRequest)
		}
		updates["spent_at"] = spentAt
	}
	if req.Note != nil {
		updates["note"] = strings.TrimSpace(*req.Note)
	}
	if req.CategoryID != nil {
		if res, ok := s.checkCategory(ctx, userID, *req.CategoryID); !ok {
			return res
		}
		updates["category_id"] = *req.CategoryID
	}
	if len(updates) == 0 {
		return s.GetDetail(ctx, userID, id)
	}

	updates["updated_by"] = userID
	if err := s.expenseRepo.Update(ctx, map[string]any{"id": id, "user_id": userID}, updates); err != nil {
		return response.SvcRespErr(fmt.Errorf("update expense: %w", err), http.StatusInternalServerError)
	}

	return s.GetDetail(ctx, userID, id)
}

func (s *expenseSvc) Delete(ctx context.Context, userID, id uint) response.SvcRes {
	if _, res, ok := s.findOwn(ctx, userID, id); !ok {
		return res
	}

	if err := s.expenseRepo.SoftDelete(ctx, userID, id, userID); err != nil {
		return response.SvcRespErr(fmt.Errorf("delete expense: %w", err), http.StatusInternalServerError)
	}

	return response.SvcRespSuccess(nil, http.StatusOK)
}

// findOwn returns the user's expense; ok is false with the error response otherwise.
func (s *expenseSvc) findOwn(ctx context.Context, userID, id uint) (*model.Expense, response.SvcRes, bool) {
	expense, err := s.expenseRepo.FindOneByID(ctx, userID, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, response.SvcRespErr(i18n.RecordNotFound, http.StatusNotFound), false
		}
		return nil, response.SvcRespErr(err, http.StatusInternalServerError), false
	}
	return expense, response.SvcRes{}, true
}

// checkCategory checks that the category is one the user can see (own or default).
func (s *expenseSvc) checkCategory(ctx context.Context, userID, categoryID uint) (response.SvcRes, bool) {
	if _, err := s.categoryRepo.FindOneByID(ctx, userID, categoryID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return response.SvcRespErr(i18n.InvalidExpenseCategory, http.StatusBadRequest), false
		}
		return response.SvcRespErr(err, http.StatusInternalServerError), false
	}
	return response.SvcRes{}, true
}

func checkCurrency(currency string) error {
	if !slices.Contains(enums.CurrencyValues, currency) {
		return i18n.Err(i18n.InvalidCurrency, i18n.Params{"currencies": strings.Join(enums.CurrencyValues, ", ")})
	}
	return nil
}

// parseAmount converts a decimal string to minor units of currency; it must be at least the currency's minimum.
func parseAmount(value string, currency enums.CurrencyEnum) (int64, error) {
	amount, err := money.ToMinor(value, currency)
	if err != nil || amount < money.MinAmount(currency) {
		return 0, i18n.Err(i18n.InvalidAmount, i18n.Params{
			"min":      money.ToDecimalString(money.MinAmount(currency), currency),
			"max":      money.ToDecimalString(money.MaxMinor, currency),
			"decimals": money.Digits(currency),
		})
	}
	return amount, nil
}

func parseDate(value string) (time.Time, error) {
	date, err := time.Parse(dateLayout, value)
	if err != nil {
		return time.Time{}, i18n.Err(i18n.FieldInvalid, i18n.Params{"field": "spent_at"})
	}
	return date, nil
}

// monthRange returns [first day of month, first day of next month); zero times when month is empty.
func monthRange(month string) (time.Time, time.Time, error) {
	if month == "" {
		return time.Time{}, time.Time{}, nil
	}
	from, err := time.Parse(monthLayout, month)
	if err != nil {
		return time.Time{}, time.Time{}, i18n.Err(i18n.FieldInvalid, i18n.Params{"field": "month"})
	}
	return from, from.AddDate(0, 1, 0), nil
}

func expenseListFilter(req *dto.ExpenseListReq) (*repo.ExpenseFilter, error) {
	filter := &repo.ExpenseFilter{
		CategoryID: req.CategoryID,
		Currency:   req.Currency,
		Search:     likeEscaper.Replace(strings.ToLower(strings.TrimSpace(req.Q))),
	}
	var err error
	if filter.SpentFrom, filter.SpentTo, err = monthRange(req.Month); err != nil {
		return nil, err
	}

	order := "asc"
	key, desc := strings.CutPrefix(req.Sort, "-")
	if desc || req.Sort == "" {
		order = "desc"
	}
	switch key {
	case "amount":
		filter.SortCurrency = new("asc")
		filter.SortAmount = &order
	default:
		filter.SortSpentAt = &order
	}
	filter.SortID = &order
	return filter, nil
}

func toExpenseRes(expense *model.Expense) *dto.ExpenseRes {
	return &dto.ExpenseRes{
		ID:         expense.ID,
		Amount:     money.ToDecimalString(expense.Amount, expense.Currency),
		Currency:   string(expense.Currency),
		CategoryID: expense.CategoryID,
		SpentAt:    expense.SpentAt.Format(dateLayout),
		Note:       expense.Note,
		CreatedAt:  expense.CreatedAt,
		UpdatedAt:  expense.UpdatedAt,
	}
}

func toExpenseResList(expenses []model.Expense) []*dto.ExpenseRes {
	res := make([]*dto.ExpenseRes, len(expenses))
	for i := range expenses {
		res[i] = toExpenseRes(&expenses[i])
	}
	return res
}
