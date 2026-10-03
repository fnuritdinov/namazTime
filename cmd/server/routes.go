package main

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"nTime/internal/handler"
)

// newRouter — все адреса сервера на Gin: /health, API /v1 из openapi.yaml и 404 для остальных.
func newRouter(api *handler.Server, db *pgxpool.Pool, rdb *redis.Client, log *slog.Logger) *gin.Engine {
	gin.SetMode(gin.ReleaseMode) // без отладочного списка роутов в консоли

	r := gin.New()
	// Обработчики получают *gin.Context как context.Context. Чтобы ctx.Value видел то,
	// что положили middleware (requestId, deviceId), и отмену запроса — нужен fallback.
	r.ContextWithFallback = true

	r.Use(handler.GinRecovery(log), handler.GinErrors(log))

	r.GET("/health", health(db, rdb))

	strict := handler.NewStrictHandler(api, nil)
	handler.RegisterHandlersWithOptions(r, strict, handler.GinServerOptions{
		BaseURL:      "/v1",
		ErrorHandler: handler.GinRequestError,
	})

	r.NoRoute(notFound) // всё, что не подошло выше
	return r
}

// health — жив ли сервер, Postgres и Redis (для Docker и мониторинга).
func health(db *pgxpool.Pool, rdb *redis.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		status := gin.H{"status": "ok", "postgres": "ok", "redis": "ok"}
		code := http.StatusOK

		if err := db.Ping(c); err != nil {
			status["postgres"], status["status"], code = "down", "degraded", http.StatusServiceUnavailable
		}
		if err := rdb.Ping(c).Err(); err != nil {
			status["redis"], status["status"], code = "down", "degraded", http.StatusServiceUnavailable
		}
		c.JSON(code, status)
	}
}

// notFound — неизвестный адрес → 404 в формате ошибок ТЗ.
func notFound(c *gin.Context) {
	c.JSON(http.StatusNotFound, gin.H{
		"error": gin.H{
			"code":      "not_found",
			"message":   "route " + c.Request.Method + " " + c.Request.URL.Path + " does not exist",
			"requestId": handler.RequestIDFrom(c),
		},
	})
}
