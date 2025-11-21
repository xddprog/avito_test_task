package entity

type Team struct {
	Name    string `json:"name"`
	Members []User `json:"members"`
}

type CreateTeamRequest struct {
	TeamName string          `json:"team_name"`
	Members  []TeamMemberDTO `json:"members"`
}

type TeamMemberDTO struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	IsActive bool   `json:"is_active"`
}

type DeactivateTeamMembersRequest struct {
	TeamName string   `json:"team_name" validate:"required"`
	UserIDs  []string `json:"user_ids"`
}

type ReassignmentResult struct {
	PullRequestID string `json:"pull_request_id"`
	OldReviewerID string `json:"old_reviewer_id"`
	NewReviewerID string `json:"new_reviewer_id,omitempty"`
	Error         string `json:"error,omitempty"`
}

type DeactivateTeamMembersResponse struct {
	DeactivatedUsers    []string             `json:"deactivated_user_ids"`
	SuccessfulReassigns []ReassignmentResult `json:"successful_reassignments"`
	FailedReassigns     []ReassignmentResult `json:"failed_reassignments"`
}

type ReviewerAssignment struct {
	PullRequestID string   `json:"pull_request_id"`
	AuthorID      string   `json:"author_id"`
	OldReviewerID string   `json:"old_reviewer_id"`
	Reviewers     []string `json:"reviewers"`
}
