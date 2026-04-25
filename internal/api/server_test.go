package api

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

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

type fakeArtifactStore struct{ items []domain.Artifact }

func (s *fakeArtifactStore) CreateArtifact(_ context.Context, artifact domain.Artifact) (domain.Artifact, error) {
	s.items = append(s.items, artifact)
	return artifact, nil
}
func (s *fakeArtifactStore) CountByProject(_ context.Context, projectID string) (int, error) {
	count := 0
	for _, item := range s.items {
		if item.ProjectID == projectID {
			count++
		}
	}
	return count, nil
}
func (s *fakeArtifactStore) ListByProject(_ context.Context, projectID string) ([]domain.Artifact, error) {
	var items []domain.Artifact
	for _, item := range s.items {
		if item.ProjectID == projectID {
			items = append(items, item)
		}
	}
	return items, nil
}

type fakeDecisionStore struct{ items []domain.Decision }

func (s *fakeDecisionStore) CreateDecision(_ context.Context, decision domain.Decision) (domain.Decision, error) {
	s.items = append(s.items, decision)
	return decision, nil
}
func (s *fakeDecisionStore) CountByProject(_ context.Context, projectID string) (int, error) {
	count := 0
	for _, item := range s.items {
		if item.ProjectID == projectID {
			count++
		}
	}
	return count, nil
}
func (s *fakeDecisionStore) ListByProject(_ context.Context, projectID string) ([]domain.Decision, error) {
	var items []domain.Decision
	for _, item := range s.items {
		if item.ProjectID == projectID {
			items = append(items, item)
		}
	}
	return items, nil
}

type fakeOpenQuestionStore struct{ items []domain.OpenQuestion }

func (s *fakeOpenQuestionStore) CreateOpenQuestion(_ context.Context, question domain.OpenQuestion) (domain.OpenQuestion, error) {
	s.items = append(s.items, question)
	return question, nil
}
func (s *fakeOpenQuestionStore) CountByProject(_ context.Context, projectID string) (int, error) {
	count := 0
	for _, item := range s.items {
		if item.ProjectID == projectID {
			count++
		}
	}
	return count, nil
}
func (s *fakeOpenQuestionStore) ListByProject(_ context.Context, projectID string) ([]domain.OpenQuestion, error) {
	var items []domain.OpenQuestion
	for _, item := range s.items {
		if item.ProjectID == projectID {
			items = append(items, item)
		}
	}
	return items, nil
}

type fakePacketStore struct{ latest domain.Packet }

func (s *fakePacketStore) CreatePacket(_ context.Context, packet domain.Packet) (domain.Packet, error) {
	s.latest = packet
	return packet, nil
}
func (s *fakePacketStore) LatestByProject(_ context.Context, _ string) (domain.Packet, error) {
	return s.latest, nil
}

type fakePacketSnapshotStore struct {
	latest domain.PacketSnapshot
}

func (s *fakePacketSnapshotStore) CreatePacketSnapshot(_ context.Context, snapshot domain.PacketSnapshot) (domain.PacketSnapshot, error) {
	s.latest = snapshot
	return snapshot, nil
}

func (s *fakePacketSnapshotStore) GetPacketSnapshot(_ context.Context, id string) (domain.PacketSnapshot, error) {
	if s.latest.ID == id {
		return s.latest, nil
	}
	return domain.PacketSnapshot{}, lib.NotFound("PACKET_SNAPSHOT_NOT_FOUND", "packet snapshot not found")
}

func (s *fakePacketSnapshotStore) LatestPacketSnapshotByProject(_ context.Context, projectID string, packetKind string, target string) (domain.PacketSnapshot, error) {
	if s.latest.ProjectID == projectID && s.latest.PacketKind == packetKind && s.latest.Target == target {
		return s.latest, nil
	}
	return domain.PacketSnapshot{}, lib.NotFound("PACKET_SNAPSHOT_NOT_FOUND", "packet snapshot not found")
}

func (s *fakePacketSnapshotStore) MakePacketSnapshotPublic(_ context.Context, snapshotID string, publicToken string, ogImagePath string) (domain.PacketSnapshot, error) {
	if s.latest.ID != snapshotID {
		return domain.PacketSnapshot{}, lib.NotFound("PACKET_SNAPSHOT_NOT_FOUND", "packet snapshot not found")
	}
	s.latest.PublicReadable = true
	s.latest.PublicToken = publicToken
	s.latest.OGImagePath = ogImagePath
	return s.latest, nil
}

