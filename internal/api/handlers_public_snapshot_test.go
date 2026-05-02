package api

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"relay/internal/app"
	"relay/internal/config"
	"relay/internal/domain"
	"relay/internal/lib"
	"relay/internal/services"
)

// fakeOGImageWriterHandler is a minimal OGImageWriter for handler-level tests.
// Pre-populate Images to have ReadOGImage return bytes without filesystem I/O.
type fakeOGImageWriterHandler struct {
	Images map[string][]byte
}

func (f *fakeOGImageWriterHandler) WriteOGImage(_ context.Context, snapshotID string, png []byte) (string, error) {
	if f.Images == nil {
		f.Images = map[string][]byte{}
	}
	path := snapshotID + ".png"
	f.Images[path] = png
	return path, nil
}

func (f *fakeOGImageWriterHandler) ReadOGImage(_ context.Context, path string) ([]byte, error) {
	if f.Images == nil {
		return nil, lib.NotFound("OG_IMAGE_NOT_FOUND", "og image not found")
	}
	data, ok := f.Images[path]
	if !ok || len(data) == 0 {
		return nil, lib.NotFound("OG_IMAGE_NOT_FOUND", "og image not found")
	}
	return data, nil
}

// testPublicSnapshotHandler returns a Handler with a pre-populated public
// snapshot. The snapshot has PublicReadable=true and the given token so that
// GetPacketSnapshotByPublicToken succeeds.
func testPublicSnapshotHandler(token string, ogWriter services.OGImageWriter) Handler {
	projectID := lib.ProjectID("relay")
	snapID := "psnap_pub1"

	snap := domain.PacketSnapshot{
		ID:             snapID,
		ProjectID:      projectID,
		PacketKind:     "resume",
		Target:         "codex",
		SchemaVersion:  "relay.packet.v1",
		RenderedBody:   "snapshot body text",
		TaskSummary:    "test task",
		StyleCues:      []byte(`[]`),
		CreatedAt:      time.Now(),
		PublicReadable: true,
		PublicToken:    token,
		OGImagePath:    snapID + ".png",
	}

	store := &fakePacketSnapshotStore{latest: snap}

	return Handler{
		services: services.New(services.Dependencies{
			Projects: &fakeProjectStore{
				projects: map[string]domain.Project{
					"relay": {ID: projectID, Name: "relay"},
				},
			},
			Notes:           &fakeNoteStore{},
			Artifacts:       &fakeArtifactStore{},
			Decisions:       &fakeDecisionStore{},
			OpenQuestions:   &fakeOpenQuestionStore{},
			Packets:         &fakePacketStore{},
			PacketSnapshots: store,
			OGImages:        ogWriter,
			PublicBaseURL:   "https://relay.example.com",
		}),
	}
}

func TestPublicSnapshotPageUnknownTokenReturns410(t *testing.T) {
	ogWriter := &fakeOGImageWriterHandler{}
	handler := testPublicSnapshotHandler("psnap_testtoken1234", ogWriter)

	req := httptest.NewRequest(http.MethodGet, "/p/unknown_token_xyz", nil)
	rec := httptest.NewRecorder()

	handler.handlePublicSnapshotPage(rec, req)

	if rec.Code != http.StatusGone {
		t.Fatalf("expected 410, got %d body=%s", rec.Code, rec.Body.String())
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte("no longer")) {
		t.Fatalf("expected 'no longer' in body, got %s", rec.Body.String())
	}
}

func TestPublicSnapshotPageValidTokenReturns200WithOGMeta(t *testing.T) {
	token := "psnap_testtoken1234"
	ogWriter := &fakeOGImageWriterHandler{
		Images: map[string][]byte{
			"psnap_pub1.png": {0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a},
		},
	}
	handler := testPublicSnapshotHandler(token, ogWriter)

	req := httptest.NewRequest(http.MethodGet, "/p/"+token, nil)
	rec := httptest.NewRecorder()

	handler.handlePublicSnapshotPage(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); ct != "text/html; charset=utf-8" {
		t.Fatalf("expected text/html content-type, got %s", ct)
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte("relay")) {
		t.Fatalf("expected project name in body, got %s", rec.Body.String())
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte(`<meta property="og:image"`)) {
		t.Fatalf("expected og:image meta tag in body, got %s", rec.Body.String())
	}
}

