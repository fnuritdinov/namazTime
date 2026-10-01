package city

// Name — название города на одном языке.
type Name struct {
	Name   string
	Region string
}

type City struct {
	ID                   string
	Country              string
	Lat, Lon             float64
	TimeZone             string
	SuggestedMethod      string
	SuggestedMadhab      string
	HasOfficialTimetable bool
	Population           *int            // nil, если неизвестно
	Names                map[string]Name // ключ — язык: "tg", "ru", "en", "ar"
}

// SearchParams — параметры поиска городов.
type SearchParams struct {
	Query   string // пустая строка — без фильтра по названию
	Country string // пустая строка — все страны
	Limit   int
	Offset  int
}
