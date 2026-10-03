package service

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/Fazann/notedbook-backend/internal/schedule/dto"
	"github.com/Fazann/notedbook-backend/internal/schedule/repo"
	"github.com/Fazann/notedbook-backend/pkg/db"
	"github.com/Fazann/notedbook-backend/pkg/enums"
	"github.com/Fazann/notedbook-backend/pkg/i18n"
	"github.com/Fazann/notedbook-backend/pkg/response"
	model "github.com/Fazann/notedbook-backend/shared/models"
	"gorm.io/gorm"
)

const (
	dateLayout = "2006-01-02"
	timeLayout = "15:04"

	activityTitleMax    = 100
	activityLocationMax = 100
	activityNoteMax     = 500
	minActivityMinutes  = 5
	maxRangeDays        = 42

	scopeThis = "this"
)

type ScheduleServiceImpl interface {
	ListOccurrences(ctx context.Context, userID uint, req *dto.OccurrenceListReq) response.SvcRes
	GetWeekSummary(ctx context.Context, userID uint, req *dto.OccurrenceListReq) response.SvcRes
	GetActivity(ctx context.Context, userID, id uint) response.SvcRes
	CreateActivity(ctx context.Context, userID uint, req *dto.ScheduleActivityReq) response.SvcRes
	UpdateActivity(ctx context.Context, userID, id uint, scope *dto.ScheduleActivityScopeReq, req *dto.ScheduleActivityReq) response.SvcRes
	DeleteActivity(ctx context.Context, userID, id uint, scope *dto.ScheduleActivityScopeReq) response.SvcRes
}

type scheduleSvc struct {
	activityRepo repo.ScheduleActivityRepoImpl
	// Runs fn in a database transaction; replaced in tests.
	transaction func(ctx context.Context, fn func(tx *gorm.DB) error) error
}

func NewScheduleSvc() *scheduleSvc {
	return &scheduleSvc{
		activityRepo: repo.NewScheduleActivityRepo(),
		transaction: func(ctx context.Context, fn func(tx *gorm.DB) error) error {
			return db.DB.WithContext(ctx).Transaction(fn)
		},
	}
}

func (s *scheduleSvc) ListOccurrences(ctx context.Context, userID uint, req *dto.OccurrenceListReq) response.SvcRes {
	from, to, err := parseRange(req)
	if err != nil {
		return response.SvcRespErr(err, http.StatusBadRequest)
	}

	activities, err := s.activityRepo.FindInRange(ctx, userID, from, to)
	if err != nil {
		return response.SvcRespErr(fmt.Errorf("list schedule activities: %w", err), http.StatusInternalServerError)
	}

	return response.SvcRespSuccess(expandOccurrences(activities, from, to), http.StatusOK)
}

func (s *scheduleSvc) GetWeekSummary(ctx context.Context, userID uint, req *dto.OccurrenceListReq) response.SvcRes {
	from, to, err := parseRange(req)
	if err != nil {
		return response.SvcRespErr(err, http.StatusBadRequest)
	}

	activities, err := s.activityRepo.FindInRange(ctx, userID, from, to)
	if err != nil {
		return response.SvcRespErr(fmt.Errorf("list schedule activities: %w", err), http.StatusInternalServerError)
	}

	return response.SvcRespSuccess(summarize(expandOccurrences(activities, from, to)), http.StatusOK)
}

func (s *scheduleSvc) GetActivity(ctx context.Context, userID, id uint) response.SvcRes {
	activity, res, ok := s.findOwn(ctx, userID, id)
	if !ok {
		return res
	}

	return response.SvcRespSuccess(toActivityRes(activity), http.StatusOK)
}

func (s *scheduleSvc) CreateActivity(ctx context.Context, userID uint, req *dto.ScheduleActivityReq) response.SvcRes {
	input, err := cleanActivityReq(req)
	if err != nil {
		return response.SvcRespErr(err, http.StatusBadRequest)
	}

	activity := &model.ScheduleActivity{
		UserID:          userID,
		Title:           input.title,
		Type:            input.typ,
		Date:            input.date,
		StartTime:       input.startTime,
		EndTime:         input.endTime,
		Location:        input.location,
		Note:            input.note,
		Recurrence:      input.recurrenceKind,
		RecurrenceUntil: input.until,
		BaseModelGorm:   model.BaseModelGorm{CreatedBy: userID, UpdatedBy: userID},
	}

	err = s.transaction(ctx, func(tx *gorm.DB) error {
		repoTx := s.activityRepo.WithTx(tx)
		if _, err := repoTx.Create(ctx, activity); err != nil {
			return fmt.Errorf("create schedule activity: %w", err)
		}
		if len(input.days) == 0 {
			return nil
		}
		return repoTx.ReplaceDays(ctx, activity.ID, toDayModels(input.days))
	})
	if err != nil {
		return response.SvcRespErr(err, http.StatusInternalServerError)
	}

	activity.Days = toDayModels(input.days)
	return response.SvcRespSuccess(toActivityRes(activity), http.StatusCreated)
}

