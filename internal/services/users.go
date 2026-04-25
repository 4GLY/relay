package services

import (
	"context"
	"strings"
	"time"

	"relay/internal/domain"
	"relay/internal/lib"
	"relay/internal/lib/oauth"
)

const (
	userSessionTTL = 30 * 24 * time.Hour
	// userSessionRefreshWindow is the rolling-refresh threshold. When a session
	// is closer to expiry than this, ResolveSession rotates the cookie token
	// and extends expires_at by another userSessionTTL. The old token is
	// invalidated server-side via atomic UPDATE on token_hash match.
	//
	// V2 baseline policy: 30-day absolute TTL + 7-day rolling refresh on
	// /v1/auth/me only. Other authenticated endpoints validate without
	// rotating to avoid the concurrent-request race that would log a sibling
	// tab out. Extending rotation to middleware paths needs a token grace
	// window; tracked in TODOS.md as V2.5.
	userSessionRefreshWindow = 7 * 24 * time.Hour
	oauthStateTTL            = 10 * time.Minute
)

type SignInResult struct {
	User             domain.User
	SessionToken     string
	SessionExpiresAt time.Time
}

// SessionResolution is the result of ResolveSession. When Refreshed is true
// the caller MUST replace the session cookie with SessionToken and
// SessionExpiresAt; the old cookie value is no longer valid.
type SessionResolution struct {
	User             domain.User
	Refreshed        bool
	SessionToken     string
	SessionExpiresAt time.Time
}

// SignInWithOAuthProfile resolves an OAuth profile to a relay user and issues a
// fresh session token. It implements the auto-link policy:
//  1. Existing oauth_identity (provider, provider_user_id) wins.
//  2. Otherwise an existing user with the same verified email gets a new
//     identity row attached (cross-provider account linking).
//  3. Otherwise a new user is created.
//
// Email may be empty (the GitHub private-primary case): in that path step 2 is
// skipped and a fresh user is created without an email.
func (s Service) SignInWithOAuthProfile(ctx context.Context, profile oauth.Profile) (SignInResult, error) {
	if profile.Provider == "" || profile.ProviderUserID == "" {
		return SignInResult{}, lib.MissingFields("MISSING_REQUIRED_FIELDS", "provider", "provider_user_id")
	}
	if s.deps.Users == nil || s.deps.OAuthIdentities == nil || s.deps.UserSessions == nil {
		return SignInResult{}, lib.Misconfigured("user identity stores are required")
	}

	user, err := s.resolveOrCreateUser(ctx, profile)
	if err != nil {
		return SignInResult{}, err
	}

	rawToken, err := lib.NewSecretToken("rsess")
	if err != nil {
		return SignInResult{}, err
	}
	expiresAt := time.Now().Add(userSessionTTL)
	if _, err := s.deps.UserSessions.CreateUserSession(ctx, domain.UserSession{
		ID:        lib.NewID("usess"),
		UserID:    user.ID,
		TokenHash: lib.TokenHash(rawToken),
		ExpiresAt: expiresAt,
	}); err != nil {
		return SignInResult{}, err
	}

	return SignInResult{User: user, SessionToken: rawToken, SessionExpiresAt: expiresAt}, nil
}

