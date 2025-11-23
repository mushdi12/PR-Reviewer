package db

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"pr-reviewer/internal/core"
)

type PRRepository struct {
	db *DB
}

func NewPRRepository(database *DB) *PRRepository {
	return &PRRepository{db: database}
}

func (r *PRRepository) Create(ctx context.Context, pr *core.PullRequest) error {
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

	var mergedAt sql.NullTime
	if pr.Status == core.PullRequestStatusMerged {
		mergedAt = sql.NullTime{Time: time.Now(), Valid: true}
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO pull_requests (id, name, author_id, status, created_at, merged_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, pr.ID, pr.Name, pr.AuthorID, string(pr.Status), time.Now(), mergedAt)
	if err != nil {
		return err
	}

	for _, reviewerID := range pr.ReviewersIDs {
		_, err = tx.ExecContext(ctx, `
			INSERT INTO pull_request_reviewers (pull_request_id, reviewer_id)
			VALUES ($1, $2)
		`, pr.ID, reviewerID)
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

func (r *PRRepository) GetByID(ctx context.Context, id string) (*core.PullRequest, error) {
	var row prRow
	err := r.db.conn.GetContext(ctx, &row, `
		SELECT id, name, author_id, status, created_at, merged_at
		FROM pull_requests
		WHERE id = $1
	`, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, core.ErrNotFound
		}
		return nil, err
	}

	var reviewerIDs []string
	err = r.db.conn.SelectContext(ctx, &reviewerIDs, `
		SELECT reviewer_id
		FROM pull_request_reviewers
		WHERE pull_request_id = $1
	`, id)
	if err != nil {
		return nil, err
	}

	return row.toCorePullRequest(reviewerIDs), nil
}

func (r *PRRepository) Update(ctx context.Context, pr *core.PullRequest) error {
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

	var mergedAt sql.NullTime
	if pr.Status == core.PullRequestStatusMerged {
		mergedAt = sql.NullTime{Time: time.Now(), Valid: true}
	}

	_, err = tx.ExecContext(ctx, `
		UPDATE pull_requests
		SET name = $1, status = $2, merged_at = $3
		WHERE id = $4
	`, pr.Name, string(pr.Status), mergedAt, pr.ID)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, "DELETE FROM pull_request_reviewers WHERE pull_request_id = $1", pr.ID)
	if err != nil {
		return err
	}

	for _, reviewerID := range pr.ReviewersIDs {
		_, err = tx.ExecContext(ctx, `
			INSERT INTO pull_request_reviewers (pull_request_id, reviewer_id)
			VALUES ($1, $2)
		`, pr.ID, reviewerID)
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

func (r *PRRepository) GetByReviewerID(ctx context.Context, userID string) ([]*core.PullRequest, error) {
	var rows []prRow
	err := r.db.conn.SelectContext(ctx, &rows, `
		SELECT pr.id, pr.name, pr.author_id, pr.status, pr.created_at, pr.merged_at
		FROM pull_requests pr
		INNER JOIN pull_request_reviewers prr ON pr.id = prr.pull_request_id
		WHERE prr.reviewer_id = $1
	`, userID)
	if err != nil {
		return nil, err
	}

	result := make([]*core.PullRequest, len(rows))
	for i, row := range rows {
		var reviewerIDs []string
		err = r.db.conn.SelectContext(ctx, &reviewerIDs, `
			SELECT reviewer_id
			FROM pull_request_reviewers
			WHERE pull_request_id = $1
		`, row.ID)
		if err != nil {
			return nil, err
		}

		result[i] = row.toCorePullRequest(reviewerIDs)
	}
	return result, nil
}

func (r *PRRepository) GetStatistics(ctx context.Context) (map[string]int, error) {
	var stats []struct {
		UserID string `db:"reviewer_id"`
		Count  int    `db:"count"`
	}

	err := r.db.conn.SelectContext(ctx, &stats, `
		SELECT reviewer_id, COUNT(*) as count
		FROM pull_request_reviewers
		GROUP BY reviewer_id
		ORDER BY count DESC
	`)
	if err != nil {
		return nil, err
	}

	result := make(map[string]int)
	for _, stat := range stats {
		result[stat.UserID] = stat.Count
	}

	return result, nil
}
