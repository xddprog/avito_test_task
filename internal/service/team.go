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
	DeactivateMembers(ctx context.Context, req *entity.DeactivateTeamMembersRequest) (*entity.DeactivateTeamMembersResponse, error)
}

type teamService struct {
	teamRepository repository.TeamRepository
	prRepository   repository.PullRequestRepository
	prService      PullRequestService
	userRepository repository.UserRepository
}

func NewTeamService(
	teamRepository repository.TeamRepository,
	prRepository repository.PullRequestRepository,
	prService PullRequestService,
	userRepository repository.UserRepository,
) TeamService {
	return &teamService{
		teamRepository: teamRepository,
		prRepository:   prRepository,
		prService:      prService,
		userRepository: userRepository,
	}
}

func (s *teamService) Create(ctx context.Context, team *entity.Team) error {
	if err := utils.ValidateForm(team); err != nil {
		return err
	}

	return s.teamRepository.Create(ctx, team)
}

func (s *teamService) GetByName(ctx context.Context, name string) (*entity.Team, error) {
	if name == "" {
		return nil, entity.ErrBadRequest
	}
	return s.teamRepository.GetByName(ctx, name)
}

func (s *teamService) DeactivateMembers(ctx context.Context, req *entity.DeactivateTeamMembersRequest) (*entity.DeactivateTeamMembersResponse, error) {
	if err := utils.ValidateForm(req); err != nil {
		return nil, err
	}
	if len(req.UserIDs) == 0 {
		return nil, entity.ErrBadRequest
	}

	result := &entity.DeactivateTeamMembersResponse{
		DeactivatedUsers:    []string{},
		SuccessfulReassigns: []entity.ReassignmentResult{},
		FailedReassigns:     []entity.ReassignmentResult{},
	}

	assignments, err := s.prRepository.GetOpenAssignmentsForUsers(ctx, req.UserIDs)
	if err != nil {
		return nil, err
	}

	activeCandidates, err := s.userRepository.GetActiveByTeamID(ctx, req.TeamName)
	if err != nil {
		return nil, err
	}
	candidatePool := filterCandidates(activeCandidates, req.UserIDs)

	replacements := make([]entity.ReassignmentResult, 0, len(assignments))
	userFailed := make(map[string]bool, len(req.UserIDs))

	for _, assignment := range assignments {
		exclude := make(map[string]struct{}, len(assignment.Reviewers)+2)
		exclude[assignment.AuthorID] = struct{}{}
		for _, reviewer := range assignment.Reviewers {
			exclude[reviewer] = struct{}{}
		}

		newReviewer := pickReplacement(candidatePool, exclude)
		if newReviewer == "" {
			result.FailedReassigns = append(result.FailedReassigns, entity.ReassignmentResult{
				PullRequestID: assignment.PullRequestID,
				OldReviewerID: assignment.OldReviewerID,
				Error:         entity.ErrNoCandidate.Error(),
			})
			userFailed[assignment.OldReviewerID] = true
			continue
		}

		replacements = append(replacements, entity.ReassignmentResult{
			PullRequestID: assignment.PullRequestID,
			OldReviewerID: assignment.OldReviewerID,
			NewReviewerID: newReviewer,
		})

		result.SuccessfulReassigns = append(result.SuccessfulReassigns, entity.ReassignmentResult{
			PullRequestID: assignment.PullRequestID,
			OldReviewerID: assignment.OldReviewerID,
			NewReviewerID: newReviewer,
		})
	}

	if err := s.prRepository.ApplyReviewerReplacements(ctx, replacements); err != nil {
		return nil, err
	}

	usersToDeactivate := make([]string, 0, len(req.UserIDs))
	for _, userID := range req.UserIDs {
		if !userFailed[userID] {
			usersToDeactivate = append(usersToDeactivate, userID)
		}
	}

	if len(usersToDeactivate) > 0 {
		actualDeactivated, err := s.teamRepository.DeactivateMembers(ctx, req.TeamName, usersToDeactivate)
		if err != nil {
			return nil, err
		}
		result.DeactivatedUsers = actualDeactivated
	}

	return result, nil
}

func filterCandidates(users []entity.User, exclude []string) []string {
	excludeSet := make(map[string]struct{}, len(exclude))
	for _, id := range exclude {
		excludeSet[id] = struct{}{}
	}
	candidates := make([]string, 0, len(users))
	for _, u := range users {
		if _, found := excludeSet[u.ID]; found {
			continue
		}
		candidates = append(candidates, u.ID)
	}
	return candidates
}

func pickReplacement(candidates []string, exclude map[string]struct{}) string {
	for _, id := range candidates {
		if _, found := exclude[id]; found {
			continue
		}
		return id
	}
	return ""
}
