package hijrimonth

import (
	"errors"
	"time"
)

// ErrNotFound — для страны/года нет данных.
var ErrNotFound = errors.New("hijri data not found")

// Month — начало месяца хиджры в стране.
type Month struct {
	Month       int               // 1..12
	Start       time.Time         // дата начала (григорианская)
	Status      string            // "expected" | "confirmed"
	ConfirmedAt *time.Time        // когда подтвердили (только для confirmed)
	SourceName  map[string]string // кто объявил, {"tg": "Шӯрои уламо…"}; пусто, если неизвестно
}

// Dua — дуа с переводами.
type Dua struct {
	ID              string
	Arabic          string
	Transliteration string
	Source          string
	Translations    map[string]string // язык → текст
}

// Ramadan — даты Рамадана и Ида для страны.
type Ramadan struct {
	HijriYear    int
	Country      string
	Status       string
	Start        time.Time
	End          time.Time // последний день поста
	Length       int       // 29 или 30
	EidAlFitr    time.Time
	LaylatAlQadr time.Time // ожидаемая 27-я ночь
	ConfirmedAt  *time.Time
	SourceName   map[string]string
	Duas         []Dua
}
