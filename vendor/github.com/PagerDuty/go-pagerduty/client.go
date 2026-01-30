// Package pagerduty is a Go API client for both the PagerDuty v2 REST and
// Events API. Most methods should be implemented, and it's recommended to use
// the WithContext variant of each method and to specify a context with a
// timeout.
//
// To debug responses from the API, you can instruct the client to capture the
// last response from the API. Please see the documentation for the
// SetDebugFlag() and LastAPIResponse() methods for more details.
package pagerduty

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"path"
	"runtime"
	"strings"
	"sync/atomic"
	"time"
)

// Version is current version of this client.
const Version = "1.8.0"

const (
	apiEndpoint         = "https://api.pagerduty.com"
	v2EventsAPIEndpoint = "https://events.pagerduty.com"
)

// The type of authentication to use with the API client
type authType int

const (
	// Account/user API token authentication
	apiToken authType = iota

	// OAuth token authentication
	oauthToken
)

// APIObject represents generic api json response that is shared by most
// domain objects (like escalation)
type APIObject struct {
	ID      string `json:"id,omitempty"`
	Type    string `json:"type,omitempty"`
	Summary string `json:"summary,omitempty"`
	Self    string `json:"self,omitempty"`
	HTMLURL string `json:"html_url,omitempty"`
}

// APIListObject are the fields used to control pagination when listing objects.
type APIListObject struct {
	Limit  uint `json:"limit,omitempty"`
	Offset uint `json:"offset,omitempty"`
	More   bool `json:"more,omitempty"`
	Total  uint `json:"total,omitempty"`
}

// APIReference are the fields required to reference another API object.
type APIReference struct {
	ID   string `json:"id,omitempty"`
	Type string `json:"type,omitempty"`
}

// APIDetails are the fields required to represent a details non-hydrated
// object.
type APIDetails struct {
	Type    string `json:"type,omitempty"`
	Details string `json:"details,omitempty"`
}

// APIErrorObject represents the object returned by the API when an error
// occurs. This includes messages that should hopefully provide useful context
// to the end user.
type APIErrorObject struct {
	Code    int      `json:"code,omitempty"`
	Message string   `json:"message,omitempty"`
	Errors  []string `json:"errors,omitempty"`
}

func unmarshalApiErrorObject(data []byte) (APIErrorObject, error) {
	var aeo APIErrorObject
	err := json.Unmarshal(data, &aeo)
	if err == nil {
		return aeo, nil
	}
	if _, ok := err.(*json.UnmarshalTypeError); !ok {
		return aeo, nil
	}
	// See - https://github.com/PagerDuty/go-pagerduty/issues/339
	// TODO: remove when PagerDuty engineering confirms bugfix to the REST API
	var fallback1 struct {
		Code    int    `json:"code,omitempty"`
		Message string `json:"message,omitempty"`
		Errors  string `json:"errors,omitempty"`
	}
	if json.Unmarshal(data, &fallback1) == nil {
		aeo.Code = fallback1.Code
		aeo.Message = fallback1.Message
		aeo.Errors = []string{fallback1.Errors}
		return aeo, nil
	}
	// See - https://github.com/PagerDuty/go-pagerduty/issues/478
	var fallback2 []string
	if json.Unmarshal(data, &fallback2) == nil {
		aeo.Message = "none"
		aeo.Errors = fallback2
		return aeo, nil
	}
	// still failed, so return the original error
	return aeo, err
}

// NullAPIErrorObject is a wrapper around the APIErrorObject type. If the Valid
// field is true, the API response included a structured error JSON object. This
// structured object is then set on the ErrorObject field.
//
// While the PagerDuty REST API is documented to always return the error object,
// we assume it's possible in exceptional failure modes for this to be omitted.
// As such, this wrapper type provides us a way to check if the object was
// provided while avoiding consumers accidentally missing a nil pointer check,
// thus crashing their whole program.
type NullAPIErrorObject struct {
	Valid       bool
	ErrorObject APIErrorObject
}

var _ json.Unmarshaler = (*NullAPIErrorObject)(nil) // assert that it satisfies the json.Unmarshaler interface.

// UnmarshalJSON satisfies encoding/json.Unmarshaler
func (n *NullAPIErrorObject) UnmarshalJSON(data []byte) error {
	aeo, err := unmarshalApiErrorObject(data)
	if err != nil {
		return err
	}

	n.ErrorObject = aeo
	n.Valid = true

	return nil
}

