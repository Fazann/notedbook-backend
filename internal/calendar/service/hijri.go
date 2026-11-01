package service

type hijriDate struct {
	year, month, day int
}

// hijriOf converts days since 1970-01-01. The table covers every date from 1900 to 2100.
func hijriOf(dayNumber int) hijriDate {
	remaining := dayNumber - hijriFirstDay
	for i, mask := range hijriYears {
		for month := range 12 {
			length := 29 + int(mask>>month&1)
			if remaining < length {
				return hijriDate{year: hijriFirstYear + i, month: month + 1, day: remaining + 1}
			}
			remaining -= length
		}
	}
	return hijriDate{}
}
