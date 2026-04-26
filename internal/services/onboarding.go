package services

import (
	"context"
	"strings"
	"time"

	"relay/internal/domain"
	"relay/internal/lib"
)

// CompleteOnboardingInput is the intentionally empty request shape for
// POST /v1/onboarding. Onboarding must not require a provider billing key;
// Anthropic key connection belongs to a later settings / Claude-powered feature
// flow, not the first-run unlock.
type CompleteOnboardingInput struct{}

// CompleteOnboardingResult is the response data envelope.
type CompleteOnboardingResult struct {
	OnboardingComplete bool   `json:"onboarding_complete"`
	DefaultProjectID   string `json:"default_project_id"`
}

// OnboardingStatus is the supplemental block surfaced via /v1/auth/me.
type OnboardingStatus struct {
	Complete         bool
	DefaultProjectID string
	LastValidatedAt  *time.Time
}

// CompleteOnboarding ensures a "Personal" project exists for the user and marks
// the first-run onboarding flow complete. Provider credentials are deliberately
// not part of this contract: users must be able to enter Relay before deciding
// whether to connect a Claude provider key.
func (s Service) CompleteOnboarding(ctx context.Context, input CompleteOnboardingInput) (CompleteOnboardingResult, error) {
	auth, err := RequireUserAuth(ctx)
	if err != nil {
		return CompleteOnboardingResult{}, err
	}
	if auth.UserID == "" {
		// Admin-bearer R1 case: this endpoint is user-only because the row is
		// keyed by user_id and admin context has no user identity to bind to.
		return CompleteOnboardingResult{}, lib.Forbidden("UNAUTHORIZED", "onboarding requires a user session")
	}
	if s.deps.Onboarding == nil {
		return CompleteOnboardingResult{}, lib.Misconfigured("onboarding store is required")
	}

	project, err := s.deps.Onboarding.EnsureProjectByOwnerName(ctx, auth.UserID, "Personal", lib.NewID("proj"))
	if err != nil {
		return CompleteOnboardingResult{}, err
	}

	now := time.Now()
	row := domain.UserOnboarding{
		UserID:                auth.UserID,
		DefaultProjectID:      project.ID,
		OnboardingCompletedAt: &now,
	}
	saved, err := s.deps.Onboarding.UpsertOnboarding(ctx, row)
	if err != nil {
		return CompleteOnboardingResult{}, err
	}

	return CompleteOnboardingResult{
		OnboardingComplete: saved.OnboardingCompletedAt != nil,
		DefaultProjectID:   saved.DefaultProjectID,
	}, nil
}

// DeleteOnboardingKey nulls provider key material columns without undoing
// onboarding. The default project and onboarding_completed_at are preserved so
// users do not get sent back through first-run setup when they disconnect a
// provider key.
func (s Service) DeleteOnboardingKey(ctx context.Context, userID string) error {
	if strings.TrimSpace(userID) == "" {
		return lib.MissingFields("MISSING_REQUIRED_FIELDS", "user_id")
	}
	if s.deps.Onboarding == nil {
		return lib.Misconfigured("onboarding store is required")
	}
	return s.deps.Onboarding.DeleteOnboardingKey(ctx, userID)
}

// GetOnboardingStatus is the supplemental block for /v1/auth/me. A missing row
// or a row whose onboarding_completed_at is NULL both report Complete=false.
// DELETE /v1/onboarding clears provider key material only, so it does not move
// an already-onboarded user back to Complete=false. Errors other than
// ONBOARDING_NOT_FOUND are surfaced so the handler can decide whether to log
// and degrade.
func (s Service) GetOnboardingStatus(ctx context.Context, userID string) (OnboardingStatus, error) {
	if strings.TrimSpace(userID) == "" {
		return OnboardingStatus{}, lib.MissingFields("MISSING_REQUIRED_FIELDS", "user_id")
	}
	if s.deps.Onboarding == nil {
		return OnboardingStatus{}, nil
	}
	row, err := s.deps.Onboarding.GetOnboardingByUserID(ctx, userID)
	if err != nil {
		if isNotFound(err, "ONBOARDING_NOT_FOUND") {
			return OnboardingStatus{}, nil
		}
		return OnboardingStatus{}, err
	}
	return OnboardingStatus{
		Complete:         row.OnboardingCompletedAt != nil,
		DefaultProjectID: row.DefaultProjectID,
		LastValidatedAt:  row.LastValidatedAt,
	}, nil
}

// keyPrefix returns the first 18 chars of the Anthropic key followed by an
// ellipsis, or the full key when shorter. Used for the masked display on the
// settings UI (S8). Never logs the full key (F7).
func keyPrefix(k string) string {
	if len(k) <= 18 {
		return k
	}
	return k[:18] + "..."
}

func keyLast4(k string) string {
	if len(k) <= 4 {
		return k
	}
	return k[len(k)-4:]
}
