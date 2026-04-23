package api

import (
	"net/http"

	"relay/internal/contracts"
	"relay/internal/services"
)

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
