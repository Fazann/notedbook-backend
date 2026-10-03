package service

import (
	"fmt"
	"sort"
	"time"

	"github.com/Fazann/notedbook-backend/internal/schedule/dto"
	"github.com/Fazann/notedbook-backend/pkg/enums"
	model "github.com/Fazann/notedbook-backend/shared/models"
)

// expandOccurrences expands every activity's series into its concrete occurrences within [from, to]
// (inclusive, date-only), sorted by date then start time. It is the backend counterpart of the
// frontend mock's expandOccurrences and is kept pure (no DB) so it can be unit-tested on its own.
func expandOccurrences(activities []model.ScheduleActivity, from, to time.Time) []*dto.OccurrenceRes {
	occurrences := []*dto.OccurrenceRes{}
	for i := range activities {
		occurrences = append(occurrences, expandActivity(&activities[i], from, to)...)
	}
	sort.Slice(occurrences, func(i, j int) bool {
		if occurrences[i].Date != occurrences[j].Date {
			return occurrences[i].Date < occurrences[j].Date
		}
		return occurrences[i].StartTime < occurrences[j].StartTime
	})
	return occurrences
}

func expandActivity(activity *model.ScheduleActivity, from, to time.Time) []*dto.OccurrenceRes {
	exceptions := make(map[string]bool, len(activity.Exceptions))
	for _, exc := range activity.Exceptions {
		exceptions[exc.Date.Format(dateLayout)] = true
	}

	if activity.Recurrence == enums.RecurrenceKindEnum(enums.RecurrenceKindNone) {
		if activity.Date.Before(from) || activity.Date.After(to) {
			return nil
		}
		if exceptions[activity.Date.Format(dateLayout)] {
			return nil
		}
		return []*dto.OccurrenceRes{toOccurrenceRes(activity, activity.Date, false)}
	}

	days := make(map[int]bool, len(activity.Days))
	for _, d := range activity.Days {
		days[d.Weekday] = true
	}

	start := activity.Date
	if start.Before(from) {
		start = from
	}
	end := to
	if activity.RecurrenceUntil != nil && activity.RecurrenceUntil.Before(end) {
		end = *activity.RecurrenceUntil
	}

	occurrences := []*dto.OccurrenceRes{}
	for d := start; !d.After(end); d = d.AddDate(0, 0, 1) {
		if d.Before(activity.Date) || !matchesRecurrence(activity.Recurrence, days, d) {
			continue
		}
		if exceptions[d.Format(dateLayout)] {
			continue
		}
		occurrences = append(occurrences, toOccurrenceRes(activity, d, true))
	}
	return occurrences
}

func matchesRecurrence(kind enums.RecurrenceKindEnum, weeklyDays map[int]bool, d time.Time) bool {
	switch kind {
	case enums.RecurrenceKindEnum(enums.RecurrenceKindDaily):
		return true
	case enums.RecurrenceKindEnum(enums.RecurrenceKindWeekdays):
		wd := isoWeekday(d)
		return wd >= 1 && wd <= 5
	case enums.RecurrenceKindEnum(enums.RecurrenceKindWeekly):
		return weeklyDays[isoWeekday(d)]
	default:
		return false
	}
}

// isoWeekday returns 1 (Monday) through 7 (Sunday), unlike time.Weekday's 0 (Sunday) through 6.
func isoWeekday(d time.Time) int {
	wd := int(d.Weekday())
	if wd == 0 {
		return 7
	}
	return wd
}

func toOccurrenceRes(activity *model.ScheduleActivity, date time.Time, isRecurring bool) *dto.OccurrenceRes {
	dateStr := date.Format(dateLayout)
	return &dto.OccurrenceRes{
		Key:         fmt.Sprintf("%d:%s", activity.ID, dateStr),
		ActivityID:  activity.ID,
		Date:        dateStr,
		Title:       activity.Title,
		Type:        activity.Type,
		StartTime:   activity.StartTime,
		EndTime:     activity.EndTime,
		Location:    activity.Location,
		Note:        activity.Note,
		IsRecurring: isRecurring,
	}
}

// summarize totals the occurrences' durations, overall and per activity type.
func summarize(occurrences []*dto.OccurrenceRes) *dto.WeekSummaryRes {
	summary := &dto.WeekSummaryRes{ByType: map[string]int{}}
	for _, o := range occurrences {
		minutes, err := durationMinutes(o.StartTime, o.EndTime)
		if err != nil || minutes <= 0 {
			continue
		}
		summary.TotalMinutes += minutes
		summary.ByType[string(o.Type)] += minutes
		summary.Count++
	}
	return summary
}
