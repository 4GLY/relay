package services

import (
	"context"
	"strings"

	"relay/internal/domain"
	"relay/internal/lib"
	"relay/internal/lib/ogimage"
)

// CacheInvalidator is invoked on revoke so any CDN-cached public surface
// for a packet snapshot can be flushed before the row's public_readable
// flag flips. The V2 implementation is the no-op (see
// cache_invalidator_noop.go) and a real CDN client lands in V2.5.
type CacheInvalidator interface {
	Invalidate(ctx context.Context, paths ...string) error
}

// OGImageWriter is the interface for persisting and re-reading the
// pre-generated OG card. The default implementation in og_image_fs.go
// stores PNGs on the local filesystem rooted at RELAY_OG_IMAGE_DIR.
type OGImageWriter interface {
	WriteOGImage(ctx context.Context, snapshotID string, png []byte) (path string, err error)
	ReadOGImage(ctx context.Context, path string) ([]byte, error)
}

// SnapshotPublishResult is returned to the caller of PublishPacketSnapshot
// (admin route) so they can echo the public + og URLs back to a UI.
type SnapshotPublishResult struct {
	SnapshotID  string `json:"snapshot_id"`
	PublicToken string `json:"public_token"`
	PublicURL   string `json:"public_url"`
	OGImageURL  string `json:"og_image_url"`
}

// PublicSnapshotView pairs the public-readable snapshot with its parent
// project so the SSR template can render both project name and packet body
// without a second service round-trip.
type PublicSnapshotView struct {
	Project  domain.Project        `json:"project"`
	Snapshot domain.PacketSnapshot `json:"snapshot"`
}

// PublishPacketSnapshot is admin-only: it generates the public token, the
// OG card PNG, persists both, and returns the share URLs. Privacy default
// is FALSE — this is the only path that flips it to TRUE.
func (s Service) PublishPacketSnapshot(ctx context.Context, snapshotID string) (SnapshotPublishResult, error) {
	if err := requireAdminAuth(ctx); err != nil {
		return SnapshotPublishResult{}, err
	}
	if strings.TrimSpace(snapshotID) == "" {
		return SnapshotPublishResult{}, lib.MissingFields("MISSING_REQUIRED_FIELDS", "snapshot_id")
	}
	if s.deps.PacketSnapshots == nil {
		return SnapshotPublishResult{}, lib.Misconfigured("packet snapshot store is required")
	}
	if s.deps.OGImages == nil {
		return SnapshotPublishResult{}, lib.Misconfigured("og image writer is required")
	}

	snap, err := s.deps.PacketSnapshots.GetPacketSnapshot(ctx, snapshotID)
	if err != nil {
		return SnapshotPublishResult{}, err
	}
	project, err := s.deps.Projects.GetByID(ctx, snap.ProjectID)
	if err != nil {
		return SnapshotPublishResult{}, err
	}
	token, err := lib.NewSecretToken("psnap")
	if err != nil {
		return SnapshotPublishResult{}, err
	}
	png, err := ogimage.Generate(ogimage.Options{
		ProjectName: project.Name,
		Subtitle:    "Snapshot from Relay",
	})
	if err != nil {
		return SnapshotPublishResult{}, err
	}
	imagePath, err := s.deps.OGImages.WriteOGImage(ctx, snap.ID, png)
	if err != nil {
		return SnapshotPublishResult{}, err
	}
	updated, err := s.deps.PacketSnapshots.MakePacketSnapshotPublic(ctx, snap.ID, token, imagePath)
	if err != nil {
		return SnapshotPublishResult{}, err
	}
	return SnapshotPublishResult{
		SnapshotID:  updated.ID,
		PublicToken: updated.PublicToken,
		PublicURL:   s.publicURL("/p/" + updated.PublicToken),
		OGImageURL:  s.publicURL("/p/" + updated.PublicToken + "/og.png"),
	}, nil
}

