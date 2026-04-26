package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"relay/internal/app"
	"relay/internal/config"
	"relay/internal/contracts"
	"relay/internal/domain"
	"relay/internal/lib"
	"relay/internal/services"
)

// fakeOnboardingStore mirrors the in-memory store from services_test but
// counts EnsureProjectByOwnerName calls so concurrent tests can assert one
// durable Personal project.
type fakeOnboardingStore struct {
	mu              sync.Mutex
	rows            map[string]domain.UserOnboarding
	projects        map[string]domain.Project
	ensureCallCount int64
}

func newFakeOnboardingStore() *fakeOnboardingStore {
	return &fakeOnboardingStore{
		rows:     map[string]domain.UserOnboarding{},
		projects: map[string]domain.Project{},
	}
}

func (s *fakeOnboardingStore) UpsertOnboarding(_ context.Context, row domain.UserOnboarding) (domain.UserOnboarding, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if existing, ok := s.rows[row.UserID]; ok {
		if row.AnthropicKeyCiphertext == nil {
			row.AnthropicKeyCiphertext = existing.AnthropicKeyCiphertext
			row.AnthropicKeyNonce = existing.AnthropicKeyNonce
			row.AnthropicKeyKEKVersion = existing.AnthropicKeyKEKVersion
			row.AnthropicKeyPrefix = existing.AnthropicKeyPrefix
			row.AnthropicKeyLast4 = existing.AnthropicKeyLast4
			row.AadSalt = existing.AadSalt
			row.LastValidatedAt = existing.LastValidatedAt
		}
	}
	s.rows[row.UserID] = row
	return row, nil
}

func (s *fakeOnboardingStore) GetOnboardingByUserID(_ context.Context, userID string) (domain.UserOnboarding, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	row, ok := s.rows[userID]
	if !ok {
		return domain.UserOnboarding{}, lib.NotFound("ONBOARDING_NOT_FOUND", "onboarding not found")
	}
	return row, nil
}

func (s *fakeOnboardingStore) DeleteOnboardingKey(_ context.Context, userID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	row, ok := s.rows[userID]
	if !ok {
		return lib.NotFound("ONBOARDING_NOT_FOUND", "onboarding not found")
	}
	row.AnthropicKeyCiphertext = nil
	row.AnthropicKeyNonce = nil
	row.AnthropicKeyPrefix = ""
	row.AnthropicKeyLast4 = ""
	row.AadSalt = nil
	row.LastValidatedAt = nil
	s.rows[userID] = row
	return nil
}

func (s *fakeOnboardingStore) EnsureProjectByOwnerName(_ context.Context, ownerUserID, name, newID string) (domain.Project, error) {
	atomic.AddInt64(&s.ensureCallCount, 1)
	s.mu.Lock()
	defer s.mu.Unlock()
	key := ownerUserID + ":" + name
	if existing, ok := s.projects[key]; ok {
		return existing, nil
	}
	project := domain.Project{ID: newID, Name: name, Status: "active", OwnerUserID: ownerUserID}
	s.projects[key] = project
	return project, nil
}

type onboardingFixture struct {
	mux        *http.ServeMux
	onboarding *fakeOnboardingStore
	cookie     *http.Cookie
	adminToken string
	userID     string
}

func newOnboardingFixture(t *testing.T) *onboardingFixture {
	t.Helper()

	users := newAuthFakeUserStore()
	const userID = "usr_onboarding_owner"
	users.items[userID] = domain.User{ID: userID, Email: "owner@relay.test", DisplayName: "owner"}

	sessions := newAuthFakeUserSessionStore()
	tok, err := lib.NewSecretToken("rsess")
	if err != nil {
		t.Fatalf("token: %v", err)
	}
	if _, err := sessions.CreateUserSession(context.Background(), domain.UserSession{
		ID:        "usess_owner",
		UserID:    userID,
		TokenHash: lib.TokenHash(tok),
		ExpiresAt: time.Now().Add(30 * 24 * time.Hour),
	}); err != nil {
		t.Fatalf("session: %v", err)
	}

	onboarding := newFakeOnboardingStore()
	svc := services.New(services.Dependencies{
		Users:        users,
		UserSessions: sessions,
		Onboarding:   onboarding,
	})

	const adminToken = "admin-token"
	mux := buildMux(Handler{services: svc}, config.Config{APIToken: adminToken}, app.Runtime{Services: svc})

	return &onboardingFixture{
		mux:        mux,
		onboarding: onboarding,
		cookie:     &http.Cookie{Name: sessionCookieName, Value: tok},
		adminToken: adminToken,
		userID:     userID,
	}
}

