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
	AladhanID int // номер метода в api.aladhan.com (параметр method)
}

// Methods — методы из ТЗ (§6). Идентификаторы СТРОГО совпадают с iOS.
var Methods = []Method{
	{
		ID: "muslimWorldLeague",
		Title: map[string]string{
			"en": "Muslim World League",
			"ru": "Всемирная исламская лига",
			"ar": "رابطة العالم الإسلامي",
			"tg": "Робитаи ҷаҳони ислом",
		},
		FajrAngle: 18,
		Isha:      IshaRule{Type: "angle", Value: 17},
		AladhanID: 3,
	},
	{
		ID: "northAmerica",
		Title: map[string]string{
			"en": "ISNA (North America)",
			"ru": "ISNA (Северная Америка)",
			"tg": "ISNA (Амрикои Шимолӣ)",
		},
		FajrAngle: 15,
		Isha:      IshaRule{Type: "angle", Value: 15},
		AladhanID: 2,
	},
	{
		ID: "egypt",
		Title: map[string]string{
			"en": "Egyptian General Authority",
			"ru": "Египетское управление геодезии",
			"tg": "Идораи генералии геодезии Миср",
		},
		FajrAngle: 19.5,
		Isha:      IshaRule{Type: "angle", Value: 17.5},
		AladhanID: 5,
	},
	{
		ID: "ummAlQura",
		Title: map[string]string{
			"en": "Umm al-Qura, Makkah",
			"ru": "Умм аль-Кура, Мекка",
			"ar": "أم القرى",
			"tg": "Умм-ул-Қуро, Макка",
		},
		FajrAngle: 18.5,
		Isha:      IshaRule{Type: "minutesAfterMaghrib", Value: 90},
		AladhanID: 4,
	},
	{
		ID: "karachi",
		Title: map[string]string{
			"en": "University of Karachi",
			"ru": "Университет Карачи",
			"tg": "Донишгоҳи Карачӣ",
		},
		FajrAngle: 18,
		Isha:      IshaRule{Type: "angle", Value: 18},
		AladhanID: 1,
	},
	{
		ID: "turkey",
		Title: map[string]string{
			"en": "Diyanet, Türkiye",
			"ru": "Диянет, Турция",
			"tg": "Диёнат, Туркия",
		},
		FajrAngle: 18,
		Isha:      IshaRule{Type: "angle", Value: 17},
		AladhanID: 13,
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
