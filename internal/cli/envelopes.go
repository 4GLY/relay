package cli

import (
	"relay/internal/contracts"
	"relay/internal/lib"
)

func notImplementedEnvelope(command string, input map[string]any) contracts.ErrorEnvelope {
	return contracts.Failure(
		command,
		"NOT_IMPLEMENTED",
		"command scaffold exists but handler is not implemented yet",
		false,
	)
}

func failureEnvelope(command string, err error) contracts.ErrorEnvelope {
	if appErr, ok := err.(lib.AppError); ok {
		return contracts.Failure(command, appErr.Code, appErr.Message, appErr.Retryable, appErr.MissingFields...)
	}
	return contracts.Failure(command, "INTERNAL_ERROR", err.Error(), true)
}
