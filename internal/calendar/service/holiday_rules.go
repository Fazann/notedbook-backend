package service

// holidayRule finds a holiday by exactly one of: its fixed date, its Khmer lunar date or its Hijri date.
type holidayRule struct {
	key, name string
	// "MM-DD"
	solar string
	lunar *lunarDate
	// month and day; the year is not used.
	hijri *hijriDate
}

func solarRule(key, name, monthDay string) holidayRule {
	return holidayRule{key: key, name: name, solar: monthDay}
}

func lunarRule(key, name string, month, day int, phase moonPhase) holidayRule {
	return holidayRule{key: key, name: name, lunar: &lunarDate{month: month, day: day, phase: phase}}
}

func hijriRule(key, name string, month, day int) holidayRule {
	return holidayRule{key: key, name: name, hijri: &hijriDate{month: month, day: day}}
}

var cambodiaRules = []holidayRule{
	solarRule("newYear", "International New Year's Day", "01-01"),
	solarRule("victoryDay", "Victory over Genocide Day", "01-07"),
	solarRule("womensDay", "International Women's Day", "03-08"),
	solarRule("khmerNewYear", "Khmer New Year", "04-14"),
	solarRule("khmerNewYear", "Khmer New Year", "04-15"),
	solarRule("khmerNewYear", "Khmer New Year", "04-16"),
	solarRule("labourDay", "International Labour Day", "05-01"),
	solarRule("kingBirthday", "King's Birthday", "05-14"),
	lunarRule("visakBochea", "Visak Bochea Day", lunarPisak, 15, moonWaxing),
	lunarRule("royalPloughing", "Royal Ploughing Ceremony", lunarPisak, 4, moonWaning),
	solarRule("queenMotherBirthday", "Queen Mother's Birthday", "06-18"),
	lunarRule("pchumBen", "Pchum Ben", lunarPhotrobot, 14, moonWaning),
	lunarRule("pchumBen", "Pchum Ben", lunarPhotrobot, 15, moonWaning),
	lunarRule("pchumBen", "Pchum Ben", lunarAssuj, 1, moonWaxing),
	solarRule("constitutionDay", "Constitution Day", "09-24"),
	solarRule("kingFatherCommemoration", "Commemoration Day of King Father", "10-15"),
	solarRule("coronationDay", "Coronation Day", "10-29"),
	solarRule("independenceDay", "Independence Day", "11-09"),
	lunarRule("waterFestival", "Water Festival", lunarKadeuk, 14, moonWaxing),
	lunarRule("waterFestival", "Water Festival", lunarKadeuk, 15, moonWaxing),
	lunarRule("waterFestival", "Water Festival", lunarKadeuk, 1, moonWaning),
}

var islamicCommonRules = []holidayRule{
	hijriRule("islamicNewYear", "Islamic New Year", 1, 1),
	hijriRule("mawlid", "Prophet Muhammad's Birthday", 3, 12),
	hijriRule("eidAlFitr", "Eid al-Fitr", 10, 1),
	hijriRule("eidAlAdha", "Eid al-Adha", 12, 10),
}

var islamicMalaysiaRules = append(append([]holidayRule{}, islamicCommonRules...),
	hijriRule("israMiraj", "Isra' and Mi'raj", 7, 27),
	hijriRule("ramadanStart", "Beginning of Ramadan", 9, 1),
	hijriRule("nuzulQuran", "Nuzul Al-Quran", 9, 17),
	hijriRule("eidAlFitr", "Eid al-Fitr", 10, 2),
)

// rulesFor returns the rules of a country and kind. Malaysia's national holidays are not covered yet.
func rulesFor(country, kind string) []holidayRule {
	switch {
	case kind == "islamic" && country == "MY":
		return islamicMalaysiaRules
	case kind == "islamic":
		return islamicCommonRules
	case country == "KH":
		return cambodiaRules
	}
	return nil
}
