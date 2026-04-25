package api

import (
	"context"
	"crypto/subtle"
	"net/http"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"relay/internal/app"
	"relay/internal/config"
	"relay/internal/contracts"
	"relay/internal/lib"
	"relay/internal/lib/oauth"
	"relay/internal/mcpserver"
	"relay/internal/services"
	"relay/internal/storage/repositories"
)

const maxJSONRequestBodyBytes = 1 << 20

func ListenAndServe(cfg config.Config) error {
	if err := requireStartupAdminToken(cfg); err != nil {
		return err
	}
	runtime, err := app.NewRuntime(context.Background(), cfg)
	if err != nil {
		return err
	}

	handler := Handler{
		services:          runtime.Services,
		oauth:             buildOAuthRegistry(cfg),
		oauthRedirectBase: oauthRedirectBase(cfg),
		cookieSecure:      cfg.UserSessionCookieSecure,
	}
	mux := buildMux(handler, cfg, runtime)

	server := &http.Server{
		Addr:    cfg.Addr,
		Handler: mux,
	}

	return server.ListenAndServe()
}

func buildOAuthRegistry(cfg config.Config) oauth.Registry {
	providers := make([]oauth.Provider, 0, 2)
	if cfg.GitHubOAuthClientID != "" && cfg.GitHubOAuthClientSecret != "" {
		providers = append(providers, oauth.NewGitHub(cfg.GitHubOAuthClientID, cfg.GitHubOAuthClientSecret))
	}
	if cfg.GoogleOAuthClientID != "" && cfg.GoogleOAuthClientSecret != "" {
		providers = append(providers, oauth.NewGoogle(cfg.GoogleOAuthClientID, cfg.GoogleOAuthClientSecret))
	}
	return oauth.NewRegistry(providers...)
}

func oauthRedirectBase(cfg config.Config) string {
	if cfg.OAuthRedirectBaseURL != "" {
		return cfg.OAuthRedirectBaseURL
	}
	return cfg.BaseURL
}

func requireStartupAdminToken(cfg config.Config) error {
	if effectiveAdminToken(cfg) == "" {
		return lib.Misconfigured("RELAY_ADMIN_TOKEN or RELAY_API_TOKEN is required for relay-api")
	}
	return nil
}

func buildMux(handler Handler, cfg config.Config, runtime app.Runtime) *http.ServeMux {
	adminToken := effectiveAdminToken(cfg)
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", handleHealth)
	mux.HandleFunc("/v1/api-keys", requireAdminBearerToken(adminToken, handler.handleListAPIKeys))
	mux.HandleFunc("/v1/api-keys/issue", requireAdminBearerToken(adminToken, handler.handleIssueAPIKey))
	mux.HandleFunc("/v1/api-keys/revoke", requireAdminBearerToken(adminToken, handler.handleRevokeAPIKey))
	mux.HandleFunc("/v1/capture", requireBearerToken(adminToken, runtime.APIKeys, handler.handleCapture))
	mux.HandleFunc("/v1/judgment-traces", requireBearerToken(adminToken, runtime.APIKeys, handler.handleJudgmentTraceWrite))
	mux.HandleFunc("/v1/heuristic-proposals", requireBearerToken(adminToken, runtime.APIKeys, handler.handleHeuristicProposalCreate))
	mux.HandleFunc("/v1/heuristic-proposals/review", requireAdminBearerToken(adminToken, handler.handleHeuristicProposalReview))
	mux.HandleFunc("/v1/approved-heuristics/update", requireAdminBearerToken(adminToken, handler.handleApprovedHeuristicUpdate))
	mux.HandleFunc("/v1/promote", requireBearerToken(adminToken, runtime.APIKeys, handler.handlePromote))
	mux.HandleFunc("/v1/packets/build", requireBearerToken(adminToken, runtime.APIKeys, handler.handlePacketBuild))
	mux.HandleFunc("/v1/projects/", requireBearerToken(adminToken, runtime.APIKeys, handler.handleProjectShow))
	mux.HandleFunc("/v1/snapshots/", requireAdminBearerToken(adminToken, handler.handleSnapshotAdmin))
	mux.HandleFunc("/p/", handler.handlePublicSnapshotPage)
	mux.HandleFunc("/v1/auth/", handler.handleAuthRouter)
	mux.Handle("/mcp", buildMCPHandler(cfg, runtime))
	return mux
}

// handleSnapshotAdmin dispatches /v1/snapshots/{id}/{publish|revoke} so
// they share a single bearer-auth gate.
func (h Handler) handleSnapshotAdmin(w http.ResponseWriter, r *http.Request) {
	_, action, ok := parseSnapshotAdminPath(r.URL.Path)
	if !ok {
		writeJSON(w, http.StatusNotFound, contracts.Failure("relay snapshot admin", "NOT_FOUND", "unknown snapshot route", false, "path"))
		return
	}
	switch action {
	case "publish":
		h.handleSnapshotPublish(w, r)
	case "revoke":
		h.handleSnapshotRevoke(w, r)
	default:
		writeJSON(w, http.StatusNotFound, contracts.Failure("relay snapshot admin", "NOT_FOUND", "unknown snapshot route", false, "path"))
	}
}

