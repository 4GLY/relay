package mcpserver

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/auth"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"relay/internal/domain"
	"relay/internal/lib"
	"relay/internal/relayapi"
	"relay/internal/services"
)

type stubBackend struct {
	adminEnabled bool
}

func (b stubBackend) Health(_ context.Context) (relayapi.HealthResult, error) {
	return relayapi.HealthResult{Status: "ok"}, nil
}

func (b stubBackend) Capture(_ context.Context, _ services.CaptureInput) (services.CaptureResult, error) {
	return services.CaptureResult{}, nil
}

func (b stubBackend) Promote(_ context.Context, _ services.PromoteInput) (services.PromoteResult, error) {
	return services.PromoteResult{}, nil
}

func (b stubBackend) BuildPacket(_ context.Context, _ services.PacketBuildInput) (services.PacketBuildResult, error) {
	return services.PacketBuildResult{}, nil
}

func (b stubBackend) Show(_ context.Context, _ string) (services.ShowResult, error) {
	return services.ShowResult{}, nil
}

func (b stubBackend) ProjectRetrieve(_ context.Context, _ string, _ string, _ int) (services.ProjectRetrieveResult, error) {
	return services.ProjectRetrieveResult{}, nil
}

func (b stubBackend) IssueAPIKey(_ context.Context, _ services.IssueAPIKeyInput) (services.IssueAPIKeyResult, error) {
	return services.IssueAPIKeyResult{}, nil
}

func (b stubBackend) ListAPIKeys(_ context.Context) (services.ListAPIKeysResult, error) {
	return services.ListAPIKeysResult{}, nil
}

func (b stubBackend) RevokeAPIKey(_ context.Context, _ services.RevokeAPIKeyInput) (services.RevokeAPIKeyResult, error) {
	return services.RevokeAPIKeyResult{}, nil
}

func (b stubBackend) HasAdminToken() bool {
	return b.adminEnabled
}

func (b stubBackend) BaseURL() string {
	return "https://relay.4gly.dev"
}

func TestListToolsDeterministicWithoutAdmin(t *testing.T) {
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/healthz" {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"ok":       true,
				"command":  "healthz",
				"data":     map[string]any{"status": "ok"},
				"warnings": []string{},
			})
			return
		}
		http.NotFound(w, r)
	}))
	defer api.Close()

	server := New(relayapi.NewClient(relayapi.Config{
		BaseURL:     api.URL,
		ClientToken: "client-token",
	}))

	serverTransport, clientTransport := mcp.NewInMemoryTransports()
	ctx := context.Background()
	go func() {
		_ = server.Run(ctx, serverTransport)
	}()

	client := mcp.NewClient(&mcp.Implementation{Name: "relay-mcp-test-client", Version: "v1.0.0"}, nil)
	session, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer session.Close()

	result, err := session.ListTools(ctx, nil)
	if err != nil {
		t.Fatalf("list tools: %v", err)
	}

	var names []string
	for _, tool := range result.Tools {
		names = append(names, tool.Name)
	}

	expected := []string{
		"relay_build_packet",
		"relay_capture",
		"relay_health",
		"relay_promote",
		"relay_retrieve_project",
		"relay_show_project",
	}
	if len(names) != len(expected) {
		t.Fatalf("expected %d tools, got %d: %v", len(expected), len(names), names)
	}
	for i, name := range expected {
		if names[i] != name {
			t.Fatalf("tool order mismatch at %d: got %q want %q", i, names[i], name)
		}
	}
}

func TestListToolsIncludesAdminWhenEnabled(t *testing.T) {
	server := New(stubBackend{adminEnabled: true})

	serverTransport, clientTransport := mcp.NewInMemoryTransports()
	ctx := context.Background()
	go func() {
		_ = server.Run(ctx, serverTransport)
	}()

	client := mcp.NewClient(&mcp.Implementation{Name: "relay-mcp-test-client", Version: "v1.0.0"}, nil)
	session, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer session.Close()

	result, err := session.ListTools(ctx, nil)
	if err != nil {
		t.Fatalf("list tools: %v", err)
	}

	names := map[string]bool{}
	for _, tool := range result.Tools {
		names[tool.Name] = true
	}

	for _, want := range []string{
		"relay_issue_api_key",
		"relay_list_api_keys",
		"relay_revoke_api_key",
	} {
		if !names[want] {
			t.Fatalf("expected admin tool %q in tool list: %#v", want, names)
		}
	}
}

