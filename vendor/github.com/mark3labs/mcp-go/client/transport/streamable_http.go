package transport

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/util"
)

type StreamableHTTPCOption func(*StreamableHTTP)

// WithContinuousListening enables receiving server-to-client notifications when no request is in flight.
// In particular, if you want to receive global notifications from the server (like ToolListChangedNotification),
// you should enable this option.
//
// It will establish a standalone long-live GET HTTP connection to the server.
// https://modelcontextprotocol.io/specification/2025-03-26/basic/transports#listening-for-messages-from-the-server
// NOTICE: Even enabled, the server may not support this feature.
func WithContinuousListening() StreamableHTTPCOption {
	return func(sc *StreamableHTTP) {
		sc.getListeningEnabled = true
	}
}

// WithHTTPClient sets a custom HTTP client on the StreamableHTTP transport.
func WithHTTPBasicClient(client *http.Client) StreamableHTTPCOption {
	return func(sc *StreamableHTTP) {
		sc.httpClient = client
	}
}

func WithHTTPHeaders(headers map[string]string) StreamableHTTPCOption {
	return func(sc *StreamableHTTP) {
		sc.headers = headers
	}
}

func WithHTTPHeaderFunc(headerFunc HTTPHeaderFunc) StreamableHTTPCOption {
	return func(sc *StreamableHTTP) {
		sc.headerFunc = headerFunc
	}
}

// WithHTTPTimeout sets the timeout for a HTTP request and stream.
func WithHTTPTimeout(timeout time.Duration) StreamableHTTPCOption {
	return func(sc *StreamableHTTP) {
		sc.httpClient.Timeout = timeout
	}
}

// WithHTTPOAuth enables OAuth authentication for the client.
func WithHTTPOAuth(config OAuthConfig) StreamableHTTPCOption {
	return func(sc *StreamableHTTP) {
		sc.oauthHandler = NewOAuthHandler(config)
	}
}

// WithHTTPLogger sets a custom logger for the StreamableHTTP transport.
func WithHTTPLogger(logger util.Logger) StreamableHTTPCOption {
	return func(sc *StreamableHTTP) {
		sc.logger = logger
	}
}

// Deprecated: Use [WithHTTPLogger] instead.
func WithLogger(logger util.Logger) StreamableHTTPCOption {
	return WithHTTPLogger(logger)
}

// WithSession creates a client with a pre-configured session
func WithSession(sessionID string) StreamableHTTPCOption {
	return func(sc *StreamableHTTP) {
		sc.sessionID.Store(sessionID)
	}
}

// StreamableHTTP implements Streamable HTTP transport.
//
// It transmits JSON-RPC messages over individual HTTP requests. One message per request.
// The HTTP response body can either be a single JSON-RPC response,
// or an upgraded SSE stream that concludes with a JSON-RPC response for the same request.
//
// https://modelcontextprotocol.io/specification/2025-03-26/basic/transports
//
// The current implementation does not support the following features:
//   - resuming stream
//     (https://modelcontextprotocol.io/specification/2025-03-26/basic/transports#resumability-and-redelivery)
type StreamableHTTP struct {
	serverURL           *url.URL
	httpClient          *http.Client
	headers             map[string]string
	headerFunc          HTTPHeaderFunc
	logger              util.Logger
	getListeningEnabled bool

	sessionID       atomic.Value // string
	protocolVersion atomic.Value // string

	initialized     chan struct{}
	initializedOnce sync.Once

	notificationHandler func(mcp.JSONRPCNotification)
	notifyMu            sync.RWMutex

	// Request handler for incoming server-to-client requests (like sampling)
	requestHandler RequestHandler
	requestMu      sync.RWMutex

	closed chan struct{}

	// OAuth support
	oauthHandler *OAuthHandler
	wg           sync.WaitGroup
}

// NewStreamableHTTP creates a new Streamable HTTP transport with the given server URL.
// Returns an error if the URL is invalid.
func NewStreamableHTTP(serverURL string, options ...StreamableHTTPCOption) (*StreamableHTTP, error) {
	parsedURL, err := url.Parse(serverURL)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}

	smc := &StreamableHTTP{
		serverURL:   parsedURL,
		httpClient:  &http.Client{},
		headers:     make(map[string]string),
		closed:      make(chan struct{}),
		logger:      util.DefaultLogger(),
		initialized: make(chan struct{}),
	}
	smc.sessionID.Store("") // set initial value to simplify later usage

	for _, opt := range options {
		if opt != nil {
			opt(smc)
		}
	}

	// If OAuth is configured, set the base URL for metadata discovery
	if smc.oauthHandler != nil {
		// Extract base URL from server URL for metadata discovery
		baseURL := fmt.Sprintf("%s://%s", parsedURL.Scheme, parsedURL.Host)
		smc.oauthHandler.SetBaseURL(baseURL)
	}

	return smc, nil
}

