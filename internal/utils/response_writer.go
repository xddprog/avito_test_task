package utils

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/xddprog/avito-test-task/internal/entity"
)

type response struct {
	Error errorDetail `json:"error"`
}
type errorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func WriteError(w http.ResponseWriter, err error) {
	var appErr *entity.AppError

	if errors.As(err, &appErr) {
		writeJSON(w, appErr.Code, appErr.SafeCode, appErr.Message)
		return
	}

	slog.Error("unknown error occurred", "error", err)
	writeJSON(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
}

func writeJSON(w http.ResponseWriter, status int, safeCode string, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(response{
		Error: errorDetail{Code: safeCode, Message: msg},
	}); err != nil {
		slog.Error("failed to encode error response", "error", err)
	}
}

func WriteOK(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if data != nil {
		if err := json.NewEncoder(w).Encode(data); err != nil {
			slog.Error("failed to encode response", "error", err)
		}
	}
}
