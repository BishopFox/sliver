package mcp

import (
	"errors"
	"fmt"
)

// Sentinel errors for common JSON-RPC error codes.
var (
	// ErrParseError indicates a JSON parsing error (code: PARSE_ERROR).
	ErrParseError = errors.New("parse error")

	// ErrInvalidRequest indicates an invalid JSON-RPC request (code: INVALID_REQUEST).
	ErrInvalidRequest = errors.New("invalid request")

	// ErrMethodNotFound indicates the requested method does not exist (code: METHOD_NOT_FOUND).
	ErrMethodNotFound = errors.New("method not found")

	// ErrInvalidParams indicates invalid method parameters (code: INVALID_PARAMS).
	ErrInvalidParams = errors.New("invalid params")

	// ErrInternalError indicates an internal JSON-RPC error (code: INTERNAL_ERROR).
	ErrInternalError = errors.New("internal error")

	// ErrRequestInterrupted indicates a request was cancelled or timed out (code: REQUEST_INTERRUPTED).
	ErrRequestInterrupted = errors.New("request interrupted")

	// ErrResourceNotFound indicates a requested resource was not found (code: RESOURCE_NOT_FOUND).
	ErrResourceNotFound = errors.New("resource not found")
)

// UnsupportedProtocolVersionError is returned when the server responds with
// a protocol version that the client doesn't support.
type UnsupportedProtocolVersionError struct {
	Version string
}

func (e UnsupportedProtocolVersionError) Error() string {
	return fmt.Sprintf("unsupported protocol version: %q", e.Version)
}

// Is implements the errors.Is interface for better error handling
func (e UnsupportedProtocolVersionError) Is(target error) bool {
	_, ok := target.(UnsupportedProtocolVersionError)
	return ok
}

// IsUnsupportedProtocolVersion checks if an error is an UnsupportedProtocolVersionError
func IsUnsupportedProtocolVersion(err error) bool {
	_, ok := err.(UnsupportedProtocolVersionError)
	return ok
}

// AsError maps JSONRPCErrorDetails to a Go error.
// Returns sentinel errors wrapped with custom messages for known codes.
// Defaults to a generic error with the original message when the code is not mapped.
func (e *JSONRPCErrorDetails) AsError() error {
	var err error

	switch e.Code {
	case PARSE_ERROR:
		err = ErrParseError
	case INVALID_REQUEST:
		err = ErrInvalidRequest
	case METHOD_NOT_FOUND:
		err = ErrMethodNotFound
	case INVALID_PARAMS:
		err = ErrInvalidParams
	case INTERNAL_ERROR:
		err = ErrInternalError
	case REQUEST_INTERRUPTED:
		err = ErrRequestInterrupted
	case RESOURCE_NOT_FOUND:
		err = ErrResourceNotFound
	default:
		return errors.New(e.Message)
	}

	// Wrap the sentinel error with the custom message if it differs from the sentinel.
	if e.Message != "" && e.Message != err.Error() {
		return fmt.Errorf("%w: %s", err, e.Message)
	}

	return err
}
