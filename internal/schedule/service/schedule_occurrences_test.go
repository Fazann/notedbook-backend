package service

import (
	"testing"
	"time"

	"github.com/Fazann/notedbook-backend/internal/schedule/dto"
	"github.com/Fazann/notedbook-backend/pkg/enums"
	model "github.com/Fazann/notedbook-backend/shared/models"
)

func mustDate(t *testing.T, value string) time.Time {
	t.Helper()
	d, err := time.Parse(dateLayout, value)
	if err != nil {
		t.Fatalf("parse %q: %v", value, err)
	}
	return d
}

// Week of 2026-09-28 (Mon) .. 2026-10-04 (Sun).
func weekRange(t *testing.T) (time.Time, time.Time) {
	return mustDate(t, "2026-09-28"), mustDate(t, "2026-10-04")
}

func TestExpandOccurrences_None(t *testing.T) {
	from, to := weekRange(t)
	activities := []model.ScheduleActivity{
		{BaseModelID: model.BaseModelID{ID: 1}, Title: "Dentist", Date: mustDate(t, "2026-09-30"),
			StartTime: "14:00", EndTime: "14:30", Recurrence: enums.RecurrenceKindEnum(enums.RecurrenceKindNone)},
		{BaseModelID: model.BaseModelID{ID: 2}, Title: "Out of range", Date: mustDate(t, "2026-10-10"),
			StartTime: "09:00", EndTime: "10:00", Recurrence: enums.RecurrenceKindEnum(enums.RecurrenceKindNone)},
	}

	got := expandOccurrences(activities, from, to)

	if len(got) != 1 || got[0].ActivityID != 1 || got[0].Date != "2026-09-30" || got[0].IsRecurring {
		t.Fatalf("got = %+v", got)
	}
	if got[0].Key != "1:2026-09-30" {
		t.Errorf("key = %q", got[0].Key)
	}
}

func TestExpandOccurrences_Daily(t *testing.T) {
	from, to := weekRange(t)
	activities := []model.ScheduleActivity{
		{BaseModelID: model.BaseModelID{ID: 1}, Title: "Run", Date: mustDate(t, "2026-09-28"),
			StartTime: "06:00", EndTime: "06:45", Recurrence: enums.RecurrenceKindEnum(enums.RecurrenceKindDaily)},
	}

	got := expandOccurrences(activities, from, to)

	if len(got) != 7 {
		t.Fatalf("got %d occurrences, want 7: %+v", len(got), got)
	}
	if got[0].Date != "2026-09-28" || got[6].Date != "2026-10-04" || !got[0].IsRecurring {
		t.Errorf("first/last = %+v / %+v", got[0], got[6])
	}
}

func TestExpandOccurrences_Weekdays(t *testing.T) {
	from, to := weekRange(t)
	activities := []model.ScheduleActivity{
		{BaseModelID: model.BaseModelID{ID: 1}, Title: "Office", Date: mustDate(t, "2026-09-01"),
			StartTime: "08:00", EndTime: "17:00", Recurrence: enums.RecurrenceKindEnum(enums.RecurrenceKindWeekdays)},
	}

	got := expandOccurrences(activities, from, to)

	if len(got) != 5 {
		t.Fatalf("got %d occurrences, want 5 (Mon-Fri): %+v", len(got), got)
	}
	if got[0].Date != "2026-09-28" || got[4].Date != "2026-10-02" {
		t.Errorf("first/last = %+v / %+v", got[0], got[4])
	}
}

func TestExpandOccurrences_WeeklyMultiDay(t *testing.T) {
	from, to := weekRange(t)
	activities := []model.ScheduleActivity{
		{BaseModelID: model.BaseModelID{ID: 1}, Title: "English class", Date: mustDate(t, "2026-01-06"),
			StartTime: "18:30", EndTime: "20:00", Recurrence: enums.RecurrenceKindEnum(enums.RecurrenceKindWeekly),
			Days: []model.ScheduleActivityDay{{Weekday: 2}, {Weekday: 4}}}, // Tue, Thu
	}

	got := expandOccurrences(activities, from, to)

	if len(got) != 2 || got[0].Date != "2026-09-29" || got[1].Date != "2026-10-01" {
		t.Fatalf("got = %+v", got)
	}
}