func TestOnboardingConcurrentDoubleSubmit(t *testing.T) {
	f := newOnboardingFixture(t)

	postOnce := func() *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/v1/onboarding", bytes.NewReader([]byte(`{}`)))
		req.AddCookie(f.cookie)
		rec := httptest.NewRecorder()
		f.mux.ServeHTTP(rec, req)
		return rec
	}

	var wg sync.WaitGroup
	codes := make([]int, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			rec := postOnce()
			codes[idx] = rec.Code
		}(i)
	}
	wg.Wait()

	for i, c := range codes {
		if c != http.StatusOK {
			t.Fatalf("request %d expected 200, got %d", i, c)
		}
	}
	if len(f.onboarding.rows) != 1 {
		t.Fatalf("expected exactly one onboarding row, got %d", len(f.onboarding.rows))
	}
	if len(f.onboarding.projects) != 1 {
		t.Fatalf("expected exactly one project, got %d", len(f.onboarding.projects))
	}
}

func TestOnboardingRejectsAnthropicKeyField(t *testing.T) {
	f := newOnboardingFixture(t)

	body, _ := json.Marshal(map[string]any{"anthropic_key": "sk-ant-test1234"})
	req := httptest.NewRequest(http.MethodPost, "/v1/onboarding", bytes.NewReader(body))
	req.AddCookie(f.cookie)
	rec := httptest.NewRecorder()
	f.mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "UNKNOWN_JSON_FIELD") {
		t.Fatalf("expected UNKNOWN_JSON_FIELD, got %s", rec.Body.String())
	}
}

func TestOnboardingStrictDecoderRejectsRelayURL(t *testing.T) {
	f := newOnboardingFixture(t)

	body, _ := json.Marshal(map[string]any{"relay_url": "https://relay.4gly.dev"})
	req := httptest.NewRequest(http.MethodPost, "/v1/onboarding", bytes.NewReader(body))
	req.AddCookie(f.cookie)
	rec := httptest.NewRecorder()
	f.mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "UNKNOWN_JSON_FIELD") {
		t.Fatalf("expected UNKNOWN_JSON_FIELD, got %s", rec.Body.String())
	}
}

func TestOnboardingRejectsAnonymous(t *testing.T) {
	f := newOnboardingFixture(t)

	req := httptest.NewRequest(http.MethodPost, "/v1/onboarding", bytes.NewReader([]byte(`{}`)))
	rec := httptest.NewRecorder()
	f.mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestOnboardingAdminBearerReturnsUnauthorized(t *testing.T) {
	f := newOnboardingFixture(t)

	req := httptest.NewRequest(http.MethodPost, "/v1/onboarding", bytes.NewReader([]byte(`{}`)))
	req.Header.Set("Authorization", "Bearer "+f.adminToken)
	rec := httptest.NewRecorder()
	f.mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for admin bearer, got %d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "UNAUTHORIZED") {
		t.Fatalf("expected UNAUTHORIZED in body, got %s", rec.Body.String())
	}
}

