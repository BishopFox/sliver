package slack

// Optional HTTP retries improve reliability when Slack is busy or the network is flaky.
// Retries are off by default; use OptionRetry or OptionRetryConfig to turn them on.
//
// Retry behavior is driven by pluggable handlers (parity with the Python SDK:
// https://github.com/slackapi/python-slack-sdk). When Handlers is nil, only rate limit
// (429) is retried (NewRateLimitErrorRetryHandler). Use
// AllBuiltinRetryHandlers(cfg) for connection + 429; ConnectionOnlyRetryHandlers(cfg) for
// connection-only; add NewServerErrorRetryHandler(cfg) to also retry 5xx.
//
// File uploads and other requests that stream the body cannot be retried (the body is sent once).
// Regular API calls (form or JSON) are retried when a handler matches (429, 5xx, or connection error).

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/slack-go/slack/internal/backoff"
)

// minRetryAfter429 is the minimum wait before retrying after 429 when Retry-After is missing
// or zero, to avoid tight retry loops when using a partial RetryConfig.
const minRetryAfter429 = time.Second

// RetryState holds the current attempt and max retries; passed to handlers.
// Backoff is set by retryClient for use by handlers that want exponential backoff (e.g. connection, server error).
// Handlers may call Backoff.Duration() when retrying; each call advances the backoff for the next retry.
type RetryState struct {
	Attempt    int // current attempt (0-based)
	MaxRetries int
	Backoff    *backoff.Backoff // optional; used by connection/server handlers for exponential backoff
}

// RetryHandler decides whether to retry a request and how long to wait.
// The first handler that returns (true, wait) wins. resp may be nil (connection failure); err may be nil (got response).
type RetryHandler interface {
	ShouldRetry(state *RetryState, req *http.Request, resp *http.Response, err error) (retry bool, wait time.Duration)
}

// RetryConfig configures HTTP retry behavior.
// When MaxRetries is 0, retries are disabled.
// If Handlers is nil, only rate limit (429) is retried (see DefaultRetryHandlers).
type RetryConfig struct {
	// MaxRetries is the maximum number of retry attempts (0 = no retries, 1 = one retry, etc.).
	MaxRetries int
	// Handlers is the list of handlers to consult; nil means 429 only (DefaultRetryHandlers).
	Handlers []RetryHandler
	// RetryAfterDuration is used for 429 when the Retry-After header is missing or invalid.
	RetryAfterDuration time.Duration
	// RetryAfterJitter adds random jitter [0, RetryAfterJitter] to 429 wait to avoid thundering herd (0 = no jitter).
	RetryAfterJitter time.Duration
	// BackoffInitial is the initial backoff for 5xx and connection errors.
	BackoffInitial time.Duration
	// BackoffMax caps the backoff duration.
	BackoffMax time.Duration
	// BackoffJitter adds random jitter [0, BackoffJitter] to backoff to avoid thundering herd (0 to disable).
	BackoffJitter time.Duration
}

// DefaultRetryConfig returns a retry config with sensible defaults.
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxRetries:         3,
		RetryAfterDuration: 60 * time.Second,
		RetryAfterJitter:   1 * time.Second,
		BackoffInitial:     100 * time.Millisecond,
		BackoffMax:         30 * time.Second,
		BackoffJitter:      50 * time.Millisecond,
	}
}

// connectionErrorRetryHandler retries on connection errors (e.g. connection reset).
type connectionErrorRetryHandler struct{}

// NewConnectionErrorRetryHandler returns a handler that retries on connection errors.
func NewConnectionErrorRetryHandler() RetryHandler {
	return &connectionErrorRetryHandler{}
}

func (h *connectionErrorRetryHandler) ShouldRetry(state *RetryState, req *http.Request, resp *http.Response, err error) (bool, time.Duration) {
	if err == nil || resp != nil {
		return false, 0
	}
	if !isRetryableConnError(err) || !requestRetryable(req) {
		return false, 0
	}
	if state.Attempt >= state.MaxRetries {
		return false, 0
	}
	// Backoff is always set by retryClient.Do().
	wait := state.Backoff.Duration()
	return true, wait
}

// rateLimitErrorRetryHandler retries on 429 Too Many Requests using Retry-After or config.
type rateLimitErrorRetryHandler struct {
	cfg RetryConfig
}

// NewRateLimitErrorRetryHandler returns a handler that retries on 429.
func NewRateLimitErrorRetryHandler(cfg RetryConfig) RetryHandler {
	return &rateLimitErrorRetryHandler{cfg: cfg}
}

func (h *rateLimitErrorRetryHandler) ShouldRetry(state *RetryState, req *http.Request, resp *http.Response, err error) (bool, time.Duration) {
	if resp == nil || resp.StatusCode != http.StatusTooManyRequests || !requestRetryable(req) {
		return false, 0
	}
	if state.Attempt >= state.MaxRetries {
		return false, 0
	}
	dur := h.cfg.RetryAfterDuration
	if s := resp.Header.Get("Retry-After"); s != "" {
		// Parsing only integer seconds is appropriate for Slack (API sends seconds; RFC 7231 also allows HTTP-date).
		if sec, parseErr := strconv.ParseInt(strings.TrimSpace(s), 10, 64); parseErr == nil {
			if sec > 0 {
				dur = time.Duration(sec) * time.Second
			} else {
				dur = minRetryAfter429 // Retry-After: 0 means use minimum delay
			}
		}
	}
	dur = max(dur, minRetryAfter429)
	if h.cfg.RetryAfterJitter > 0 {
		dur += time.Duration(rand.IntN(int(h.cfg.RetryAfterJitter)))
	}
	return true, dur
}

// serverErrorRetryHandler retries on 5xx server errors (opt-in).
type serverErrorRetryHandler struct{}

