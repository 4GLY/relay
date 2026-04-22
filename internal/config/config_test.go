package config

import (
	"testing"
)

func TestLoadKeepsMCPTokenSeparateFromAPIToken(t *testing.T) {
	t.Setenv("RELAY_API_TOKEN", "admin-token")
	t.Setenv("RELAY_MCP_TOKEN", "")

	cfg := Load()
	if cfg.APIToken != "admin-token" {
		t.Fatalf("expected admin token from RELAY_API_TOKEN, got %q", cfg.APIToken)
	}
	if cfg.MCPToken != "" {
		t.Fatalf("expected empty MCP token when RELAY_MCP_TOKEN is unset, got %q", cfg.MCPToken)
	}
}

func TestLoadUsesExplicitMCPToken(t *testing.T) {
	t.Setenv("RELAY_API_TOKEN", "admin-token")
	t.Setenv("RELAY_MCP_TOKEN", "client-token")

	cfg := Load()
	if cfg.APIToken != "admin-token" {
		t.Fatalf("expected admin token from RELAY_API_TOKEN, got %q", cfg.APIToken)
	}
	if cfg.MCPToken != "client-token" {
		t.Fatalf("expected MCP token from RELAY_MCP_TOKEN, got %q", cfg.MCPToken)
	}
}
