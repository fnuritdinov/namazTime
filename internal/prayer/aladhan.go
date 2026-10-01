package prayer

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type AladhanClient struct {
	baseURL string
	http    *http.Client
}

func NewAladhanClient(baseURL string) *AladhanClient {
	return &AladhanClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		http:    &http.Client{Timeout: 10 * time.Second},
	}
}

type aladhanResponse struct {
	Code int `json:"code"`
	Data struct {
		Timings struct {
			Fajr    string `json:"Fajr"`
			Sunrise string `json:"Sunrise"`
			Dhuhr   string `json:"Dhuhr"`
			Asr     string `json:"Asr"`
			Maghrib string `json:"Maghrib"`
			Isha    string `json:"Isha"`
		} `json:"timings"`
		Date struct {
			Hijri struct {
				Date string `json:"date"`
			} `json:"hijri"`
		} `json:"date"`
	} `json:"data"`
}

// Fetch запрашивает время намаза на дату для координат.
func (c *AladhanClient) Fetch(ctx context.Context, lat, lon float64, date time.Time, method, school int) (DayTimings, error) {
	u, err := url.Parse(c.baseURL + "/timings/" + date.Format("02-01-2006"))
	if err != nil {
		return DayTimings{}, fmt.Errorf("aladhan url: %w", err)
	}
	q := u.Query()
	q.Set("latitude", strconv.FormatFloat(lat, 'f', 2, 64))
	q.Set("longitude", strconv.FormatFloat(lon, 'f', 2, 64))
	q.Set("method", strconv.Itoa(method))
	q.Set("school", strconv.Itoa(school))
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return DayTimings{}, fmt.Errorf("aladhan request: %w", err)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return DayTimings{}, fmt.Errorf("aladhan do: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return DayTimings{}, fmt.Errorf("aladhan status %d: %s", resp.StatusCode, body)
	}

	var body aladhanResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return DayTimings{}, fmt.Errorf("aladhan decode: %w", err)
	}

	t := body.Data.Timings
	return DayTimings{
		Date:      date,
		HijriDate: body.Data.Date.Hijri.Date,
		Source:    "aladhan",
		Timings: Timings{
			Fajr:    cleanTime(t.Fajr),
			Sunrise: cleanTime(t.Sunrise),
			Dhuhr:   cleanTime(t.Dhuhr),
			Asr:     cleanTime(t.Asr),
			Maghrib: cleanTime(t.Maghrib),
			Isha:    cleanTime(t.Isha),
		},
	}, nil
}

// cleanTime убирает возможный суффикс часового пояса: "04:52 (+05)" → "04:52".
func cleanTime(s string) string {
	if f := strings.Fields(s); len(f) > 0 {
		return f[0]
	}
	return s
}
