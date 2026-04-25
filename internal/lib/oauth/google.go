package oauth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

const (
	googleAuthorizeURL = "https://accounts.google.com/o/oauth2/v2/auth"
	googleTokenURL     = "https://oauth2.googleapis.com/token"
	googleUserinfoURL  = "https://openidconnect.googleapis.com/v1/userinfo"
)

type googleProvider struct {
	clientID     string
	clientSecret string
	httpClient   *http.Client
}

// NewGoogle constructs a Google OAuth provider using OpenID Connect for the
// userinfo lookup. clientID and clientSecret come from the Google Cloud Console.
func NewGoogle(clientID string, clientSecret string) Provider {
	return &googleProvider{
		clientID:     clientID,
		clientSecret: clientSecret,
		httpClient:   http.DefaultClient,
	}
}

func (p *googleProvider) Name() string { return "google" }

func (p *googleProvider) AuthURL(state string, redirectURI string) string {
	params := url.Values{}
	params.Set("client_id", p.clientID)
	params.Set("redirect_uri", redirectURI)
	params.Set("response_type", "code")
	params.Set("scope", "openid email profile")
	params.Set("state", state)
	params.Set("access_type", "online")
	params.Set("prompt", "select_account")
	return googleAuthorizeURL + "?" + params.Encode()
}

func (p *googleProvider) Exchange(ctx context.Context, code string, redirectURI string) (Profile, error) {
	token, err := p.exchangeCode(ctx, code, redirectURI)
	if err != nil {
		return Profile{}, err
	}

	info, err := p.fetchUserinfo(ctx, token)
	if err != nil {
		return Profile{}, err
	}
	if info.Sub == "" {
		return Profile{}, fmt.Errorf("google userinfo missing sub")
	}

	email := ""
	if info.EmailVerified {
		email = strings.TrimSpace(info.Email)
	}

	login := info.Email
	if login == "" {
		login = info.Sub
	}

	return Profile{
		Provider:       p.Name(),
		ProviderUserID: info.Sub,
		Login:          login,
		Email:          email,
		DisplayName:    strings.TrimSpace(info.Name),
		AvatarURL:      info.Picture,
	}, nil
}

type googleTokenResponse struct {
	AccessToken      string `json:"access_token"`
	IDToken          string `json:"id_token"`
	ExpiresIn        int    `json:"expires_in"`
	TokenType        string `json:"token_type"`
	Scope            string `json:"scope"`
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description"`
}

type googleUserinfoResponse struct {
	Sub           string `json:"sub"`
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	Name          string `json:"name"`
	Picture       string `json:"picture"`
}

func (p *googleProvider) exchangeCode(ctx context.Context, code string, redirectURI string) (string, error) {
	form := url.Values{}
	form.Set("client_id", p.clientID)
	form.Set("client_secret", p.clientSecret)
	form.Set("code", code)
	form.Set("redirect_uri", redirectURI)
	form.Set("grant_type", "authorization_code")

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, googleTokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("google token exchange failed: status=%d body=%s", resp.StatusCode, truncateForError(body))
	}

	var token googleTokenResponse
	if err := json.Unmarshal(body, &token); err != nil {
		return "", fmt.Errorf("google token exchange decode: %w", err)
	}
	if token.Error != "" {
		return "", fmt.Errorf("google token exchange error: %s (%s)", token.Error, token.ErrorDescription)
	}
	if token.AccessToken == "" {
		return "", fmt.Errorf("google token exchange returned empty access_token")
	}
	return token.AccessToken, nil
}

func (p *googleProvider) fetchUserinfo(ctx context.Context, token string) (googleUserinfoResponse, error) {
	var info googleUserinfoResponse
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, googleUserinfoURL, nil)
	if err != nil {
		return info, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return info, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return info, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return info, fmt.Errorf("google userinfo failed: status=%d body=%s", resp.StatusCode, truncateForError(body))
	}
	if err := json.Unmarshal(body, &info); err != nil {
		return info, fmt.Errorf("google userinfo decode: %w", err)
	}
	return info, nil
}
