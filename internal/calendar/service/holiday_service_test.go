package service

import (
	"context"
	"net/http"
	"slices"
	"testing"

	"github.com/Fazann/notedbook-backend/internal/calendar/dto"
)

func datesOf(t *testing.T, key string, year int, country, kind string) []string {
	t.Helper()
	res := NewHolidaySvc().List(context.Background(), &dto.HolidayListReq{Year: year, Country: country, Kind: kind})
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", res.StatusCode)
	}
	dates := []string{}
	for _, h := range res.Data.([]dto.HolidayRes) {
		if h.Key == key {
			dates = append(dates, h.Date)
		}
	}
	return dates
}

func TestHolidays(t *testing.T) {
	tests := []struct {
		name    string
		key     string
		year    int
		country string
		kind    string
		want    []string
	}{
		{"fixed date", "independenceDay", 2026, "KH", "national", []string{"2026-11-09"}},
		{"several fixed days", "khmerNewYear", 2026, "KH", "national", []string{"2026-04-14", "2026-04-15", "2026-04-16"}},
		{"lunar", "pchumBen", 2026, "KH", "national", []string{"2026-10-10", "2026-10-11", "2026-10-12"}},
		{"lunar, another year", "pchumBen", 2025, "KH", "national", []string{"2025-09-21", "2025-09-22", "2025-09-23"}},
		{"hijri", "eidAlFitr", 2026, "KH", "islamic", []string{"2026-03-20"}},
		{"hijri, two days in Malaysia", "eidAlFitr", 2026, "MY", "islamic", []string{"2026-03-20", "2026-03-21"}},
		{"Malaysia-only rule is left out for Cambodia", "nuzulQuran", 2026, "KH", "islamic", []string{}},
		{"Malaysia has no national holidays yet", "newYear", 2026, "MY", "national", []string{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := datesOf(t, tt.key, tt.year, tt.country, tt.kind); !slices.Equal(got, tt.want) {
				t.Errorf("dates = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHolidaysAreSortedByDate(t *testing.T) {
	res := holidaysOf(2026, cambodiaRules)
	if !slices.IsSortedFunc(res, func(a, b dto.HolidayRes) int {
		if a.Date < b.Date {
			return -1
		}
		if a.Date > b.Date {
			return 1
		}
		return 0
	}) {
		t.Error("holidays are not sorted by date")
	}
}

func TestSupportedYearsAreCovered(t *testing.T) {
	for _, year := range []int{1900, 2100} {
		for _, rules := range [][]holidayRule{cambodiaRules, islamicMalaysiaRules} {
			if len(holidaysOf(year, rules)) == 0 {
				t.Errorf("no holidays in %d", year)
			}
		}
	}
}
