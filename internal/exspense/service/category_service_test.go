package service

import (
	"context"
	"errors"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/Fazann/notedbook-backend/internal/exspense/dto"
	"github.com/Fazann/notedbook-backend/internal/exspense/repo"
	"github.com/Fazann/notedbook-backend/pkg/i18n"
	model "github.com/Fazann/notedbook-backend/shared/models"
	"gorm.io/gorm"
)

// fakeCategoryRepo keeps categories in memory; methods the service does not call panic via the nil embed.
type fakeCategoryRepo struct {
	repo.CategoryRepoImpl
	categories map[uint]*model.ExpenseCategory
	nextID     uint
	created    *model.ExpenseCategory
	updates    map[string]any
	trs        []model.ExpenseCategoryTr
	deleted    []uint
	filter     *repo.CategoryFilter
	listUser   uint
	createdAll []*model.ExpenseCategory
}

func newFakeCategoryRepo() *fakeCategoryRepo {
	return &fakeCategoryRepo{
		nextID: 10,
		categories: map[uint]*model.ExpenseCategory{
			1: {BaseModelID: model.BaseModelID{ID: 1}, UserID: 1, Name: "Gym", Icon: "dumbbell", Color: "green",
				Translations: []model.ExpenseCategoryTr{{Lang: "km", Name: "ហាត់ប្រាណ"}}},
			2: {BaseModelID: model.BaseModelID{ID: 2}, UserID: 1, Name: "Books", Icon: "book-open", Color: "blue"},
			3: {BaseModelID: model.BaseModelID{ID: 3}, UserID: 2, Name: "Travel", Icon: "plane", Color: "teal"},
			// Default (admin) category: user_id NULL.
			4: {BaseModelID: model.BaseModelID{ID: 4}, Name: "Food", Icon: "utensils", Color: "chart-1",
				Translations: []model.ExpenseCategoryTr{{Lang: "km", Name: "អាហារ"}}},
		},
	}
}

func (r *fakeCategoryRepo) WithTx(_ *gorm.DB) repo.CategoryRepoImpl { return r }

func (r *fakeCategoryRepo) FindOneByID(_ context.Context, userID, id uint) (*model.ExpenseCategory, error) {
	if c, ok := r.categories[id]; ok && (c.UserID == userID || c.UserID == 0) {
		return c, nil
	}
	return nil, gorm.ErrRecordNotFound
}

