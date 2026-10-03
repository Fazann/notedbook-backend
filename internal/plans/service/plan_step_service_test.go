package service

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/Fazann/notedbook-backend/internal/plans/dto"
	"github.com/Fazann/notedbook-backend/internal/plans/repo"
	"github.com/Fazann/notedbook-backend/pkg/enums"
	"github.com/Fazann/notedbook-backend/pkg/i18n"
	model "github.com/Fazann/notedbook-backend/shared/models"
	"gorm.io/gorm"
)

// fakePlanStepRepo keeps plan steps in memory; methods the service does not call panic via the nil embed.
type fakePlanStepRepo struct {
	repo.PlanStepRepoImpl
	steps      map[uint]*model.PlanStep
	nextID     uint
	created    *model.PlanStep
	updates    map[string]any
	deletedIDs []uint
}

func newFakePlanStepRepo() *fakePlanStepRepo {
	return &fakePlanStepRepo{
		nextID: 100,
		steps: map[uint]*model.PlanStep{
			1: {BaseModelID: model.BaseModelID{ID: 1}, PlanID: 1, Title: "Open account", Done: false, Position: 1000},
			2: {BaseModelID: model.BaseModelID{ID: 2}, PlanID: 1, Title: "Save $100", Done: true, Position: 2000},
			3: {BaseModelID: model.BaseModelID{ID: 3}, PlanID: 2, Title: "Other user's step", Done: false},
		},
	}
}

func (r *fakePlanStepRepo) WithTx(_ *gorm.DB) repo.PlanStepRepoImpl { return r }

func (r *fakePlanStepRepo) FindOneByID(_ context.Context, id uint) (*model.PlanStep, error) {
	if s, ok := r.steps[id]; ok {
		return s, nil
	}
	return nil, gorm.ErrRecordNotFound
}

func (r *fakePlanStepRepo) FindByPlan(_ context.Context, planID uint) ([]model.PlanStep, error) {
	out := []model.PlanStep{}
	for _, s := range r.steps {
		if s.PlanID == planID {
			out = append(out, *s)
		}
	}
	return out, nil
}

func (r *fakePlanStepRepo) LastPosition(_ context.Context, planID uint) (float64, error) {
	last := 0.0
	for _, s := range r.steps {
		if s.PlanID == planID && s.Position > last {
			last = s.Position
		}
	}
	return last, nil
}

func (r *fakePlanStepRepo) Create(_ context.Context, s *model.PlanStep) (*model.PlanStep, error) {
	s.ID = r.nextID
	r.nextID++
	r.created = s
	r.steps[s.ID] = s
	return s, nil
}

func (r *fakePlanStepRepo) Update(_ context.Context, condition any, updates map[string]any) error {
	r.updates = updates
	cond, _ := condition.(map[string]any)
	id, _ := cond["id"].(uint)
	s, ok := r.steps[id]
	if !ok {
		return nil
	}
	for k, v := range updates {
		switch k {
		case "title":
			s.Title = v.(string)
		case "due_date":
			if v == nil {
				s.DueDate = nil
			} else {
				s.DueDate = v.(*time.Time)
			}
		case "done":
			s.Done = v.(bool)
		case "position":
			s.Position = v.(float64)
		}
	}
	return nil
}

func (r *fakePlanStepRepo) SoftDelete(_ context.Context, id uint) error {
	r.deletedIDs = append(r.deletedIDs, id)
	delete(r.steps, id)
	return nil
}

func newTestPlanStepSvc(stepRepo *fakePlanStepRepo, planRepo *fakePlanRepo, areaRepo *fakePlanAreaRepo) *planStepSvc {
	return &planStepSvc{
		planStepRepo: stepRepo,
		planRepo:     planRepo,
		planAreaRepo: areaRepo,
		transaction:  func(_ context.Context, fn func(tx *gorm.DB) error) error { return fn(nil) },
	}
}

