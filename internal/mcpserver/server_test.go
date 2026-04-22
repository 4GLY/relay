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

	"relay/internal/relayapi"
)

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
			"project": "relay",
			"source":  "chat",
			"body":    "from mcp test",
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
	if body["project"] != "relay" || body["body"] != "from mcp test" {
		t.Fatalf("unexpected request body: %#v", body)
	}
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