// UpdateActivity applies req either to the whole series (scope "all") or, for scope "this", creates
// a standalone activity for the one occurrence named by scope.Date and marks that date an exception
// on the series.
func (s *scheduleSvc) UpdateActivity(ctx context.Context, userID, id uint, scope *dto.ScheduleActivityScopeReq, req *dto.ScheduleActivityReq) response.SvcRes {
	series, res, ok := s.findOwn(ctx, userID, id)
	if !ok {
		return res
	}

	input, err := cleanActivityReq(req)
	if err != nil {
		return response.SvcRespErr(err, http.StatusBadRequest)
	}

	if scope.Scope == scopeThis {
		return s.updateThisOccurrence(ctx, userID, series, scope.Date, input)
	}
	return s.updateSeries(ctx, series, input)
}

func (s *scheduleSvc) updateSeries(ctx context.Context, series *model.ScheduleActivity, input *activityInput) response.SvcRes {
	err := s.transaction(ctx, func(tx *gorm.DB) error {
		repoTx := s.activityRepo.WithTx(tx)
		updates := map[string]any{
			"title":            input.title,
			"type":             input.typ,
			"date":             input.date,
			"start_time":       input.startTime,
			"end_time":         input.endTime,
			"location":         input.location,
			"note":             input.note,
			"recurrence":       input.recurrenceKind,
			"recurrence_until": input.until,
			"updated_by":       series.UserID,
		}
		if err := repoTx.Update(ctx, map[string]any{"id": series.ID, "user_id": series.UserID}, updates); err != nil {
			return fmt.Errorf("update schedule activity: %w", err)
		}
		return repoTx.ReplaceDays(ctx, series.ID, toDayModels(input.days))
	})
	if err != nil {
		return response.SvcRespErr(err, http.StatusInternalServerError)
	}

	return s.GetActivity(ctx, series.UserID, series.ID)
}

func (s *scheduleSvc) updateThisOccurrence(ctx context.Context, userID uint, series *model.ScheduleActivity, dateStr string, input *activityInput) response.SvcRes {
	if dateStr == "" {
		return response.SvcRespErr(i18n.ScopeDateRequired, http.StatusBadRequest)
	}
	exceptionDate, err := parseDate(dateStr)
	if err != nil {
		return response.SvcRespErr(err, http.StatusBadRequest)
	}

	created := &model.ScheduleActivity{
		UserID:        userID,
		Title:         input.title,
		Type:          input.typ,
		Date:          input.date,
		StartTime:     input.startTime,
		EndTime:       input.endTime,
		Location:      input.location,
		Note:          input.note,
		Recurrence:    enums.RecurrenceKindEnum(enums.RecurrenceKindNone),
		BaseModelGorm: model.BaseModelGorm{CreatedBy: userID, UpdatedBy: userID},
	}

	err = s.transaction(ctx, func(tx *gorm.DB) error {
		repoTx := s.activityRepo.WithTx(tx)
		if err := repoTx.AddException(ctx, series.ID, exceptionDate); err != nil {
			return fmt.Errorf("add schedule exception: %w", err)
		}
		if _, err := repoTx.Create(ctx, created); err != nil {
			return fmt.Errorf("create schedule activity: %w", err)
		}
		return nil
	})
	if err != nil {
		return response.SvcRespErr(err, http.StatusInternalServerError)
	}

	seriesReloaded, err := s.activityRepo.FindOneByID(ctx, userID, series.ID)
	if err != nil {
		return response.SvcRespErr(fmt.Errorf("reload schedule series: %w", err), http.StatusInternalServerError)
	}

	return response.SvcRespSuccess(&dto.ScheduleActivityUpdateRes{
		Series:  toActivityRes(seriesReloaded),
		Created: toActivityRes(created),
	}, http.StatusOK)
}

