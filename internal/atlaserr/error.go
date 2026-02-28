// Package atlaserr defines structured CLI errors.
package atlaserr

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
)

const emptyDetails = "{}"

// Code is a stable machine error code.
type Code string

const (
	// CodeInvalidArgument indicates invalid command input.
	CodeInvalidArgument Code = "INVALID_ARGUMENT"
	// CodeAuthFailed indicates authentication failure.
	CodeAuthFailed Code = "AUTH_FAILED"
	// CodeForbidden indicates authorization failure.
	CodeForbidden Code = "FORBIDDEN"
	// CodeNotFound indicates missing upstream resource.
	CodeNotFound Code = "NOT_FOUND"
	// CodeRateLimited indicates upstream rate limiting.
	CodeRateLimited Code = "RATE_LIMITED"
	// CodeUpstreamError indicates uncategorized upstream non-2xx responses.
	CodeUpstreamError Code = "UPSTREAM_ERROR"
	// CodeNetworkError indicates transport failures.
	CodeNetworkError Code = "NETWORK_ERROR"
)

// Error is a structured CLI error value.
type Error struct {
	Code      Code
	Message   string
	Retryable bool
	Details   json.RawMessage
}

type errorBody struct {
	Code      Code            `json:"code"`
	Message   string          `json:"message"`
	Retryable bool            `json:"retryable"`
	Details   json.RawMessage `json:"details"`
}

// ErrorEnvelope is the serialized error wrapper.
type ErrorEnvelope struct {
	Error errorBody `json:"error"`
}

// New builds a structured error.
func New(code Code, message string, retryable bool, details json.RawMessage) *Error {
	return &Error{
		Code:      code,
		Message:   message,
		Retryable: retryable,
		Details:   normalizeDetails(details),
	}
}

// InvalidArgument builds an invalid argument error.
func InvalidArgument(message string) *Error {
	return New(CodeInvalidArgument, message, false, nil)
}

// Network builds a network error.
func Network(err error) *Error {
	return New(CodeNetworkError, err.Error(), true, nil)
}

// FromHTTPStatus maps an HTTP status to a structured error.
func FromHTTPStatus(statusCode int, headers http.Header) *Error {
	message := fmt.Sprintf("upstream returned %d", statusCode)

	switch statusCode {
	case http.StatusUnauthorized:
		return New(CodeAuthFailed, message, false, nil)
	case http.StatusForbidden:
		return New(CodeForbidden, message, false, nil)
	case http.StatusNotFound:
		return New(CodeNotFound, message, false, nil)
	case http.StatusTooManyRequests:
		return New(CodeRateLimited, message, true, retryAfterDetails(headers))
	default:
		retryable := statusCode >= http.StatusInternalServerError
		return New(CodeUpstreamError, message, retryable, nil)
	}
}

// Envelope returns the JSON-serializable envelope.
func (e *Error) Envelope() ErrorEnvelope {
	return ErrorEnvelope{
		Error: errorBody{
			Code:      e.Code,
			Message:   e.Message,
			Retryable: e.Retryable,
			Details:   normalizeDetails(e.Details),
		},
	}
}

// Error returns a compact text representation.
func (e *Error) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func normalizeDetails(details json.RawMessage) json.RawMessage {
	if len(details) == 0 {
		return json.RawMessage(emptyDetails)
	}

	return details
}

func retryAfterDetails(headers http.Header) json.RawMessage {
	retryAfterRaw := headers.Get("Retry-After")
	if retryAfterRaw == "" {
		return json.RawMessage(emptyDetails)
	}

	retryAfterSeconds, err := strconv.Atoi(retryAfterRaw)
	if err != nil {
		return json.RawMessage(emptyDetails)
	}

	payload := struct {
		RetryAfterSeconds int `json:"retryAfterSeconds"`
	}{
		RetryAfterSeconds: retryAfterSeconds,
	}

	details, marshalErr := json.Marshal(payload)
	if marshalErr != nil {
		return json.RawMessage(emptyDetails)
	}

	return details
}
