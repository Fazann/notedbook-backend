package service

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/Fazann/notedbook-backend/internal/plans/dto"
	"github.com/Fazann/notedbook-backend/internal/plans/repo"
	"github.com/Fazann/notedbook-backend/pkg/constants"
	"github.com/Fazann/notedbook-backend/pkg/enums"
	"github.com/Fazann/notedbook-backend/pkg/i18n"
	"github.com/Fazann/notedbook-backend/pkg/response"
	model "github.com/Fazann/notedbook-backend/shared/models"
	"gorm.io/gorm"
)

const (
	planDateLayout = "2006-01-02"
	planTitleMin   = 2
	planTitleMax   = 255
	planDescMax    = 2000
)

type PlanServiceImpl interface {
	List(ctx context.Context, userID uint, lang i18n.Lang, req *dto.PlanListReq) response.SvcResWithPaginate
	GetDetail(ctx context.Context, userID, id uint, lang i18n.Lang) response.SvcRes
	Create(ctx context.Context, userID uint, lang i18n.Lang, req *dto.PlanReq) response.SvcRes
	Update(ctx context.Context, userID, id uint, lang i18n.Lang, req *dto.PlanUpdateReq) response.SvcRes
	UpdateStatus(ctx context.Context, userID, id uint, lang i18n.Lang, req *dto.PlanStatusReq) response.SvcRes
	Delete(ctx context.Context, userID, id uint) response.SvcRes
	Stats(ctx context.Context, userID uint, req *dto.PlanStatsReq) response.SvcRes
}

type planSvc struct {
	planRepo     repo.PlanRepoImpl
	planAreaRepo repo.PlanAreaRepoImpl
	planStepRepo repo.PlanStepRepoImpl
}

func NewPlanSvc() *planSvc {
	return &planSvc{
		planRepo:     repo.NewPlanRepo(),
		planAreaRepo: repo.NewPlanAreaRepo(),
		planStepRepo: repo.NewPlanStepRepo(),
	}
}

func (s *planSvc) List(ctx context.Context, userID uint, lang i18n.Lang, req *dto.PlanListReq) response.SvcResWithPaginate {
	page, limit := req.Page, req.Limit
	if page <= 0 {
		page = constants.DefaultPage
	}
	if limit <= 0 {
		limit = constants.DefaultLimit
	}

	filter, err := planListFilter(req)
	if err != nil {
		return response.SvcResWithPaginate{Error: err, StatusCode: http.StatusBadRequest}
	}

	plans, total, err := s.planRepo.List(ctx, userID, filter, page, limit)
	if err != nil {
		return response.SvcResWithPaginate{Error: fmt.Errorf("list plans: %w", err), StatusCode: http.StatusInternalServerError}
	}

	areas, err := s.areaMap(ctx)
	if err != nil {
		return response.SvcResWithPaginate{Error: fmt.Errorf("list plan areas: %w", err), StatusCode: http.StatusInternalServerError}
	}

	return response.SvcRespPage(toPlanResList(plans, areas, lang), page, limit, total)
}

func (s *planSvc) GetDetail(ctx context.Context, userID, id uint, lang i18n.Lang) response.SvcRes {
	plan, res, ok := s.findOwn(ctx, userID, id)
	if !ok {
		return res
	}

	steps, err := s.planStepRepo.FindByPlan(ctx, plan.ID)
	if err != nil {
		return response.SvcRespErr(fmt.Errorf("list plan steps: %w", err), http.StatusInternalServerError)
	}

	area, res, ok := s.findArea(ctx, plan.AreaID)
	if !ok {
		return res
	}

	return response.SvcRespSuccess(toPlanDetailRes(plan, area, lang, steps), http.StatusOK)
}

