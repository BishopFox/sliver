package fantasy

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/charmbracelet/x/exp/slice"
)

// Error is a custom error type for the fantasy package.
type Error struct {
	Message string
	Title   string
	Cause   error
}

func (err *Error) Error() string {
	if err.Title == "" {
		return err.Message
	}
	return fmt.Sprintf("%s: %s", err.Title, err.Message)
}

func (err Error) Unwrap() error {
	return err.Cause
}

// ProviderError represents an error returned by an external provider.
type ProviderError struct {
	Message string
	Title   string
	Cause   error

	URL             string
	StatusCode      int
	RequestBody     []byte
	ResponseHeaders map[string]string
	ResponseBody    []byte
}

func (m *ProviderError) Error() string {
	if m.Title == "" {
		return m.Message
	}
	return fmt.Sprintf("%s: %s", m.Title, m.Message)
}

// IsRetryable checks if the error is retryable based on the status code.
func (m *ProviderError) IsRetryable() bool {
	return m.StatusCode == http.StatusRequestTimeout || m.StatusCode == http.StatusConflict || m.StatusCode == http.StatusTooManyRequests
}

// RetryError represents an error that occurred during retry operations.
type RetryError struct {
	Errors []error
}

func (e *RetryError) Error() string {
	if err, ok := slice.Last(e.Errors); ok {
		return fmt.Sprintf("retry error: %v", err)
	}
	return "retry error: no underlying errors"
}

func (e RetryError) Unwrap() error {
	if err, ok := slice.Last(e.Errors); ok {
		return err
	}
	return nil
}

// ErrorTitleForStatusCode returns a human-readable title for a given HTTP status code.
func ErrorTitleForStatusCode(statusCode int) string {
	return strings.ToLower(http.StatusText(statusCode))
}

// NoObjectGeneratedError is returned when object generation fails
// due to parsing errors, validation errors, or model failures.
type NoObjectGeneratedError struct {
	RawText         string
	ParseError      error
	ValidationError error
	Usage           Usage
	FinishReason    FinishReason
}

// Error implements the error interface.
func (e *NoObjectGeneratedError) Error() string {
	if e.ValidationError != nil {
		return fmt.Sprintf("object validation failed: %v", e.ValidationError)
	}
	if e.ParseError != nil {
		return fmt.Sprintf("failed to parse object: %v", e.ParseError)
	}
	return "failed to generate object"
}

// IsNoObjectGeneratedError checks if an error is of type NoObjectGeneratedError.
func IsNoObjectGeneratedError(err error) bool {
	var target *NoObjectGeneratedError
	return errors.As(err, &target)
}
