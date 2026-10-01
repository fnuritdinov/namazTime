package prayertime

import (
	"errors"
	"fmt"
	"math"
	"time"
)

// Madhab влияет только на время Асра: длина тени = factor × высота предмета + тень в полдень.
const (
	Shafii = "shafii" // factor 1
	Hanafi = "hanafi" // factor 2
)

// Поправки по ТЗ (§6): Зухр = истинный полдень + 2 мин, Магриб = закат + 1 мин, Имсак = Фаджр − 10 мин.
const (
	dhuhrOffset   = 2 * time.Minute
	maghribOffset = 1 * time.Minute
	imsakBefore   = 10 * time.Minute
	sunsetAngle   = 0.833 // рефракция + радиус диска солнца, градусы
)

// ErrPolar — полярный день/ночь: солнце не восходит или не заходит.
var ErrPolar = errors.New("sun does not rise or set on this date")

// Times — время намазов на один день (в часовом поясе города).
type Times struct {
	Imsak, Fajr, Sunrise, Dhuhr, Asr, Maghrib, Isha time.Time
}

// Calculate считает время намазов на дату date (берутся год/месяц/день) для координат lat/lon
// в часовом поясе loc.
func Calculate(date time.Time, lat, lon float64, loc *time.Location, m Method, madhab string) (Times, error) {
	asrFactor := 1.0
	switch madhab {
	case Hanafi:
		asrFactor = 2
	case Shafii:
		asrFactor = 1
	default:
		return Times{}, fmt.Errorf("unknown madhab %q", madhab)
	}

	y, mo, d := date.Date()
	// Точка отсчёта — полночь UTC нужной даты. Расчёт ниже идёт в «местном солнечном» времени (часы).
	base := time.Date(y, mo, d, 0, 0, 0, 0, time.UTC)
	jd := julianDay(y, int(mo), d) - lon/(15*24)

	// Начальные приближения (доли суток), затем одно уточнение — как в PrayTimes.
	t := map[string]float64{"fajr": 5, "sunrise": 6, "dhuhr": 12, "asr": 13, "sunset": 18, "isha": 18}
	for i := 0; i < 2; i++ {
		p := func(k string) float64 { return t[k] / 24 }
		t = map[string]float64{
			"fajr":    sunAngleTime(jd, m.FajrAngle, p("fajr"), lat, true),
			"sunrise": sunAngleTime(jd, sunsetAngle, p("sunrise"), lat, true),
			"dhuhr":   midDay(jd, p("dhuhr")),
			"asr":     asrTime(jd, asrFactor, p("asr"), lat),
			"sunset":  sunAngleTime(jd, sunsetAngle, p("sunset"), lat, false),
			"isha":    sunAngleTime(jd, ishaAngleOrZero(m), p("isha"), lat, false),
		}
	}

	// Высокие широты (Москва летом): солнце не опускается на 18°, формула даёт NaN.
	// Правило «середина ночи»: Фаджр/Иша не позже/раньше середины ночи (§17 ТЗ).
	night := 24 - t["sunset"] + t["sunrise"]
	if math.IsNaN(t["fajr"]) {
		t["fajr"] = t["sunrise"] - night/2
	}
	if math.IsNaN(t["isha"]) {
		t["isha"] = t["sunset"] + night/2
	}
	if math.IsNaN(t["sunrise"]) || math.IsNaN(t["sunset"]) {
		return Times{}, ErrPolar
	}

	// Перевод из «местного солнечного» времени в UTC: вычитаем долготу.
	at := func(hours float64) time.Time {
		utcHours := hours - lon/15
		return base.Add(time.Duration(utcHours * float64(time.Hour))).In(loc)
	}

	res := Times{
		Fajr:    at(t["fajr"]),
		Sunrise: at(t["sunrise"]),
		Dhuhr:   at(t["dhuhr"]).Add(dhuhrOffset),
		Asr:     at(t["asr"]),
		Maghrib: at(t["sunset"]).Add(maghribOffset),
	}
	if m.Isha.Type == "minutesAfterMaghrib" {
		res.Isha = at(t["sunset"]).Add(time.Duration(m.Isha.Value * float64(time.Minute)))
	} else {
		res.Isha = at(t["isha"])
	}
	res.Imsak = res.Fajr.Add(-imsakBefore)

	// Округляем до минуты — так время показывают пользователю.
	for _, p := range []*time.Time{&res.Imsak, &res.Fajr, &res.Sunrise, &res.Dhuhr, &res.Asr, &res.Maghrib, &res.Isha} {
		*p = p.Round(time.Minute)
	}
	return res, nil
}