// APIError represents the error response received when an API call fails. The
// HTTP response code is set inside of the StatusCode field, with the APIError
// field being the structured JSON error object returned from the API.
//
// This type also provides some helper methods like .RateLimited(), .NotFound(),
// and .Temporary() to help callers reason about how to handle the error.
//
// You can read more about the HTTP status codes and API error codes returned
// from the API here: https://developer.pagerduty.com/docs/rest-api-v2/errors/
type APIError struct {
	// StatusCode is the HTTP response status code
	StatusCode int `json:"-"`

	// APIError represents the object returned by the API when an error occurs,
	// which includes messages that should hopefully provide useful context
	// to the end user.
	//
	// If the API response did not contain an error object, the .Valid field of
	// APIError will be false. If .Valid is true, the .ErrorObject field is
	// valid and should be consulted.
	APIError NullAPIErrorObject `json:"error"`

	message string
}

// Error satisfies the error interface, and should contain the StatusCode,
// APIErrorObject.Message, and APIErrorObject.Code.
func (a APIError) Error() string {
	if len(a.message) > 0 {
		return a.message
	}

	if !a.APIError.Valid {
		return fmt.Sprintf("HTTP response failed with status code %d and no JSON error object was present", a.StatusCode)
	}

	if len(a.APIError.ErrorObject.Errors) == 0 {
		return fmt.Sprintf(
			"HTTP response failed with status code %d, message: %s (code: %d)",
			a.StatusCode, a.APIError.ErrorObject.Message, a.APIError.ErrorObject.Code,
		)
	}

	return fmt.Sprintf(
		"HTTP response failed with status code %d, message: %s (code: %d): %s",
		a.StatusCode,
		a.APIError.ErrorObject.Message,
		a.APIError.ErrorObject.Code,
		apiErrorsDetailString(a.APIError.ErrorObject.Errors),
	)
}

func apiErrorsDetailString(errs []string) string {
	switch n := len(errs); n {
	case 0:
		panic("errs slice is empty")

	case 1:
		return errs[0]

	default:
		e := "error"
		if n > 2 {
			e += "s"
		}

		return fmt.Sprintf("%s (and %d more %s...)", errs[0], n-1, e)
	}
}

// RateLimited returns whether the response had a status of 429, and as such the
// client is rate limited. The PagerDuty rate limits should reset once per
// minute, and for the REST API they are an account-wide rate limit (not per
// API key or IP).
func (a APIError) RateLimited() bool {
	return a.StatusCode == http.StatusTooManyRequests
}

// Temporary returns whether it was a temporary error, one of which is a
// RateLimited error.
func (a APIError) Temporary() bool {
	return a.RateLimited() || (a.StatusCode >= 500 && a.StatusCode < 600)
}

// NotFound returns whether this was an error where it seems like the resource
// was not found.
func (a APIError) NotFound() bool {
	return a.StatusCode == http.StatusNotFound || (a.APIError.Valid && a.APIError.ErrorObject.Code == 2100)
}

func newDefaultHTTPClient() *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			MaxIdleConns:          10,
			IdleConnTimeout:       60 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
			MaxIdleConnsPerHost:   runtime.GOMAXPROCS(0) + 1,
		},
	}
}

// HTTPClient is an interface which declares the functionality we need from an
// HTTP client. This is to allow consumers to provide their own HTTP client as
// needed, without restricting them to only using *http.Client.
type HTTPClient interface {
	Do(*http.Request) (*http.Response, error)
}

// defaultHTTPClient is our own default HTTP client. We use this, instead of
// http.DefaultClient, to avoid other packages tweaks to http.DefaultClient
// causing issues with our HTTP calls. This also allows us to tweak the
// transport values to be more resilient without making changes to the
// http.DefaultClient.
//
// Keep this unexported so consumers of the package can't make changes to it.
var defaultHTTPClient HTTPClient = newDefaultHTTPClient()

// Client wraps http client
type Client struct {
	debugFlag    *uint64
	lastRequest  *atomic.Value
	lastResponse *atomic.Value

	authToken           string
	apiEndpoint         string
	v2EventsAPIEndpoint string

	// Authentication type to use for API
	authType authType

	// HTTPClient is the HTTP client used for making requests against the
	// PagerDuty API. You can use either *http.Client here, or your own
	// implementation.
	HTTPClient HTTPClient

	userAgent string
}

