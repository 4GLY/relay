package main

import (
	"testing"

	"relay/internal/config"
)

func TestAdminToolsEnabledFollowsLoadedConfigPrecedence(t *testing.T) {
	t.Setenv("RELAY_ADMIN_TOKEN", "")
	t.Setenv("RELAY_API_TOKEN", "api-token")

	cfg := config.Load()
	if cfg.AdminToken != "api-token" {
		t.Fatalf("expected RELAY_API_TOKEN to populate AdminToken, got %q", cfg.AdminToken)
	}
	if !adminToolsEnabled(cfg) {
		t.Fatal("expected admin tools to be enabled when legacy api token populates admin token")
	}

	if adminToolsEnabled(config.Config{AdminToken: "admin-token"}) != true {
		t.Fatal("expected admin tools to be enabled when admin token is configured")
	}
}
