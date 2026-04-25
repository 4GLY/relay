package postgres

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"relay/internal/domain"
	"relay/internal/lib"
)

func (s Stores) CreateUser(ctx context.Context, user domain.User) (domain.User, error) {
	_, err := s.db.Exec(ctx, `
		INSERT INTO users (id, email, display_name, avatar_url)
		VALUES ($1, NULLIF($2, ''), $3, $4)
		ON CONFLICT (id) DO NOTHING
	`, user.ID, user.Email, user.DisplayName, user.AvatarURL)
	if err != nil {
		return domain.User{}, err
	}
	return s.GetUserByID(ctx, user.ID)
}

func (s Stores) GetUserByID(ctx context.Context, id string) (domain.User, error) {
	var user domain.User
	err := s.db.QueryRow(ctx, `
		SELECT id, COALESCE(email, ''), display_name, avatar_url, created_at, updated_at
		FROM users
		WHERE id = $1
	`, id).Scan(&user.ID, &user.Email, &user.DisplayName, &user.AvatarURL, &user.CreatedAt, &user.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.User{}, lib.NotFound("USER_NOT_FOUND", "user not found")
	}
	return user, err
}

func (s Stores) GetUserByEmail(ctx context.Context, email string) (domain.User, error) {
	if strings.TrimSpace(email) == "" {
		return domain.User{}, lib.NotFound("USER_NOT_FOUND", "user not found")
	}
	var user domain.User
	err := s.db.QueryRow(ctx, `
		SELECT id, COALESCE(email, ''), display_name, avatar_url, created_at, updated_at
		FROM users
		WHERE LOWER(email) = LOWER($1)
	`, email).Scan(&user.ID, &user.Email, &user.DisplayName, &user.AvatarURL, &user.CreatedAt, &user.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.User{}, lib.NotFound("USER_NOT_FOUND", "user not found")
	}
	return user, err
}

func (s Stores) UpdateUser(ctx context.Context, user domain.User) (domain.User, error) {
	err := s.db.QueryRow(ctx, `
		UPDATE users
		SET email = NULLIF($2, ''),
		    display_name = $3,
		    avatar_url = $4,
		    updated_at = NOW()
		WHERE id = $1
		RETURNING id
	`, user.ID, user.Email, user.DisplayName, user.AvatarURL).Scan(&user.ID)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.User{}, lib.NotFound("USER_NOT_FOUND", "user not found")
	}
	if err != nil {
		return domain.User{}, err
	}
	return s.GetUserByID(ctx, user.ID)
}

func (s Stores) UpsertOAuthIdentity(ctx context.Context, identity domain.OAuthIdentity) (domain.OAuthIdentity, error) {
	_, err := s.db.Exec(ctx, `
		INSERT INTO oauth_identities (id, user_id, provider, provider_user_id, provider_login, verified_email)
		VALUES ($1, $2, $3, $4, $5, NULLIF($6, ''))
		ON CONFLICT (provider, provider_user_id) DO UPDATE
		SET user_id = EXCLUDED.user_id,
		    provider_login = EXCLUDED.provider_login,
		    verified_email = COALESCE(EXCLUDED.verified_email, oauth_identities.verified_email),
		    updated_at = NOW()
	`, identity.ID, identity.UserID, identity.Provider, identity.ProviderUserID, identity.ProviderLogin, identity.VerifiedEmail)
	if err != nil {
		return domain.OAuthIdentity{}, err
	}
	return s.GetOAuthIdentityByProvider(ctx, identity.Provider, identity.ProviderUserID)
}

func (s Stores) GetOAuthIdentityByProvider(ctx context.Context, provider string, providerUserID string) (domain.OAuthIdentity, error) {
	var identity domain.OAuthIdentity
	err := s.db.QueryRow(ctx, `
		SELECT id, user_id, provider, provider_user_id, provider_login, COALESCE(verified_email, ''), created_at, updated_at
		FROM oauth_identities
		WHERE provider = $1
		  AND provider_user_id = $2
	`, provider, providerUserID).Scan(&identity.ID, &identity.UserID, &identity.Provider, &identity.ProviderUserID, &identity.ProviderLogin, &identity.VerifiedEmail, &identity.CreatedAt, &identity.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.OAuthIdentity{}, lib.NotFound("OAUTH_IDENTITY_NOT_FOUND", "oauth identity not found")
	}
	return identity, err
}

