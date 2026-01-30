// Copyright 2017 Google Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package internal

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"net/http"
	"runtime"
	"strconv"
	"strings"
	"time"

	"google.golang.org/api/option"
	"google.golang.org/api/transport"
)

// HTTPClient is a convenient API to make HTTP calls.
//
// This API handles repetitive tasks such as entity serialization and deserialization
// when making HTTP calls. It provides a convenient mechanism to set headers and query
// parameters on outgoing requests, while enforcing that an explicit context is used per request.
// Responses returned by HTTPClient can be easily unmarshalled as JSON.
//
// HTTPClient also handles automatically retrying failed HTTP requests.
type HTTPClient struct {
	Client      *http.Client
	RetryConfig *RetryConfig
	CreateErrFn CreateErrFn
	SuccessFn   SuccessFn
	Opts        []HTTPOption
}

// SuccessFn is a function that checks if a Response indicates success.
type SuccessFn func(r *Response) bool

// CreateErrFn is a function that creates an error from a given Response.
type CreateErrFn func(r *Response) error

// NewHTTPClient creates a new HTTPClient using the provided client options and the default
// RetryConfig.
//
// NewHTTPClient returns the created HTTPClient along with the target endpoint URL. The endpoint
// is obtained from the client options passed into the function.
func NewHTTPClient(ctx context.Context, opts ...option.ClientOption) (*HTTPClient, string, error) {
	hc, endpoint, err := transport.NewHTTPClient(ctx, opts...)
	if err != nil {
		return nil, "", err
	}

	return WithDefaultRetryConfig(hc), endpoint, nil
}

// WithDefaultRetryConfig creates a new HTTPClient using the provided client and the default
// RetryConfig.
//
// The default RetryConfig retries requests on all low-level network errors as well as on HTTP
// ServiceUnavailable (503) error. Repeatedly failing requests are retried up to 4 times
// with exponential backoff. Retry delay is never longer than 2 minutes.
func WithDefaultRetryConfig(hc *http.Client) *HTTPClient {
	twoMinutes := time.Duration(2) * time.Minute
	return &HTTPClient{
		Client: hc,
		RetryConfig: &RetryConfig{
			MaxRetries: 4,
			CheckForRetry: retryNetworkAndHTTPErrors(
				http.StatusServiceUnavailable,
			),
			ExpBackoffFactor: 0.5,
			MaxDelay:         &twoMinutes,
		},
	}
}

// Request contains all the parameters required to construct an outgoing HTTP request.
type Request struct {
	Method      string
	URL         string
	Body        HTTPEntity
	Opts        []HTTPOption
	SuccessFn   SuccessFn
	CreateErrFn CreateErrFn
}

// Response contains information extracted from an HTTP response.
type Response struct {
	Status int
	Header http.Header
	Body   []byte
	resp   *http.Response
}

// LowLevelResponse returns an http.Response that represents the underlying low-level HTTP
// response.
//
// This always returns a buffered copy of the original HTTP response. Body can be read from the
// returned response with no impact on the underlying HTTP connection. Closing the Body on the
// returned response is a No-op.
func (r *Response) LowLevelResponse() *http.Response {
	// If the Response instance was initialized manually (as is the case when parsing batch
	// responses) the resp field may be nil.
	if r.resp == nil {
		return nil
	}

	resp := *r.resp
	resp.Body = ioutil.NopCloser(bytes.NewBuffer(r.Body))
	return &resp
}

// Do executes the given Request, and returns a Response.
//
// If a RetryConfig is specified on the client, Do attempts to retry failing requests.
//
// If SuccessFn is set on the client or on the request, the response is validated against that
// function. If this validation fails, returns an error. These errors are created using the
// CreateErrFn on the client or on the request. If neither is set, CreatePlatformError is
// used as the default error function.
func (c *HTTPClient) Do(ctx context.Context, req *Request) (*Response, error) {
	var result *attemptResult

	for retries := 0; ; retries++ {
		hr, err := req.buildHTTPRequest(c.Opts)
		if err != nil {
			return nil, err
		}

		result = c.attempt(ctx, hr, retries)
		if !result.Retry {
			break
		}

		if err = result.waitForRetry(ctx); err != nil {
			return nil, err
		}
	}

	return c.handleResult(req, result)
}

