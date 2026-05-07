package reddit

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// APIError is an error coming from Reddit.
type APIError struct {
	Label  string
	Reason string
	Field  string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("field %q caused %s: %s", e.Field, e.Label, e.Reason)
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (e *APIError) UnmarshalJSON(data []byte) error {
	var info [3]string

	err := json.Unmarshal(data, &info)
	if err != nil {
		return err
	}

	e.Label = info[0]
	e.Reason = info[1]
	e.Field = info[2]

	return nil
}

// JSONErrorResponse is an error response that sometimes gets returned with a 200 code.
type JSONErrorResponse struct {
	// HTTP response that caused this error.
	Response *http.Response `json:"-"`

	JSON struct {
		Errors []APIError `json:"errors,omitempty"`
	} `json:"json"`
}

func (r *JSONErrorResponse) Error() string {
	errorMessages := make([]string, len(r.JSON.Errors))
	for i, err := range r.JSON.Errors {
		errorMessages[i] = err.Error()
	}

	return fmt.Sprintf(
		"%s %s: %d %s",
		r.Response.Request.Method, r.Response.Request.URL, r.Response.StatusCode, strings.Join(errorMessages, ";"),
	)
}

// An ErrorResponse reports the error caused by an API request
type ErrorResponse struct {
	// HTTP response that caused this error
	Response *http.Response `json:"-"`

	// Error message
	Message string `json:"message"`
}

func (r *ErrorResponse) Error() string {
	return fmt.Sprintf(
		"%s %s: %d %s",
		r.Response.Request.Method, r.Response.Request.URL, r.Response.StatusCode, r.Message,
	)
}

// RateLimitError occurs when the client is sending too many requests to Reddit in a given time frame.
type RateLimitError struct {
	// Rate specifies the last known rate limit for the client
	Rate Rate
	// HTTP response that caused this error
	Response *http.Response
	// Error message
	Message string
}

func (e *RateLimitError) Error() string {
	return fmt.Sprintf(
		"%s %s: %d %s %s",
		e.Response.Request.Method, e.Response.Request.URL, e.Response.StatusCode, e.Message, e.formateRateReset(),
	)
}

func (e *RateLimitError) formateRateReset() string {
	d := time.Until(e.Rate.Reset).Round(time.Second)

	isNegative := d < 0
	if isNegative {
		d *= -1
	}

	if isNegative {
		return fmt.Sprintf("[rate limit was reset %s ago]", d)
	}
	return fmt.Sprintf("[rate limit will reset in %s]", d)
}
