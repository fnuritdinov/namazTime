package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

func writeJSONError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(newError(code, msg))
}

func RequestErrorHandler(w http.ResponseWriter, r *http.Request, err error) {
	writeJSONError(w, http.StatusBadRequest, "invalid_request", err.Error())
}

func InternalErrorHandler(log *slog.Logger) func(w http.ResponseWriter, r *http.Request, err error) {
	return func(w http.ResponseWriter, r *http.Request, err error) {
		log.Error("request failed", "method", r.Method, "path", r.URL.Path, "err", err)
		writeJSONError(w, http.StatusInternalServerError, "internal_error", "internal server error")
	}
}
