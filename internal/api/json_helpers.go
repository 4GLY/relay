package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"unicode/utf8"

	"relay/internal/contracts"
	"relay/internal/lib"
)

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
		if appErr.Code == "PROJECT_NOT_FOUND" ||
			appErr.Code == "JUDGMENT_TRACE_NOT_FOUND" ||
			appErr.Code == "HEURISTIC_PROPOSAL_NOT_FOUND" ||
			appErr.Code == "APPROVED_HEURISTIC_NOT_FOUND" ||
			appErr.Code == "PACKET_SNAPSHOT_NOT_FOUND" {
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
		if appErr.Code == "PROPOSAL_ALREADY_RESOLVED" {
			status = http.StatusConflict
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
