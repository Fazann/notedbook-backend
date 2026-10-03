package service

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/Fazann/notedbook-backend/internal/exspense/dto"
	"github.com/Fazann/notedbook-backend/internal/exspense/repo"
	"github.com/Fazann/notedbook-backend/pkg/constants"
	"github.com/Fazann/notedbook-backend/pkg/db"
	"github.com/Fazann/notedbook-backend/pkg/enums"
	"github.com/Fazann/notedbook-backend/pkg/i18n"
	"github.com/Fazann/notedbook-backend/pkg/response"
	model "github.com/Fazann/notedbook-backend/shared/models"
	"gorm.io/gorm"
)

const categoryNameMax = 50

type CategoryServiceImpl interface {
	List(ctx context.Context, userID uint, lang i18n.Lang, req *dto.CategoryListReq) response.SvcResWithPaginate
	ListAll(ctx context.Context, userID uint, lang i18n.Lang) response.SvcRes
	GetDetail(ctx context.Context, userID, id uint, lang i18n.Lang) response.SvcRes
	Create(ctx context.Context, userID uint, lang i18n.Lang, req *dto.CategoryReq) response.SvcRes
	Update(ctx context.Context, userID, id uint, lang i18n.Lang, req *dto.CategoryUpdateReq) response.SvcRes
	Delete(ctx context.Context, userID, id, reassignTo uint) response.SvcRes
}

type categorySvc struct {
	categoryRepo repo.CategoryRepoImpl
	// Counts and moves the expenses that use a category.
	expenseRepo repo.ExpenseRepoImpl
	// Runs fn in a database transaction; replaced in tests.
	transaction func(ctx context.Context, fn func(tx *gorm.DB) error) error
}

func NewCategorySvc() *categorySvc {
	return &categorySvc{
		categoryRepo: repo.NewCategoryRepo(),
		expenseRepo:  repo.NewExpenseRepo(),
		transaction: func(ctx context.Context, fn func(tx *gorm.DB) error) error {
			return db.DB.WithContext(ctx).Transaction(fn)
		},
	}
}

func (s *categorySvc) List(ctx context.Context, userID uint, lang i18n.Lang, req *dto.CategoryListReq) response.SvcResWithPaginate {
	page, limit := req.Page, req.Limit
	if page <= 0 {
		page = constants.DefaultPage
	}
	if limit <= 0 {
		limit = constants.DefaultLimit
	}

	categories, total, err := s.categoryRepo.List(ctx, userID, listFilter(req), page, limit)
	if err != nil {
		return response.SvcResWithPaginate{Error: fmt.Errorf("list categories: %w", err), StatusCode: http.StatusInternalServerError}
	}
	counts, err := s.expenseCounts(ctx, userID, categories)
	if err != nil {
		return response.SvcResWithPaginate{Error: err, StatusCode: http.StatusInternalServerError}
	}

	return response.SvcRespPage(toCategoryResList(categories, lang, counts), page, limit, total)
}

func (s *categorySvc) ListAll(ctx context.Context, userID uint, lang i18n.Lang) response.SvcRes {
	categories, err := s.categoryRepo.FindAllByUser(ctx, userID)
	if err != nil {
		return response.SvcRespErr(fmt.Errorf("list all categories: %w", err), http.StatusInternalServerError)
	}
	counts, err := s.expenseCounts(ctx, userID, categories)
	if err != nil {
		return response.SvcRespErr(err, http.StatusInternalServerError)
	}

	return response.SvcRespSuccess(toCategoryResList(categories, lang, counts), http.StatusOK)
}

func (s *categorySvc) GetDetail(ctx context.Context, userID, id uint, lang i18n.Lang) response.SvcRes {
	category, err := s.categoryRepo.FindOneByID(ctx, userID, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return response.SvcRespErr(i18n.RecordNotFound, http.StatusNotFound)
		}
		return response.SvcRespErr(err, http.StatusInternalServerError)
	}
	counts, err := s.expenseCounts(ctx, userID, []model.ExpenseCategory{*category})
	if err != nil {
		return response.SvcRespErr(err, http.StatusInternalServerError)
	}

	return response.SvcRespSuccess(toCategoryRes(category, lang, counts[category.ID]), http.StatusOK)
}

