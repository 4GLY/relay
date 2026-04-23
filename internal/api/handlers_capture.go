package api

import (
	"net/http"

	"relay/internal/contracts"
	"relay/internal/services"
)

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
