package mcp

import (
	"encoding/json"
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

// URLElicitationRequiredError is returned when the server requires URL elicitation to proceed.
type URLElicitationRequiredError struct {
	Elicitations []ElicitationParams `json:"elicitations"`
}

func (e URLElicitationRequiredError) Error() string {
	return fmt.Sprintf("URL elicitation required: %d elicitation(s) needed", len(e.Elicitations))
}

func (e URLElicitationRequiredError) JSONRPCError() JSONRPCError {
	return JSONRPCError{
		JSONRPC: JSONRPC_VERSION,
		Error: JSONRPCErrorDetails{
			Code:    URL_ELICITATION_REQUIRED,
			Message: e.Error(),
			Data: map[string]any{
				"elicitations": e.Elicitations,
			},
		},
	}
}

// UnsupportedProtocolVersionError is returned when the server responds with
// a protocol version that the client doesn't support.
type UnsupportedProtocolVersionError struct {
	Version string
}

func (e UnsupportedProtocolVersionError) Error() string {
	return fmt.Sprintf("unsupported protocol version: %q", e.Version)
}

// Is implements the errors.Is interface for better error handling
func (e URLElicitationRequiredError) Is(target error) bool {
	_, ok := target.(URLElicitationRequiredError)
	return ok
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
	case URL_ELICITATION_REQUIRED:
		// Attempt to reconstruct URLElicitationRequiredError from Data
		if e.Data != nil {
			// Round-trip through JSON to parse into struct
			// This handles both map[string]any (from unmarshal) and other forms
			if dataBytes, marshalErr := json.Marshal(e.Data); marshalErr == nil {
				var data struct {
					Elicitations []ElicitationParams `json:"elicitations"`
				}
				if unmarshalErr := json.Unmarshal(dataBytes, &data); unmarshalErr == nil {
					return URLElicitationRequiredError{
						Elicitations: data.Elicitations,
					}
				}
			}
		}
		// Fallback if data is missing or invalid
		return URLElicitationRequiredError{}
	default:
		return errors.New(e.Message)
	}

	// Wrap the sentinel error with the custom message if it differs from the sentinel.
	if e.Message != "" && e.Message != err.Error() {
		return fmt.Errorf("%w: %s", err, e.Message)
	}

	return err
}
