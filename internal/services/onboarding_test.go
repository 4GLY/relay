package services

import (
	"context"
	"os"
	"sync"
	"testing"

	"relay/internal/domain"
	"relay/internal/lib"
	"relay/internal/lib/crypto"
)

// fakeOnboardingStore is an in-memory OnboardingStore for service-layer tests.
// It serializes upsert/delete on a mutex so concurrent tests can read the
// post-condition without racing the goroutines that wrote it.
type fakeOnboardingStore struct {
	mu       sync.Mutex
	rows     map[string]domain.UserOnboarding
	projects map[string]domain.Project // keyed by ownerUserID + ":" + name
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
	s.mu.Lock()
	defer s.mu.Unlock()
	key := ownerUserID + ":" + name
	if existing, ok := s.projects[key]; ok {
		return existing, nil
	}
	project := domain.Project{
		ID:          newID,
		Name:        name,
		Status:      "active",
		OwnerUserID: ownerUserID,
	}
	s.projects[key] = project
	return project, nil
}

// authedCtxFor sets a cookie-session AuthInfo so RequireUserAuth resolves a
// user-id (matching what the requireSessionOrAdmin middleware injects).
func authedCtxFor(userID string) context.Context {
	return ContextWithAuthInfo(context.Background(), AuthInfo{UserID: userID, Scope: APIKeyScopeGlobal})
}

func TestCompleteOnboardingDoesNotRequireAnthropicKey(t *testing.T) {
	store := newFakeOnboardingStore()
	svc := New(Dependencies{Onboarding: store})

	result, err := svc.CompleteOnboarding(authedCtxFor("usr_test"), CompleteOnboardingInput{})
	if err != nil {
		t.Fatalf("CompleteOnboarding returned error: %v", err)
	}
	if !result.OnboardingComplete {
		t.Fatalf("expected onboarding_complete=true, got %#v", result)
	}
	if result.DefaultProjectID == "" {
		t.Fatal("expected default_project_id to be set")
	}

	row := store.rows["usr_test"]
	if row.OnboardingCompletedAt == nil {
		t.Fatal("expected onboarding_completed_at to be set")
	}
	if len(row.AnthropicKeyCiphertext) != 0 || len(row.AnthropicKeyNonce) != 0 || len(row.AadSalt) != 0 {
		t.Fatalf("expected no provider key material during onboarding, got %#v", row)
	}
}

func TestCompleteOnboardingIsIdempotent(t *testing.T) {
	store := newFakeOnboardingStore()
	svc := New(Dependencies{Onboarding: store})
	ctx := authedCtxFor("usr_repeat")

	first, err := svc.CompleteOnboarding(ctx, CompleteOnboardingInput{})
	if err != nil {
		t.Fatalf("first onboarding: %v", err)
	}
	second, err := svc.CompleteOnboarding(ctx, CompleteOnboardingInput{})
	if err != nil {
		t.Fatalf("second onboarding: %v", err)
	}
	if first.DefaultProjectID != second.DefaultProjectID {
		t.Fatalf("expected default project preserved, got %q then %q", first.DefaultProjectID, second.DefaultProjectID)
	}
	if len(store.rows) != 1 {
		t.Fatalf("expected one onboarding row, got %d", len(store.rows))
	}
	if len(store.projects) != 1 {
		t.Fatalf("expected one Personal project, got %d", len(store.projects))
	}
}

