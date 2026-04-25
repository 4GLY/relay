package services

import "context"

// NoopCacheInvalidator is the V2 baseline implementation of
// CacheInvalidator. It implements the interface so the revoke path can be
// wired end-to-end without a real CDN, and V2.5 swaps it for a CDN-backed
// implementation without touching the service surface.
type NoopCacheInvalidator struct{}

// Invalidate intentionally does nothing.
func (NoopCacheInvalidator) Invalidate(_ context.Context, _ ...string) error {
	return nil
}
