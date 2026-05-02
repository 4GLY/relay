package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestResolvePublicSnapshotLocalePrefersCookie(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/p/token", nil)
	req.AddCookie(&http.Cookie{Name: relayLocaleCookie, Value: "ko"})
	req.Header.Set("Accept-Language", "en-US,en;q=0.8")

	if got := resolvePublicSnapshotLocale(req); got != publicSnapshotLocaleKO {
		t.Fatalf("expected ko from cookie, got %q", got)
	}

	req = httptest.NewRequest(http.MethodGet, "/p/token", nil)
	req.AddCookie(&http.Cookie{Name: relayLocaleCookie, Value: "en"})
	req.Header.Set("Accept-Language", "ko-KR,ko;q=0.9")

	if got := resolvePublicSnapshotLocale(req); got != publicSnapshotLocaleEN {
		t.Fatalf("expected en from cookie, got %q", got)
	}
}

func TestResolvePublicSnapshotLocaleFallsBackToAcceptLanguage(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/p/token", nil)
	req.Header.Set("Accept-Language", "fr-FR,fr;q=0.9,ko-KR;q=0.8,en;q=0.7")

	if got := resolvePublicSnapshotLocale(req); got != publicSnapshotLocaleKO {
		t.Fatalf("expected ko from Accept-Language fallback, got %q", got)
	}

	req = httptest.NewRequest(http.MethodGet, "/p/token", nil)
	req.Header.Set("Accept-Language", "fr-FR,fr;q=0.9,ko;q=0,en-US;q=0.5")

	if got := resolvePublicSnapshotLocale(req); got != publicSnapshotLocaleEN {
		t.Fatalf("expected en after skipping zero-quality ko, got %q", got)
	}
}

func TestResolvePublicSnapshotLocaleDefaultsToEnglish(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/p/token", nil)
	req.Header.Set("Accept-Language", "fr-FR,fr;q=0.9")

	if got := resolvePublicSnapshotLocale(req); got != publicSnapshotLocaleEN {
		t.Fatalf("expected english default, got %q", got)
	}
}
