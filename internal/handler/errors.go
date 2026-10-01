package handler

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"log/slog"
	"net/http"
)

// ---------- requestId ----------

type requestIDKey struct{}

// RequestID — middleware: каждому запросу свой ID.
// Берём из заголовка X-Request-Id (если прислал прокси/клиент) или генерируем.
// ID возвращается в заголовке ответа и попадает в каждую ошибку — по нему ищут запрос в логах.
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Request-Id")
		if id == "" {
			id = newRequestID()
		}
		w.Header().Set("X-Request-Id", id)
		ctx := context.WithValue(r.Context(), requestIDKey{}, id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func RequestIDFrom(ctx context.Context) string {
	id, _ := ctx.Value(requestIDKey{}).(string)
	return id
}

func newRequestID() string {
	b := make([]byte, 12)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// ---------- ошибки по ТЗ (§2.6) ----------

func newError(ctx context.Context, code, msg string) Error {
	return Error{Error: ErrorBody{
		Code:      ErrorCode(code),
		Message:   msg,
		RequestId: RequestIDFrom(ctx),
	}}
}

func writeJSONError(w http.ResponseWriter, r *http.Request, status int, code, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(newError(r.Context(), code, msg))
}

// RequestErrorHandler — ошибки разбора параметров (нет from, lat="abc") → 400.
func RequestErrorHandler(w http.ResponseWriter, r *http.Request, err error) {
	writeJSONError(w, r, http.StatusBadRequest, "invalid_argument", err.Error())
}

// InternalErrorHandler — непредвиденные ошибки → 500.
// Подробности — в лог (с requestId), клиенту — общее сообщение.
func InternalErrorHandler(log *slog.Logger) func(http.ResponseWriter, *http.Request, error) {
	return func(w http.ResponseWriter, r *http.Request, err error) {
		log.Error("request failed",
			"requestId", RequestIDFrom(r.Context()),
			"method", r.Method,
			"path", r.URL.Path, // только путь, без query — там могут быть координаты (§16)
			"err", err,
		)
		writeJSONError(w, r, http.StatusInternalServerError, "internal", "internal server error")
	}
}
