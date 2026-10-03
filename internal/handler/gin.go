package handler

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
)

// writeErr — ответ-ошибка в формате ТЗ: {"error": {code, message, requestId}}.
func writeErr(c *gin.Context, status int, code, msg string) {
	c.AbortWithStatusJSON(status, newError(c, code, msg))
}

// GinRequestError — неверный параметр запроса (нет lat, дата не дата) → 400.
func GinRequestError(c *gin.Context, err error, status int) {
	writeErr(c, status, "invalid_argument", err.Error())
}

// GinErrors — если обработчик завершился ошибкой и ничего не ответил, отвечаем JSON.
func GinErrors(log *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next() // сначала выполняется обработчик

		err := c.Errors.Last()
		if err == nil || c.Writer.Written() {
			return // ошибок нет или ответ уже отправлен
		}

		if c.Writer.Status() == http.StatusBadRequest { // кривой JSON в теле запроса
			writeErr(c, http.StatusBadRequest, "invalid_argument", err.Error())
			return
		}

		log.Error("request failed", "requestId", RequestIDFrom(c), "path", c.Request.URL.Path, "err", err)
		writeErr(c, http.StatusInternalServerError, "internal", "internal server error")
	}
}

// GinRecovery — паника в обработчике → 500, сервер продолжает работать.
func GinRecovery(log *slog.Logger) gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, p any) {
		log.Error("panic", "path", c.Request.URL.Path, "panic", p)
		writeErr(c, http.StatusInternalServerError, "internal", "internal server error")
	})
}
