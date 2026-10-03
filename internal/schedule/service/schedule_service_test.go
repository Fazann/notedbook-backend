package service

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/Fazann/notedbook-backend/internal/schedule/dto"
	"github.com/Fazann/notedbook-backend/internal/schedule/repo"
	"github.com/Fazann/notedbook-backend/pkg/enums"
	"github.com/Fazann/notedbook-backend/pkg/i18n"
	model "github.com/Fazann/notedbook-backend/shared/models"
	"gorm.io/gorm"
)

// fakeScheduleActivityRepo keeps activities in memory; methods the service does not call panic via
// the nil embed.
type fakeScheduleActivityRepo struct {
	repo.ScheduleActivityRepoImpl
	activities map[uint]*model.ScheduleActivity
	nextID     uint
	created    []*model.ScheduleActivity
	exceptions []struct {
		activityID uint
		date       time.Time
	}
	deleted []uint
	days    map[uint][]model.ScheduleActivityDay
}

func newFakeScheduleActivityRepo() *fakeScheduleActivityRepo {
	return &fakeScheduleActivityRepo{
		nextID: 10,
		days:   map[uint][]model.ScheduleActivityDay{},
		activities: map[uint]*model.ScheduleActivity{
			1: {BaseModelID: model.BaseModelID{ID: 1}, UserID: 1, Title: "Team meeting",
				Type: enums.ScheduleActivityTypeEnum(enums.ScheduleActivityTypeMeeting),
				Date: mustParseDate("2026-09-28"), StartTime: "10:10", EndTime: "10:40",
				Recurrence: enums.RecurrenceKindEnum(enums.RecurrenceKindWeekdays)},
			2: {BaseModelID: model.BaseModelID{ID: 2}, UserID: 2, Title: "Other user's activity",
				Type: enums.ScheduleActivityTypeEnum(enums.ScheduleActivityTypeWork),
				Date: mustParseDate("2026-09-28"), StartTime: "08:00", EndTime: "09:00",
				Recurrence: enums.RecurrenceKindEnum(enums.RecurrenceKindNone)},
		},
	}
}

func mustParseDate(v string) time.Time {
	d, err := time.Parse(dateLayout, v)
	if err != nil {
		panic(err)
	}
	return d
}

func (r *fakeScheduleActivityRepo) WithTx(_ *gorm.DB) repo.ScheduleActivityRepoImpl { return r }

func (r *fakeScheduleActivityRepo) FindOneByID(_ context.Context, userID, id uint) (*model.ScheduleActivity, error) {
	if a, ok := r.activities[id]; ok && a.UserID == userID {
		return a, nil
	}
	return nil, gorm.ErrRecordNotFound
}

func (r *fakeScheduleActivityRepo) Create(_ context.Context, a *model.ScheduleActivity) (*model.ScheduleActivity, error) {
	a.ID = r.nextID
	r.nextID++
	r.created = append(r.created, a)
	r.activities[a.ID] = a
	return a, nil
}

func (r *fakeScheduleActivityRepo) Update(_ context.Context, condition any, updates map[string]any) error {
	cond, _ := condition.(map[string]any)
	id, _ := cond["id"].(uint)
	a, ok := r.activities[id]
	if !ok {
		return nil
	}
	for k, v := range updates {
		switch k {
		case "title":
			a.Title = v.(string)
		case "type":
			a.Type = v.(enums.ScheduleActivityTypeEnum)
		case "date":
			a.Date = v.(time.Time)
		case "start_time":
			a.StartTime = v.(string)
		case "end_time":
			a.EndTime = v.(string)
		case "location":
			a.Location = v.(string)
		case "note":
			a.Note = v.(string)
		case "recurrence":
			a.Recurrence = v.(enums.RecurrenceKindEnum)
		case "recurrence_until":
			if v == nil {
				a.RecurrenceUntil = nil
			} else {
				a.RecurrenceUntil = v.(*time.Time)
			}
		}
	}
	return nil
}

func (r *fakeScheduleActivityRepo) ReplaceDays(_ context.Context, activityID uint, days []model.ScheduleActivityDay) error {
	r.days[activityID] = days
	if a, ok := r.activities[activityID]; ok {
		a.Days = days
	}
	return nil
}

func (r *fakeScheduleActivityRepo) AddException(_ context.Context, activityID uint, date time.Time) error {
	r.exceptions = append(r.exceptions, struct {
		activityID uint
		date       time.Time
	}{activityID, date})
	if a, ok := r.activities[activityID]; ok {
		a.Exceptions = append(a.Exceptions, model.ScheduleActivityException{ActivityID: activityID, Date: date})
	}
	return nil
}

func (r *fakeScheduleActivityRepo) SoftDelete(_ context.Context, userID, id uint) error {
	a, ok := r.activities[id]
	if !ok || a.UserID != userID {
		return gorm.ErrRecordNotFound
	}
	r.deleted = append(r.deleted, id)
	delete(r.activities, id)
	return nil
}

