package api

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"relay/internal/app"
	"relay/internal/config"
	"relay/internal/domain"
	"relay/internal/lib"
	"relay/internal/services"
)

type fakeProjectStore struct {
	projects map[string]domain.Project
}

func (s *fakeProjectStore) EnsureProject(_ context.Context, project domain.Project) (domain.Project, error) {
	if s.projects == nil {
		s.projects = map[string]domain.Project{}
	}
	s.projects[project.Name] = project
	return project, nil
}

func (s *fakeProjectStore) GetByName(_ context.Context, name string) (domain.Project, error) {
	project, ok := s.projects[name]
	if !ok {
		return domain.Project{}, lib.NotFound("PROJECT_NOT_FOUND", "project not found")
	}
	return project, nil
}

func (s *fakeProjectStore) GetByID(_ context.Context, id string) (domain.Project, error) {
	for _, project := range s.projects {
		if project.ID == id {
			return project, nil
		}
	}
	return domain.Project{}, lib.NotFound("PROJECT_NOT_FOUND", "project not found")
}

type fakeNoteStore struct{ items []domain.Note }

func (s *fakeNoteStore) CreateNote(_ context.Context, note domain.Note) (domain.Note, error) {
	s.items = append(s.items, note)
	return note, nil
}
func (s *fakeNoteStore) CountByProject(_ context.Context, projectID string) (int, error) {
	count := 0
	for _, item := range s.items {
		if item.ProjectID == projectID {
			count++
		}
	}
	return count, nil
}
func (s *fakeNoteStore) ListByProject(_ context.Context, projectID string) ([]domain.Note, error) {
	var items []domain.Note
	for _, item := range s.items {
		if item.ProjectID == projectID {
			items = append(items, item)
		}
	}
	return items, nil
}

type fakeArtifactStore struct{}

func (s *fakeArtifactStore) CreateArtifact(_ context.Context, artifact domain.Artifact) (domain.Artifact, error) {
	return artifact, nil
}
func (s *fakeArtifactStore) CountByProject(_ context.Context, _ string) (int, error) {
	return 0, nil
}
func (s *fakeArtifactStore) ListByProject(_ context.Context, _ string) ([]domain.Artifact, error) {
	return nil, nil
}

type fakeDecisionStore struct{}

func (s *fakeDecisionStore) CreateDecision(_ context.Context, decision domain.Decision) (domain.Decision, error) {
	return decision, nil
}
func (s *fakeDecisionStore) CountByProject(_ context.Context, _ string) (int, error) {
	return 0, nil
}
func (s *fakeDecisionStore) ListByProject(_ context.Context, _ string) ([]domain.Decision, error) {
	return nil, nil
}

type fakeOpenQuestionStore struct{}

func (s *fakeOpenQuestionStore) CreateOpenQuestion(_ context.Context, question domain.OpenQuestion) (domain.OpenQuestion, error) {
	return question, nil
}
func (s *fakeOpenQuestionStore) CountByProject(_ context.Context, _ string) (int, error) {
	return 0, nil
}
func (s *fakeOpenQuestionStore) ListByProject(_ context.Context, _ string) ([]domain.OpenQuestion, error) {
	return nil, nil
}

type fakePacketStore struct{ latest domain.Packet }

func (s *fakePacketStore) CreatePacket(_ context.Context, packet domain.Packet) (domain.Packet, error) {
	s.latest = packet
	return packet, nil
}
func (s *fakePacketStore) LatestByProject(_ context.Context, _ string) (domain.Packet, error) {
	return s.latest, nil
}

type fakeAPIKeyStore struct {
	itemsByHash map[string]domain.APIKey
	created     []domain.APIKey
}

func (s *fakeAPIKeyStore) CreateAPIKey(_ context.Context, key domain.APIKey) (domain.APIKey, error) {
	if s.itemsByHash == nil {
		s.itemsByHash = map[string]domain.APIKey{}
	}
	s.itemsByHash[key.TokenHash] = key
	s.created = append(s.created, key)
	return key, nil
}

func (s *fakeAPIKeyStore) GetByTokenHash(_ context.Context, tokenHash string) (domain.APIKey, error) {
	if s.itemsByHash == nil {
		return domain.APIKey{}, lib.NotFound("API_KEY_NOT_FOUND", "api key not found")
	}
	key, ok := s.itemsByHash[tokenHash]
	if !ok || key.Revoked {
		return domain.APIKey{}, lib.NotFound("API_KEY_NOT_FOUND", "api key not found")
	}
	return key, nil
}

