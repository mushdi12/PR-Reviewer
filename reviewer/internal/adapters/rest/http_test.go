package rest_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"pr-reviewer/internal/adapters/db"
	"pr-reviewer/internal/adapters/rest"
	"pr-reviewer/internal/core"
)

func setupTestDB(t *testing.T) *db.DB {
	dbAddress := os.Getenv("TEST_DB_ADDRESS")
	if dbAddress == "" {
		dbAddress = "postgres://postgres:postgres@localhost:5432/pr_reviewer?sslmode=disable"
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	storage, err := db.New(logger, dbAddress)
	if err != nil {
		t.Fatalf("failed to connect to test db: %v", err)
	}

	if err := storage.Migrate(); err != nil {
		t.Fatalf("failed to migrate test db: %v", err)
	}

	cleanupDB(t, storage)

	return storage
}

func cleanupDB(_ *testing.T, storage *db.DB) {
	ctx := context.Background()
	_, _ = storage.Conn().ExecContext(ctx, "TRUNCATE TABLE pull_request_reviewers, pull_requests, users, teams CASCADE")
}

func TestCreateTeam_Integration(t *testing.T) {
	storage := setupTestDB(t)
	defer storage.Close()

	service := core.NewService(storage.Team, storage.User, storage.PR)
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

	handler := rest.CreateTeamHandler(logger, service)

	teamData := rest.TeamDTO{
		TeamName: "test-team",
		Members: []rest.TeamMemberDTO{
			{UserID: "u1", Username: "Alice", IsActive: true},
			{UserID: "u2", Username: "Bob", IsActive: true},
		},
	}

	body, _ := json.Marshal(teamData)
	req := httptest.NewRequest(http.MethodPost, "/team/add", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if team, ok := response["team"].(map[string]interface{}); ok {
		if team["team_name"] != "test-team" {
			t.Errorf("expected team_name 'test-team', got %v", team["team_name"])
		}
	} else {
		t.Error("response should contain 'team' field")
	}
}

func TestCreatePR_Integration(t *testing.T) {
	storage := setupTestDB(t)
	defer storage.Close()

	service := core.NewService(storage.Team, storage.User, storage.PR)
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

	// Сначала создаем команду
	teamHandler := rest.CreateTeamHandler(logger, service)
	teamData := rest.TeamDTO{
		TeamName: "backend",
		Members: []rest.TeamMemberDTO{
			{UserID: "u1", Username: "Alice", IsActive: true},
			{UserID: "u2", Username: "Bob", IsActive: true},
			{UserID: "u3", Username: "Charlie", IsActive: true},
		},
	}
	body, _ := json.Marshal(teamData)
	req := httptest.NewRequest(http.MethodPost, "/team/add", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	teamHandler(w, req)

	prHandler := rest.CreatePRHandler(logger, service)
	prData := rest.CreatePRDTO{
		PullRequestID:   "pr-1",
		PullRequestName: "Add feature",
		AuthorID:        "u1",
	}
	body, _ = json.Marshal(prData)
	req = httptest.NewRequest(http.MethodPost, "/pullRequest/create", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()

	prHandler(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d, body: %s", w.Code, w.Body.String())
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if pr, ok := response["pr"].(map[string]interface{}); ok {
		if pr["pull_request_id"] != "pr-1" {
			t.Errorf("expected pull_request_id 'pr-1', got %v", pr["pull_request_id"])
		}

		if reviewers, ok := pr["assigned_reviewers"].([]interface{}); ok {
			if len(reviewers) == 0 {
				t.Error("expected at least one reviewer to be assigned")
			}
		}
	} else {
		t.Error("response should contain 'pr' field")
	}
}

func TestReassignReviewer_E2E(t *testing.T) {
	storage := setupTestDB(t)
	defer storage.Close()

	service := core.NewService(storage.Team, storage.User, storage.PR)

	ctx := context.Background()

	err := service.CreateTeam(ctx, "backend", []core.User{
		{ID: "u1", Username: "Alice", TeamName: "backend", IsActive: true},
		{ID: "u2", Username: "Bob", TeamName: "backend", IsActive: true},
		{ID: "u3", Username: "Charlie", TeamName: "backend", IsActive: true},
	})
	if err != nil {
		t.Fatalf("failed to create team: %v", err)
	}

	pr, err := service.CreatePR(ctx, "pr-1", "Add feature", "u1")
	if err != nil {
		t.Fatalf("failed to create PR: %v", err)
	}

	if len(pr.ReviewersIDs) == 0 {
		t.Fatal("expected at least one reviewer to be assigned")
	}

	oldReviewerID := pr.ReviewersIDs[0]

	updatedPR, newReviewerID, err := service.ReassignReviewer(ctx, "pr-1", oldReviewerID)
	if err != nil {
		t.Fatalf("failed to reassign reviewer: %v", err)
	}

	if newReviewerID == oldReviewerID {
		t.Error("new reviewer should be different from old reviewer")
	}

	found := false
	for _, reviewerID := range updatedPR.ReviewersIDs {
		if reviewerID == newReviewerID {
			found = true
			break
		}
	}
	if !found {
		t.Error("new reviewer should be in the reviewers list")
	}

	for _, reviewerID := range updatedPR.ReviewersIDs {
		if reviewerID == oldReviewerID {
			t.Error("old reviewer should not be in the reviewers list")
		}
	}
}

func TestMergePR_Integration(t *testing.T) {
	storage := setupTestDB(t)
	defer storage.Close()

	service := core.NewService(storage.Team, storage.User, storage.PR)
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

	ctx := context.Background()

	err := service.CreateTeam(ctx, "backend", []core.User{
		{ID: "u1", Username: "Alice", TeamName: "backend", IsActive: true},
		{ID: "u2", Username: "Bob", TeamName: "backend", IsActive: true},
	})
	if err != nil {
		t.Fatalf("failed to create team: %v", err)
	}

	_, err = service.CreatePR(ctx, "pr-1", "Add feature", "u1")
	if err != nil {
		t.Fatalf("failed to create PR: %v", err)
	}

	mergeHandler := rest.MergePRHandler(logger, service)
	mergeData := rest.MergePRDTO{
		PullRequestID: "pr-1",
	}
	body, _ := json.Marshal(mergeData)
	req := httptest.NewRequest(http.MethodPost, "/pullRequest/merge", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	mergeHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d, body: %s", w.Code, w.Body.String())
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if pr, ok := response["pr"].(map[string]interface{}); ok {
		if pr["status"] != "MERGED" {
			t.Errorf("expected status 'MERGED', got %v", pr["status"])
		}
	} else {
		t.Error("response should contain 'pr' field")
	}

	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodPost, "/pullRequest/merge", bytes.NewReader(body))
	req2.Header.Set("Content-Type", "application/json")
	mergeHandler(w2, req2)

	if w2.Code != http.StatusOK {
		t.Errorf("expected status 200 on second merge (idempotency), got %d", w2.Code)
	}
}

func TestReassignReviewer_Integration(t *testing.T) {
	storage := setupTestDB(t)
	defer storage.Close()

	service := core.NewService(storage.Team, storage.User, storage.PR)
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

	ctx := context.Background()

	err := service.CreateTeam(ctx, "backend", []core.User{
		{ID: "u1", Username: "Alice", TeamName: "backend", IsActive: true},
		{ID: "u2", Username: "Bob", TeamName: "backend", IsActive: true},
		{ID: "u3", Username: "Charlie", TeamName: "backend", IsActive: true},
		{ID: "u4", Username: "David", TeamName: "backend", IsActive: true},
	})
	if err != nil {
		t.Fatalf("failed to create team: %v", err)
	}

	createPRHandler := rest.CreatePRHandler(logger, service)
	prData := rest.CreatePRDTO{
		PullRequestID:   "pr-1",
		PullRequestName: "Add feature",
		AuthorID:        "u1",
	}
	body, _ := json.Marshal(prData)
	req := httptest.NewRequest(http.MethodPost, "/pullRequest/create", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	createPRHandler(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("failed to create PR: status %d, body: %s", w.Code, w.Body.String())
	}

	var createResponse map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &createResponse); err != nil {
		t.Fatalf("failed to unmarshal create response: %v", err)
	}

	pr, ok := createResponse["pr"].(map[string]interface{})
	if !ok {
		t.Fatal("response should contain 'pr' field")
	}

	reviewers, ok := pr["assigned_reviewers"].([]interface{})
	if !ok || len(reviewers) == 0 {
		t.Fatal("PR should have at least one reviewer")
	}

	oldReviewerID := reviewers[0].(string)

	reassignHandler := rest.ReassignReviewerHandler(logger, service)
	reassignData := rest.ReassignReviewerDTO{
		PullRequestID: "pr-1",
		OldUserID:     oldReviewerID,
	}
	body, _ = json.Marshal(reassignData)
	req = httptest.NewRequest(http.MethodPost, "/pullRequest/reassign", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()

	reassignHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d, body: %s", w.Code, w.Body.String())
	}

	var reassignResponse rest.ReassignReviewerResponseDTO
	if err := json.Unmarshal(w.Body.Bytes(), &reassignResponse); err != nil {
		t.Fatalf("failed to unmarshal reassign response: %v", err)
	}

	if reassignResponse.ReplacedBy == oldReviewerID {
		t.Error("new reviewer should be different from old reviewer")
	}

	if reassignResponse.PR.Status != "OPEN" {
		t.Errorf("PR status should remain OPEN after reassign, got %s", reassignResponse.PR.Status)
	}

	found := false
	for _, reviewerID := range reassignResponse.PR.AssignedReviewers {
		if reviewerID == reassignResponse.ReplacedBy {
			found = true
			break
		}
	}
	if !found {
		t.Error("new reviewer should be in the assigned reviewers list")
	}
}

func TestGetStatistics_Integration(t *testing.T) {
	storage := setupTestDB(t)
	defer storage.Close()

	service := core.NewService(storage.Team, storage.User, storage.PR)
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

	ctx := context.Background()

	err := service.CreateTeam(ctx, "backend", []core.User{
		{ID: "u1", Username: "Alice", TeamName: "backend", IsActive: true},
		{ID: "u2", Username: "Bob", TeamName: "backend", IsActive: true},
		{ID: "u3", Username: "Charlie", TeamName: "backend", IsActive: true},
	})
	if err != nil {
		t.Fatalf("failed to create team: %v", err)
	}

	_, err = service.CreatePR(ctx, "pr-1", "Feature 1", "u1")
	if err != nil {
		t.Fatalf("failed to create PR 1: %v", err)
	}

	_, err = service.CreatePR(ctx, "pr-2", "Feature 2", "u1")
	if err != nil {
		t.Fatalf("failed to create PR 2: %v", err)
	}

	statsHandler := rest.GetStatisticsHandler(logger, service)
	req := httptest.NewRequest(http.MethodGet, "/statistics", http.NoBody)
	w := httptest.NewRecorder()

	statsHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d, body: %s", w.Code, w.Body.String())
	}

	var statsResponse rest.StatisticsResponseDTO
	if err := json.Unmarshal(w.Body.Bytes(), &statsResponse); err != nil {
		t.Fatalf("failed to unmarshal statistics response: %v", err)
	}

	if statsResponse.TotalAssignments < 2 {
		t.Errorf("expected at least 2 total assignments (2 PRs with reviewers), got %d", statsResponse.TotalAssignments)
	}

	if len(statsResponse.ByUsers) == 0 {
		t.Error("expected at least one user in statistics")
	}
}
