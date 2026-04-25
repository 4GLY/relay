package oauth

import "context"

// Profile is the normalized identity returned by an OAuth provider.
// Email is empty when the provider could not produce a verified email
// (for example a GitHub user who has hidden their primary email).
type Profile struct {
	Provider       string
	ProviderUserID string
	Login          string
	Email          string
	DisplayName    string
	AvatarURL      string
}

// Provider exchanges an OAuth authorization code for a normalized Profile.
type Provider interface {
	Name() string
	AuthURL(state string, redirectURI string) string
	Exchange(ctx context.Context, code string, redirectURI string) (Profile, error)
}
