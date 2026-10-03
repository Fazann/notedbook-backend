package service

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/Fazann/notedbook-backend/internal/plans/dto"
	"github.com/Fazann/notedbook-backend/internal/plans/repo"
	"github.com/Fazann/notedbook-backend/pkg/i18n"
	model "github.com/Fazann/notedbook-backend/shared/models"
	"gorm.io/gorm"
)

// fakePlanAreaRepo keeps plan areas in memory; methods the service does not call panic via the nil embed.
type fakePlanAreaRepo struct {
	repo.PlanAreaRepoImpl
	areas      map[uint]*model.PlanArea
	nextID     uint
	trs        []model.PlanAreaTr
	createdAll []*model.PlanArea
	findAllN   int
}

func newFakePlanAreaRepo() *fakePlanAreaRepo {
	return &fakePlanAreaRepo{
		nextID: 10,
		areas: map[uint]*model.PlanArea{
			1: {BaseModelID: model.BaseModelID{ID: 1}, Title: "Finance", Icon: "wallet",
				Translations: []model.PlanAreaTr{{Lang: "km", Title: "ហិរញ្ញវត្ថុ"}}},
		},
	}
}

func (r *fakePlanAreaRepo) WithTx(_ *gorm.DB) repo.PlanAreaRepoImpl { return r }

func (r *fakePlanAreaRepo) FindAll(_ context.Context) ([]model.PlanArea, error) {
	r.findAllN++
	areas := make([]model.PlanArea, 0, len(r.areas))
	for _, a := range r.areas {
		areas = append(areas, *a)
	}
	return areas, nil
}