func TestPublicSnapshotPageKoreanRequestLocalizesChromeOnly(t *testing.T) {
	token := "psnap_testtoken1234"
	ogWriter := &fakeOGImageWriterHandler{
		Images: map[string][]byte{
			"psnap_pub1.png": {0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a},
		},
	}
	handler := testPublicSnapshotHandler(token, ogWriter)

	req := httptest.NewRequest(http.MethodGet, "/p/"+token, nil)
	req.Header.Set("Accept-Language", "ko-KR,ko;q=0.9,en;q=0.8")
	rec := httptest.NewRecorder()

	handler.handlePublicSnapshotPage(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	assertBodyContains(t, rec.Body.Bytes(), `<html lang="ko">`)
	assertBodyContains(t, rec.Body.Bytes(), `<title>relay — Relay 스냅샷</title>`)
	assertBodyContains(t, rec.Body.Bytes(), `content="판단과 맥락을 담은 공개 Relay 스냅샷입니다."`)
	assertBodyContains(t, rec.Body.Bytes(), `Relay 스냅샷 — 판단을 기록하고 맥락을 지킵니다.`)
	assertBodyContains(t, rec.Body.Bytes(), `relay.4gly.dev에서 직접 만들기 →`)
	assertBodyContains(t, rec.Body.Bytes(), `snapshot body text`)
}

func TestPublicSnapshotOGImageValidTokenReturnsPNG(t *testing.T) {
	token := "psnap_testtoken1234"
	fakePNG := []byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a, 0x01, 0x02}
	ogWriter := &fakeOGImageWriterHandler{
		Images: map[string][]byte{
			"psnap_pub1.png": fakePNG,
		},
	}
	handler := testPublicSnapshotHandler(token, ogWriter)

	req := httptest.NewRequest(http.MethodGet, "/p/"+token+"/og.png", nil)
	rec := httptest.NewRecorder()

	handler.handlePublicSnapshotPage(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); ct != "image/png" {
		t.Fatalf("expected image/png, got %s", ct)
	}
	if !bytes.Equal(rec.Body.Bytes(), fakePNG) {
		t.Fatalf("expected fake PNG bytes, got %v", rec.Body.Bytes())
	}
}

func assertBodyContains(t *testing.T, body []byte, want string) {
	t.Helper()
	if !bytes.Contains(body, []byte(want)) {
		t.Fatalf("expected body to contain %q, got %s", want, string(body))
	}
}

func TestSnapshotPublishRouteReturnsPublicURL(t *testing.T) {
	token := "psnap_testtoken1234"
	ogWriter := &fakeOGImageWriterHandler{}
	handler := testPublicSnapshotHandler(token, ogWriter)

	// The fakePacketSnapshotStore.latest has PublicReadable=true with token,
	// but MakePacketSnapshotPublic identifies by ID. We need a snapshot whose
	// ID is known and which is addressable by the store. The store's latest is
	// psnap_pub1 — publish it.
	snapID := "psnap_pub1"
	mux := buildMux(handler, config.Config{APIToken: "admin-token"}, app.Runtime{Services: handler.services})

	req := httptest.NewRequest(http.MethodPost, "/v1/snapshots/"+snapID+"/publish", nil)
	req.Header.Set("Authorization", "Bearer admin-token")
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte(`"public_url"`)) {
		t.Fatalf("expected public_url in body, got %s", rec.Body.String())
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte("relay.example.com")) {
		t.Fatalf("expected base URL in public_url, got %s", rec.Body.String())
	}
}

func TestSnapshotRevokeRouteRevokesSnapshot(t *testing.T) {
	token := "psnap_testtoken1234"
	ogWriter := &fakeOGImageWriterHandler{}
	handler := testPublicSnapshotHandler(token, ogWriter)

	snapID := "psnap_pub1"
	mux := buildMux(handler, config.Config{APIToken: "admin-token"}, app.Runtime{Services: handler.services})

	req := httptest.NewRequest(http.MethodPost, "/v1/snapshots/"+snapID+"/revoke", nil)
	req.Header.Set("Authorization", "Bearer admin-token")
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	// After revoke the public token lookup should fail → 410.
	req2 := httptest.NewRequest(http.MethodGet, "/p/"+token, nil)
	rec2 := httptest.NewRecorder()
	mux.ServeHTTP(rec2, req2)

	if rec2.Code != http.StatusGone {
		t.Fatalf("expected 410 after revoke, got %d body=%s", rec2.Code, rec2.Body.String())
	}
}
