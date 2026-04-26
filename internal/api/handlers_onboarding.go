package api

import (
	"net/http"

	"relay/internal/contracts"
	"relay/internal/services"
)

// handleOnboardingComplete is the POST /v1/onboarding entry point. The request
// body is intentionally empty: first-run onboarding creates the user's
// workspace without requiring a provider key. The strict decoder rejects any
// stray fields, including anthropic_key, relay_url, or session_cookie.
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

// handleOnboardingDeleteKey is the DELETE /v1/onboarding entry point for
// disconnecting provider credentials. The onboarding row and default project
// are preserved so the user does not lose prior packets or return to first-run
// setup.
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
