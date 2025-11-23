package db

import (
	"context"
	"database/sql"
	"errors"

	"pr-reviewer/internal/core"
)

type UserRepository struct {
	db *DB
}

func NewUserRepository(database *DB) *UserRepository {
	return &UserRepository{db: database}
}

func (r *UserRepository) GetByID(ctx context.Context, id string) (*core.User, error) {
	var row userRow
	err := r.db.conn.GetContext(ctx, &row, "SELECT id, username, team_name, is_active FROM users WHERE id = $1", id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, core.ErrNotFound
		}
		return nil, err
	}
	return row.toCoreUser(), nil
}

func (r *UserRepository) Update(ctx context.Context, user *core.User) error {
	_, err := r.db.conn.ExecContext(ctx,
		"UPDATE users SET username = $1, team_name = $2, is_active = $3 WHERE id = $4",
		user.Username, user.TeamName, user.IsActive, user.ID,
	)
	return err
}

func (r *UserRepository) GetActiveByTeamName(ctx context.Context, teamName string) ([]*core.User, error) {
	var rows []userRow
	err := r.db.conn.SelectContext(ctx, &rows,
		"SELECT id, username, team_name, is_active FROM users WHERE team_name = $1 AND is_active = true",
		teamName,
	)
	if err != nil {
		return nil, err
	}

	result := make([]*core.User, len(rows))
	for i, row := range rows {
		result[i] = row.toCoreUser()
	}
	return result, nil
}
