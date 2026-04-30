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

func (h Handler) handleUserAPIKeys(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		result, err := h.services.ListUserAPIKeys(r.Context())
		if err != nil {
			writeServiceError(w, "relay user api-key list", err)
			return
		}
		writeJSON(w, http.StatusOK, contracts.Success("relay user api-key list", result))
	case http.MethodPost:
		var input services.IssueAPIKeyInput
		if !decodeStrictJSONBody(w, r, "relay user api-key issue", &input) {
			return
		}
		result, err := h.services.IssueUserAPIKey(r.Context(), input)
		if err != nil {
			writeServiceError(w, "relay user api-key issue", err)
			return
		}
		writeJSON(w, http.StatusOK, contracts.Success("relay user api-key issue", result))
	default:
		writeJSON(w, http.StatusMethodNotAllowed, contracts.Failure("relay user api-keys", "METHOD_NOT_ALLOWED", "method not allowed", false))
	}
}

func (h Handler) handleRevokeUserAPIKey(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, contracts.Failure("relay user api-key revoke", "METHOD_NOT_ALLOWED", "method not allowed", false))
		return
	}

	var input services.RevokeAPIKeyInput
	if !decodeStrictJSONBody(w, r, "relay user api-key revoke", &input) {
		return
	}
	result, err := h.services.RevokeUserAPIKey(r.Context(), input)
	if err != nil {
		writeServiceError(w, "relay user api-key revoke", err)
		return
	}
	writeJSON(w, http.StatusOK, contracts.Success("relay user api-key revoke", result))
}