// NewClient creates an API client using an account/user API token
func NewClient(authToken string, options ...ClientOptions) *Client {
	client := Client{
		debugFlag:           new(uint64),
		lastRequest:         &atomic.Value{},
		lastResponse:        &atomic.Value{},
		authToken:           authToken,
		apiEndpoint:         apiEndpoint,
		v2EventsAPIEndpoint: v2EventsAPIEndpoint,
		authType:            apiToken,
		HTTPClient:          defaultHTTPClient,
	}

	for _, opt := range options {
		opt(&client)
	}

	return &client
}

// NewOAuthClient creates an API client using an OAuth token
func NewOAuthClient(authToken string, options ...ClientOptions) *Client {
	return NewClient(authToken, WithOAuth())
}

// ClientOptions allows for options to be passed into the Client for customization
type ClientOptions func(*Client)

// WithAPIEndpoint allows for a custom API endpoint to be passed into the the client
func WithAPIEndpoint(endpoint string) ClientOptions {
	return func(c *Client) {
		c.apiEndpoint = endpoint
	}
}

// WithTerraformProvider configures the client to be used as the PagerDuty
// Terraform provider
func WithTerraformProvider(version string) ClientOptions {
	return func(c *Client) {
		c.userAgent = fmt.Sprintf("(%s %s) Terraform/%s", runtime.GOOS, runtime.GOARCH, version)
	}
}

// WithV2EventsAPIEndpoint allows for a custom V2 Events API endpoint to be passed into the client
func WithV2EventsAPIEndpoint(endpoint string) ClientOptions {
	return func(c *Client) {
		c.v2EventsAPIEndpoint = endpoint
	}
}

// WithOAuth allows for an OAuth token to be passed into the the client
func WithOAuth() ClientOptions {
	return func(c *Client) {
		c.authType = oauthToken
	}
}

// DebugFlag represents a set of debug bit flags that can be bitwise-ORed
// together to configure the different behaviors. This allows us to expand
// functionality in the future without introducing breaking changes.
type DebugFlag uint64

const (
	// DebugDisabled disables all debug behaviors.
	DebugDisabled DebugFlag = 0

	// DebugCaptureLastRequest captures the last HTTP request made to the API
	// (if there was one) and makes it available via the LastAPIRequest()
	// method.
	//
	// This may increase memory usage / GC, as we'll be making a copy of the
	// full HTTP request body on each request and capturing it for inspection.
	DebugCaptureLastRequest DebugFlag = 1 << 0

	// DebugCaptureLastResponse captures the last HTTP response from the API (if
	// there was one) and makes it available via the LastAPIResponse() method.
	//
	// This may increase memory usage / GC, as we'll be making a copy of the
	// full HTTP response body on each request and capturing it for inspection.
	DebugCaptureLastResponse DebugFlag = 1 << 1
)

// SetDebugFlag sets the DebugFlag of the client, which are just bit flags that
// tell the client how to behave. They can be bitwise-ORed together to enable
// multiple behaviors.
func (c *Client) SetDebugFlag(flag DebugFlag) {
	atomic.StoreUint64(c.debugFlag, uint64(flag))
}

func (c *Client) debugCaptureRequest() bool {
	return atomic.LoadUint64(c.debugFlag)&uint64(DebugCaptureLastRequest) > 0
}

func (c *Client) debugCaptureResponse() bool {
	return atomic.LoadUint64(c.debugFlag)&uint64(DebugCaptureLastResponse) > 0
}

// LastAPIRequest returns the last request sent to the API, if enabled. This can
// be turned on by using the SetDebugFlag() method while providing the
// DebugCaptureLastRequest flag.
//
// The bool value returned from this method is false if the request is unset or
// is nil. If there was an error prepping the request to be sent to the server,
// there will be no *http.Request to capture so this will return (<nil>, false).
//
// This is meant to help with debugging unexpected API interactions, so most
// won't need to use it. Also, callers will need to ensure the *Client isn't
// used concurrently, otherwise they might receive another method's *http.Request
// and not the one they anticipated.
//
// The *http.Request made within the Do() method is not captured by the client,
// and thus won't be returned by this method.
func (c *Client) LastAPIRequest() (*http.Request, bool) {
	v := c.lastRequest.Load()
	if v == nil {
		return nil, false
	}

	// comma ok idiom omitted, if this is something else explode
	r := v.(*http.Request)
	if r == nil {
		return nil, false
	}

	return r, true
}

