package ogimage

import (
	"bytes"
	"errors"
	"io/fs"
	"testing"
)

func TestGenerateProducesPNG(t *testing.T) {
	if !fontsAvailable() {
		t.Skip("ttf font files not embedded; skipping generator test")
	}
	png, err := Generate(Options{
		ProjectName: "relay",
		Subtitle:    "Snapshot from Relay",
	})
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}
	if len(png) == 0 {
		t.Fatal("Generate returned empty bytes")
	}
	magic := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}
	if !bytes.HasPrefix(png, magic) {
		t.Fatalf("expected PNG magic prefix, got %x", png[:8])
	}
}

func TestGenerateEmptyTitleFallsBack(t *testing.T) {
	if !fontsAvailable() {
		t.Skip("ttf font files not embedded; skipping generator test")
	}
	if _, err := Generate(Options{ProjectName: "  ", Subtitle: ""}); err != nil {
		t.Fatalf("Generate returned error for empty input: %v", err)
	}
}

func fontsAvailable() bool {
	_, fraErr := fontFS.ReadFile("fonts/Fraunces-Regular.ttf")
	_, nunErr := fontFS.ReadFile("fonts/Nunito-Regular.ttf")
	if errors.Is(fraErr, fs.ErrNotExist) || errors.Is(nunErr, fs.ErrNotExist) {
		return false
	}
	if fraErr != nil || nunErr != nil {
		return false
	}
	return true
}
