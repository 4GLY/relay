package api

import (
	"net/http"
	"net/url"
	"strings"
	"time"

	"relay/internal/contracts"
	"relay/internal/services"
)

const sessionCookieName = "relay_session"

type authMeResponse struct {
	UserID      string `json:"user_id"`
	Email       string `json:"email,omitempty"`
	DisplayName string `json:"display_name,omitempty"`
	AvatarURL   string `json:"avatar_url,omitempty"`
}

func (h Handler) handleAuthRouter(w http.ResponseWriter, r *http.Request) {
	rest := strings.TrimPrefix(r.URL.Path, "/v1/auth/")
	if rest == "" {
		writeJSON(w, http.StatusNotFound, contracts.Failure("relay auth", "NOT_FOUND", "unknown auth route", false))
		return
	}
	parts := strings.Split(rest, "/")

	switch parts[0] {
	case "me":
		if r.Method != http.MethodGet {
			writeJSON(w, http.StatusMethodNotAllowed, contracts.Failure("relay auth me", "METHOD_NOT_ALLOWED", "method not allowed", false))
			return
		}
		h.handleAuthMe(w, r)
		return
	case "logout":
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, contracts.Failure("relay auth logout", "METHOD_NOT_ALLOWED", "method not allowed", false))
			return
		}
		h.handleAuthLogout(w, r)
		return
	}

	if len(parts) < 2 {
		writeJSON(w, http.StatusNotFound, contracts.Failure("relay auth", "NOT_FOUND", "unknown auth route", false))
		return
	}
	provider := parts[0]
	action := parts[1]

	switch action {
	case "start":
		if r.Method != http.MethodGet {
			writeJSON(w, http.StatusMethodNotAllowed, contracts.Failure("relay auth start", "METHOD_NOT_ALLOWED", "method not allowed", false))
			return
		}
		h.handleAuthStart(w, r, provider)
	case "callback":
		if r.Method != http.MethodGet {
			writeJSON(w, http.StatusMethodNotAllowed, contracts.Failure("relay auth callback", "METHOD_NOT_ALLOWED", "method not allowed", false))
			return
		}
		h.handleAuthCallback(w, r, provider)
	default:
		writeJSON(w, http.StatusNotFound, contracts.Failure("relay auth", "NOT_FOUND", "unknown auth route", false))
	}
}

func (h Handler) handleAuthStart(w http.ResponseWriter, r *http.Request, provider string) {
	prov, ok := h.oauth.Get(provider)
	if !ok {
		writeJSON(w, http.StatusNotFound, contracts.Failure("relay auth start", "OAUTH_PROVIDER_NOT_CONFIGURED", "provider is not configured", false, "provider"))
		return
	}

	redirectTo := strings.TrimSpace(r.URL.Query().Get("redirect_to"))
	stateID, err := h.services.StartOAuth(r.Context(), provider, redirectTo)
	if err != nil {
		writeServiceError(w, "relay auth start", err)
		return
	}
	authURL := prov.AuthURL(stateID, h.oauthRedirectURI(provider))
	http.Redirect(w, r, authURL, http.StatusFound)
}