func (s *categorySvc) Create(ctx context.Context, userID uint, lang i18n.Lang, req *dto.CategoryReq) response.SvcRes {
	input, err := cleanCategoryReq(req)
	if err != nil {
		return response.SvcRespErr(err, http.StatusBadRequest)
	}

	if res, taken := s.checkNameFree(ctx, userID, 0, input.name); taken {
		return res
	}

	category := &model.ExpenseCategory{
		UserID:        userID,
		Name:          input.name,
		NameSearch:    buildNameSearch(input.name, input.translations),
		Icon:          req.Icon,
		Color:         req.Color,
		BaseModelGorm: model.BaseModelGorm{CreatedBy: userID, UpdatedBy: userID},
	}
	trs := toTranslationModels(input.translations)

	err = s.transaction(ctx, func(tx *gorm.DB) error {
		repoTx := s.categoryRepo.WithTx(tx)
		if _, err := repoTx.Create(ctx, category); err != nil {
			return fmt.Errorf("create category: %w", err)
		}
		if err := repoTx.ReplaceTranslations(ctx, category.ID, trs); err != nil {
			return fmt.Errorf("create category translations: %w", err)
		}
		return nil
	})
	if err != nil {
		return response.SvcRespErr(err, http.StatusInternalServerError)
	}

	category.Translations = trs
	return response.SvcRespSuccess(toCategoryRes(category, lang, 0), http.StatusCreated)
}

func (s *categorySvc) Update(ctx context.Context, userID, id uint, lang i18n.Lang, req *dto.CategoryUpdateReq) response.SvcRes {
	category, res, ok := s.findOwn(ctx, userID, id)
	if !ok {
		return res
	}

	updates := map[string]any{}
	name := category.Name
	if req.Name != nil {
		var err error
		if name, err = cleanCategoryName(*req.Name); err != nil {
			return response.SvcRespErr(err, http.StatusBadRequest)
		}
		updates["name"] = name
	}

	translations := make(map[i18n.Lang]string, len(category.Translations))
	for _, tr := range category.Translations {
		translations[i18n.Lang(tr.Lang)] = tr.Name
	}
	if req.Translations != nil {
		var err error
		if translations, err = cleanTranslations(req.Translations); err != nil {
			return response.SvcRespErr(err, http.StatusBadRequest)
		}
	}

	if req.Icon != nil {
		if err := checkCategoryIcon(*req.Icon); err != nil {
			return response.SvcRespErr(err, http.StatusBadRequest)
		}
		updates["icon"] = *req.Icon
	}
	if req.Color != nil {
		if err := checkCategoryColor(*req.Color); err != nil {
			return response.SvcRespErr(err, http.StatusBadRequest)
		}
		updates["color"] = *req.Color
	}

	if req.Name != nil || req.Translations != nil {
		updates["name_search"] = buildNameSearch(name, translations)
	}
	if len(updates) == 0 {
		return s.GetDetail(ctx, userID, id, lang)
	}

	if req.Name != nil {
		if res, taken := s.checkNameFree(ctx, userID, id, name); taken {
			return res
		}
	}

	updates["updated_by"] = userID
	err := s.transaction(ctx, func(tx *gorm.DB) error {
		repoTx := s.categoryRepo.WithTx(tx)
		if err := repoTx.Update(ctx, map[string]any{"id": id, "user_id": userID}, updates); err != nil {
			return fmt.Errorf("update category: %w", err)
		}
		if req.Translations == nil {
			return nil
		}
		if err := repoTx.ReplaceTranslations(ctx, id, toTranslationModels(translations)); err != nil {
			return fmt.Errorf("update category translations: %w", err)
		}
		return nil
	})
	if err != nil {
		return response.SvcRespErr(err, http.StatusInternalServerError)
	}

	return s.GetDetail(ctx, userID, id, lang)
}

