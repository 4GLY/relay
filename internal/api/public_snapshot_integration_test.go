package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"relay/internal/app"
	"relay/internal/config"
)

func TestPublicSnapshotPublishViewOGAndRevokeIntegration(t *testing.T) {
	token := "psnap_integration_token"
	ogWriter := &fakeOGImageWriterHandler{}
	handler := testPublicSnapshotHandler(token, ogWriter)
	mux := buildMux(handler, config.Config{APIToken: "admin-token"}, app.Runtime{Services: handler.services})

	publishReq := httptest.NewRequest(http.MethodPost, "/v1/snapshots/psnap_pub1/publish", nil)
	publishReq.Header.Set("Authorization", "Bearer admin-token")
	publishRec := httptest.NewRecorder()
	mux.ServeHTTP(publishRec, publishReq)

	if publishRec.Code != http.StatusOK {
		t.Fatalf("publish expected 200, got %d body=%s", publishRec.Code, publishRec.Body.String())
	}

	var publishBody struct {
		OK   bool `json:"ok"`
		Data struct {
			PublicURL   string `json:"public_url"`
			OGImageURL string `json:"og_image_url"`
			PublicToken string `json:"public_token"`
		} `json:"data"`
	}
	if err := json.Unmarshal(publishRec.Body.Bytes(), &publishBody); err != nil {
		t.Fatalf("decode publish response: %v", err)
	}
	if !publishBody.OK {
		t.Fatalf("expected ok publish response: %s", publishRec.Body.String())
	}
	if publishBody.Data.PublicToken == "" {
		t.Fatalf("expected public token in response: %s", publishRec.Body.String())
	}
	if !strings.HasPrefix(publishBody.Data.PublicURL, "https://relay.example.com/p/") {
		t.Fatalf("unexpected public_url %q", publishBody.Data.PublicURL)
	}
	if !strings.HasSuffix(publishBody.Data.OGImageURL, "/og.png") {
		t.Fatalf("unexpected og_image_url %q", publishBody.Data.OGImageURL)
	}

	pageReq := httptest.NewRequest(http.MethodGet, "/p/"+publishBody.Data.PublicToken, nil)
	pageRec := httptest.NewRecorder()
	mux.ServeHTTP(pageRec, pageReq)
	if pageRec.Code != http.StatusOK {
		t.Fatalf("page expected 200, got %d body=%s", pageRec.Code, pageRec.Body.String())
	}
	if !bytes.Contains(pageRec.Body.Bytes(), []byte(`<meta property="og:image"`)) {
		t.Fatalf("expected og:image meta in page: %s", pageRec.Body.String())
	}

	ogReq := httptest.NewRequest(http.MethodGet, "/p/"+publishBody.Data.PublicToken+"/og.png", nil)
	ogRec := httptest.NewRecorder()
	mux.ServeHTTP(ogRec, ogReq)
	if ogRec.Code != http.StatusOK {
		t.Fatalf("og expected 200, got %d body=%s", ogRec.Code, ogRec.Body.String())
	}
	if ogRec.Header().Get("Content-Type") != "image/png" {
		t.Fatalf("expected image/png, got %q", ogRec.Header().Get("Content-Type"))
	}
	if len(ogRec.Body.Bytes()) == 0 {
		t.Fatal("expected non-empty og png")
	}

	revokeReq := httptest.NewRequest(http.MethodPost, "/v1/snapshots/psnap_pub1/revoke", nil)
	revokeReq.Header.Set("Authorization", "Bearer admin-token")
	revokeRec := httptest.NewRecorder()
	mux.ServeHTTP(revokeRec, revokeReq)
	if revokeRec.Code != http.StatusOK {
		t.Fatalf("revoke expected 200, got %d body=%s", revokeRec.Code, revokeRec.Body.String())
	}

	revokedReq := httptest.NewRequest(http.MethodGet, "/p/"+publishBody.Data.PublicToken, nil)
	revokedRec := httptest.NewRecorder()
	mux.ServeHTTP(revokedRec, revokedReq)
	if revokedRec.Code != http.StatusGone {
		t.Fatalf("revoked page expected 410, got %d body=%s", revokedRec.Code, revokedRec.Body.String())
	}
}