func (s *planSvc) Create(ctx context.Context, userID uint, lang i18n.Lang, req *dto.PlanReq) response.SvcRes {
	title, err := cleanPlanTitle(req.Title)
	if err != nil {
		return response.SvcRespErr(err, http.StatusBadRequest)
	}
	description, err := cleanPlanDescription(req.Description)
	if err != nil {
		return response.SvcRespErr(err, http.StatusBadRequest)
	}
	area, res, ok := s.findArea(ctx, req.AreaID)
	if !ok {
		return res
	}
	targetDate, err := parseOptionalDate(req.TargetDate)
	if err != nil {
		return response.SvcRespErr(err, http.StatusBadRequest)
	}

	plan := &model.Plan{
		UserID:      userID,
		Title:       title,
		Description: description,
		AreaID:      area.ID,
		Priority:    req.Priority,
		TargetDate:  targetDate,
		Status:      enums.PlanStatusEnum(enums.PlanStatusNotStart),
	}
	if _, err := s.planRepo.Create(ctx, plan); err != nil {
		return response.SvcRespErr(fmt.Errorf("create plan: %w", err), http.StatusInternalServerError)
	}

	return response.SvcRespSuccess(toPlanRes(plan, area, lang), http.StatusCreated)
}

func (s *planSvc) Update(ctx context.Context, userID, id uint, lang i18n.Lang, req *dto.PlanUpdateReq) response.SvcRes {
	_, res, ok := s.findOwn(ctx, userID, id)
	if !ok {
		return res
	}

	updates := map[string]any{}
	if req.Title != nil {
		title, err := cleanPlanTitle(*req.Title)
		if err != nil {
			return response.SvcRespErr(err, http.StatusBadRequest)
		}
		updates["title"] = title
	}
	if req.Description != nil {
		description, err := cleanPlanDescription(*req.Description)
		if err != nil {
			return response.SvcRespErr(err, http.StatusBadRequest)
		}
		updates["description"] = description
	}
	if req.AreaID != nil {
		area, res, ok := s.findArea(ctx, *req.AreaID)
		if !ok {
			return res
		}
		updates["area_id"] = area.ID
	}
	if req.Priority != nil {
		updates["priority"] = *req.Priority
	}
	if req.TargetDate != nil {
		targetDate, err := parseOptionalDate(req.TargetDate)
		if err != nil {
			return response.SvcRespErr(err, http.StatusBadRequest)
		}
		updates["target_date"] = targetDate
	}
	if len(updates) == 0 {
		return s.GetDetail(ctx, userID, id, lang)
	}

	if err := s.planRepo.Update(ctx, map[string]any{"id": id, "user_id": userID}, updates); err != nil {
		return response.SvcRespErr(fmt.Errorf("update plan: %w", err), http.StatusInternalServerError)
	}

	return s.GetDetail(ctx, userID, id, lang)
}

// UpdateStatus sets the status directly; it does not check or uncheck steps (the reverse of a step
// toggle, which can move the status between IN_PROGRESS and DONE on its own).
func (s *planSvc) UpdateStatus(ctx context.Context, userID, id uint, lang i18n.Lang, req *dto.PlanStatusReq) response.SvcRes {
	plan, res, ok := s.findOwn(ctx, userID, id)
	if !ok {
		return res
	}
	if plan.Status == req.Status {
		return s.GetDetail(ctx, userID, id, lang)
	}

	updates := map[string]any{"status": req.Status}
	if req.Status == enums.PlanStatusEnum(enums.PlanStatusDone) {
		updates["done_at"] = time.Now()
	} else {
		updates["done_at"] = nil
	}

	if err := s.planRepo.Update(ctx, map[string]any{"id": id, "user_id": userID}, updates); err != nil {
		return response.SvcRespErr(fmt.Errorf("update plan status: %w", err), http.StatusInternalServerError)
	}

	return s.GetDetail(ctx, userID, id, lang)
}

// Delete soft-deletes the plan; its steps are deleted too.
func (s *planSvc) Delete(ctx context.Context, userID, id uint) response.SvcRes {
	if _, res, ok := s.findOwn(ctx, userID, id); !ok {
		return res
	}

	if err := s.planRepo.SoftDelete(ctx, userID, id); err != nil {
		return response.SvcRespErr(fmt.Errorf("delete plan: %w", err), http.StatusInternalServerError)
	}

	return response.SvcRespSuccess(nil, http.StatusOK)
}

