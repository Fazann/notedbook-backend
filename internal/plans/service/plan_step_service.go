package service

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net/http"
	"strings"

	"github.com/Fazann/notedbook-backend/internal/plans/dto"
	"github.com/Fazann/notedbook-backend/internal/plans/repo"
	"github.com/Fazann/notedbook-backend/pkg/db"
	"github.com/Fazann/notedbook-backend/pkg/enums"
	"github.com/Fazann/notedbook-backend/pkg/i18n"
	"github.com/Fazann/notedbook-backend/pkg/response"
	model "github.com/Fazann/notedbook-backend/shared/models"
	"gorm.io/gorm"
)

const (
	planStepTitleMax = 255
	// Gap between positions of steps added at the end (the first step is at 1000).
	planStepPositionGap = 1000
)

type PlanStepServiceImpl interface {
	Create(ctx context.Context, userID, planID uint, req *dto.PlanStepReq) response.SvcRes
	Update(ctx context.Context, userID, id uint, req *dto.PlanStepUpdateReq) response.SvcRes
	Toggle(ctx context.Context, userID, id uint, lang i18n.Lang, req *dto.PlanStepToggleReq) response.SvcRes
	Move(ctx context.Context, userID, id uint, req *dto.PlanStepMoveReq) response.SvcRes
	Delete(ctx context.Context, userID, id uint, lang i18n.Lang) response.SvcRes
}

type planStepSvc struct {
	planStepRepo repo.PlanStepRepoImpl
	planRepo     repo.PlanRepoImpl
	planAreaRepo repo.PlanAreaRepoImpl
	// Runs fn in a database transaction; replaced in tests.
	transaction func(ctx context.Context, fn func(tx *gorm.DB) error) error
}

func NewPlanStepSvc() *planStepSvc {
	return &planStepSvc{
		planStepRepo: repo.NewPlanStepRepo(),
		planRepo:     repo.NewPlanRepo(),
		planAreaRepo: repo.NewPlanAreaRepo(),
		transaction: func(ctx context.Context, fn func(tx *gorm.DB) error) error {
			return db.DB.WithContext(ctx).Transaction(fn)
		},
	}
}

// BackfillStepPositions numbers the steps created before step positions existed; see repo.BackfillPositions.
func BackfillStepPositions(ctx context.Context) (int64, error) {
	return repo.BackfillPositions(ctx, db.DB)
}

// Create adds a step at the end of the plan (or at req.Position) and increments its step_count.
func (s *planStepSvc) Create(ctx context.Context, userID, planID uint, req *dto.PlanStepReq) response.SvcRes {
	plan, res, ok := s.findOwnPlan(ctx, userID, planID)
	if !ok {
		return res
	}

	title, err := cleanStepTitle(req.Title)
	if err != nil {
		return response.SvcRespErr(err, http.StatusBadRequest)
	}
	dueDate, err := parseOptionalDate(req.DueDate)
	if err != nil {
		return response.SvcRespErr(err, http.StatusBadRequest)
	}

	step := &model.PlanStep{PlanID: plan.ID, Title: title, DueDate: dueDate}
	if req.Position != nil {
		if !validPosition(*req.Position) {
			return response.SvcRespErr(i18n.Err(i18n.FieldInvalid, i18n.Params{"field": "position"}), http.StatusBadRequest)
		}
		step.Position = *req.Position
	}

	err = s.transaction(ctx, func(tx *gorm.DB) error {
		if req.Position == nil {
			last, err := s.planStepRepo.WithTx(tx).LastPosition(ctx, plan.ID)
			if err != nil {
				return fmt.Errorf("find last plan step position: %w", err)
			}
			step.Position = last + planStepPositionGap
		}
		if _, err := s.planStepRepo.WithTx(tx).Create(ctx, step); err != nil {
			return fmt.Errorf("create plan step: %w", err)
		}
		return s.planRepo.WithTx(tx).AdjustStepCounts(ctx, plan.ID, 1, 0)
	})
	if err != nil {
		return response.SvcRespErr(err, http.StatusInternalServerError)
	}

	return response.SvcRespSuccess(toPlanStepRes(step), http.StatusCreated)
}