func (s *fakePacketSnapshotStore) RevokePacketSnapshotPublic(_ context.Context, snapshotID string) (domain.PacketSnapshot, error) {
	if s.latest.ID != snapshotID {
		return domain.PacketSnapshot{}, lib.NotFound("PACKET_SNAPSHOT_NOT_FOUND", "packet snapshot not found")
	}
	s.latest.PublicReadable = false
	return s.latest, nil
}

func (s *fakePacketSnapshotStore) GetPacketSnapshotByPublicToken(_ context.Context, token string) (domain.PacketSnapshot, error) {
	if s.latest.PublicToken == token && s.latest.PublicReadable {
		return s.latest, nil
	}
	return domain.PacketSnapshot{}, lib.NotFound("PUBLIC_SNAPSHOT_NOT_FOUND", "public snapshot not found")
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

func TestListenAndServeRequiresEffectiveAdminToken(t *testing.T) {
	if err := requireStartupAdminToken(config.Config{}); err == nil {
		t.Fatal("expected error")
	} else if got := err.Error(); !strings.Contains(got, "RELAY_ADMIN_TOKEN or RELAY_API_TOKEN is required for relay-api") {
		t.Fatalf("expected missing admin token error, got %v", err)
	}

	if err := requireStartupAdminToken(config.Config{AdminToken: "admin-token"}); err != nil {
		t.Fatalf("expected admin token to satisfy startup validation, got %v", err)
	}

	if err := requireStartupAdminToken(config.Config{APIToken: "legacy-token"}); err != nil {
		t.Fatalf("expected legacy api token to satisfy startup validation, got %v", err)
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

func TestHandleProjectGraphUsesProjectID(t *testing.T) {
	projectID := lib.ProjectID("relay")
	handler := testHandler(projectID)

	req := httptest.NewRequest(http.MethodGet, "/v1/projects/"+projectID+"/graph", nil)
	rec := httptest.NewRecorder()

	handler.handleProjectShow(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte(`"command":"relay project graph"`)) {
		t.Fatalf("expected graph command, got %s", rec.Body.String())
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte(`"type":"derived_from"`)) {
		t.Fatalf("expected derived_from edges, got %s", rec.Body.String())
	}
}

func TestHandleProjectRetrieveUsesQueryParam(t *testing.T) {
	projectID := lib.ProjectID("relay")
	handler := testHandler(projectID)

	req := httptest.NewRequest(http.MethodGet, "/v1/projects/"+projectID+"/retrieve?query=design&limit=5", nil)
	rec := httptest.NewRecorder()

	handler.handleProjectShow(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte(`"command":"relay project retrieve"`)) {
		t.Fatalf("expected retrieve command, got %s", rec.Body.String())
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte(`"kind":"artifact"`)) {
		t.Fatalf("expected artifact hit in retrieval response, got %s", rec.Body.String())
	}
}

func TestHandleLatestPacketSnapshotUsesProjectID(t *testing.T) {
	projectID := lib.ProjectID("relay")
	handler := testHandler(projectID)

	req := httptest.NewRequest(http.MethodGet, "/v1/projects/"+projectID+"/packet-snapshots/latest?target=codex", nil)
	rec := httptest.NewRecorder()

	handler.handleProjectShow(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte(`"command":"relay latest packet snapshot"`)) {
		t.Fatalf("expected latest snapshot command, got %s", rec.Body.String())
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte(`"snapshot_id":"psnap_1"`)) {
		t.Fatalf("expected snapshot id, got %s", rec.Body.String())
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

func TestProtectedRoutesAcceptProjectScopedKeyForBoundProject(t *testing.T) {
	projectID := lib.ProjectID("relay")
	key := "relay_live_testtoken"
	keyStore := &fakeAPIKeyStore{
		itemsByHash: map[string]domain.APIKey{
			lib.TokenHash(key): {
				ID:          "key_1",
				Name:        "agent",
				TokenHash:   lib.TokenHash(key),
				TokenPrefix: lib.TokenPrefix(key),
				Scope:       services.APIKeyScopeProject,
				ProjectID:   projectID,
			},
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

func TestProtectedRoutesRejectUnknownPersistedKeyScope(t *testing.T) {
	projectID := lib.ProjectID("relay")
	key := "relay_live_testtoken"
	keyStore := &fakeAPIKeyStore{
		itemsByHash: map[string]domain.APIKey{
			lib.TokenHash(key): {
				ID:          "key_1",
				Name:        "agent",
				TokenHash:   lib.TokenHash(key),
				TokenPrefix: lib.TokenPrefix(key),
				Scope:       "corrupted",
			},
		},
	}
	mux := buildMux(testHandler(projectID), config.Config{APIToken: "admin-token"}, testRuntime(projectID, keyStore))

	req := httptest.NewRequest(http.MethodGet, "/v1/projects/"+projectID, nil)
	req.Header.Set("Authorization", "Bearer "+key)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestProtectedRoutesRejectProjectScopedKeyForDifferentProject(t *testing.T) {
	projectID := lib.ProjectID("relay")
	otherID := lib.ProjectID("other")
	key := "relay_live_testtoken"
	keyStore := &fakeAPIKeyStore{
		itemsByHash: map[string]domain.APIKey{
			lib.TokenHash(key): {
				ID:          "key_1",
				Name:        "agent",
				TokenHash:   lib.TokenHash(key),
				TokenPrefix: lib.TokenPrefix(key),
				Scope:       services.APIKeyScopeProject,
				ProjectID:   projectID,
			},
		},
	}
	mux := buildMux(testHandler(projectID), config.Config{APIToken: "admin-token"}, testRuntime(projectID, keyStore))

	req := httptest.NewRequest(http.MethodGet, "/v1/projects/"+otherID, nil)
	req.Header.Set("Authorization", "Bearer "+key)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestProtectedRoutesRejectMalformedProjectScopedKey(t *testing.T) {
	projectID := lib.ProjectID("relay")
	key := "relay_live_testtoken"
	keyStore := &fakeAPIKeyStore{
		itemsByHash: map[string]domain.APIKey{
			lib.TokenHash(key): {
				ID:          "key_1",
				Name:        "agent",
				TokenHash:   lib.TokenHash(key),
				TokenPrefix: lib.TokenPrefix(key),
				Scope:       services.APIKeyScopeProject,
			},
		},
	}
	mux := buildMux(testHandler(projectID), config.Config{APIToken: "admin-token"}, testRuntime(projectID, keyStore))

	req := httptest.NewRequest(http.MethodGet, "/v1/projects/"+projectID, nil)
	req.Header.Set("Authorization", "Bearer "+key)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestProtectedRoutesAllowGlobalKeyAcrossProjects(t *testing.T) {
	projectID := lib.ProjectID("relay")
	otherID := lib.ProjectID("other")
	key := "relay_live_testtoken"
	keyStore := &fakeAPIKeyStore{
		itemsByHash: map[string]domain.APIKey{
			lib.TokenHash(key): {
				ID:          "key_1",
				Name:        "agent",
				TokenHash:   lib.TokenHash(key),
				TokenPrefix: lib.TokenPrefix(key),
				Scope:       services.APIKeyScopeGlobal,
			},
		},
	}
	mux := buildMux(testHandler(projectID), config.Config{APIToken: "admin-token"}, testRuntime(projectID, keyStore))

	req := httptest.NewRequest(http.MethodGet, "/v1/projects/"+otherID, nil)
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

func TestAdminRoutesFailClosedWithoutAdminToken(t *testing.T) {
	keyStore := &fakeAPIKeyStore{}
	mux := buildMux(testHandler(lib.ProjectID("relay"), keyStore), config.Config{}, testRuntime(lib.ProjectID("relay"), keyStore))

	tests := []struct {
		name   string
		method string
		path   string
		body   []byte
	}{
		{name: "list", method: http.MethodGet, path: "/v1/api-keys"},
		{name: "issue", method: http.MethodPost, path: "/v1/api-keys/issue", body: []byte(`{"name":"agent"}`)},
		{name: "revoke", method: http.MethodPost, path: "/v1/api-keys/revoke", body: []byte(`{"key_id":"key_1"}`)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, bytes.NewReader(tt.body))
			if tt.body != nil {
				req.Header.Set("Content-Type", "application/json")
			}
			rec := httptest.NewRecorder()

			mux.ServeHTTP(rec, req)

			if rec.Code != http.StatusInternalServerError {
				t.Fatalf("expected 500, got %d body=%s", rec.Code, rec.Body.String())
			}
			if !bytes.Contains(rec.Body.Bytes(), []byte("MISCONFIGURED")) {
				t.Fatalf("expected misconfigured response, got %s", rec.Body.String())
			}
		})
	}
}

func TestAdminRoutesPreferConfiguredAdminToken(t *testing.T) {
	keyStore := &fakeAPIKeyStore{}
	mux := buildMux(testHandler(lib.ProjectID("relay"), keyStore), config.Config{
		AdminToken: "admin-token",
		APIToken:   "client-token",
	}, testRuntime(lib.ProjectID("relay"), keyStore))

	req := httptest.NewRequest(http.MethodGet, "/v1/api-keys", nil)
	req.Header.Set("Authorization", "Bearer admin-token")
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/v1/api-keys", nil)
	req.Header.Set("Authorization", "Bearer client-token")
	rec = httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for client token on admin route, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestBearerProtectedRoutesFailClosedWithoutBearerConfig(t *testing.T) {
	mux := buildMux(testHandler(lib.ProjectID("relay")), config.Config{}, testRuntime(lib.ProjectID("relay")))

	req := httptest.NewRequest(http.MethodGet, "/v1/projects/relay", nil)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d body=%s", rec.Code, rec.Body.String())
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte("MISCONFIGURED")) {
		t.Fatalf("expected misconfigured response, got %s", rec.Body.String())
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

func TestIssueAPIKeyRouteCreatesProjectScopedKey(t *testing.T) {
	projectID := lib.ProjectID("relay")
	keyStore := &fakeAPIKeyStore{}
	mux := buildMux(testHandler(projectID, keyStore), config.Config{APIToken: "admin-token"}, testRuntime(projectID, keyStore))

	req := httptest.NewRequest(http.MethodPost, "/v1/api-keys/issue", bytes.NewReader([]byte(`{"name":"agent","scope":"project","project":"relay"}`)))
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
	if keyStore.created[0].Scope != services.APIKeyScopeProject {
		t.Fatalf("expected project scope, got %q", keyStore.created[0].Scope)
	}
	if keyStore.created[0].ProjectID != projectID {
		t.Fatalf("expected project id %q, got %q", projectID, keyStore.created[0].ProjectID)
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte(`"scope":"project"`)) {
		t.Fatalf("expected scope in response body, got %s", rec.Body.String())
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

func TestCaptureRouteRejectsUnknownJSONField(t *testing.T) {
	projectID := lib.ProjectID("relay")
	mux := buildMux(testHandler(projectID), config.Config{APIToken: "admin-token"}, testRuntime(projectID))

	req := httptest.NewRequest(http.MethodPost, "/v1/capture", bytes.NewReader([]byte(`{"project":"relay","source":"chat","body":"hello","unexpected":"value"}`)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer admin-token")
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", rec.Code, rec.Body.String())
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte(`"code":"UNKNOWN_JSON_FIELD"`)) {
		t.Fatalf("expected unknown field error, got %s", rec.Body.String())
	}
}

func TestCaptureRouteAcceptsExtraArtifacts(t *testing.T) {
	projectID := lib.ProjectID("relay")
	mux := buildMux(testHandler(projectID), config.Config{APIToken: "admin-token"}, testRuntime(projectID))

	req := httptest.NewRequest(http.MethodPost, "/v1/capture", bytes.NewReader([]byte(`{
		"project":"relay",
		"source":"chat",
		"body":"hello",
		"extra_artifacts":[
			{"type":"code_path","source_path":"internal/services/capture.go"},
			{"type":"pr_diff","source_path":"scripts/evals/fixtures/pr-diffs/api-first-boundary.diff.md","trust_level":"trusted"}
		]
	}`)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer admin-token")
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte(`"created_artifact_ids"`)) {
		t.Fatalf("expected created_artifact_ids in response, got %s", rec.Body.String())
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte(`"art_`)) {
		t.Fatalf("expected artifact ids in response, got %s", rec.Body.String())
	}
}

func TestCaptureRouteRejectsMalformedUTF8(t *testing.T) {
	projectID := lib.ProjectID("relay")
	mux := buildMux(testHandler(projectID), config.Config{APIToken: "admin-token"}, testRuntime(projectID))

	req := httptest.NewRequest(http.MethodPost, "/v1/capture", bytes.NewReader([]byte("{\"project\":\"relay\",\"source\":\"chat\",\"body\":\"\xff\"}")))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer admin-token")
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", rec.Code, rec.Body.String())
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte(`"code":"INVALID_JSON"`)) {
		t.Fatalf("expected invalid json error, got %s", rec.Body.String())
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte("malformed UTF-8")) {
		t.Fatalf("expected malformed utf-8 message, got %s", rec.Body.String())
	}
}

func TestCaptureRouteRejectsOversizedBody(t *testing.T) {
	projectID := lib.ProjectID("relay")
	mux := buildMux(testHandler(projectID), config.Config{APIToken: "admin-token"}, testRuntime(projectID))

	oversized := []byte(`{"project":"relay","source":"chat","body":"` + strings.Repeat("a", maxJSONRequestBodyBytes) + `"}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/capture", bytes.NewReader(oversized))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer admin-token")
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected 413, got %d body=%s", rec.Code, rec.Body.String())
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte(`"code":"REQUEST_TOO_LARGE"`)) {
		t.Fatalf("expected request too large error, got %s", rec.Body.String())
	}
}

func TestIssueAPIKeyRouteRejectsInvalidScopeWithValidationError(t *testing.T) {
	keyStore := &fakeAPIKeyStore{}
	mux := buildMux(testHandler(lib.ProjectID("relay"), keyStore), config.Config{APIToken: "admin-token"}, testRuntime(lib.ProjectID("relay"), keyStore))

	req := httptest.NewRequest(http.MethodPost, "/v1/api-keys/issue", bytes.NewReader([]byte(`{"name":"agent","scope":"invalid"}`)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer admin-token")
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", rec.Code, rec.Body.String())
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte(`"code":"INVALID_API_KEY_SCOPE"`)) {
		t.Fatalf("expected invalid scope code, got %s", rec.Body.String())
	}
}

func TestMCPRouteRejectsOversizedBody(t *testing.T) {
	mux := buildMux(testHandler(lib.ProjectID("relay")), config.Config{APIToken: "admin-token"}, testRuntime(lib.ProjectID("relay")))

	oversized := []byte(strings.Repeat("a", maxJSONRequestBodyBytes+1))
	req := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewReader(oversized))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	req.Header.Set("Authorization", "Bearer admin-token")
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected 413, got %d body=%s", rec.Code, rec.Body.String())
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte(`"code":"REQUEST_TOO_LARGE"`)) {
		t.Fatalf("expected request too large error, got %s", rec.Body.String())
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

func TestMCPRouteFailsClosedWithoutBearerConfig(t *testing.T) {
	mux := buildMux(testHandler(lib.ProjectID("relay")), config.Config{}, testRuntime(lib.ProjectID("relay")))

	req := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewReader(mcpInitializeBody()))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d body=%s", rec.Code, rec.Body.String())
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte("MISCONFIGURED")) {
		t.Fatalf("expected misconfigured response, got %s", rec.Body.String())
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

func TestMCPRouteAcceptsConfiguredAdminToken(t *testing.T) {
	mux := buildMux(testHandler(lib.ProjectID("relay")), config.Config{
		AdminToken: "admin-token",
		APIToken:   "client-token",
	}, testRuntime(lib.ProjectID("relay")))

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
	otherProjectID := lib.ProjectID("other")
	return Handler{
		services: services.New(services.Dependencies{
			Projects: &fakeProjectStore{
				projects: map[string]domain.Project{
					"relay": {ID: projectID, Name: "relay"},
					"other": {ID: otherProjectID, Name: "other"},
				},
			},
			Notes: &fakeNoteStore{
				items: []domain.Note{
					{ID: "note_1", ProjectID: projectID, Source: "chat", Body: "hello"},
				},
			},
			Artifacts: &fakeArtifactStore{
				items: []domain.Artifact{
					{ID: "art_1", ProjectID: projectID, Type: "design_doc", SourcePath: "docs/design.md", TrustLevel: "trusted"},
				},
			},
			Decisions: &fakeDecisionStore{
				items: []domain.Decision{
					{ID: "dec_1", ProjectID: projectID, Summary: "preserve style continuity", SourceNoteIDs: []string{"note_1"}, SourceArtifactIDs: []string{"art_1"}},
				},
			},
			OpenQuestions: &fakeOpenQuestionStore{
				items: []domain.OpenQuestion{
					{ID: "oq_1", ProjectID: projectID, Summary: "how to rank retrieval evidence", SourceArtifactIDs: []string{"art_1"}},
				},
			},
			Packets: &fakePacketStore{},
			PacketSnapshots: &fakePacketSnapshotStore{
				latest: domain.PacketSnapshot{
					ID:            "psnap_1",
					ProjectID:     projectID,
					PacketKind:    "resume",
					Target:        "codex",
					SchemaVersion: "relay.packet.v1",
					RenderedBody:  "latest body",
					StyleCues:     []byte(`[]`),
					CreatedAt:     time.Now(),
				},
			},
			APIKeys: apiKeys,
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
