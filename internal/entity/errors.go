package entity

import "net/http"

type AppError struct {
	Code    int
	Message string
	SafeCode string 
}

func (e *AppError) Error() string {
	return e.Message
}

func New(code int, safeCode string, message string) *AppError {
	return &AppError{
		Code:     code,
		SafeCode: safeCode,
		Message:  message,
	}
}


var (
	ErrNotFound = &AppError{
		Code:     http.StatusNotFound,
		SafeCode: "NOT_FOUND",
		Message:  "resource not found",
	}
	
	ErrInternal = &AppError{
		Code:     http.StatusInternalServerError,
		SafeCode: "INTERNAL_ERROR",
		Message:  "internal server error",
	}

	ErrBadRequest = &AppError{
		Code:     http.StatusBadRequest,
		SafeCode: "BAD_REQUEST",
		Message:  "invalid request parameters",
	}

	ErrTeamExists = &AppError{
		Code:     http.StatusBadRequest, 
		SafeCode: "TEAM_EXISTS",
		Message:  "team name already exists",
	}

	ErrPRExists = &AppError{
		Code:     http.StatusConflict,
		SafeCode: "PR_EXISTS",
		Message:  "pull request already exists",
	}

	ErrPRMerged = &AppError{
		Code:     http.StatusConflict,
		SafeCode: "PR_MERGED",
		Message:  "cannot reassign on merged PR",
	}

	ErrNotAssigned = &AppError{
		Code:     http.StatusConflict,
		SafeCode: "NOT_ASSIGNED",
		Message:  "reviewer is not assigned to this PR",
	}

	ErrNoCandidate = &AppError{
		Code:     http.StatusConflict,
		SafeCode: "NO_CANDIDATE",
		Message:  "no active replacement candidate in team",
	}

	ErrNotFoundAuthor = &AppError{
		
	}
)