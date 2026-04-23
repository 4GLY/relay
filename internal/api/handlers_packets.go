package api

import (
	"net/http"
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