// TestCreatePlanStep_IncrementsStepCount covers the rule the task called out: adding a step must
// bump the plan's step_count (and leave done_step_count untouched).
func TestCreatePlanStep_IncrementsStepCount(t *testing.T) {
	stepRepo, planRepo, areaRepo := newFakePlanStepRepo(), newFakePlanRepo(), newFakePlanAreaRepo()
	planRepo.plans[1].StepCount = 2
	planRepo.plans[1].DoneStepCount = 1

	res := newTestPlanStepSvc(stepRepo, planRepo, areaRepo).
		Create(context.Background(), 1, 1, &dto.PlanStepReq{Title: " Compare laptop prices "})

	if res.Error != nil || res.StatusCode != http.StatusCreated {
		t.Fatalf("status = %d, err = %v", res.StatusCode, res.Error)
	}
	if stepRepo.created.Title != "Compare laptop prices" || stepRepo.created.PlanID != 1 {
		t.Errorf("created = %+v", stepRepo.created)
	}
	p := planRepo.plans[1]
	if p.StepCount != 3 || p.DoneStepCount != 1 {
		t.Errorf("counts = %d/%d, want 3/1", p.StepCount, p.DoneStepCount)
	}
	if len(planRepo.adjustments) != 1 || planRepo.adjustments[0] != (stepCountAdjustment{planID: 1, stepDelta: 1, doneDelta: 0}) {
		t.Errorf("adjustments = %+v", planRepo.adjustments)
	}
}

func TestCreatePlanStep_AnotherUsersPlan(t *testing.T) {
	stepRepo, planRepo, areaRepo := newFakePlanStepRepo(), newFakePlanRepo(), newFakePlanAreaRepo()

	res := newTestPlanStepSvc(stepRepo, planRepo, areaRepo).
		Create(context.Background(), 1, 2, &dto.PlanStepReq{Title: "Not allowed"})

	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", res.StatusCode)
	}
	if stepRepo.created != nil {
		t.Error("step created despite not owning the plan")
	}
}

// TestDeletePlanStep_DecrementsStepCount covers the other half of the task: deleting a step must
// bring step_count down, and done_step_count too when the removed step was done.
func TestDeletePlanStep_DecrementsStepCount(t *testing.T) {
	tests := []struct {
		name          string
		stepID        uint
		wantStepDelta int
		wantDoneDelta int
	}{
		{name: "not-done step", stepID: 1, wantStepDelta: -1, wantDoneDelta: 0},
		{name: "done step also drops done count", stepID: 2, wantStepDelta: -1, wantDoneDelta: -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stepRepo, planRepo, areaRepo := newFakePlanStepRepo(), newFakePlanRepo(), newFakePlanAreaRepo()
			planRepo.plans[1].StepCount = 2
			planRepo.plans[1].DoneStepCount = 1

			res := newTestPlanStepSvc(stepRepo, planRepo, areaRepo).Delete(context.Background(), 1, tt.stepID, i18n.EN)

			if res.Error != nil || res.StatusCode != http.StatusOK {
				t.Fatalf("status = %d, err = %v", res.StatusCode, res.Error)
			}
			if len(stepRepo.deletedIDs) != 1 || stepRepo.deletedIDs[0] != tt.stepID {
				t.Errorf("deleted = %v, want [%d]", stepRepo.deletedIDs, tt.stepID)
			}
			if len(planRepo.adjustments) != 1 {
				t.Fatalf("adjustments = %+v", planRepo.adjustments)
			}
			adj := planRepo.adjustments[0]
			if adj.stepDelta != tt.wantStepDelta || adj.doneDelta != tt.wantDoneDelta {
				t.Errorf("adjustment = %+v, want step %d / done %d", adj, tt.wantStepDelta, tt.wantDoneDelta)
			}
			got := res.Data.(*dto.PlanStepDeleteRes)
			if got.Plan.StepCount != 2+tt.wantStepDelta || got.Plan.DoneStepCount != 1+tt.wantDoneDelta {
				t.Errorf("returned plan counts = %d/%d", got.Plan.StepCount, got.Plan.DoneStepCount)
			}
		})
	}
}

