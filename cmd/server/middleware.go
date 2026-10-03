package main

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"

	"nTime/internal/accesslog"
	"nTime/internal/compress"
	"nTime/internal/config"
	"nTime/internal/device"
	"nTime/internal/handler"
	"nTime/internal/httpcache"
	"nTime/internal/ratelimit"
)

// Сколько клиенту можно кешировать ответы (§2.5 ТЗ): расписания — час, справочники — сутки.
var cacheRules = []httpcache.Rule{
	{Prefix: "/v1/prayer-times", MaxAge: time.Hour},
	{Prefix: "/v1/config", MaxAge: time.Hour},
	{Prefix: "/v1/ramadan", MaxAge: time.Hour},
	{Prefix: "/v1/cities", MaxAge: 24 * time.Hour},
}

// withMiddleware оборачивает роутер фильтрами.
//
// Читать снизу вверх: последний обёрнутый получает запрос первым.
// Путь запроса: RequestID → журнал → лимиты → размер тела → gzip → кеш → токен → роутер.
func withMiddleware(h http.Handler, cfg config.Config, rdb *redis.Client, devices *device.Service, log *slog.Logger) http.Handler {
	counter := ratelimit.RedisCounter{RDB: rdb}
	limit := func(name, prefix string, perMinute int) func(http.Handler) http.Handler {
		return ratelimit.Middleware(counter, ratelimit.Options{
			Name:       name,
			Prefix:     prefix,
			Limit:      perMinute,
			Window:     time.Minute,
			TrustProxy: cfg.TrustProxy,
			RequestID:  handler.RequestIDFrom,
			Log:        log,
		})
	}

	h = device.Auth(devices, "/v1/devices/me", handler.RequestIDFrom, log)(h) // токен устройства
	h = httpcache.Middleware(cacheRules)(h)                                   // ETag, 304
	h = compress.Gzip(h)                                                      // сжатие ответа
	h = http.MaxBytesHandler(h, 64<<10)                                       // тело запроса ≤ 64 КБ
	h = limit("devices", "/v1/devices", cfg.DevicesRateLimitPerMinute)(h)     // 30/мин на /devices
	h = limit("all", "", cfg.RateLimitPerMinute)(h)                           // 120/мин на всё
	h = accesslog.Middleware(log, handler.RequestIDFrom)(h)                   // журнал без координат
	h = handler.RequestID(h)                                                  // ID запроса
	return h
}