// Update changes only the title / due date; it does not affect step_count or done_step_count.
func (s *planStepSvc) Update(ctx context.Context, userID, id uint, req *dto.PlanStepUpdateReq) response.SvcRes {
	step, _, res, ok := s.findOwnStep(ctx, userID, id)
	if !ok {
		return res
	}

	updates := map[string]any{}
	if req.Title != nil {
		title, err := cleanStepTitle(*req.Title)
		if err != nil {
			return response.SvcRespErr(err, http.StatusBadRequest)
		}
		updates["title"] = title
	}
	if req.DueDate != nil {
		dueDate, err := parseOptionalDate(req.DueDate)
		if err != nil {
			return response.SvcRespErr(err, http.StatusBadRequest)
		}
		updates["due_date"] = dueDate
	}
	if len(updates) == 0 {
		return response.SvcRespSuccess(toPlanStepRes(step), http.StatusOK)
	}

	if err := s.planStepRepo.Update(ctx, map[string]any{"id": id}, updates); err != nil {
		return response.SvcRespErr(fmt.Errorf("update plan step: %w", err), http.StatusInternalServerError)
	}

	step, err := s.planStepRepo.FindOneByID(ctx, id)
	if err != nil {
		return response.SvcRespErr(fmt.Errorf("reload plan step: %w", err), http.StatusInternalServerError)
	}
	return response.SvcRespSuccess(toPlanStepRes(step), http.StatusOK)
}

// Toggle checks or unchecks the step and keeps the plan's done_step_count in sync:
//   - checking the first step of a NOT_START plan moves it to IN_PROGRESS;
//   - unchecking a step on a DONE plan moves it back to IN_PROGRESS and clears done_at.
//
// Reaching done_step_count == step_count does not auto-complete the plan; that is a deliberate,
// separate action through UpdateStatus.
func (s *planStepSvc) Toggle(ctx context.Context, userID, id uint, lang i18n.Lang, req *dto.PlanStepToggleReq) response.SvcRes {
	step, plan, res, ok := s.findOwnStep(ctx, userID, id)
	if !ok {
		return res
	}

	if step.Done != req.Done {
		doneDelta := 1
		planUpdates := map[string]any{}
		if !req.Done {
			doneDelta = -1
			if plan.Status == enums.PlanStatusEnum(enums.PlanStatusDone) {
				planUpdates["status"] = enums.PlanStatusEnum(enums.PlanStatusInProgress)
				planUpdates["done_at"] = nil
			}
		} else if plan.Status == enums.PlanStatusEnum(enums.PlanStatusNotStart) {
			planUpdates["status"] = enums.PlanStatusEnum(enums.PlanStatusInProgress)
		}

		err := s.transaction(ctx, func(tx *gorm.DB) error {
			if err := s.planStepRepo.WithTx(tx).Update(ctx, map[string]any{"id": id}, map[string]any{"done": req.Done}); err != nil {
				return fmt.Errorf("toggle plan step: %w", err)
			}
			if err := s.planRepo.WithTx(tx).AdjustStepCounts(ctx, plan.ID, 0, doneDelta); err != nil {
				return fmt.Errorf("adjust plan step counts: %w", err)
			}
			if len(planUpdates) > 0 {
				if err := s.planRepo.WithTx(tx).Update(ctx, map[string]any{"id": plan.ID}, planUpdates); err != nil {
					return fmt.Errorf("update plan status: %w", err)
				}
			}
			return nil
		})
		if err != nil {
			return response.SvcRespErr(err, http.StatusInternalServerError)
		}
		step.Done = req.Done
	}

	plan, err := s.planRepo.FindOneByID(ctx, userID, plan.ID)
	if err != nil {
		return response.SvcRespErr(fmt.Errorf("reload plan: %w", err), http.StatusInternalServerError)
	}
	area, res, ok := s.findArea(ctx, plan.AreaID)
	if !ok {
		return res
	}

	return response.SvcRespSuccess(&dto.PlanStepToggleRes{
		Step: toPlanStepRes(step),
		Plan: toPlanRes(plan, area, lang),
	}, http.StatusOK)
}

// Move sets the step's position inside its plan; the client computes it from the new neighbours.
func (s *planStepSvc) Move(ctx context.Context, userID, id uint, req *dto.PlanStepMoveReq) response.SvcRes {
	step, _, res, ok := s.findOwnStep(ctx, userID, id)
	if !ok {
		return res
	}
	if req.Position == nil || !validPosition(*req.Position) {
		return response.SvcRespErr(i18n.Err(i18n.FieldInvalid, i18n.Params{"field": "position"}), http.StatusBadRequest)
	}

	if err := s.planStepRepo.Update(ctx, map[string]any{"id": id}, map[string]any{"position": *req.Position}); err != nil {
		return response.SvcRespErr(fmt.Errorf("move plan step: %w", err), http.StatusInternalServerError)
	}
	step.Position = *req.Position

	return response.SvcRespSuccess(toPlanStepRes(step), http.StatusOK)
}