func TestExpandOccurrences_Until(t *testing.T) {
	from, to := weekRange(t)
	activities := []model.ScheduleActivity{
		{BaseModelID: model.BaseModelID{ID: 1}, Title: "Gym", Date: mustDate(t, "2026-09-01"),
			StartTime: "16:00", EndTime: "17:00", Recurrence: enums.RecurrenceKindEnum(enums.RecurrenceKindWeekly),
			Days:            []model.ScheduleActivityDay{{Weekday: 6}}, // Sat
			RecurrenceUntil: timePtr(mustDate(t, "2026-10-02"))},       // ends before this week's Saturday
	}

	got := expandOccurrences(activities, from, to)

	if len(got) != 0 {
		t.Fatalf("got = %+v, want none (series ended before this week's Saturday)", got)
	}
}

func TestExpandOccurrences_StartsAfterFrom(t *testing.T) {
	from, to := weekRange(t)
	activities := []model.ScheduleActivity{
		{BaseModelID: model.BaseModelID{ID: 1}, Title: "New habit", Date: mustDate(t, "2026-10-01"),
			StartTime: "07:00", EndTime: "07:30", Recurrence: enums.RecurrenceKindEnum(enums.RecurrenceKindDaily)},
	}

	got := expandOccurrences(activities, from, to)

	if len(got) != 4 || got[0].Date != "2026-10-01" || got[3].Date != "2026-10-04" {
		t.Fatalf("got = %+v, want 2026-10-01..2026-10-04", got)
	}
}

func TestExpandOccurrences_Exception(t *testing.T) {
	from, to := weekRange(t)
	activities := []model.ScheduleActivity{
		{BaseModelID: model.BaseModelID{ID: 1}, Title: "English class", Date: mustDate(t, "2026-01-06"),
			StartTime: "18:30", EndTime: "20:00", Recurrence: enums.RecurrenceKindEnum(enums.RecurrenceKindWeekly),
			Days:       []model.ScheduleActivityDay{{Weekday: 4}}, // Thu
			Exceptions: []model.ScheduleActivityException{{Date: mustDate(t, "2026-10-01")}}},
	}

	got := expandOccurrences(activities, from, to)

	if len(got) != 0 {
		t.Fatalf("got = %+v, want none (the only Thursday this week is an exception)", got)
	}
}

func TestExpandOccurrences_SortedAcrossActivities(t *testing.T) {
	from, to := weekRange(t)
	activities := []model.ScheduleActivity{
		{BaseModelID: model.BaseModelID{ID: 1}, Title: "Later", Date: mustDate(t, "2026-09-30"),
			StartTime: "18:00", EndTime: "19:00", Recurrence: enums.RecurrenceKindEnum(enums.RecurrenceKindNone)},
		{BaseModelID: model.BaseModelID{ID: 2}, Title: "Earlier same day", Date: mustDate(t, "2026-09-30"),
			StartTime: "08:00", EndTime: "09:00", Recurrence: enums.RecurrenceKindEnum(enums.RecurrenceKindNone)},
		{BaseModelID: model.BaseModelID{ID: 3}, Title: "Day before", Date: mustDate(t, "2026-09-29"),
			StartTime: "23:00", EndTime: "23:30", Recurrence: enums.RecurrenceKindEnum(enums.RecurrenceKindNone)},
	}

	got := expandOccurrences(activities, from, to)

	if len(got) != 3 || got[0].ActivityID != 3 || got[1].ActivityID != 2 || got[2].ActivityID != 1 {
		t.Fatalf("order = %+v", got)
	}
}

func TestSummarize(t *testing.T) {
	occurrences := []*dto.OccurrenceRes{
		{Type: enums.ScheduleActivityTypeEnum(enums.ScheduleActivityTypeWork), StartTime: "08:00", EndTime: "12:00"},
		{Type: enums.ScheduleActivityTypeEnum(enums.ScheduleActivityTypeWork), StartTime: "13:30", EndTime: "17:30"},
		{Type: enums.ScheduleActivityTypeEnum(enums.ScheduleActivityTypeLearning), StartTime: "18:30", EndTime: "20:00"},
	}

	got := summarize(occurrences)

	if got.Count != 3 || got.TotalMinutes != 240+240+90 {
		t.Fatalf("summary = %+v", got)
	}
	if got.ByType["WORK"] != 480 || got.ByType["LEARNING"] != 90 {
		t.Errorf("by_type = %+v", got.ByType)
	}
}

func timePtr(t time.Time) *time.Time { return &t }
