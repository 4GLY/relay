package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"relay/internal/domain"
	"relay/internal/lib"
)

// UpsertOnboarding writes the row's caller-supplied fields. Onboarding itself
// does not require Anthropic key material, so the key columns may be NULL. The
// default_project_id column is set via NULLIF so an empty string maps to SQL
// NULL — the FK has ON DELETE SET NULL semantics so a deleted project leaves
// the row intact.
func (s Stores) UpsertOnboarding(ctx context.Context, row domain.UserOnboarding) (domain.UserOnboarding, error) {
	var saved domain.UserOnboarding
	var defaultProjectID *string
	err := s.db.QueryRow(ctx, `
		INSERT INTO user_onboarding (
			user_id,
			anthropic_key_ciphertext,
			anthropic_key_nonce,
			anthropic_key_kek_version,
			anthropic_key_prefix,
			anthropic_key_last4,
			aad_salt,
			default_project_id,
			onboarding_completed_at,
			last_validated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NULLIF($8, ''), $9, $10)
		ON CONFLICT (user_id) DO UPDATE SET
			anthropic_key_ciphertext  = EXCLUDED.anthropic_key_ciphertext,
			anthropic_key_nonce       = EXCLUDED.anthropic_key_nonce,
			anthropic_key_kek_version = EXCLUDED.anthropic_key_kek_version,
			anthropic_key_prefix      = EXCLUDED.anthropic_key_prefix,
			anthropic_key_last4       = EXCLUDED.anthropic_key_last4,
			aad_salt                  = EXCLUDED.aad_salt,
			default_project_id        = COALESCE(EXCLUDED.default_project_id, user_onboarding.default_project_id),
			onboarding_completed_at   = EXCLUDED.onboarding_completed_at,
			last_validated_at         = EXCLUDED.last_validated_at,
			updated_at                = NOW()
		RETURNING
			user_id,
			anthropic_key_ciphertext,
			anthropic_key_nonce,
			anthropic_key_kek_version,
			anthropic_key_prefix,
			anthropic_key_last4,
			aad_salt,
			default_project_id,
			onboarding_completed_at,
			last_validated_at,
			created_at,
			updated_at
	`,
		row.UserID,
		row.AnthropicKeyCiphertext,
		row.AnthropicKeyNonce,
		int16(row.AnthropicKeyKEKVersion),
		row.AnthropicKeyPrefix,
		row.AnthropicKeyLast4,
		row.AadSalt,
		row.DefaultProjectID,
		row.OnboardingCompletedAt,
		row.LastValidatedAt,
	).Scan(
		&saved.UserID,
		&saved.AnthropicKeyCiphertext,
		&saved.AnthropicKeyNonce,
		&saved.AnthropicKeyKEKVersion,
		&saved.AnthropicKeyPrefix,
		&saved.AnthropicKeyLast4,
		&saved.AadSalt,
		&defaultProjectID,
		&saved.OnboardingCompletedAt,
		&saved.LastValidatedAt,
		&saved.CreatedAt,
		&saved.UpdatedAt,
	)
	if err != nil {
		return domain.UserOnboarding{}, err
	}
	if defaultProjectID != nil {
		saved.DefaultProjectID = *defaultProjectID
	}
	return saved, nil
}

func (s Stores) GetOnboardingByUserID(ctx context.Context, userID string) (domain.UserOnboarding, error) {
	var saved domain.UserOnboarding
	var defaultProjectID *string
	err := s.db.QueryRow(ctx, `
		SELECT
			user_id,
			anthropic_key_ciphertext,
			anthropic_key_nonce,
			anthropic_key_kek_version,
			anthropic_key_prefix,
			anthropic_key_last4,
			aad_salt,
			default_project_id,
			onboarding_completed_at,
			last_validated_at,
			created_at,
			updated_at
		FROM user_onboarding
		WHERE user_id = $1
	`, userID).Scan(
		&saved.UserID,
		&saved.AnthropicKeyCiphertext,
		&saved.AnthropicKeyNonce,
		&saved.AnthropicKeyKEKVersion,
		&saved.AnthropicKeyPrefix,
		&saved.AnthropicKeyLast4,
		&saved.AadSalt,
		&defaultProjectID,
		&saved.OnboardingCompletedAt,
		&saved.LastValidatedAt,
		&saved.CreatedAt,
		&saved.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.UserOnboarding{}, lib.NotFound("ONBOARDING_NOT_FOUND", "onboarding not found")
	}
	if err != nil {
		return domain.UserOnboarding{}, err
	}
	if defaultProjectID != nil {
		saved.DefaultProjectID = *defaultProjectID
	}
	return saved, nil
}

// DeleteOnboardingKey NULLs the key material columns so the row remains for
// onboarding/project-FK purposes but no longer represents an active provider
// key. This does not clear onboarding_completed_at.
// pgx.ErrNoRows on RETURNING means the user never onboarded — surface it as
// a not-found error so the handler can return 404.
func (s Stores) DeleteOnboardingKey(ctx context.Context, userID string) error {
	var returned string
	err := s.db.QueryRow(ctx, `
		UPDATE user_onboarding
		SET anthropic_key_ciphertext = NULL,
		    anthropic_key_nonce      = NULL,
		    anthropic_key_prefix     = '',
		    anthropic_key_last4      = '',
		    aad_salt                 = NULL,
		    last_validated_at        = NULL,
		    updated_at               = NOW()
		WHERE user_id = $1
		RETURNING user_id
	`, userID).Scan(&returned)
	if errors.Is(err, pgx.ErrNoRows) {
		return lib.NotFound("ONBOARDING_NOT_FOUND", "onboarding not found")
	}
	return err
}

// EnsureProjectByOwnerName resolves the user's project by (owner_user_id, name)
// — distinct from the V1 EnsureProject which keys on id. The unique index
// projects_owner_name_uniq (migration 0009) makes the upsert idempotent under
// concurrent onboarding calls.
func (s Stores) EnsureProjectByOwnerName(ctx context.Context, ownerUserID, name, newID string) (domain.Project, error) {
	var project domain.Project
	// projects has no updated_at column (migration 0001 + 0007 added owner_user_id only),
	// so the conflict update is a no-op SET status = projects.status — kept solely so
	// RETURNING fires for the conflicting row (DO NOTHING would skip the RETURNING).
	err := s.db.QueryRow(ctx, `
		INSERT INTO projects (id, owner_user_id, name, status, created_at)
		VALUES ($1, $2, $3, 'active', NOW())
		ON CONFLICT (owner_user_id, name) DO UPDATE
		SET status = projects.status
		RETURNING id, name, COALESCE(root_path, ''), status, COALESCE(owner_user_id, '')
	`, newID, ownerUserID, name).Scan(
		&project.ID,
		&project.Name,
		&project.RootPath,
		&project.Status,
		&project.OwnerUserID,
	)
	if err != nil {
		return domain.Project{}, err
	}
	return project, nil
}
