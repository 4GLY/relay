package api

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"relay/internal/app"
	"relay/internal/config"
	"relay/internal/contracts"
	"relay/internal/lib"
	"relay/internal/mcpserver"
	"relay/internal/services"
	"relay/internal/storage/repositories"
)

func ListenAndServe(cfg config.Config) error {
	if cfg.APIToken == "" {
		return lib.Misconfigured("RELAY_API_TOKEN is required for relay-api")
	}

	runtime, err := app.NewRuntime(context.Background(), cfg)
	if err != nil {
		return err
	}

	handler := Handler{services: runtime.Services}
	mux := buildMux(handler, cfg, runtime)

	server := &http.Server{
		Addr:    cfg.Addr,
		Handler: mux,
	}

	return server.ListenAndServe()
}

func buildMux(handler Handler, cfg config.Config, runtime app.Runtime) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", handleHealth)
	mux.HandleFunc("/v1/api-keys", requireAdminBearerToken(cfg.APIToken, handler.handleListAPIKeys))
	mux.HandleFunc("/v1/api-keys/issue", requireAdminBearerToken(cfg.APIToken, handler.handleIssueAPIKey))
	mux.HandleFunc("/v1/api-keys/revoke", requireAdminBearerToken(cfg.APIToken, handler.handleRevokeAPIKey))
	mux.HandleFunc("/v1/capture", requireBearerToken(cfg.APIToken, runtime.APIKeys, handler.handleCapture))
	mux.HandleFunc("/v1/promote", requireBearerToken(cfg.APIToken, runtime.APIKeys, handler.handlePromote))
	mux.HandleFunc("/v1/packets/build", requireBearerToken(cfg.APIToken, runtime.APIKeys, handler.handlePacketBuild))
	mux.HandleFunc("/v1/projects/", requireBearerToken(cfg.APIToken, runtime.APIKeys, handler.handleProjectShow))
	mux.Handle("/mcp", buildMCPHandler(cfg, runtime))
	return mux
}

func buildMCPHandler(cfg config.Config, runtime app.Runtime) http.Handler {
	server := mcpserver.NewFromService(runtime.Services, cfg.BaseURL, false)
	handler := mcp.NewStreamableHTTPHandler(func(_ *http.Request) *mcp.Server {
		return server.Server()
	}, &mcp.StreamableHTTPOptions{
		Stateless:    true,
		JSONResponse: true,
	})
	return requireBearerTokenHandler(cfg.APIToken, runtime.APIKeys, handler)
}

type Handler struct {
	services services.Service
}

func handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, contracts.Success("healthz", map[string]string{"status": "ok"}))
}

func requireAdminBearerToken(token string, next http.HandlerFunc) http.HandlerFunc {
	if token == "" {
		return func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusInternalServerError, contracts.Failure("api auth", "MISCONFIGURED", "admin token is not configured", false, "RELAY_API_TOKEN"))
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

		next(w, r)
	}
}

func requireBearerToken(adminToken string, apiKeys repositories.APIKeyStore, next http.HandlerFunc) http.HandlerFunc {
	if adminToken == "" && apiKeys == nil {
		return next
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
		return next
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
		return services.AuthInfo{Scope: services.APIKeyScopeGlobal}, true
	}

	if apiKeys != nil {
		if key, err := apiKeys.GetByTokenHash(r.Context(), lib.TokenHash(provided)); err == nil {
			return services.AuthInfo{
				KeyID:     key.ID,
				Scope:     services.NormalizeAPIKeyScope(key.Scope),
				ProjectID: key.ProjectID,
			}, true
		}
	}

	return services.AuthInfo{}, false
}

func (h Handler) handleCapture(w http.ResponseWriter, r *http.Request) {
	var input services.CaptureInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeJSON(w, http.StatusBadRequest, contracts.Failure("relay capture", "INVALID_JSON", err.Error(), false))
		return
	}
	result, err := h.services.Capture(r.Context(), input)
	if err != nil {
		writeServiceError(w, "relay capture", err)
		return
	}
	writeJSON(w, http.StatusOK, contracts.Success("relay capture", result))
}