// Start initiates the HTTP connection to the server.
func (c *StreamableHTTP) Start(ctx context.Context) error {
	// Start is idempotent - check if already initialized
	select {
	case <-c.initialized:
		return nil
	default:
	}

	// For Streamable HTTP, we don't need to establish a persistent connection by default
	if c.getListeningEnabled {
		go func() {
			select {
			case <-c.initialized:
				ctx, cancel := c.contextAwareOfClientClose(ctx)
				defer cancel()
				c.listenForever(ctx)
			case <-c.closed:
				return
			}
		}()
	}

	return nil
}

// Close closes the all the HTTP connections to the server.
func (c *StreamableHTTP) Close() error {
	select {
	case <-c.closed:
		return nil
	default:
	}
	// Cancel all in-flight requests
	close(c.closed)

	sessionId := c.sessionID.Load().(string)
	if sessionId != "" {
		c.sessionID.Store("")
		c.wg.Add(1)
		// notify server session closed
		go func() {
			defer c.wg.Done()
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			req, err := http.NewRequestWithContext(ctx, http.MethodDelete, c.serverURL.String(), nil)
			if err != nil {
				c.logger.Errorf("failed to create close request: %v", err)
				return
			}
			req.Header.Set(HeaderKeySessionID, sessionId)
			// Set protocol version header if negotiated
			if v := c.protocolVersion.Load(); v != nil {
				if version, ok := v.(string); ok && version != "" {
					req.Header.Set(HeaderKeyProtocolVersion, version)
				}
			}
			res, err := c.httpClient.Do(req)
			if err != nil {
				c.logger.Errorf("failed to send close request: %v", err)
				return
			}
			res.Body.Close()
		}()
	}
	c.wg.Wait()
	return nil
}

// SetProtocolVersion sets the negotiated protocol version for this connection.
func (c *StreamableHTTP) SetProtocolVersion(version string) {
	c.protocolVersion.Store(version)
}

// ErrOAuthAuthorizationRequired is a sentinel error for OAuth authorization required
var ErrOAuthAuthorizationRequired = errors.New("no valid token available, authorization required")

// OAuthAuthorizationRequiredError is returned when OAuth authorization is required
type OAuthAuthorizationRequiredError struct {
	Handler *OAuthHandler
}

func (e *OAuthAuthorizationRequiredError) Error() string {
	return ErrOAuthAuthorizationRequired.Error()
}

func (e *OAuthAuthorizationRequiredError) Unwrap() error {
	return ErrOAuthAuthorizationRequired
}

// SendRequest sends a JSON-RPC request to the server and waits for a response.
// Returns the raw JSON response message or an error if the request fails.
func (c *StreamableHTTP) SendRequest(
	ctx context.Context,
	request JSONRPCRequest,
) (*JSONRPCResponse, error) {
	// Marshal request
	requestBody, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	ctx, cancel := c.contextAwareOfClientClose(ctx)
	defer cancel()

	resp, err := c.sendHTTP(ctx, http.MethodPost, bytes.NewReader(requestBody), "application/json, text/event-stream", request.Header)
	if err != nil {
		if errors.Is(err, ErrSessionTerminated) && request.Method == string(mcp.MethodInitialize) {
			// If the request is initialize, should not return a SessionTerminated error
			// It should be a genuine endpoint-routing issue.
			// ( Fall through to return StatusCode checking. )
		} else {
			return nil, fmt.Errorf("failed to send request: %w", err)
		}
	}

	// Only proceed if we have a valid response.
	// When sendHTTP fails and resp is nil but method is mcp.MethodInitialize
	// defer resp.Body.Close() fails with nil pointer dereference.
	if resp == nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Check if we got an error response
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {

		// Handle OAuth unauthorized error
		if resp.StatusCode == http.StatusUnauthorized && c.oauthHandler != nil {
			return nil, &OAuthAuthorizationRequiredError{
				Handler: c.oauthHandler,
			}
		}

		// handle error response
		var errResponse JSONRPCResponse
		body, _ := io.ReadAll(resp.Body)
		if err := json.Unmarshal(body, &errResponse); err == nil {
			return &errResponse, nil
		}
		return nil, fmt.Errorf("request failed with status %d: %s", resp.StatusCode, body)
	}

	if request.Method == string(mcp.MethodInitialize) {
		// saved the received session ID in the response
		// empty session ID is allowed
		if sessionID := resp.Header.Get(HeaderKeySessionID); sessionID != "" {
			c.sessionID.Store(sessionID)
		}

		c.initializedOnce.Do(func() {
			close(c.initialized)
		})
	}

	// Handle different response types
	mediaType, _, _ := mime.ParseMediaType(resp.Header.Get("Content-Type"))
	switch mediaType {
	case "application/json":
		// Single response
		var response JSONRPCResponse
		if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
			return nil, fmt.Errorf("failed to decode response: %w", err)
		}

		// should not be a notification
		if response.ID.IsNil() {
			return nil, fmt.Errorf("response should contain RPC id: %v", response)
		}

		return &response, nil

	case "text/event-stream":
		// Server is using SSE for streaming responses
		return c.handleSSEResponse(ctx, resp.Body, false)

	default:
		return nil, fmt.Errorf("unexpected content type: %s", resp.Header.Get("Content-Type"))
	}
}