func (s Stores) ListOAuthIdentitiesByUser(ctx context.Context, userID string) ([]domain.OAuthIdentity, error) {
	rows, err := s.db.Query(ctx, `
		SELECT id, user_id, provider, provider_user_id, provider_login, COALESCE(verified_email, ''), created_at, updated_at
		FROM oauth_identities
		WHERE user_id = $1
		ORDER BY created_at ASC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []domain.OAuthIdentity
	for rows.Next() {
		var item domain.OAuthIdentity
		if err := rows.Scan(&item.ID, &item.UserID, &item.Provider, &item.ProviderUserID, &item.ProviderLogin, &item.VerifiedEmail, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s Stores) CreateUserSession(ctx context.Context, session domain.UserSession) (domain.UserSession, error) {
	_, err := s.db.Exec(ctx, `
		INSERT INTO user_sessions (id, user_id, token_hash, expires_at)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (id) DO NOTHING
	`, session.ID, session.UserID, session.TokenHash, session.ExpiresAt)
	if err != nil {
		return domain.UserSession{}, err
	}
	return session, nil
}

func (s Stores) GetUserSessionByTokenHash(ctx context.Context, tokenHash string) (domain.UserSession, error) {
	var session domain.UserSession
	var revokedAt *time.Time
	err := s.db.QueryRow(ctx, `
		SELECT id, user_id, token_hash, expires_at, created_at, revoked_at
		FROM user_sessions
		WHERE token_hash = $1
		  AND revoked_at IS NULL
		  AND expires_at > NOW()
	`, tokenHash).Scan(&session.ID, &session.UserID, &session.TokenHash, &session.ExpiresAt, &session.CreatedAt, &revokedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.UserSession{}, lib.NotFound("USER_SESSION_NOT_FOUND", "user session not found")
	}
	if err != nil {
		return domain.UserSession{}, err
	}
	session.RevokedAt = revokedAt
	return session, nil
}

// RotateUserSession atomically rotates a session's token_hash + expires_at,
// guarded by the session id and the caller's current token_hash. Concurrent
// rotations cannot both succeed: the WHERE clause matches at most one row, and
// the loser sees rotated=false. Already-revoked or expired sessions also
// return rotated=false.
func (s Stores) RotateUserSession(ctx context.Context, sessionID string, currentTokenHash string, newTokenHash string, newExpiresAt time.Time) (bool, error) {
	cmd, err := s.db.Exec(ctx, `
		UPDATE user_sessions
		SET token_hash = $3,
		    expires_at = $4
		WHERE id = $1
		  AND token_hash = $2
		  AND revoked_at IS NULL
		  AND expires_at > NOW()
	`, sessionID, currentTokenHash, newTokenHash, newExpiresAt)
	if err != nil {
		return false, err
	}
	return cmd.RowsAffected() == 1, nil
}

func (s Stores) RevokeUserSession(ctx context.Context, sessionID string) error {
	_, err := s.db.Exec(ctx, `
		UPDATE user_sessions
		SET revoked_at = COALESCE(revoked_at, NOW())
		WHERE id = $1
	`, sessionID)
	return err
}

func (s Stores) CreateOAuthState(ctx context.Context, state domain.OAuthState) (domain.OAuthState, error) {
	_, err := s.db.Exec(ctx, `
		INSERT INTO oauth_states (id, provider, redirect_to, expires_at)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (id) DO NOTHING
	`, state.ID, state.Provider, state.RedirectTo, state.ExpiresAt)
	if err != nil {
		return domain.OAuthState{}, err
	}
	return state, nil
}

func (s Stores) ConsumeOAuthState(ctx context.Context, stateID string) (domain.OAuthState, error) {
	var state domain.OAuthState
	var consumedAt *time.Time
	err := s.db.QueryRow(ctx, `
		UPDATE oauth_states
		SET consumed_at = NOW()
		WHERE id = $1
		  AND consumed_at IS NULL
		  AND expires_at > NOW()
		RETURNING id, provider, redirect_to, expires_at, created_at, consumed_at
	`, stateID).Scan(&state.ID, &state.Provider, &state.RedirectTo, &state.ExpiresAt, &state.CreatedAt, &consumedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.OAuthState{}, lib.NotFound("OAUTH_STATE_INVALID", "oauth state is invalid or expired")
	}
	if err != nil {
		return domain.OAuthState{}, err
	}
	state.ConsumedAt = consumedAt
	return state, nil
}
