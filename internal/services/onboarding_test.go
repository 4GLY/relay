package services

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
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
	row.OnboardingCompletedAt = nil
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

func newTestKEKs() (map[crypto.KEKVersion][]byte, crypto.KEKVersion) {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}
	return map[crypto.KEKVersion][]byte{1: key}, 1
}

// authedCtxFor sets a cookie-session AuthInfo so RequireUserAuth resolves a
// user-id (matching what the requireSessionOrAdmin middleware injects).
func authedCtxFor(userID string) context.Context {
	return ContextWithAuthInfo(context.Background(), AuthInfo{UserID: userID, Scope: APIKeyScopeGlobal})
}

// stubAnthropicServer returns an httptest server that responds with the given
// status code on every request. Restore() must be called to undo the
// validate-URL override.
func stubAnthropicServer(t *testing.T, status int) (cleanup func()) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(status)
	}))
	restore := SetValidateURL(srv.URL)
	return func() {
		restore()
		srv.Close()
	}
}

func newOnboardingService(t *testing.T, store *fakeOnboardingStore) Service {
	t.Helper()
	keks, active := newTestKEKs()
	return NewWithKEKs(Dependencies{Onboarding: store}, keks, active)
}

// T2: cookie-session user → 200 happy path. Steps both ok, project created,
// raw key not present anywhere on the result struct.
func TestCompleteOnboardingHappyPath(t *testing.T) {
	cleanup := stubAnthropicServer(t, http.StatusOK)
	defer cleanup()

	store := newFakeOnboardingStore()
	svc := newOnboardingService(t, store)

	ctx := authedCtxFor("usr_test")
	rawKey := "sk-ant-test-real-looking-1234"
	result, err := svc.CompleteOnboarding(ctx, CompleteOnboardingInput{AnthropicKey: rawKey})
	if err != nil {
		t.Fatalf("CompleteOnboarding returned error: %v", err)
	}
	if !result.OnboardingComplete {
		t.Fatalf("expected onboarding_complete=true, got %#v", result)
	}
	if result.DefaultProjectID == "" {
		t.Fatal("expected default_project_id to be set")
	}
	if len(result.Steps) != 2 || result.Steps[0].Status != StepOK || result.Steps[1].Status != StepOK {
		t.Fatalf("expected both steps ok, got %#v", result.Steps)
	}
	if result.AnthropicKeyPrefix == rawKey || result.AnthropicKeyLast4 == rawKey {
		t.Fatalf("raw key leaked into result: %#v", result)
	}
	if !strings.HasSuffix(result.AnthropicKeyLast4, "1234") {
		t.Fatalf("expected last4 to end with 1234, got %q", result.AnthropicKeyLast4)
	}
}

// T2 + T12: raw key is never marshaled or logged. The tests assert against
// the result object (no logger captured separately because the service does
// not log; F7 is enforced by the contract that error messages are static).
func TestCompleteOnboardingDoesNotLeakRawKey(t *testing.T) {
	cleanup := stubAnthropicServer(t, http.StatusOK)
	defer cleanup()

	store := newFakeOnboardingStore()
	svc := newOnboardingService(t, store)
	rawKey := "sk-ant-secretsuperhidden-zzzz"

	res, err := svc.CompleteOnboarding(authedCtxFor("usr_t12"), CompleteOnboardingInput{AnthropicKey: rawKey})
	if err != nil {
		t.Fatalf("CompleteOnboarding error: %v", err)
	}
	for _, field := range []string{res.AnthropicKeyPrefix, res.AnthropicKeyLast4, res.DefaultProjectID} {
		if strings.Contains(field, "secretsuperhidden") {
			t.Fatalf("raw key fragment leaked into result field %q", field)
		}
	}
	row := store.rows["usr_t12"]
	if bytes.Contains(row.AnthropicKeyCiphertext, []byte(rawKey)) {
		t.Fatal("ciphertext contains plaintext bytes — encryption did not happen")
	}
}

