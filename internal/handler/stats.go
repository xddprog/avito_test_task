package handler

import (
	"net/http"

	"github.com/xddprog/avito-test-task/internal/service"
	"github.com/xddprog/avito-test-task/internal/utils"
)

type StatsHandler struct {
	statsService service.StatsService
}

func NewStatsHandler(statsService service.StatsService) *StatsHandler {
	return &StatsHandler{statsService: statsService}
}

func (h *StatsHandler) Summary(w http.ResponseWriter, r *http.Request) {
	stats, err := h.statsService.GetSummary(r.Context())
	if err != nil {
		utils.WriteError(w, err)
		return
	}

	utils.WriteOK(w, http.StatusOK, map[string]any{
		"stats": stats,
	})
}
