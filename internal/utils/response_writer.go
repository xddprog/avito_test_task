package utils

import (
	"encoding/json"
	"errors"
	"log"
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

	log.Printf("Unknown error: %v", err)
	writeJSON(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
}


func writeJSON(w http.ResponseWriter, status int, safeCode string, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(response{
		Error: errorDetail{Code: safeCode, Message: msg},
	})
}


func WriteOK(w http.ResponseWriter, status int, data any) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(status)
    if data != nil {
        json.NewEncoder(w).Encode(data)
    }
}