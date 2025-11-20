package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/xddprog/avito-test-task/internal/entity"
)

type PullRequestRepository interface {
	Create(ctx context.Context, pr *entity.PullRequest) error
	GetByID(ctx context.Context, id string) (*entity.PullRequest, error)
	Merge(ctx context.Context, id string) (*entity.PullRequest, error)
	Reassign(ctx context.Context, prID, oldUserID, newUserID string) error
}

type prRepo struct {
	db *pgxpool.Pool
}

func NewPullRequestRepository(db *pgxpool.Pool) PullRequestRepository {
	return &prRepo{db: db}
}

func (r *prRepo) Create(ctx context.Context, pr *entity.PullRequest) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `
		INSERT INTO pull_requests (id, name, author_id, status, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`, pr.ID, pr.Name, pr.AuthorID, entity.StatusOpen, time.Now())

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation {
			return entity.ErrPRExists
		}
		if errors.As(err, &pgErr) && (pgErr.Code == pgerrcode.ForeignKeyViolation || pgErr.ConstraintName == "fk_author") {
			return entity.ErrNotFound
		}
		return err
	}

	if len(pr.Reviewers) > 0 {
		for _, reviewerID := range pr.Reviewers {
			_, err := tx.Exec(ctx, `
				INSERT INTO pr_reviewers (pr_id, user_id) VALUES ($1, $2)
			`, pr.ID, reviewerID)
			if err != nil {
				return fmt.Errorf("failed to add reviewer %s: %w", reviewerID, err)
			}
		}
	}

	return tx.Commit(ctx)
}

func (r *prRepo) GetByID(ctx context.Context, id string) (*entity.PullRequest, error) {
	var pr entity.PullRequest
	err := r.db.QueryRow(ctx, `
		SELECT id, name, author_id, status, created_at, merged_at 
		FROM pull_requests WHERE id = $1
	`, id).Scan(&pr.ID, &pr.Name, &pr.AuthorID, &pr.Status, &pr.CreatedAt, &pr.MergedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, entity.ErrNotFound
		}
		return nil, err
	}

	rows, err := r.db.Query(ctx, `SELECT user_id FROM pr_reviewers WHERE pr_id = $1`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var uID string
		if err := rows.Scan(&uID); err != nil {
			return nil, err
		}
		pr.Reviewers = append(pr.Reviewers, uID)
	}
	if pr.Reviewers == nil {
		pr.Reviewers = []string{}
	}

	return &pr, nil
}

func (r *prRepo) Merge(ctx context.Context, id string) (*entity.PullRequest, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	pr, err := r.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if pr.Status == entity.StatusMerged {
		return pr, nil
	}

	now := time.Now()
	_, err = tx.Exec(ctx, `
		UPDATE pull_requests 
		SET status = $1, merged_at = $2
		WHERE id = $3
	`, entity.StatusMerged, now, id)

	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	pr.Status = entity.StatusMerged
	pr.MergedAt = &now
	return pr, nil
}

func (r *prRepo) Reassign(ctx context.Context, prID, oldUserID, newUserID string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	tag, err := tx.Exec(ctx, `DELETE FROM pr_reviewers WHERE pr_id = $1 AND user_id = $2`, prID, oldUserID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return entity.ErrNotAssigned
	}

	_, err = tx.Exec(ctx, `INSERT INTO pr_reviewers (pr_id, user_id) VALUES ($1, $2)`, prID, newUserID)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}
