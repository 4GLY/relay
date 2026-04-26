package api

import (
	"net/http"
	"strconv"
	"strings"

	"relay/internal/contracts"
	"relay/internal/services"
)

const styleMemoryListMaxLimit = 100

func (h Handler) handleJudgmentTraceWrite(w http.ResponseWriter, r *http.Request) {
	var input services.JudgmentTraceWriteInput
	if !decodeStrictJSONBody(w, r, "relay judgment-trace write", &input) {
		return
	}
	result, err := h.services.WriteJudgmentTrace(r.Context(), input)
	if err != nil {
		writeServiceError(w, "relay judgment-trace write", err)
		return
	}
	writeJSON(w, http.StatusOK, contracts.Success("relay judgment-trace write", result))
}

func (h Handler) handleHeuristicProposalCreate(w http.ResponseWriter, r *http.Request) {
	var input services.HeuristicProposalCreateInput
	if !decodeStrictJSONBody(w, r, "relay heuristic-proposal create", &input) {
		return
	}
	result, err := h.services.CreateHeuristicProposal(r.Context(), input)
	if err != nil {
		writeServiceError(w, "relay heuristic-proposal create", err)
		return
	}
	writeJSON(w, http.StatusOK, contracts.Success("relay heuristic-proposal create", result))
}

func (h Handler) handleHeuristicProposalsList(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	limit, ok := parseStyleMemoryListLimit(w, q.Get("limit"), "relay heuristic-proposals list")
	if !ok {
		return
	}
	input := services.ListHeuristicProposalsInput{
		ProjectID: strings.TrimSpace(q.Get("project_id")),
		Project:   strings.TrimSpace(q.Get("project")),
		State:     strings.TrimSpace(q.Get("state")),
		Cursor:    strings.TrimSpace(q.Get("cursor")),
		Limit:     limit,
	}
	if input.ProjectID == "" && input.Project == "" {
		writeJSON(w, http.StatusBadRequest, contracts.Failure("relay heuristic-proposals list", "MISSING_REQUIRED_FIELDS", "project_id or project is required", false, "project_id"))
		return
	}
	result, err := h.services.ListHeuristicProposals(r.Context(), input)
	if err != nil {
		writeServiceError(w, "relay heuristic-proposals list", err)
		return
	}
	writeJSON(w, http.StatusOK, contracts.Success("relay heuristic-proposals list", result))
}

func (h Handler) handleApprovedHeuristicsList(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	limit, ok := parseStyleMemoryListLimit(w, q.Get("limit"), "relay approved-heuristics list")
	if !ok {
		return
	}
	input := services.ListApprovedHeuristicsInput{
		ProjectID:    strings.TrimSpace(q.Get("project_id")),
		Project:      strings.TrimSpace(q.Get("project")),
		Workflow:     strings.TrimSpace(q.Get("workflow")),
		ArtifactType: strings.TrimSpace(q.Get("artifact_type")),
		Cursor:       strings.TrimSpace(q.Get("cursor")),
		Limit:        limit,
	}
	if input.ProjectID == "" && input.Project == "" {
		writeJSON(w, http.StatusBadRequest, contracts.Failure("relay approved-heuristics list", "MISSING_REQUIRED_FIELDS", "project_id or project is required", false, "project_id"))
		return
	}
	result, err := h.services.ListApprovedHeuristics(r.Context(), input)
	if err != nil {
		writeServiceError(w, "relay approved-heuristics list", err)
		return
	}
	writeJSON(w, http.StatusOK, contracts.Success("relay approved-heuristics list", result))
}

func parseStyleMemoryListLimit(w http.ResponseWriter, raw string, command string) (int, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, true
	}
	parsed, err := strconv.Atoi(raw)
	if err != nil || parsed < 0 {
		writeJSON(w, http.StatusBadRequest, contracts.Failure(command, "INVALID_QUERY_PARAM", "limit must be a non-negative integer", false, "limit"))
		return 0, false
	}
	if parsed > styleMemoryListMaxLimit {
		parsed = styleMemoryListMaxLimit
	}
	return parsed, true
}

func (h Handler) handleHeuristicProposalReview(w http.ResponseWriter, r *http.Request) {
	var input services.HeuristicProposalReviewInput
	if !decodeStrictJSONBody(w, r, "relay heuristic-proposal review", &input) {
		return
	}
	result, err := h.services.ReviewHeuristicProposal(r.Context(), input)
	if err != nil {
		writeServiceError(w, "relay heuristic-proposal review", err)
		return
	}
	writeJSON(w, http.StatusOK, contracts.Success("relay heuristic-proposal review", result))
}

func (h Handler) handleApprovedHeuristicUpdate(w http.ResponseWriter, r *http.Request) {
	var input services.ApprovedHeuristicUpdateInput
	if !decodeStrictJSONBody(w, r, "relay approved-heuristic update", &input) {
		return
	}
	result, err := h.services.UpdateApprovedHeuristic(r.Context(), input)
	if err != nil {
		writeServiceError(w, "relay approved-heuristic update", err)
		return
	}
	writeJSON(w, http.StatusOK, contracts.Success("relay approved-heuristic update", result))
}
