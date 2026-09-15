package mailgun

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

const rateLimitResetHeader = "X-RateLimit-Reset"

// UnexpectedResponseError this error will be returned whenever a Mailgun API returns an error response.
// Your application can check the Actual field to see the actual HTTP response code returned.
// URL contains the base URL accessed, sans any query parameters.
type UnexpectedResponseError struct {
	Expected []int
	Actual   int
	Method   string
	URL      string
	Data     []byte
	Header   http.Header
}

// String() converts the error into a human-readable, logfmt-compliant string.
// See http://godoc.org/github.com/kr/logfmt for details on logfmt formatting.
//
// TODO(v6): this method is redundant: move into Error().
// TODO(v6): this returns `ExpectedOneOf=[]int{200, 201}` but should return `ExpectedOneOf=[200 201]`.
//
// Deprecated: use Error() instead.
func (e *UnexpectedResponseError) String() string {
	return fmt.Sprintf("UnexpectedResponseError Method=%s URL=%s ExpectedOneOf=%#v Got=%d Error: %s",
		e.Method, e.URL, e.Expected, e.Actual, string(e.Data))
}

// Error() performs as String().
func (e *UnexpectedResponseError) Error() string {
	return e.String()
}

type RateLimitedError struct {
	Err     error
	ResetAt *time.Time // If provided, the time at which the rate limit will reset.
}

func (e *RateLimitedError) Error() string {
	return fmt.Sprintf("RateLimitedError: %v", e.Err)
}

func (e *RateLimitedError) Unwrap() error {
	return e.Err
}

// newError creates a new error condition to be returned.
func newError(method, url string, expected []int, got *httpResponse) error {
	apiErr := &UnexpectedResponseError{
		Expected: expected,
		Actual:   got.Code,
		Method:   method,
		URL:      url,
		Data:     got.Data,
		Header:   got.Header,
	}

	if apiErr.Actual == http.StatusTooManyRequests {
		ret := &RateLimitedError{Err: apiErr}

		if reset := got.Header.Get(rateLimitResetHeader); reset != "" {
			f, err := strconv.ParseFloat(reset, 64)
			if err == nil {
				t := time.Unix(int64(f), 0).UTC()
				ret.ResetAt = &t
			}
		}

		return ret
	}

	return apiErr
}

// GetStatusFromErr extracts the http status code from error object
func GetStatusFromErr(err error) int {
	var obj *UnexpectedResponseError
	if errors.As(err, &obj) {
		return obj.Actual
	}

	return -1
}
