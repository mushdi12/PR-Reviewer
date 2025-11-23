package rest

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"pr-reviewer/internal/core"
)

// POST /team/add.
func CreateTeamHandler(log *slog.Logger, service *core.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var teamRequest TeamDTO
		if err := json.NewDecoder(r.Body).Decode(&teamRequest); err != nil {
			log.Error("failed to decode request", "error", err)
			writeError(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
			return
		}

		team, err := teamFromDTO(teamRequest)
		if err != nil {
			log.Error("failed to validate team DTO", "error", err)
			writeError(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
			return
		}

		err = service.CreateTeam(r.Context(), team.Name, team.Members)
		if err != nil {
			if errorCode, ok := mapErrorToCode(err); ok {
				log.Error("failed to create team", "error", err, "code", errorCode)
				writeError(w, http.StatusBadRequest, errorCode, err.Error())
				return
			}
			log.Error("failed to create team", "error", err)
			writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
			return
		}

		createdTeam, err := service.GetTeam(r.Context(), team.Name)
		if err != nil {
			log.Error("failed to get created team", "error", err)
			writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
			return
		}

		teamDTO, err := teamToDTO(createdTeam)
		if err != nil {
			log.Error("failed to convert team to DTO", "error", err)
			writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
			return
		}

		writeJSON(w, http.StatusCreated, map[string]interface{}{"team": teamDTO})

	}
}

// GetTeam получает команду по имени
// GET /team/get?team_name=...
func GetTeamHandler(log *slog.Logger, service *core.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		teamName := r.URL.Query().Get("team_name")
		if teamName == "" {
			log.Error("team_name is required")
			writeError(w, http.StatusBadRequest, "BAD_REQUEST", "team_name is required")
			return
		}

		team, err := service.GetTeam(r.Context(), teamName)
		if err != nil {
			if errorCode, ok := mapErrorToCode(err); ok {
				log.Error("failed to get team", "error", err, "code", errorCode)
				writeError(w, http.StatusNotFound, errorCode, err.Error())
				return
			}
			log.Error("failed to get team", "error", err)
			writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
			return
		}

		teamDTO, err := teamToDTO(team)
		if err != nil {
			log.Error("failed to convert team to DTO", "error", err)
			writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
			return
		}

		writeJSON(w, http.StatusOK, teamDTO)
	}
}

// POST /users/setIsActive.
func SetUserActiveHandler(log *slog.Logger, service *core.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req SetUserActiveDTO
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			log.Error("failed to decode request", "error", err)
			writeError(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
			return
		}

		user, err := service.SetUserActive(r.Context(), req.UserID, req.IsActive)
		if err != nil {
			if errorCode, ok := mapErrorToCode(err); ok {
				log.Error("failed to set user active", "error", err, "code", errorCode)
				writeError(w, http.StatusNotFound, errorCode, err.Error())
				return
			}
			log.Error("failed to set user active", "error", err)
			writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
			return
		}
		userDTO, err := userToDTO(user)
		if err != nil {
			log.Error("failed to convert user to DTO", "error", err)
			writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
			return
		}

		writeJSON(w, http.StatusOK, map[string]interface{}{"user": userDTO})
	}
}

// POST /pullRequest/create.
func CreatePRHandler(log *slog.Logger, service *core.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req CreatePRDTO
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			log.Error("failed to decode request", "error", err)
			writeError(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
			return
		}

		pr, err := service.CreatePR(r.Context(), req.PullRequestID, req.PullRequestName, req.AuthorID)
		if err != nil {
			if errorCode, ok := mapErrorToCode(err); ok {
				statusCode := http.StatusNotFound
				if errorCode == "PR_EXISTS" {
					statusCode = http.StatusConflict
				}
				log.Error("failed to create PR", "error", err, "code", errorCode)
				writeError(w, statusCode, errorCode, err.Error())
				return
			}
			log.Error("failed to create PR", "error", err)
			writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
			return
		}

		prDTO, err := prToDTO(pr)
		if err != nil {
			log.Error("failed to convert PR to DTO", "error", err)
			writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
			return
		}

		writeJSON(w, http.StatusCreated, map[string]interface{}{"pr": prDTO})
	}
}

// POST /pullRequest/merge.
func MergePRHandler(log *slog.Logger, service *core.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req MergePRDTO
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			log.Error("failed to decode request", "error", err)
			writeError(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
			return
		}

		pr, err := service.MergePR(r.Context(), req.PullRequestID)
		if err != nil {
			if errorCode, ok := mapErrorToCode(err); ok {
				log.Error("failed to merge PR", "error", err, "code", errorCode)
				writeError(w, http.StatusNotFound, errorCode, err.Error())
				return
			}
			log.Error("failed to merge PR", "error", err)
			writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
			return
		}

		prDTO, err := prToDTO(pr)
		if err != nil {
			log.Error("failed to convert PR to DTO", "error", err)
			writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
			return
		}

		writeJSON(w, http.StatusOK, map[string]interface{}{"pr": prDTO})
	}
}

// POST /pullRequest/reassign.
func ReassignReviewerHandler(log *slog.Logger, service *core.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req ReassignReviewerDTO
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			log.Error("failed to decode request", "error", err)
			writeError(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
			return
		}

		pr, replacedBy, err := service.ReassignReviewer(r.Context(), req.PullRequestID, req.OldUserID)
		if err != nil {
			if errorCode, ok := mapErrorToCode(err); ok {
				statusCode := http.StatusNotFound
				if errorCode == "PR_MERGED" || errorCode == "NOT_ASSIGNED" || errorCode == "NO_CANDIDATE" {
					statusCode = http.StatusConflict
				}
				log.Error("failed to reassign reviewer", "error", err, "code", errorCode)
				writeError(w, statusCode, errorCode, err.Error())
				return
			}
			log.Error("failed to reassign reviewer", "error", err)
			writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
			return
		}

		prDTO, err := prToDTO(pr)
		if err != nil {
			log.Error("failed to convert PR to DTO", "error", err)
			writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
			return
		}

		response := ReassignReviewerResponseDTO{
			PR:         prDTO,
			ReplacedBy: replacedBy,
		}

		writeJSON(w, http.StatusOK, response)
	}
}

// GetUserReviews получает все PR пользователя
// GET /users/getReview?user_id=...
func GetUserReviewsHandler(log *slog.Logger, service *core.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := r.URL.Query().Get("user_id")
		if userID == "" {
			log.Error("user_id is required")
			writeError(w, http.StatusBadRequest, "BAD_REQUEST", "user_id is required")
			return
		}

		prs, err := service.GetUserReviews(r.Context(), userID)
		if err != nil {
			if errorCode, ok := mapErrorToCode(err); ok {
				log.Error("failed to get user reviews", "error", err, "code", errorCode)
				writeError(w, http.StatusNotFound, errorCode, err.Error())
				return
			}
			log.Error("failed to get user reviews", "error", err)
			writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
			return
		}

		prsDTO, err := prsToShortDTOs(prs)
		if err != nil {
			log.Error("failed to convert PRs to DTOs", "error", err)
			writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
			return
		}

		response := GetUserReviewsResponseDTO{
			UserID:       userID,
			PullRequests: prsDTO,
		}

		writeJSON(w, http.StatusOK, response)
	}
}

// GET /statistics.
func GetStatisticsHandler(log *slog.Logger, service *core.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		stats, err := service.GetStatistics(r.Context())
		if err != nil {
			log.Error("failed to get statistics", "error", err)
			writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
			return
		}

		response := statisticsToDTO(stats)

		writeJSON(w, http.StatusOK, response)
	}
}
