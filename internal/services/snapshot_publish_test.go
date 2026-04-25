package services

import (
	"context"
	"errors"
	"strings"
	"testing"

	"relay/internal/domain"
	"relay/internal/lib"
)

type fakeOGImageWriter struct {
	written map[string][]byte
	calls   int
}

func (w *fakeOGImageWriter) WriteOGImage(_ context.Context, snapshotID string, png []byte) (string, error) {
	if w.written == nil {
		w.written = map[string][]byte{}
	}
	path := "/og/" + snapshotID + ".png"
	w.written[path] = png
	w.calls++
	return path, nil
}

func (w *fakeOGImageWriter) ReadOGImage(_ context.Context, path string) ([]byte, error) {
	if png, ok := w.written[path]; ok {
		return png, nil
	}
	return nil, errors.New("og image not on disk")
}

type fakeCacheInvalidator struct {
	paths []string
	calls int
}

func (c *fakeCacheInvalidator) Invalidate(_ context.Context, paths ...string) error {
	c.calls++
	c.paths = append(c.paths, paths...)
	return nil
}

func newPublishService(t *testing.T) (Service, *fakePacketSnapshotStore, *fakeOGImageWriter, *fakeCacheInvalidator) {
	t.Helper()
	projectID := lib.ProjectID("relay")
	snapshots := &fakePacketSnapshotStore{items: map[string]domain.PacketSnapshot{
		"psnap_1": {
			ID:           "psnap_1",
			ProjectID:    projectID,
			PacketKind:   "resume",
			Target:       "codex",
			RenderedBody: "Project: relay\nCurrent goal: ship V2",
			TaskSummary:  "ship V2",
		},
	}}
	ogWriter := &fakeOGImageWriter{}
	invalidator := &fakeCacheInvalidator{}
	service := New(Dependencies{
		Projects: &fakeProjectStore{projects: map[string]domain.Project{
			"relay": {ID: projectID, Name: "relay"},
		}},
		PacketSnapshots:  snapshots,
		OGImages:         ogWriter,
		CacheInvalidator: invalidator,
		PublicBaseURL:    "https://relay.example.com",
	})
	return service, snapshots, ogWriter, invalidator
}

func TestPublishPacketSnapshotAdminCreatesPublicURLs(t *testing.T) {
	service, snapshots, ogWriter, _ := newPublishService(t)
	adminCtx := ContextWithAuthInfo(context.Background(), AuthInfo{IsAdmin: true, Scope: APIKeyScopeGlobal})

	result, err := service.PublishPacketSnapshot(adminCtx, "psnap_1")
	if err != nil {
		t.Fatalf("PublishPacketSnapshot returned error: %v", err)
	}
	if result.PublicToken == "" {
		t.Fatal("expected public token")
	}
	if !strings.HasPrefix(result.PublicURL, "https://relay.example.com/p/") {
		t.Fatalf("expected absolute public URL, got %q", result.PublicURL)
	}
	if !strings.HasSuffix(result.OGImageURL, "/og.png") {
		t.Fatalf("expected og.png suffix, got %q", result.OGImageURL)
	}
	if ogWriter.calls != 1 {
		t.Fatalf("expected one OG write, got %d", ogWriter.calls)
	}
	stored := snapshots.items["psnap_1"]
	if !stored.PublicReadable {
		t.Fatal("expected snapshot to be marked public")
	}
	if stored.PublicToken != result.PublicToken {
		t.Fatalf("expected stored token %q, got %q", result.PublicToken, stored.PublicToken)
	}
	if stored.OGImagePath == "" {
		t.Fatal("expected stored og image path")
	}
}

func TestPublishPacketSnapshotRequiresAdmin(t *testing.T) {
	service, _, _, _ := newPublishService(t)
	_, err := service.PublishPacketSnapshot(context.Background(), "psnap_1")
	if err == nil {
		t.Fatal("expected non-admin publish to be rejected")
	}
	appErr, ok := err.(lib.AppError)
	if !ok || appErr.Code != "FORBIDDEN" {
		t.Fatalf("expected FORBIDDEN, got %#v", err)
	}
}