func (h Handler) handleAuthCallback(w http.ResponseWriter, r *http.Request, provider string) {
	prov, ok := h.oauth.Get(provider)
	if !ok {
		writeJSON(w, http.StatusNotFound, contracts.Failure("relay auth callback", "OAUTH_PROVIDER_NOT_CONFIGURED", "provider is not configured", false, "provider"))
		return
	}

	stateID := strings.TrimSpace(r.URL.Query().Get("state"))
	code := strings.TrimSpace(r.URL.Query().Get("code"))
	if stateID == "" || code == "" {
		writeJSON(w, http.StatusBadRequest, contracts.Failure("relay auth callback", "MISSING_REQUIRED_FIELDS", "state and code are required", false, "state", "code"))
		return
	}

	redirectTo, err := h.services.ConsumeOAuthState(r.Context(), stateID, provider)
	if err != nil {
		writeServiceError(w, "relay auth callback", err)
		return
	}

	profile, err := prov.Exchange(r.Context(), code, h.oauthRedirectURI(provider))
	if err != nil {
		writeJSON(w, http.StatusBadGateway, contracts.Failure("relay auth callback", "OAUTH_EXCHANGE_FAILED", err.Error(), true))
		return
	}

	result, err := h.services.SignInWithOAuthProfile(r.Context(), profile)
	if err != nil {
		writeServiceError(w, "relay auth callback", err)
		return
	}

	setSessionCookie(w, h.cookieSecure, result.SessionToken, result.SessionExpiresAt)

	target := strings.TrimSpace(redirectTo)
	if target == "" {
		target = "/"
	}
	http.Redirect(w, r, target, http.StatusFound)
}

func (h Handler) handleAuthMe(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil || cookie.Value == "" {
		writeJSON(w, http.StatusUnauthorized, contracts.Failure("relay auth me", "UNAUTHORIZED", "missing session cookie", false))
		return
	}
	user, err := h.services.GetUserBySessionToken(r.Context(), cookie.Value)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, contracts.Failure("relay auth me", "UNAUTHORIZED", "invalid or expired session", false))
		return
	}
	writeJSON(w, http.StatusOK, contracts.Success("relay auth me", authMeResponse{
		UserID:      user.ID,
		Email:       user.Email,
		DisplayName: user.DisplayName,
		AvatarURL:   user.AvatarURL,
	}))
}

func (h Handler) handleAuthLogout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(sessionCookieName)
	if err == nil && cookie.Value != "" {
		if session, err := h.services.GetSessionByToken(r.Context(), cookie.Value); err == nil {
			_ = h.services.SignOut(r.Context(), session.ID)
		}
	}
	clearSessionCookie(w, h.cookieSecure)
	writeJSON(w, http.StatusOK, contracts.Success("relay auth logout", map[string]string{"status": "ok"}))
}

func (h Handler) oauthRedirectURI(provider string) string {
	base := strings.TrimRight(h.oauthRedirectBase, "/")
	return base + "/v1/auth/" + provider + "/callback"
}

func setSessionCookie(w http.ResponseWriter, secure bool, token string, expires time.Time) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    token,
		Path:     "/",
		Expires:  expires,
		MaxAge:   int(time.Until(expires).Seconds()),
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
}

func clearSessionCookie(w http.ResponseWriter, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
}

func requireUserSessionCookie(svc services.Service, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(sessionCookieName)
		if err != nil || cookie.Value == "" {
			writeJSON(w, http.StatusUnauthorized, contracts.Failure("relay auth", "UNAUTHORIZED", "missing session cookie", false))
			return
		}
		user, err := svc.GetUserBySessionToken(r.Context(), cookie.Value)
		if err != nil {
			writeJSON(w, http.StatusUnauthorized, contracts.Failure("relay auth", "UNAUTHORIZED", "invalid or expired session", false))
			return
		}
		ctx := services.ContextWithAuthInfo(r.Context(), services.AuthInfo{
			UserID: user.ID,
			Scope:  services.APIKeyScopeGlobal,
		})
		next(w, r.WithContext(ctx))
	}
}

// validateRedirectTo is a defensive helper for any future endpoint that accepts
// arbitrary redirect URLs from the client; OAuth start currently passes the raw
// query value through to ConsumeOAuthState which roundtrips it untouched.
func validateRedirectTo(raw string) string {
	target := strings.TrimSpace(raw)
	if target == "" {
		return ""
	}
	parsed, err := url.Parse(target)
	if err != nil {
		return ""
	}
	if parsed.Scheme != "" && parsed.Scheme != "http" && parsed.Scheme != "https" {
		return ""
	}
	return parsed.String()
}
