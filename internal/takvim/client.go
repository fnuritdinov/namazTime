// Package takvim — запасной источник официального времени намаза для Душанбе: takvim.tj
// (данные Исламского центра РТ). Сайт показывает только 3 дня: сегодня, завтра, послезавтра.
// Нужен, когда Шӯрои уламо ещё не опубликовал месяц.
package takvim

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"nTime/internal/hijri"
	"nTime/internal/official"
)

const pageURL = "https://takvim.tj/main_namaz_tj.php"

type Client struct {
	http *http.Client
	url  string
}

func NewClient() *Client {
	return &Client{http: &http.Client{Timeout: 20 * time.Second}, url: pageURL}
}

// FetchDays скачивает страницу и возвращает 3 дня для Душанбе.
func (c *Client) FetchDays(ctx context.Context) ([]official.Row, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "ZikrAPI/1.0 (+https://api.zikr.cloud)")
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("takvim: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("takvim: status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return nil, fmt.Errorf("takvim: read: %w", err)
	}
	return Parse(string(body))
}

// var bomdod_from_imruz = "05:22"; — имя_день = "значение"
var varRe = regexp.MustCompile(`var\s+(\w+?)_(imruz|pagohi_digar|pagoh)\s*=\s*"([^"]*)"`)

var days = []string{"imruz", "pagoh", "pagohi_digar"} // сегодня, завтра, послезавтра

// Parse достаёт из страницы блок Душанбе (case 0 — сдвиг 0 минут) и собирает 3 строки таблицы.
func Parse(page string) ([]official.Row, error) {
	start := strings.Index(page, "switch (shift_for_namaz)")
	if start < 0 {
		return nil, errors.New("takvim: block shift_for_namaz not found")
	}
	block := page[start:]
	if end := strings.Index(block, "break;"); end > 0 {
		block = block[:end] // только case 0 — Душанбе
	}

	vals := map[string]map[string]string{} // день → имя → значение
	for _, m := range varRe.FindAllStringSubmatch(block, -1) {
		if vals[m[2]] == nil {
			vals[m[2]] = map[string]string{}
		}
		vals[m[2]][m[1]] = strings.TrimSpace(m[3])
	}

	var rows []official.Row
	for _, day := range days {
		v := vals[day]
		date, err := time.Parse("02-01-2006", v["date"])
		if err != nil {
			return nil, fmt.Errorf("takvim: %s: bad date %q", day, v["date"])
		}
		h := hijri.FromGregorian(date)
		r := official.Row{
			Date:     date,
			HijriDay: h.Day, HijriMonth: h.Month, HijriYear: h.Year,
			FajrStart: v["bomdod_from"], FajrEnd: v["bomdod_to"],
			DhuhrStart: v["peshin_from"], DhuhrEnd: v["peshin_to"],
			AsrStart: v["asr_from"], AsrEnd: v["asr_to"],
			MakruhStart: v["forbidden_time_from"], MakruhEnd: v["forbidden_time_to"],
			MaghribStart: v["shom_from"], MaghribEnd: v["shom_to"],
			IshaStart: v["khuftan_from"], IshaEnd: v["khuftan_to"],
		}
		if err := r.Validate(); err != nil {
			return nil, fmt.Errorf("takvim: %s: %w", date.Format("2006-01-02"), err)
		}
		rows = append(rows, r)
	}
	return rows, nil
}
