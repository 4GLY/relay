package config

import (
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Addr                    string
	BaseURL                 string
	PublicBaseURL           string
	DatabaseURL             string
	AdminToken              string
	APIToken                string
	OGImageDir              string
	CuratorWorkerID         string
	CuratorProvider         string
	CuratorModel            string
	CuratorBatchSize        int
	CuratorPollInterval     time.Duration
	CuratorLeaseDuration    time.Duration
	CuratorRetryBackoff     time.Duration
	CuratorMaxAttempts      int
	GitHubOAuthClientID     string
	GitHubOAuthClientSecret string
	GoogleOAuthClientID     string
	GoogleOAuthClientSecret string
	OAuthRedirectBaseURL    string
	UserSessionCookieSecure bool
	DataEncryptionKey       string
}

func Load() Config {
	addr := os.Getenv("RELAY_ADDR")
	if addr == "" {
		addr = ":8080"
	}

	databaseURL := os.Getenv("RELAY_DATABASE_URL")
	adminToken := firstNonEmpty(os.Getenv("RELAY_ADMIN_TOKEN"), os.Getenv("RELAY_API_TOKEN"))
	apiToken := os.Getenv("RELAY_API_TOKEN")
	baseURL := os.Getenv("RELAY_BASE_URL")
	workerID := os.Getenv("RELAY_CURATOR_WORKER_ID")
	if workerID == "" {
		workerID = "relay-curator"
	}
	provider := os.Getenv("RELAY_CURATOR_PROVIDER")
	if provider == "" {
		provider = "rule-based"
	}

	publicBaseURL := firstNonEmpty(os.Getenv("RELAY_PUBLIC_BASE_URL"), baseURL)
	ogImageDir := os.Getenv("RELAY_OG_IMAGE_DIR")
	if ogImageDir == "" {
		ogImageDir = "./og-images"
	}

	oauthRedirectBase := os.Getenv("RELAY_OAUTH_REDIRECT_BASE_URL")
	if oauthRedirectBase == "" {
		oauthRedirectBase = baseURL
	}

	return Config{
		Addr:                    addr,
		BaseURL:                 baseURL,
		PublicBaseURL:           publicBaseURL,
		DatabaseURL:             databaseURL,
		AdminToken:              adminToken,
		APIToken:                apiToken,
		OGImageDir:              ogImageDir,
		CuratorWorkerID:         workerID,
		CuratorProvider:         provider,
		CuratorModel:            os.Getenv("RELAY_CURATOR_MODEL"),
		CuratorBatchSize:        envInt("RELAY_CURATOR_BATCH_SIZE", 5),
		CuratorPollInterval:     envDuration("RELAY_CURATOR_POLL_INTERVAL", 5*time.Second),
		CuratorLeaseDuration:    envDuration("RELAY_CURATOR_LEASE_DURATION", 30*time.Second),
		CuratorRetryBackoff:     envDuration("RELAY_CURATOR_RETRY_BACKOFF", 30*time.Second),
		CuratorMaxAttempts:      envInt("RELAY_CURATOR_MAX_ATTEMPTS", 5),
		GitHubOAuthClientID:     os.Getenv("RELAY_GITHUB_OAUTH_CLIENT_ID"),
		GitHubOAuthClientSecret: os.Getenv("RELAY_GITHUB_OAUTH_CLIENT_SECRET"),
		GoogleOAuthClientID:     os.Getenv("RELAY_GOOGLE_OAUTH_CLIENT_ID"),
		GoogleOAuthClientSecret: os.Getenv("RELAY_GOOGLE_OAUTH_CLIENT_SECRET"),
		OAuthRedirectBaseURL:    oauthRedirectBase,
		UserSessionCookieSecure: cookieSecureForBaseURL(baseURL),
		DataEncryptionKey:       os.Getenv("RELAY_DATA_ENCRYPTION_KEY"),
	}
}

// cookieSecureForBaseURL returns false only when the base URL is local-development
// over plain HTTP. Empty / unparseable / production base URLs default to secure.
func cookieSecureForBaseURL(baseURL string) bool {
	trimmed := strings.TrimSpace(baseURL)
	if trimmed == "" {
		return true
	}
	parsed, err := url.Parse(trimmed)
	if err != nil {
		return true
	}
	if parsed.Scheme != "http" {
		return true
	}
	host := strings.ToLower(parsed.Hostname())
	if host == "localhost" || host == "127.0.0.1" || host == "::1" {
		return false
	}
	return true
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func envInt(name string, fallback int) int {
	value := os.Getenv(name)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}

func envDuration(name string, fallback time.Duration) time.Duration {
	value := os.Getenv(name)
	if value == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(value)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}