func TestRevokePacketSnapshotPublicCallsCacheInvalidator(t *testing.T) {
	service, snapshots, _, invalidator := newPublishService(t)
	adminCtx := ContextWithAuthInfo(context.Background(), AuthInfo{IsAdmin: true, Scope: APIKeyScopeGlobal})

	if _, err := service.PublishPacketSnapshot(adminCtx, "psnap_1"); err != nil {
		t.Fatalf("publish setup returned error: %v", err)
	}
	publishedToken := snapshots.items["psnap_1"].PublicToken

	if err := service.RevokePacketSnapshotPublic(adminCtx, "psnap_1"); err != nil {
		t.Fatalf("RevokePacketSnapshotPublic returned error: %v", err)
	}
	if snapshots.items["psnap_1"].PublicReadable {
		t.Fatal("expected snapshot to be private after revoke")
	}
	if invalidator.calls != 1 {
		t.Fatalf("expected one invalidator call, got %d", invalidator.calls)
	}
	if len(invalidator.paths) != 2 {
		t.Fatalf("expected two invalidated paths, got %d", len(invalidator.paths))
	}
	want := "/p/" + publishedToken
	if invalidator.paths[0] != want {
		t.Fatalf("expected first path %q, got %q", want, invalidator.paths[0])
	}
	if invalidator.paths[1] != want+"/og.png" {
		t.Fatalf("expected og path %q, got %q", want+"/og.png", invalidator.paths[1])
	}
}

func TestGetPublicSnapshotReturnsNotFoundForUnknownToken(t *testing.T) {
	service, _, _, _ := newPublishService(t)
	_, err := service.GetPublicSnapshot(context.Background(), "no-such-token")
	if err == nil {
		t.Fatal("expected NOT_FOUND for unknown token")
	}
	appErr, ok := err.(lib.AppError)
	if !ok || appErr.Code != "PUBLIC_SNAPSHOT_NOT_FOUND" {
		t.Fatalf("expected PUBLIC_SNAPSHOT_NOT_FOUND, got %#v", err)
	}
}

func TestGetPublicSnapshotReturnsNotFoundWhenRevoked(t *testing.T) {
	service, snapshots, _, _ := newPublishService(t)
	adminCtx := ContextWithAuthInfo(context.Background(), AuthInfo{IsAdmin: true, Scope: APIKeyScopeGlobal})

	result, err := service.PublishPacketSnapshot(adminCtx, "psnap_1")
	if err != nil {
		t.Fatalf("publish setup returned error: %v", err)
	}
	// Verify GetPublicSnapshot finds it while it is public.
	view, err := service.GetPublicSnapshot(context.Background(), result.PublicToken)
	if err != nil {
		t.Fatalf("GetPublicSnapshot before revoke returned error: %v", err)
	}
	if view.Project.Name != "relay" {
		t.Fatalf("expected project name relay, got %q", view.Project.Name)
	}

	// Flip public_readable=false directly to simulate revoke without
	// touching cache invalidator (already covered).
	snapshots.items["psnap_1"] = domain.PacketSnapshot{
		ID:             "psnap_1",
		ProjectID:      view.Snapshot.ProjectID,
		PacketKind:     view.Snapshot.PacketKind,
		Target:         view.Snapshot.Target,
		RenderedBody:   view.Snapshot.RenderedBody,
		TaskSummary:    view.Snapshot.TaskSummary,
		PublicReadable: false,
		PublicToken:    result.PublicToken,
	}

	if _, err := service.GetPublicSnapshot(context.Background(), result.PublicToken); err == nil {
		t.Fatal("expected NOT_FOUND after revoke")
	} else {
		appErr, ok := err.(lib.AppError)
		if !ok || appErr.Code != "PUBLIC_SNAPSHOT_NOT_FOUND" {
			t.Fatalf("expected PUBLIC_SNAPSHOT_NOT_FOUND, got %#v", err)
		}
	}
}

func TestPublicSnapshotOGImageRegeneratesWhenFileMissing(t *testing.T) {
	service, snapshots, ogWriter, _ := newPublishService(t)
	adminCtx := ContextWithAuthInfo(context.Background(), AuthInfo{IsAdmin: true, Scope: APIKeyScopeGlobal})

	result, err := service.PublishPacketSnapshot(adminCtx, "psnap_1")
	if err != nil {
		t.Fatalf("publish setup returned error: %v", err)
	}

	// Drop the on-disk file to simulate ephemeral container restart.
	for path := range ogWriter.written {
		delete(ogWriter.written, path)
	}
	prevWrites := ogWriter.calls

	png, err := service.PublicSnapshotOGImage(context.Background(), result.PublicToken)
	if err != nil {
		t.Fatalf("PublicSnapshotOGImage returned error: %v", err)
	}
	if len(png) == 0 {
		t.Fatal("expected non-empty regenerated PNG")
	}
	if ogWriter.calls <= prevWrites {
		t.Fatal("expected regeneration to write the OG image back to disk")
	}
	// Snapshot row still exists in the fake.
	if _, ok := snapshots.items["psnap_1"]; !ok {
		t.Fatal("expected snapshot row to remain")
	}
}
