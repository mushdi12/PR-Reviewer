package core

type PullRequestStatus string

const (
	PullRequestStatusOpen   PullRequestStatus = "OPEN"
	PullRequestStatusMerged PullRequestStatus = "MERGED"
)

type User struct {
	ID       string
	Username string
	TeamName string
	IsActive bool
}

func (u *User) CanBeReviewer() bool {
	return u.IsActive
}

type Team struct {
	Name    string
	Members []User
}

type PullRequest struct {
	ID           string
	Name         string
	Status       PullRequestStatus
	AuthorID     string
	ReviewersIDs []string
}

func (pr *PullRequest) CanReassign() bool {
	return pr.Status == PullRequestStatusOpen
}

func (pr *PullRequest) IsMerged() bool {
	return pr.Status == PullRequestStatusMerged
}

func (pr *PullRequest) HasReviewer(userID string) bool {
	for _, reviewerID := range pr.ReviewersIDs {
		if reviewerID == userID {
			return true
		}
	}
	return false
}
