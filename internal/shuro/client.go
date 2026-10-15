// Package shuro забирает официальное расписание намазов с сайта Шурои уламо (shuroiulamo.tj).
package shuro

import (
	"context"
	"errors"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"nTime/internal/official"
)

const baseURL = "https://shuroiulamo.tj"

// ErrNotPublished — таблицы на этот месяц на сайте ещё нет.
var ErrNotPublished = errors.New("timetable is not published yet")

type Client struct {
	http *http.Client
	base string
}

func NewClient() *Client {
	jar, _ := cookiejar.New(nil) // сайт ставит cookie сессии — храним их, как браузер
	return &Client{
		http: &http.Client{Timeout: 30 * time.Second, Jar: jar},
		base: baseURL,
	}
}

// FetchMonth скачивает таблицу Душанбе за месяц и возвращает проверенные строки.
// Сайт отдаёт таблицу на POST /ru/namaz/ntime с полями fmonth, fyear, fday=0.
func (c *Client) FetchMonth(ctx context.Context, year int, month time.Month) ([]official.Row, error) {
	// 1. Открываем страницу, как браузер: сайт выдаёт cookie сессии
	if err := c.get(ctx, c.base+"/ru/namaz"); err != nil {
		return nil, fmt.Errorf("open page: %w", err)
	}

	// 2. Запрос таблицы за месяц
	form := url.Values{
		"fmonth": {fmt.Sprintf("%02d", int(month))},
		"fyear":  {strconv.Itoa(year)},
		"fday":   {"0"},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.base+"/ru/namaz/ntime", strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	req.Header.Set("Origin", c.base)
	req.Header.Set("Referer", c.base+"/ru/namaz")
	req.Header.Set("User-Agent", "ZikrBot/1.0 (+https://zikr.cloud)")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("post ntime: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("post ntime: status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return nil, fmt.Errorf("read ntime: %w", err)
	}
	return ParseMonth(body, year, month)
}

func (c *Client) get(ctx context.Context, u string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "ZikrBot/1.0 (+https://zikr.cloud)")
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 2<<20))
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("status %d", resp.StatusCode)
	}
	return nil
}

var (
	reRow   = regexp.MustCompile(`(?s)<tr[^>]*>(.*?)</tr>`)
	reCell  = regexp.MustCompile(`(?s)<td[^>]*>(.*?)</td>`)
	reTag   = regexp.MustCompile(`<[^>]+>`)
	reRange = regexp.MustCompile(`^(\d{2}:\d{2})\s*-\s*(\d{2}:\d{2})$`)
)

// ParseMonth разбирает HTML-ответ сайта. Строка таблицы:
//
//	<tr><td>Вторник</td><td>01.09.2026</td><td>19.03.1448</td>
//	    <td>04:24 - 05:51</td> ... 6 диапазонов: Бомдод, Пешин, Аср, Макрух, Шом, Хуфтан</tr>
//
// Строки других месяцев и прочие <tr> пропускаются. Если строк нужного месяца нет — ErrNotPublished.
// Если месяц неполный или строка не проходит проверку — ошибка, ничего не сохраняется.
func ParseMonth(body []byte, year int, month time.Month) ([]official.Row, error) {
	var rows []official.Row
	seen := map[int]bool{}

	for _, tr := range reRow.FindAllSubmatch(body, -1) {
		var cells []string
		for _, td := range reCell.FindAllSubmatch(tr[1], -1) {
			text := html.UnescapeString(reTag.ReplaceAllString(string(td[1]), ""))
			cells = append(cells, strings.TrimSpace(text))
		}
		if len(cells) != 9 {
			continue // заголовок или чужая строка
		}
		date, err := time.Parse("02.01.2006", cells[1])
		if err != nil {
			continue
		}
		if date.Year() != year || date.Month() != month {
			continue
		}

		r := official.Row{Date: date}
		if _, err := fmt.Sscanf(cells[2], "%d.%d.%d", &r.HijriDay, &r.HijriMonth, &r.HijriYear); err != nil {
			return nil, fmt.Errorf("%s: hijri %q: %w", cells[1], cells[2], err)
		}

		dst := [][2]*string{
			{&r.FajrStart, &r.FajrEnd},
			{&r.DhuhrStart, &r.DhuhrEnd},
			{&r.AsrStart, &r.AsrEnd},
			{&r.MakruhStart, &r.MakruhEnd},
			{&r.MaghribStart, &r.MaghribEnd},
			{&r.IshaStart, &r.IshaEnd},
		}
		for i, d := range dst {
			m := reRange.FindStringSubmatch(cells[3+i])
			if m == nil {
				return nil, fmt.Errorf("%s: column %d %q: expected \"HH:MM - HH:MM\"", cells[1], 4+i, cells[3+i])
			}
			*d[0], *d[1] = m[1], m[2]
		}
		fixIshaEnd(&r)
		if err := r.Validate(); err != nil {
			return nil, fmt.Errorf("%s: %w", cells[1], err)
		}
		if seen[date.Day()] {
			return nil, fmt.Errorf("%s: duplicate date", cells[1])
		}
		seen[date.Day()] = true
		rows = append(rows, r)
	}

	if len(rows) == 0 {
		return nil, ErrNotPublished
	}
	days := time.Date(year, month+1, 0, 0, 0, 0, 0, time.UTC).Day()
	if len(rows) != days {
		return nil, fmt.Errorf("incomplete month %04d-%02d: got %d of %d days", year, month, len(rows), days)
	}
	return rows, nil
}

// fixIshaEnd исправляет опечатки сайта в конце Хуфтана (например, 11.10.2026: "04:00" вместо "05:00").
// На сайте конец Хуфтана всегда = начало Бомдода этого же дня − 1 минута.
func fixIshaEnd(r *official.Row) {
	f, err1 := time.Parse("15:04", r.FajrStart)
	e, err2 := time.Parse("15:04", r.IshaEnd)
	if err1 != nil || err2 != nil {
		return
	}
	want := f.Add(-time.Minute)
	if d := e.Sub(want); d > 30*time.Minute || d < -30*time.Minute {
		r.IshaEnd = want.Format("15:04")
	}
}
