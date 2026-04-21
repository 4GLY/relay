package contracts

type SuccessEnvelope struct {
	OK       bool     `json:"ok"`
	Command  string   `json:"command"`
	Data     any      `json:"data"`
	Warnings []string `json:"warnings"`
}

type ErrorEnvelope struct {
	OK      bool         `json:"ok"`
	Command string       `json:"command"`
	Error   ErrorPayload `json:"error"`
}

type ErrorPayload struct {
	Code          string   `json:"code"`
	Message       string   `json:"message"`
	Retryable     bool     `json:"retryable"`
	MissingFields []string `json:"missing_fields,omitempty"`
}

func Success(command string, data any) SuccessEnvelope {
	return SuccessEnvelope{
		OK:       true,
		Command:  command,
		Data:     data,
		Warnings: []string{},
	}
}

func Failure(command string, code string, message string, retryable bool, missingFields ...string) ErrorEnvelope {
	return ErrorEnvelope{
		OK:      false,
		Command: command,
		Error: ErrorPayload{
			Code:          code,
			Message:       message,
			Retryable:     retryable,
			MissingFields: missingFields,
		},
	}
}