// DeleteActivity deletes the whole series (scope "all") or, for scope "this", only removes the one
// occurrence named by scope.Date by adding it as an exception.
func (s *scheduleSvc) DeleteActivity(ctx context.Context, userID, id uint, scope *dto.ScheduleActivityScopeReq) response.SvcRes {
	series, res, ok := s.findOwn(ctx, userID, id)
	if !ok {
		return res
	}

	if scope.Scope == scopeThis {
		if scope.Date == "" {
			return response.SvcRespErr(i18n.ScopeDateRequired, http.StatusBadRequest)
		}
		exceptionDate, err := parseDate(scope.Date)
		if err != nil {
			return response.SvcRespErr(err, http.StatusBadRequest)
		}
		if err := s.activityRepo.AddException(ctx, series.ID, exceptionDate); err != nil {
			return response.SvcRespErr(fmt.Errorf("add schedule exception: %w", err), http.StatusInternalServerError)
		}
		return response.SvcRespSuccess(nil, http.StatusOK)
	}

	if err := s.activityRepo.SoftDelete(ctx, userID, id); err != nil {
		return response.SvcRespErr(fmt.Errorf("delete schedule activity: %w", err), http.StatusInternalServerError)
	}
	return response.SvcRespSuccess(nil, http.StatusOK)
}

// findOwn returns the user's activity; ok is false with the error response otherwise.
func (s *scheduleSvc) findOwn(ctx context.Context, userID, id uint) (*model.ScheduleActivity, response.SvcRes, bool) {
	activity, err := s.activityRepo.FindOneByID(ctx, userID, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, response.SvcRespErr(i18n.RecordNotFound, http.StatusNotFound), false
		}
		return nil, response.SvcRespErr(err, http.StatusInternalServerError), false
	}
	return activity, response.SvcRes{}, true
}

// parseRange parses from/to and checks the range is ordered and at most maxRangeDays (inclusive).
func parseRange(req *dto.OccurrenceListReq) (time.Time, time.Time, error) {
	from, err := parseDate(req.From)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	to, err := parseDate(req.To)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	days := int(to.Sub(from).Hours()/24) + 1
	if days < 1 || days > maxRangeDays {
		return time.Time{}, time.Time{}, i18n.Err(i18n.InvalidDateRange, i18n.Params{"max": maxRangeDays})
	}
	return from, to, nil
}

type activityInput struct {
	title          string
	typ            enums.ScheduleActivityTypeEnum
	date           time.Time
	startTime      string
	endTime        string
	location       string
	note           string
	recurrenceKind enums.RecurrenceKindEnum
	days           []int
	until          *time.Time
}

// cleanActivityReq trims strings and checks the business rules the form can't express with struct
// tags alone: end after start, minimum duration, weekly needs at least one day, until >= date.
func cleanActivityReq(req *dto.ScheduleActivityReq) (*activityInput, error) {
	title := strings.TrimSpace(req.Title)
	if n := utf8.RuneCountInString(title); n == 0 || n > activityTitleMax {
		return nil, i18n.Err(i18n.FieldInvalid, i18n.Params{"field": "title"})
	}

	date, err := parseDate(req.Date)
	if err != nil {
		return nil, err
	}
	startTime, err := cleanTime(req.StartTime, "start_time")
	if err != nil {
		return nil, err
	}
	endTime, err := cleanTime(req.EndTime, "end_time")
	if err != nil {
		return nil, err
	}
	duration, err := durationMinutes(startTime, endTime)
	if err != nil {
		return nil, err
	}
	if duration <= 0 {
		return nil, i18n.InvalidTimeRange
	}
	if duration < minActivityMinutes {
		return nil, i18n.Err(i18n.ActivityTooShort, i18n.Params{"min": minActivityMinutes})
	}

	location := strings.TrimSpace(req.Location)
	if utf8.RuneCountInString(location) > activityLocationMax {
		return nil, i18n.Err(i18n.FieldMaxLength, i18n.Params{"field": "location", "max": activityLocationMax})
	}
	note := strings.TrimSpace(req.Note)
	if utf8.RuneCountInString(note) > activityNoteMax {
		return nil, i18n.Err(i18n.FieldMaxLength, i18n.Params{"field": "note", "max": activityNoteMax})
	}

	days, until, err := cleanRecurrence(req.Recurrence, date)
	if err != nil {
		return nil, err
	}

	return &activityInput{
		title: title, typ: req.Type, date: date, startTime: startTime, endTime: endTime,
		location: location, note: note, recurrenceKind: req.Recurrence.Kind, days: days, until: until,
	}, nil
}