func (c *StreamableHTTP) sendHTTP(
	ctx context.Context,
	method string,
	body io.Reader,
	acceptType string,
	header http.Header,
) (resp *http.Response, err error) {
	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, method, c.serverURL.String(), body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// request headers
	if header != nil {
		req.Header = header
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", acceptType)
	sessionID := c.sessionID.Load().(string)
	if sessionID != "" {
		req.Header.Set(HeaderKeySessionID, sessionID)
	}
	// Set protocol version header if negotiated
	if v := c.protocolVersion.Load(); v != nil {
		if version, ok := v.(string); ok && version != "" {
			req.Header.Set(HeaderKeyProtocolVersion, version)
		}
	}
	for k, v := range c.headers {
		req.Header.Set(k, v)
	}

	// Add OAuth authorization if configured
	if c.oauthHandler != nil {
		authHeader, err := c.oauthHandler.GetAuthorizationHeader(ctx)
		if err != nil {
			// If we get an authorization error, return a specific error that can be handled by the client
			if errors.Is(err, ErrOAuthAuthorizationRequired) {
				return nil, &OAuthAuthorizationRequiredError{
					Handler: c.oauthHandler,
				}
			}
			return nil, fmt.Errorf("failed to get authorization header: %w", err)
		}
		req.Header.Set("Authorization", authHeader)
	}

	if c.headerFunc != nil {
		for k, v := range c.headerFunc(ctx) {
			req.Header.Set(k, v)
		}
	}

	// Send request
	resp, err = c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	// universal handling for session terminated
	if resp.StatusCode == http.StatusNotFound {
		c.sessionID.CompareAndSwap(sessionID, "")
		return nil, ErrSessionTerminated
	}

	return resp, nil
}

// handleSSEResponse processes an SSE stream for a specific request.
// It returns the final result for the request once received, or an error.
// If ignoreResponse is true, it won't return when a response messge is received. This is for continuous listening.
func (c *StreamableHTTP) handleSSEResponse(ctx context.Context, reader io.ReadCloser, ignoreResponse bool) (*JSONRPCResponse, error) {
	// Create a channel for this specific request
	responseChan := make(chan *JSONRPCResponse, 1)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Start a goroutine to process the SSE stream
	go func() {
		// Ensure this goroutine respects the context
		defer close(responseChan)

		c.readSSE(ctx, reader, func(event, data string) {
			// Try to unmarshal as a response first
			var message JSONRPCResponse
			if err := json.Unmarshal([]byte(data), &message); err != nil {
				c.logger.Infof("failed to unmarshal message (non-fatal): %v", err, "message", data)
				return
			}

			// Handle notification
			if message.ID.IsNil() {
				var notification mcp.JSONRPCNotification
				if err := json.Unmarshal([]byte(data), &notification); err != nil {
					c.logger.Errorf("failed to unmarshal notification: %v", err)
					return
				}
				c.notifyMu.RLock()
				if c.notificationHandler != nil {
					c.notificationHandler(notification)
				}
				c.notifyMu.RUnlock()
				return
			}

			// Check if this is actually a request from the server by looking for method field
			var rawMessage map[string]json.RawMessage
			if err := json.Unmarshal([]byte(data), &rawMessage); err == nil {
				if _, hasMethod := rawMessage["method"]; hasMethod && !message.ID.IsNil() {
					var request JSONRPCRequest
					if err := json.Unmarshal([]byte(data), &request); err == nil {
						// This is a request from the server
						c.handleIncomingRequest(ctx, request)
						return
					}
				}
			}

			if !ignoreResponse {
				responseChan <- &message
			}
		})
	}()

	// Wait for the response or context cancellation
	select {
	case response := <-responseChan:
		if response == nil {
			return nil, fmt.Errorf("unexpected nil response")
		}
		return response, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// readSSE reads the SSE stream(reader) and calls the handler for each event and data pair.
// It will end when the reader is closed (or the context is done).
func (c *StreamableHTTP) readSSE(ctx context.Context, reader io.ReadCloser, handler func(event, data string)) {
	defer reader.Close()

	br := bufio.NewReader(reader)
	var event, data string

	for {
		select {
		case <-ctx.Done():
			return
		default:
			line, err := br.ReadString('\n')
			if err != nil {
				if err == io.EOF {
					// Process any pending event before exit
					if data != "" {
						// If no event type is specified, use empty string (default event type)
						if event == "" {
							event = "message"
						}
						handler(event, data)
					}
					return
				}
				select {
				case <-ctx.Done():
					return
				default:
					c.logger.Errorf("SSE stream error: %v", err)
					return
				}
			}

			// Remove only newline markers
			line = strings.TrimRight(line, "\r\n")
			if line == "" {
				// Empty line means end of event
				if data != "" {
					// If no event type is specified, use empty string (default event type)
					if event == "" {
						event = "message"
					}
					handler(event, data)
					event = ""
					data = ""
				}
				continue
			}

			if strings.HasPrefix(line, "event:") {
				event = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
			} else if strings.HasPrefix(line, "data:") {
				data = strings.TrimSpace(strings.TrimPrefix(line, "data:"))
			}
		}
	}
}

func (c *StreamableHTTP) SendNotification(ctx context.Context, notification mcp.JSONRPCNotification) error {
	// Marshal request
	requestBody, err := json.Marshal(notification)
	if err != nil {
		return fmt.Errorf("failed to marshal notification: %w", err)
	}

	// Create HTTP request
	ctx, cancel := c.contextAwareOfClientClose(ctx)
	defer cancel()

	resp, err := c.sendHTTP(ctx, http.MethodPost, bytes.NewReader(requestBody), "application/json, text/event-stream", nil)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		// Handle OAuth unauthorized error
		if resp.StatusCode == http.StatusUnauthorized && c.oauthHandler != nil {
			return &OAuthAuthorizationRequiredError{
				Handler: c.oauthHandler,
			}
		}

		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf(
			"notification failed with status %d: %s",
			resp.StatusCode,
			body,
		)
	}

	return nil
}

func (c *StreamableHTTP) SetNotificationHandler(handler func(mcp.JSONRPCNotification)) {
	c.notifyMu.Lock()
	defer c.notifyMu.Unlock()
	c.notificationHandler = handler
}

// SetRequestHandler sets the handler for incoming requests from the server.
func (c *StreamableHTTP) SetRequestHandler(handler RequestHandler) {
	c.requestMu.Lock()
	defer c.requestMu.Unlock()
	c.requestHandler = handler
}

func (c *StreamableHTTP) GetSessionId() string {
	return c.sessionID.Load().(string)
}

// GetOAuthHandler returns the OAuth handler if configured
func (c *StreamableHTTP) GetOAuthHandler() *OAuthHandler {
	return c.oauthHandler
}

// IsOAuthEnabled returns true if OAuth is enabled
func (c *StreamableHTTP) IsOAuthEnabled() bool {
	return c.oauthHandler != nil
}

func (c *StreamableHTTP) listenForever(ctx context.Context) {
	c.logger.Infof("listening to server forever")
	for {
		// Use the original context for continuous listening - no per-iteration timeout
		// The SSE connection itself will detect disconnections via the underlying HTTP transport,
		// and the context cancellation will propagate from the parent to stop listening gracefully.
		// We don't add an artificial timeout here because:
		// 1. Persistent SSE connections are meant to stay open indefinitely
		// 2. Network-level timeouts and keep-alives handle connection health
		// 3. Context cancellation (user-initiated or system shutdown) provides clean shutdown
		err := c.createGETConnectionToServer(ctx)
		if errors.Is(err, ErrGetMethodNotAllowed) {
			// server does not support listening
			c.logger.Errorf("server does not support listening")
			return
		}

		select {
		case <-ctx.Done():
			return
		default:
		}

		if err != nil {
			c.logger.Errorf("failed to listen to server. retry in 1 second: %v", err)
		}

		// Use context-aware sleep
		select {
		case <-time.After(retryInterval):
		case <-ctx.Done():
			return
		}
	}
}

var (
	ErrSessionTerminated   = fmt.Errorf("session terminated (404). need to re-initialize")
	ErrGetMethodNotAllowed = fmt.Errorf("GET method not allowed")

	retryInterval = 1 * time.Second // a variable is convenient for testing
)

func (c *StreamableHTTP) createGETConnectionToServer(ctx context.Context) error {
	resp, err := c.sendHTTP(ctx, http.MethodGet, nil, "text/event-stream", nil)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Check if we got an error response
	if resp.StatusCode == http.StatusMethodNotAllowed {
		return ErrGetMethodNotAllowed
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("request failed with status %d: %s", resp.StatusCode, body)
	}

	// handle SSE response
	contentType := resp.Header.Get("Content-Type")
	if contentType != "text/event-stream" {
		return fmt.Errorf("unexpected content type: %s", contentType)
	}

	// When ignoreResponse is true, the function will never return expect context is done.
	// NOTICE: Due to the ambiguity of the specification, other SDKs may use the GET connection to transfer the response
	// messages. To be more compatible, we should handle this response, however, as the transport layer is message-based,
	// currently, there is no convenient way to handle this response.
	// So we ignore the response here. It's not a bug, but may be not compatible with other SDKs.
	_, err = c.handleSSEResponse(ctx, resp.Body, true)
	if err != nil {
		return fmt.Errorf("failed to handle SSE response: %w", err)
	}

	return nil
}

// handleIncomingRequest processes requests from the server (like sampling requests)
func (c *StreamableHTTP) handleIncomingRequest(ctx context.Context, request JSONRPCRequest) {
	c.requestMu.RLock()
	handler := c.requestHandler
	c.requestMu.RUnlock()

	if handler == nil {
		c.logger.Errorf("received request from server but no handler set: %s", request.Method)
		// Send method not found error
		errorResponse := NewJSONRPCErrorResponse(
			request.ID,
			mcp.METHOD_NOT_FOUND,
			fmt.Sprintf("no handler configured for method: %s", request.Method),
			nil,
		)
		c.sendResponseToServer(ctx, errorResponse)
		return
	}

	// Handle the request in a goroutine to avoid blocking the SSE reader
	go func() {
		// Create a new context with timeout for request handling, respecting parent context
		requestCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()

		response, err := handler(requestCtx, request)
		if err != nil {
			c.logger.Errorf("error handling request %s: %v", request.Method, err)

			// Determine appropriate JSON-RPC error code based on error type
			var errorCode int
			var errorMessage string

			// Check for specific sampling-related errors
			if errors.Is(err, context.Canceled) {
				errorCode = mcp.REQUEST_INTERRUPTED
				errorMessage = "request was cancelled"
			} else if errors.Is(err, context.DeadlineExceeded) {
				errorCode = mcp.REQUEST_INTERRUPTED
				errorMessage = "request timed out"
			} else {
				// Generic error cases
				switch request.Method {
				case string(mcp.MethodSamplingCreateMessage):
					errorCode = mcp.INTERNAL_ERROR
					errorMessage = fmt.Sprintf("sampling request failed: %v", err)
				default:
					errorCode = mcp.INTERNAL_ERROR
					errorMessage = err.Error()
				}
			}

			// Send error response
			errorResponse := NewJSONRPCErrorResponse(request.ID, errorCode, errorMessage, nil)
			c.sendResponseToServer(requestCtx, errorResponse)
			return
		}

		if response != nil {
			c.sendResponseToServer(requestCtx, response)
		}
	}()
}

// sendResponseToServer sends a response back to the server via HTTP POST
func (c *StreamableHTTP) sendResponseToServer(ctx context.Context, response *JSONRPCResponse) {
	if response == nil {
		c.logger.Errorf("cannot send nil response to server")
		return
	}

	responseBody, err := json.Marshal(response)
	if err != nil {
		c.logger.Errorf("failed to marshal response: %v", err)
		return
	}

	ctx, cancel := c.contextAwareOfClientClose(ctx)
	defer cancel()

	resp, err := c.sendHTTP(ctx, http.MethodPost, bytes.NewReader(responseBody), "application/json, text/event-stream", nil)
	if err != nil {
		c.logger.Errorf("failed to send response to server: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		body, _ := io.ReadAll(resp.Body)
		c.logger.Errorf("server rejected response with status %d: %s", resp.StatusCode, body)
	}
}

func (c *StreamableHTTP) contextAwareOfClientClose(ctx context.Context) (context.Context, context.CancelFunc) {
	newCtx, cancel := context.WithCancel(ctx)
	go func() {
		select {
		case <-c.closed:
			cancel()
		case <-newCtx.Done():
			// The original context was canceled
			cancel()
		}
	}()
	return newCtx, cancel
}