func (h Handler) handleIssueAPIKey(w http.ResponseWriter, r *http.Request) {
	var input services.IssueAPIKeyInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeJSON(w, http.StatusBadRequest, contracts.Failure("relay api-key issue", "INVALID_JSON", err.Error(), false))
		return
	}
	result, err := h.services.IssueAPIKey(r.Context(), input)
	if err != nil {
		writeServiceError(w, "relay api-key issue", err)
		return
	}
	writeJSON(w, http.StatusOK, contracts.Success("relay api-key issue", result))
}

func (h Handler) handleListAPIKeys(w http.ResponseWriter, r *http.Request) {
	result, err := h.services.ListAPIKeys(r.Context())
	if err != nil {
		writeServiceError(w, "relay api-key list", err)
		return
	}
	writeJSON(w, http.StatusOK, contracts.Success("relay api-key list", result))
}

func (h Handler) handleRevokeAPIKey(w http.ResponseWriter, r *http.Request) {
	var input services.RevokeAPIKeyInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeJSON(w, http.StatusBadRequest, contracts.Failure("relay api-key revoke", "INVALID_JSON", err.Error(), false))
		return
	}
	result, err := h.services.RevokeAPIKey(r.Context(), input)
	if err != nil {
		writeServiceError(w, "relay api-key revoke", err)
		return
	}
	writeJSON(w, http.StatusOK, contracts.Success("relay api-key revoke", result))
}

func (h Handler) handlePromote(w http.ResponseWriter, r *http.Request) {
	var input services.PromoteInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeJSON(w, http.StatusBadRequest, contracts.Failure("relay promote", "INVALID_JSON", err.Error(), false))
		return
	}
	result, err := h.services.Promote(r.Context(), input)
	if err != nil {
		writeServiceError(w, "relay promote", err)
		return
	}
	writeJSON(w, http.StatusOK, contracts.Success("relay promote", result))
}

func (h Handler) handlePacketBuild(w http.ResponseWriter, r *http.Request) {
	var input services.PacketBuildInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeJSON(w, http.StatusBadRequest, contracts.Failure("relay packet build", "INVALID_JSON", err.Error(), false))
		return
	}
	result, err := h.services.BuildPacket(r.Context(), input)
	if err != nil {
		writeServiceError(w, "relay packet build", err)
		return
	}
	writeJSON(w, http.StatusOK, contracts.Success("relay packet build", result))
}

func (h Handler) handleProjectShow(w http.ResponseWriter, r *http.Request) {
	projectID := strings.TrimPrefix(r.URL.Path, "/v1/projects/")
	if projectID == "" {
		writeJSON(w, http.StatusBadRequest, contracts.Failure("relay show", "PROJECT_ID_REQUIRED", "missing project id in path", false, "project_id"))
		return
	}
	result, err := h.services.Show(r.Context(), services.ShowInput{ProjectID: projectID})
	if err != nil {
		writeServiceError(w, "relay show", err)
		return
	}
	writeJSON(w, http.StatusOK, contracts.Success("relay show", result))
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeServiceError(w http.ResponseWriter, command string, err error) {
	if appErr, ok := err.(lib.AppError); ok {
		status := http.StatusBadRequest
		if appErr.Code == "PROJECT_NOT_FOUND" {
			status = http.StatusNotFound
		}
		if appErr.Code == "API_KEY_NOT_FOUND" {
			status = http.StatusUnauthorized
		}
		if appErr.Code == "API_KEY_NOT_FOUND_BY_ID" {
			status = http.StatusNotFound
		}
		if appErr.Code == "FORBIDDEN" {
			status = http.StatusForbidden
		}
		if appErr.Code == "MISCONFIGURED" {
			status = http.StatusInternalServerError
		}
		writeJSON(w, status, contracts.Failure(command, appErr.Code, appErr.Message, appErr.Retryable, appErr.MissingFields...))
		return
	}
	writeJSON(w, http.StatusInternalServerError, contracts.Failure(command, "INTERNAL_ERROR", err.Error(), true))
}