func (s *fakeAPIKeyStore) ListAPIKeys(_ context.Context) ([]domain.APIKey, error) {
	var items []domain.APIKey
	for _, item := range s.itemsByHash {
		items = append(items, item)
	}
	for _, item := range s.created {
		seen := false
		for _, existing := range items {
			if existing.ID == item.ID {
				seen = true
				break
			}
		}
		if !seen {
			items = append(items, item)
		}
	}
	return items, nil
}

func (s *fakeAPIKeyStore) RevokeAPIKey(_ context.Context, keyID string) (domain.APIKey, error) {
	for hash, item := range s.itemsByHash {
		if item.ID == keyID {
			item.Revoked = true
			s.itemsByHash[hash] = item
			return item, nil
		}
	}
	for i, item := range s.created {
		if item.ID == keyID {
			item.Revoked = true
			s.created[i] = item
			return item, nil
		}
	}
	return domain.APIKey{}, lib.NotFound("API_KEY_NOT_FOUND_BY_ID", "api key not found")
}

func TestHealthz(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()

	handleHealth(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestHandleProjectShowUsesProjectID(t *testing.T) {
	projectID := lib.ProjectID("relay")
	handler := testHandler(projectID)

	req := httptest.NewRequest(http.MethodGet, "/v1/projects/"+projectID, bytes.NewReader(nil))
	rec := httptest.NewRecorder()

	handler.handleProjectShow(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte(projectID)) {
		t.Fatalf("expected response to include project id, got %s", rec.Body.String())
	}
}

func TestProtectedRoutesRequireBearerToken(t *testing.T) {
	projectID := lib.ProjectID("relay")
	mux := buildMux(testHandler(projectID), config.Config{APIToken: "secret-token"}, testRuntime(projectID))

	req := httptest.NewRequest(http.MethodGet, "/v1/projects/"+projectID, nil)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestProtectedRoutesRejectWrongBearerToken(t *testing.T) {
	projectID := lib.ProjectID("relay")
	mux := buildMux(testHandler(projectID), config.Config{APIToken: "secret-token"}, testRuntime(projectID))

	req := httptest.NewRequest(http.MethodGet, "/v1/projects/"+projectID, nil)
	req.Header.Set("Authorization", "Bearer wrong-token")
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestProtectedRoutesAcceptCorrectBearerToken(t *testing.T) {
	projectID := lib.ProjectID("relay")
	mux := buildMux(testHandler(projectID), config.Config{APIToken: "secret-token"}, testRuntime(projectID))

	req := httptest.NewRequest(http.MethodGet, "/v1/projects/"+projectID, nil)
	req.Header.Set("Authorization", "Bearer secret-token")
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestHealthzStaysOpenWhenBearerTokenConfigured(t *testing.T) {
	mux := buildMux(testHandler(lib.ProjectID("relay")), config.Config{APIToken: "secret-token"}, testRuntime(lib.ProjectID("relay")))

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestProtectedRoutesAcceptIssuedAPIKey(t *testing.T) {
	projectID := lib.ProjectID("relay")
	key := "relay_live_testtoken"
	keyStore := &fakeAPIKeyStore{
		itemsByHash: map[string]domain.APIKey{
			lib.TokenHash(key): {ID: "key_1", Name: "agent", TokenHash: lib.TokenHash(key), TokenPrefix: lib.TokenPrefix(key)},
		},
	}
	mux := buildMux(testHandler(projectID), config.Config{APIToken: "admin-token"}, testRuntime(projectID, keyStore))

	req := httptest.NewRequest(http.MethodGet, "/v1/projects/"+projectID, nil)
	req.Header.Set("Authorization", "Bearer "+key)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestIssueAPIKeyRouteRequiresAdminToken(t *testing.T) {
	keyStore := &fakeAPIKeyStore{}
	mux := buildMux(testHandler(lib.ProjectID("relay")), config.Config{APIToken: "admin-token"}, testRuntime(lib.ProjectID("relay"), keyStore))

	req := httptest.NewRequest(http.MethodPost, "/v1/api-keys/issue", bytes.NewReader([]byte(`{"name":"agent"}`)))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestIssueAPIKeyRouteCreatesKey(t *testing.T) {
	keyStore := &fakeAPIKeyStore{}
	mux := buildMux(testHandler(lib.ProjectID("relay"), keyStore), config.Config{APIToken: "admin-token"}, testRuntime(lib.ProjectID("relay"), keyStore))

	req := httptest.NewRequest(http.MethodPost, "/v1/api-keys/issue", bytes.NewReader([]byte(`{"name":"agent"}`)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer admin-token")
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	if len(keyStore.created) != 1 {
		t.Fatalf("expected one created key, got %d", len(keyStore.created))
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte(`"token"`)) {
		t.Fatalf("expected token in response body, got %s", rec.Body.String())
	}
}

func TestListAPIKeysRouteReturnsItems(t *testing.T) {
	keyStore := &fakeAPIKeyStore{
		itemsByHash: map[string]domain.APIKey{
			"hash": {ID: "key_1", Name: "agent", TokenHash: "hash", TokenPrefix: "relay_live_abc", Revoked: false},
		},
	}
	mux := buildMux(testHandler(lib.ProjectID("relay"), keyStore), config.Config{APIToken: "admin-token"}, testRuntime(lib.ProjectID("relay"), keyStore))

	req := httptest.NewRequest(http.MethodGet, "/v1/api-keys", nil)
	req.Header.Set("Authorization", "Bearer admin-token")
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte(`"key_id":"key_1"`)) {
		t.Fatalf("expected key listing in response, got %s", rec.Body.String())
	}
}

func TestRevokeAPIKeyRouteRevokesIssuedKey(t *testing.T) {
	key := "relay_live_testtoken"
	keyStore := &fakeAPIKeyStore{
		itemsByHash: map[string]domain.APIKey{
			lib.TokenHash(key): {ID: "key_1", Name: "agent", TokenHash: lib.TokenHash(key), TokenPrefix: lib.TokenPrefix(key)},
		},
	}
	mux := buildMux(testHandler(lib.ProjectID("relay"), keyStore), config.Config{APIToken: "admin-token"}, testRuntime(lib.ProjectID("relay"), keyStore))

	req := httptest.NewRequest(http.MethodPost, "/v1/api-keys/revoke", bytes.NewReader([]byte(`{"key_id":"key_1"}`)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer admin-token")
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	req2 := httptest.NewRequest(http.MethodGet, "/v1/projects/"+lib.ProjectID("relay"), nil)
	req2.Header.Set("Authorization", "Bearer "+key)
	rec2 := httptest.NewRecorder()
	mux.ServeHTTP(rec2, req2)

	if rec2.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 after revoke, got %d body=%s", rec2.Code, rec2.Body.String())
	}
}

func TestMCPRouteRequiresBearerToken(t *testing.T) {
	mux := buildMux(testHandler(lib.ProjectID("relay")), config.Config{APIToken: "admin-token"}, testRuntime(lib.ProjectID("relay")))

	req := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewReader(mcpInitializeBody()))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestMCPRouteAcceptsAdminBearerToken(t *testing.T) {
	mux := buildMux(testHandler(lib.ProjectID("relay")), config.Config{APIToken: "admin-token"}, testRuntime(lib.ProjectID("relay")))

	req := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewReader(mcpInitializeBody()))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	req.Header.Set("Authorization", "Bearer admin-token")
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestMCPRouteAcceptsIssuedAPIKey(t *testing.T) {
	key := "relay_live_testtoken"
	keyStore := &fakeAPIKeyStore{
		itemsByHash: map[string]domain.APIKey{
			lib.TokenHash(key): {ID: "key_1", Name: "agent", TokenHash: lib.TokenHash(key), TokenPrefix: lib.TokenPrefix(key)},
		},
	}
	mux := buildMux(testHandler(lib.ProjectID("relay"), keyStore), config.Config{APIToken: "admin-token"}, testRuntime(lib.ProjectID("relay"), keyStore))

	req := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewReader(mcpInitializeBody()))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	req.Header.Set("Authorization", "Bearer "+key)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func mcpInitializeBody() []byte {
	return []byte(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-11-05","capabilities":{},"clientInfo":{"name":"server-test","version":"0.0.1"}}}`)
}

func testHandler(projectID string, apiKeyStores ...*fakeAPIKeyStore) Handler {
	var apiKeys *fakeAPIKeyStore
	if len(apiKeyStores) > 0 {
		apiKeys = apiKeyStores[0]
	}
	return Handler{
		services: services.New(services.Dependencies{
			Projects: &fakeProjectStore{
				projects: map[string]domain.Project{
					"relay": {ID: projectID, Name: "relay"},
				},
			},
			Notes: &fakeNoteStore{
				items: []domain.Note{
					{ID: "note_1", ProjectID: projectID, Source: "chat", Body: "hello"},
				},
			},
			Artifacts:     &fakeArtifactStore{},
			Decisions:     &fakeDecisionStore{},
			OpenQuestions: &fakeOpenQuestionStore{},
			Packets:       &fakePacketStore{},
			APIKeys:       apiKeys,
		}),
	}
}

func testRuntime(projectID string, apiKeyStores ...*fakeAPIKeyStore) app.Runtime {
	var apiKeys *fakeAPIKeyStore
	if len(apiKeyStores) > 0 {
		apiKeys = apiKeyStores[0]
	}
	runtime := app.Runtime{
		Services: testHandler(projectID, apiKeyStores...).services,
	}
	if apiKeys != nil {
		runtime.APIKeys = apiKeys
	}
	return runtime
}
