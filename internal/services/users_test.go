package services

import (
	"context"
	"testing"

	"relay/internal/lib/oauth"
)

func newAuthService() (Service, *fakeUserStore, *fakeOAuthIdentityStore, *fakeUserSessionStore, *fakeOAuthStateStore) {
	users := &fakeUserStore{}
	identities := &fakeOAuthIdentityStore{}
	sessions := &fakeUserSessionStore{}
	states := &fakeOAuthStateStore{}
	svc := New(Dependencies{
		Users:           users,
		OAuthIdentities: identities,
		UserSessions:    sessions,
		OAuthStates:     states,
	})
	return svc, users, identities, sessions, states
}

func TestSignInWithOAuthProfileCreatesNewUser(t *testing.T) {
	svc, users, identities, sessions, _ := newAuthService()

	result, err := svc.SignInWithOAuthProfile(context.Background(), oauth.Profile{
		Provider:       "github",
		ProviderUserID: "1001",
		Login:          "octo",
		Email:          "octo@example.com",
		DisplayName:    "Octo Cat",
		AvatarURL:      "https://avatars/1001",
	})
	if err != nil {
		t.Fatalf("SignInWithOAuthProfile returned error: %v", err)
	}
	if result.SessionToken == "" {
		t.Fatal("expected session token")
	}
	if len(users.items) != 1 {
		t.Fatalf("expected one user, got %d", len(users.items))
	}
	if len(identities.items) != 1 {
		t.Fatalf("expected one identity, got %d", len(identities.items))
	}
	if len(sessions.items) != 1 {
		t.Fatalf("expected one session, got %d", len(sessions.items))
	}
	if result.User.Email != "octo@example.com" || result.User.DisplayName != "Octo Cat" {
		t.Fatalf("unexpected user payload: %#v", result.User)
	}
}

func TestSignInWithOAuthProfileReusesExistingIdentity(t *testing.T) {
	svc, users, identities, _, _ := newAuthService()

	first, err := svc.SignInWithOAuthProfile(context.Background(), oauth.Profile{
		Provider:       "github",
		ProviderUserID: "1001",
		Login:          "octo",
		Email:          "octo@example.com",
		DisplayName:    "Octo Cat",
	})
	if err != nil {
		t.Fatalf("first sign-in error: %v", err)
	}

	second, err := svc.SignInWithOAuthProfile(context.Background(), oauth.Profile{
		Provider:       "github",
		ProviderUserID: "1001",
		Login:          "octo",
		Email:          "octo@example.com",
		DisplayName:    "Octo Cat",
	})
	if err != nil {
		t.Fatalf("second sign-in error: %v", err)
	}

	if second.User.ID != first.User.ID {
		t.Fatalf("expected stable user id, got %q vs %q", first.User.ID, second.User.ID)
	}
	if len(users.items) != 1 {
		t.Fatalf("expected one user, got %d", len(users.items))
	}
	if len(identities.items) != 1 {
		t.Fatalf("expected one identity, got %d", len(identities.items))
	}
	if first.SessionToken == second.SessionToken {
		t.Fatalf("expected fresh session token on each sign-in")
	}
}

func TestSignInWithOAuthProfileAutoLinksByVerifiedEmail(t *testing.T) {
	svc, users, identities, _, _ := newAuthService()

	first, err := svc.SignInWithOAuthProfile(context.Background(), oauth.Profile{
		Provider:       "github",
		ProviderUserID: "1001",
		Login:          "octo",
		Email:          "octo@example.com",
		DisplayName:    "Octo Cat",
	})
	if err != nil {
		t.Fatalf("github sign-in error: %v", err)
	}

	second, err := svc.SignInWithOAuthProfile(context.Background(), oauth.Profile{
		Provider:       "google",
		ProviderUserID: "google-2002",
		Login:          "octo@example.com",
		Email:          "octo@example.com",
		DisplayName:    "Octo Cat",
	})
	if err != nil {
		t.Fatalf("google sign-in error: %v", err)
	}

	if second.User.ID != first.User.ID {
		t.Fatalf("expected auto-link to existing user, got %q vs %q", first.User.ID, second.User.ID)
	}
	if len(users.items) != 1 {
		t.Fatalf("expected one user, got %d", len(users.items))
	}
	if len(identities.items) != 2 {
		t.Fatalf("expected two oauth identities (github + google), got %d", len(identities.items))
	}
}

func TestSignInWithOAuthProfileWithoutEmailCreatesUser(t *testing.T) {
	svc, users, identities, _, _ := newAuthService()

	result, err := svc.SignInWithOAuthProfile(context.Background(), oauth.Profile{
		Provider:       "github",
		ProviderUserID: "1001",
		Login:          "stealth",
		DisplayName:    "Stealth Mode",
	})
	if err != nil {
		t.Fatalf("SignInWithOAuthProfile error: %v", err)
	}
	if result.User.Email != "" {
		t.Fatalf("expected empty email when profile has no email, got %q", result.User.Email)
	}
	if len(users.items) != 1 {
		t.Fatalf("expected one user, got %d", len(users.items))
	}
	if len(identities.items) != 1 {
		t.Fatalf("expected one identity, got %d", len(identities.items))
	}
}

func TestGetUserBySessionTokenAndSignOut(t *testing.T) {
	svc, _, _, _, _ := newAuthService()

	result, err := svc.SignInWithOAuthProfile(context.Background(), oauth.Profile{
		Provider:       "github",
		ProviderUserID: "1001",
		Login:          "octo",
		Email:          "octo@example.com",
		DisplayName:    "Octo",
	})
	if err != nil {
		t.Fatalf("sign-in error: %v", err)
	}

	user, err := svc.GetUserBySessionToken(context.Background(), result.SessionToken)
	if err != nil {
		t.Fatalf("GetUserBySessionToken error: %v", err)
	}
	if user.ID != result.User.ID {
		t.Fatalf("expected user %q, got %q", result.User.ID, user.ID)
	}

	session, err := svc.GetSessionByToken(context.Background(), result.SessionToken)
	if err != nil {
		t.Fatalf("GetSessionByToken error: %v", err)
	}
	if err := svc.SignOut(context.Background(), session.ID); err != nil {
		t.Fatalf("SignOut error: %v", err)
	}
	if _, err := svc.GetUserBySessionToken(context.Background(), result.SessionToken); err == nil {
		t.Fatal("expected revoked session to be rejected")
	}
}

func TestStartAndConsumeOAuthState(t *testing.T) {
	svc, _, _, _, _ := newAuthService()
	stateID, err := svc.StartOAuth(context.Background(), "github", "/dashboard")
	if err != nil {
		t.Fatalf("StartOAuth error: %v", err)
	}
	redirect, err := svc.ConsumeOAuthState(context.Background(), stateID, "github")
	if err != nil {
		t.Fatalf("ConsumeOAuthState error: %v", err)
	}
	if redirect != "/dashboard" {
		t.Fatalf("expected /dashboard, got %q", redirect)
	}
	if _, err := svc.ConsumeOAuthState(context.Background(), stateID, "github"); err == nil {
		t.Fatal("expected single-use state to fail second consumption")
	}
}

func TestConsumeOAuthStateRejectsProviderMismatch(t *testing.T) {
	svc, _, _, _, _ := newAuthService()
	stateID, err := svc.StartOAuth(context.Background(), "github", "/")
	if err != nil {
		t.Fatalf("StartOAuth error: %v", err)
	}
	if _, err := svc.ConsumeOAuthState(context.Background(), stateID, "google"); err == nil {
		t.Fatal("expected provider mismatch to fail")
	}
}
