package services

import (
	"context"
	"crypto/rand"
	"strings"
	"time"

	"relay/internal/domain"
	"relay/internal/lib"
	"relay/internal/lib/crypto"
)

// CompleteOnboardingInput is the (locked, post-D3) request shape for
// POST /v1/onboarding. RelayURL and SessionCookie were intentionally removed
// in D3 — adding them back will break the strict-decoder contract (T6).
type CompleteOnboardingInput struct {
	AnthropicKey string `json:"anthropic_key"`
}

// CompleteOnboardingResult is the response data envelope. Steps is always
// populated, even on the error paths, so the web client can render Frame 3
// chips without branching on success/error (D2).
type CompleteOnboardingResult struct {
	OnboardingComplete bool             `json:"onboarding_complete"`
	DefaultProjectID   string           `json:"default_project_id"`
	AnthropicKeyPrefix string           `json:"anthropic_key_prefix"`
	AnthropicKeyLast4  string           `json:"anthropic_key_last4"`
	Steps              []ValidationStep `json:"steps"`
}

// OnboardingStatus is the supplemental block surfaced via /v1/auth/me.
type OnboardingStatus struct {
	Complete         bool
	DefaultProjectID string
	LastValidatedAt  *time.Time
}

const completeOnboardingProbeBudget = 25 * time.Second

// CompleteOnboarding validates the user's Anthropic API key against
// api.anthropic.com, envelope-encrypts it under the active KEK, ensures a
// "Personal" project exists for the user (D4), and upserts the row. The probe
// has a 25s budget inside the 60s product promise; the remainder is HTTP
// overhead + client retries.
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
	if len(s.keks) == 0 || s.activeKEKVersion == 0 {
		return CompleteOnboardingResult{}, lib.Misconfigured("data encryption key is not configured")
	}

	rawKey := strings.TrimSpace(input.AnthropicKey)
	if rawKey == "" {
		return CompleteOnboardingResult{}, lib.MissingFields("MISSING_REQUIRED_FIELDS", "anthropic_key")
	}
	if !strings.HasPrefix(rawKey, "sk-ant-") {
		return CompleteOnboardingResult{Steps: validationSteps(StepFailed, StepSkipped)}, lib.AppError{
			Code:      "INVALID_ANTHROPIC_KEY",
			Message:   "key must start with sk-ant-",
			Retryable: false,
		}
	}

	probeCtx, cancel := context.WithTimeout(ctx, completeOnboardingProbeBudget)
	defer cancel()

	steps, err := validateAnthropicKey(probeCtx, rawKey)
	if err != nil {
		return CompleteOnboardingResult{Steps: steps}, err
	}

	// E2: salt generated in Go (no pgcrypto on Neon).
	// E8: a fresh salt every upsert binds the new ciphertext to a new AAD.
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return CompleteOnboardingResult{Steps: steps}, err
	}

	envelope, err := crypto.Encrypt(s.keks, s.activeKEKVersion, []byte(rawKey), salt)
	if err != nil {
		return CompleteOnboardingResult{Steps: steps}, err
	}

	project, err := s.deps.Onboarding.EnsureProjectByOwnerName(ctx, auth.UserID, "Personal", lib.NewID("proj"))
	if err != nil {
		return CompleteOnboardingResult{Steps: steps}, err
	}

	now := time.Now()
	row := domain.UserOnboarding{
		UserID:                 auth.UserID,
		AnthropicKeyCiphertext: envelope.Ciphertext,
		AnthropicKeyNonce:      envelope.Nonce,
		AnthropicKeyKEKVersion: uint8(envelope.KEKVersion),
		AnthropicKeyPrefix:     keyPrefix(rawKey),
		AnthropicKeyLast4:      keyLast4(rawKey),
		AadSalt:                salt,
		DefaultProjectID:       project.ID,
		OnboardingCompletedAt:  &now,
		LastValidatedAt:        &now,
	}
	saved, err := s.deps.Onboarding.UpsertOnboarding(ctx, row)
	if err != nil {
		return CompleteOnboardingResult{Steps: steps}, err
	}

	return CompleteOnboardingResult{
		OnboardingComplete: saved.OnboardingCompletedAt != nil,
		DefaultProjectID:   saved.DefaultProjectID,
		AnthropicKeyPrefix: saved.AnthropicKeyPrefix,
		AnthropicKeyLast4:  saved.AnthropicKeyLast4,
		Steps:              steps,
	}, nil
}

// DeleteOnboardingKey nulls the key material columns. The default project is
// preserved so the user can re-onboard without losing their packet history (D5).
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
// or a row whose onboarding_completed_at is NULL (post-D5 DELETE) both report
// Complete=false (E7). Errors other than ONBOARDING_NOT_FOUND are surfaced so
// the handler can decide whether to log and degrade.
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
