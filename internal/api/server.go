package api

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"net/http"
	"strings"

	"relay/internal/app"
	"relay/internal/config"
	"relay/internal/contracts"
	"relay/internal/lib"
	"relay/internal/services"
)

func ListenAndServe(cfg config.Config) error {
	runtime, err := app.NewRuntime(context.Background(), cfg)
	if err != nil {
		return err
	}

	handler := Handler{services: runtime.Services}
	mux := buildMux(handler, cfg)

	server := &http.Server{
		Addr:    cfg.Addr,
		Handler: mux,
	}

	return server.ListenAndServe()
}

func buildMux(handler Handler, cfg config.Config) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", handleHealth)
	mux.HandleFunc("/v1/capture", requireBearerToken(cfg.APIToken, handler.handleCapture))
	mux.HandleFunc("/v1/promote", requireBearerToken(cfg.APIToken, handler.handlePromote))
	mux.HandleFunc("/v1/packets/build", requireBearerToken(cfg.APIToken, handler.handlePacketBuild))
	mux.HandleFunc("/v1/projects/", requireBearerToken(cfg.APIToken, handler.handleProjectShow))
	return mux
}

type Handler struct {
	services services.Service
}

func handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, contracts.Success("healthz", map[string]string{"status": "ok"}))
}

func requireBearerToken(token string, next http.HandlerFunc) http.HandlerFunc {
	if token == "" {
		return next
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
		if appErr.Code == "MISCONFIGURED" {
			status = http.StatusInternalServerError
		}
		writeJSON(w, status, contracts.Failure(command, appErr.Code, appErr.Message, appErr.Retryable, appErr.MissingFields...))
		return
	}
	writeJSON(w, http.StatusInternalServerError, contracts.Failure(command, "INTERNAL_ERROR", err.Error(), true))
}
