package sms

import (
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode"
)

// MaxTwoSMS — столько кириллических символов помещается в 2 SMS (2 × 67).
const MaxTwoSMS = 134

// MaxNameLen — максимум символов в имени отправителя.
const MaxNameLen = 20

var ErrBadName = errors.New("name: only letters, space and hyphen, up to 20 characters")

// Prayers — время пяти намазов на один день.
type Prayers struct {
	Date                            time.Time
	Fajr, Dhuhr, Asr, Maghrib, Isha time.Time
}

var prayerNames = map[string][5]string{
	"tg": {"Бомдод", "Пешин", "Аср", "Шом", "Хуфтан"},
	"ru": {"Фаджр", "Зухр", "Аср", "Магриб", "Иша"},
}

// CleanName проверяет имя отправителя. Пустое имя допустимо.
// Только буквы, пробел и дефис — чтобы в SMS нельзя было вписать ссылку, номер или рекламу.
func CleanName(raw string) (string, error) {
	name := strings.Join(strings.Fields(raw), " ")
	if name == "" {
		return "", nil
	}
	if len([]rune(name)) > MaxNameLen {
		return "", ErrBadName
	}
	for _, r := range name {
		if !unicode.IsLetter(r) && r != ' ' && r != '-' {
			return "", ErrBadName
		}
	}
	return name, nil
}

// PrayerText собирает текст SMS. name должен быть уже проверен через CleanName.
func PrayerText(lang, name string, p Prayers) string {
	if lang != "ru" {
		lang = "tg"
	}
	date := p.Date.Format("02.01")

	var head string
	switch {
	case lang == "ru" && name != "":
		head = fmt.Sprintf("%s напоминает тебе время намаза (%s):", name, date)
	case lang == "ru":
		head = fmt.Sprintf("Время намаза (%s):", date)
	case name != "":
		head = fmt.Sprintf("%s вақти намозро хотиррасон мекунад (%s):", name, date)
	default:
		head = fmt.Sprintf("Вақти намоз (%s):", date)
	}

	n := prayerNames[lang]
	t := [5]time.Time{p.Fajr, p.Dhuhr, p.Asr, p.Maghrib, p.Isha}
	var b strings.Builder
	b.WriteString(head)
	for i := range n {
		b.WriteString("\n" + n[i] + " " + t[i].Format("15:04"))
	}
	return b.String()
}
