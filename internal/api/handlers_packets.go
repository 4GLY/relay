package api

import (
	"net/http"
	"strconv"
	"strings"

	"relay/internal/contracts"
	"relay/internal/services"
)

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
	projectID, action := projectPathParts(r.URL.Path)
	if projectID == "" {
		writeJSON(w, http.StatusBadRequest, contracts.Failure("relay show", "PROJECT_ID_REQUIRED", "missing project id in path", false, "project_id"))
		return
	}
	if action == "graph" {
		h.handleProjectGraph(w, r, projectID)
		return
	}
	if action == "retrieve" {
		h.handleProjectRetrieve(w, r, projectID)
		return
	}
	result, err := h.services.Show(r.Context(), services.ShowInput{ProjectID: projectID})
	if err != nil {
		writeServiceError(w, "relay show", err)
		return
	}
	writeJSON(w, http.StatusOK, contracts.Success("relay show", result))
}

func (h Handler) handleProjectGraph(w http.ResponseWriter, r *http.Request, projectID string) {
	result, err := h.services.ProjectGraph(r.Context(), services.ProjectGraphInput{ProjectID: projectID})
	if err != nil {
		writeServiceError(w, "relay project graph", err)
		return
	}
	writeJSON(w, http.StatusOK, contracts.Success("relay project graph", result))
}

func (h Handler) handleProjectRetrieve(w http.ResponseWriter, r *http.Request, projectID string) {
	query := strings.TrimSpace(r.URL.Query().Get("query"))
	limit := 0
	if rawLimit := strings.TrimSpace(r.URL.Query().Get("limit")); rawLimit != "" {
		parsed, err := strconv.Atoi(rawLimit)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, contracts.Failure("relay project retrieve", "INVALID_LIMIT", "limit must be an integer", false, "limit"))
			return
		}
		limit = parsed
	}
	result, err := h.services.ProjectRetrieve(r.Context(), services.ProjectRetrieveInput{
		ProjectID: projectID,
		Query:     query,
		Limit:     limit,
	})
	if err != nil {
		writeServiceError(w, "relay project retrieve", err)
		return
	}
	writeJSON(w, http.StatusOK, contracts.Success("relay project retrieve", result))
}

func projectPathParts(path string) (projectID string, action string) {
	trimmed := strings.Trim(strings.TrimPrefix(path, "/v1/projects/"), "/")
	if trimmed == "" {
		return "", ""
	}
	parts := strings.Split(trimmed, "/")
	projectID = parts[0]
	if len(parts) > 1 {
		action = parts[1]
	}
	return projectID, action
}