// T4: Anthropic 401 → INVALID_ANTHROPIC_KEY, retryable:false, steps[0].failed,
// steps[1].skipped. Mapping is locked (E4).
func TestCompleteOnboardingMaps401ToInvalidKey(t *testing.T) {
	cleanup := stubAnthropicServer(t, http.StatusUnauthorized)
	defer cleanup()

	store := newFakeOnboardingStore()
	svc := newOnboardingService(t, store)

	res, err := svc.CompleteOnboarding(authedCtxFor("usr_t4"), CompleteOnboardingInput{AnthropicKey: "sk-ant-bad"})
	if err == nil {
		t.Fatal("expected error on 401")
	}
	appErr, ok := err.(lib.AppError)
	if !ok || appErr.Code != "INVALID_ANTHROPIC_KEY" || appErr.Retryable {
		t.Fatalf("expected INVALID_ANTHROPIC_KEY retryable:false, got %#v", err)
	}
	if len(res.Steps) != 2 || res.Steps[0].Status != StepFailed || res.Steps[1].Status != StepSkipped {
		t.Fatalf("expected steps[failed,skipped], got %#v", res.Steps)
	}
}

// T10: per-step status — table-driven assertion of every Anthropic status
// code we care about, including the ANTHROPIC_QUOTA case where the key is
// reported ok but the quota chip is failed.
func TestCompleteOnboardingPerStepStatus(t *testing.T) {
	cases := []struct {
		name      string
		httpCode  int
		wantCode  string // "" = no error
		wantSteps [2]StepStatus
		retryable bool
	}{
		{name: "200 ok", httpCode: 200, wantCode: "", wantSteps: [2]StepStatus{StepOK, StepOK}},
		{name: "401 invalid", httpCode: 401, wantCode: "INVALID_ANTHROPIC_KEY", wantSteps: [2]StepStatus{StepFailed, StepSkipped}},
		{name: "403 invalid", httpCode: 403, wantCode: "INVALID_ANTHROPIC_KEY", wantSteps: [2]StepStatus{StepFailed, StepSkipped}},
		{name: "429 quota", httpCode: 429, wantCode: "ANTHROPIC_QUOTA", wantSteps: [2]StepStatus{StepOK, StepFailed}},
		{name: "500 unreachable", httpCode: 500, wantCode: "ANTHROPIC_UNREACHABLE", wantSteps: [2]StepStatus{StepFailed, StepSkipped}, retryable: true},
		{name: "503 unreachable", httpCode: 503, wantCode: "ANTHROPIC_UNREACHABLE", wantSteps: [2]StepStatus{StepFailed, StepSkipped}, retryable: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cleanup := stubAnthropicServer(t, tc.httpCode)
			defer cleanup()
			store := newFakeOnboardingStore()
			svc := newOnboardingService(t, store)

			res, err := svc.CompleteOnboarding(authedCtxFor("usr_t10"), CompleteOnboardingInput{AnthropicKey: "sk-ant-x"})
			if tc.wantCode == "" {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
			} else {
				appErr, ok := err.(lib.AppError)
				if !ok || appErr.Code != tc.wantCode {
					t.Fatalf("expected %s, got %#v", tc.wantCode, err)
				}
				if appErr.Retryable != tc.retryable {
					t.Fatalf("expected retryable=%v, got %v", tc.retryable, appErr.Retryable)
				}
			}
			if len(res.Steps) != 2 {
				t.Fatalf("expected 2 steps, got %d", len(res.Steps))
			}
			if res.Steps[0].Status != tc.wantSteps[0] || res.Steps[1].Status != tc.wantSteps[1] {
				t.Fatalf("expected steps %v, got %#v", tc.wantSteps, res.Steps)
			}
		})
	}
}