func (r *fakeCategoryRepo) FindByName(_ context.Context, userID uint, name string) (*model.ExpenseCategory, error) {
	for _, c := range r.categories {
		if (c.UserID == userID || c.UserID == 0) && strings.EqualFold(c.Name, name) {
			return c, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (r *fakeCategoryRepo) FindDefaultByName(ctx context.Context, name string) (*model.ExpenseCategory, error) {
	return r.FindByName(ctx, 0, name)
}

func (r *fakeCategoryRepo) List(_ context.Context, userID uint, filter *repo.CategoryFilter, _, _ int) ([]model.ExpenseCategory, int64, error) {
	r.listUser, r.filter = userID, filter
	return []model.ExpenseCategory{*r.categories[1], *r.categories[2], *r.categories[4]}, 12, nil
}

func (r *fakeCategoryRepo) Create(_ context.Context, c *model.ExpenseCategory) (*model.ExpenseCategory, error) {
	c.ID = r.nextID
	r.nextID++
	r.created = c
	r.createdAll = append(r.createdAll, c)
	r.categories[c.ID] = c
	return c, nil
}

func (r *fakeCategoryRepo) Update(_ context.Context, _ any, updates map[string]any) error {
	r.updates = updates
	return nil
}

func (r *fakeCategoryRepo) ReplaceTranslations(_ context.Context, _ uint, trs []model.ExpenseCategoryTr) error {
	r.trs = trs
	return nil
}

func (r *fakeCategoryRepo) SoftDelete(_ context.Context, _, id, _ uint) error {
	r.deleted = append(r.deleted, id)
	return nil
}

func newTestCategorySvc(r *fakeCategoryRepo) *categorySvc {
	return &categorySvc{
		categoryRepo: r,
		transaction:  func(_ context.Context, fn func(tx *gorm.DB) error) error { return fn(nil) },
	}
}

func validReq() dto.CategoryReq {
	return dto.CategoryReq{Name: " Pet food ", Translations: map[string]string{"km": " ចំណីសត្វ ", "ms": " "}, Icon: "dog", Color: "amber"}
}

func TestCreateCategory(t *testing.T) {
	tests := []struct {
		name       string
		edit       func(*dto.CategoryReq)
		wantStatus int
		wantErr    error
	}{
		{name: "created", wantStatus: http.StatusCreated},
		{name: "blank name", edit: func(r *dto.CategoryReq) { r.Name = "   " },
			wantStatus: http.StatusBadRequest, wantErr: i18n.InvalidCategoryName},
		{name: "name too long", edit: func(r *dto.CategoryReq) { r.Name = strings.Repeat("ក", 51) },
			wantStatus: http.StatusBadRequest, wantErr: i18n.InvalidCategoryName},
		{name: "translation too long", edit: func(r *dto.CategoryReq) { r.Translations["km"] = strings.Repeat("a", 51) },
			wantStatus: http.StatusBadRequest, wantErr: i18n.InvalidCategoryName},
		{name: "unsupported language", edit: func(r *dto.CategoryReq) { r.Translations["fr"] = "Nourriture" },
			wantStatus: http.StatusBadRequest, wantErr: i18n.UnsupportedLanguage},
		{name: "unknown icon", edit: func(r *dto.CategoryReq) { r.Icon = "rocket" },
			wantStatus: http.StatusBadRequest, wantErr: i18n.InvalidCategoryIcon},
		{name: "unknown color", edit: func(r *dto.CategoryReq) { r.Color = "#ff0000" },
			wantStatus: http.StatusBadRequest, wantErr: i18n.InvalidCategoryColor},
		{name: "name taken, any case", edit: func(r *dto.CategoryReq) { r.Name = "gYm" },
			wantStatus: http.StatusConflict, wantErr: i18n.CategoryNameTaken},
		{name: "other user's name is free", edit: func(r *dto.CategoryReq) { r.Name = "Travel" },
			wantStatus: http.StatusCreated},
		{name: "default category's name is taken", edit: func(r *dto.CategoryReq) { r.Name = "food" },
			wantStatus: http.StatusConflict, wantErr: i18n.CategoryNameTaken},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := newFakeCategoryRepo()
			req := validReq()
			if tt.edit != nil {
				tt.edit(&req)
			}

			res := newTestCategorySvc(r).Create(context.Background(), 1, i18n.KM, &req)

			if res.StatusCode != tt.wantStatus {
				t.Fatalf("status = %d, want %d (err %v)", res.StatusCode, tt.wantStatus, res.Error)
			}
			if tt.wantErr != nil {
				if key, _, _ := i18n.FromError(res.Error); key != tt.wantErr {
					t.Fatalf("err = %v, want %v", res.Error, tt.wantErr)
				}
				if r.created != nil {
					t.Fatal("category created on error")
				}
				return
			}
			if res.Error != nil {
				t.Fatalf("unexpected err: %v", res.Error)
			}
			if r.created.UserID != 1 || r.created.CreatedBy != 1 {
				t.Errorf("owner = %d / created_by = %d, want 1", r.created.UserID, r.created.CreatedBy)
			}
		})
	}
}

func TestCreateCategoryResult(t *testing.T) {
	r := newFakeCategoryRepo()
	req := validReq()

	res := newTestCategorySvc(r).Create(context.Background(), 1, i18n.KM, &req)

	if r.created.Name != "Pet food" || r.created.NameSearch != "pet food | ចំណីសត្វ" {
		t.Errorf("stored name = %q, name_search = %q", r.created.Name, r.created.NameSearch)
	}
	wantTrs := []model.ExpenseCategoryTr{{Lang: "km", Name: "ចំណីសត្វ"}}
	if !reflect.DeepEqual(r.trs, wantTrs) {
		t.Errorf("translations = %+v, want %+v (blank ms dropped)", r.trs, wantTrs)
	}
	got := res.Data.(*dto.CategoryRes)
	if got.ID != 10 || got.DisplayName != "ចំណីសត្វ" || got.Translations["km"] != "ចំណីសត្វ" {
		t.Errorf("response = %+v", got)
	}
}

func TestUpdateCategory(t *testing.T) {
	tests := []struct {
		name        string
		userID, id  uint
		req         dto.CategoryUpdateReq
		wantStatus  int
		wantErr     error
		wantUpdates map[string]any
		// nil: translations not replaced.
		wantTrs []model.ExpenseCategoryTr
	}{
		{name: "rename keeps translations", userID: 1, id: 1,
			req:        dto.CategoryUpdateReq{Name: new(" Fitness ")},
			wantStatus: http.StatusOK,
			wantUpdates: map[string]any{"name": "Fitness", "name_search": "fitness | ហាត់ប្រាណ",
				"updated_by": uint(1)}},
		{name: "color only", userID: 1, id: 1,
			req:         dto.CategoryUpdateReq{Color: new("red")},
			wantStatus:  http.StatusOK,
			wantUpdates: map[string]any{"color": "red", "updated_by": uint(1)}},
		{name: "replace translations", userID: 1, id: 1,
			req:         dto.CategoryUpdateReq{Translations: map[string]string{"ms": " Gim ", "km": " "}},
			wantStatus:  http.StatusOK,
			wantUpdates: map[string]any{"name_search": "gym | gim", "updated_by": uint(1)},
			wantTrs:     []model.ExpenseCategoryTr{{Lang: "ms", Name: "Gim"}}},
		{name: "empty translations remove all", userID: 1, id: 1,
			req:         dto.CategoryUpdateReq{Translations: map[string]string{}},
			wantStatus:  http.StatusOK,
			wantUpdates: map[string]any{"name_search": "gym", "updated_by": uint(1)},
			wantTrs:     []model.ExpenseCategoryTr{}},
		{name: "nothing sent", userID: 1, id: 1, wantStatus: http.StatusOK},
		{name: "keeping its own name is not taken", userID: 1, id: 1,
			req:        dto.CategoryUpdateReq{Name: new("GYM")},
			wantStatus: http.StatusOK,
			wantUpdates: map[string]any{"name": "GYM", "name_search": "gym | ហាត់ប្រាណ",
				"updated_by": uint(1)}},
		{name: "name of another category", userID: 1, id: 1,
			req:        dto.CategoryUpdateReq{Name: new("books")},
			wantStatus: http.StatusConflict, wantErr: i18n.CategoryNameTaken},
		{name: "blank name", userID: 1, id: 1,
			req:        dto.CategoryUpdateReq{Name: new(" ")},
			wantStatus: http.StatusBadRequest, wantErr: i18n.InvalidCategoryName},
		{name: "another user's category", userID: 1, id: 3,
			req:        dto.CategoryUpdateReq{Name: new("Trips")},
			wantStatus: http.StatusNotFound, wantErr: i18n.RecordNotFound},
		{name: "default category is read-only", userID: 1, id: 4,
			req:        dto.CategoryUpdateReq{Color: new("red")},
			wantStatus: http.StatusForbidden, wantErr: i18n.CategoryIsDefault},
		{name: "invalid icon", userID: 1, id: 1,
			req:        dto.CategoryUpdateReq{Icon: new("rocket")},
			wantStatus: http.StatusBadRequest, wantErr: i18n.InvalidCategoryIcon},
		{name: "invalid color", userID: 1, id: 1,
			req:        dto.CategoryUpdateReq{Name: new("Fitness"), Color: new("gold")},
			wantStatus: http.StatusBadRequest, wantErr: i18n.InvalidCategoryColor},
		{name: "unsupported language", userID: 1, id: 1,
			req:        dto.CategoryUpdateReq{Translations: map[string]string{"fr": "Gym"}},
			wantStatus: http.StatusBadRequest, wantErr: i18n.UnsupportedLanguage},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := newFakeCategoryRepo()

			res := newTestCategorySvc(r).Update(context.Background(), tt.userID, tt.id, i18n.EN, &tt.req)

			if res.StatusCode != tt.wantStatus {
				t.Fatalf("status = %d, want %d (err %v)", res.StatusCode, tt.wantStatus, res.Error)
			}
			if tt.wantErr != nil {
				if key, _, _ := i18n.FromError(res.Error); key != tt.wantErr {
					t.Fatalf("err = %v, want %v", res.Error, tt.wantErr)
				}
				if r.updates != nil || r.trs != nil {
					t.Fatal("category updated on error")
				}
				return
			}
			if !reflect.DeepEqual(r.updates, tt.wantUpdates) {
				t.Errorf("updates = %v, want %v", r.updates, tt.wantUpdates)
			}
			if !reflect.DeepEqual(r.trs, tt.wantTrs) {
				t.Errorf("translations = %+v, want %+v", r.trs, tt.wantTrs)
			}
		})
	}
}

