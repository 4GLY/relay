package api

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"relay/internal/app"
	"relay/internal/config"
	"relay/internal/lib/oauth"
	"relay/internal/services"
)

type stubOAuthProvider struct {
	name    string
	profile oauth.Profile
}

func (s *stubOAuthProvider) Name() string { return s.name }

func (s *stubOAuthProvider) AuthURL(state string, redirectURI string) string {
	return "https://example.com/oauth/" + s.name + "?state=" + state + "&redirect_uri=" + redirectURI
}

func (s *stubOAuthProvider) Exchange(_ context.Context, _ string, _ string) (oauth.Profile, error) {
	return s.profile, nil
}

type authTestHarness struct {
	handler Handler
	mux     *http.ServeMux
	states  *apiFakeOAuthStateStore
}

func newAuthTestHarness(t *testing.T) authTestHarness {
	t.Helper()
	states := newAuthFakeOAuthStateStore()
	svc := services.New(services.Dependencies{
		Users:           newAuthFakeUserStore(),
		OAuthIdentities: newAuthFakeOAuthIdentityStore(),
		UserSessions:    newAuthFakeUserSessionStore(),
		OAuthStates:     states,
	})
	provider := &stubOAuthProvider{
		name: "github",
		profile: oauth.Profile{
			Provider:       "github",
			ProviderUserID: "1001",
			Login:          "octo",
			Email:          "octo@example.com",
			DisplayName:    "Octo",
			AvatarURL:      "https://avatars/1001",
		},
	}
	handler := Handler{
		services:          svc,
		oauth:             oauth.NewRegistry(provider),
		oauthRedirectBase: "http://localhost:8080",
		cookieSecure:      false,
	}
	mux := buildMux(handler, config.Config{APIToken: "admin-token"}, app.Runtime{Services: svc})
	return authTestHarness{handler: handler, mux: mux, states: states}
}

func TestAuthMeRejectsMissingCookie(t *testing.T) {
	h := newAuthTestHarness(t)

	req := httptest.NewRequest(http.MethodGet, "/v1/auth/me", nil)
	rec := httptest.NewRecorder()
	h.mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestAuthFlowSetsAndReadsSessionCookie(t *testing.T) {
	h := newAuthTestHarness(t)

	startReq := httptest.NewRequest(http.MethodGet, "/v1/auth/github/start?redirect_to=/dashboard", nil)
	startRec := httptest.NewRecorder()
	h.mux.ServeHTTP(startRec, startReq)
	if startRec.Code != http.StatusFound {
		t.Fatalf("expected 302 from /v1/auth/github/start, got %d body=%s", startRec.Code, startRec.Body.String())
	}
	if loc := startRec.Header().Get("Location"); loc == "" {
		t.Fatal("expected Location header on start redirect")
	}

	if len(h.states.created) == 0 {
		t.Fatal("expected oauth state to be inserted")
	}
	stateID := h.states.created[len(h.states.created)-1]

	cbReq := httptest.NewRequest(http.MethodGet, "/v1/auth/github/callback?code=test-code&state="+stateID, nil)
	cbRec := httptest.NewRecorder()
	h.mux.ServeHTTP(cbRec, cbReq)
	if cbRec.Code != http.StatusFound {
		t.Fatalf("expected 302 from callback, got %d body=%s", cbRec.Code, cbRec.Body.String())
	}
	cookies := cbRec.Result().Cookies()
	var sessionCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == sessionCookieName {
			sessionCookie = c
			break
		}
	}
	if sessionCookie == nil || sessionCookie.Value == "" {
		t.Fatalf("expected relay_session cookie to be set, got cookies=%+v", cookies)
	}

	meReq := httptest.NewRequest(http.MethodGet, "/v1/auth/me", nil)
	meReq.AddCookie(sessionCookie)
	meRec := httptest.NewRecorder()
	h.mux.ServeHTTP(meRec, meReq)
	if meRec.Code != http.StatusOK {
		t.Fatalf("expected 200 from /v1/auth/me, got %d body=%s", meRec.Code, meRec.Body.String())
	}
	if !bytes.Contains(meRec.Body.Bytes(), []byte(`"email":"octo@example.com"`)) {
		t.Fatalf("expected email in /v1/auth/me response, got %s", meRec.Body.String())
	}

	logoutReq := httptest.NewRequest(http.MethodPost, "/v1/auth/logout", nil)
	logoutReq.AddCookie(sessionCookie)
	logoutRec := httptest.NewRecorder()
	h.mux.ServeHTTP(logoutRec, logoutReq)
	if logoutRec.Code != http.StatusOK {
		t.Fatalf("expected 200 from /v1/auth/logout, got %d body=%s", logoutRec.Code, logoutRec.Body.String())
	}

	meAgainReq := httptest.NewRequest(http.MethodGet, "/v1/auth/me", nil)
	meAgainReq.AddCookie(sessionCookie)
	meAgainRec := httptest.NewRecorder()
	h.mux.ServeHTTP(meAgainRec, meAgainReq)
	if meAgainRec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 after logout, got %d body=%s", meAgainRec.Code, meAgainRec.Body.String())
	}
}

func TestSetSessionCookieValues(t *testing.T) {
	rec := httptest.NewRecorder()
	setSessionCookie(rec, true, "tok-123", timeFutureExample())
	cookies := rec.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("expected exactly one cookie, got %d", len(cookies))
	}
	cookie := cookies[0]
	if cookie.Name != sessionCookieName {
		t.Fatalf("expected cookie name %q, got %q", sessionCookieName, cookie.Name)
	}
	if !cookie.HttpOnly || cookie.SameSite != http.SameSiteLaxMode || !cookie.Secure {
		t.Fatalf("expected HttpOnly + SameSite=Lax + Secure cookie, got %+v", cookie)
	}
}

func TestClearSessionCookieMaxAgeNegative(t *testing.T) {
	rec := httptest.NewRecorder()
	clearSessionCookie(rec, false)
	cookies := rec.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("expected exactly one cookie, got %d", len(cookies))
	}
	if cookies[0].MaxAge >= 0 {
		t.Fatalf("expected negative MaxAge to clear cookie, got %d", cookies[0].MaxAge)
	}
}
