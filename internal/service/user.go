package service

import (
	"context"

	"github.com/xddprog/avito-test-task/internal/entity"
	"github.com/xddprog/avito-test-task/internal/repository"
)

type UserService interface {
	SetIsActive(ctx context.Context, userID string, isActive bool) (*entity.User, error)
	GetReviews(ctx context.Context, userID string) ([]entity.BasePullRequest, error)
}

type userService struct {
	userRepository repository.UserRepository
}

func NewUserService(userRepository repository.UserRepository) UserService {
	return &userService{userRepository: userRepository}
}

func (s *userService) SetIsActive(ctx context.Context, userID string, isActive bool) (*entity.User, error) {
	if userID == "" {
		return nil, entity.ErrBadRequest
	}
	return s.userRepository.UpdateActivity(ctx, userID, isActive)
}

func (s *userService) GetReviews(ctx context.Context, userID string) ([]entity.BasePullRequest, error) {
	if userID == "" {
		return nil, entity.ErrBadRequest
	}
	
	return s.userRepository.GetAssignedPRs(ctx, userID)
}