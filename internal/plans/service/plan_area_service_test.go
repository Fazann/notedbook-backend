package service

import (
	"context"
	"net/http"
	"strings"
	"testing"

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

func newTestPlanAreaSvc(r *fakePlanAreaRepo) *planAreaSvc {
	return &planAreaSvc{
		planAreaRepo: r,
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
