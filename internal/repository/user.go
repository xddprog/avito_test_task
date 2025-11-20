package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/xddprog/avito-test-task/internal/entity"
)

type UserRepository interface {
	GetByID(ctx context.Context, userID string) (*entity.User, error)
	GetActiveByTeamID(ctx context.Context, teamName string) ([]entity.User, error)
	UpdateActivity(ctx context.Context, userID string, isActive bool) (*entity.User, error)
	GetAssignedPRs(ctx context.Context, userID string) ([]entity.BasePullRequest, error)
}

type userRepo struct {
	db *pgxpool.Pool
}

func NewUserRepository(db *pgxpool.Pool) UserRepository {
	return &userRepo{db: db}
}

func (r *userRepo) GetByID(ctx context.Context, userID string) (*entity.User, error) {
	var user entity.User
	err := r.db.QueryRow(ctx, `
		SELECT id, username, is_active, team_name
		FROM users
		WHERE id = $1
	`, userID).Scan(&user.ID, &user.Username, &user.IsActive, &user.TeamName)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, entity.ErrNotFound
		}
		return nil, err
	}
	return &user, nil
}

func (r *userRepo) GetActiveByTeamID(ctx context.Context, teamName string) ([]entity.User, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, username, is_active, team_name
		FROM users
		WHERE team_name = $1 AND is_active = true
	`, teamName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []entity.User
	for rows.Next() {
		var user entity.User
		if err := rows.Scan(&user.ID, &user.Username, &user.IsActive, &user.TeamName); err != nil {
			return nil, err
		}
		users = append(users, user)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	if users == nil {
		users = []entity.User{}
	}

	return users, nil
}

func (r *userRepo) UpdateActivity(ctx context.Context, userID string, isActive bool) (*entity.User, error) {
	var user entity.User

	query := `
		UPDATE users 
		SET is_active = $1 
		WHERE id = $2 
		RETURNING id, username, is_active, team_name
	`

	err := r.db.QueryRow(ctx, query, isActive, userID).Scan(
		&user.ID, &user.Username, &user.IsActive, &user.TeamName,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, entity.ErrNotFound
		}
		return nil, err
	}

	return &user, nil
}

func (r *userRepo) GetAssignedPRs(ctx context.Context, userID string) ([]entity.BasePullRequest, error) {
	query := `
		SELECT pr.id, pr.name, pr.author_id, pr.status
		FROM pull_requests pr
		JOIN pr_reviewers r ON pr.id = r.pr_id
		WHERE r.user_id = $1
	`

	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var prs []entity.BasePullRequest
	for rows.Next() {
		var pr entity.BasePullRequest
		if err := rows.Scan(&pr.ID, &pr.Name, &pr.AuthorID, &pr.Status); err != nil {
			return nil, err
		}
		prs = append(prs, pr)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	if prs == nil {
		prs = []entity.BasePullRequest{}
	}

	return prs, nil
}
