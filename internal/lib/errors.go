package lib

import "fmt"

type AppError struct {
	Code          string
	Message       string
	Retryable     bool
	MissingFields []string
}

func (e AppError) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func MissingFields(code string, fields ...string) AppError {
	return AppError{
		Code:          code,
		Message:       "missing required fields",
		Retryable:     false,
		MissingFields: fields,
	}
}

func Misconfigured(message string) AppError {
	return AppError{
		Code:      "MISCONFIGURED",
		Message:   message,
		Retryable: false,
	}
}

func NotFound(code string, message string) AppError {
	return AppError{
		Code:      code,
		Message:   message,
		Retryable: false,
	}
}
