package service

import (
	"context"

	"github.com/xddprog/avito-test-task/internal/entity"
	"github.com/xddprog/avito-test-task/internal/repository"
)

type StatsService interface {
	GetSummary(ctx context.Context) (*entity.StatsSummary, error)
}

type statsService struct {
	repo repository.StatsRepository
}

func NewStatsService(repo repository.StatsRepository) StatsService {
	return &statsService{repo: repo}
}

func (s *statsService) GetSummary(ctx context.Context) (*entity.StatsSummary, error) {
	reviewerStats, err := s.repo.GetReviewerAssignments(ctx)
	if err != nil {
		return nil, err
	}

	prStatus, err := s.repo.GetPRStatus(ctx)
	if err != nil {
		return nil, err
	}

	teamMembers, err := s.repo.GetTeamMembers(ctx)
	if err != nil {
		return nil, err
	}

	prLifetime, err := s.repo.GetPRLifetime(ctx)
	if err != nil {
		return nil, err
	}

	return &entity.StatsSummary{
		ReviewerAssignments: reviewerStats,
		PRStatus:            prStatus,
		TeamMembers:         teamMembers,
		PRLifetime:          prLifetime,
	}, nil
}