// NewServerErrorRetryHandler returns a handler that retries on 5xx. Opt-in; not in DefaultRetryHandlers, ConnectionOnlyRetryHandlers, or AllBuiltinRetryHandlers.
func NewServerErrorRetryHandler(cfg RetryConfig) RetryHandler {
	return &serverErrorRetryHandler{}
}

func (h *serverErrorRetryHandler) ShouldRetry(state *RetryState, req *http.Request, resp *http.Response, err error) (bool, time.Duration) {
	if resp == nil || resp.StatusCode < http.StatusInternalServerError || !requestRetryable(req) {
		return false, 0
	}
	if state.Attempt >= state.MaxRetries {
		return false, 0
	}
	// Backoff is always set by retryClient.Do().
	wait := state.Backoff.Duration()
	return true, wait
}

// DefaultRetryHandlers returns the default handler when retries are on: rate limit (429) only.
// Used when Handlers is nil. Use AllBuiltinRetryHandlers(cfg) for connection + 429.
func DefaultRetryHandlers(cfg RetryConfig) []RetryHandler {
	return []RetryHandler{NewRateLimitErrorRetryHandler(cfg)}
}

// ConnectionOnlyRetryHandlers returns connection-only handlers (no 429 retries).
func ConnectionOnlyRetryHandlers() []RetryHandler {
	return []RetryHandler{NewConnectionErrorRetryHandler()}
}

// AllBuiltinRetryHandlers returns connection + rate limit (429) handlers; no 5xx.
func AllBuiltinRetryHandlers(cfg RetryConfig) []RetryHandler {
	return []RetryHandler{
		NewConnectionErrorRetryHandler(),
		NewRateLimitErrorRetryHandler(cfg),
	}
}

// retryClient wraps an httpClient and retries according to config.Handlers.
type retryClient struct {
	client httpClient
	config RetryConfig
	debug  Debug // optional; when set and Debug() is true, retries are logged
}

var _ httpClient = (*retryClient)(nil)

// handlers returns the list of retry handlers. Empty Handlers slice is treated like nil (default 429 only).
func (c *retryClient) handlers() []RetryHandler {
	if len(c.config.Handlers) > 0 {
		return c.config.Handlers
	}
	return DefaultRetryHandlers(c.config)
}

func (c *retryClient) logRetry(attempt int, reason string, detail any) {
	if c.debug == nil || !c.debug.Debug() {
		return
	}
	c.debug.Debugf("slack retry: %s (attempt %d/%d), detail: %v", reason, attempt+1, c.config.MaxRetries+1, detail)
}

// requestRetryable reports whether the request can be safely retried: either it has no body
// (e.g. GET) or the body can be replayed via GetBody (e.g. POST with GetBody set).
// Requests with a non-nil body and nil GetBody (e.g. streaming uploads) must not be retried.
func requestRetryable(req *http.Request) bool {
	return req.Body == nil || req.GetBody != nil
}

// sleepWithContext sleeps for up to d, or until ctx is done. Returns true if the full duration
// elapsed, false if ctx was cancelled (caller should return ctx.Err()).
func sleepWithContext(ctx context.Context, d time.Duration) bool {
	if d <= 0 {
		return true
	}
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-timer.C:
		return true
	case <-ctx.Done():
		return false
	}
}

func (c *retryClient) Do(req *http.Request) (*http.Response, error) {
	handlers := c.handlers()
	bo := &backoff.Backoff{
		Initial: c.config.BackoffInitial,
		Max:     c.config.BackoffMax,
		Jitter:  c.config.BackoffJitter,
	}
	var lastErr error
	maxAttempts := c.config.MaxRetries + 1

	for attempt := range maxAttempts {
		state := &RetryState{Attempt: attempt, MaxRetries: c.config.MaxRetries, Backoff: bo}

		// Rewind body for retries (POST/PUT with GetBody).
		if attempt > 0 && req.GetBody != nil {
			newBody, err := req.GetBody()
			if err != nil {
				return nil, err
			}
			req.Body = newBody
		}

		resp, err := c.client.Do(req)
		if err != nil {
			lastErr = err
		}

		// Success: got response and status is not 429/5xx.
		if err == nil && resp.StatusCode != http.StatusTooManyRequests && resp.StatusCode < http.StatusInternalServerError {
			return resp, nil
		}

		// No retries left or request body cannot be replayed — return now.
		if attempt >= c.config.MaxRetries || !requestRetryable(req) {
			if err != nil {
				return nil, err
			}
			return resp, nil
		}

		// First handler that wants to retry wins.
		var wait time.Duration
		retry := false
		for _, h := range handlers {
			if r, w := h.ShouldRetry(state, req, resp, err); r {
				retry, wait = true, w
				break
			}
		}
		if !retry {
			if err != nil {
				return nil, err
			}
			return resp, nil
		}

		// Log, discard response body if we have one, sleep, then next attempt.
		if err != nil {
			c.logRetry(attempt, "connection error", err)
		} else {
			reason := fmt.Sprintf("%d %s", resp.StatusCode, http.StatusText(resp.StatusCode))
			c.logRetry(attempt, reason, wait)
			_, _ = io.Copy(io.Discard, resp.Body)
			_ = resp.Body.Close()
		}
		if !sleepWithContext(req.Context(), wait) {
			return nil, req.Context().Err()
		}
	}

	return nil, lastErr
}

func isRetryableConnError(err error) bool {
	for ; err != nil; err = errors.Unwrap(err) {
		s := err.Error()
		if strings.Contains(s, "connection reset") ||
			strings.Contains(s, "connection refused") ||
			strings.Contains(s, "EOF") {
			return true
		}
	}
	return false
}
