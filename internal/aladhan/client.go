// Package aladhan берёт время намазов из api.aladhan.com (метод Карачи и др.) — сами мы не считаем.
package aladhan

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const baseURL = "https://api.aladhan.com/v1"

// ErrUnavailable — Aladhan не ответил или ответил ошибкой.
var ErrUnavailable = errors.New("aladhan unavailable")

// Day — время одного дня в часовом поясе запроса.
type Day struct {
	Date                                            time.Time
	Imsak, Fajr, Sunrise, Dhuhr, Asr, Maghrib, Isha time.Time

	// Дополнительно (только для стран, где время из Aladhan).
	// Midnight, FirstThird, LastThird могут быть уже следующим днём (после 00:00).
	Sunset, Midnight, FirstThird, LastThird time.Time
}

type Client struct {
	http *http.Client
	base string
}

func NewClient() *Client {
	return &Client{http: &http.Client{Timeout: 15 * time.Second}, base: baseURL}
}

// Month — параметры запроса за месяц.
type Month struct {
	Year   int
	Month  time.Month
	Lat    float64
	Lon    float64
	Loc    *time.Location
	Method int // 1 = Карачи
	School int // 0 = шафии, 1 = ханафи
}

// FetchMonth возвращает все дни месяца.
func (c *Client) FetchMonth(ctx context.Context, m Month) ([]Day, error) {
	q := url.Values{
		"latitude":       {strconv.FormatFloat(m.Lat, 'f', 4, 64)},
		"longitude":      {strconv.FormatFloat(m.Lon, 'f', 4, 64)},
		"method":         {strconv.Itoa(m.Method)},
		"school":         {strconv.Itoa(m.School)},
		"timezonestring": {m.Loc.String()},
	}
	u := fmt.Sprintf("%s/calendar/%d/%d?%s", c.base, m.Year, int(m.Month), q.Encode())

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "ZikrBot/1.0 (+https://zikr.cloud)")
	resp, err := c.http.Do(req)
	if err != nil {
		var ue *url.Error
		if errors.As(err, &ue) {
			err = ue.Err // без адреса запроса: в нём координаты пользователя (§16)
		}
		return nil, fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return nil, fmt.Errorf("%w: read: %v", ErrUnavailable, err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: status %d", ErrUnavailable, resp.StatusCode)
	}
	return parseMonth(body, m.Year, m.Month, m.Loc)
}

type apiResponse struct {
	Code int `json:"code"`
	Data []struct {
		Timings map[string]string `json:"timings"`
		Date    struct {
			Gregorian struct {
				Date string `json:"date"` // "01-10-2026"
			} `json:"gregorian"`
		} `json:"date"`
	} `json:"data"`
}

func parseMonth(body []byte, year int, month time.Month, loc *time.Location) ([]Day, error) {
	var r apiResponse
	if err := json.Unmarshal(body, &r); err != nil {
		return nil, fmt.Errorf("%w: bad json: %v", ErrUnavailable, err)
	}
	if r.Code != 200 {
		return nil, fmt.Errorf("%w: code %d", ErrUnavailable, r.Code)
	}

	want := time.Date(year, month+1, 0, 0, 0, 0, 0, time.UTC).Day()
	if len(r.Data) != want {
		return nil, fmt.Errorf("%w: got %d days, want %d", ErrUnavailable, len(r.Data), want)
	}

	days := make([]Day, 0, len(r.Data))
	for _, d := range r.Data {
		date, err := time.ParseInLocation("02-01-2006", d.Date.Gregorian.Date, loc)
		if err != nil || date.Year() != year || date.Month() != month {
			return nil, fmt.Errorf("%w: bad date %q", ErrUnavailable, d.Date.Gregorian.Date)
		}
		day := Day{Date: date}
		for _, f := range []struct {
			key string
			dst *time.Time
		}{
			{"Imsak", &day.Imsak}, {"Fajr", &day.Fajr}, {"Sunrise", &day.Sunrise},
			{"Dhuhr", &day.Dhuhr}, {"Asr", &day.Asr}, {"Maghrib", &day.Maghrib}, {"Isha", &day.Isha},
			{"Sunset", &day.Sunset}, {"Midnight", &day.Midnight},
			{"Firstthird", &day.FirstThird}, {"Lastthird", &day.LastThird},
		} {
			t, err := clock(date, d.Timings[f.key], loc)
			if err != nil {
				return nil, fmt.Errorf("%w: %s %s: %v", ErrUnavailable, d.Date.Gregorian.Date, f.key, err)
			}
			*f.dst = t
		}
		// Полночь и трети ночи после 00:00 — это уже следующий календарный день
		for _, t := range []*time.Time{&day.Midnight, &day.FirstThird, &day.LastThird} {
			if t.Before(day.Maghrib) {
				*t = t.AddDate(0, 0, 1)
			}
		}
		if !(day.Fajr.Before(day.Sunrise) && day.Sunrise.Before(day.Dhuhr) && day.Dhuhr.Before(day.Asr) &&
			day.Asr.Before(day.Maghrib) && day.Maghrib.Before(day.Isha)) {
			return nil, fmt.Errorf("%w: %s: prayers out of order", ErrUnavailable, d.Date.Gregorian.Date)
		}
		days = append(days, day)
	}
	return days, nil
}

// clock превращает "04:27 (MSK)" в момент времени этого дня.
func clock(date time.Time, s string, loc *time.Location) (time.Time, error) {
	s = strings.TrimSpace(s)
	if i := strings.IndexByte(s, ' '); i >= 0 {
		s = s[:i]
	}
	t, err := time.Parse("15:04", s)
	if err != nil {
		return time.Time{}, fmt.Errorf("time %q", s)
	}
	return time.Date(date.Year(), date.Month(), date.Day(), t.Hour(), t.Minute(), 0, 0, loc), nil
}
