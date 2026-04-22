package config

import "testing"

func TestLoadPrefersAdminTokenForMCPEnablement(t *testing.T) {
	t.Setenv("RELAY_ADMIN_TOKEN", "admin-token")
	t.Setenv("RELAY_API_TOKEN", "legacy-admin-token")
	t.Setenv("RELAY_CLIENT_TOKEN", "")
	t.Setenv("RELAY_MCP_TOKEN", "")

	cfg := Load()
	if cfg.AdminToken != "admin-token" {
		t.Fatalf("expected RELAY_ADMIN_TOKEN to win, got %q", cfg.AdminToken)
	}
	if cfg.APIToken != "legacy-admin-token" {
		t.Fatalf("expected RELAY_API_TOKEN to still populate APIToken, got %q", cfg.APIToken)
	}
}

func TestLoadFallsBackAdminTokenFromLegacyAPIToken(t *testing.T) {
	t.Setenv("RELAY_ADMIN_TOKEN", "")
	t.Setenv("RELAY_API_TOKEN", "legacy-admin-token")

	cfg := Load()
	if cfg.AdminToken != "legacy-admin-token" {
		t.Fatalf("expected RELAY_API_TOKEN fallback for admin token, got %q", cfg.AdminToken)
	}
}
