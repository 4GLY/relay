package main

import (
	"testing"

	"relay/internal/config"
)

func TestAdminToolsEnabledUsesAdminToken(t *testing.T) {
	if adminToolsEnabled(config.Config{AdminToken: "admin-token"}) != true {
		t.Fatal("expected admin tools to be enabled when admin token is configured")
	}
	if adminToolsEnabled(config.Config{APIToken: "api-token"}) != false {
		t.Fatal("expected api token alone to not enable stdio admin tools")
	}
}