func ishaAngleOrZero(m Method) float64 {
	if m.Isha.Type == "angle" {
		return m.Isha.Value
	}
	return 0
}

// ---------- астрономия (формулы PrayTimes.org / Adhan) ----------

func julianDay(y, m, d int) float64 {
	if m <= 2 {
		y--
		m += 12
	}
	a := math.Floor(float64(y) / 100)
	b := 2 - a + math.Floor(a/4)
	return math.Floor(365.25*float64(y+4716)) + math.Floor(30.6001*float64(m+1)) + float64(d) + b - 1524.5
}

// sunPosition возвращает склонение солнца (градусы) и уравнение времени (часы).
func sunPosition(jd float64) (decl, eqt float64) {
	D := jd - 2451545.0
	g := fixAngle(357.529 + 0.98560028*D)
	q := fixAngle(280.459 + 0.98564736*D)
	L := fixAngle(q + 1.915*dsin(g) + 0.020*dsin(2*g))
	e := 23.439 - 0.00000036*D

	RA := darctan2(dcos(e)*dsin(L), dcos(L)) / 15
	eqt = q/15 - fixHour(RA)
	decl = darcsin(dsin(e) * dsin(L))
	return decl, eqt
}

// midDay — истинный полдень (часы, местное солнечное время).
func midDay(jd, t float64) float64 {
	_, eqt := sunPosition(jd + t)
	return fixHour(12 - eqt)
}

// sunAngleTime — когда солнце на angle градусов под горизонтом (до полудня, если ccw).
func sunAngleTime(jd, angle, t, lat float64, ccw bool) float64 {
	decl, _ := sunPosition(jd + t)
	noon := midDay(jd, t)
	cosT := (-dsin(angle) - dsin(decl)*dsin(lat)) / (dcos(decl) * dcos(lat))
	T := darccos(cosT) / 15
	if ccw {
		return noon - T
	}
	return noon + T
}

// asrTime — Аср: тень = factor × предмет + полуденная тень.
func asrTime(jd, factor, t, lat float64) float64 {
	decl, _ := sunPosition(jd + t)
	angle := -darccot(factor + dtan(math.Abs(lat-decl)))
	return sunAngleTime(jd, angle, t, lat, false)
}

// ---------- тригонометрия в градусах ----------

func rad(d float64) float64         { return d * math.Pi / 180 }
func deg(r float64) float64         { return r * 180 / math.Pi }
func dsin(d float64) float64        { return math.Sin(rad(d)) }
func dcos(d float64) float64        { return math.Cos(rad(d)) }
func dtan(d float64) float64        { return math.Tan(rad(d)) }
func darcsin(x float64) float64     { return deg(math.Asin(x)) }
func darccos(x float64) float64     { return deg(math.Acos(x)) }
func darctan2(y, x float64) float64 { return deg(math.Atan2(y, x)) }
func darccot(x float64) float64     { return deg(math.Atan(1 / x)) }
func fixAngle(a float64) float64    { return fix(a, 360) }
func fixHour(h float64) float64     { return fix(h, 24) }
func fix(a, b float64) float64 {
	a = a - b*math.Floor(a/b)
	if a < 0 {
		a += b
	}
	return a
}