// RevokePacketSnapshotPublic is admin-only: it flips public_readable to
// FALSE and asks the CacheInvalidator to flush the public + og URLs. Token
// and og_image_path are left in the row so the noop / real invalidator can
// see what to clear (the partial unique index ignores rows with
// public_readable=FALSE so re-publishing later is still safe).
func (s Service) RevokePacketSnapshotPublic(ctx context.Context, snapshotID string) error {
	if err := requireAdminAuth(ctx); err != nil {
		return err
	}
	if strings.TrimSpace(snapshotID) == "" {
		return lib.MissingFields("MISSING_REQUIRED_FIELDS", "snapshot_id")
	}
	if s.deps.PacketSnapshots == nil {
		return lib.Misconfigured("packet snapshot store is required")
	}
	updated, err := s.deps.PacketSnapshots.RevokePacketSnapshotPublic(ctx, snapshotID)
	if err != nil {
		return err
	}
	if s.deps.CacheInvalidator != nil && updated.PublicToken != "" {
		_ = s.deps.CacheInvalidator.Invalidate(
			ctx,
			"/p/"+updated.PublicToken,
			"/p/"+updated.PublicToken+"/og.png",
		)
	}
	return nil
}

// GetPublicSnapshot is unauthenticated: callers should be the public web
// handler. It returns lib.NotFound("PUBLIC_SNAPSHOT_NOT_FOUND", ...) when
// the token is unknown OR when the row exists but is not public-readable,
// which the handler maps to a 410 Gone HTML response.
func (s Service) GetPublicSnapshot(ctx context.Context, token string) (PublicSnapshotView, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return PublicSnapshotView{}, lib.NotFound("PUBLIC_SNAPSHOT_NOT_FOUND", "public snapshot not found")
	}
	if s.deps.PacketSnapshots == nil {
		return PublicSnapshotView{}, lib.Misconfigured("packet snapshot store is required")
	}
	snap, err := s.deps.PacketSnapshots.GetPacketSnapshotByPublicToken(ctx, token)
	if err != nil {
		return PublicSnapshotView{}, err
	}
	project, err := s.deps.Projects.GetByID(ctx, snap.ProjectID)
	if err != nil {
		return PublicSnapshotView{}, err
	}
	return PublicSnapshotView{Project: project, Snapshot: snap}, nil
}

// PublicSnapshotOGImage looks up the snapshot by token and returns the
// stored PNG. If the on-disk image is missing (ephemeral container restart),
// it regenerates from the project name and persists it back.
func (s Service) PublicSnapshotOGImage(ctx context.Context, token string) ([]byte, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return nil, lib.NotFound("PUBLIC_SNAPSHOT_NOT_FOUND", "public snapshot not found")
	}
	if s.deps.PacketSnapshots == nil {
		return nil, lib.Misconfigured("packet snapshot store is required")
	}
	if s.deps.OGImages == nil {
		return nil, lib.Misconfigured("og image writer is required")
	}
	snap, err := s.deps.PacketSnapshots.GetPacketSnapshotByPublicToken(ctx, token)
	if err != nil {
		return nil, err
	}
	if snap.OGImagePath != "" {
		if png, err := s.deps.OGImages.ReadOGImage(ctx, snap.OGImagePath); err == nil && len(png) > 0 {
			return png, nil
		}
	}
	project, err := s.deps.Projects.GetByID(ctx, snap.ProjectID)
	if err != nil {
		return nil, err
	}
	png, err := ogimage.Generate(ogimage.Options{
		ProjectName: project.Name,
		Subtitle:    "Snapshot from Relay",
	})
	if err != nil {
		return nil, err
	}
	if _, err := s.deps.OGImages.WriteOGImage(ctx, snap.ID, png); err != nil {
		// Soft failure — we still serve the freshly generated bytes.
		_ = err
	}
	return png, nil
}

// PublicURL composes an absolute URL using the configured PublicBaseURL.
// If no base URL is configured, paths are returned as-is so MCP clients +
// tests still see something sensible.
func (s Service) PublicURL(path string) string {
	return s.publicURL(path)
}

func (s Service) publicURL(path string) string {
	base := strings.TrimRight(s.deps.PublicBaseURL, "/")
	if base == "" {
		return path
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return base + path
}