// DoAndUnmarshal behaves similar to Do, but additionally unmarshals the response payload into
// the given pointer.
//
// Unmarshal takes place only if the response does not represent an error (as determined by
// the Do function) and v is not nil. If the unmarshal fails, an error is returned even if the
// original response indicated success.
func (c *HTTPClient) DoAndUnmarshal(ctx context.Context, req *Request, v interface{}) (*Response, error) {
	resp, err := c.Do(ctx, req)
	if err != nil {
		return nil, err
	}

	if v != nil {
		if err := json.Unmarshal(resp.Body, v); err != nil {
			return nil, fmt.Errorf("error while parsing response: %v", err)
		}
	}

	return resp, nil
}

func (c *HTTPClient) attempt(ctx context.Context, hr *http.Request, retries int) *attemptResult {
	resp, err := c.Client.Do(hr.WithContext(ctx))
	result := &attemptResult{}
	if err != nil {
		result.Err = err
	} else {
		// Read the response body here forcing any I/O errors to occur so that retry logic will
		// cover them as well.
		ir, err := newResponse(resp)
		result.Resp = ir
		result.Err = err
	}

	// If a RetryConfig is available, always consult it to determine if the request should be retried
	// or not. Even if there was a network error, we may not want to retry the request based on the
	// RetryConfig that is in effect.
	if c.RetryConfig != nil {
		delay, retry := c.RetryConfig.retryDelay(retries, resp, result.Err)
		result.RetryAfter = delay
		result.Retry = retry
	}

	return result
}

func (c *HTTPClient) handleResult(req *Request, result *attemptResult) (*Response, error) {
	if result.Err != nil {
		return nil, newFirebaseErrorTransport(result.Err)
	}

	if !c.success(req, result.Resp) {
		return nil, c.newError(req, result.Resp)
	}

	return result.Resp, nil
}

func (c *HTTPClient) success(req *Request, resp *Response) bool {
	var successFn SuccessFn
	if req.SuccessFn != nil {
		successFn = req.SuccessFn
	} else if c.SuccessFn != nil {
		successFn = c.SuccessFn
	} else {
		successFn = HasSuccessStatus
	}

	return successFn(resp)
}

func (c *HTTPClient) newError(req *Request, resp *Response) error {
	createErr := func(r *Response) error {
		return NewFirebaseErrorOnePlatform(r)
	}

	if req.CreateErrFn != nil {
		createErr = req.CreateErrFn
	} else if c.CreateErrFn != nil {
		createErr = c.CreateErrFn
	}

	return createErr(resp)
}

type attemptResult struct {
	Resp       *Response
	Err        error
	Retry      bool
	RetryAfter time.Duration
}

func (r *attemptResult) waitForRetry(ctx context.Context) error {
	if r.RetryAfter > 0 {
		select {
		case <-ctx.Done():
		case <-time.After(r.RetryAfter):
		}
	}
	return ctx.Err()
}

func (r *Request) buildHTTPRequest(opts []HTTPOption) (*http.Request, error) {
	var data io.Reader
	if r.Body != nil {
		b, err := r.Body.Bytes()
		if err != nil {
			return nil, err
		}
		data = bytes.NewBuffer(b)
		opts = append(opts, WithHeader("Content-Type", r.Body.Mime()))
	}

	req, err := http.NewRequest(r.Method, r.URL, data)
	if err != nil {
		return nil, err
	}

	opts = append(opts, r.Opts...)
	for _, o := range opts {
		o(req)
	}
	return req, nil
}

// HTTPEntity represents a payload that can be included in an outgoing HTTP request.
type HTTPEntity interface {
	Bytes() ([]byte, error)
	Mime() string
}

type jsonEntity struct {
	Val interface{}
}

// NewJSONEntity creates a new HTTPEntity that will be serialized into JSON.
func NewJSONEntity(v interface{}) HTTPEntity {
	return &jsonEntity{Val: v}
}

func (e *jsonEntity) Bytes() ([]byte, error) {
	return json.Marshal(e.Val)
}

func (e *jsonEntity) Mime() string {
	return "application/json"
}

func newResponse(resp *http.Response) (*Response, error) {
	defer resp.Body.Close()
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return &Response{
		Status: resp.StatusCode,
		Body:   b,
		Header: resp.Header,
		resp:   resp,
	}, nil
}

// HTTPOption is an additional parameter that can be specified to customize an outgoing request.
type HTTPOption func(*http.Request)

// WithHeader creates an HTTPOption that will set an HTTP header on the request.
func WithHeader(key, value string) HTTPOption {
	return func(r *http.Request) {
		r.Header.Set(key, value)
	}
}

