package service

import (
	"context"
	"math/rand"
	"time"

	"github.com/xddprog/avito-test-task/internal/entity"
	"github.com/xddprog/avito-test-task/internal/repository"
)

type PullRequestService interface {
	Create(ctx context.Context, req *entity.CreatePRRequest) (*entity.PullRequest, error)
	Merge(ctx context.Context, prID string) (*entity.PullRequest, error)
	Reassign(ctx context.Context, prID, oldUserID string) (*entity.PullRequest, string, error)
}

type prService struct {
	prRepo   repository.PullRequestRepository
	userRepo repository.UserRepository
}

func NewPullRequestService(prRepo repository.PullRequestRepository, userRepo repository.UserRepository) PullRequestService {
	return &prService{
		prRepo:   prRepo,
		userRepo: userRepo,
	}
}

func (s *prService) Create(ctx context.Context, req *entity.CreatePRRequest) (*entity.PullRequest, error) {
	author, err := s.userRepo.GetByID(ctx, req.AuthorID)
	if err != nil {
		return nil, entity.ErrNotFound
	}

	candidates, err := s.userRepo.GetActiveByTeamID(ctx, author.TeamName)
	if err != nil {
		return nil, err
	}

	reviewers := selectRandomReviewers(candidates, author.ID, nil, 2)

	pr := &entity.PullRequest{
		BasePullRequest: entity.BasePullRequest{
			ID:       req.ID,
			Name:     req.Name,
			AuthorID: req.AuthorID,
			Status:   entity.StatusOpen,
		},
		Reviewers: make([]string, 0, len(reviewers)),
	}

	for _, r := range reviewers {
		pr.Reviewers = append(pr.Reviewers, r.ID)
	}

	if err := s.prRepo.Create(ctx, pr); err != nil {
		return nil, err
	}

	return pr, nil
}

func (s *prService) Merge(ctx context.Context, prID string) (*entity.PullRequest, error) {
	return s.prRepo.Merge(ctx, prID)
}

func (s *prService) Reassign(ctx context.Context, prID, oldUserID string) (*entity.PullRequest, string, error) {
	pr, err := s.prRepo.GetByID(ctx, prID)
	if err != nil {
		return nil, "", err
	}

	if pr.Status == entity.StatusMerged {
		return nil, "", entity.ErrPRMerged
	}

	found := false
	for _, reviewerID := range pr.Reviewers {
		if reviewerID == oldUserID {
			found = true
			break
		}
	}
	if !found {
		return nil, "", entity.ErrNotAssigned
	}

	oldReviewer, err := s.userRepo.GetByID(ctx, oldUserID)
	if err != nil {
		return nil, "", entity.ErrNotFound
	}

	candidates, err := s.userRepo.GetActiveByTeamID(ctx, oldReviewer.TeamName)
	if err != nil {
		return nil, "", err
	}

	excludeIDs := make([]string, 0, len(pr.Reviewers)+1)
	excludeIDs = append(excludeIDs, pr.AuthorID)
	excludeIDs = append(excludeIDs, pr.Reviewers...)

	newReviewers := selectRandomReviewers(candidates, pr.AuthorID, excludeIDs, 1)
	if len(newReviewers) == 0 {
		return nil, "", entity.ErrNoCandidate
	}

	newUserID := newReviewers[0].ID

	if err := s.prRepo.Reassign(ctx, prID, oldUserID, newUserID); err != nil {
		return nil, "", err
	}

	pr, err = s.prRepo.GetByID(ctx, prID)
	if err != nil {
		return nil, "", err
	}

	return pr, newUserID, nil
}

func selectRandomReviewers(candidates []entity.User, authorID string, excludeIDs []string, limit int) []entity.User {
	excludeMap := make(map[string]bool)
	excludeMap[authorID] = true
	for _, id := range excludeIDs {
		excludeMap[id] = true
	}

	var valid []entity.User
	for _, u := range candidates {
		if !excludeMap[u.ID] {
			valid = append(valid, u)
		}
	}

	rand.New(rand.NewSource(time.Now().UnixNano()))
	rand.Shuffle(len(valid), func(i, j int) {
		valid[i], valid[j] = valid[j], valid[i]
	})

	if len(valid) > limit {
		return valid[:limit]
	}
	return valid
}