func TestPublicMCPRejectsStyleMemoryMutationTools(t *testing.T) {
	server := New(stubBackend{})

	serverTransport, clientTransport := mcp.NewInMemoryTransports()
	ctx := context.Background()
	go func() {
		_ = server.Run(ctx, serverTransport)
	}()

	client := mcp.NewClient(&mcp.Implementation{Name: "relay-mcp-test-client", Version: "v1.0.0"}, nil)
	session, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer session.Close()

	for _, name := range []string{
		"relay_write_judgment_trace",
		"relay_create_heuristic_proposal",
		"relay_review_heuristic_proposal",
		"relay_update_approved_heuristic",
	} {
		result, err := session.CallTool(ctx, &mcp.CallToolParams{
			Name:      name,
			Arguments: map[string]any{},
		})
		if err == nil && (result == nil || !result.IsError) {
			t.Fatalf("expected public MCP to reject style-memory mutation tool %q, result=%#v", name, result)
		}
	}
}

func TestBuildPacketToolForwardsStyleSelectors(t *testing.T) {
	backend := &packetInputBackend{}
	server := New(backend)

	serverTransport, clientTransport := mcp.NewInMemoryTransports()
	ctx := context.Background()
	go func() {
		_ = server.Run(ctx, serverTransport)
	}()

	client := mcp.NewClient(&mcp.Implementation{Name: "relay-mcp-test-client", Version: "v1.0.0"}, nil)
	session, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer session.Close()

	result, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "relay_build_packet",
		Arguments: map[string]any{
			"project":            "relay",
			"workflow":           "design_handoff",
			"artifact_type":      "design_doc",
			"task_summary":       "continue the V1 handoff proof",
			"disable_style_cues": true,
			"persist_snapshot":   true,
			"idempotency_key":    "packet-1",
		},
	})
	if err != nil {
		t.Fatalf("call build packet tool: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected tool error: %#v", result)
	}
	if backend.input.Project != "relay" ||
		backend.input.Type != "resume" ||
		backend.input.Target != "codex" ||
		backend.input.Workflow != "design_handoff" ||
		backend.input.ArtifactType != "design_doc" ||
		backend.input.TaskSummary != "continue the V1 handoff proof" ||
		!backend.input.DisableStyleCues ||
		!backend.input.PersistSnapshot ||
		backend.input.IdempotencyKey != "packet-1" {
		t.Fatalf("unexpected packet input: %#v", backend.input)
	}
}

func TestCaptureToolForwardsExtraArtifacts(t *testing.T) {
	backend := &captureInputBackend{}
	server := New(backend)

	serverTransport, clientTransport := mcp.NewInMemoryTransports()
	ctx := context.Background()
	go func() {
		_ = server.Run(ctx, serverTransport)
	}()

	client := mcp.NewClient(&mcp.Implementation{Name: "relay-mcp-test-client", Version: "v1.0.0"}, nil)
	session, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer session.Close()

	result, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "relay_capture",
		Arguments: map[string]any{
			"project": "relay",
			"body":    "remember the latest code paths",
			"extra_artifacts": []map[string]any{
				{"type": "code_path", "source_path": "internal/services/capture.go"},
				{"type": "changed_files", "source_path": "scripts/evals/fixtures/changed-files/api-first-boundary.txt", "trust_level": "trusted"},
			},
		},
	})
	if err != nil {
		t.Fatalf("call capture tool: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected tool error: %#v", result)
	}
	if backend.input.Project != "relay" || backend.input.Body != "remember the latest code paths" {
		t.Fatalf("unexpected capture input: %#v", backend.input)
	}
	if len(backend.input.ExtraArtifacts) != 2 {
		t.Fatalf("expected 2 extra artifacts, got %#v", backend.input.ExtraArtifacts)
	}
	if backend.input.ExtraArtifacts[0].Type != "code_path" || backend.input.ExtraArtifacts[1].Type != "changed_files" {
		t.Fatalf("unexpected extra artifact forwarding: %#v", backend.input.ExtraArtifacts)
	}
}

type packetInputBackend struct {
	stubBackend
	input services.PacketBuildInput
}

func (b *packetInputBackend) BuildPacket(_ context.Context, input services.PacketBuildInput) (services.PacketBuildResult, error) {
	b.input = input
	return services.PacketBuildResult{ProjectID: lib.ProjectID(input.Project), Type: input.Type, Target: input.Target}, nil
}

type captureInputBackend struct {
	stubBackend
	input services.CaptureInput
}

func (b *captureInputBackend) Capture(_ context.Context, input services.CaptureInput) (services.CaptureResult, error) {
	b.input = input
	return services.CaptureResult{ProjectID: lib.ProjectID(input.Project)}, nil
}

type retrieveInputBackend struct {
	stubBackend
	input services.ProjectRetrieveInput
}

