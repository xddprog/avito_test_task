package handler

import (
	"encoding/json"
	"net/http"

	"github.com/xddprog/avito-test-task/internal/entity"
	"github.com/xddprog/avito-test-task/internal/service"
	"github.com/xddprog/avito-test-task/internal/utils"
)

type TeamHandler struct {
	teamService service.TeamService
}

func NewTeamHandler(teamService service.TeamService) *TeamHandler {
	return &TeamHandler{teamService: teamService}
}


func (h *TeamHandler) AddTeam(w http.ResponseWriter, r *http.Request) {
	var req entity.CreateTeamRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.WriteError(w, entity.ErrBadRequest)
		return
	}

	members := make([]entity.User, 0, len(req.Members))
	for _, m := range req.Members {
		members = append(members, entity.User{
			ID:       m.UserID,
			Username: m.Username,
			IsActive: m.IsActive,
			TeamName: req.TeamName,
		})
	}

	newTeam := &entity.Team{
		Name:    req.TeamName,
		Members: members,
	}

	if err := h.teamService.Create(r.Context(), newTeam); err != nil {
		utils.WriteError(w, err)
		return
	}

	utils.WriteOK(w, http.StatusCreated, map[string]any{
		"team": newTeam,
	})
}

func (h *TeamHandler) GetTeam(w http.ResponseWriter, r *http.Request) {
	teamName := r.URL.Query().Get("team_name")

	team, err := h.teamService.GetByName(r.Context(), teamName)
	if err != nil {
		utils.WriteError(w, err)
		return
	}

	utils.WriteOK(w, http.StatusOK, team)
}