// Stats summarizes the user's plans as of req.Today (default: today in UTC).
func (s *planSvc) Stats(ctx context.Context, userID uint, req *dto.PlanStatsReq) response.SvcRes {
	today := time.Now().UTC().Truncate(24 * time.Hour)
	if req.Today != "" {
		date, err := time.Parse(planDateLayout, req.Today)
		if err != nil {
			return response.SvcRespErr(i18n.Err(i18n.FieldInvalid, i18n.Params{"field": "today"}), http.StatusBadRequest)
		}
		today = date
	}
	yearStart := time.Date(today.Year(), time.January, 1, 0, 0, 0, 0, time.UTC)

	stats, err := s.planRepo.Stats(ctx, userID, today, yearStart, yearStart.AddDate(1, 0, 0))
	if err != nil {
		return response.SvcRespErr(fmt.Errorf("plan stats: %w", err), http.StatusInternalServerError)
	}

	return response.SvcRespSuccess(&dto.PlanStatsRes{
		Active:          stats.Active,
		DoneThisYear:    stats.DoneThisYear,
		Overdue:         stats.Overdue,
		AverageProgress: int(math.Round(stats.AverageProgress)),
	}, http.StatusOK)
}

// findOwn returns the user's plan; ok is false with the error response otherwise.
func (s *planSvc) findOwn(ctx context.Context, userID, id uint) (*model.Plan, response.SvcRes, bool) {
	plan, err := s.planRepo.FindOneByID(ctx, userID, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, response.SvcRespErr(i18n.RecordNotFound, http.StatusNotFound), false
		}
		return nil, response.SvcRespErr(err, http.StatusInternalServerError), false
	}
	return plan, response.SvcRes{}, true
}

// findArea checks that areaID is one of the plan areas.
func (s *planSvc) findArea(ctx context.Context, areaID uint) (*model.PlanArea, response.SvcRes, bool) {
	area, err := s.planAreaRepo.FindOne(ctx, "id = ?", areaID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, response.SvcRespErr(i18n.InvalidPlanArea, http.StatusBadRequest), false
		}
		return nil, response.SvcRespErr(err, http.StatusInternalServerError), false
	}
	return area, response.SvcRes{}, true
}

// areaMap loads every plan area, keyed by ID, for resolving a list of plans without one query per row.
func (s *planSvc) areaMap(ctx context.Context) (map[uint]model.PlanArea, error) {
	areas, err := s.planAreaRepo.FindAll(ctx)
	if err != nil {
		return nil, err
	}
	m := make(map[uint]model.PlanArea, len(areas))
	for _, area := range areas {
		m[area.ID] = area
	}
	return m, nil
}

func cleanPlanTitle(title string) (string, error) {
	title = strings.TrimSpace(title)
	if n := len([]rune(title)); n < planTitleMin || n > planTitleMax {
		return "", i18n.Err(i18n.FieldMinLength, i18n.Params{"field": "title", "min": planTitleMin})
	}
	return title, nil
}

func cleanPlanDescription(description string) (string, error) {
	description = strings.TrimSpace(description)
	if n := len([]rune(description)); n > planDescMax {
		return "", i18n.Err(i18n.FieldMaxLength, i18n.Params{"field": "description", "max": planDescMax})
	}
	return description, nil
}

// parseOptionalDate parses a plain YYYY-MM-DD date; nil or "" clears the date.
func parseOptionalDate(value *string) (*time.Time, error) {
	if value == nil || *value == "" {
		return nil, nil
	}
	date, err := time.Parse(planDateLayout, *value)
	if err != nil {
		return nil, i18n.Err(i18n.FieldInvalid, i18n.Params{"field": "target_date"})
	}
	return &date, nil
}

var likeEscaper = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)

var planStatuses = []string{enums.PlanStatusNotStart, enums.PlanStatusInProgress, enums.PlanStatusDone}

