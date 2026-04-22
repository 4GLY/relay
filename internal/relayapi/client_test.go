package relayapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"relay/internal/services"
)

func TestLoadConfigFromEnvKeepsAdminAndClientTokensSeparate(t *testing.T) {
	t.Setenv("RELAY_ADMIN_TOKEN", "admin-token")
	t.Setenv("RELAY_API_TOKEN", "legacy-admin-token")
	t.Setenv("RELAY_CLIENT_TOKEN", "client-token")
	t.Setenv("RELAY_TOKEN", "")

	cfg := LoadConfigFromEnv()
	if cfg.AdminToken != "admin-token" {
		t.Fatalf("expected RELAY_ADMIN_TOKEN to win, got %q", cfg.AdminToken)
	}
	if cfg.ClientToken != "client-token" {
		t.Fatalf("expected RELAY_CLIENT_TOKEN to populate client token, got %q", cfg.ClientToken)
	}
}

func TestLoadConfigFromEnvDoesNotCopyAdminTokenIntoClientToken(t *testing.T) {
	t.Setenv("RELAY_ADMIN_TOKEN", "")
	t.Setenv("RELAY_API_TOKEN", "admin-token")
	t.Setenv("RELAY_CLIENT_TOKEN", "")
	t.Setenv("RELAY_TOKEN", "")

	cfg := LoadConfigFromEnv()
	if cfg.AdminToken != "admin-token" {
		t.Fatalf("expected RELAY_API_TOKEN to populate admin token, got %q", cfg.AdminToken)
	}
	if cfg.ClientToken != "" {
		t.Fatalf("expected empty client token when no client credential is configured, got %q", cfg.ClientToken)
	}
}

func TestNewClientDoesNotFallbackClientTokenToAdminToken(t *testing.T) {
	client := NewClient(Config{
		BaseURL:    "https://relay.example",
		AdminToken: "admin-token",
	})

	if client.clientToken != "" {
		t.Fatalf("expected empty client token without explicit client credentials, got %q", client.clientToken)
	}
	if client.adminToken != "admin-token" {
		t.Fatalf("expected admin token to remain available, got %q", client.adminToken)
	}
}

func TestClientCaptureDoesNotUseAdminTokenFallback(t *testing.T) {
	var authHeader string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"ok":false,"command":"relay capture","error":{"code":"UNAUTHORIZED","message":"missing or invalid bearer token","retryable":false,"missing_fields":[]}}`))
	}))
	defer server.Close()

	client := NewClient(Config{
		BaseURL:    server.URL,
		AdminToken: "admin-token",
	})

	_, err := client.Capture(context.Background(), services.CaptureInput{})
	if err == nil {
		t.Fatal("expected capture to fail without a client token")
	}
	if authHeader != "" {
		t.Fatalf("expected no authorization header without a client token, got %q", authHeader)
	}
}