func TestDeletePlanStep_AnotherUsersStep(t *testing.T) {
	stepRepo, planRepo, areaRepo := newFakePlanStepRepo(), newFakePlanRepo(), newFakePlanAreaRepo()

	res := newTestPlanStepSvc(stepRepo, planRepo, areaRepo).Delete(context.Background(), 1, 3, i18n.EN)

	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", res.StatusCode)
	}
	if len(stepRepo.deletedIDs) != 0 {
		t.Error("step deleted despite not owning its plan")
	}
}

func TestTogglePlanStep(t *testing.T) {
	tests := []struct {
		name           string
		stepID         uint
		done           bool
		initialStatus  enums.PlanStatusEnum
		wantDoneDelta  int
		wantStatus     enums.PlanStatusEnum
		wantDoneAtNil  bool
		wantAdjustCall bool
	}{
		{name: "checking first step moves NOT_START to IN_PROGRESS", stepID: 1, done: true,
			initialStatus: enums.PlanStatusEnum(enums.PlanStatusNotStart),
			wantDoneDelta: 1, wantStatus: enums.PlanStatusEnum(enums.PlanStatusInProgress), wantAdjustCall: true},
		{name: "unchecking a done step on a DONE plan reopens it", stepID: 2, done: false,
			initialStatus: enums.PlanStatusEnum(enums.PlanStatusDone),
			wantDoneDelta: -1, wantStatus: enums.PlanStatusEnum(enums.PlanStatusInProgress), wantDoneAtNil: true, wantAdjustCall: true},
		{name: "unchecking on an IN_PROGRESS plan does not change status", stepID: 2, done: false,
			initialStatus: enums.PlanStatusEnum(enums.PlanStatusInProgress),
			wantDoneDelta: -1, wantStatus: enums.PlanStatusEnum(enums.PlanStatusInProgress), wantAdjustCall: true},
		{name: "same value is a no-op", stepID: 2, done: true,
			initialStatus: enums.PlanStatusEnum(enums.PlanStatusInProgress),
			wantDoneDelta: 0, wantStatus: enums.PlanStatusEnum(enums.PlanStatusInProgress), wantAdjustCall: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stepRepo, planRepo, areaRepo := newFakePlanStepRepo(), newFakePlanRepo(), newFakePlanAreaRepo()
			planRepo.plans[1].StepCount = 2
			planRepo.plans[1].DoneStepCount = 1
			planRepo.plans[1].Status = tt.initialStatus

			res := newTestPlanStepSvc(stepRepo, planRepo, areaRepo).
				Toggle(context.Background(), 1, tt.stepID, i18n.EN, &dto.PlanStepToggleReq{Done: tt.done})

			if res.Error != nil || res.StatusCode != http.StatusOK {
				t.Fatalf("status = %d, err = %v", res.StatusCode, res.Error)
			}
			if tt.wantAdjustCall && len(planRepo.adjustments) != 1 {
				t.Fatalf("adjustments = %+v, want 1 call", planRepo.adjustments)
			}
			if !tt.wantAdjustCall && len(planRepo.adjustments) != 0 {
				t.Fatalf("adjustments = %+v, want none", planRepo.adjustments)
			}
			p := planRepo.plans[1]
			if p.DoneStepCount != 1+tt.wantDoneDelta {
				t.Errorf("done_step_count = %d, want %d", p.DoneStepCount, 1+tt.wantDoneDelta)
			}
			if p.StepCount != 2 {
				t.Errorf("step_count = %d, want unchanged 2", p.StepCount)
			}
			if p.Status != tt.wantStatus {
				t.Errorf("status = %v, want %v", p.Status, tt.wantStatus)
			}
			if tt.wantDoneAtNil && p.DoneAt != nil {
				t.Errorf("done_at = %v, want nil", p.DoneAt)
			}
			if stepRepo.steps[tt.stepID].Done != tt.done {
				t.Errorf("step done = %v, want %v", stepRepo.steps[tt.stepID].Done, tt.done)
			}
			got := res.Data.(*dto.PlanStepToggleRes)
			if got.Step.Done != tt.done || got.Plan.DoneStepCount != p.DoneStepCount {
				t.Errorf("response = step done %v, plan done_step_count %d", got.Step.Done, got.Plan.DoneStepCount)
			}
		})
	}
}

