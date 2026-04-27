package api

import (
	"net/http"
	"strings"

	"relay/internal/contracts"
	"relay/internal/services"
)

func (h Handler) handleProviderCredentials(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		result, err := h.services.ListProviderCredentials(r.Context())
		if err != nil {
			writeServiceError(w, "relay provider credentials list", err)
			return
		}
		writeJSON(w, http.StatusOK, contracts.Success("relay provider credentials list", result))
	case http.MethodPost:
		var input services.ProviderCredentialUpsertInput
		if !decodeStrictJSONBody(w, r, "relay provider credential upsert", &input) {
			return
		}
		result, err := h.services.UpsertProviderCredential(r.Context(), input)
		if err != nil {
			writeServiceError(w, "relay provider credential upsert", err)
			return
		}
		writeJSON(w, http.StatusOK, contracts.Success("relay provider credential upsert", result))
	default:
		writeJSON(w, http.StatusMethodNotAllowed, contracts.Failure("relay provider credentials", "METHOD_NOT_ALLOWED", "method not allowed", false))
	}
}

func (h Handler) handleProviderCredential(w http.ResponseWriter, r *http.Request) {
	provider := strings.Trim(strings.TrimPrefix(r.URL.Path, "/v1/settings/provider-credentials/"), "/")
	if provider == "" {
		writeJSON(w, http.StatusNotFound, contracts.Failure("relay provider credential", "NOT_FOUND", "unknown provider credential route", false, "provider"))
		return
	}
	switch r.Method {
	case http.MethodDelete:
		if err := h.services.DeleteProviderCredential(r.Context(), provider); err != nil {
			writeServiceError(w, "relay provider credential delete", err)
			return
		}
		writeJSON(w, http.StatusOK, contracts.Success("relay provider credential delete", map[string]string{"status": "ok"}))
	default:
		writeJSON(w, http.StatusMethodNotAllowed, contracts.Failure("relay provider credential", "METHOD_NOT_ALLOWED", "method not allowed", false))
	}
}