// cleanRecurrence checks weekly has at least one (deduplicated, sorted) day and until is not before
// the activity's own date; days and until are dropped for a non-weekly / non-repeating kind.
func cleanRecurrence(req dto.RecurrenceReq, activityDate time.Time) ([]int, *time.Time, error) {
	var days []int
	if req.Kind == enums.RecurrenceKindEnum(enums.RecurrenceKindWeekly) {
		if len(req.Days) == 0 {
			return nil, nil, i18n.RecurrenceDaysRequired
		}
		seen := make(map[int]bool, len(req.Days))
		for _, d := range req.Days {
			if d < 1 || d > 7 {
				return nil, nil, i18n.Err(i18n.FieldInvalid, i18n.Params{"field": "recurrence.days"})
			}
			if !seen[d] {
				seen[d] = true
				days = append(days, d)
			}
		}
		sort.Ints(days)
	}

	if req.Kind == enums.RecurrenceKindEnum(enums.RecurrenceKindNone) || req.Until == nil {
		return days, nil, nil
	}

	until, err := parseDate(*req.Until)
	if err != nil {
		return nil, nil, err
	}
	if until.Before(activityDate) {
		return nil, nil, i18n.InvalidRecurrenceUntil
	}
	return days, &until, nil
}

func parseDate(value string) (time.Time, error) {
	date, err := time.Parse(dateLayout, value)
	if err != nil {
		return time.Time{}, i18n.Err(i18n.FieldInvalid, i18n.Params{"field": "date"})
	}
	return date, nil
}

func cleanTime(value, field string) (string, error) {
	if _, err := time.Parse(timeLayout, value); err != nil {
		return "", i18n.Err(i18n.FieldInvalid, i18n.Params{"field": field})
	}
	return value, nil
}

// durationMinutes is end - start, which is negative when end is not after start.
func durationMinutes(start, end string) (int, error) {
	s, err := time.Parse(timeLayout, start)
	if err != nil {
		return 0, i18n.Err(i18n.FieldInvalid, i18n.Params{"field": "start_time"})
	}
	e, err := time.Parse(timeLayout, end)
	if err != nil {
		return 0, i18n.Err(i18n.FieldInvalid, i18n.Params{"field": "end_time"})
	}
	return (e.Hour()*60 + e.Minute()) - (s.Hour()*60 + s.Minute()), nil
}

func toDayModels(days []int) []model.ScheduleActivityDay {
	models := make([]model.ScheduleActivityDay, len(days))
	for i, d := range days {
		models[i] = model.ScheduleActivityDay{Weekday: d}
	}
	return models
}

func toRecurrenceRes(activity *model.ScheduleActivity) dto.RecurrenceRes {
	res := dto.RecurrenceRes{Kind: activity.Recurrence}
	if activity.Recurrence == enums.RecurrenceKindEnum(enums.RecurrenceKindWeekly) {
		days := make([]int, 0, len(activity.Days))
		for _, d := range activity.Days {
			days = append(days, d.Weekday)
		}
		sort.Ints(days)
		res.Days = days
	}
	if activity.RecurrenceUntil != nil {
		until := activity.RecurrenceUntil.Format(dateLayout)
		res.Until = &until
	}
	return res
}

func toActivityRes(activity *model.ScheduleActivity) *dto.ScheduleActivityRes {
	exceptions := make([]string, 0, len(activity.Exceptions))
	for _, exc := range activity.Exceptions {
		exceptions = append(exceptions, exc.Date.Format(dateLayout))
	}
	sort.Strings(exceptions)

	return &dto.ScheduleActivityRes{
		ID:         activity.ID,
		Title:      activity.Title,
		Type:       activity.Type,
		Date:       activity.Date.Format(dateLayout),
		StartTime:  activity.StartTime,
		EndTime:    activity.EndTime,
		Location:   activity.Location,
		Note:       activity.Note,
		Recurrence: toRecurrenceRes(activity),
		Exceptions: exceptions,
		CreatedAt:  activity.CreatedAt,
		UpdatedAt:  activity.UpdatedAt,
	}
}
