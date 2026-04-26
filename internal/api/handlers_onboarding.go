package api

import (
	"net/http"

	"relay/internal/contracts"
	"relay/internal/services"
)

// handleOnboardingComplete is the POST /v1/onboarding entry point. The
// request body is the locked, post-D3 shape — the strict decoder rejects any
// stray relay_url or session_cookie fields with UNKNOWN_JSON_FIELD (T6).
func (h Handler) handleOnboardingComplete(w http.ResponseWriter, r *http.Request) {
	var input services.CompleteOnboardingInput
	if !decodeStrictJSONBody(w, r, "relay onboarding complete", &input) {
		return
	}
	result, err := h.services.CompleteOnboarding(r.Context(), input)
	if err != nil {
		writeServiceError(w, "relay onboarding complete", err)
		return
	}
	writeJSON(w, http.StatusOK, contracts.Success("relay onboarding complete", result))
}

// handleOnboardingDeleteKey is the DELETE /v1/onboarding entry point — the
// "forget my key" trust escape hatch (D5). The project row is preserved so
// the user can re-onboard without losing prior packets.
//
// RequireUserAuth refuses admin-only callers (UserID is empty under admin
// bearer auth) so this endpoint is user-session-only by construction.
func (h Handler) handleOnboardingDeleteKey(w http.ResponseWriter, r *http.Request) {
	auth, err := services.RequireUserAuth(r.Context())
	if err != nil {
		writeServiceError(w, "relay onboarding delete key", err)
		return
	}
	if err := h.services.DeleteOnboardingKey(r.Context(), auth.UserID); err != nil {
		writeServiceError(w, "relay onboarding delete key", err)
		return
	}
	writeJSON(w, http.StatusOK, contracts.Success("relay onboarding delete key", map[string]string{"status": "ok"}))
}

// routeOnboarding dispatches /v1/onboarding by HTTP method. Both branches sit
// behind requireSessionOrAdmin in server.go so the shared auth gate runs once.
func routeOnboarding(handler Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			handler.handleOnboardingComplete(w, r)
		case http.MethodDelete:
			handler.handleOnboardingDeleteKey(w, r)
		default:
			writeJSON(w, http.StatusMethodNotAllowed, contracts.Failure("relay onboarding", "METHOD_NOT_ALLOWED", "method not allowed", false))
		}
	}
}
