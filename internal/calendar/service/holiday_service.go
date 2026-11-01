package service

import (
	"context"
	"net/http"
	"time"

	"github.com/Fazann/notedbook-backend/internal/calendar/dto"
	"github.com/Fazann/notedbook-backend/pkg/response"
)

type HolidayServiceImpl interface {
	List(ctx context.Context, req *dto.HolidayListReq) response.SvcRes
}

type holidaySvc struct{}

func NewHolidaySvc() *holidaySvc {
	return &holidaySvc{}
}

func (s *holidaySvc) List(_ context.Context, req *dto.HolidayListReq) response.SvcRes {
	return response.SvcRespSuccess(holidaysOf(req.Year, rulesFor(req.Country, req.Kind)), http.StatusOK)
}

// holidaysOf walks the year day by day; the rules of one day stay in rule order.
func holidaysOf(year int, rules []holidayRule) []dto.HolidayRes {
	res := []dto.HolidayRes{}
	for date := time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC); date.Year() == year; date = date.AddDate(0, 0, 1) {
		dayNumber := dayNumberOf(date)
		lunar := lunarDateOf(dayNumber)
		hijri := hijriOf(dayNumber)
		monthDay := date.Format("01-02")
		for i := range rules {
			if rules[i].matches(monthDay, lunar, hijri) {
				res = append(res, dto.HolidayRes{Date: date.Format("2006-01-02"), Key: rules[i].key, Name: rules[i].name})
			}
		}
	}
	return res
}

func (r *holidayRule) matches(monthDay string, lunar lunarDate, hijri hijriDate) bool {
	switch {
	case r.lunar != nil:
		return *r.lunar == lunar
	case r.hijri != nil:
		return r.hijri.month == hijri.month && r.hijri.day == hijri.day
	}
	return r.solar == monthDay
}
