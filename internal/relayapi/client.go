package relayapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"

	"relay/internal/contracts"
	"relay/internal/services"
)

const (
	defaultBaseURL         = "https://relay.4gly.dev"
	defaultKeychainService = "codex.relay-api"
	defaultAdminAccount    = "admin-token"
	defaultClientAccount   = "client-token"
	defaultBaseURLAccount  = "base-url"
)

type Client struct {
	baseURL     string
	adminToken  string
	clientToken string
	httpClient  *http.Client
}

type Config struct {
	BaseURL         string
	AdminToken      string
	ClientToken     string
	KeychainService string
	AdminAccount    string
	ClientAccount   string
	BaseURLAccount  string
	HTTPClient      *http.Client
}

type HealthResult struct {
	Status string `json:"status"`
}

type AppError struct {
	Command string
	Code    string
	Message string
}

func (e AppError) Error() string {
	if e.Code == "" {
		return e.Message
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func LoadConfigFromEnv() Config {
	return Config{
		BaseURL:         os.Getenv("RELAY_BASE_URL"),
		AdminToken:      firstNonEmpty(os.Getenv("RELAY_ADMIN_TOKEN"), os.Getenv("RELAY_API_TOKEN")),
		ClientToken:     firstNonEmpty(os.Getenv("RELAY_CLIENT_TOKEN"), os.Getenv("RELAY_MCP_TOKEN")),
		KeychainService: firstNonEmpty(os.Getenv("RELAY_KEYCHAIN_SERVICE"), defaultKeychainService),
		AdminAccount:    firstNonEmpty(os.Getenv("RELAY_KEYCHAIN_ADMIN_ACCOUNT"), defaultAdminAccount),
		ClientAccount:   firstNonEmpty(os.Getenv("RELAY_KEYCHAIN_CLIENT_ACCOUNT"), defaultClientAccount),
		BaseURLAccount:  firstNonEmpty(os.Getenv("RELAY_KEYCHAIN_BASE_URL_ACCOUNT"), defaultBaseURLAccount),
	}
}

func NewClient(cfg Config) *Client {
	if cfg.KeychainService == "" {
		cfg.KeychainService = defaultKeychainService
	}
	if cfg.AdminAccount == "" {
		cfg.AdminAccount = defaultAdminAccount
	}
	if cfg.ClientAccount == "" {
		cfg.ClientAccount = defaultClientAccount
	}
	if cfg.BaseURLAccount == "" {
		cfg.BaseURLAccount = defaultBaseURLAccount
	}
	if cfg.BaseURL == "" {
		if v, ok := readKeychainSecret(cfg.KeychainService, cfg.BaseURLAccount); ok {
			cfg.BaseURL = v
		}
	}
	if cfg.BaseURL == "" {
		cfg.BaseURL = defaultBaseURL
	}
	if cfg.AdminToken == "" {
		if v, ok := readKeychainSecret(cfg.KeychainService, cfg.AdminAccount); ok {
			cfg.AdminToken = v
		}
	}
	if cfg.ClientToken == "" {
		if v, ok := readKeychainSecret(cfg.KeychainService, cfg.ClientAccount); ok {
			cfg.ClientToken = v
		}
	}
	if cfg.HTTPClient == nil {
		cfg.HTTPClient = &http.Client{Timeout: 15 * time.Second}
	}
	return &Client{
		baseURL:     strings.TrimRight(cfg.BaseURL, "/"),
		adminToken:  cfg.AdminToken,
		clientToken: cfg.ClientToken,
		httpClient:  cfg.HTTPClient,
	}
}

func (c *Client) HasAdminToken() bool {
	return c.adminToken != ""
}

func (c *Client) BaseURL() string {
	return c.baseURL
}

func (c *Client) Health(ctx context.Context) (HealthResult, error) {
	return doJSON[HealthResult](ctx, c.httpClient, "", http.MethodGet, c.baseURL+"/healthz", nil)
}

func (c *Client) Capture(ctx context.Context, input services.CaptureInput) (services.CaptureResult, error) {
	return doJSON[services.CaptureResult](ctx, c.httpClient, c.clientToken, http.MethodPost, c.baseURL+"/v1/capture", input)
}

func (c *Client) Promote(ctx context.Context, input services.PromoteInput) (services.PromoteResult, error) {
	return doJSON[services.PromoteResult](ctx, c.httpClient, c.clientToken, http.MethodPost, c.baseURL+"/v1/promote", input)
}

func (c *Client) BuildPacket(ctx context.Context, input services.PacketBuildInput) (services.PacketBuildResult, error) {
	return doJSON[services.PacketBuildResult](ctx, c.httpClient, c.clientToken, http.MethodPost, c.baseURL+"/v1/packets/build", input)
}

func (c *Client) Show(ctx context.Context, projectID string) (services.ShowResult, error) {
	return doJSON[services.ShowResult](ctx, c.httpClient, c.clientToken, http.MethodGet, c.baseURL+"/v1/projects/"+projectID, nil)
}

func (c *Client) ProjectGraph(ctx context.Context, projectID string) (services.ProjectGraphResult, error) {
	return doJSON[services.ProjectGraphResult](ctx, c.httpClient, c.clientToken, http.MethodGet, c.baseURL+"/v1/projects/"+projectID+"/graph", nil)
}

func (c *Client) ProjectRetrieve(ctx context.Context, projectID string, query string, limit int) (services.ProjectRetrieveResult, error) {
	endpoint := c.baseURL + "/v1/projects/" + projectID + "/retrieve?query=" + url.QueryEscape(query)
	if limit > 0 {
		endpoint += "&limit=" + strconv.Itoa(limit)
	}
	return doJSON[services.ProjectRetrieveResult](ctx, c.httpClient, c.clientToken, http.MethodGet, endpoint, nil)
}

func (c *Client) IssueAPIKey(ctx context.Context, input services.IssueAPIKeyInput) (services.IssueAPIKeyResult, error) {
	return doJSON[services.IssueAPIKeyResult](ctx, c.httpClient, c.adminToken, http.MethodPost, c.baseURL+"/v1/api-keys/issue", input)
}

func (c *Client) ListAPIKeys(ctx context.Context) (services.ListAPIKeysResult, error) {
	return doJSON[services.ListAPIKeysResult](ctx, c.httpClient, c.adminToken, http.MethodGet, c.baseURL+"/v1/api-keys", nil)
}

func (c *Client) RevokeAPIKey(ctx context.Context, input services.RevokeAPIKeyInput) (services.RevokeAPIKeyResult, error) {
	return doJSON[services.RevokeAPIKeyResult](ctx, c.httpClient, c.adminToken, http.MethodPost, c.baseURL+"/v1/api-keys/revoke", input)
}

func doJSON[T any](ctx context.Context, client *http.Client, token string, method string, url string, payload any) (T, error) {
	var zero T
	var body io.Reader
	if payload != nil {
		raw, err := json.Marshal(payload)
		if err != nil {
			return zero, err
		}
		body = bytes.NewReader(raw)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return zero, err
	}
	req.Header.Set("Accept", "application/json")
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	res, err := client.Do(req)
	if err != nil {
		return zero, err
	}
	defer res.Body.Close()

	if res.StatusCode >= 400 {
		var failure contracts.ErrorEnvelope
		if err := json.NewDecoder(res.Body).Decode(&failure); err == nil && failure.Error.Code != "" {
			return zero, AppError{Command: failure.Command, Code: failure.Error.Code, Message: failure.Error.Message}
		}
		raw, _ := io.ReadAll(res.Body)
		return zero, fmt.Errorf("relay api %s %s failed with status %d: %s", method, url, res.StatusCode, strings.TrimSpace(string(raw)))
	}

	var success struct {
		OK       bool            `json:"ok"`
		Command  string          `json:"command"`
		Data     json.RawMessage `json:"data"`
		Warnings []string        `json:"warnings"`
	}
	if err := json.NewDecoder(res.Body).Decode(&success); err != nil {
		return zero, err
	}
	if len(success.Data) == 0 || string(success.Data) == "null" {
		return zero, nil
	}
	if err := json.Unmarshal(success.Data, &zero); err != nil {
		return zero, err
	}
	return zero, nil
}

func readKeychainSecret(service string, account string) (string, bool) {
	if runtime.GOOS != "darwin" {
		return "", false
	}
	if _, err := exec.LookPath("security"); err != nil {
		return "", false
	}
	out, err := exec.Command("security", "find-generic-password", "-w", "-s", service, "-a", account).Output()
	if err != nil {
		return "", false
	}
	value := strings.TrimSpace(string(out))
	if value == "" {
		return "", false
	}
	return value, true
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
