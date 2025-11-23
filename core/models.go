package core

type PullRequestStatus string

const (
	PullRequestStatusOpen   PullRequestStatus = "OPEN"
	PullRequestStatusMerged PullRequestStatus = "MERGED"
)

type User struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	TeamName string `json:"team_name"`
	IsActive bool   `json:"is_active"`
}

type Team struct {
	Name    string `json:"name"`
	Members []User `json:"members"`
}

type PullRequest struct {
	ID        string            `json:"id"`
	Name      string            `json:"name"`
	Status    PullRequestStatus `json:"status"`
	AuthorID  string            `json:"author_id"`
	Reviewers []int             `json:"reviewers"`
}
