package handler

import (
	"net/http"

	"github.com/xddprog/avito-test-task/internal/utils"
)

type HealthHandler struct{}

func NewHealthHandler() *HealthHandler {
	return &HealthHandler{}
}

func (h *HealthHandler) Check(w http.ResponseWriter, r *http.Request) {
	utils.WriteOK(w, 200, map[string]string{
		"status": "OK",
	})
}