// WithQueryParam creates an HTTPOption that will set a query parameter on the request.
func WithQueryParam(key, value string) HTTPOption {
	return func(r *http.Request) {
		q := r.URL.Query()
		q.Add(key, value)
		r.URL.RawQuery = q.Encode()
	}
}

// WithQueryParams creates an HTTPOption that will set all the entries of qp as query parameters
// on the request.
func WithQueryParams(qp map[string]string) HTTPOption {
	return func(r *http.Request) {
		q := r.URL.Query()
		for k, v := range qp {
			q.Add(k, v)
		}
		r.URL.RawQuery = q.Encode()
	}
}

// HasSuccessStatus returns true if the response status code is in the 2xx range.
func HasSuccessStatus(r *Response) bool {
	return r.Status >= http.StatusOK && r.Status < http.StatusNotModified
}

// RetryConfig specifies how the HTTPClient should retry failing HTTP requests.
//
// A request is never retried more than MaxRetries times. If CheckForRetry is nil, all network
// errors, and all 400+ HTTP status codes are retried. If an HTTP error response contains the
// Retry-After header, it is always respected. Otherwise retries are delayed with exponential
// backoff. Set ExpBackoffFactor to 0 to disable exponential backoff, and retry immediately
// after each error.
//
// If MaxDelay is set, retries delay gets capped by that value. If the Retry-After header
// requires a longer delay than MaxDelay, retries are not attempted.
type RetryConfig struct {
	MaxRetries       int
	CheckForRetry    RetryCondition
	ExpBackoffFactor float64
	MaxDelay         *time.Duration
}

// RetryCondition determines if an HTTP request should be retried depending on its last outcome.
type RetryCondition func(resp *http.Response, networkErr error) bool

func (rc *RetryConfig) retryDelay(retries int, resp *http.Response, err error) (time.Duration, bool) {
	if !rc.retryEligible(retries, resp, err) {
		return 0, false
	}
	estimatedDelay := rc.estimateDelayBeforeNextRetry(retries)
	serverRecommendedDelay := parseRetryAfterHeader(resp)
	if serverRecommendedDelay > estimatedDelay {
		estimatedDelay = serverRecommendedDelay
	}
	if rc.MaxDelay != nil && estimatedDelay > *rc.MaxDelay {
		return 0, false
	}
	return estimatedDelay, true
}

func (rc *RetryConfig) retryEligible(retries int, resp *http.Response, err error) bool {
	if retries >= rc.MaxRetries {
		return false
	}
	if rc.CheckForRetry == nil {
		return err != nil || resp.StatusCode >= 500
	}
	return rc.CheckForRetry(resp, err)
}

func (rc *RetryConfig) estimateDelayBeforeNextRetry(retries int) time.Duration {
	if retries == 0 {
		return 0
	}
	delayInSeconds := int64(math.Pow(2, float64(retries)) * rc.ExpBackoffFactor)
	estimatedDelay := time.Duration(delayInSeconds) * time.Second
	if rc.MaxDelay != nil && estimatedDelay > *rc.MaxDelay {
		estimatedDelay = *rc.MaxDelay
	}
	return estimatedDelay
}

var retryTimeClock Clock = SystemClock

func parseRetryAfterHeader(resp *http.Response) time.Duration {
	if resp == nil {
		return 0
	}
	retryAfterHeader := resp.Header.Get("retry-after")
	if retryAfterHeader == "" {
		return 0
	}
	if delayInSeconds, err := strconv.ParseInt(retryAfterHeader, 10, 64); err == nil {
		return time.Duration(delayInSeconds) * time.Second
	}
	if timestamp, err := http.ParseTime(retryAfterHeader); err == nil {
		return timestamp.Sub(retryTimeClock.Now())
	}
	return 0
}

func retryNetworkAndHTTPErrors(statusCodes ...int) RetryCondition {
	return func(resp *http.Response, networkErr error) bool {
		if networkErr != nil {
			return true
		}
		for _, retryOnStatus := range statusCodes {
			if resp.StatusCode == retryOnStatus {
				return true
			}
		}
		return false
	}
}

// GetMetricsHeader constructs header value for metrics attribution
func GetMetricsHeader(sdkVersion string) string {
	goVersion := strings.TrimPrefix(runtime.Version(), "go")
	return fmt.Sprintf("gl-go/%s fire-admin/%s", goVersion, sdkVersion)
}