// Delete soft-deletes one of the user's categories. When it still has expenses, reassignTo must name another
// category the user can see; the expenses are moved there in the same transaction.
func (s *categorySvc) Delete(ctx context.Context, userID, id, reassignTo uint) response.SvcRes {
	if _, res, ok := s.findOwn(ctx, userID, id); !ok {
		return res
	}

	counts, err := s.expenseRepo.CountByCategory(ctx, userID, []uint{id})
	if err != nil {
		return response.SvcRespErr(fmt.Errorf("count category expenses: %w", err), http.StatusInternalServerError)
	}
	inUse := counts[id]
	if inUse > 0 && reassignTo == 0 {
		return response.SvcRespErr(i18n.Err(i18n.CategoryInUse, i18n.Params{"count": inUse}), http.StatusConflict)
	}
	if reassignTo != 0 {
		if reassignTo == id {
			return response.SvcRespErr(i18n.InvalidReassignCategory, http.StatusBadRequest)
		}
		if _, err := s.categoryRepo.FindOneByID(ctx, userID, reassignTo); err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return response.SvcRespErr(i18n.InvalidReassignCategory, http.StatusBadRequest)
			}
			return response.SvcRespErr(err, http.StatusInternalServerError)
		}
	}

	err = s.transaction(ctx, func(tx *gorm.DB) error {
		// Also moves expenses added since the count above.
		if reassignTo != 0 {
			if err := s.expenseRepo.WithTx(tx).ReassignCategory(ctx, userID, id, reassignTo); err != nil {
				return fmt.Errorf("reassign category expenses: %w", err)
			}
		}
		if err := s.categoryRepo.WithTx(tx).SoftDelete(ctx, userID, id, userID); err != nil {
			return fmt.Errorf("delete category: %w", err)
		}
		return nil
	})
	if err != nil {
		return response.SvcRespErr(err, http.StatusInternalServerError)
	}

	return response.SvcRespSuccess(nil, http.StatusOK)
}

// expenseCounts returns how many of the user's expenses use each of the categories.
func (s *categorySvc) expenseCounts(ctx context.Context, userID uint, categories []model.ExpenseCategory) (map[uint]int64, error) {
	ids := make([]uint, len(categories))
	for i := range categories {
		ids[i] = categories[i].ID
	}
	counts, err := s.expenseRepo.CountByCategory(ctx, userID, ids)
	if err != nil {
		return nil, fmt.Errorf("count category expenses: %w", err)
	}
	return counts, nil
}

// findOwn returns the category when it exists and belongs to userID; default (admin) categories are read-only.
// ok is false with the error response otherwise.
func (s *categorySvc) findOwn(ctx context.Context, userID, id uint) (*model.ExpenseCategory, response.SvcRes, bool) {
	category, err := s.categoryRepo.FindOneByID(ctx, userID, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, response.SvcRespErr(i18n.RecordNotFound, http.StatusNotFound), false
		}
		return nil, response.SvcRespErr(err, http.StatusInternalServerError), false
	}
	if category.UserID != userID {
		return nil, response.SvcRespErr(i18n.CategoryIsDefault, http.StatusForbidden), false
	}
	return category, response.SvcRes{}, true
}

// checkNameFree reports taken (with the error response) when another category the user sees
// (own or default) has name.
func (s *categorySvc) checkNameFree(ctx context.Context, userID, exceptID uint, name string) (response.SvcRes, bool) {
	other, err := s.categoryRepo.FindByName(ctx, userID, name)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return response.SvcRes{}, false
	}
	if err != nil {
		return response.SvcRespErr(err, http.StatusInternalServerError), true
	}
	if other.ID == exceptID {
		return response.SvcRes{}, false
	}
	return response.SvcRespErr(i18n.CategoryNameTaken, http.StatusConflict), true
}

type categoryInput struct {
	name         string
	translations map[i18n.Lang]string
}

// cleanCategoryReq trims the names and checks the name, translations, icon and color.
func cleanCategoryReq(req *dto.CategoryReq) (*categoryInput, error) {
	name, err := cleanCategoryName(req.Name)
	if err != nil {
		return nil, err
	}
	translations, err := cleanTranslations(req.Translations)
	if err != nil {
		return nil, err
	}
	if err := checkCategoryIcon(req.Icon); err != nil {
		return nil, err
	}
	if err := checkCategoryColor(req.Color); err != nil {
		return nil, err
	}

	return &categoryInput{name: name, translations: translations}, nil
}