// LastAPIResponse returns the last response received from the API, if enabled.
// This can be turned on by using the SetDebugFlag() method while providing the
// DebugCaptureLastResponse flag.
//
// The bool value returned from this method is false if the response is unset or
// is nil. If the HTTP exchange failed (e.g., there was a connection error)
// there will be no *http.Response to capture so this will return (<nil>,
// false).
//
// This is meant to help with debugging unexpected API interactions, so most
// won't need to use it. Also, callers will need to ensure the *Client isn't
// used concurrently, otherwise they might receive another method's *http.Response
// and not the one they anticipated.
//
// The *http.Response from the Do() method is not captured by the client, and thus
// won't be returned by this method.
func (c *Client) LastAPIResponse() (*http.Response, bool) {
	v := c.lastResponse.Load()
	if v == nil {
		return nil, false
	}

	// comma ok idiom omitted, if this is something else explode
	r := v.(*http.Response)
	if r == nil {
		return nil, false
	}

	return r, true
}

// Do sets some headers on the request, before actioning it using the internal
// HTTPClient. If the PagerDuty API you're communicating with requires
// authentication, such as the REST API, set authRequired to true and the client
// will set the proper authentication headers onto the request. This also
// assumes any request body is in JSON format and sets the Content-Type to
// application/json.
func (c *Client) Do(r *http.Request, authRequired bool) (*http.Response, error) {
	c.prepRequest(r, authRequired, nil)

	return c.HTTPClient.Do(r)
}

func (c *Client) delete(ctx context.Context, path string) (*http.Response, error) {
	return c.do(ctx, http.MethodDelete, path, nil, nil)
}

func (c *Client) put(ctx context.Context, path string, payload interface{}, headers map[string]string) (*http.Response, error) {
	if payload != nil {
		data, err := json.Marshal(payload)
		if err != nil {
			return nil, err
		}
		return c.do(ctx, http.MethodPut, path, bytes.NewBuffer(data), headers)
	}
	return c.do(ctx, http.MethodPut, path, nil, headers)
}

func (c *Client) post(ctx context.Context, path string, payload interface{}, headers map[string]string) (*http.Response, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return c.do(ctx, http.MethodPost, path, bytes.NewBuffer(data), headers)
}

func (c *Client) get(ctx context.Context, path string, headers map[string]string) (*http.Response, error) {
	return c.do(ctx, http.MethodGet, path, nil, headers)
}

const (
	userAgentHeader   = "go-pagerduty/" + Version
	acceptHeader      = "application/vnd.pagerduty+json;version=2"
	contentTypeHeader = "application/json"
)

