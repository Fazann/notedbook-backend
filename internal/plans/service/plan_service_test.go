package service

import (
	"context"
	"net/http"
	"reflect"
	"testing"
	"time"

	"github.com/Fazann/notedbook-backend/internal/plans/dto"
	"github.com/Fazann/notedbook-backend/internal/plans/repo"
	"github.com/Fazann/notedbook-backend/pkg/enums"
	"github.com/Fazann/notedbook-backend/pkg/i18n"
	model "github.com/Fazann/notedbook-backend/shared/models"
	"gorm.io/gorm"
)

// FindOne lets planSvc/planStepSvc validate an area_id against the fake plan areas (defined alongside
// fakePlanAreaRepo in plan_area_service_test.go), as the real repo's embedded baserepo.BaseRepo does.
func (r *fakePlanAreaRepo) FindOne(_ context.Context, cond ...any) (*model.PlanArea, error) {
	if len(cond) < 2 {
		return nil, gorm.ErrRecordNotFound
	}
	id, ok := cond[1].(uint)
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	if a, ok := r.areas[id]; ok {
		return a, nil
	}
	return nil, gorm.ErrRecordNotFound
}

type stepCountAdjustment struct {
	planID               uint
	stepDelta, doneDelta int
}

// fakePlanRepo keeps plans in memory; methods the service does not call panic via the nil embed.
type fakePlanRepo struct {
	repo.PlanRepoImpl
	plans       map[uint]*model.Plan
	nextID      uint
	created     *model.Plan
	updates     map[string]any
	updateCond  map[string]any
	adjustments []stepCountAdjustment
	deleted     []uint
	filter      *repo.PlanFilter
	listUser    uint
}

func newFakePlanRepo() *fakePlanRepo {
	return &fakePlanRepo{
		nextID: 10,
		plans: map[uint]*model.Plan{
			1: {BaseModelID: model.BaseModelID{ID: 1}, UserID: 1, Title: "Save for laptop", AreaID: 1,
				Priority: enums.PlanPriorityEnum(enums.PlanPriorityHigh), Status: enums.PlanStatusEnum(enums.PlanStatusNotStart)},
			2: {BaseModelID: model.BaseModelID{ID: 2}, UserID: 2, Title: "Other user's plan", AreaID: 1,
				Status: enums.PlanStatusEnum(enums.PlanStatusNotStart)},
		},
	}
}

func (r *fakePlanRepo) WithTx(_ *gorm.DB) repo.PlanRepoImpl { return r }

func (r *fakePlanRepo) FindOneByID(_ context.Context, userID, id uint) (*model.Plan, error) {
	if p, ok := r.plans[id]; ok && p.UserID == userID {
		return p, nil
	}
	return nil, gorm.ErrRecordNotFound
}

func (r *fakePlanRepo) List(_ context.Context, userID uint, filter *repo.PlanFilter, _, _ int) ([]model.Plan, int64, error) {
	r.listUser, r.filter = userID, filter
	out := []model.Plan{}
	for _, p := range r.plans {
		if p.UserID == userID {
			out = append(out, *p)
		}
	}
	return out, int64(len(out)), nil
}

func (r *fakePlanRepo) Create(_ context.Context, p *model.Plan) (*model.Plan, error) {
	p.ID = r.nextID
	r.nextID++
	r.created = p
	r.plans[p.ID] = p
	return p, nil
}

func (r *fakePlanRepo) Update(_ context.Context, condition any, updates map[string]any) error {
	r.updates = updates
	r.updateCond, _ = condition.(map[string]any)
	id, _ := r.updateCond["id"].(uint)
	if p, ok := r.plans[id]; ok {
		applyPlanUpdates(p, updates)
	}
	return nil
}

func (r *fakePlanRepo) AdjustStepCounts(_ context.Context, planID uint, stepDelta, doneDelta int) error {
	r.adjustments = append(r.adjustments, stepCountAdjustment{planID, stepDelta, doneDelta})
	if p, ok := r.plans[planID]; ok {
		p.StepCount += stepDelta
		p.DoneStepCount += doneDelta
	}
	return nil
}

func (r *fakePlanRepo) SoftDelete(_ context.Context, userID, id uint) error {
	p, ok := r.plans[id]
	if !ok || p.UserID != userID {
		return gorm.ErrRecordNotFound
	}
	r.deleted = append(r.deleted, id)
	delete(r.plans, id)
	return nil
}

func applyPlanUpdates(p *model.Plan, updates map[string]any) {
	for k, v := range updates {
		switch k {
		case "title":
			p.Title = v.(string)
		case "description":
			p.Description = v.(string)
		case "area_id":
			p.AreaID = v.(uint)
		case "priority":
			p.Priority = v.(enums.PlanPriorityEnum)
		case "target_date":
			if v == nil {
				p.TargetDate = nil
			} else {
				p.TargetDate = v.(*time.Time)
			}
		case "status":
			p.Status = v.(enums.PlanStatusEnum)
		case "done_at":
			if v == nil {
				p.DoneAt = nil
			} else {
				t := v.(time.Time)
				p.DoneAt = &t
			}
		}
	}
}

func newTestPlanSvc(planRepo *fakePlanRepo, areaRepo *fakePlanAreaRepo, stepRepo *fakePlanStepRepo) *planSvc {
	return &planSvc{planRepo: planRepo, planAreaRepo: areaRepo, planStepRepo: stepRepo}
}

func validPlanReq() dto.PlanReq {
	return dto.PlanReq{Title: " Save for laptop ", Description: " Need it for class ", AreaID: 1,
		Priority: enums.PlanPriorityEnum(enums.PlanPriorityHigh)}
}

