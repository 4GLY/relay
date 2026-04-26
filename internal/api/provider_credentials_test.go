package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"relay/internal/app"
	"relay/internal/config"
	"relay/internal/domain"
	"relay/internal/lib"
	"relay/internal/lib/crypto"
	"relay/internal/services"
)

type apiFakeProviderCredentialStore struct {
	rows map[string]domain.UserProviderCredential
}

func newAPIFakeProviderCredentialStore() *apiFakeProviderCredentialStore {
	return &apiFakeProviderCredentialStore{rows: map[string]domain.UserProviderCredential{}}
}

func apiProviderCredentialKey(userID string, provider string) string {
	return userID + ":" + provider
}

func (s *apiFakeProviderCredentialStore) UpsertUserProviderCredential(_ context.Context, credential domain.UserProviderCredential) (domain.UserProviderCredential, error) {
	now := time.Now()
	key := apiProviderCredentialKey(credential.UserID, credential.Provider)
	if existing, ok := s.rows[key]; ok {
		credential.CreatedAt = existing.CreatedAt
	} else {
		credential.CreatedAt = now
	}
	credential.UpdatedAt = now
	credential.DeletedAt = nil
	s.rows[key] = credential
	return credential, nil
}

func (s *apiFakeProviderCredentialStore) GetUserProviderCredential(_ context.Context, userID string, provider string) (domain.UserProviderCredential, error) {
	row, ok := s.rows[apiProviderCredentialKey(userID, provider)]
	if !ok || row.DeletedAt != nil {
		return domain.UserProviderCredential{}, lib.NotFound("PROVIDER_CREDENTIAL_NOT_FOUND", "provider credential not found")
	}
	return row, nil
}

func (s *apiFakeProviderCredentialStore) ListUserProviderCredentials(_ context.Context, userID string) ([]domain.UserProviderCredential, error) {
	var rows []domain.UserProviderCredential
	for _, row := range s.rows {
		if row.UserID == userID && row.DeletedAt == nil {
			rows = append(rows, row)
		}
	}
	return rows, nil
}

func (s *apiFakeProviderCredentialStore) DeleteUserProviderCredential(_ context.Context, userID string, provider string) error {
	key := apiProviderCredentialKey(userID, provider)
	row, ok := s.rows[key]
	if !ok || row.DeletedAt != nil {
		return lib.NotFound("PROVIDER_CREDENTIAL_NOT_FOUND", "provider credential not found")
	}
	now := time.Now()
	row.DeletedAt = &now
	row.UpdatedAt = now
	s.rows[key] = row
	return nil
}

type providerCredentialFixture struct {
	mux         *http.ServeMux
	providerKey *apiFakeProviderCredentialStore
	cookie      *http.Cookie
}

func newProviderCredentialFixture(t *testing.T) *providerCredentialFixture {
	t.Helper()
	users := newAuthFakeUserStore()
	const userID = "usr_provider_owner"
	users.items[userID] = domain.User{ID: userID, Email: "owner@relay.test", DisplayName: "owner"}

	sessions := newAuthFakeUserSessionStore()
	tok, err := lib.NewSecretToken("rsess")
	if err != nil {
		t.Fatalf("token: %v", err)
	}
	if _, err := sessions.CreateUserSession(context.Background(), domain.UserSession{
		ID:        "usess_provider_owner",
		UserID:    userID,
		TokenHash: lib.TokenHash(tok),
		ExpiresAt: time.Now().Add(30 * 24 * time.Hour),
	}); err != nil {
		t.Fatalf("session: %v", err)
	}

	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i + 1)
	}
	store := newAPIFakeProviderCredentialStore()
	svc := services.NewWithKEKs(
		services.Dependencies{
			Users:        users,
			UserSessions: sessions,
			ProviderKeys: store,
		},
		map[crypto.KEKVersion][]byte{1: key},
		1,
	)
	mux := buildMux(Handler{services: svc}, config.Config{APIToken: "admin-token"}, app.Runtime{Services: svc})
	return &providerCredentialFixture{
		mux:         mux,
		providerKey: store,
		cookie:      &http.Cookie{Name: sessionCookieName, Value: tok},
	}
}

func TestProviderCredentialsRoundtrip(t *testing.T) {
	f := newProviderCredentialFixture(t)

	body, _ := json.Marshal(map[string]any{
		"provider": "anthropic",
		"api_key":  "sk-ant-provider-settings-1234",
	})
	postReq := httptest.NewRequest(http.MethodPost, "/v1/settings/provider-credentials", bytes.NewReader(body))
	postReq.AddCookie(f.cookie)
	postRec := httptest.NewRecorder()
	f.mux.ServeHTTP(postRec, postReq)
	if postRec.Code != http.StatusOK {
		t.Fatalf("expected POST 200, got %d body=%s", postRec.Code, postRec.Body.String())
	}
	if len(f.providerKey.rows) != 1 {
		t.Fatalf("expected provider credential row, got %d", len(f.providerKey.rows))
	}

	listReq := httptest.NewRequest(http.MethodGet, "/v1/settings/provider-credentials", nil)
	listReq.AddCookie(f.cookie)
	listRec := httptest.NewRecorder()
	f.mux.ServeHTTP(listRec, listReq)
	if listRec.Code != http.StatusOK {
		t.Fatalf("expected GET 200, got %d body=%s", listRec.Code, listRec.Body.String())
	}
	if !bytes.Contains(listRec.Body.Bytes(), []byte(`"provider":"anthropic"`)) {
		t.Fatalf("expected anthropic status, body=%s", listRec.Body.String())
	}

	delReq := httptest.NewRequest(http.MethodDelete, "/v1/settings/provider-credentials/anthropic", nil)
	delReq.AddCookie(f.cookie)
	delRec := httptest.NewRecorder()
	f.mux.ServeHTTP(delRec, delReq)
	if delRec.Code != http.StatusOK {
		t.Fatalf("expected DELETE 200, got %d body=%s", delRec.Code, delRec.Body.String())
	}
}

func TestProviderCredentialsRejectUnknownField(t *testing.T) {
	f := newProviderCredentialFixture(t)
	body, _ := json.Marshal(map[string]any{
		"provider":  "anthropic",
		"api_key":   "sk-ant-provider-settings-1234",
		"relay_url": "https://relay.4gly.dev",
	})
	req := httptest.NewRequest(http.MethodPost, "/v1/settings/provider-credentials", bytes.NewReader(body))
	req.AddCookie(f.cookie)
	rec := httptest.NewRecorder()
	f.mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", rec.Code, rec.Body.String())
	}
}