func validPosition(p float64) bool {
	return !math.IsNaN(p) && !math.IsInf(p, 0)
}

// Delete removes the step and decrements the plan's step_count (and done_step_count if it was done).
func (s *planStepSvc) Delete(ctx context.Context, userID, id uint, lang i18n.Lang) response.SvcRes {
	step, plan, res, ok := s.findOwnStep(ctx, userID, id)
	if !ok {
		return res
	}

	doneDelta := 0
	if step.Done {
		doneDelta = -1
	}

	err := s.transaction(ctx, func(tx *gorm.DB) error {
		if err := s.planStepRepo.WithTx(tx).SoftDelete(ctx, id); err != nil {
			return fmt.Errorf("delete plan step: %w", err)
		}
		return s.planRepo.WithTx(tx).AdjustStepCounts(ctx, plan.ID, -1, doneDelta)
	})
	if err != nil {
		return response.SvcRespErr(err, http.StatusInternalServerError)
	}

	plan, err = s.planRepo.FindOneByID(ctx, userID, plan.ID)
	if err != nil {
		return response.SvcRespErr(fmt.Errorf("reload plan: %w", err), http.StatusInternalServerError)
	}
	area, res, ok := s.findArea(ctx, plan.AreaID)
	if !ok {
		return res
	}

	return response.SvcRespSuccess(&dto.PlanStepDeleteRes{Plan: toPlanRes(plan, area, lang)}, http.StatusOK)
}

// findOwnPlan returns the user's plan; ok is false with the error response otherwise.
func (s *planStepSvc) findOwnPlan(ctx context.Context, userID, planID uint) (*model.Plan, response.SvcRes, bool) {
	plan, err := s.planRepo.FindOneByID(ctx, userID, planID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, response.SvcRespErr(i18n.RecordNotFound, http.StatusNotFound), false
		}
		return nil, response.SvcRespErr(err, http.StatusInternalServerError), false
	}
	return plan, response.SvcRes{}, true
}

// findOwnStep returns the step and its plan when the plan belongs to userID; ok is false with the
// error response otherwise.
func (s *planStepSvc) findOwnStep(ctx context.Context, userID, id uint) (*model.PlanStep, *model.Plan, response.SvcRes, bool) {
	step, err := s.planStepRepo.FindOneByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, response.SvcRespErr(i18n.RecordNotFound, http.StatusNotFound), false
		}
		return nil, nil, response.SvcRespErr(err, http.StatusInternalServerError), false
	}
	plan, res, ok := s.findOwnPlan(ctx, userID, step.PlanID)
	if !ok {
		return nil, nil, res, false
	}
	return step, plan, response.SvcRes{}, true
}

// findArea checks that areaID is one of the plan areas.
func (s *planStepSvc) findArea(ctx context.Context, areaID uint) (*model.PlanArea, response.SvcRes, bool) {
	area, err := s.planAreaRepo.FindOne(ctx, "id = ?", areaID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, response.SvcRespErr(i18n.InvalidPlanArea, http.StatusBadRequest), false
		}
		return nil, response.SvcRespErr(err, http.StatusInternalServerError), false
	}
	return area, response.SvcRes{}, true
}

func cleanStepTitle(title string) (string, error) {
	title = strings.TrimSpace(title)
	if n := len([]rune(title)); n == 0 || n > planStepTitleMax {
		return "", i18n.Err(i18n.FieldMaxLength, i18n.Params{"field": "title", "max": planStepTitleMax})
	}
	return title, nil
}

func toPlanStepRes(step *model.PlanStep) *dto.PlanStepRes {
	return &dto.PlanStepRes{
		ID:        step.ID,
		PlanID:    step.PlanID,
		Title:     step.Title,
		Done:      step.Done,
		DueDate:   formatOptionalDate(step.DueDate),
		Position:  step.Position,
		CreatedAt: step.CreatedAt,
		UpdatedAt: step.UpdatedAt,
	}
}

func toPlanStepResList(steps []model.PlanStep) []*dto.PlanStepRes {
	res := make([]*dto.PlanStepRes, len(steps))
	for i := range steps {
		res[i] = toPlanStepRes(&steps[i])
	}
	return res
}