func TestCreatePlan(t *testing.T) {
	tests := []struct {
		name       string
		edit       func(*dto.PlanReq)
		wantStatus int
		wantErr    error
	}{
		{name: "created", wantStatus: http.StatusCreated},
		{name: "title too short", edit: func(r *dto.PlanReq) { r.Title = "A" },
			wantStatus: http.StatusBadRequest, wantErr: i18n.FieldMinLength},
		{name: "unknown area", edit: func(r *dto.PlanReq) { r.AreaID = 99 },
			wantStatus: http.StatusBadRequest, wantErr: i18n.InvalidPlanArea},
		{name: "invalid target date", edit: func(r *dto.PlanReq) { d := "not-a-date"; r.TargetDate = &d },
			wantStatus: http.StatusBadRequest, wantErr: i18n.FieldInvalid},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			planRepo, areaRepo := newFakePlanRepo(), newFakePlanAreaRepo()
			req := validPlanReq()
			if tt.edit != nil {
				tt.edit(&req)
			}

			res := newTestPlanSvc(planRepo, areaRepo, nil).Create(context.Background(), 1, i18n.EN, &req)

			if res.StatusCode != tt.wantStatus {
				t.Fatalf("status = %d, want %d (err %v)", res.StatusCode, tt.wantStatus, res.Error)
			}
			if tt.wantErr != nil {
				if key, _, _ := i18n.FromError(res.Error); key != tt.wantErr {
					t.Fatalf("err = %v, want %v", res.Error, tt.wantErr)
				}
				if planRepo.created != nil {
					t.Fatal("plan created on error")
				}
				return
			}
			if planRepo.created.UserID != 1 || planRepo.created.Title != "Save for laptop" {
				t.Errorf("created = %+v", planRepo.created)
			}
			if planRepo.created.Status != enums.PlanStatusEnum(enums.PlanStatusNotStart) {
				t.Errorf("status = %v, want NOT_START", planRepo.created.Status)
			}
			if planRepo.created.StepCount != 0 || planRepo.created.DoneStepCount != 0 {
				t.Errorf("counts = %d/%d, want 0/0", planRepo.created.StepCount, planRepo.created.DoneStepCount)
			}
			got := res.Data.(*dto.PlanRes)
			if got.Area.ID != 1 || got.Area.Title != "Finance" {
				t.Errorf("area = %+v", got.Area)
			}
		})
	}
}

func TestUpdatePlanStatus(t *testing.T) {
	tests := []struct {
		name       string
		req        dto.PlanStatusReq
		wantStatus enums.PlanStatusEnum
		wantDoneAt bool
	}{
		{name: "to done sets done_at", req: dto.PlanStatusReq{Status: enums.PlanStatusEnum(enums.PlanStatusDone)},
			wantStatus: enums.PlanStatusEnum(enums.PlanStatusDone), wantDoneAt: true},
		{name: "same status is a no-op", req: dto.PlanStatusReq{Status: enums.PlanStatusEnum(enums.PlanStatusNotStart)},
			wantStatus: enums.PlanStatusEnum(enums.PlanStatusNotStart), wantDoneAt: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			planRepo, areaRepo := newFakePlanRepo(), newFakePlanAreaRepo()

			res := newTestPlanSvc(planRepo, areaRepo, newFakePlanStepRepo()).
				UpdateStatus(context.Background(), 1, 1, i18n.EN, &tt.req)

			if res.Error != nil || res.StatusCode != http.StatusOK {
				t.Fatalf("status = %d, err = %v", res.StatusCode, res.Error)
			}
			p := planRepo.plans[1]
			if p.Status != tt.wantStatus {
				t.Errorf("status = %v, want %v", p.Status, tt.wantStatus)
			}
			if (p.DoneAt != nil) != tt.wantDoneAt {
				t.Errorf("done_at = %v, want set = %v", p.DoneAt, tt.wantDoneAt)
			}
		})
	}

	// Leaving DONE clears done_at.
	planRepo, areaRepo := newFakePlanRepo(), newFakePlanAreaRepo()
	now := time.Now()
	planRepo.plans[1].Status = enums.PlanStatusEnum(enums.PlanStatusDone)
	planRepo.plans[1].DoneAt = &now

	res := newTestPlanSvc(planRepo, areaRepo, newFakePlanStepRepo()).
		UpdateStatus(context.Background(), 1, 1, i18n.EN, &dto.PlanStatusReq{Status: enums.PlanStatusEnum(enums.PlanStatusInProgress)})
	if res.Error != nil {
		t.Fatalf("err = %v", res.Error)
	}
	if planRepo.plans[1].DoneAt != nil {
		t.Errorf("done_at = %v, want nil", planRepo.plans[1].DoneAt)
	}
}

func TestDeletePlan(t *testing.T) {
	tests := []struct {
		name       string
		userID, id uint
		wantStatus int
	}{
		{name: "own plan", userID: 1, id: 1, wantStatus: http.StatusOK},
		{name: "another user's plan", userID: 1, id: 2, wantStatus: http.StatusNotFound},
		{name: "missing", userID: 1, id: 99, wantStatus: http.StatusNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			planRepo, areaRepo := newFakePlanRepo(), newFakePlanAreaRepo()

			res := newTestPlanSvc(planRepo, areaRepo, nil).Delete(context.Background(), tt.userID, tt.id)

			if res.StatusCode != tt.wantStatus {
				t.Fatalf("status = %d, want %d", res.StatusCode, tt.wantStatus)
			}
			if tt.wantStatus == http.StatusOK && !reflect.DeepEqual(planRepo.deleted, []uint{tt.id}) {
				t.Errorf("deleted = %v", planRepo.deleted)
			}
		})
	}
}
