package device

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
)

type ctxKey struct{}

// IDFrom — id устройства, которое прислало запрос (кладёт Auth).
func IDFrom(ctx context.Context) string {
	id, _ := ctx.Value(ctxKey{}).(string)
	return id
}

// Auth проверяет "Authorization: Bearer <token>" для путей, начинающихся с prefix.
// Остальные пути пропускает без проверки: контент публичный (§2.7 ТЗ).
func Auth(svc *Service, prefix string, requestID func(context.Context) string, log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !strings.HasPrefix(r.URL.Path, prefix) {
				next.ServeHTTP(w, r)
				return
			}

			token, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
			id, err := "", ErrUnauthorized
			if ok {
				id, err = svc.Authenticate(r.Context(), strings.TrimSpace(token))
			}
			switch {
			case errors.Is(err, ErrUnauthorized):
				writeError(w, r, requestID, http.StatusUnauthorized, "unauthorized", "missing or invalid device token")
				return
			case err != nil:
				log.Error("device auth failed", "err", err)
				writeError(w, r, requestID, http.StatusInternalServerError, "internal", "internal server error")
				return
			}

			ctx := context.WithValue(r.Context(), ctxKey{}, id)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func writeError(w http.ResponseWriter, r *http.Request, requestID func(context.Context) string, status int, code, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"error": map[string]string{"code": code, "message": msg, "requestId": requestID(r.Context())},
	})
}