func TestOnboardingDeleteNotFound(t *testing.T) {
	f := newOnboardingFixture(t)

	req := httptest.NewRequest(http.MethodDelete, "/v1/onboarding", nil)
	req.AddCookie(f.cookie)
	rec := httptest.NewRecorder()
	f.mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for missing row, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestOnboardingDeleteKeepsOnboardingComplete(t *testing.T) {
	f := newOnboardingFixture(t)

	postReq := httptest.NewRequest(http.MethodPost, "/v1/onboarding", bytes.NewReader([]byte(`{}`)))
	postReq.AddCookie(f.cookie)
	postRec := httptest.NewRecorder()
	f.mux.ServeHTTP(postRec, postReq)
	if postRec.Code != http.StatusOK {
		t.Fatalf("expected POST 200, got %d body=%s", postRec.Code, postRec.Body.String())
	}

	row := f.onboarding.rows[f.userID]
	row.AnthropicKeyCiphertext = []byte("ciphertext")
	row.AnthropicKeyNonce = []byte("nonce")
	row.AadSalt = []byte("salt")
	f.onboarding.rows[f.userID] = row

	delReq := httptest.NewRequest(http.MethodDelete, "/v1/onboarding", nil)
	delReq.AddCookie(f.cookie)
	delRec := httptest.NewRecorder()
	f.mux.ServeHTTP(delRec, delReq)
	if delRec.Code != http.StatusOK {
		t.Fatalf("expected DELETE 200, got %d body=%s", delRec.Code, delRec.Body.String())
	}

	meReq := httptest.NewRequest(http.MethodGet, "/v1/auth/me", nil)
	meReq.AddCookie(f.cookie)
	meRec := httptest.NewRecorder()
	f.mux.ServeHTTP(meRec, meReq)

	var envelope struct {
		Data map[string]any `json:"data"`
	}
	if err := json.Unmarshal(meRec.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode envelope: %v", err)
	}
	if got, _ := envelope.Data["onboarding_complete"].(bool); !got {
		t.Fatalf("expected onboarding_complete=true after key deletion, body=%s", meRec.Body.String())
	}
}

func TestOnboardingRejectsUnsupportedMethod(t *testing.T) {
	f := newOnboardingFixture(t)

	req := httptest.NewRequest(http.MethodGet, "/v1/onboarding", nil)
	req.AddCookie(f.cookie)
	rec := httptest.NewRecorder()
	f.mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestAuthMeReflectsKeylessOnboardingStatus(t *testing.T) {
	f := newOnboardingFixture(t)

	postReq := httptest.NewRequest(http.MethodPost, "/v1/onboarding", bytes.NewReader([]byte(`{}`)))
	postReq.AddCookie(f.cookie)
	f.mux.ServeHTTP(httptest.NewRecorder(), postReq)

	meReq := httptest.NewRequest(http.MethodGet, "/v1/auth/me", nil)
	meReq.AddCookie(f.cookie)
	meRec := httptest.NewRecorder()
	f.mux.ServeHTTP(meRec, meReq)
	if meRec.Code != http.StatusOK {
		t.Fatalf("expected /me 200, got %d body=%s", meRec.Code, meRec.Body.String())
	}
	var envelope struct {
		Data map[string]any `json:"data"`
	}
	if err := json.Unmarshal(meRec.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode envelope: %v", err)
	}
	if got, _ := envelope.Data["onboarding_complete"].(bool); !got {
		t.Fatalf("expected onboarding_complete=true, got %v body=%s", envelope.Data, meRec.Body.String())
	}
	if pid, _ := envelope.Data["default_project_id"].(string); pid == "" {
		t.Fatalf("expected default_project_id, got body=%s", meRec.Body.String())
	}
}

func TestAuthMeOnboardingFalseForNewUser(t *testing.T) {
	f := newOnboardingFixture(t)

	meReq := httptest.NewRequest(http.MethodGet, "/v1/auth/me", nil)
	meReq.AddCookie(f.cookie)
	meRec := httptest.NewRecorder()
	f.mux.ServeHTTP(meRec, meReq)
	if meRec.Code != http.StatusOK {
		t.Fatalf("expected /me 200, got %d body=%s", meRec.Code, meRec.Body.String())
	}
	var envelope struct {
		Data map[string]any `json:"data"`
	}
	_ = json.Unmarshal(meRec.Body.Bytes(), &envelope)
	if got, _ := envelope.Data["onboarding_complete"].(bool); got {
		t.Fatalf("expected onboarding_complete=false for fresh user, body=%s", meRec.Body.String())
	}
}

func TestWriteServiceErrorStatusTable(t *testing.T) {
	cases := []struct {
		code     string
		expected int
	}{
		{"ONBOARDING_NOT_FOUND", http.StatusNotFound},
		{"PROJECT_NOT_FOUND", http.StatusNotFound},
		{"PACKET_SNAPSHOT_NOT_FOUND", http.StatusNotFound},
		{"API_KEY_NOT_FOUND", http.StatusUnauthorized},
		{"API_KEY_NOT_FOUND_BY_ID", http.StatusNotFound},
		{"FORBIDDEN", http.StatusForbidden},
		{"UNAUTHORIZED", http.StatusUnauthorized},
		{"MISCONFIGURED", http.StatusInternalServerError},
		{"PROPOSAL_ALREADY_RESOLVED", http.StatusConflict},
		{"MISSING_REQUIRED_FIELDS", http.StatusBadRequest},
		{"UNKNOWN_FUTURE_CODE", http.StatusBadRequest},
	}
	for _, tc := range cases {
		t.Run(tc.code, func(t *testing.T) {
			rec := httptest.NewRecorder()
			writeServiceError(rec, "test", lib.AppError{Code: tc.code, Message: "x"})
			if rec.Code != tc.expected {
				t.Fatalf("code %s: expected status %d, got %d", tc.code, tc.expected, rec.Code)
			}
			var env contracts.ErrorEnvelope
			if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
				t.Fatalf("decode envelope: %v", err)
			}
			if env.Error.Code != tc.code {
				t.Fatalf("envelope code mismatch: got %q want %q", env.Error.Code, tc.code)
			}
		})
	}
}
