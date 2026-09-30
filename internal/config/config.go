package config

import (
	"fmt"
	"os"
)

type Config struct {
	HTTPPort       string
	DatabaseURL    string
	RedisAddr      string
	AladhanBaseURL string
}

func Load() (Config, error) {
	cfg := Config{
		HTTPPort:       getEnv("HTTP_PORT", "8080"),
		DatabaseURL:    os.Getenv("DATABASE_URL"),
		RedisAddr:      getEnv("REDIS_ADDR", "localhost:6379"),
		AladhanBaseURL: getEnv("ALADHAN_BASE_URL", "https://api.aladhan.com/v1"),
	}

	if cfg.DatabaseURL == "" {
		return Config{}, fmt.Errorf("DATABASE_URL is required")
	}
	return cfg, nil
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