func (b *retrieveInputBackend) ProjectRetrieve(_ context.Context, projectID string, query string, limit int) (services.ProjectRetrieveResult, error) {
	b.input = services.ProjectRetrieveInput{ProjectID: projectID, Query: query, Limit: limit}
	return services.ProjectRetrieveResult{ProjectID: projectID, Query: query}, nil
}

func TestRetrieveProjectToolForwardsQuery(t *testing.T) {
	backend := &retrieveInputBackend{}
	server := New(backend)

	serverTransport, clientTransport := mcp.NewInMemoryTransports()
	ctx := context.Background()
	go func() {
		_ = server.Run(ctx, serverTransport)
	}()

	client := mcp.NewClient(&mcp.Implementation{Name: "relay-mcp-test-client", Version: "v1.0.0"}, nil)
	session, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer session.Close()

	result, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "relay_retrieve_project",
		Arguments: map[string]any{
			"project_id": "proj_relay",
			"query":      "continue api packet boundary work",
			"limit":      5,
		},
	})
	if err != nil {
		t.Fatalf("call retrieve tool: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected tool error: %#v", result)
	}
	if backend.input.ProjectID != "proj_relay" || backend.input.Query != "continue api packet boundary work" || backend.input.Limit != 5 {
		t.Fatalf("unexpected retrieve input: %#v", backend.input)
	}
}

func TestServiceBackedStdIOAdminToolIssuesAPIKey(t *testing.T) {
	keys := &fakeAPIKeyStore{}
	service := services.New(services.Dependencies{
		Projects:      &servicesTestProjectStore{},
		Notes:         &servicesTestNoteStore{},
		Artifacts:     &servicesTestArtifactStore{},
		Decisions:     &servicesTestDecisionStore{},
		OpenQuestions: &servicesTestOpenQuestionStore{},
		Packets:       &servicesTestPacketStore{},
		APIKeys:       keys,
	})

	server := NewFromService(service, "https://relay.4gly.dev", true)
	serverTransport, clientTransport := mcp.NewInMemoryTransports()
	ctx := context.Background()
	go func() {
		_ = server.Run(ctx, serverTransport)
	}()

	client := mcp.NewClient(&mcp.Implementation{Name: "relay-mcp-test-client", Version: "v1.0.0"}, nil)
	session, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer session.Close()

	result, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "relay_issue_api_key",
		Arguments: map[string]any{
			"name": "agent",
		},
	})
	if err != nil {
		t.Fatalf("call admin tool: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected tool error: %#v", result)
	}
	if len(keys.created) != 1 {
		t.Fatalf("expected one created key, got %d", len(keys.created))
	}
}

func TestCaptureToolCallsRelayAPI(t *testing.T) {
	var authHeader string
	var body map[string]any
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/capture":
			authHeader = r.Header.Get("Authorization")
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode body: %v", err)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"ok":      true,
				"command": "relay capture",
				"data": map[string]any{
					"project_id":           "proj_test",
					"created_note_ids":     []string{"note_1"},
					"created_artifact_ids": []string{},
				},
				"warnings": []string{},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer api.Close()

	server := New(relayapi.NewClient(relayapi.Config{
		BaseURL:     api.URL,
		ClientToken: "client-token",
	}))

	serverTransport, clientTransport := mcp.NewInMemoryTransports()
	ctx := context.Background()
	go func() {
		_ = server.Run(ctx, serverTransport)
	}()

	client := mcp.NewClient(&mcp.Implementation{Name: "relay-mcp-test-client", Version: "v1.0.0"}, nil)
	session, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer session.Close()

	result, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "relay_capture",
		Arguments: map[string]any{
			"note": "from mcp test",
		},
	})
	if err != nil {
		t.Fatalf("call tool: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected tool error: %#v", result)
	}
	if authHeader != "Bearer client-token" {
		t.Fatalf("expected bearer auth header, got %q", authHeader)
	}
	if body["project"] != "" || body["source"] != "" || body["note"] != "from mcp test" || body["body"] != "" {
		t.Fatalf("unexpected request body: %#v", body)
	}
}

type servicesTestProjectStore struct{}

func (s *servicesTestProjectStore) EnsureProject(_ context.Context, project domain.Project) (domain.Project, error) {
	return project, nil
}

func (s *servicesTestProjectStore) GetByName(_ context.Context, name string) (domain.Project, error) {
	return domain.Project{ID: lib.ProjectID(name), Name: name}, nil
}

func (s *servicesTestProjectStore) GetByID(_ context.Context, id string) (domain.Project, error) {
	return domain.Project{ID: id, Name: id}, nil
}

type servicesTestNoteStore struct{}

func (s *servicesTestNoteStore) CreateNote(_ context.Context, note domain.Note) (domain.Note, error) {
	return note, nil
}