func buildMCPHandler(cfg config.Config, runtime app.Runtime) http.Handler {
	server := mcpserver.NewFromService(runtime.Services, cfg.BaseURL, false)
	handler := mcp.NewStreamableHTTPHandler(func(_ *http.Request) *mcp.Server {
		return server.Server()
	}, &mcp.StreamableHTTPOptions{
		Stateless:    true,
		JSONResponse: true,
	})
	return limitRequestBody(requireBearerTokenHandler(effectiveAdminToken(cfg), runtime.APIKeys, handler), maxJSONRequestBodyBytes)
}

type Handler struct {
	services          services.Service
	oauth             oauth.Registry
	oauthRedirectBase string
	cookieSecure      bool
}

func handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, contracts.Success("healthz", map[string]string{"status": "ok"}))
}

func requireAdminBearerToken(token string, next http.HandlerFunc) http.HandlerFunc {
	if token == "" {
		return func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusInternalServerError, contracts.Failure("api auth", "MISCONFIGURED", "admin token is not configured", false, "RELAY_ADMIN_TOKEN or RELAY_API_TOKEN"))
		}
	}

	return func(w http.ResponseWriter, r *http.Request) {
		header := r.Header.Get("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			writeJSON(w, http.StatusUnauthorized, contracts.Failure("api auth", "UNAUTHORIZED", "missing or invalid bearer token", false, "Authorization"))
			return
		}

		provided := strings.TrimSpace(strings.TrimPrefix(header, "Bearer "))
		if provided == "" || subtle.ConstantTimeCompare([]byte(provided), []byte(token)) != 1 {
			writeJSON(w, http.StatusUnauthorized, contracts.Failure("api auth", "UNAUTHORIZED", "missing or invalid bearer token", false, "Authorization"))
			return
		}

		next(w, r.WithContext(services.ContextWithAuthInfo(r.Context(), services.AuthInfo{
			IsAdmin: true,
			Scope:   services.APIKeyScopeGlobal,
		})))
	}
}

func requireBearerToken(adminToken string, apiKeys repositories.APIKeyStore, next http.HandlerFunc) http.HandlerFunc {
	if adminToken == "" && apiKeys == nil {
		return func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusInternalServerError, contracts.Failure("api auth", "MISCONFIGURED", "bearer auth is not configured", false, "RELAY_ADMIN_TOKEN or RELAY_API_TOKEN"))
		}
	}

	return func(w http.ResponseWriter, r *http.Request) {
		authInfo, ok := authorizeBearerToken(r, adminToken, apiKeys)
		if !ok {
			writeJSON(w, http.StatusUnauthorized, contracts.Failure("api auth", "UNAUTHORIZED", "missing or invalid bearer token", false, "Authorization"))
			return
		}
		next(w, r.WithContext(services.ContextWithAuthInfo(r.Context(), authInfo)))
	}
}

func requireBearerTokenHandler(adminToken string, apiKeys repositories.APIKeyStore, next http.Handler) http.Handler {
	if adminToken == "" && apiKeys == nil {
		return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusInternalServerError, contracts.Failure("mcp auth", "MISCONFIGURED", "bearer auth is not configured", false, "RELAY_ADMIN_TOKEN or RELAY_API_TOKEN"))
		})
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authInfo, ok := authorizeBearerToken(r, adminToken, apiKeys)
		if !ok {
			writeJSON(w, http.StatusUnauthorized, contracts.Failure("mcp auth", "UNAUTHORIZED", "missing or invalid bearer token", false, "Authorization"))
			return
		}
		next.ServeHTTP(w, r.WithContext(services.ContextWithAuthInfo(r.Context(), authInfo)))
	})
}

func effectiveAdminToken(cfg config.Config) string {
	if cfg.AdminToken != "" {
		return cfg.AdminToken
	}
	return cfg.APIToken
}

func authorizeBearerToken(r *http.Request, adminToken string, apiKeys repositories.APIKeyStore) (services.AuthInfo, bool) {
	header := r.Header.Get("Authorization")
	if !strings.HasPrefix(header, "Bearer ") {
		return services.AuthInfo{}, false
	}

	provided := strings.TrimSpace(strings.TrimPrefix(header, "Bearer "))
	if provided == "" {
		return services.AuthInfo{}, false
	}

	if adminToken != "" && subtle.ConstantTimeCompare([]byte(provided), []byte(adminToken)) == 1 {
		return services.AuthInfo{
			IsAdmin: true,
			Scope:   services.APIKeyScopeGlobal,
		}, true
	}

	if apiKeys != nil {
		if key, err := apiKeys.GetByTokenHash(r.Context(), lib.TokenHash(provided)); err == nil {
			if !services.IsKnownAPIKeyScope(key.Scope) {
				return services.AuthInfo{}, false
			}
			if services.NormalizeAPIKeyScope(key.Scope) == services.APIKeyScopeProject && key.ProjectID == "" {
				return services.AuthInfo{}, false
			}
			return services.AuthInfo{
				KeyID:     key.ID,
				Scope:     services.NormalizeAPIKeyScope(key.Scope),
				ProjectID: key.ProjectID,
			}, true
		}
	}

	return services.AuthInfo{}, false
}
