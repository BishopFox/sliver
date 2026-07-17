package slack

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
)

const (
	// APIURL of the slack api.
	APIURL = "https://slack.com/api/"
	// AuditAPIURL is the base URL for the Audit Logs API.
	AuditAPIURL = "https://api.slack.com/"
	// WEBAPIURLFormat ...
	WEBAPIURLFormat = "https://%s.slack.com/api/users.admin.%s?t=%d"
)

// httpClient defines the minimal interface needed for an http.Client to be implemented.
type httpClient interface {
	Do(*http.Request) (*http.Response, error)
}

// ResponseMetadata holds pagination metadata
type ResponseMetadata struct {
	Cursor   string   `json:"next_cursor"`
	Messages []string `json:"messages"`
	Warnings []string `json:"warnings"`
}

func (t *ResponseMetadata) initialize() *ResponseMetadata {
	if t != nil {
		return t
	}

	return &ResponseMetadata{}
}

// AuthTestResponse ...
type AuthTestResponse struct {
	URL    string `json:"url"`
	Team   string `json:"team"`
	User   string `json:"user"`
	TeamID string `json:"team_id"`
	UserID string `json:"user_id"`
	// EnterpriseID is only returned when an enterprise id present
	EnterpriseID string      `json:"enterprise_id,omitempty"`
	BotID        string      `json:"bot_id"`
	Header       http.Header `json:"-"`
}

type authTestResponseFull struct {
	SlackResponse
	AuthTestResponse
	responseHeaders
}

type ParamOption func(*url.Values)

// Client for the slack api.
type Client struct {
	token              string
	appLevelToken      string
	configToken        string
	configRefreshToken string
	endpoint           string
	auditEndpoint      string
	debug              bool
	log                ilogger
	httpclient         httpClient
	onWarning          func(path string, request any, w *Warning)
	onResponseHeaders  func(path string, headers http.Header)
}

// Option defines an option for a Client
type Option func(*Client)

// OptionHTTPClient - provide a custom http client to the slack client.
func OptionHTTPClient(client httpClient) func(*Client) {
	return func(c *Client) {
		c.httpclient = client
	}
}

// OptionDebug enable debugging for the client
func OptionDebug(b bool) func(*Client) {
	return func(c *Client) {
		c.debug = b
	}
}

// OptionLog set logging for client.
func OptionLog(l logger) func(*Client) {
	return func(c *Client) {
		c.log = internalLog{logger: l}
	}
}

// OptionOnWarning sets a callback invoked whenever an API response contains
// warnings. The callback receives the API method path (e.g.
// "conversations.join"), the request payload ([url.Values] for form-encoded
// requests or []byte for JSON requests), and a [Warning] with the warning
// codes and messages.
//
// Example:
//
//	api := slack.New("YOUR_TOKEN",
//		slack.OptionOnWarning(func(path string, request any, w *slack.Warning) {
//			log.Printf("slack warnings for %s: codes=%v warnings=%v", path, w.Codes, w.Warnings)
//		}),
//	)
func OptionOnWarning(fn func(path string, request any, w *Warning)) func(*Client) {
	return func(c *Client) {
		c.onWarning = fn
	}
}

// OptionOnResponseHeaders sets a callback invoked after every API request
// with the API method path and the HTTP response headers. This allows
// accessing headers like X-OAuth-Scopes and X-Ratelimit-* for any request.
func OptionOnResponseHeaders(fn func(path string, headers http.Header)) func(*Client) {
	return func(c *Client) {
		c.onResponseHeaders = fn
	}
}

// OptionAPIURL set the url for the client. only useful for testing.
func OptionAPIURL(u string) func(*Client) {
	return func(c *Client) { c.endpoint = u }
}

// OptionAuditAPIURL set the url for the Audit Logs API. only useful for testing.
func OptionAuditAPIURL(u string) func(*Client) {
	return func(c *Client) { c.auditEndpoint = u }
}

// OptionAppLevelToken sets an app-level token for the client.
func OptionAppLevelToken(token string) func(*Client) {
	return func(c *Client) { c.appLevelToken = token }
}

// OptionConfigToken sets a configuration token for the client.
func OptionConfigToken(token string) func(*Client) {
	return func(c *Client) { c.configToken = token }
}

// OptionConfigRefreshToken sets a configuration refresh token for the client.
func OptionConfigRefreshToken(token string) func(*Client) {
	return func(c *Client) { c.configRefreshToken = token }
}