// cleanTranslations checks the languages and trims the names; blank names are dropped.
func cleanTranslations(values map[string]string) (map[i18n.Lang]string, error) {
	translations := make(map[i18n.Lang]string, len(values))
	for key, value := range values {
		lang := i18n.Lang(key)
		if !slices.Contains(i18n.Langs, lang) {
			return nil, i18n.Err(i18n.UnsupportedLanguage, i18n.Params{"langs": langList()})
		}
		if strings.TrimSpace(value) == "" {
			continue
		}
		name, err := cleanCategoryName(value)
		if err != nil {
			return nil, err
		}
		translations[lang] = name
	}
	return translations, nil
}

func checkCategoryIcon(icon string) error {
	if !slices.Contains(enums.CategoryIconValues, icon) {
		return i18n.InvalidCategoryIcon
	}
	return nil
}

func checkCategoryColor(color string) error {
	if !slices.Contains(enums.CategoryColorValues, color) {
		return i18n.InvalidCategoryColor
	}
	return nil
}

func cleanCategoryName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if n := utf8.RuneCountInString(name); n == 0 || n > categoryNameMax {
		return "", i18n.Err(i18n.InvalidCategoryName, i18n.Params{"max": categoryNameMax})
	}
	return name, nil
}

func langList() string {
	langs := make([]string, len(i18n.Langs))
	for i, lang := range i18n.Langs {
		langs[i] = string(lang)
	}
	return strings.Join(langs, ", ")
}

// buildNameSearch joins the lowercased base name and translations (in i18n.Langs order), without duplicates.
func buildNameSearch(name string, translations map[i18n.Lang]string) string {
	names := []string{strings.ToLower(name)}
	for _, lang := range i18n.Langs {
		if tr := strings.ToLower(translations[lang]); tr != "" && !slices.Contains(names, tr) {
			names = append(names, tr)
		}
	}
	return strings.Join(names, " | ")
}

var likeEscaper = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)

func listFilter(req *dto.CategoryListReq) *repo.CategoryFilter {
	filter := &repo.CategoryFilter{
		Search: likeEscaper.Replace(strings.ToLower(strings.TrimSpace(req.Q))),
		SortID: new("asc"),
	}

	order := "asc"
	key, desc := strings.CutPrefix(req.Sort, "-")
	if desc {
		order = "desc"
	}
	switch key {
	case "created_at":
		filter.SortCreatedAt = &order
	case "expense_count":
		filter.SortExpenseCount = &order
		filter.SortName = new("asc")
	default:
		filter.SortName = &order
	}
	return filter
}

func toTranslationModels(translations map[i18n.Lang]string) []model.ExpenseCategoryTr {
	trs := make([]model.ExpenseCategoryTr, 0, len(translations))
	for _, lang := range i18n.Langs {
		if name, ok := translations[lang]; ok {
			trs = append(trs, model.ExpenseCategoryTr{Lang: string(lang), Name: name})
		}
	}
	return trs
}

func toCategoryRes(category *model.ExpenseCategory, lang i18n.Lang, expenseCount int64) *dto.CategoryRes {
	res := &dto.CategoryRes{
		ID:           category.ID,
		Name:         category.Name,
		DisplayName:  category.Name,
		IsDefault:    category.UserID == 0,
		ExpenseCount: expenseCount,
		Translations: make(map[string]string, len(category.Translations)),
		Icon:         category.Icon,
		Color:        category.Color,
		CreatedAt:    category.CreatedAt,
		UpdatedAt:    category.UpdatedAt,
	}
	for _, tr := range category.Translations {
		res.Translations[tr.Lang] = tr.Name
		if tr.Lang == string(lang) && tr.Name != "" {
			res.DisplayName = tr.Name
		}
	}
	return res
}

func toCategoryResList(categories []model.ExpenseCategory, lang i18n.Lang, counts map[uint]int64) []*dto.CategoryRes {
	res := make([]*dto.CategoryRes, len(categories))
	for i := range categories {
		res[i] = toCategoryRes(&categories[i], lang, counts[categories[i].ID])
	}
	return res
}
