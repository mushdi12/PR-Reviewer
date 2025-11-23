package db

import (
	"context"
	"database/sql"
	"errors"

	"pr-reviewer/internal/core"
)

type TeamRepository struct {
	db *DB
}

func NewTeamRepository(database *DB) *TeamRepository {
	return &TeamRepository{db: database}
}

func (r *TeamRepository) Create(ctx context.Context, team *core.Team) error {
	tx, err := r.db.conn.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	committed := false
	defer func() {
		if !committed {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				r.db.log.Error("failed to rollback transaction", "error", rollbackErr)
			}
		}
	}()

	_, err = tx.ExecContext(ctx, "INSERT INTO teams (name) VALUES ($1) ON CONFLICT (name) DO NOTHING", team.Name)
	if err != nil {
		return err
	}

	for _, member := range team.Members {
		_, err = tx.ExecContext(ctx, `
			INSERT INTO users (id, username, team_name, is_active) 
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (id) 
			DO UPDATE SET username = $2, team_name = $3, is_active = $4
		`, member.ID, member.Username, team.Name, member.IsActive)
		if err != nil {
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		return err
	}
	committed = true
	return nil
}

func (r *TeamRepository) GetByName(ctx context.Context, name string) (*core.Team, error) {
	var teamName string
	err := r.db.conn.GetContext(ctx, &teamName, "SELECT name FROM teams WHERE name = $1", name)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, core.ErrNotFound
		}
		return nil, err
	}

	var rows []userRow
	err = r.db.conn.SelectContext(ctx, &rows, "SELECT id, username, team_name, is_active FROM users WHERE team_name = $1", name)
	if err != nil {
		return nil, err
	}

	members := make([]core.User, len(rows))
	for i, row := range rows {
		members[i] = *row.toCoreUser()
	}

	return &core.Team{
		Name:    teamName,
		Members: members,
	}, nil
}
