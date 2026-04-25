package oauth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

const (
	githubAuthorizeURL = "https://github.com/login/oauth/authorize"
	githubTokenURL     = "https://github.com/login/oauth/access_token"
	githubUserURL      = "https://api.github.com/user"
	githubEmailsURL    = "https://api.github.com/user/emails"
)

type githubProvider struct {
	clientID     string
	clientSecret string
	httpClient   *http.Client
}

// NewGitHub constructs a GitHub OAuth provider. clientID and clientSecret are
// taken from the GitHub OAuth application settings.
func NewGitHub(clientID string, clientSecret string) Provider {
	return &githubProvider{
		clientID:     clientID,
		clientSecret: clientSecret,
		httpClient:   http.DefaultClient,
	}
}

func (p *githubProvider) Name() string { return "github" }

func (p *githubProvider) AuthURL(state string, redirectURI string) string {
	params := url.Values{}
	params.Set("client_id", p.clientID)
	params.Set("redirect_uri", redirectURI)
	params.Set("scope", "read:user user:email")
	params.Set("state", state)
	params.Set("allow_signup", "true")
	return githubAuthorizeURL + "?" + params.Encode()
}

func (p *githubProvider) Exchange(ctx context.Context, code string, redirectURI string) (Profile, error) {
	token, err := p.exchangeCode(ctx, code, redirectURI)
	if err != nil {
		return Profile{}, err
	}

	user, err := p.fetchUser(ctx, token)
	if err != nil {
		return Profile{}, err
	}

	email := strings.TrimSpace(user.Email)
	primary, err := p.fetchPrimaryVerifiedEmail(ctx, token)
	if err == nil && primary != "" {
		email = primary
	}

	displayName := strings.TrimSpace(user.Name)
	if displayName == "" {
		displayName = user.Login
	}

	return Profile{
		Provider:       p.Name(),
		ProviderUserID: strconv.FormatInt(user.ID, 10),
		Login:          user.Login,
		Email:          email,
		DisplayName:    displayName,
		AvatarURL:      user.AvatarURL,
	}, nil
}

type githubUserResponse struct {
	ID        int64  `json:"id"`
	Login     string `json:"login"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	AvatarURL string `json:"avatar_url"`
}

type githubEmailEntry struct {
	Email    string `json:"email"`
	Primary  bool   `json:"primary"`
	Verified bool   `json:"verified"`
}

type githubTokenResponse struct {
	AccessToken      string `json:"access_token"`
	Scope            string `json:"scope"`
	TokenType        string `json:"token_type"`
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description"`
}

func (p *githubProvider) exchangeCode(ctx context.Context, code string, redirectURI string) (string, error) {
	form := url.Values{}
	form.Set("client_id", p.clientID)
	form.Set("client_secret", p.clientSecret)
	form.Set("code", code)
	form.Set("redirect_uri", redirectURI)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, githubTokenURL, strings.NewReader(form.Encode()))
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
		return "", fmt.Errorf("github token exchange failed: status=%d body=%s", resp.StatusCode, truncateForError(body))
	}

	var token githubTokenResponse
	if err := json.Unmarshal(body, &token); err != nil {
		return "", fmt.Errorf("github token exchange decode: %w", err)
	}
	if token.Error != "" {
		return "", fmt.Errorf("github token exchange error: %s (%s)", token.Error, token.ErrorDescription)
	}
	if token.AccessToken == "" {
		return "", fmt.Errorf("github token exchange returned empty access_token")
	}
	return token.AccessToken, nil
}

func (p *githubProvider) fetchUser(ctx context.Context, token string) (githubUserResponse, error) {
	var user githubUserResponse
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, githubUserURL, nil)
	if err != nil {
		return user, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return user, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return user, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return user, fmt.Errorf("github user fetch failed: status=%d body=%s", resp.StatusCode, truncateForError(body))
	}
	if err := json.Unmarshal(body, &user); err != nil {
		return user, fmt.Errorf("github user decode: %w", err)
	}
	if user.ID == 0 {
		return user, fmt.Errorf("github user response missing id")
	}
	return user, nil
}

// fetchPrimaryVerifiedEmail mitigates failure mode E4: GitHub's /user endpoint
// returns an empty email when the primary address is private. We always look up
// /user/emails after the user fetch and pick the primary verified entry.
func (p *githubProvider) fetchPrimaryVerifiedEmail(ctx context.Context, token string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, githubEmailsURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("github emails fetch failed: status=%d body=%s", resp.StatusCode, truncateForError(body))
	}

	var entries []githubEmailEntry
	if err := json.NewDecoder(resp.Body).Decode(&entries); err != nil {
		return "", fmt.Errorf("github emails decode: %w", err)
	}
	for _, entry := range entries {
		if entry.Primary && entry.Verified {
			return strings.TrimSpace(entry.Email), nil
		}
	}
	for _, entry := range entries {
		if entry.Verified {
			return strings.TrimSpace(entry.Email), nil
		}
	}
	return "", nil
}

func truncateForError(body []byte) string {
	const limit = 256
	if len(body) <= limit {
		return string(body)
	}
	return string(body[:limit]) + "..."
}
