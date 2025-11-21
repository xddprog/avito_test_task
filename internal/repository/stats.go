package repository

import (
	"context"
	"database/sql"
	"math"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/xddprog/avito-test-task/internal/entity"
)

type StatsRepository interface {
	GetReviewerAssignments(ctx context.Context) ([]entity.ReviewerAssignmentStat, error)
	GetPRStatus(ctx context.Context) (entity.PRStatusStat, error)
	GetTeamMembers(ctx context.Context) ([]entity.TeamMemberStat, error)
	GetPRLifetime(ctx context.Context) (entity.PRLifetimeStat, error)
}

type statsRepo struct {
	db *pgxpool.Pool
}

func NewStatsRepository(db *pgxpool.Pool) StatsRepository {
	return &statsRepo{db: db}
}

func (r *statsRepo) GetReviewerAssignments(ctx context.Context) ([]entity.ReviewerAssignmentStat, error) {
	rows, err := r.db.Query(ctx, `
		SELECT user_id, COUNT(*) AS assignments
		FROM pr_reviewers
		GROUP BY user_id
		ORDER BY assignments DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []entity.ReviewerAssignmentStat
	for rows.Next() {
		var s entity.ReviewerAssignmentStat
		if err := rows.Scan(&s.UserID, &s.Assignments); err != nil {
			return nil, err
		}
		stats = append(stats, s)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if stats == nil {
		stats = []entity.ReviewerAssignmentStat{}
	}
	return stats, nil
}

func (r *statsRepo) GetPRStatus(ctx context.Context) (entity.PRStatusStat, error) {
	rows, err := r.db.Query(ctx, `
		SELECT status, COUNT(*) AS cnt
		FROM pull_requests
		GROUP BY status
	`)
	if err != nil {
		return entity.PRStatusStat{}, err
	}
	defer rows.Close()

	var stat entity.PRStatusStat
	for rows.Next() {
		var status string
		var cnt int
		if err := rows.Scan(&status, &cnt); err != nil {
			return entity.PRStatusStat{}, err
		}
		stat.Total += cnt
		if status == string(entity.StatusOpen) {
			stat.Open = cnt
		}
		if status == string(entity.StatusMerged) {
			stat.Merged = cnt
		}
	}
	if err := rows.Err(); err != nil {
		return entity.PRStatusStat{}, err
	}

	var avg sql.NullFloat64
	err = r.db.QueryRow(ctx, `
		SELECT AVG(reviewer_count)
		FROM (
			SELECT COUNT(*) AS reviewer_count
			FROM pr_reviewers
			GROUP BY pr_id
		) AS counts
	`).Scan(&avg)
	if err != nil && err != sql.ErrNoRows {
		return entity.PRStatusStat{}, err
	}
	if avg.Valid {
		stat.AverageReviewers = math.Round(avg.Float64*10) / 10
	}

	return stat, nil
}

func (r *statsRepo) GetTeamMembers(ctx context.Context) ([]entity.TeamMemberStat, error) {
	rows, err := r.db.Query(ctx, `
		SELECT team_name,
		       COUNT(*) FILTER (WHERE is_active)  AS active_members,
		       COUNT(*) FILTER (WHERE NOT is_active) AS inactive_members
		FROM users
		GROUP BY team_name
		ORDER BY team_name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []entity.TeamMemberStat
	for rows.Next() {
		var s entity.TeamMemberStat
		if err := rows.Scan(&s.TeamName, &s.Active, &s.Inactive); err != nil {
			return nil, err
		}
		stats = append(stats, s)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if stats == nil {
		stats = []entity.TeamMemberStat{}
	}
	return stats, nil
}

func (r *statsRepo) GetPRLifetime(ctx context.Context) (entity.PRLifetimeStat, error) {
	var averageSeconds float64

	err := r.db.QueryRow(ctx, `
		SELECT COALESCE(AVG(EXTRACT(EPOCH FROM (merged_at - created_at))), 0)
		FROM pull_requests
		WHERE status = $1 AND merged_at IS NOT NULL
	`, entity.StatusMerged).Scan(&averageSeconds)
	if err != nil && err != sql.ErrNoRows {
		return entity.PRLifetimeStat{}, err
	}
	lifetime := entity.PRLifetimeStat{
		AverageMerge: secondsToDuration(averageSeconds),
	}

	err = r.db.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM pull_requests
		WHERE status = $1 AND created_at <= NOW() - INTERVAL '7 days'
	`, entity.StatusOpen).Scan(&lifetime.OpenOlderThan7Days)
	if err != nil {
		return entity.PRLifetimeStat{}, err
	}

	return lifetime, nil
}

func secondsToDuration(seconds float64) entity.DurationBreakdown {
	totalMinutes := int(math.Round(seconds / 60))
	days := totalMinutes / (24 * 60)
	hours := (totalMinutes % (24 * 60)) / 60
	minutes := totalMinutes % 60
	return entity.DurationBreakdown{
		Days:    days,
		Hours:   hours,
		Minutes: minutes,
	}
}