// planListFilter builds the list query; status and area_id are comma-separated lists.
func planListFilter(req *dto.PlanListReq) (*repo.PlanFilter, error) {
	filter := &repo.PlanFilter{
		Search: likeEscaper.Replace(strings.ToLower(strings.TrimSpace(req.Q))),
		SortID: new("asc"),
	}

	for _, part := range splitList(req.Status) {
		if !slices.Contains(planStatuses, part) {
			return nil, i18n.Err(i18n.FieldInvalid, i18n.Params{"field": "status"})
		}
		filter.Statuses = append(filter.Statuses, part)
	}
	for _, part := range splitList(req.AreaID) {
		id, err := strconv.Atoi(part)
		if err != nil || id <= 0 {
			return nil, i18n.Err(i18n.FieldInvalid, i18n.Params{"field": "area_id"})
		}
		filter.AreaIDs = append(filter.AreaIDs, id)
	}

	order := "asc"
	key, desc := strings.CutPrefix(req.Sort, "-")
	if desc {
		order = "desc"
	}
	switch key {
	case "created_at":
		filter.SortCreatedAt = &order
	case "title":
		filter.SortTitle = &order
	case "progress":
		filter.SortProgress = &order
	default:
		filter.SortTargetDate = &order
	}
	return filter, nil
}

// splitList splits "a, b,,c" into ["a" "b" "c"].
func splitList(value string) []string {
	var parts []string
	for part := range strings.SplitSeq(value, ",") {
		if part = strings.TrimSpace(part); part != "" {
			parts = append(parts, part)
		}
	}
	return parts
}

// calcProgress rounds done/total to a percentage; 0 with no steps, 100 once status is DONE.
func calcProgress(done, total int, status enums.PlanStatusEnum) int {
	if status == enums.PlanStatusEnum(enums.PlanStatusDone) {
		return 100
	}
	if total <= 0 {
		return 0
	}
	return int(math.Round(float64(done) / float64(total) * 100))
}

func formatOptionalDate(date *time.Time) *string {
	if date == nil {
		return nil
	}
	formatted := date.Format(planDateLayout)
	return &formatted
}

func toAreaPlanRes(area *model.PlanArea, lang i18n.Lang) dto.AreaPlanRes {
	title := area.Title
	for _, tr := range area.Translations {
		if tr.Lang == string(lang) && tr.Title != "" {
			title = tr.Title
			break
		}
	}
	return dto.AreaPlanRes{ID: area.ID, Title: title}
}

func toPlanRes(plan *model.Plan, area *model.PlanArea, lang i18n.Lang) *dto.PlanRes {
	return &dto.PlanRes{
		ID:            plan.ID,
		Title:         plan.Title,
		Description:   plan.Description,
		Area:          toAreaPlanRes(area, lang),
		Priority:      plan.Priority,
		Status:        plan.Status,
		TargetDate:    formatOptionalDate(plan.TargetDate),
		StepCount:     plan.StepCount,
		DoneStepCount: plan.DoneStepCount,
		Progress:      calcProgress(plan.DoneStepCount, plan.StepCount, plan.Status),
		DoneAt:        plan.DoneAt,
		CreatedAt:     plan.CreatedAt,
		UpdatedAt:     plan.UpdatedAt,
	}
}

func toPlanResList(plans []model.Plan, areas map[uint]model.PlanArea, lang i18n.Lang) []*dto.PlanRes {
	res := make([]*dto.PlanRes, len(plans))
	for i := range plans {
		area := areas[plans[i].AreaID]
		res[i] = toPlanRes(&plans[i], &area, lang)
	}
	return res
}

func toPlanDetailRes(plan *model.Plan, area *model.PlanArea, lang i18n.Lang, steps []model.PlanStep) *dto.PlanDetailRes {
	return &dto.PlanDetailRes{
		PlanRes: *toPlanRes(plan, area, lang),
		Steps:   toPlanStepResList(steps),
	}
}
