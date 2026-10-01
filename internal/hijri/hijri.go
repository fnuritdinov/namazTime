package hijri

import "time"

// Date — дата по исламскому календарю.
type Date struct {
	Day, Month, Year int
}

// FromGregorian переводит дату по табличному (арифметическому) исламскому календарю.
//
// Это приближение: реальное начало месяца зависит от наблюдения луны и отличается
// по странам на ±1 день. Официальные даты по странам — /hijri/months (§7 ТЗ),
// они будут иметь приоритет над этим расчётом.
func FromGregorian(t time.Time) Date {
	y, m, d := t.Date()
	jdn := julianDayNumber(y, int(m), d)

	l := jdn - 1948440 + 10632
	n := (l - 1) / 10631
	l = l - 10631*n + 354
	j := ((10985-l)/5316)*((50*l)/17719) + (l/5670)*((43*l)/15238)
	l = l - ((30-j)/15)*((17719*j)/50) - (j/16)*((15238*j)/43) + 29
	month := (24 * l) / 709
	day := l - (709*month)/24
	year := 30*n + j - 30

	return Date{Day: day, Month: month, Year: year}
}

// julianDayNumber — номер юлианского дня для григорианской даты (целочисленная формула).
func julianDayNumber(y, m, d int) int {
	a := (14 - m) / 12
	y2 := y + 4800 - a
	m2 := m + 12*a - 3
	return d + (153*m2+2)/5 + 365*y2 + y2/4 - y2/100 + y2/400 - 32045
}

// MonthName — название месяца на разных языках.
func (d Date) MonthName() map[string]string {
	if d.Month < 1 || d.Month > 12 {
		return nil
	}
	return monthNames[d.Month-1]
}

var monthNames = [12]map[string]string{
	{"en": "Muharram", "ru": "Мухаррам", "ar": "محرم"},
	{"en": "Safar", "ru": "Сафар", "ar": "صفر"},
	{"en": "Rabiʿ al-Awwal", "ru": "Раби аль-авваль", "ar": "ربيع الأول"},
	{"en": "Rabiʿ al-Thani", "ru": "Раби ас-сани", "ar": "ربيع الآخر"},
	{"en": "Jumada al-Ula", "ru": "Джумада аль-уля", "ar": "جمادى الأولى"},
	{"en": "Jumada al-Akhirah", "ru": "Джумада ас-сани", "ar": "جمادى الآخرة"},
	{"en": "Rajab", "ru": "Раджаб", "ar": "رجب"},
	{"en": "Shaʿban", "ru": "Шаабан", "ar": "شعبان"},
	{"en": "Ramadan", "ru": "Рамадан", "ar": "رمضان"},
	{"en": "Shawwal", "ru": "Шавваль", "ar": "شوال"},
	{"en": "Dhu al-Qaʿdah", "ru": "Зуль-када", "ar": "ذو القعدة"},
	{"en": "Dhu al-Hijjah", "ru": "Зуль-хиджа", "ar": "ذو الحجة"},
}
