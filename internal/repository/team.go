package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/xddprog/avito-test-task/internal/entity"
)

type TeamRepository interface {
	Create(ctx context.Context, team *entity.Team) error
	GetByName(ctx context.Context, name string) (*entity.Team, error)
	DeactivateMembers(ctx context.Context, teamName string, userIDs []string) ([]string, error)
}

type teamRepo struct {
	db *pgxpool.Pool
}

func NewTeamRepository(db *pgxpool.Pool) TeamRepository {
	return &teamRepo{
		db: db,
	}
}

func (r *teamRepo) Create(ctx context.Context, team *entity.Team) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `INSERT INTO teams (name) VALUES ($1)`, team.Name)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation {
			return entity.ErrTeamExists
		}
		return err
	}

	if len(team.Members) > 0 {
		batch := &pgx.Batch{}
		for _, member := range team.Members {
			batch.Queue(`
				INSERT INTO users (id, username, is_active, team_name)
				VALUES ($1, $2, $3, $4)
				ON CONFLICT (id) DO UPDATE
				SET username = EXCLUDED.username,
				    is_active = EXCLUDED.is_active,
				    team_name = EXCLUDED.team_name
			`, member.ID, member.Username, member.IsActive, team.Name)
		}

		br := tx.SendBatch(ctx, batch)
		for i := 0; i < len(team.Members); i++ {
			_, err := br.Exec()
			if err != nil {
				br.Close()
				return err
			}
		}
		if err := br.Close(); err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

func (r *teamRepo) GetByName(ctx context.Context, name string) (*entity.Team, error) {
	var teamName string
	err := r.db.QueryRow(ctx, `SELECT name FROM teams WHERE name = $1`, name).Scan(&teamName)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, entity.ErrNotFound
		}
		return nil, err
	}

	rows, err := r.db.Query(
		ctx,
		`SELECT id, username, is_active, team_name FROM users WHERE team_name = $1`,
		name,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var members []entity.User
	for rows.Next() {
		var u entity.User
		if err := rows.Scan(&u.ID, &u.Username, &u.IsActive, &u.TeamName); err != nil {
			return nil, err
		}
		members = append(members, u)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return &entity.Team{
		Name:    teamName,
		Members: members,
	}, nil
}

func (r *teamRepo) DeactivateMembers(ctx context.Context, teamName string, userIDs []string) ([]string, error) {
	if teamName == "" || len(userIDs) == 0 {
		return nil, entity.ErrBadRequest
	}

	var exists bool
	if err := r.db.QueryRow(ctx, `SELECT true FROM teams WHERE name = $1`, teamName).Scan(&exists); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, entity.ErrNotFound
		}
		return nil, err
	}

	rows, err := r.db.Query(ctx, `
		UPDATE users
		SET is_active = false
		WHERE team_name = $1 AND id = ANY($2) AND is_active = true
		RETURNING id
	`, teamName, userIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var affected []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		affected = append(affected, id)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if affected == nil {
		affected = []string{}
	}
	return affected, nil
}