func TestDeleteCategory(t *testing.T) {
	tests := []struct {
		name        string
		userID, id  uint
		wantStatus  int
		wantDeleted []uint
	}{
		{name: "own category", userID: 1, id: 2, wantStatus: http.StatusOK, wantDeleted: []uint{2}},
		{name: "another user's category", userID: 1, id: 3, wantStatus: http.StatusNotFound},
		{name: "default category", userID: 1, id: 4, wantStatus: http.StatusForbidden},
		{name: "missing", userID: 1, id: 99, wantStatus: http.StatusNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := newFakeCategoryRepo()

			res := newTestCategorySvc(r).Delete(context.Background(), tt.userID, tt.id)

			if res.StatusCode != tt.wantStatus {
				t.Fatalf("status = %d, want %d", res.StatusCode, tt.wantStatus)
			}
			if !reflect.DeepEqual(r.deleted, tt.wantDeleted) {
				t.Errorf("deleted = %v, want %v", r.deleted, tt.wantDeleted)
			}
		})
	}
}

func TestListCategories(t *testing.T) {
	r := newFakeCategoryRepo()

	res := newTestCategorySvc(r).List(context.Background(), 1, i18n.KM, &dto.CategoryListReq{Page: 2, Limit: 5, Q: " 50%_OFF ", Sort: "-created_at"})

	if res.Error != nil || res.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, err = %v", res.StatusCode, res.Error)
	}
	if res.Total != 12 || res.TotalPage != 3 || !res.IsNext || !res.IsPrev {
		t.Errorf("paging = total %d, pages %d, next %v, prev %v", res.Total, res.TotalPage, res.IsNext, res.IsPrev)
	}
	if r.listUser != 1 || r.filter.Search != `50\%\_off` {
		t.Errorf("list user = %d, search = %q", r.listUser, r.filter.Search)
	}
	if r.filter.SortCreatedAt == nil || *r.filter.SortCreatedAt != "desc" || r.filter.SortName != nil {
		t.Errorf("sort = created_at %v, name %v", r.filter.SortCreatedAt, r.filter.SortName)
	}
	items := res.Data.([]*dto.CategoryRes)
	if items[0].DisplayName != "ហាត់ប្រាណ" || items[1].DisplayName != "Books" || items[2].DisplayName != "អាហារ" {
		t.Errorf("display names = %q, %q, %q", items[0].DisplayName, items[1].DisplayName, items[2].DisplayName)
	}
	if items[0].IsDefault || !items[2].IsDefault {
		t.Errorf("is_default = %v, %v, want false, true", items[0].IsDefault, items[2].IsDefault)
	}
}