// T5: re-onboard rotates the AAD salt. Two successive POSTs with two
// different keys leave one row whose ciphertext+salt reflect the second key,
// and the salt MUST differ between the two calls.
func TestCompleteOnboardingReOnboardRegeneratesSalt(t *testing.T) {
	cleanup := stubAnthropicServer(t, http.StatusOK)
	defer cleanup()

	store := newFakeOnboardingStore()
	svc := newOnboardingService(t, store)
	ctx := authedCtxFor("usr_t5")

	if _, err := svc.CompleteOnboarding(ctx, CompleteOnboardingInput{AnthropicKey: "sk-ant-key-one1111"}); err != nil {
		t.Fatalf("first onboarding: %v", err)
	}
	firstRow := store.rows["usr_t5"]
	if len(firstRow.AadSalt) != 16 {
		t.Fatalf("expected 16-byte salt, got %d bytes", len(firstRow.AadSalt))
	}
	firstSalt := append([]byte(nil), firstRow.AadSalt...)
	firstProject := firstRow.DefaultProjectID

	if _, err := svc.CompleteOnboarding(ctx, CompleteOnboardingInput{AnthropicKey: "sk-ant-key-two2222"}); err != nil {
		t.Fatalf("second onboarding: %v", err)
	}
	secondRow := store.rows["usr_t5"]

	if bytes.Equal(firstSalt, secondRow.AadSalt) {
		t.Fatal("expected aad_salt to be regenerated on re-onboard, got identical bytes")
	}
	if !strings.HasSuffix(secondRow.AnthropicKeyLast4, "2222") {
		t.Fatalf("expected last4=2222 after re-onboard, got %q", secondRow.AnthropicKeyLast4)
	}
	if secondRow.DefaultProjectID != firstProject {
		t.Fatalf("expected default project preserved across re-onboard, got %q then %q", firstProject, secondRow.DefaultProjectID)
	}
}

// T8: DELETE smoke — POST then DELETE leaves the row but with NULL key
// material; status reflects "incomplete"; re-POST succeeds and regenerates a
// fresh aad_salt.
func TestDeleteOnboardingKeySmoke(t *testing.T) {
	cleanup := stubAnthropicServer(t, http.StatusOK)
	defer cleanup()

	store := newFakeOnboardingStore()
	svc := newOnboardingService(t, store)
	ctx := authedCtxFor("usr_t8")

	if _, err := svc.CompleteOnboarding(ctx, CompleteOnboardingInput{AnthropicKey: "sk-ant-original1234"}); err != nil {
		t.Fatalf("initial onboarding: %v", err)
	}
	preDeleteSalt := append([]byte(nil), store.rows["usr_t8"].AadSalt...)

	if err := svc.DeleteOnboardingKey(ctx, "usr_t8"); err != nil {
		t.Fatalf("DeleteOnboardingKey: %v", err)
	}

	status, err := svc.GetOnboardingStatus(ctx, "usr_t8")
	if err != nil {
		t.Fatalf("GetOnboardingStatus post-delete: %v", err)
	}
	if status.Complete {
		t.Fatalf("expected complete=false after delete, got %#v", status)
	}

	postDeleteRow := store.rows["usr_t8"]
	if len(postDeleteRow.AnthropicKeyCiphertext) != 0 {
		t.Fatalf("expected ciphertext cleared, got %d bytes", len(postDeleteRow.AnthropicKeyCiphertext))
	}

	if _, err := svc.CompleteOnboarding(ctx, CompleteOnboardingInput{AnthropicKey: "sk-ant-replacement5678"}); err != nil {
		t.Fatalf("re-onboard after delete: %v", err)
	}
	if bytes.Equal(preDeleteSalt, store.rows["usr_t8"].AadSalt) {
		t.Fatal("expected new aad_salt after re-onboard following delete")
	}
}

// Re-onboarding without an existing row hits the not-found branch; verify
// it surfaces the right code instead of a generic error.
func TestDeleteOnboardingKeyNotFound(t *testing.T) {
	store := newFakeOnboardingStore()
	svc := newOnboardingService(t, store)

	err := svc.DeleteOnboardingKey(context.Background(), "usr_missing")
	if err == nil {
		t.Fatal("expected ONBOARDING_NOT_FOUND")
	}
	appErr, ok := err.(lib.AppError)
	if !ok || appErr.Code != "ONBOARDING_NOT_FOUND" {
		t.Fatalf("expected ONBOARDING_NOT_FOUND, got %#v", err)
	}
}

// Empty key after trim → MISSING_REQUIRED_FIELDS, no probe issued.
func TestCompleteOnboardingRejectsEmptyKey(t *testing.T) {
	store := newFakeOnboardingStore()
	svc := newOnboardingService(t, store)
	_, err := svc.CompleteOnboarding(authedCtxFor("usr_x"), CompleteOnboardingInput{AnthropicKey: "   "})
	appErr, ok := err.(lib.AppError)
	if !ok || appErr.Code != "MISSING_REQUIRED_FIELDS" {
		t.Fatalf("expected MISSING_REQUIRED_FIELDS, got %#v", err)
	}
}