func (s Service) resolveOrCreateUser(ctx context.Context, profile oauth.Profile) (domain.User, error) {
	identity, err := s.deps.OAuthIdentities.GetOAuthIdentityByProvider(ctx, profile.Provider, profile.ProviderUserID)
	if err == nil {
		user, err := s.deps.Users.GetUserByID(ctx, identity.UserID)
		if err != nil {
			return domain.User{}, err
		}
		// refresh login + verified_email
		if _, err := s.deps.OAuthIdentities.UpsertOAuthIdentity(ctx, domain.OAuthIdentity{
			ID:             identity.ID,
			UserID:         user.ID,
			Provider:       profile.Provider,
			ProviderUserID: profile.ProviderUserID,
			ProviderLogin:  profile.Login,
			VerifiedEmail:  profile.Email,
		}); err != nil {
			return domain.User{}, err
		}
		return user, nil
	}
	if !isNotFound(err, "OAUTH_IDENTITY_NOT_FOUND") {
		return domain.User{}, err
	}

	if profile.Email != "" {
		existing, err := s.deps.Users.GetUserByEmail(ctx, profile.Email)
		if err == nil {
			if _, err := s.deps.OAuthIdentities.UpsertOAuthIdentity(ctx, domain.OAuthIdentity{
				ID:             lib.NewID("oid"),
				UserID:         existing.ID,
				Provider:       profile.Provider,
				ProviderUserID: profile.ProviderUserID,
				ProviderLogin:  profile.Login,
				VerifiedEmail:  profile.Email,
			}); err != nil {
				return domain.User{}, err
			}
			return existing, nil
		}
		if !isNotFound(err, "USER_NOT_FOUND") {
			return domain.User{}, err
		}
	}

	displayName := strings.TrimSpace(profile.DisplayName)
	if displayName == "" {
		displayName = profile.Login
	}
	user, err := s.deps.Users.CreateUser(ctx, domain.User{
		ID:          lib.NewID("usr"),
		Email:       profile.Email,
		DisplayName: displayName,
		AvatarURL:   profile.AvatarURL,
	})
	if err != nil {
		return domain.User{}, err
	}
	if _, err := s.deps.OAuthIdentities.UpsertOAuthIdentity(ctx, domain.OAuthIdentity{
		ID:             lib.NewID("oid"),
		UserID:         user.ID,
		Provider:       profile.Provider,
		ProviderUserID: profile.ProviderUserID,
		ProviderLogin:  profile.Login,
		VerifiedEmail:  profile.Email,
	}); err != nil {
		return domain.User{}, err
	}
	return user, nil
}

// GetUserBySessionToken hashes the cookie token and resolves the user. The
// underlying store is expected to filter expired/revoked sessions.
func (s Service) GetUserBySessionToken(ctx context.Context, rawToken string) (domain.User, error) {
	if strings.TrimSpace(rawToken) == "" {
		return domain.User{}, lib.Forbidden("UNAUTHORIZED", "session token is required")
	}
	if s.deps.UserSessions == nil || s.deps.Users == nil {
		return domain.User{}, lib.Misconfigured("user session store is required")
	}
	session, err := s.deps.UserSessions.GetUserSessionByTokenHash(ctx, lib.TokenHash(rawToken))
	if err != nil {
		return domain.User{}, err
	}
	return s.deps.Users.GetUserByID(ctx, session.UserID)
}

// ResolveSession validates the cookie token, returns the user, and rotates
// the token if the session is within userSessionRefreshWindow of expiry.
// Rotation is race-safe: the underlying store performs an atomic UPDATE on
// the (id, current_token_hash) pair, so concurrent rotations cannot both
// succeed. Use this for /v1/auth/me; other authenticated paths call
// GetUserBySessionToken which validates without rotating.
func (s Service) ResolveSession(ctx context.Context, rawToken string) (SessionResolution, error) {
	if strings.TrimSpace(rawToken) == "" {
		return SessionResolution{}, lib.Forbidden("UNAUTHORIZED", "session token is required")
	}
	if s.deps.UserSessions == nil || s.deps.Users == nil {
		return SessionResolution{}, lib.Misconfigured("user session store is required")
	}

	currentHash := lib.TokenHash(rawToken)
	session, err := s.deps.UserSessions.GetUserSessionByTokenHash(ctx, currentHash)
	if err != nil {
		return SessionResolution{}, err
	}
	user, err := s.deps.Users.GetUserByID(ctx, session.UserID)
	if err != nil {
		return SessionResolution{}, err
	}

	if time.Until(session.ExpiresAt) > userSessionRefreshWindow {
		return SessionResolution{User: user}, nil
	}

	newToken, err := lib.NewSecretToken("rsess")
	if err != nil {
		return SessionResolution{}, err
	}
	newExpiresAt := time.Now().Add(userSessionTTL)
	rotated, err := s.deps.UserSessions.RotateUserSession(ctx, session.ID, currentHash, lib.TokenHash(newToken), newExpiresAt)
	if err != nil {
		return SessionResolution{}, err
	}
	if !rotated {
		// Lost the rotation race. Caller's cookie is now stale; the next
		// request will 401 and the user will sign in again. Acceptable V2
		// edge case for concurrent /me polls within the rotation window.
		return SessionResolution{User: user}, nil
	}
	return SessionResolution{
		User:             user,
		Refreshed:        true,
		SessionToken:     newToken,
		SessionExpiresAt: newExpiresAt,
	}, nil
}

