package api

import (
	"net/http"

	"relay/internal/contracts"
	"relay/internal/services"
)

func (h Handler) handleIssueAPIKey(w http.ResponseWriter, r *http.Request) {
	var input services.IssueAPIKeyInput
	if !decodeStrictJSONBody(w, r, "relay api-key issue", &input) {
		return
	}
	result, err := h.services.IssueAPIKey(r.Context(), input)
	if err != nil {
		writeServiceError(w, "relay api-key issue", err)
		return
	}
	writeJSON(w, http.StatusOK, contracts.Success("relay api-key issue", result))
}

func (h Handler) handleListAPIKeys(w http.ResponseWriter, r *http.Request) {
	result, err := h.services.ListAPIKeys(r.Context())
	if err != nil {
		writeServiceError(w, "relay api-key list", err)
		return
	}
	writeJSON(w, http.StatusOK, contracts.Success("relay api-key list", result))
}

func (h Handler) handleRevokeAPIKey(w http.ResponseWriter, r *http.Request) {
	var input services.RevokeAPIKeyInput
	if !decodeStrictJSONBody(w, r, "relay api-key revoke", &input) {
		return
	}
	result, err := h.services.RevokeAPIKey(r.Context(), input)
	if err != nil {
		writeServiceError(w, "relay api-key revoke", err)
		return
	}
	writeJSON(w, http.StatusOK, contracts.Success("relay api-key revoke", result))
}