func TestTogglePlanStep_ReachingAllDoneDoesNotAutoComplete(t *testing.T) {
	stepRepo, planRepo, areaRepo := newFakePlanStepRepo(), newFakePlanRepo(), newFakePlanAreaRepo()
	planRepo.plans[1].StepCount = 2
	planRepo.plans[1].DoneStepCount = 1
	planRepo.plans[1].Status = enums.PlanStatusEnum(enums.PlanStatusInProgress)

	res := newTestPlanStepSvc(stepRepo, planRepo, areaRepo).
		Toggle(context.Background(), 1, 1, i18n.EN, &dto.PlanStepToggleReq{Done: true})

	if res.Error != nil {
		t.Fatalf("err = %v", res.Error)
	}
	p := planRepo.plans[1]
	if p.DoneStepCount != p.StepCount {
		t.Fatalf("done_step_count = %d, step_count = %d, want equal for this test", p.DoneStepCount, p.StepCount)
	}
	if p.Status == enums.PlanStatusEnum(enums.PlanStatusDone) {
		t.Error("plan auto-completed; marking done must stay a separate, explicit action")
	}
}

func TestCreatePlanStep_Position(t *testing.T) {
	tests := []struct {
		name       string
		position   *float64
		planID     uint
		want       float64
		wantStatus int
	}{
		{name: "after the last step", planID: 1, want: 3000, wantStatus: http.StatusCreated},
		{name: "explicit position (undo a delete)", planID: 1, position: new(1500.0), want: 1500, wantStatus: http.StatusCreated},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stepRepo, planRepo, areaRepo := newFakePlanStepRepo(), newFakePlanRepo(), newFakePlanAreaRepo()

			res := newTestPlanStepSvc(stepRepo, planRepo, areaRepo).
				Create(context.Background(), 1, tt.planID, &dto.PlanStepReq{Title: "New step", Position: tt.position})

			if res.StatusCode != tt.wantStatus {
				t.Fatalf("status = %d, want %d (err %v)", res.StatusCode, tt.wantStatus, res.Error)
			}
			if got := res.Data.(*dto.PlanStepRes).Position; got != tt.want {
				t.Errorf("position = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCreatePlanStep_FirstStepAt1000(t *testing.T) {
	stepRepo, planRepo, areaRepo := newFakePlanStepRepo(), newFakePlanRepo(), newFakePlanAreaRepo()
	planRepo.plans[3] = &model.Plan{BaseModelID: model.BaseModelID{ID: 3}, UserID: 1, AreaID: 1}

	res := newTestPlanStepSvc(stepRepo, planRepo, areaRepo).
		Create(context.Background(), 1, 3, &dto.PlanStepReq{Title: "First"})

	if res.Error != nil || res.Data.(*dto.PlanStepRes).Position != 1000 {
		t.Fatalf("res = %+v", res)
	}
}

func TestMovePlanStep(t *testing.T) {
	tests := []struct {
		name       string
		userID     uint
		position   *float64
		wantStatus int
	}{
		{name: "moved", userID: 1, position: new(500.0), wantStatus: http.StatusOK},
		{name: "position missing", userID: 1, wantStatus: http.StatusBadRequest},
		{name: "another user's step", userID: 2, position: new(500.0), wantStatus: http.StatusNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stepRepo, planRepo, areaRepo := newFakePlanStepRepo(), newFakePlanRepo(), newFakePlanAreaRepo()

			res := newTestPlanStepSvc(stepRepo, planRepo, areaRepo).
				Move(context.Background(), tt.userID, 2, &dto.PlanStepMoveReq{Position: tt.position})

			if res.StatusCode != tt.wantStatus {
				t.Fatalf("status = %d, want %d (err %v)", res.StatusCode, tt.wantStatus, res.Error)
			}
			want := 2000.0
			if tt.wantStatus == http.StatusOK {
				want = *tt.position
			}
			if got := stepRepo.steps[2].Position; got != want {
				t.Errorf("stored position = %v, want %v", got, want)
			}
		})
	}
}
