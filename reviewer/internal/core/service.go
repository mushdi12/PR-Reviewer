package core

import (
	"context"
	"errors"
	"math/rand"
)

type Service struct {
	teamStore TeamStore
	userStore UserStore
	prStore   PRStore
}

func NewService(teamStore TeamStore, userStore UserStore, prStore PRStore) *Service {
	return &Service{
		teamStore: teamStore,
		userStore: userStore,
		prStore:   prStore,
	}
}

func (s *Service) CreateTeam(ctx context.Context, name string, members []User) error {
	existing, err := s.teamStore.GetByName(ctx, name)
	if err == nil && existing != nil {
		return ErrTeamExists
	}

	return s.teamStore.Create(ctx, &Team{Name: name, Members: members})
}

func (s *Service) GetTeam(ctx context.Context, name string) (*Team, error) {
	team, err := s.teamStore.GetByName(ctx, name)
	if err != nil {
		return nil, err
	}
	return team, nil
}

func (s *Service) SetUserActive(ctx context.Context, userID string, isActive bool) (*User, error) {
	user, err := s.userStore.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	user.IsActive = isActive
	if err := s.userStore.Update(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}

func (s *Service) CreatePR(ctx context.Context, prID, name, authorID string) (*PullRequest, error) {
	existing, err := s.prStore.GetByID(ctx, prID)
	if err == nil && existing != nil {
		return nil, ErrPRExists
	}

	if err != nil && !errors.Is(err, ErrNotFound) {
		return nil, err
	}

	author, err := s.userStore.GetByID(ctx, authorID)
	if err != nil {
		return nil, err
	}

	candidates, err := s.userStore.GetActiveByTeamName(ctx, author.TeamName)
	if err != nil {
		return nil, err
	}

	availableCandidates := make([]*User, 0)
	for _, candidate := range candidates {

		if candidate.ID != authorID && candidate.CanBeReviewer() {
			availableCandidates = append(availableCandidates, candidate)
		}
	}

	reviewerIDs := selectRandomReviewers(availableCandidates, 2)

	pr := &PullRequest{
		ID:           prID,
		Name:         name,
		Status:       PullRequestStatusOpen,
		AuthorID:     authorID,
		ReviewersIDs: reviewerIDs,
	}

	if err := s.prStore.Create(ctx, pr); err != nil {
		return nil, err
	}

	return pr, nil
}

func (s *Service) MergePR(ctx context.Context, prID string) (*PullRequest, error) {
	pr, err := s.prStore.GetByID(ctx, prID)
	if err != nil {
		return nil, err
	}

	if pr.IsMerged() {
		return pr, nil
	}

	pr.Status = PullRequestStatusMerged
	if err := s.prStore.Update(ctx, pr); err != nil {
		return nil, err
	}

	return pr, nil
}

func (s *Service) ReassignReviewer(ctx context.Context, prID, oldReviewerID string) (*PullRequest, string, error) {
	pr, err := s.prStore.GetByID(ctx, prID)
	if err != nil {
		return nil, "", err
	}

	if !pr.CanReassign() {
		return nil, "", ErrPRMerged
	}

	if !pr.HasReviewer(oldReviewerID) {
		return nil, "", ErrNotAssigned
	}

	oldReviewer, err := s.userStore.GetByID(ctx, oldReviewerID)
	if err != nil {
		return nil, "", ErrNotFound
	}

	candidates, err := s.userStore.GetActiveByTeamName(ctx, oldReviewer.TeamName)
	if err != nil {
		return nil, "", err
	}

	availableCandidates := make([]*User, 0)
	assignedMap := make(map[string]bool)
	for _, reviewerID := range pr.ReviewersIDs {
		assignedMap[reviewerID] = true
	}

	for _, candidate := range candidates {
		if candidate.ID != oldReviewerID && !assignedMap[candidate.ID] && candidate.CanBeReviewer() {
			availableCandidates = append(availableCandidates, candidate)
		}
	}

	if len(availableCandidates) == 0 {
		return nil, "", ErrNoCandidate
	}

	newReviewer := availableCandidates[rand.Intn(len(availableCandidates))]

	for i, reviewerID := range pr.ReviewersIDs {
		if reviewerID == oldReviewerID {
			pr.ReviewersIDs[i] = newReviewer.ID
			break
		}
	}

	if err := s.prStore.Update(ctx, pr); err != nil {
		return nil, "", err
	}

	return pr, newReviewer.ID, nil
}

func (s *Service) GetUserReviews(ctx context.Context, userID string) ([]*PullRequest, error) {
	_, err := s.userStore.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	prs, err := s.prStore.GetByReviewerID(ctx, userID)
	if err != nil {
		return nil, err
	}

	return prs, nil
}

func (s *Service) GetStatistics(ctx context.Context) (map[string]int, error) {
	return s.prStore.GetStatistics(ctx)
}

func selectRandomReviewers(candidates []*User, maxCount int) []string {
	if len(candidates) == 0 {
		return []string{}
	}

	count := maxCount
	if len(candidates) < maxCount {
		count = len(candidates)
	}

	shuffled := make([]*User, len(candidates))
	copy(shuffled, candidates)
	rand.Shuffle(len(shuffled), func(i, j int) {
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
	})

	result := make([]string, count)
	for i := 0; i < count; i++ {
		result[i] = shuffled[i].ID
	}

	return result
}