// GetSessionByToken returns the session metadata (id, user_id) given a raw
// cookie token. Useful when the caller needs the session id (e.g. logout).
func (s Service) GetSessionByToken(ctx context.Context, rawToken string) (domain.UserSession, error) {
	if strings.TrimSpace(rawToken) == "" {
		return domain.UserSession{}, lib.Forbidden("UNAUTHORIZED", "session token is required")
	}
	if s.deps.UserSessions == nil {
		return domain.UserSession{}, lib.Misconfigured("user session store is required")
	}
	return s.deps.UserSessions.GetUserSessionByTokenHash(ctx, lib.TokenHash(rawToken))
}

// SignOut revokes a session by id. Errors from the store are surfaced.
func (s Service) SignOut(ctx context.Context, sessionID string) error {
	if strings.TrimSpace(sessionID) == "" {
		return lib.MissingFields("MISSING_REQUIRED_FIELDS", "session_id")
	}
	if s.deps.UserSessions == nil {
		return lib.Misconfigured("user session store is required")
	}
	return s.deps.UserSessions.RevokeUserSession(ctx, sessionID)
}

// StartOAuth records a pending OAuth flow and returns the state id used as the
// `state` URL parameter. The state is single-use and expires after 10 minutes.
func (s Service) StartOAuth(ctx context.Context, provider string, redirectTo string) (string, error) {
	if strings.TrimSpace(provider) == "" {
		return "", lib.MissingFields("MISSING_REQUIRED_FIELDS", "provider")
	}
	if s.deps.OAuthStates == nil {
		return "", lib.Misconfigured("oauth state store is required")
	}
	state := domain.OAuthState{
		ID:         lib.NewID("ostate"),
		Provider:   provider,
		RedirectTo: redirectTo,
		ExpiresAt:  time.Now().Add(oauthStateTTL),
	}
	if _, err := s.deps.OAuthStates.CreateOAuthState(ctx, state); err != nil {
		return "", err
	}
	return state.ID, nil
}

// ConsumeOAuthState atomically marks the state consumed, validates the provider
// matches, and returns the original redirect_to. A reused or expired state
// produces OAUTH_STATE_INVALID.
func (s Service) ConsumeOAuthState(ctx context.Context, stateID string, expectedProvider string) (string, error) {
	if strings.TrimSpace(stateID) == "" {
		return "", lib.Forbidden("OAUTH_STATE_INVALID", "oauth state is invalid or expired")
	}
	if s.deps.OAuthStates == nil {
		return "", lib.Misconfigured("oauth state store is required")
	}
	state, err := s.deps.OAuthStates.ConsumeOAuthState(ctx, stateID)
	if err != nil {
		return "", err
	}
	if expectedProvider != "" && state.Provider != expectedProvider {
		return "", lib.Forbidden("OAUTH_STATE_INVALID", "oauth state provider mismatch")
	}
	return state.RedirectTo, nil
}

func isNotFound(err error, code string) bool {
	if appErr, ok := err.(lib.AppError); ok {
		return appErr.Code == code
	}
	return false
}