func (c *Client) prepRequest(req *http.Request, authRequired bool, headers map[string]string) {
	req.Header.Set("Accept", acceptHeader)

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	if authRequired {
		switch c.authType {
		case oauthToken:
			req.Header.Set("Authorization", "Bearer "+c.authToken)
		default:
			req.Header.Set("Authorization", "Token token="+c.authToken)
		}
	}

	var userAgent string
	if c.userAgent != "" {
		userAgent = c.userAgent
	} else {
		userAgent = userAgentHeader
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Content-Type", contentTypeHeader)
}

func dupeRequest(r *http.Request) (*http.Request, error) {
	dreq := r.Clone(r.Context())

	if r.Body != nil {
		data, err := ioutil.ReadAll(r.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to copy request body: %w", err)
		}

		_ = r.Body.Close()

		r.Body = ioutil.NopCloser(bytes.NewReader(data))
		dreq.Body = ioutil.NopCloser(bytes.NewReader(data))
	}

	return dreq, nil
}

// needed where pagerduty use a different endpoint for certain actions (eg: v2 events)
func (c *Client) doWithEndpoint(ctx context.Context, endpoint, method, path string, authRequired bool, body io.Reader, headers map[string]string) (*http.Response, error) {
	var dreq *http.Request
	var resp *http.Response

	// so that the last request and response can be nil if there was an error
	// before the request could be fully processed by the origin, we defer these
	// calls here
	if c.debugCaptureResponse() {
		defer func() {
			c.lastResponse.Store(resp)
		}()
	}

	if c.debugCaptureRequest() {
		defer func() {
			c.lastRequest.Store(dreq)
		}()
	}

	req, err := http.NewRequestWithContext(ctx, method, endpoint+path, body)
	if err != nil {
		return nil, fmt.Errorf("failed to build request: %w", err)
	}

	c.prepRequest(req, authRequired, headers)

	// if in debug mode, copy request before making it
	if c.debugCaptureRequest() {
		if dreq, err = dupeRequest(req); err != nil {
			return nil, fmt.Errorf("failed to duplicate request for debug capture: %w", err)
		}
	}

	resp, err = c.HTTPClient.Do(req)

	return c.checkResponse(resp, err)
}

func (c *Client) do(ctx context.Context, method, path string, body io.Reader, headers map[string]string) (*http.Response, error) {
	return c.doWithEndpoint(ctx, c.apiEndpoint, method, path, true, body, headers)
}

func (c *Client) decodeJSON(resp *http.Response, payload interface{}) error {
	// close the original response body, and not the copy we may make if
	// debugCaptureResponse is true
	orb := resp.Body
	defer func() { _ = orb.Close() }() // explicitly discard error

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	if c.debugCaptureResponse() { // reset body as we capture the response elsewhere
		resp.Body = ioutil.NopCloser(bytes.NewReader(body))
	}

	return json.Unmarshal(body, payload)
}

func (c *Client) checkResponse(resp *http.Response, err error) (*http.Response, error) {
	if err != nil {
		return resp, fmt.Errorf("error calling the API endpoint: %v", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return resp, c.getErrorFromResponse(resp)
	}

	return resp, nil
}

func (c *Client) getErrorFromResponse(resp *http.Response) APIError {
	// check whether the error response is declared as JSON
	if !strings.HasPrefix(resp.Header.Get("Content-Type"), "application/json") {
		defer resp.Body.Close()

		aerr := APIError{
			StatusCode: resp.StatusCode,
			message:    fmt.Sprintf("HTTP response with status code %d does not contain Content-Type: application/json", resp.StatusCode),
		}

		return aerr
	}

	var document APIError

	// because of above check this probably won't fail, but it's possible...
	if err := c.decodeJSON(resp, &document); err != nil {
		aerr := APIError{
			StatusCode: resp.StatusCode,
			message:    fmt.Sprintf("HTTP response with status code %d, JSON error object decode failed: %s", resp.StatusCode, err),
		}

		return aerr
	}

	document.StatusCode = resp.StatusCode

	return document
}

// Helper function to determine wither additional parameters should use ? or & to append args
func getBasePrefix(basePath string) string {
	if strings.Contains(path.Base(basePath), "?") {
		return basePath + "&"
	}
	return basePath + "?"
}

// responseHandler is capable of parsing a response. At a minimum it must
// extract the page information for the current page. It can also execute
// additional necessary handling; for example, if a closure, it has access
// to the scope in which it was defined, and can be used to append data to
// a specific slice. The responseHandler is responsible for closing the response.
type responseHandler func(response *http.Response) (APIListObject, error)

func (c *Client) pagedGet(ctx context.Context, basePath string, handler responseHandler) error {
	// Indicates whether there are still additional pages associated with request.
	var stillMore bool

	// Offset to set for the next page request.
	var nextOffset uint

	basePrefix := getBasePrefix(basePath)
	// While there are more pages, keep adjusting the offset to get all results.
	for stillMore, nextOffset = true, 0; stillMore; {
		response, err := c.do(ctx, http.MethodGet, fmt.Sprintf("%soffset=%d", basePrefix, nextOffset), nil, nil)
		if err != nil {
			return err
		}

		// Call handler to extract page information and execute additional necessary handling.
		pageInfo, err := handler(response)
		if err != nil {
			return err
		}

		// Bump the offset as necessary and set whether more results exist.
		nextOffset = pageInfo.Offset + pageInfo.Limit
		stillMore = pageInfo.More
	}

	return nil
}

type cursor struct {
	// The minimum of the 'limit' parameter used in the request or
	// the maximum request size of the API.
	Limit uint
	// An opaque string used to fetch the next set of results or 'null'
	// if no additional results are available.
	NextCursor string
}

// cursorHandler is capable of parsing a response, using cursor-based pagination.
// At a minimum it must extract the page information for the current page.
type cursorHandler func(r *http.Response) (cursor, error)

func (c *Client) cursorGet(ctx context.Context, basePath string, handler cursorHandler) error {
	var next string

	basePrefix := getBasePrefix(basePath)

	for {
		var cs string
		if len(next) > 0 {
			cs = fmt.Sprintf("cursor=%s", next)
		}

		// The next set of results can be obtained by providing the
		// NextCursor value from the previous request in a cursor
		// query parameter on the subsequent request.
		resp, err := c.do(ctx, http.MethodGet, fmt.Sprintf("%s%s", basePrefix, cs), nil, nil)
		if err != nil {
			return err
		}

		// Call handler to extract page information and execute additional necessary handling.
		c, err := handler(resp)
		if err != nil {
			return err
		}

		// Stop parsing if there are no more results.
		if len(c.NextCursor) == 0 {
			break
		}

		// Otherwise, update next to parse more results.
		next = c.NextCursor
	}

	return nil
}
