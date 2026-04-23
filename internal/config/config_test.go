package config

import (
	"testing"
	"time"
)

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

func TestLoadCuratorWorkerConfig(t *testing.T) {
	t.Setenv("RELAY_CURATOR_WORKER_ID", "worker-a")
	t.Setenv("RELAY_CURATOR_PROVIDER", "rule-based")
	t.Setenv("RELAY_CURATOR_MODEL", "noop-model")
	t.Setenv("RELAY_CURATOR_BATCH_SIZE", "9")
	t.Setenv("RELAY_CURATOR_POLL_INTERVAL", "2s")
	t.Setenv("RELAY_CURATOR_LEASE_DURATION", "45s")
	t.Setenv("RELAY_CURATOR_RETRY_BACKOFF", "3s")
	t.Setenv("RELAY_CURATOR_MAX_ATTEMPTS", "7")

	cfg := Load()
	if cfg.CuratorWorkerID != "worker-a" ||
		cfg.CuratorProvider != "rule-based" ||
		cfg.CuratorModel != "noop-model" ||
		cfg.CuratorBatchSize != 9 ||
		cfg.CuratorPollInterval != 2*time.Second ||
		cfg.CuratorLeaseDuration != 45*time.Second ||
		cfg.CuratorRetryBackoff != 3*time.Second ||
		cfg.CuratorMaxAttempts != 7 {
		t.Fatalf("unexpected curator config: %#v", cfg)
	}
}