func TestListFilterDefaults(t *testing.T) {
	filter := listFilter(&dto.CategoryListReq{})
	if filter.SortName == nil || *filter.SortName != "asc" || filter.SortCreatedAt != nil || filter.Search != "" {
		t.Errorf("filter = %+v", filter)
	}
}

func TestBuildNameSearch(t *testing.T) {
	tests := []struct {
		name         string
		base         string
		translations map[i18n.Lang]string
		want         string
	}{
		{name: "base only", base: "Gym", want: "gym"},
		{name: "languages in fixed order", base: "Food",
			translations: map[i18n.Lang]string{i18n.MS: "Makanan", i18n.KM: "អាហារ"}, want: "food | អាហារ | makanan"},
		{name: "duplicates dropped", base: "Netflix",
			translations: map[i18n.Lang]string{i18n.EN: "NETFLIX", i18n.KM: "Netflix"}, want: "netflix"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := buildNameSearch(tt.base, tt.translations); got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestUnsupportedLanguageListsLangs(t *testing.T) {
	_, err := cleanCategoryReq(&dto.CategoryReq{Name: "Gym", Translations: map[string]string{"KM": "x"}, Icon: "dumbbell", Color: "green"})
	var e *i18n.Error
	if !errors.As(err, &e) || e.Key != i18n.UnsupportedLanguage || e.Params["langs"] != "en, km, ms" {
		t.Fatalf("err = %v", err)
	}
}

func TestSeedDefaultCategories(t *testing.T) {
	r := newFakeCategoryRepo() // already has "Food"
	svc := newTestCategorySvc(r)

	created, err := svc.seedDefaults(context.Background())
	if err != nil || created != len(defaultCategories)-1 {
		t.Fatalf("created = %d, err = %v, want %d", created, err, len(defaultCategories)-1)
	}
	for _, c := range r.createdAll {
		if c.UserID != 0 || c.CreatedBy != 0 {
			t.Errorf("%s: user_id = %d, created_by = %d, want default (0)", c.Name, c.UserID, c.CreatedBy)
		}
		if c.Name == "Food" {
			t.Error("existing default created again")
		}
	}
	coffee := r.createdAll[0]
	if coffee.Name != "Coffee" || coffee.NameSearch != "coffee | កាហ្វេ | kopi" || coffee.Icon != "coffee" {
		t.Errorf("first seeded = %+v", coffee)
	}

	again, err := svc.seedDefaults(context.Background())
	if err != nil || again != 0 {
		t.Errorf("second run created = %d, err = %v, want 0", again, err)
	}
}

func TestDefaultCategoriesAreValid(t *testing.T) {
	for _, item := range defaultCategories {
		translations := map[string]string{}
		for lang, name := range item.translations {
			translations[string(lang)] = name
		}
		req := dto.CategoryReq{Name: item.name, Translations: translations, Icon: item.icon, Color: item.color}
		if _, err := cleanCategoryReq(&req); err != nil {
			t.Errorf("%s: %v", item.name, err)
		}
	}
}
