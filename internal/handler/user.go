package handler

import (
	"encoding/json"
	"net/http"

	"github.com/xddprog/avito-test-task/internal/entity"
	"github.com/xddprog/avito-test-task/internal/service"
	"github.com/xddprog/avito-test-task/internal/utils"
)

type UserHandler struct {
	userService service.UserService
}

func NewUserHandler(userService service.UserService) *UserHandler {
	return &UserHandler{userService: userService}
}

type setIsActiveRequest struct {
	UserID   string `json:"user_id" validate:"required"`
	IsActive bool   `json:"is_active"`
}

func (h *UserHandler) SetIsActive(w http.ResponseWriter, r *http.Request) {
	var req setIsActiveRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.WriteError(w, entity.ErrBadRequest)
		return
	}

	if err := utils.ValidateForm(req); err != nil {
		utils.WriteError(w, err)
		return
	}

	updatedUser, err := h.userService.SetIsActive(r.Context(), req.UserID, req.IsActive)
	if err != nil {
		utils.WriteError(w, err)
		return
	}

	utils.WriteOK(w, http.StatusOK, map[string]any{
		"user": updatedUser,
	})
}

func (h *UserHandler) GetReview(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		utils.WriteError(w, entity.ErrBadRequest)
		return
	}

	prs, err := h.userService.GetReviews(r.Context(), userID)
	if err != nil {
		utils.WriteError(w, err)
		return
	}

	utils.WriteOK(w, http.StatusOK, map[string]any{
		"user_id":       userID,
		"pull_requests": prs,
	})
}
