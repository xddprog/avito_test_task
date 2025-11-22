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
	GetOpenAssignmentsForUsers(ctx context.Context, userIDs []string) ([]entity.ReviewerAssignment, error)
	ApplyReviewerReplacements(ctx context.Context, replacements []entity.ReassignmentResult) error
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
	defer func() { _ = tx.Rollback(ctx) }()

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
	defer func() { _ = tx.Rollback(ctx) }()

	pr, err := r.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if pr.Status == entity.StatusMerged {
		return pr, nil
	}

	now := time.Now()
	var mergedAt *time.Time
	err = tx.QueryRow(ctx, `
		UPDATE pull_requests 
		SET status = $1, merged_at = $2
		WHERE id = $3
		RETURNING merged_at
	`, entity.StatusMerged, now, id).Scan(&mergedAt)

	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	pr.Status = entity.StatusMerged
	pr.MergedAt = mergedAt
	return pr, nil
}

func (r *prRepo) Reassign(ctx context.Context, prID, oldUserID, newUserID string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

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

func (r *prRepo) GetOpenAssignmentsForUsers(ctx context.Context, userIDs []string) ([]entity.ReviewerAssignment, error) {
	if len(userIDs) == 0 {
		return []entity.ReviewerAssignment{}, nil
	}

	rows, err := r.db.Query(ctx, `
		SELECT
			pr.id,
			pr.author_id,
			prr.user_id AS old_reviewer_id,
			(
				SELECT ARRAY_AGG(prr2.user_id)
				FROM pr_reviewers prr2
				WHERE prr2.pr_id = pr.id
			) AS reviewers
		FROM pull_requests pr
		JOIN pr_reviewers prr ON pr.id = prr.pr_id
		WHERE pr.status = $1
		  AND prr.user_id = ANY($2)
	`, entity.StatusOpen, userIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var assignments []entity.ReviewerAssignment
	for rows.Next() {
		var assignment entity.ReviewerAssignment
		if err := rows.Scan(
			&assignment.PullRequestID,
			&assignment.AuthorID,
			&assignment.OldReviewerID,
			&assignment.Reviewers,
		); err != nil {
			return nil, err
		}
		assignments = append(assignments, assignment)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if assignments == nil {
		assignments = []entity.ReviewerAssignment{}
	}
	return assignments, nil
}

func (r *prRepo) ApplyReviewerReplacements(ctx context.Context, replacements []entity.ReassignmentResult) error {
	if len(replacements) == 0 {
		return nil
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	batch := &pgx.Batch{}
	for _, repl := range replacements {
		batch.Queue(`
			DELETE FROM pr_reviewers
			WHERE pr_id = $1 AND user_id = $2
		`, repl.PullRequestID, repl.OldReviewerID)

		batch.Queue(`
			INSERT INTO pr_reviewers (pr_id, user_id)
			VALUES ($1, $2)
			ON CONFLICT DO NOTHING
		`, repl.PullRequestID, repl.NewReviewerID)
	}

	br := tx.SendBatch(ctx, batch)
	for range replacements {
		if _, err := br.Exec(); err != nil {
			br.Close()
			return err
		}
	}
	if err := br.Close(); err != nil {
		return err
	}

	return tx.Commit(ctx)
}