// OptionRetry enables HTTP retries for rate limit (429) only; 5xx and connection errors are not retried.
// Uses DefaultRetryHandlers. Use OptionRetryConfig with AllBuiltinRetryHandlers for connection + 429.
// If maxRetries is zero or negative, the client is not wrapped (no retries).
// When using a custom HTTP client, pass OptionRetry after OptionHTTPClient so the retry wrapper is applied to it.
func OptionRetry(maxRetries int) func(*Client) {
	return func(c *Client) {
		if maxRetries <= 0 {
			return
		}
		cfg := DefaultRetryConfig()
		cfg.MaxRetries = maxRetries
		cfg.Handlers = DefaultRetryHandlers(cfg)
		c.httpclient = &retryClient{client: c.httpclient, config: cfg, debug: c}
	}
}

// OptionRetryConfig enables HTTP retries with a custom config.
// If config.MaxRetries is 0, the client is not wrapped (no retries).
// If config.Handlers is nil, DefaultRetryHandlers(cfg) is used (429 only).
// When using a custom HTTP client, pass OptionRetryConfig after OptionHTTPClient so the retry wrapper is applied to it.
func OptionRetryConfig(config RetryConfig) func(*Client) {
	return func(c *Client) {
		if config.MaxRetries <= 0 {
			return
		}
		cfg := config
		if cfg.Handlers == nil {
			cfg.Handlers = DefaultRetryHandlers(cfg)
		}
		c.httpclient = &retryClient{client: c.httpclient, config: cfg, debug: c}
	}
}

// New builds a slack client from the provided token and options.
func New(token string, options ...Option) *Client {
	s := &Client{
		token:         token,
		endpoint:      APIURL,
		auditEndpoint: AuditAPIURL,
		httpclient:    &http.Client{},
		log:           log.New(os.Stderr, "slack-go/slack", log.LstdFlags|log.Lshortfile),
	}

	for _, opt := range options {
		opt(s)
	}

	return s
}

// AuthTest tests if the user is able to do authenticated requests or not
func (api *Client) AuthTest() (response *AuthTestResponse, error error) {
	return api.AuthTestContext(context.Background())
}

// AuthTestContext tests if the user is able to do authenticated requests or not with a custom context
func (api *Client) AuthTestContext(ctx context.Context) (response *AuthTestResponse, err error) {
	api.Debugf("Challenging auth...")
	responseFull := &authTestResponseFull{}
	err = api.postMethod(ctx, "auth.test", url.Values{"token": {api.token}}, responseFull)
	if err != nil {
		return nil, err
	}

	responseFull.AuthTestResponse.Header = responseFull.responseHeaders.header
	return &responseFull.AuthTestResponse, responseFull.Err()
}

// Debugf print a formatted debug line.
func (api *Client) Debugf(format string, v ...any) {
	if api.debug {
		api.log.Output(2, fmt.Sprintf(format, v...))
	}
}

// Debugln print a debug line.
func (api *Client) Debugln(v ...any) {
	if api.debug {
		api.log.Output(2, fmt.Sprintln(v...))
	}
}

// Debug returns if debug is enabled.
func (api *Client) Debug() bool {
	return api.debug
}

// post to a slack web method.
func (api *Client) postMethod(ctx context.Context, path string, values url.Values, intf any) error {
	headers, err := postForm(ctx, api.httpclient, api.endpoint+path, values, intf, api)
	api.checkWarnings(intf, path, values)
	api.fireResponseHeaders(path, headers)
	return err
}

// get a slack web method.
func (api *Client) getMethod(ctx context.Context, path string, token string, values url.Values, intf any) error {
	headers, err := getResource(ctx, api.httpclient, api.endpoint+path, token, values, intf, api)
	api.checkWarnings(intf, path, values)
	api.fireResponseHeaders(path, headers)
	return err
}

// postJSONMethod posts JSON to a slack web method.
func (api *Client) postJSONMethod(ctx context.Context, path string, token string, jsonBody []byte, intf any) error {
	headers, err := postJSON(ctx, api.httpclient, api.endpoint+path, token, jsonBody, intf, api)
	api.checkWarnings(intf, path, jsonBody)
	api.fireResponseHeaders(path, headers)
	return err
}

func (api *Client) checkWarnings(intf any, path string, request any) {
	if api.onWarning == nil {
		return
	}
	if w, ok := intf.(warner); ok {
		if warning := w.Warn(); warning != nil {
			api.onWarning(path, request, warning)
		}
	}
}

func (api *Client) fireResponseHeaders(path string, headers http.Header) {
	if api.onResponseHeaders != nil && headers != nil {
		api.onResponseHeaders(path, headers)
	}
}
