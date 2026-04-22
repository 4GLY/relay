package api

import (
	"bytes"
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"relay/internal/app"
	"relay/internal/config"
	"relay/internal/contracts"
	"relay/internal/lib"
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

	handler := Handler{services: runtime.Services}
	mux := buildMux(handler, cfg, runtime)

	server := &http.Server{
		Addr:    cfg.Addr,
		Handler: mux,
	}

	return server.ListenAndServe()
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
	mux.HandleFunc("/v1/promote", requireBearerToken(adminToken, runtime.APIKeys, handler.handlePromote))
	mux.HandleFunc("/v1/packets/build", requireBearerToken(adminToken, runtime.APIKeys, handler.handlePacketBuild))
	mux.HandleFunc("/v1/projects/", requireBearerToken(adminToken, runtime.APIKeys, handler.handleProjectShow))
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
	return limitRequestBody(requireBearerTokenHandler(effectiveAdminToken(cfg), runtime.APIKeys, handler), maxJSONRequestBodyBytes)
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

func (h Handler) handleCapture(w http.ResponseWriter, r *http.Request) {
	var input services.CaptureInput
	if !decodeStrictJSONBody(w, r, "relay capture", &input) {
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
	if !decodeStrictJSONBody(w, r, "relay api-key issue", &input) {
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
	if !decodeStrictJSONBody(w, r, "relay api-key revoke", &input) {
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
	if !decodeStrictJSONBody(w, r, "relay promote", &input) {
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
	if !decodeStrictJSONBody(w, r, "relay packet build", &input) {
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

func decodeStrictJSONBody(w http.ResponseWriter, r *http.Request, command string, dst any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, maxJSONRequestBodyBytes)

	raw, err := io.ReadAll(r.Body)
	if err != nil {
		writeJSON(w, validationJSONStatus(err), contracts.Failure(command, validationJSONCode(err), validationJSONMessage(err), false))
		return false
	}
	if !utf8.Valid(raw) {
		writeJSON(w, http.StatusBadRequest, contracts.Failure(command, "INVALID_JSON", "request body contains malformed UTF-8", false))
		return false
	}

	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()

	if err := dec.Decode(dst); err != nil {
		writeJSON(w, validationJSONStatus(err), contracts.Failure(command, validationJSONCode(err), validationJSONMessage(err), false))
		return false
	}

	if err := dec.Decode(&struct{}{}); err != io.EOF {
		writeJSON(w, http.StatusBadRequest, contracts.Failure(command, "INVALID_JSON", "request body must contain a single JSON object", false))
		return false
	}

	return true
}

func validationJSONStatus(err error) int {
	var maxBytesErr *http.MaxBytesError
	if errors.As(err, &maxBytesErr) {
		return http.StatusRequestEntityTooLarge
	}
	return http.StatusBadRequest
}

func validationJSONCode(err error) string {
	var maxBytesErr *http.MaxBytesError
	if errors.As(err, &maxBytesErr) {
		return "REQUEST_TOO_LARGE"
	}

	if strings.HasPrefix(err.Error(), "json: unknown field ") {
		return "UNKNOWN_JSON_FIELD"
	}

	return "INVALID_JSON"
}

func validationJSONMessage(err error) string {
	var maxBytesErr *http.MaxBytesError
	if errors.As(err, &maxBytesErr) {
		return "request body exceeds 1 MiB"
	}

	if field, ok := unknownJSONField(err); ok {
		return "unknown JSON field " + field
	}

	return err.Error()
}

func unknownJSONField(err error) (string, bool) {
	const prefix = "json: unknown field "
	msg := err.Error()
	if !strings.HasPrefix(msg, prefix) {
		return "", false
	}
	return strings.TrimPrefix(msg, prefix), true
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

func limitRequestBody(next http.Handler, max int64) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Body == nil {
			next.ServeHTTP(w, r)
			return
		}

		raw, err := io.ReadAll(io.LimitReader(r.Body, max+1))
		if err != nil {
			writeJSON(w, http.StatusBadRequest, contracts.Failure("mcp transport", "INVALID_JSON", err.Error(), false))
			return
		}
		if int64(len(raw)) > max {
			writeJSON(w, http.StatusRequestEntityTooLarge, contracts.Failure("mcp transport", "REQUEST_TOO_LARGE", "request body exceeds 1 MiB", false))
			return
		}

		r.Body = io.NopCloser(bytes.NewReader(raw))
		next.ServeHTTP(w, r)
	})
}