func newTestScheduleSvc(r *fakeScheduleActivityRepo) *scheduleSvc {
	return &scheduleSvc{
		activityRepo: r,
		transaction:  func(_ context.Context, fn func(tx *gorm.DB) error) error { return fn(nil) },
	}
}

func validActivityReq() dto.ScheduleActivityReq {
	return dto.ScheduleActivityReq{
		Title: " Team meeting ", Type: enums.ScheduleActivityTypeEnum(enums.ScheduleActivityTypeMeeting),
		Date: "2026-09-30", StartTime: "10:10", EndTime: "10:40",
		Recurrence: dto.RecurrenceReq{Kind: enums.RecurrenceKindEnum(enums.RecurrenceKindNone)},
	}
}

func TestCreateActivity(t *testing.T) {
	tests := []struct {
		name       string
		edit       func(*dto.ScheduleActivityReq)
		wantStatus int
		wantErr    error
	}{
		{name: "created", wantStatus: http.StatusCreated},
		{name: "end before start", edit: func(r *dto.ScheduleActivityReq) { r.EndTime = "09:00" },
			wantStatus: http.StatusBadRequest, wantErr: i18n.InvalidTimeRange},
		{name: "too short", edit: func(r *dto.ScheduleActivityReq) { r.EndTime = "10:12" },
			wantStatus: http.StatusBadRequest, wantErr: i18n.ActivityTooShort},
		{name: "weekly without days", edit: func(r *dto.ScheduleActivityReq) {
			r.Recurrence = dto.RecurrenceReq{Kind: enums.RecurrenceKindEnum(enums.RecurrenceKindWeekly)}
		}, wantStatus: http.StatusBadRequest, wantErr: i18n.RecurrenceDaysRequired},
		{name: "until before date", edit: func(r *dto.ScheduleActivityReq) {
			until := "2026-09-01"
			r.Recurrence = dto.RecurrenceReq{Kind: enums.RecurrenceKindEnum(enums.RecurrenceKindDaily), Until: &until}
		}, wantStatus: http.StatusBadRequest, wantErr: i18n.InvalidRecurrenceUntil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := newFakeScheduleActivityRepo()
			req := validActivityReq()
			if tt.edit != nil {
				tt.edit(&req)
			}

			res := newTestScheduleSvc(r).CreateActivity(context.Background(), 1, &req)

			if res.StatusCode != tt.wantStatus {
				t.Fatalf("status = %d, want %d (err %v)", res.StatusCode, tt.wantStatus, res.Error)
			}
			if tt.wantErr != nil {
				if key, _, _ := i18n.FromError(res.Error); key != tt.wantErr {
					t.Fatalf("err = %v, want %v", res.Error, tt.wantErr)
				}
				if len(r.created) != 0 {
					t.Fatal("activity created on error")
				}
				return
			}
			if len(r.created) != 1 || r.created[0].Title != "Team meeting" || r.created[0].UserID != 1 {
				t.Errorf("created = %+v", r.created)
			}
		})
	}
}

func TestCreateActivity_WeeklyDaysDedupedAndSorted(t *testing.T) {
	r := newFakeScheduleActivityRepo()
	req := validActivityReq()
	req.Recurrence = dto.RecurrenceReq{Kind: enums.RecurrenceKindEnum(enums.RecurrenceKindWeekly), Days: []int{4, 2, 4}}

	res := newTestScheduleSvc(r).CreateActivity(context.Background(), 1, &req)

	if res.Error != nil || res.StatusCode != http.StatusCreated {
		t.Fatalf("status = %d, err = %v", res.StatusCode, res.Error)
	}
	got := res.Data.(*dto.ScheduleActivityRes)
	if len(got.Recurrence.Days) != 2 || got.Recurrence.Days[0] != 2 || got.Recurrence.Days[1] != 4 {
		t.Errorf("days = %v, want [2 4]", got.Recurrence.Days)
	}
}

func TestGetActivity_NotOwner(t *testing.T) {
	r := newFakeScheduleActivityRepo()

	res := newTestScheduleSvc(r).GetActivity(context.Background(), 1, 2)

	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", res.StatusCode)
	}
}

func TestUpdateActivity_ScopeAll(t *testing.T) {
	r := newFakeScheduleActivityRepo()
	req := validActivityReq()
	req.Title = "Renamed meeting"

	res := newTestScheduleSvc(r).UpdateActivity(context.Background(), 1, 1, &dto.ScheduleActivityScopeReq{Scope: "all"}, &req)

	if res.Error != nil || res.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, err = %v", res.StatusCode, res.Error)
	}
	if r.activities[1].Title != "Renamed meeting" {
		t.Errorf("title = %q", r.activities[1].Title)
	}
	if r.activities[1].Recurrence != enums.RecurrenceKindEnum(enums.RecurrenceKindNone) {
		t.Errorf("recurrence = %v, want NONE (replaced by req)", r.activities[1].Recurrence)
	}
}

