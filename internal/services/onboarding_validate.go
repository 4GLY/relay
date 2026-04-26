package services

import (
	"context"
	"net/http"
	"time"

	"relay/internal/lib"
)

const (
	anthropicValidateURL = "https://api.anthropic.com/v1/models"
	anthropicTimeout     = 8 * time.Second
)

// validateHTTPClient is a package-level client so the per-request
// http.Client.Timeout is honored uniformly. Tests can swap it via
// SetValidateHTTPClient.
var validateHTTPClient = &http.Client{Timeout: anthropicTimeout}

// SetValidateHTTPClient is for tests that need to point validateAnthropicKey
// at an httptest.Server. Callers must restore the previous client.
func SetValidateHTTPClient(c *http.Client) (restore func()) {
	prev := validateHTTPClient
	validateHTTPClient = c
	return func() { validateHTTPClient = prev }
}

// SetValidateURL points the probe at a test server. Callers must restore the
// previous URL.
func SetValidateURL(url string) (restore func()) {
	prev := anthropicValidateURLOverride
	anthropicValidateURLOverride = url
	return func() { anthropicValidateURLOverride = prev }
}

var anthropicValidateURLOverride string

// StepStatus is one of three locked outcomes per validation chip (D2 + E4).
type StepStatus string

const (
	StepOK      StepStatus = "ok"
	StepFailed  StepStatus = "failed"
	StepSkipped StepStatus = "skipped"
)

// ValidationStep is a single chip in the Frame 3 progress payload.
type ValidationStep struct {
	ID     string     `json:"id"`
	Status StepStatus `json:"status"`
}

// validateAnthropicKey performs a single GET /v1/models against api.anthropic.com,
// then derives the two validation chips (anthropic_key, anthropic_quota) from the
// HTTP status code. Locked mapping (E4):
//
//	200     → [{anthropic_key:ok},     {anthropic_quota:ok}]
//	401/403 → [{anthropic_key:failed}, {anthropic_quota:skipped}]  INVALID_ANTHROPIC_KEY (retryable:false)
//	429     → [{anthropic_key:ok},     {anthropic_quota:failed}]   ANTHROPIC_QUOTA      (retryable:false)
//	net/5xx → [{anthropic_key:failed}, {anthropic_quota:skipped}]  ANTHROPIC_UNREACHABLE (retryable:true)
func validateAnthropicKey(ctx context.Context, rawKey string) ([]ValidationStep, error) {
	url := anthropicValidateURL
	if anthropicValidateURLOverride != "" {
		url = anthropicValidateURLOverride
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return validationSteps(StepFailed, StepSkipped), lib.AppError{
			Code:      "ANTHROPIC_UNREACHABLE",
			Message:   "could not build Anthropic probe request",
			Retryable: true,
		}
	}
	req.Header.Set("x-api-key", rawKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := validateHTTPClient.Do(req)
	if err != nil {
		return validationSteps(StepFailed, StepSkipped), lib.AppError{
			Code:      "ANTHROPIC_UNREACHABLE",
			Message:   "could not reach api.anthropic.com",
			Retryable: true,
		}
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		return validationSteps(StepOK, StepOK), nil
	case http.StatusUnauthorized, http.StatusForbidden:
		return validationSteps(StepFailed, StepSkipped), lib.AppError{
			Code:      "INVALID_ANTHROPIC_KEY",
			Message:   "Anthropic API rejected the key",
			Retryable: false,
		}
	case http.StatusTooManyRequests:
		return validationSteps(StepOK, StepFailed), lib.AppError{
			Code:      "ANTHROPIC_QUOTA",
			Message:   "Anthropic API quota exceeded",
			Retryable: false,
		}
	default:
		return validationSteps(StepFailed, StepSkipped), lib.AppError{
			Code:      "ANTHROPIC_UNREACHABLE",
			Message:   "unexpected Anthropic response: " + resp.Status,
			Retryable: true,
		}
	}
}

func validationSteps(keyStatus, quotaStatus StepStatus) []ValidationStep {
	return []ValidationStep{
		{ID: "anthropic_key", Status: keyStatus},
		{ID: "anthropic_quota", Status: quotaStatus},
	}
}
