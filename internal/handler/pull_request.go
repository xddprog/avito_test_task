package handler

import (
	"encoding/json"
	"net/http"

	"github.com/xddprog/avito-test-task/internal/entity"
	"github.com/xddprog/avito-test-task/internal/service"
	"github.com/xddprog/avito-test-task/internal/utils"
)

type PullRequestHandler struct {
	prService service.PullRequestService
}

func NewPullRequestHandler(prService service.PullRequestService) *PullRequestHandler {
	return &PullRequestHandler{prService: prService}
}

func (h *PullRequestHandler) CreatePullRequest(w http.ResponseWriter, r *http.Request) {
	var req entity.CreatePRRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.WriteError(w, entity.ErrBadRequest)
		return
	}

	if req.ID == "" || req.Name == "" || req.AuthorID == "" {
		utils.WriteError(w, entity.ErrBadRequest)
		return
	}

	pr, err := h.prService.Create(r.Context(), &req)
	if err != nil {
		utils.WriteError(w, err)
		return
	}

	utils.WriteOK(w, http.StatusCreated, map[string]any{
		"pr": pr,
	})
}

func (h *PullRequestHandler) MergePullRequest(w http.ResponseWriter, r *http.Request) {
	var req entity.MergePRRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.WriteError(w, entity.ErrBadRequest)
		return
	}

	if req.ID == "" {
		utils.WriteError(w, entity.ErrBadRequest)
		return
	}

	pr, err := h.prService.Merge(r.Context(), req.ID)
	if err != nil {
		utils.WriteError(w, err)
		return
	}

	utils.WriteOK(w, http.StatusOK, map[string]any{
		"pr": pr,
	})
}

func (h *PullRequestHandler) ReassignPullRequest(w http.ResponseWriter, r *http.Request) {
	var req entity.ReassignPRRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.WriteError(w, entity.ErrBadRequest)
		return
	}

	if req.PRID == "" || req.OldUserID == "" {
		utils.WriteError(w, entity.ErrBadRequest)
		return
	}

	pr, newUserID, err := h.prService.Reassign(r.Context(), req.PRID, req.OldUserID)
	if err != nil {
		utils.WriteError(w, err)
		return
	}

	utils.WriteOK(w, http.StatusOK, map[string]any{
		"pr":          pr,
		"replaced_by": newUserID,
	})
}
