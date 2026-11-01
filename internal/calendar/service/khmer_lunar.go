package service

import "time"

// Khmer lunar calendar (Chhankitek): the traditional calculation, working on whole days counted from 1970-01-01.

const (
	lunarMigasir = iota
	lunarBos
	lunarMeak
	lunarPhalkun
	lunarCheit
	lunarPisak
	lunarJesth
	lunarAsadh
	lunarSrap
	lunarPhotrobot
	lunarAssuj
	lunarKadeuk
	// The two Asadh months replace lunarAsadh in a leap-month year.
	lunarPathamasadh
	lunarTutiyasadh
)

type moonPhase int

const (
	moonWaxing moonPhase = iota
	moonWaning
)

type lunarDate struct {
	month int
	// 1–15 within the phase.
	day   int
	phase moonPhase
}

const secondsPerDay = 86400

// 1 January 1900 was 1 waxing of Bos.
var lunarEpochDay = dayNumberOf(time.Date(1900, 1, 1, 0, 0, 0, 0, time.UTC))

func dayNumberOf(date time.Time) int {
	return int(date.Unix() / secondsPerDay)
}

func dateOf(dayNumber int) time.Time {
	return time.Unix(int64(dayNumber)*secondsPerDay, 0).UTC()
}

func aharkunBase(be int) int { return be*292207 + 499 }
func aharkun(be int) int     { return aharkunBase(be)/800 + 4 }
func isSolarLeap(be int) bool {
	return 800-aharkunBase(be)%800 <= 207
}
func avoman(be int) int { return (11*aharkun(be) + 25) % 692 }
func bodithey(be int) int {
	a := aharkun(be)
	return ((11*a+25)/692 + a + 29) % 30
}

// boditheyLeap returns 0 for none, 1 for a bodithey (month) leap, 2 for an avoman (day) leap, 3 for both.
func boditheyLeap(be int) int {
	b := bodithey(be)
	a := avoman(be)
	monthLeap := b >= 25 || b <= 5
	// 25 then 5 in consecutive years: only the 5 is a leap-month year. 24 then 6: the 24 is.
	if b == 25 && bodithey(be+1) == 5 {
		monthLeap = false
	}
	if b == 24 && bodithey(be+1) == 6 {
		monthLeap = true
	}
	var dayLeap bool
	if isSolarLeap(be) {
		dayLeap = a <= 126
	} else {
		dayLeap = a <= 137 && avoman(be+1) != 0
	}
	switch {
	case monthLeap && dayLeap:
		return 3
	case monthLeap:
		return 1
	case dayLeap:
		return 2
	}
	return 0
}

// leapType is the leap applied to a Buddhist Era year: 0 none, 1 leap month, 2 leap day.
func leapType(be int) int {
	b := boditheyLeap(be)
	// A year cannot have both: the leap day moves to the next year.
	if b == 3 {
		return 1
	}
	if b != 0 {
		return b
	}
	if boditheyLeap(be-1) == 3 {
		return 2
	}
	return 0
}

// beYearOf is the BE year used for leap rules of a day: the Khmer year changes around April.
func beYearOf(dayNumber int) int {
	date := dateOf(dayNumber)
	if date.Month() <= time.April {
		return date.Year() + 543
	}
	return date.Year() + 544
}

func daysInLunarMonth(month, be int) int {
	if month == lunarJesth && leapType(be) == 2 {
		return 30
	}
	if month == lunarPathamasadh || month == lunarTutiyasadh {
		return 30
	}
	if month%2 == 0 {
		return 29
	}
	return 30
}

func daysInLunarYear(be int) int {
	switch leapType(be) {
	case 1:
		return 384
	case 2:
		return 355
	}
	return 354
}

func nextLunarMonth(month, be int) int {
	switch month {
	case lunarJesth:
		if leapType(be) == 1 {
			return lunarPathamasadh
		}
		return lunarAsadh
	case lunarPathamasadh:
		return lunarTutiyasadh
	case lunarTutiyasadh, lunarAsadh:
		return lunarSrap
	}
	return (month + 1) % 12
}

func oneYearLater(dayNumber int) int {
	return dayNumberOf(dateOf(dayNumber).AddDate(1, 0, 0))
}

// lunarDateOf converts days since 1970-01-01; dayNumber must not be before 1900-01-01.
func lunarDateOf(dayNumber int) lunarDate {
	start := lunarEpochDay
	month := lunarBos
	// Jump whole lunar years (each starts on 1 waxing of Bos), then whole months.
	for dayNumber-start > daysInLunarYear(beYearOf(oneYearLater(start))) {
		start += daysInLunarYear(beYearOf(oneYearLater(start)))
	}
	for dayNumber-start > daysInLunarMonth(month, beYearOf(start)) {
		start += daysInLunarMonth(month, beYearOf(start))
		month = nextLunarMonth(month, beYearOf(start))
	}
	dayIndex := dayNumber - start
	if length := daysInLunarMonth(month, beYearOf(dayNumber)); dayIndex >= length {
		dayIndex %= length
		month = nextLunarMonth(month, beYearOf(start))
	}

	if dayIndex < 15 {
		return lunarDate{month: month, day: dayIndex + 1, phase: moonWaxing}
	}
	return lunarDate{month: month, day: dayIndex - 14, phase: moonWaning}
}