func (s *servicesTestNoteStore) CountByProject(_ context.Context, _ string) (int, error) {
	return 0, nil
}

func (s *servicesTestNoteStore) ListByProject(_ context.Context, _ string) ([]domain.Note, error) {
	return nil, nil
}

type servicesTestArtifactStore struct{}

func (s *servicesTestArtifactStore) CreateArtifact(_ context.Context, artifact domain.Artifact) (domain.Artifact, error) {
	return artifact, nil
}

func (s *servicesTestArtifactStore) CountByProject(_ context.Context, _ string) (int, error) {
	return 0, nil
}

func (s *servicesTestArtifactStore) ListByProject(_ context.Context, _ string) ([]domain.Artifact, error) {
	return nil, nil
}

type servicesTestDecisionStore struct{}

func (s *servicesTestDecisionStore) CreateDecision(_ context.Context, decision domain.Decision) (domain.Decision, error) {
	return decision, nil
}

func (s *servicesTestDecisionStore) CountByProject(_ context.Context, _ string) (int, error) {
	return 0, nil
}

func (s *servicesTestDecisionStore) ListByProject(_ context.Context, _ string) ([]domain.Decision, error) {
	return nil, nil
}

type servicesTestOpenQuestionStore struct{}

func (s *servicesTestOpenQuestionStore) CreateOpenQuestion(_ context.Context, question domain.OpenQuestion) (domain.OpenQuestion, error) {
	return question, nil
}

func (s *servicesTestOpenQuestionStore) CountByProject(_ context.Context, _ string) (int, error) {
	return 0, nil
}

func (s *servicesTestOpenQuestionStore) ListByProject(_ context.Context, _ string) ([]domain.OpenQuestion, error) {
	return nil, nil
}

type servicesTestPacketStore struct{}

func (s *servicesTestPacketStore) CreatePacket(_ context.Context, packet domain.Packet) (domain.Packet, error) {
	return packet, nil
}

func (s *servicesTestPacketStore) LatestByProject(_ context.Context, _ string) (domain.Packet, error) {
	return domain.Packet{}, nil
}

type fakeAPIKeyStore struct {
	created []domain.APIKey
}

func (s *fakeAPIKeyStore) CreateAPIKey(_ context.Context, key domain.APIKey) (domain.APIKey, error) {
	s.created = append(s.created, key)
	return key, nil
}

func (s *fakeAPIKeyStore) GetByTokenHash(_ context.Context, _ string) (domain.APIKey, error) {
	return domain.APIKey{}, nil
}

func (s *fakeAPIKeyStore) ListAPIKeys(_ context.Context) ([]domain.APIKey, error) {
	return s.created, nil
}

func (s *fakeAPIKeyStore) RevokeAPIKey(_ context.Context, keyID string) (domain.APIKey, error) {
	for i, key := range s.created {
		if key.ID == keyID {
			s.created[i].Revoked = true
			return s.created[i], nil
		}
	}
	return domain.APIKey{}, nil
}

func TestHTTPStreamableTransport(t *testing.T) {
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/healthz" {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"ok":       true,
				"command":  "healthz",
				"data":     map[string]any{"status": "ok"},
				"warnings": []string{},
			})
			return
		}
		http.NotFound(w, r)
	}))
	defer api.Close()

	relayServer := New(relayapi.NewClient(relayapi.Config{
		BaseURL:     api.URL,
		ClientToken: "client-token",
	}))

	handler := mcp.NewStreamableHTTPHandler(func(_ *http.Request) *mcp.Server {
		return relayServer.Server()
	}, &mcp.StreamableHTTPOptions{Stateless: true})
	httpServer := httptest.NewServer(handler)
	defer httpServer.Close()

	client := mcp.NewClient(&mcp.Implementation{Name: "relay-mcp-http-client", Version: "v1.0.0"}, nil)
	transport := &mcp.StreamableClientTransport{
		Endpoint:             httpServer.URL,
		DisableStandaloneSSE: true,
	}

	session, err := client.Connect(context.Background(), transport, nil)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer session.Close()

	result, err := session.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatalf("list tools: %v", err)
	}
	if len(result.Tools) == 0 {
		t.Fatal("expected tools from http mcp server")
	}
}

func TestHTTPBearerMiddlewareRejectsMissingToken(t *testing.T) {
	verifier := func(_ context.Context, token string, _ *http.Request) (*auth.TokenInfo, error) {
		if token == "relay-mcp-token" {
			return &auth.TokenInfo{Expiration: time.Now().Add(time.Hour)}, nil
		}
		return nil, auth.ErrInvalidToken
	}
	handler := auth.RequireBearerToken(verifier, nil)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest(http.MethodPost, "/mcp", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}
