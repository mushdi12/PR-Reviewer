package core

type TeamRepository interface {
	Create(team *Team) error
	GetByName(name string) (*Team, error)
}

type UserRepository interface {
	GetByID(id string) (*User, error)
	Update(user *User) error
	GetActiveByTeamName(teamName string) ([]*User, error)
}

type PRRepository interface {
	Create(pr *PullRequest) error
	GetByID(id string) (*PullRequest, error)
	Update(pr *PullRequest) error
	GetByReviewerID(userID string) ([]*PullRequest, error)
}