// TestUpdateActivity_ScopeThis covers the "edit this occurrence only" rule: the series gets an
// exception for that date, and a brand-new standalone activity is created with the new values.
func TestUpdateActivity_ScopeThis(t *testing.T) {
	r := newFakeScheduleActivityRepo()
	req := validActivityReq()
	req.Title = "Team meeting (moved room)"
	req.Date = "2026-09-30"

	res := newTestScheduleSvc(r).UpdateActivity(context.Background(), 1, 1,
		&dto.ScheduleActivityScopeReq{Scope: "this", Date: "2026-09-30"}, &req)

	if res.Error != nil || res.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, err = %v", res.StatusCode, res.Error)
	}
	if len(r.exceptions) != 1 || r.exceptions[0].activityID != 1 {
		t.Fatalf("exceptions = %+v", r.exceptions)
	}
	if len(r.created) != 1 || r.created[0].Title != "Team meeting (moved room)" ||
		r.created[0].Recurrence != enums.RecurrenceKindEnum(enums.RecurrenceKindNone) {
		t.Fatalf("created = %+v", r.created)
	}
	got := res.Data.(*dto.ScheduleActivityUpdateRes)
	if got.Series.ID != 1 || got.Created.Title != "Team meeting (moved room)" {
		t.Errorf("response = %+v", got)
	}
	// The original series itself must be untouched.
	if r.activities[1].Title != "Team meeting" {
		t.Errorf("series title = %q, want unchanged", r.activities[1].Title)
	}
}

func TestUpdateActivity_ScopeThisRequiresDate(t *testing.T) {
	r := newFakeScheduleActivityRepo()
	req := validActivityReq()

	res := newTestScheduleSvc(r).UpdateActivity(context.Background(), 1, 1, &dto.ScheduleActivityScopeReq{Scope: "this"}, &req)

	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", res.StatusCode)
	}
	if key, _, _ := i18n.FromError(res.Error); key != i18n.ScopeDateRequired {
		t.Errorf("err = %v, want ScopeDateRequired", res.Error)
	}
}

func TestDeleteActivity_ScopeAll(t *testing.T) {
	r := newFakeScheduleActivityRepo()

	res := newTestScheduleSvc(r).DeleteActivity(context.Background(), 1, 1, &dto.ScheduleActivityScopeReq{Scope: "all"})

	if res.Error != nil || res.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, err = %v", res.StatusCode, res.Error)
	}
	if len(r.deleted) != 1 || r.deleted[0] != 1 {
		t.Errorf("deleted = %v", r.deleted)
	}
	if _, exists := r.activities[1]; exists {
		t.Error("activity still present after scope=all delete")
	}
}

// TestDeleteActivity_ScopeThis covers "delete this occurrence only": the series survives, only an
// exception is added, and no new activity is created.
func TestDeleteActivity_ScopeThis(t *testing.T) {
	r := newFakeScheduleActivityRepo()

	res := newTestScheduleSvc(r).DeleteActivity(context.Background(), 1, 1,
		&dto.ScheduleActivityScopeReq{Scope: "this", Date: "2026-09-30"})

	if res.Error != nil || res.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, err = %v", res.StatusCode, res.Error)
	}
	if len(r.deleted) != 0 {
		t.Error("series deleted on scope=this delete")
	}
	if len(r.exceptions) != 1 || r.exceptions[0].activityID != 1 {
		t.Fatalf("exceptions = %+v", r.exceptions)
	}
	if len(r.created) != 0 {
		t.Error("a new activity should not be created on delete")
	}
	if _, exists := r.activities[1]; !exists {
		t.Error("series removed, want it to survive a scope=this delete")
	}
}

func TestDeleteActivity_AnotherUsersActivity(t *testing.T) {
	r := newFakeScheduleActivityRepo()

	res := newTestScheduleSvc(r).DeleteActivity(context.Background(), 1, 2, &dto.ScheduleActivityScopeReq{Scope: "all"})

	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", res.StatusCode)
	}
}

func TestParseRange(t *testing.T) {
	tests := []struct {
		name       string
		from, to   string
		wantStatus string // "" = ok
	}{
		{name: "one week", from: "2026-09-28", to: "2026-10-04"},
		{name: "single day", from: "2026-09-28", to: "2026-09-28"},
		{name: "42 days (max)", from: "2026-01-01", to: "2026-02-11"},
		{name: "43 days (too many)", from: "2026-01-01", to: "2026-02-12", wantStatus: "range"},
		{name: "reversed", from: "2026-10-04", to: "2026-09-28", wantStatus: "range"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := parseRange(&dto.OccurrenceListReq{From: tt.from, To: tt.to})
			if tt.wantStatus == "" && err != nil {
				t.Errorf("unexpected err: %v", err)
			}
			if tt.wantStatus != "" {
				if key, _, _ := i18n.FromError(err); key != i18n.InvalidDateRange {
					t.Errorf("err = %v, want InvalidDateRange", err)
				}
			}
		})
	}
}
