package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestUsagePrintedWithoutArgs(t *testing.T) {
	var out bytes.Buffer
	if err := Run(nil, strings.NewReader(""), &out, &out); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if !strings.Contains(out.String(), "relay capture") {
		t.Fatalf("expected usage output, got %q", out.String())
	}
	if !strings.Contains(out.String(), "relay migrate") {
		t.Fatalf("expected migrate in usage output, got %q", out.String())
	}
}

func TestCaptureJSONScaffold(t *testing.T) {
	var out bytes.Buffer
	input := strings.NewReader(`{"project":"relay","source":"chat","body":"hello"}`)
	if err := Run([]string{"capture", "--stdin-json"}, input, &out, &out); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if !strings.Contains(out.String(), `"MISCONFIGURED"`) {
		t.Fatalf("expected misconfigured envelope, got %q", out.String())
	}
}

func TestMigrateRequiresDatabaseURL(t *testing.T) {
	t.Setenv("RELAY_DATABASE_URL", "")

	var out bytes.Buffer
	if err := Run([]string{"migrate"}, strings.NewReader(""), &out, &out); err == nil {
		t.Fatal("expected migrate to fail without RELAY_DATABASE_URL")
	}
}
