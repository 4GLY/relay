package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"relay/internal/domain"
	"relay/internal/lib"
)

func (s Stores) UpsertUserProviderCredential(ctx context.Context, credential domain.UserProviderCredential) (domain.UserProviderCredential, error) {
	var saved domain.UserProviderCredential
	err := s.db.QueryRow(ctx, `
		INSERT INTO user_provider_credentials (
			user_id,
			provider,
			key_ciphertext,
			key_nonce,
			key_kek_version,
			key_prefix,
			key_last4,
			aad_salt,
			deleted_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NULL)
		ON CONFLICT (user_id, provider) DO UPDATE
		SET key_ciphertext = EXCLUDED.key_ciphertext,
		    key_nonce = EXCLUDED.key_nonce,
		    key_kek_version = EXCLUDED.key_kek_version,
		    key_prefix = EXCLUDED.key_prefix,
		    key_last4 = EXCLUDED.key_last4,
		    aad_salt = EXCLUDED.aad_salt,
		    deleted_at = NULL,
		    updated_at = NOW()
		RETURNING user_id, provider, key_ciphertext, key_nonce, key_kek_version,
		          key_prefix, key_last4, aad_salt, created_at, updated_at, deleted_at
	`,
		credential.UserID,
		credential.Provider,
		credential.KeyCiphertext,
		credential.KeyNonce,
		int16(credential.KeyKEKVersion),
		credential.KeyPrefix,
		credential.KeyLast4,
		credential.AadSalt,
	).Scan(
		&saved.UserID,
		&saved.Provider,
		&saved.KeyCiphertext,
		&saved.KeyNonce,
		&saved.KeyKEKVersion,
		&saved.KeyPrefix,
		&saved.KeyLast4,
		&saved.AadSalt,
		&saved.CreatedAt,
		&saved.UpdatedAt,
		&saved.DeletedAt,
	)
	return saved, err
}

func (s Stores) GetUserProviderCredential(ctx context.Context, userID string, provider string) (domain.UserProviderCredential, error) {
	var saved domain.UserProviderCredential
	err := s.db.QueryRow(ctx, `
		SELECT user_id, provider, key_ciphertext, key_nonce, key_kek_version,
		       key_prefix, key_last4, aad_salt, created_at, updated_at, deleted_at
		FROM user_provider_credentials
		WHERE user_id = $1
		  AND provider = $2
		  AND deleted_at IS NULL
	`, userID, provider).Scan(
		&saved.UserID,
		&saved.Provider,
		&saved.KeyCiphertext,
		&saved.KeyNonce,
		&saved.KeyKEKVersion,
		&saved.KeyPrefix,
		&saved.KeyLast4,
		&saved.AadSalt,
		&saved.CreatedAt,
		&saved.UpdatedAt,
		&saved.DeletedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.UserProviderCredential{}, lib.NotFound("PROVIDER_CREDENTIAL_NOT_FOUND", "provider credential not found")
	}
	return saved, err
}

func (s Stores) ListUserProviderCredentials(ctx context.Context, userID string) ([]domain.UserProviderCredential, error) {
	rows, err := s.db.Query(ctx, `
		SELECT user_id, provider, key_ciphertext, key_nonce, key_kek_version,
		       key_prefix, key_last4, aad_salt, created_at, updated_at, deleted_at
		FROM user_provider_credentials
		WHERE user_id = $1
		  AND deleted_at IS NULL
		ORDER BY provider ASC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []domain.UserProviderCredential
	for rows.Next() {
		var item domain.UserProviderCredential
		if err := rows.Scan(
			&item.UserID,
			&item.Provider,
			&item.KeyCiphertext,
			&item.KeyNonce,
			&item.KeyKEKVersion,
			&item.KeyPrefix,
			&item.KeyLast4,
			&item.AadSalt,
			&item.CreatedAt,
			&item.UpdatedAt,
			&item.DeletedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s Stores) DeleteUserProviderCredential(ctx context.Context, userID string, provider string) error {
	var returned string
	err := s.db.QueryRow(ctx, `
		UPDATE user_provider_credentials
		SET deleted_at = NOW(),
		    updated_at = NOW()
		WHERE user_id = $1
		  AND provider = $2
		  AND deleted_at IS NULL
		RETURNING provider
	`, userID, provider).Scan(&returned)
	if errors.Is(err, pgx.ErrNoRows) {
		return lib.NotFound("PROVIDER_CREDENTIAL_NOT_FOUND", "provider credential not found")
	}
	return err
}
