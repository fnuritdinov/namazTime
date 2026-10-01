package prayertime

// IshaRule — как определяется время Иши.
type IshaRule struct {
	Type  string  // "angle" — по углу солнца под горизонтом; "minutesAfterMaghrib" — N минут после Магриба
	Value float64 // градусы или минуты
}

// Method — метод расчёта времени намазов.
type Method struct {
	ID        string            // совпадает с CalculationMethod в iOS
	Title     map[string]string // язык → название
	FajrAngle float64           // угол солнца под горизонтом для Фаджра, градусы
	Isha      IshaRule
}

// Methods — методы из ТЗ (§6). Идентификаторы СТРОГО совпадают с iOS.
var Methods = []Method{
	{
		ID: "muslimWorldLeague",
		Title: map[string]string{
			"en": "Muslim World League",
			"ru": "Всемирная исламская лига",
			"ar": "رابطة العالم الإسلامي",
		},
		FajrAngle: 18,
		Isha:      IshaRule{Type: "angle", Value: 17},
	},
	{
		ID: "northAmerica",
		Title: map[string]string{
			"en": "ISNA (North America)",
			"ru": "ISNA (Северная Америка)",
		},
		FajrAngle: 15,
		Isha:      IshaRule{Type: "angle", Value: 15},
	},
	{
		ID: "egypt",
		Title: map[string]string{
			"en": "Egyptian General Authority",
			"ru": "Египетское управление геодезии",
		},
		FajrAngle: 19.5,
		Isha:      IshaRule{Type: "angle", Value: 17.5},
	},
	{
		ID: "ummAlQura",
		Title: map[string]string{
			"en": "Umm al-Qura, Makkah",
			"ru": "Умм аль-Кура, Мекка",
			"ar": "أم القرى",
		},
		FajrAngle: 18.5,
		Isha:      IshaRule{Type: "minutesAfterMaghrib", Value: 90},
	},
	{
		ID: "karachi",
		Title: map[string]string{
			"en": "University of Karachi",
			"ru": "Университет Карачи",
		},
		FajrAngle: 18,
		Isha:      IshaRule{Type: "angle", Value: 18},
	},
	{
		ID: "turkey",
		Title: map[string]string{
			"en": "Diyanet, Türkiye",
			"ru": "Диянет, Турция",
		},
		FajrAngle: 18,
		Isha:      IshaRule{Type: "angle", Value: 17},
	},
}

// MethodByID ищет метод по идентификатору.
func MethodByID(id string) (Method, bool) {
	for _, m := range Methods {
		if m.ID == id {
			return m, true
		}
	}
	return Method{}, false
}
