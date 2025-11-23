package rest

import (
	"errors"
	"fmt"

	"pr-reviewer/internal/core"
)

var (
	ErrInvalidTeam      = errors.New("invalid team: team is nil or name is empty")
	ErrInvalidTeamDTO   = errors.New("invalid team DTO: team_name is required")
	ErrInvalidUser      = errors.New("invalid user: user is nil or required fields are empty")
	ErrInvalidUserDTO   = errors.New("invalid user DTO: user_id and username are required")
	ErrInvalidPR        = errors.New("invalid pull request: PR is nil or required fields are empty")
	ErrInvalidStatus    = errors.New("invalid status: must be OPEN or MERGED")
	ErrInvalidMemberDTO = errors.New("invalid team member: user_id and username are required")
)

func teamToDTO(team *core.Team) (TeamDTO, error) {
	if team == nil {
		return TeamDTO{}, ErrInvalidTeam
	}
	if team.Name == "" {
		return TeamDTO{}, fmt.Errorf("%w: team name is empty", ErrInvalidTeam)
	}

	members := make([]TeamMemberDTO, len(team.Members))
	for i, member := range team.Members {
		members[i] = TeamMemberDTO{
			UserID:   member.ID,
			Username: member.Username,
			IsActive: member.IsActive,
		}
	}
	return TeamDTO{
		TeamName: team.Name,
		Members:  members,
	}, nil
}

func teamFromDTO(dto TeamDTO) (*core.Team, error) {
	if dto.TeamName == "" {
		return nil, ErrInvalidTeamDTO
	}

	members := make([]core.User, len(dto.Members))
	for i, member := range dto.Members {
		if member.UserID == "" || member.Username == "" {
			return nil, fmt.Errorf("%w: member at index %d", ErrInvalidMemberDTO, i)
		}
		members[i] = core.User{
			ID:       member.UserID,
			Username: member.Username,
			TeamName: dto.TeamName,
			IsActive: member.IsActive,
		}
	}
	return &core.Team{
		Name:    dto.TeamName,
		Members: members,
	}, nil
}

func userToDTO(user *core.User) (UserDTO, error) {
	if user == nil {
		return UserDTO{}, ErrInvalidUser
	}
	if user.ID == "" || user.Username == "" {
		return UserDTO{}, fmt.Errorf("%w: user_id and username are required", ErrInvalidUser)
	}

	return UserDTO{
		UserID:   user.ID,
		Username: user.Username,
		TeamName: user.TeamName,
		IsActive: user.IsActive,
	}, nil
}

func prToDTO(pr *core.PullRequest) (PullRequestDTO, error) {
	if pr == nil {
		return PullRequestDTO{}, ErrInvalidPR
	}
	if pr.ID == "" || pr.Name == "" || pr.AuthorID == "" {
		return PullRequestDTO{}, fmt.Errorf("%w: pull_request_id, pull_request_name and author_id are required", ErrInvalidPR)
	}

	status := string(pr.Status)
	if status != string(core.PullRequestStatusOpen) && status != string(core.PullRequestStatusMerged) {
		return PullRequestDTO{}, ErrInvalidStatus
	}

	return PullRequestDTO{
		PullRequestID:     pr.ID,
		PullRequestName:   pr.Name,
		AuthorID:          pr.AuthorID,
		Status:            status,
		AssignedReviewers: pr.ReviewersIDs,
	}, nil
}

func prToShortDTO(pr *core.PullRequest) (PullRequestShortDTO, error) {
	if pr == nil {
		return PullRequestShortDTO{}, ErrInvalidPR
	}
	if pr.ID == "" || pr.Name == "" || pr.AuthorID == "" {
		return PullRequestShortDTO{}, fmt.Errorf("%w: pull_request_id, pull_request_name and author_id are required", ErrInvalidPR)
	}

	status := string(pr.Status)
	if status != string(core.PullRequestStatusOpen) && status != string(core.PullRequestStatusMerged) {
		return PullRequestShortDTO{}, ErrInvalidStatus
	}

	return PullRequestShortDTO{
		PullRequestID:   pr.ID,
		PullRequestName: pr.Name,
		AuthorID:        pr.AuthorID,
		Status:          status,
	}, nil
}

func prsToShortDTOs(prs []*core.PullRequest) ([]PullRequestShortDTO, error) {
	if prs == nil {
		return nil, errors.New("prs slice is nil")
	}

	result := make([]PullRequestShortDTO, len(prs))
	for i, pr := range prs {
		dto, err := prToShortDTO(pr)
		if err != nil {
			return nil, fmt.Errorf("failed to convert PR at index %d: %w", i, err)
		}
		result[i] = dto
	}
	return result, nil
}

func statisticsToDTO(stats map[string]int) StatisticsResponseDTO {
	byUsers := make([]UserStatisticDTO, 0, len(stats))
	totalAssignments := 0

	for userID, count := range stats {
		byUsers = append(byUsers, UserStatisticDTO{
			UserID:           userID,
			AssignmentsCount: count,
		})
		totalAssignments += count
	}

	return StatisticsResponseDTO{
		ByUsers:          byUsers,
		TotalAssignments: totalAssignments,
	}
}

func mapErrorToCode(err error) (string, bool) {
	switch {
	case errors.Is(err, core.ErrTeamExists):
		return "TEAM_EXISTS", true
	case errors.Is(err, core.ErrPRExists):
		return "PR_EXISTS", true
	case errors.Is(err, core.ErrPRMerged):
		return "PR_MERGED", true
	case errors.Is(err, core.ErrNotAssigned):
		return "NOT_ASSIGNED", true
	case errors.Is(err, core.ErrNoCandidate):
		return "NO_CANDIDATE", true
	case errors.Is(err, core.ErrNotFound):
		return "NOT_FOUND", true
	default:
		return "", false
	}
}