func (r *fakePlanAreaRepo) FindByTitle(_ context.Context, title string) (*model.PlanArea, error) {
	for _, a := range r.areas {
		if strings.EqualFold(a.Title, title) {
			return a, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (r *fakePlanAreaRepo) Create(_ context.Context, a *model.PlanArea) (*model.PlanArea, error) {
	a.ID = r.nextID
	r.nextID++
	r.createdAll = append(r.createdAll, a)
	r.areas[a.ID] = a
	return a, nil
}

func (r *fakePlanAreaRepo) ReplaceTranslations(_ context.Context, _ uint, trs []model.PlanAreaTr) error {
	r.trs = trs
	return nil
}

// fakePlanAreaCache is an in-memory cache; err, when set, is returned by every call.
type fakePlanAreaCache struct {
	values  map[string]string
	ttl     time.Duration
	deleted int
	err     error
}

func newFakePlanAreaCache() *fakePlanAreaCache {
	return &fakePlanAreaCache{values: map[string]string{}}
}

func (c *fakePlanAreaCache) Get(_ context.Context, key string) (string, bool, error) {
	if c.err != nil {
		return "", false, c.err
	}
	v, ok := c.values[key]
	return v, ok, nil
}

func (c *fakePlanAreaCache) Set(_ context.Context, key, value string, ttl time.Duration) error {
	if c.err != nil {
		return c.err
	}
	c.values[key] = value
	c.ttl = ttl
	return nil
}

func (c *fakePlanAreaCache) Del(_ context.Context, keys ...string) error {
	if c.err != nil {
		return c.err
	}
	for _, k := range keys {
		delete(c.values, k)
	}
	c.deleted++
	return nil
}

func newTestPlanAreaSvc(r *fakePlanAreaRepo) *planAreaSvc {
	return newTestPlanAreaSvcWithCache(r, newFakePlanAreaCache())
}

func newTestPlanAreaSvcWithCache(r *fakePlanAreaRepo, c *fakePlanAreaCache) *planAreaSvc {
	return &planAreaSvc{
		planAreaRepo: r,
		cache:        c,
		transaction:  func(_ context.Context, fn func(tx *gorm.DB) error) error { return fn(nil) },
	}
}

func TestListPlanAreas(t *testing.T) {
	r := newFakePlanAreaRepo()

	res := newTestPlanAreaSvc(r).List(context.Background(), i18n.KM)

	if res.Error != nil || res.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, err = %v", res.StatusCode, res.Error)
	}
	items := res.Data.([]*dto.PlanAreaRes)
	if len(items) != 1 || items[0].Title != "Finance" || items[0].DisplayTitle != "ហិរញ្ញវត្ថុ" || items[0].Translations["km"] != "ហិរញ្ញវត្ថុ" {
		t.Errorf("items = %+v", items)
	}
}

func TestSeedDefaultPlanAreas(t *testing.T) {
	r := newFakePlanAreaRepo() // already has "Finance"
	svc := newTestPlanAreaSvc(r)

	created, err := svc.seedDefaults(context.Background())
	if err != nil || created != len(defaultPlanAreas)-1 {
		t.Fatalf("created = %d, err = %v, want %d", created, err, len(defaultPlanAreas)-1)
	}
	for _, a := range r.createdAll {
		if a.Title == "Finance" {
			t.Error("existing default area created again")
		}
	}
	personal := r.createdAll[0]
	if personal.Title != "Personal" || personal.Icon != "user" {
		t.Errorf("first seeded = %+v", personal)
	}

	again, err := svc.seedDefaults(context.Background())
	if err != nil || again != 0 {
		t.Errorf("second run created = %d, err = %v, want 0", again, err)
	}
}

func TestListPlanAreasCache(t *testing.T) {
	tests := []struct {
		name        string
		cacheErr    error
		calls       int
		wantFindAll int
		wantCached  bool
	}{
		{name: "second list is served from the cache", calls: 2, wantFindAll: 1, wantCached: true},
		{name: "redis down falls back to the database", cacheErr: errors.New("redis down"), calls: 2, wantFindAll: 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := newFakePlanAreaRepo()
			c := newFakePlanAreaCache()
			c.err = tt.cacheErr
			svc := newTestPlanAreaSvcWithCache(r, c)

			for i := 0; i < tt.calls; i++ {
				res := svc.List(context.Background(), i18n.KM)
				if res.Error != nil {
					t.Fatalf("call %d: err = %v", i, res.Error)
				}
				items := res.Data.([]*dto.PlanAreaRes)
				if len(items) != 1 || items[0].DisplayTitle != "ហិរញ្ញវត្ថុ" {
					t.Fatalf("call %d: items = %+v", i, items)
				}
			}
			if r.findAllN != tt.wantFindAll {
				t.Errorf("FindAll called %d times, want %d", r.findAllN, tt.wantFindAll)
			}
			if _, ok := c.values[planAreaCacheKey]; ok != tt.wantCached {
				t.Errorf("cached = %v, want %v", ok, tt.wantCached)
			}
			if tt.wantCached && c.ttl != planAreaCacheTTL {
				t.Errorf("ttl = %v, want %v", c.ttl, planAreaCacheTTL)
			}
		})
	}
}

func TestListPlanAreasBadCacheValue(t *testing.T) {
	r := newFakePlanAreaRepo()
	c := newFakePlanAreaCache()
	c.values[planAreaCacheKey] = "not json"

	res := newTestPlanAreaSvcWithCache(r, c).List(context.Background(), i18n.EN)
	if res.Error != nil || len(res.Data.([]*dto.PlanAreaRes)) != 1 || r.findAllN != 1 {
		t.Errorf("err = %v, data = %+v, FindAll calls = %d", res.Error, res.Data, r.findAllN)
	}
}

func TestResetPlanAreaCache(t *testing.T) {
	tests := []struct {
		name       string
		cacheErr   error
		wantStatus int
	}{
		{name: "deletes the cached list", wantStatus: http.StatusOK},
		{name: "redis down", cacheErr: errors.New("redis down"), wantStatus: http.StatusInternalServerError},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := newFakePlanAreaCache()
			c.values[planAreaCacheKey] = "[]"
			c.err = tt.cacheErr

			res := newTestPlanAreaSvcWithCache(newFakePlanAreaRepo(), c).ResetCache(context.Background())
			if res.StatusCode != tt.wantStatus {
				t.Fatalf("status = %d, want %d (err %v)", res.StatusCode, tt.wantStatus, res.Error)
			}
			if _, ok := c.values[planAreaCacheKey]; ok == (tt.cacheErr == nil) {
				t.Errorf("cache key present = %v after reset", ok)
			}
		})
	}
}

func TestSeedDefaultPlanAreasInvalidatesCache(t *testing.T) {
	r := newFakePlanAreaRepo()
	c := newFakePlanAreaCache()
	svc := newTestPlanAreaSvcWithCache(r, c)
	svc.List(context.Background(), i18n.EN) // fills the cache

	if _, err := svc.seedDefaults(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, ok := c.values[planAreaCacheKey]; ok {
		t.Error("cache not cleared after seeding new areas")
	}

	svc.List(context.Background(), i18n.EN)
	if _, err := svc.seedDefaults(context.Background()); err != nil {
		t.Fatal(err)
	}
	if c.deleted != 1 {
		t.Errorf("cache deleted %d times, want 1 (nothing new on the second seed)", c.deleted)
	}
}
