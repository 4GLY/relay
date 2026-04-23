package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"

	"relay/internal/domain"
	"relay/internal/lib"
)

type idempotencyLookup struct {
	found      bool
	responseID string
}

func normalizedRequestHash(value any) string {
	raw, _ := json.Marshal(value)
	hash := sha256.Sum256(raw)
	return hex.EncodeToString(hash[:])
}

func (s Service) lookupIdempotency(ctx context.Context, scopeKind string, projectID string, key string, requestHash string) (idempotencyLookup, error) {
	if key == "" || s.deps.Idempotency == nil {
		return idempotencyLookup{}, nil
	}
	record, err := s.deps.Idempotency.GetIdempotencyRecord(ctx, scopeKind, projectID, key)
	if err != nil {
		if appErr, ok := err.(lib.AppError); ok && appErr.Code == "IDEMPOTENCY_RECORD_NOT_FOUND" {
			return idempotencyLookup{}, nil
		}
		return idempotencyLookup{}, err
	}
	if record.RequestHash != requestHash {
		return idempotencyLookup{}, lib.AppError{
			Code:      "IDEMPOTENCY_CONFLICT",
			Message:   "idempotency key was already used with a different payload",
			Retryable: false,
		}
	}
	return idempotencyLookup{found: true, responseID: record.ResponseID}, nil
}

func (s Service) recordIdempotency(ctx context.Context, scopeKind string, projectID string, key string, requestHash string, responseKind string, responseID string) error {
	if key == "" || s.deps.Idempotency == nil {
		return nil
	}
	_, err := s.deps.Idempotency.CreateIdempotencyRecord(ctx, domain.IdempotencyRecord{
		ID:             lib.StableID("idem", scopeKind+":"+projectID+":"+key),
		ScopeKind:      scopeKind,
		ScopeProjectID: projectID,
		IdempotencyKey: key,
		RequestHash:    requestHash,
		ResponseKind:   responseKind,
		ResponseID:     responseID,
	})
	return err
}