func TestDeleteOnboardingKeyPreservesOnboardingStatus(t *testing.T) {
	store := newFakeOnboardingStore()
	svc := New(Dependencies{Onboarding: store})
	ctx := authedCtxFor("usr_delete_key")

	if _, err := svc.CompleteOnboarding(ctx, CompleteOnboardingInput{}); err != nil {
		t.Fatalf("initial onboarding: %v", err)
	}
	row := store.rows["usr_delete_key"]
	row.AnthropicKeyCiphertext = []byte("ciphertext")
	row.AnthropicKeyNonce = []byte("nonce")
	row.AnthropicKeyPrefix = "sk-ant-example..."
	row.AnthropicKeyLast4 = "1234"
	row.AadSalt = []byte("salt")
	store.rows["usr_delete_key"] = row

	if err := svc.DeleteOnboardingKey(ctx, "usr_delete_key"); err != nil {
		t.Fatalf("DeleteOnboardingKey: %v", err)
	}

	status, err := svc.GetOnboardingStatus(ctx, "usr_delete_key")
	if err != nil {
		t.Fatalf("GetOnboardingStatus post-delete: %v", err)
	}
	if !status.Complete {
		t.Fatalf("expected onboarding to remain complete after key deletion, got %#v", status)
	}
	postDeleteRow := store.rows["usr_delete_key"]
	if len(postDeleteRow.AnthropicKeyCiphertext) != 0 || len(postDeleteRow.AnthropicKeyNonce) != 0 || len(postDeleteRow.AadSalt) != 0 {
		t.Fatalf("expected key material cleared, got %#v", postDeleteRow)
	}
}

func TestDeleteOnboardingKeyNotFound(t *testing.T) {
	store := newFakeOnboardingStore()
	svc := New(Dependencies{Onboarding: store})

	err := svc.DeleteOnboardingKey(context.Background(), "usr_missing")
	if err == nil {
		t.Fatal("expected ONBOARDING_NOT_FOUND")
	}
	appErr, ok := err.(lib.AppError)
	if !ok || appErr.Code != "ONBOARDING_NOT_FOUND" {
		t.Fatalf("expected ONBOARDING_NOT_FOUND, got %#v", err)
	}
}

func TestCompleteOnboardingRequiresUserSession(t *testing.T) {
	store := newFakeOnboardingStore()
	svc := New(Dependencies{Onboarding: store})
	_, err := svc.CompleteOnboarding(context.Background(), CompleteOnboardingInput{})
	appErr, ok := err.(lib.AppError)
	if !ok || appErr.Code != "UNAUTHORIZED" {
		t.Fatalf("expected UNAUTHORIZED, got %#v", err)
	}
}

func TestCompleteOnboardingRejectsAdminBearer(t *testing.T) {
	store := newFakeOnboardingStore()
	svc := New(Dependencies{Onboarding: store})
	adminCtx := ContextWithAuthInfo(context.Background(), AuthInfo{IsAdmin: true, Scope: APIKeyScopeGlobal})
	_, err := svc.CompleteOnboarding(adminCtx, CompleteOnboardingInput{})
	appErr, ok := err.(lib.AppError)
	if !ok || appErr.Code != "UNAUTHORIZED" {
		t.Fatalf("expected UNAUTHORIZED for admin bearer, got %#v", err)
	}
}

// keyPrefix / keyLast4 remain available for the later provider-key settings
// flow. Keep edge cases covered while the feature moves out of onboarding.
func TestKeyPrefixAndLast4(t *testing.T) {
	if got := keyPrefix("sk-ant-fakekeythatishuge"); got != "sk-ant-fakekeythat..." {
		t.Fatalf("expected ellipsis on long key, got %q", got)
	}
	if got := keyPrefix("short"); got != "short" {
		t.Fatalf("expected pass-through for short key, got %q", got)
	}
	if got := keyLast4("abc"); got != "abc" {
		t.Fatalf("expected pass-through for sub-4-char key, got %q", got)
	}
	if got := keyLast4("longerkey"); got != "rkey" {
		t.Fatalf("expected last 4 chars, got %q", got)
	}
}

// Sanity: the active KEK loader from env is still reachable for the later
// provider-key storage path.
func TestLoadKEKsFromEnv_ValidatesHexLength(t *testing.T) {
	prev := os.Getenv("RELAY_DATA_ENCRYPTION_KEY")
	defer os.Setenv("RELAY_DATA_ENCRYPTION_KEY", prev)

	os.Setenv("RELAY_DATA_ENCRYPTION_KEY", "tooshort")
	if _, _, err := crypto.LoadKEKsFromEnv(); err == nil {
		t.Fatal("expected misconfigured for short key")
	}
}
