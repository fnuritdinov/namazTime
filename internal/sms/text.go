package sms

import (
	"errors"
	"strings"
	"time"
	"unicode"
)

// MaxOneSMS — латиница (GSM-7): в одно SMS помещается 160 символов.
const MaxOneSMS = 160

// MaxNameLen — максимум символов в имени и фамилии после перевода в латиницу.
const MaxNameLen = 20

var ErrBadName = errors.New("name is required: only letters, space and hyphen, up to 20 latin characters")

// Slot — начало и конец времени намаза.
type Slot struct{ From, To time.Time }

// Prayers — пять намазов на один день.
type Prayers struct {
	Date                            time.Time
	Fajr, Dhuhr, Asr, Maghrib, Isha Slot
}

var prayerNames = [5]string{"Bomdod", "Peshin", "Asr", "Shom", "Khuftan"}

// Кириллица (русская и таджикская) → латиница.
var latin = map[rune]string{
	'а': "a", 'б': "b", 'в': "v", 'г': "g", 'ғ': "gh", 'д': "d", 'е': "e", 'ё': "yo",
	'ж': "zh", 'з': "z", 'и': "i", 'ӣ': "i", 'й': "y", 'к': "k", 'қ': "q", 'л': "l",
	'м': "m", 'н': "n", 'о': "o", 'п': "p", 'р': "r", 'с': "s", 'т': "t", 'у': "u",
	'ӯ': "u", 'ф': "f", 'х': "kh", 'ҳ': "h", 'ц': "ts", 'ч': "ch", 'ҷ': "j", 'ш': "sh",
	'щ': "sh", 'ъ': "", 'ы': "y", 'ь': "", 'э': "e", 'ю': "yu", 'я': "ya",
}

// toLatin переводит кириллицу в латиницу, остальное оставляет как есть.
func toLatin(s string) string {
	var b strings.Builder
	for _, r := range s {
		l, ok := latin[unicode.ToLower(r)]
		if !ok {
			b.WriteRune(r)
			continue
		}
		if unicode.IsUpper(r) && l != "" {
			l = strings.ToUpper(l[:1]) + l[1:]
		}
		b.WriteString(l)
	}
	return b.String()
}

// CleanName проверяет имя и фамилию и переводит их в латиницу. Имя обязательно.
// После перевода — только латинские буквы, пробел и дефис: так нельзя вписать ссылку
// или номер, и текст остаётся латиницей (1 SMS).
func CleanName(raw string) (string, error) {
	name := toLatin(strings.Join(strings.Fields(raw), " "))
	if name == "" || len(name) > MaxNameLen {
		return "", ErrBadName
	}
	for _, r := range name {
		if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r == ' ' || r == '-') {
			return "", ErrBadName
		}
	}
	return name, nil
}

// PrayerText собирает текст SMS. name должен быть уже проверен через CleanName.
// Шаблон постоянный, меняются только дата, имя и время:
//
//	Vaqti namoz 25.10.2026 az nomi Farrukh:
//	Bomdod 05:15 - 06:35
//	...
func PrayerText(name string, p Prayers) string {
	slots := [5]Slot{p.Fajr, p.Dhuhr, p.Asr, p.Maghrib, p.Isha}
	var b strings.Builder
	b.WriteString("Vaqti namoz " + p.Date.Format("02.01.2006") + " az nomi " + name + ":")
	for i, s := range slots {
		b.WriteString("\n" + prayerNames[i] + " " + s.From.Format("15:04") + " - " + s.To.Format("15:04"))
	}
	return b.String()
}
