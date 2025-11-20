package service

import (
	"context"

	"github.com/xddprog/avito-test-task/internal/entity"
	"github.com/xddprog/avito-test-task/internal/repository"
	"github.com/xddprog/avito-test-task/internal/utils"
)

type TeamService interface {
	Create(ctx context.Context, team *entity.Team) error
	GetByName(ctx context.Context, name string) (*entity.Team, error)
}

type teamService struct {
	repo repository.TeamRepository
}

func NewTeamService(repo repository.TeamRepository) TeamService {
	return &teamService{repo: repo}
}

func (s *teamService) Create(ctx context.Context, team *entity.Team) error {
	if err := utils.ValidateForm(team); err != nil {
		return err
	}

	return s.repo.Create(ctx, team)
}

func (s *teamService) GetByName(ctx context.Context, name string) (*entity.Team, error) {
	if name == "" {
		return nil, entity.ErrBadRequest
	}
	return s.repo.GetByName(ctx, name)
}