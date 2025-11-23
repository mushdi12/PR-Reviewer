package core

import "context"

type TeamStore interface {
	Create(ctx context.Context, team *Team) error
	GetByName(ctx context.Context, name string) (*Team, error)
}

type UserStore interface {
	GetByID(ctx context.Context, id string) (*User, error)
	Update(ctx context.Context, user *User) error
	GetActiveByTeamName(ctx context.Context, teamName string) ([]*User, error)
}

type PRStore interface {
	Create(ctx context.Context, pr *PullRequest) error
	GetByID(ctx context.Context, id string) (*PullRequest, error)
	Update(ctx context.Context, pr *PullRequest) error
	GetByReviewerID(ctx context.Context, userID string) ([]*PullRequest, error)
	GetStatistics(ctx context.Context) (map[string]int, error)
}
