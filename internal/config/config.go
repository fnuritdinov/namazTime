package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	HTTPPort    string
	DatabaseURL string
	RedisAddr   string

	// Для GET /v1/config (§4 ТЗ) — меняются без перевыпуска кода
	MinAppVersion      string         // APP_MIN_VERSION
	LatestAppVersion   string         // APP_LATEST_VERSION
	SupportURL         string         // SUPPORT_URL
	FeatureSync        bool           // FEATURE_SYNC
	FeatureQuranSearch bool           // FEATURE_QURAN_SEARCH
	ContentVersions    map[string]int // CONTENT_VERSIONS="cities=1,quran=3"
}

func Load() (Config, error) {
	cfg := Config{
		HTTPPort:         getEnv("HTTP_PORT", "8080"),
		DatabaseURL:      os.Getenv("DATABASE_URL"),
		RedisAddr:        getEnv("REDIS_ADDR", "localhost:6379"),
		MinAppVersion:    getEnv("APP_MIN_VERSION", "1.0.0"),
		LatestAppVersion: getEnv("APP_LATEST_VERSION", "1.0.0"),
		SupportURL:       getEnv("SUPPORT_URL", "https://namoz.tj/support"),
	}
	if cfg.DatabaseURL == "" {
		return Config{}, fmt.Errorf("DATABASE_URL is required")
	}

	var err error
	if cfg.FeatureSync, err = getBool("FEATURE_SYNC", false); err != nil {
		return Config{}, err
	}
	if cfg.FeatureQuranSearch, err = getBool("FEATURE_QURAN_SEARCH", false); err != nil {
		return Config{}, err
	}
	if cfg.ContentVersions, err = parseVersions(getEnv("CONTENT_VERSIONS", "cities=1")); err != nil {
		return Config{}, fmt.Errorf("CONTENT_VERSIONS: %w", err)
	}
	return cfg, nil
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getBool(key string, def bool) (bool, error) {
	v := os.Getenv(key)
	if v == "" {
		return def, nil
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return false, fmt.Errorf("%s must be true or false, got %q", key, v)
	}
	return b, nil
}

// parseVersions разбирает "cities=1,quran=3" → {"cities": 1, "quran": 3}.
func parseVersions(s string) (map[string]int, error) {
	out := map[string]int{}
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		name, value, ok := strings.Cut(part, "=")
		if !ok || strings.TrimSpace(name) == "" {
			return nil, fmt.Errorf("expected name=number, got %q", part)
		}
		n, err := strconv.Atoi(strings.TrimSpace(value))
		if err != nil || n < 0 {
			return nil, fmt.Errorf("version of %q must be a non-negative number, got %q", name, value)
		}
		out[strings.TrimSpace(name)] = n
	}
	return out, nil
}