// Wrong prefix → INVALID_ANTHROPIC_KEY before any probe; steps must still be
// populated so the caller can render Frame 3.
func TestCompleteOnboardingRejectsWrongPrefix(t *testing.T) {
	store := newFakeOnboardingStore()
	svc := newOnboardingService(t, store)
	res, err := svc.CompleteOnboarding(authedCtxFor("usr_y"), CompleteOnboardingInput{AnthropicKey: "key-without-prefix"})
	appErr, ok := err.(lib.AppError)
	if !ok || appErr.Code != "INVALID_ANTHROPIC_KEY" {
		t.Fatalf("expected INVALID_ANTHROPIC_KEY, got %#v", err)
	}
	if len(res.Steps) != 2 || res.Steps[0].Status != StepFailed || res.Steps[1].Status != StepSkipped {
		t.Fatalf("expected steps[failed,skipped] on prefix rejection, got %#v", res.Steps)
	}
}

// Without a session, RequireUserAuth fires UNAUTHORIZED before anything else.
func TestCompleteOnboardingRequiresUserSession(t *testing.T) {
	store := newFakeOnboardingStore()
	svc := newOnboardingService(t, store)
	_, err := svc.CompleteOnboarding(context.Background(), CompleteOnboardingInput{AnthropicKey: "sk-ant-x"})
	appErr, ok := err.(lib.AppError)
	if !ok || appErr.Code != "UNAUTHORIZED" {
		t.Fatalf("expected UNAUTHORIZED, got %#v", err)
	}
}

// Admin-bearer (no UserID) MUST be rejected — onboarding is user-only because
// the row is keyed by user_id (R1 acceptance — error code "UNAUTHORIZED" per
// E10 correction).
func TestCompleteOnboardingRejectsAdminBearer(t *testing.T) {
	store := newFakeOnboardingStore()
	svc := newOnboardingService(t, store)
	adminCtx := ContextWithAuthInfo(context.Background(), AuthInfo{IsAdmin: true, Scope: APIKeyScopeGlobal})
	_, err := svc.CompleteOnboarding(adminCtx, CompleteOnboardingInput{AnthropicKey: "sk-ant-x"})
	appErr, ok := err.(lib.AppError)
	if !ok || appErr.Code != "UNAUTHORIZED" {
		t.Fatalf("expected UNAUTHORIZED for admin bearer, got %#v", err)
	}
}

// MISCONFIGURED if no KEKs are loaded — boot fail-closed (F1).
func TestCompleteOnboardingFailsClosedWithoutKEKs(t *testing.T) {
	store := newFakeOnboardingStore()
	svc := New(Dependencies{Onboarding: store}) // no KEKs
	_, err := svc.CompleteOnboarding(authedCtxFor("usr_no_keks"), CompleteOnboardingInput{AnthropicKey: "sk-ant-x"})
	appErr, ok := err.(lib.AppError)
	if !ok || appErr.Code != "MISCONFIGURED" {
		t.Fatalf("expected MISCONFIGURED, got %#v", err)
	}
}

// keyPrefix / keyLast4 edge cases.
func TestKeyPrefixAndLast4(t *testing.T) {
	if got := keyPrefix("sk-ant-fakekeythatishuge"); !strings.HasSuffix(got, "...") {
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

// Sanity: the active KEK loader from env is reachable through the package
// (T13 placeholder — full migration smoke is a CI step, not a unit test).
func TestLoadKEKsFromEnv_ValidatesHexLength(t *testing.T) {
	prev := os.Getenv("RELAY_DATA_ENCRYPTION_KEY")
	defer os.Setenv("RELAY_DATA_ENCRYPTION_KEY", prev)

	os.Setenv("RELAY_DATA_ENCRYPTION_KEY", "tooshort")
	if _, _, err := crypto.LoadKEKsFromEnv(); err == nil {
		t.Fatal("expected misconfigured for short key")
	}
}

// T13 placeholder — CI migration smoke (boot against Postgres without
// pgcrypto extension and verify migration 0009 succeeds) is intentionally
// not implemented as a unit test. Kept as a marker so the task is visible
// during code review.
func TestT13MigrationSmokePlaceholder(t *testing.T) {
	t.Skip("CI-level: runs migration 0009 against pgcrypto-less Postgres; tracked outside the unit test harness")
}
