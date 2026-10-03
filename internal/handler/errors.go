package handler

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
)

type requestIDKey struct{}

// RequestID — middleware: даёт каждому запросу свой ID (или берёт из заголовка X-Request-Id).
// ID уходит в заголовок ответа и в каждую ошибку — по нему запрос ищут в журнале.
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Request-Id")
		if id == "" {
			b := make([]byte, 12)
			_, _ = rand.Read(b)
			id = hex.EncodeToString(b)
		}
		w.Header().Set("X-Request-Id", id)
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), requestIDKey{}, id)))
	})
}

// RequestIDFrom — ID текущего запроса ("" если его нет).
func RequestIDFrom(ctx context.Context) string {
	id, _ := ctx.Value(requestIDKey{}).(string)
	return id
}

// newError — ошибка в формате ТЗ (§2.6): {"error": {code, message, requestId}}.
func newError(ctx context.Context, code, msg string) Error {
	return Error{Error: ErrorBody{
		Code:      ErrorCode(code),
		Message:   msg,
		RequestId: RequestIDFrom(ctx),
	}}
}
