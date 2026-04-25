package api

import (
	"context"
	"strings"
	"time"

	"relay/internal/domain"
	"relay/internal/lib"
)

type apiFakeUserStore struct {
	items map[string]domain.User
}

func newAuthFakeUserStore() *apiFakeUserStore { return &apiFakeUserStore{items: map[string]domain.User{}} }

func (s *apiFakeUserStore) CreateUser(_ context.Context, user domain.User) (domain.User, error) {
	if _, ok := s.items[user.ID]; ok {
		return user, nil
	}
	now := time.Now()
	user.CreatedAt = now
	user.UpdatedAt = now
	s.items[user.ID] = user
	return user, nil
}

func (s *apiFakeUserStore) GetUserByID(_ context.Context, id string) (domain.User, error) {
	user, ok := s.items[id]
	if !ok {
		return domain.User{}, lib.NotFound("USER_NOT_FOUND", "user not found")
	}
	return user, nil
}

func (s *apiFakeUserStore) GetUserByEmail(_ context.Context, email string) (domain.User, error) {
	if strings.TrimSpace(email) == "" {
		return domain.User{}, lib.NotFound("USER_NOT_FOUND", "user not found")
	}
	want := strings.ToLower(email)
	for _, user := range s.items {
		if user.Email != "" && strings.ToLower(user.Email) == want {
			return user, nil
		}
	}
	return domain.User{}, lib.NotFound("USER_NOT_FOUND", "user not found")
}

func (s *apiFakeUserStore) UpdateUser(_ context.Context, user domain.User) (domain.User, error) {
	if _, ok := s.items[user.ID]; !ok {
		return domain.User{}, lib.NotFound("USER_NOT_FOUND", "user not found")
	}
	user.UpdatedAt = time.Now()
	s.items[user.ID] = user
	return user, nil
}

type apiFakeOAuthIdentityStore struct {
	items map[string]domain.OAuthIdentity
}

func newAuthFakeOAuthIdentityStore() *apiFakeOAuthIdentityStore {
	return &apiFakeOAuthIdentityStore{items: map[string]domain.OAuthIdentity{}}
}

func (s *apiFakeOAuthIdentityStore) UpsertOAuthIdentity(_ context.Context, identity domain.OAuthIdentity) (domain.OAuthIdentity, error) {
	key := identity.Provider + ":" + identity.ProviderUserID
	if existing, ok := s.items[key]; ok {
		existing.UserID = identity.UserID
		existing.ProviderLogin = identity.ProviderLogin
		if identity.VerifiedEmail != "" {
			existing.VerifiedEmail = identity.VerifiedEmail
		}
		existing.UpdatedAt = time.Now()
		s.items[key] = existing
		return existing, nil
	}
	now := time.Now()
	identity.CreatedAt = now
	identity.UpdatedAt = now
	s.items[key] = identity
	return identity, nil
}

func (s *apiFakeOAuthIdentityStore) GetOAuthIdentityByProvider(_ context.Context, provider string, providerUserID string) (domain.OAuthIdentity, error) {
	if identity, ok := s.items[provider+":"+providerUserID]; ok {
		return identity, nil
	}
	return domain.OAuthIdentity{}, lib.NotFound("OAUTH_IDENTITY_NOT_FOUND", "oauth identity not found")
}

func (s *apiFakeOAuthIdentityStore) ListOAuthIdentitiesByUser(_ context.Context, userID string) ([]domain.OAuthIdentity, error) {
	var items []domain.OAuthIdentity
	for _, identity := range s.items {
		if identity.UserID == userID {
			items = append(items, identity)
		}
	}
	return items, nil
}

type apiFakeUserSessionStore struct {
	items map[string]domain.UserSession
}

func newAuthFakeUserSessionStore() *apiFakeUserSessionStore {
	return &apiFakeUserSessionStore{items: map[string]domain.UserSession{}}
}

func (s *apiFakeUserSessionStore) CreateUserSession(_ context.Context, session domain.UserSession) (domain.UserSession, error) {
	session.CreatedAt = time.Now()
	s.items[session.ID] = session
	return session, nil
}

func (s *apiFakeUserSessionStore) GetUserSessionByTokenHash(_ context.Context, tokenHash string) (domain.UserSession, error) {
	now := time.Now()
	for _, session := range s.items {
		if session.TokenHash != tokenHash {
			continue
		}
		if session.RevokedAt != nil {
			continue
		}
		if !session.ExpiresAt.After(now) {
			continue
		}
		return session, nil
	}
	return domain.UserSession{}, lib.NotFound("USER_SESSION_NOT_FOUND", "user session not found")
}

func (s *apiFakeUserSessionStore) RotateUserSession(_ context.Context, sessionID string, currentTokenHash string, newTokenHash string, newExpiresAt time.Time) (bool, error) {
	session, ok := s.items[sessionID]
	if !ok {
		return false, nil
	}
	if session.RevokedAt != nil {
		return false, nil
	}
	if session.TokenHash != currentTokenHash {
		return false, nil
	}
	if !session.ExpiresAt.After(time.Now()) {
		return false, nil
	}
	session.TokenHash = newTokenHash
	session.ExpiresAt = newExpiresAt
	s.items[sessionID] = session
	return true, nil
}

func (s *apiFakeUserSessionStore) RevokeUserSession(_ context.Context, sessionID string) error {
	session, ok := s.items[sessionID]
	if !ok {
		return lib.NotFound("USER_SESSION_NOT_FOUND", "user session not found")
	}
	now := time.Now()
	session.RevokedAt = &now
	s.items[sessionID] = session
	return nil
}

type apiFakeOAuthStateStore struct {
	items   map[string]domain.OAuthState
	created []string
}

func newAuthFakeOAuthStateStore() *apiFakeOAuthStateStore {
	return &apiFakeOAuthStateStore{items: map[string]domain.OAuthState{}}
}

func (s *apiFakeOAuthStateStore) CreateOAuthState(_ context.Context, state domain.OAuthState) (domain.OAuthState, error) {
	state.CreatedAt = time.Now()
	s.items[state.ID] = state
	s.created = append(s.created, state.ID)
	return state, nil
}

func (s *apiFakeOAuthStateStore) ConsumeOAuthState(_ context.Context, stateID string) (domain.OAuthState, error) {
	state, ok := s.items[stateID]
	if !ok {
		return domain.OAuthState{}, lib.NotFound("OAUTH_STATE_INVALID", "oauth state is invalid or expired")
	}
	if state.ConsumedAt != nil {
		return domain.OAuthState{}, lib.NotFound("OAUTH_STATE_INVALID", "oauth state is invalid or expired")
	}
	now := time.Now()
	if !state.ExpiresAt.After(now) {
		return domain.OAuthState{}, lib.NotFound("OAUTH_STATE_INVALID", "oauth state is invalid or expired")
	}
	state.ConsumedAt = &now
	s.items[stateID] = state
	return state, nil
}

// timeFutureExample returns a moment ~30 days in the future, used by the
// session cookie helper test.
func timeFutureExample() time.Time {
	return time.Now().Add(30 * 24 * time.Hour)
}
