package services

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"relay/internal/lib"
)

// FilesystemOGImageWriter persists OG PNGs in a single directory rooted at
// BaseDir. Files are named "{snapshot_id}.png". Snapshot IDs are produced
// by lib.NewID / lib.StableID (hex-only with a small prefix), but we
// defensively reject any path-traversal characters anyway so this layer is
// safe to construct from user-derived inputs in the future.
type FilesystemOGImageWriter struct {
	BaseDir string
}

// NewFilesystemOGImageWriter is a small constructor that mkdir-p's the
// base directory. Callers should ensure the directory is writable by the
// process — runtime startup runs `os.MkdirAll(cfg.OGImageDir, 0o755)`.
func NewFilesystemOGImageWriter(baseDir string) (FilesystemOGImageWriter, error) {
	if strings.TrimSpace(baseDir) == "" {
		return FilesystemOGImageWriter{}, lib.Misconfigured("og image base dir is required")
	}
	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		return FilesystemOGImageWriter{}, fmt.Errorf("og image dir mkdir: %w", err)
	}
	return FilesystemOGImageWriter{BaseDir: baseDir}, nil
}

// WriteOGImage persists png bytes for snapshotID and returns the absolute
// (or BaseDir-relative) on-disk path.
func (w FilesystemOGImageWriter) WriteOGImage(_ context.Context, snapshotID string, png []byte) (string, error) {
	cleanID, err := safeSnapshotID(snapshotID)
	if err != nil {
		return "", err
	}
	if w.BaseDir == "" {
		return "", lib.Misconfigured("og image writer is not configured")
	}
	if err := os.MkdirAll(w.BaseDir, 0o755); err != nil {
		return "", fmt.Errorf("og image dir mkdir: %w", err)
	}
	dest := filepath.Join(w.BaseDir, cleanID+".png")
	if err := os.WriteFile(dest, png, 0o644); err != nil {
		return "", fmt.Errorf("og image write: %w", err)
	}
	return dest, nil
}

// ReadOGImage loads previously-written PNG bytes. Returns an os-level
// error (callers in the service layer interpret a missing file as a
// signal to regenerate inline).
func (w FilesystemOGImageWriter) ReadOGImage(_ context.Context, path string) ([]byte, error) {
	if strings.TrimSpace(path) == "" {
		return nil, lib.NotFound("OG_IMAGE_NOT_FOUND", "og image not found")
	}
	// Refuse to read anything that escapes the configured base dir, even
	// if the stored path is malformed.
	if w.BaseDir != "" {
		absBase, err := filepath.Abs(w.BaseDir)
		if err != nil {
			return nil, fmt.Errorf("og image base abs: %w", err)
		}
		absPath, err := filepath.Abs(path)
		if err != nil {
			return nil, fmt.Errorf("og image path abs: %w", err)
		}
		if !strings.HasPrefix(absPath+string(filepath.Separator), absBase+string(filepath.Separator)) && absPath != absBase {
			return nil, lib.NotFound("OG_IMAGE_NOT_FOUND", "og image not found")
		}
	}
	bytes, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return bytes, nil
}

func safeSnapshotID(id string) (string, error) {
	clean := strings.TrimSpace(id)
	if clean == "" {
		return "", lib.MissingFields("MISSING_REQUIRED_FIELDS", "snapshot_id")
	}
	if strings.ContainsAny(clean, "/\\") || strings.Contains(clean, "..") {
		return "", lib.AppError{
			Code:      "INVALID_SNAPSHOT_ID",
			Message:   "snapshot id contains path-traversal characters",
			Retryable: false,
		}
	}
	return clean, nil
}
