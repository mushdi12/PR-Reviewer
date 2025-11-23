package db

import (
	"database/sql"
	"time"

	"pr-reviewer/internal/core"
)

type userRow struct {
	ID       string `db:"id"`
	Username string `db:"username"`
	TeamName string `db:"team_name"`
	IsActive bool   `db:"is_active"`
}

func (r *userRow) toCoreUser() *core.User {
	return &core.User{
		ID:       r.ID,
		Username: r.Username,
		TeamName: r.TeamName,
		IsActive: r.IsActive,
	}
}

type prRow struct {
	ID        string       `db:"id"`
	Name      string       `db:"name"`
	AuthorID  string       `db:"author_id"`
	Status    string       `db:"status"`
	CreatedAt time.Time    `db:"created_at"`
	MergedAt  sql.NullTime `db:"merged_at"`
}

func (r *prRow) toCorePullRequest(reviewerIDs []string) *core.PullRequest {
	return &core.PullRequest{
		ID:           r.ID,
		Name:         r.Name,
		AuthorID:     r.AuthorID,
		Status:       core.PullRequestStatus(r.Status),
		ReviewersIDs: reviewerIDs,
	}
}
