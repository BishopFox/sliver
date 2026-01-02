package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"mime"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unicode"

	"github.com/google/uuid"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/util"
)

// StreamableHTTPOption defines a function type for configuring StreamableHTTPServer
type StreamableHTTPOption func(*StreamableHTTPServer)

// WithEndpointPath sets the endpoint path for the server.
// The default is "/mcp".
// It's only works for `Start` method. When used as a http.Handler, it has no effect.
func WithEndpointPath(endpointPath string) StreamableHTTPOption {
	return func(s *StreamableHTTPServer) {
		// Normalize the endpoint path to ensure it starts with a slash and doesn't end with one
		normalizedPath := "/" + strings.Trim(endpointPath, "/")
		s.endpointPath = normalizedPath
	}
}

// WithStateLess sets the server to stateless mode.
// If true, the server will manage no session information. Every request will be treated
// as a new session. No session id returned to the client.
// The default is false.
//
// Note: This is a convenience method. It's identical to set WithSessionIdManager option
// to StatelessSessionIdManager.
func WithStateLess(stateLess bool) StreamableHTTPOption {
	return func(s *StreamableHTTPServer) {
		if stateLess {
			s.sessionIdManagerResolver = NewDefaultSessionIdManagerResolver(&StatelessSessionIdManager{})
		}
	}
}

// WithSessionIdManager sets a custom session id generator for the server.
// By default, the server uses StatelessGeneratingSessionIdManager (generates IDs but no local validation).
// Note: Options are applied in order; the last one wins. If combined with
// WithStateLess or WithSessionIdManagerResolver, whichever is applied last takes effect.
func WithSessionIdManager(manager SessionIdManager) StreamableHTTPOption {
	return func(s *StreamableHTTPServer) {
		if manager == nil {
			s.sessionIdManagerResolver = NewDefaultSessionIdManagerResolver(&StatelessSessionIdManager{})
			return
		}
		s.sessionIdManagerResolver = NewDefaultSessionIdManagerResolver(manager)
	}
}

// WithSessionIdManagerResolver sets a custom session id manager resolver for the server.
// This allows for request-based session id management strategies.
// Note: Options are applied in order; the last one wins. If combined with
// WithStateLess or WithSessionIdManager, whichever is applied last takes effect.
func WithSessionIdManagerResolver(resolver SessionIdManagerResolver) StreamableHTTPOption {
	return func(s *StreamableHTTPServer) {
		if resolver == nil {
			s.sessionIdManagerResolver = NewDefaultSessionIdManagerResolver(&StatelessSessionIdManager{})
			return
		}
		s.sessionIdManagerResolver = resolver
	}
}

// WithStateful enables stateful session management using InsecureStatefulSessionIdManager.
// This requires sticky sessions in multi-instance deployments.
func WithStateful(stateful bool) StreamableHTTPOption {
	return func(s *StreamableHTTPServer) {
		if stateful {
			s.sessionIdManagerResolver = NewDefaultSessionIdManagerResolver(&InsecureStatefulSessionIdManager{})
		}
	}
}

// WithHeartbeatInterval sets the heartbeat interval. Positive interval means the
// server will send a heartbeat to the client through the GET connection, to keep
// the connection alive from being closed by the network infrastructure (e.g.
// gateways). If the client does not establish a GET connection, it has no
// effect. The default is not to send heartbeats.
func WithHeartbeatInterval(interval time.Duration) StreamableHTTPOption {
	return func(s *StreamableHTTPServer) {
		s.listenHeartbeatInterval = interval
	}
}

// WithDisableStreaming prevents the server from responding to GET requests with
// a streaming response. Instead, it will respond with a 405 Method Not Allowed status.
// This can be useful in scenarios where streaming is not desired or supported.
// The default is false, meaning streaming is enabled.
func WithDisableStreaming(disable bool) StreamableHTTPOption {
	return func(s *StreamableHTTPServer) {
		s.disableStreaming = disable
	}
}

// WithHTTPContextFunc sets a function that will be called to customise the context
// to the server using the incoming request.
// This can be used to inject context values from headers, for example.
func WithHTTPContextFunc(fn HTTPContextFunc) StreamableHTTPOption {
	return func(s *StreamableHTTPServer) {
		s.contextFunc = fn
	}
}

// WithStreamableHTTPServer sets the HTTP server instance for StreamableHTTPServer.
// NOTE: When providing a custom HTTP server, you must handle routing yourself
// If routing is not set up, the server will start but won't handle any MCP requests.
func WithStreamableHTTPServer(srv *http.Server) StreamableHTTPOption {
	return func(s *StreamableHTTPServer) {
		s.httpServer = srv
	}
}

// WithLogger sets the logger for the server
func WithLogger(logger util.Logger) StreamableHTTPOption {
	return func(s *StreamableHTTPServer) {
		s.logger = logger
	}
}

// WithTLSCert sets the TLS certificate and key files for HTTPS support.
// Both certFile and keyFile must be provided to enable TLS.
func WithTLSCert(certFile, keyFile string) StreamableHTTPOption {
	return func(s *StreamableHTTPServer) {
		s.tlsCertFile = certFile
		s.tlsKeyFile = keyFile
	}
}

// StreamableHTTPServer implements a Streamable-http based MCP server.
// It communicates with clients over HTTP protocol, supporting both direct HTTP responses, and SSE streams.
// https://modelcontextprotocol.io/specification/2025-03-26/basic/transports#streamable-http
//
// Usage:
//
//	server := NewStreamableHTTPServer(mcpServer)
//	server.Start(":8080") // The final url for client is http://xxxx:8080/mcp by default
//
// or the server itself can be used as a http.Handler, which is convenient to
// integrate with existing http servers, or advanced usage:
//
//	handler := NewStreamableHTTPServer(mcpServer)
//	http.Handle("/streamable-http", handler)
//	http.ListenAndServe(":8080", nil)
//
// Notice:
// Except for the GET handlers(listening), the POST handlers(request/notification) will
// not trigger the session registration. So the methods like `SendNotificationToSpecificClient`
// or `hooks.onRegisterSession` will not be triggered for POST messages.
//
// The current implementation does not support the following features from the specification:
//   - Stream Resumability
type StreamableHTTPServer struct {
	server                   *MCPServer
	sessionTools             *sessionToolsStore
	sessionResources         *sessionResourcesStore
	sessionResourceTemplates *sessionResourceTemplatesStore
	sessionRequestIDs        sync.Map // sessionId --> last requestID(*atomic.Int64)
	activeSessions           sync.Map // sessionId --> *streamableHttpSession (for sampling responses)

	httpServer *http.Server
	mu         sync.RWMutex

	endpointPath             string
	contextFunc              HTTPContextFunc
	sessionIdManagerResolver SessionIdManagerResolver
	listenHeartbeatInterval  time.Duration
	logger                   util.Logger
	sessionLogLevels         *sessionLogLevelsStore
	disableStreaming         bool

	tlsCertFile string
	tlsKeyFile  string
}

// NewStreamableHTTPServer creates a new streamable-http server instance
func NewStreamableHTTPServer(server *MCPServer, opts ...StreamableHTTPOption) *StreamableHTTPServer {
	s := &StreamableHTTPServer{
		server:                   server,
		sessionTools:             newSessionToolsStore(),
		sessionLogLevels:         newSessionLogLevelsStore(),
		endpointPath:             "/mcp",
		sessionIdManagerResolver: NewDefaultSessionIdManagerResolver(&StatelessGeneratingSessionIdManager{}),
		logger:                   util.DefaultLogger(),
		sessionResources:         newSessionResourcesStore(),
		sessionResourceTemplates: newSessionResourceTemplatesStore(),
	}

	// Apply all options
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// ServeHTTP implements the http.Handler interface.
func (s *StreamableHTTPServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		s.handlePost(w, r)
	case http.MethodGet:
		s.handleGet(w, r)
	case http.MethodDelete:
		s.handleDelete(w, r)
	default:
		http.NotFound(w, r)
	}
}

// Start begins serving the http server on the specified address and path
// (endpointPath). like:
//
//	s.Start(":8080")
func (s *StreamableHTTPServer) Start(addr string) error {
	s.mu.Lock()
	if s.httpServer == nil {
		mux := http.NewServeMux()
		mux.Handle(s.endpointPath, s)
		s.httpServer = &http.Server{
			Addr:    addr,
			Handler: mux,
		}
	} else {
		if s.httpServer.Addr == "" {
			s.httpServer.Addr = addr
		} else if s.httpServer.Addr != addr {
			return fmt.Errorf("conflicting listen address: WithStreamableHTTPServer(%q) vs Start(%q)", s.httpServer.Addr, addr)
		}
	}
	srv := s.httpServer
	s.mu.Unlock()

	if s.tlsCertFile != "" || s.tlsKeyFile != "" {
		if s.tlsCertFile == "" || s.tlsKeyFile == "" {
			return fmt.Errorf("both TLS cert and key must be provided")
		}
		if _, err := os.Stat(s.tlsCertFile); err != nil {
			return fmt.Errorf("failed to find TLS certificate file: %w", err)
		}
		if _, err := os.Stat(s.tlsKeyFile); err != nil {
			return fmt.Errorf("failed to find TLS key file: %w", err)
		}
		return srv.ListenAndServeTLS(s.tlsCertFile, s.tlsKeyFile)
	}

	return srv.ListenAndServe()
}

// Shutdown gracefully stops the server, closing all active sessions
// and shutting down the HTTP server.
func (s *StreamableHTTPServer) Shutdown(ctx context.Context) error {

	// shutdown the server if needed (may use as a http.Handler)
	s.mu.RLock()
	srv := s.httpServer
	s.mu.RUnlock()
	if srv != nil {
		return srv.Shutdown(ctx)
	}
	return nil
}

// --- internal methods ---

func (s *StreamableHTTPServer) handlePost(w http.ResponseWriter, r *http.Request) {
	// post request carry request/notification message

	// Check content type
	contentType := r.Header.Get("Content-Type")
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil || mediaType != "application/json" {
		http.Error(w, "Invalid content type: must be 'application/json'", http.StatusBadRequest)
		return
	}

	// Check the request body is valid json, meanwhile, get the request Method
	rawData, err := io.ReadAll(r.Body)
	if err != nil {
		s.writeJSONRPCError(w, nil, mcp.PARSE_ERROR, fmt.Sprintf("read request body error: %v", err))
		return
	}
	// First, try to parse as a response (sampling responses don't have a method field)
	var jsonMessage struct {
		ID     json.RawMessage `json:"id"`
		Result json.RawMessage `json:"result,omitempty"`
		Error  json.RawMessage `json:"error,omitempty"`
		Method mcp.MCPMethod   `json:"method,omitempty"`
	}
	if err := json.Unmarshal(rawData, &jsonMessage); err != nil {
		s.writeJSONRPCError(w, nil, mcp.PARSE_ERROR, "request body is not valid json")
		return
	}

	// detect empty ping response, skip session ID validation
	isPingResponse := jsonMessage.Method == "" && jsonMessage.ID != nil &&
		(isJSONEmpty(jsonMessage.Result) && isJSONEmpty(jsonMessage.Error))

	if isPingResponse {
		return
	}

	// Check if this is a sampling response (has result/error but no method)
	isSamplingResponse := jsonMessage.Method == "" && jsonMessage.ID != nil &&
		(jsonMessage.Result != nil || jsonMessage.Error != nil)

	isInitializeRequest := jsonMessage.Method == mcp.MethodInitialize

	// Handle sampling responses separately
	if isSamplingResponse {
		if err := s.handleSamplingResponse(w, r, jsonMessage); err != nil {
			s.logger.Errorf("Failed to handle sampling response: %v", err)
			http.Error(w, "Failed to handle sampling response", http.StatusInternalServerError)
		}
		return
	}

	// Prepare the session for the mcp server
	// The session is ephemeral. Its life is the same as the request. It's only created
	// for interaction with the mcp server.
	var sessionID string
	sessionIdManager := s.sessionIdManagerResolver.ResolveSessionIdManager(r)
	if isInitializeRequest {
		// generate a new one for initialize request
		sessionID = sessionIdManager.Generate()
	} else {
		// Get session ID from header.
		// Stateful servers need the client to carry the session ID.
		sessionID = r.Header.Get(HeaderKeySessionID)
		isTerminated, err := sessionIdManager.Validate(sessionID)
		if err != nil {
			http.Error(w, "Invalid session ID", http.StatusBadRequest)
			return
		}
		if isTerminated {
			http.Error(w, "Session terminated", http.StatusNotFound)
			return
		}
	}

	// For non-initialize requests, try to reuse existing registered session
	var session *streamableHttpSession
	if !isInitializeRequest {
		if sessionValue, ok := s.server.sessions.Load(sessionID); ok {
			if existingSession, ok := sessionValue.(*streamableHttpSession); ok {
				session = existingSession
			}
		}
	}

	// Check if a persistent session exists (for sampling support), otherwise create ephemeral session
	// Persistent sessions are created by GET (continuous listening) connections
	if session == nil {
		if sessionInterface, exists := s.activeSessions.Load(sessionID); exists {
			if persistentSession, ok := sessionInterface.(*streamableHttpSession); ok {
				session = persistentSession
			}
		}
	}

	// Create ephemeral session if no persistent session exists
	if session == nil {
		session = newStreamableHttpSession(sessionID, s.sessionTools, s.sessionResources, s.sessionResourceTemplates, s.sessionLogLevels)
	}

	// Set the client context before handling the message
	ctx := s.server.WithContext(r.Context(), session)
	if s.contextFunc != nil {
		ctx = s.contextFunc(ctx, r)
	}

	// handle potential notifications
	mu := sync.Mutex{}
	upgradedHeader := false
	done := make(chan struct{})

	ctx = context.WithValue(ctx, requestHeader, r.Header)
	go func() {
		for {
			select {
			case nt := <-session.notificationChannel:
				func() {
					mu.Lock()
					defer mu.Unlock()
					// if the done chan is closed, as the request is terminated, just return
					select {
					case <-done:
						return
					default:
					}
					defer func() {
						flusher, ok := w.(http.Flusher)
						if ok {
							flusher.Flush()
						}
					}()

					// if there's notifications, upgradedHeader to SSE response
					if !upgradedHeader {
						w.Header().Set("Content-Type", "text/event-stream")
						w.Header().Set("Connection", "keep-alive")
						w.Header().Set("Cache-Control", "no-cache")
						w.WriteHeader(http.StatusOK)
						upgradedHeader = true
					}
					err := writeSSEEvent(w, nt)
					if err != nil {
						s.logger.Errorf("Failed to write SSE event: %v", err)
						return
					}
				}()
			case <-done:
				return
			case <-ctx.Done():
				return
			}
		}
	}()

	// Process message through MCPServer
	response := s.server.HandleMessage(ctx, rawData)
	if response == nil {
		// For notifications, just send 202 Accepted with no body
		w.WriteHeader(http.StatusAccepted)
		return
	}

	// Write response
	mu.Lock()
	defer mu.Unlock()
	// close the done chan before unlock
	defer close(done)
	if ctx.Err() != nil {
		return
	}
	// If client-server communication already upgraded to SSE stream
	if session.upgradeToSSE.Load() {
		if !upgradedHeader {
			w.Header().Set("Content-Type", "text/event-stream")
			w.Header().Set("Connection", "keep-alive")
			w.Header().Set("Cache-Control", "no-cache")
			w.WriteHeader(http.StatusOK)
			upgradedHeader = true
		}
		if err := writeSSEEvent(w, response); err != nil {
			s.logger.Errorf("Failed to write final SSE response event: %v", err)
		}
	} else {
		w.Header().Set("Content-Type", "application/json")
		if isInitializeRequest && sessionID != "" {
			// send the session ID back to the client
			w.Header().Set(HeaderKeySessionID, sessionID)
		}
		w.WriteHeader(http.StatusOK)
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			s.logger.Errorf("Failed to write response: %v", err)
		}
	}

	// Register session after successful initialization
	// Only register if not already registered (e.g., by a GET connection)
	if isInitializeRequest && sessionID != "" {
		if _, exists := s.server.sessions.Load(sessionID); !exists {
			// Store in activeSessions to prevent duplicate registration from GET
			s.activeSessions.Store(sessionID, session)
			// Register the session with the MCPServer for notification support
			if err := s.server.RegisterSession(ctx, session); err != nil {
				s.logger.Errorf("Failed to register POST session: %v", err)
				s.activeSessions.Delete(sessionID)
				// Don't fail the request, just log the error
			}
		}
	}
}

func (s *StreamableHTTPServer) handleGet(w http.ResponseWriter, r *http.Request) {
	// get request is for listening to notifications
	// https://modelcontextprotocol.io/specification/2025-03-26/basic/transports#listening-for-messages-from-the-server
	if s.disableStreaming {
		s.logger.Infof("Rejected GET request: streaming is disabled (session: %s)", r.Header.Get(HeaderKeySessionID))
		http.Error(w, "Streaming is disabled on this server", http.StatusMethodNotAllowed)
		return
	}

	sessionID := r.Header.Get(HeaderKeySessionID)
	// the specification didn't say we should validate the session id

	if sessionID == "" {
		// It's a stateless server,
		// but the MCP server requires a unique ID for registering, so we use a random one
		sessionID = uuid.New().String()
	}

	// Get or create session atomically to prevent TOCTOU races
	// where concurrent GETs could both create and register duplicate sessions
	var session *streamableHttpSession
	newSession := newStreamableHttpSession(sessionID, s.sessionTools, s.sessionResources, s.sessionResourceTemplates, s.sessionLogLevels)
	actual, loaded := s.activeSessions.LoadOrStore(sessionID, newSession)
	session = actual.(*streamableHttpSession)

	if !loaded {
		// We created a new session, need to register it
		if err := s.server.RegisterSession(r.Context(), session); err != nil {
			s.activeSessions.Delete(sessionID)
			http.Error(w, fmt.Sprintf("Session registration failed: %v", err), http.StatusBadRequest)
			return
		}
		defer s.server.UnregisterSession(r.Context(), sessionID)
		defer s.activeSessions.Delete(sessionID)
	}

	// Set the client context before handling the message
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}
	flusher.Flush()

	// Start notification handler for this session
	done := make(chan struct{})
	defer close(done)
	writeChan := make(chan any, 16)

	go func() {
		for {
			select {
			case nt := <-session.notificationChannel:
				select {
				case writeChan <- &nt:
				case <-done:
					return
				}
			case samplingReq := <-session.samplingRequestChan:
				// Send sampling request to client via SSE
				jsonrpcRequest := mcp.JSONRPCRequest{
					JSONRPC: "2.0",
					ID:      mcp.NewRequestId(samplingReq.requestID),
					Request: mcp.Request{
						Method: string(mcp.MethodSamplingCreateMessage),
					},
					Params: samplingReq.request.CreateMessageParams,
				}
				select {
				case writeChan <- jsonrpcRequest:
				case <-done:
					return
				}
			case elicitationReq := <-session.elicitationRequestChan:
				// Send elicitation request to client via SSE
				jsonrpcRequest := mcp.JSONRPCRequest{
					JSONRPC: "2.0",
					ID:      mcp.NewRequestId(elicitationReq.requestID),
					Request: mcp.Request{
						Method: string(mcp.MethodElicitationCreate),
					},
					Params: elicitationReq.request.Params,
				}
				select {
				case writeChan <- jsonrpcRequest:
				case <-done:
					return
				}
			case rootsReq := <-session.rootsRequestChan:
				// Send list roots request to client via SSE
				jsonrpcRequest := mcp.JSONRPCRequest{
					JSONRPC: "2.0",
					ID:      mcp.NewRequestId(rootsReq.requestID),
					Request: mcp.Request{
						Method: string(mcp.MethodListRoots),
					},
				}
				select {
				case writeChan <- jsonrpcRequest:
				case <-done:
					return
				}
			case <-done:
				return
			}
		}
	}()

	if s.listenHeartbeatInterval > 0 {
		// heartbeat to keep the connection alive
		go func() {
			ticker := time.NewTicker(s.listenHeartbeatInterval)
			defer ticker.Stop()
			for {
				select {
				case <-ticker.C:
					message := mcp.JSONRPCRequest{
						JSONRPC: "2.0",
						ID:      mcp.NewRequestId(s.nextRequestID(sessionID)),
						Request: mcp.Request{
							Method: "ping",
						},
					}
					select {
					case writeChan <- message:
					case <-done:
						return
					}
				case <-done:
					return
				}
			}
		}()
	}

	// Keep the connection open until the client disconnects
	//
	// There's will a Available() check when handler ends, and it maybe race with Flush(),
	// so we use a separate channel to send the data, inteading of flushing directly in other goroutine.
	for {
		select {
		case data := <-writeChan:
			if data == nil {
				continue
			}
			if err := writeSSEEvent(w, data); err != nil {
				s.logger.Errorf("Failed to write SSE event: %v", err)
				return
			}
			flusher.Flush()
		case <-r.Context().Done():
			return
		}
	}
}

func (s *StreamableHTTPServer) handleDelete(w http.ResponseWriter, r *http.Request) {
	// delete request terminate the session
	sessionID := r.Header.Get(HeaderKeySessionID)
	sessionIdManager := s.sessionIdManagerResolver.ResolveSessionIdManager(r)
	notAllowed, err := sessionIdManager.Terminate(sessionID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Session termination failed: %v", err), http.StatusInternalServerError)
		return
	}
	if notAllowed {
		http.Error(w, "Session termination not allowed", http.StatusMethodNotAllowed)
		return
	}

	// remove the session relateddata from the sessionToolsStore
	s.sessionTools.delete(sessionID)
	s.sessionResources.delete(sessionID)
	s.sessionResourceTemplates.delete(sessionID)
	s.sessionLogLevels.delete(sessionID)
	// remove current session's requstID information
	s.sessionRequestIDs.Delete(sessionID)

	w.WriteHeader(http.StatusOK)
}

func writeSSEEvent(w io.Writer, data any) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal data: %w", err)
	}
	_, err = fmt.Fprintf(w, "event: message\ndata: %s\n\n", jsonData)
	if err != nil {
		return fmt.Errorf("failed to write SSE event: %w", err)
	}
	return nil
}

// handleSamplingResponse processes incoming sampling responses from clients
func (s *StreamableHTTPServer) handleSamplingResponse(w http.ResponseWriter, r *http.Request, responseMessage struct {
	ID     json.RawMessage `json:"id"`
	Result json.RawMessage `json:"result,omitempty"`
	Error  json.RawMessage `json:"error,omitempty"`
	Method mcp.MCPMethod   `json:"method,omitempty"`
}) error {
	// Get session ID from header
	sessionID := r.Header.Get(HeaderKeySessionID)
	if sessionID == "" {
		http.Error(w, "Missing session ID for sampling response", http.StatusBadRequest)
		return fmt.Errorf("missing session ID")
	}

	// Validate session
	sessionIdManager := s.sessionIdManagerResolver.ResolveSessionIdManager(r)
	isTerminated, err := sessionIdManager.Validate(sessionID)
	if err != nil {
		http.Error(w, "Invalid session ID", http.StatusBadRequest)
		return err
	}
	if isTerminated {
		http.Error(w, "Session terminated", http.StatusNotFound)
		return fmt.Errorf("session terminated")
	}

	// Parse the request ID
	var requestID int64
	if err := json.Unmarshal(responseMessage.ID, &requestID); err != nil {
		http.Error(w, "Invalid request ID in sampling response", http.StatusBadRequest)
		return err
	}

	// Create the sampling response item
	response := samplingResponseItem{
		requestID: requestID,
	}

	// Parse result or error
	if responseMessage.Error != nil {
		// Parse error
		var jsonrpcError struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		}
		if err := json.Unmarshal(responseMessage.Error, &jsonrpcError); err != nil {
			response.err = fmt.Errorf("failed to parse error: %v", err)
		} else {
			response.err = fmt.Errorf("sampling error %d: %s", jsonrpcError.Code, jsonrpcError.Message)
		}
	} else if responseMessage.Result != nil {
		// Store the result to be unmarshaled later
		response.result = responseMessage.Result
	} else {
		response.err = fmt.Errorf("sampling response has neither result nor error")
	}

	// Find the corresponding session and deliver the response
	// The response is delivered to the specific session identified by sessionID
	if err := s.deliverSamplingResponse(sessionID, response); err != nil {
		s.logger.Errorf("Failed to deliver sampling response: %v", err)
		http.Error(w, "Failed to deliver response", http.StatusInternalServerError)
		return err
	}

	// Acknowledge receipt
	w.WriteHeader(http.StatusOK)
	return nil
}

// deliverSamplingResponse delivers a sampling response to the appropriate session
func (s *StreamableHTTPServer) deliverSamplingResponse(sessionID string, response samplingResponseItem) error {
	// Look up the active session
	sessionInterface, ok := s.activeSessions.Load(sessionID)
	if !ok {
		return fmt.Errorf("no active session found for session %s", sessionID)
	}

	session, ok := sessionInterface.(*streamableHttpSession)
	if !ok {
		return fmt.Errorf("invalid session type for session %s", sessionID)
	}

	// Look up the dedicated response channel for this specific request
	responseChannelInterface, exists := session.samplingRequests.Load(response.requestID)
	if !exists {
		return fmt.Errorf("no pending request found for session %s, request %d", sessionID, response.requestID)
	}

	responseChan, ok := responseChannelInterface.(chan samplingResponseItem)
	if !ok {
		return fmt.Errorf("invalid response channel type for session %s, request %d", sessionID, response.requestID)
	}

	// Attempt to deliver the response with timeout to prevent indefinite blocking
	select {
	case responseChan <- response:
		s.logger.Infof("Delivered sampling response for session %s, request %d", sessionID, response.requestID)
		return nil
	default:
		return fmt.Errorf("failed to deliver sampling response for session %s, request %d: channel full or blocked", sessionID, response.requestID)
	}
}

// writeJSONRPCError writes a JSON-RPC error response with the given error details.
func (s *StreamableHTTPServer) writeJSONRPCError(
	w http.ResponseWriter,
	id any,
	code int,
	message string,
) {
	response := createErrorResponse(id, code, message)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	err := json.NewEncoder(w).Encode(response)
	if err != nil {
		s.logger.Errorf("Failed to write JSONRPCError: %v", err)
	}
}

// nextRequestID gets the next incrementing requestID for the current session
func (s *StreamableHTTPServer) nextRequestID(sessionID string) int64 {
	actual, _ := s.sessionRequestIDs.LoadOrStore(sessionID, new(atomic.Int64))
	counter := actual.(*atomic.Int64)
	return counter.Add(1)
}

// --- session ---
type sessionLogLevelsStore struct {
	mu   sync.RWMutex
	logs map[string]mcp.LoggingLevel
}

func newSessionLogLevelsStore() *sessionLogLevelsStore {
	return &sessionLogLevelsStore{
		logs: make(map[string]mcp.LoggingLevel),
	}
}

func (s *sessionLogLevelsStore) get(sessionID string) mcp.LoggingLevel {
	s.mu.RLock()
	defer s.mu.RUnlock()
	val, ok := s.logs[sessionID]
	if !ok {
		return mcp.LoggingLevelError
	}
	return val
}

func (s *sessionLogLevelsStore) set(sessionID string, level mcp.LoggingLevel) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.logs[sessionID] = level
}

func (s *sessionLogLevelsStore) delete(sessionID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.logs, sessionID)
}

type sessionResourcesStore struct {
	mu        sync.RWMutex
	resources map[string]map[string]ServerResource // sessionID -> resourceURI -> resource
}

func newSessionResourcesStore() *sessionResourcesStore {
	return &sessionResourcesStore{
		resources: make(map[string]map[string]ServerResource),
	}
}

func (s *sessionResourcesStore) get(sessionID string) map[string]ServerResource {
	s.mu.RLock()
	defer s.mu.RUnlock()
	cloned := make(map[string]ServerResource, len(s.resources[sessionID]))
	maps.Copy(cloned, s.resources[sessionID])
	return cloned
}

func (s *sessionResourcesStore) set(sessionID string, resources map[string]ServerResource) {
	s.mu.Lock()
	defer s.mu.Unlock()
	cloned := make(map[string]ServerResource, len(resources))
	maps.Copy(cloned, resources)
	s.resources[sessionID] = cloned
}

func (s *sessionResourcesStore) delete(sessionID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.resources, sessionID)
}

type sessionResourceTemplatesStore struct {
	mu        sync.RWMutex
	templates map[string]map[string]ServerResourceTemplate // sessionID -> uriTemplate -> template
}

func newSessionResourceTemplatesStore() *sessionResourceTemplatesStore {
	return &sessionResourceTemplatesStore{
		templates: make(map[string]map[string]ServerResourceTemplate),
	}
}

func (s *sessionResourceTemplatesStore) get(sessionID string) map[string]ServerResourceTemplate {
	s.mu.RLock()
	defer s.mu.RUnlock()
	cloned := make(map[string]ServerResourceTemplate, len(s.templates[sessionID]))
	maps.Copy(cloned, s.templates[sessionID])
	return cloned
}

func (s *sessionResourceTemplatesStore) set(sessionID string, templates map[string]ServerResourceTemplate) {
	s.mu.Lock()
	defer s.mu.Unlock()
	cloned := make(map[string]ServerResourceTemplate, len(templates))
	maps.Copy(cloned, templates)
	s.templates[sessionID] = cloned
}

func (s *sessionResourceTemplatesStore) delete(sessionID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.templates, sessionID)
}

type sessionToolsStore struct {
	mu    sync.RWMutex
	tools map[string]map[string]ServerTool // sessionID -> toolName -> tool
}

func newSessionToolsStore() *sessionToolsStore {
	return &sessionToolsStore{
		tools: make(map[string]map[string]ServerTool),
	}
}

func (s *sessionToolsStore) get(sessionID string) map[string]ServerTool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	cloned := make(map[string]ServerTool, len(s.tools[sessionID]))
	maps.Copy(cloned, s.tools[sessionID])
	return cloned
}

func (s *sessionToolsStore) set(sessionID string, tools map[string]ServerTool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	cloned := make(map[string]ServerTool, len(tools))
	maps.Copy(cloned, tools)
	s.tools[sessionID] = cloned
}

func (s *sessionToolsStore) delete(sessionID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.tools, sessionID)
}

// Sampling support types for HTTP transport
type samplingRequestItem struct {
	requestID int64
	request   mcp.CreateMessageRequest
	response  chan samplingResponseItem
}

type samplingResponseItem struct {
	requestID int64
	result    json.RawMessage
	err       error
}

// Elicitation support types for HTTP transport
type elicitationRequestItem struct {
	requestID int64
	request   mcp.ElicitationRequest
	response  chan samplingResponseItem
}

// Roots support types for HTTP transport
type rootsRequestItem struct {
	requestID int64
	request   mcp.ListRootsRequest
	response  chan samplingResponseItem
}

// streamableHttpSession is a session for streamable-http transport
// When in POST handlers(request/notification), it's ephemeral, and only exists in the life of the request handler.
// When in GET handlers(listening), it's a real session, and will be registered in the MCP server.
type streamableHttpSession struct {
	sessionID           string
	notificationChannel chan mcp.JSONRPCNotification // server -> client notifications
	tools               *sessionToolsStore
	resources           *sessionResourcesStore
	resourceTemplates   *sessionResourceTemplatesStore
	upgradeToSSE        atomic.Bool
	logLevels           *sessionLogLevelsStore
	clientInfo          atomic.Value // stores session-specific client info
	clientCapabilities  atomic.Value // stores session-specific client capabilities

	// Sampling support for bidirectional communication
	samplingRequestChan    chan samplingRequestItem    // server -> client sampling requests
	elicitationRequestChan chan elicitationRequestItem // server -> client elicitation requests
	rootsRequestChan       chan rootsRequestItem       // server -> client list roots requests

	samplingRequests sync.Map     // requestID -> pending sampling request context
	requestIDCounter atomic.Int64 // for generating unique request IDs
}

func newStreamableHttpSession(sessionID string, toolStore *sessionToolsStore, resourcesStore *sessionResourcesStore, templatesStore *sessionResourceTemplatesStore, levels *sessionLogLevelsStore) *streamableHttpSession {
	s := &streamableHttpSession{
		sessionID:              sessionID,
		notificationChannel:    make(chan mcp.JSONRPCNotification, 100),
		tools:                  toolStore,
		resources:              resourcesStore,
		resourceTemplates:      templatesStore,
		logLevels:              levels,
		samplingRequestChan:    make(chan samplingRequestItem, 10),
		elicitationRequestChan: make(chan elicitationRequestItem, 10),
		rootsRequestChan:       make(chan rootsRequestItem, 10),
	}
	return s
}

func (s *streamableHttpSession) SessionID() string {
	return s.sessionID
}

func (s *streamableHttpSession) NotificationChannel() chan<- mcp.JSONRPCNotification {
	return s.notificationChannel
}

func (s *streamableHttpSession) Initialize() {
	// do nothing
	// the session is ephemeral, no real initialized action needed
}

func (s *streamableHttpSession) Initialized() bool {
	// the session is ephemeral, no real initialized action needed
	return true
}

func (s *streamableHttpSession) SetLogLevel(level mcp.LoggingLevel) {
	s.logLevels.set(s.sessionID, level)
}

func (s *streamableHttpSession) GetLogLevel() mcp.LoggingLevel {
	return s.logLevels.get(s.sessionID)
}

var _ ClientSession = (*streamableHttpSession)(nil)

func (s *streamableHttpSession) GetSessionTools() map[string]ServerTool {
	return s.tools.get(s.sessionID)
}

func (s *streamableHttpSession) SetSessionTools(tools map[string]ServerTool) {
	s.tools.set(s.sessionID, tools)
}

func (s *streamableHttpSession) GetSessionResources() map[string]ServerResource {
	return s.resources.get(s.sessionID)
}

func (s *streamableHttpSession) SetSessionResources(resources map[string]ServerResource) {
	s.resources.set(s.sessionID, resources)
}

func (s *streamableHttpSession) GetSessionResourceTemplates() map[string]ServerResourceTemplate {
	return s.resourceTemplates.get(s.sessionID)
}

func (s *streamableHttpSession) SetSessionResourceTemplates(templates map[string]ServerResourceTemplate) {
	s.resourceTemplates.set(s.sessionID, templates)
}

func (s *streamableHttpSession) GetClientInfo() mcp.Implementation {
	if value := s.clientInfo.Load(); value != nil {
		if clientInfo, ok := value.(mcp.Implementation); ok {
			return clientInfo
		}
	}
	return mcp.Implementation{}
}

func (s *streamableHttpSession) SetClientInfo(clientInfo mcp.Implementation) {
	s.clientInfo.Store(clientInfo)
}

func (s *streamableHttpSession) GetClientCapabilities() mcp.ClientCapabilities {
	if value := s.clientCapabilities.Load(); value != nil {
		if clientCapabilities, ok := value.(mcp.ClientCapabilities); ok {
			return clientCapabilities
		}
	}
	return mcp.ClientCapabilities{}
}

func (s *streamableHttpSession) SetClientCapabilities(clientCapabilities mcp.ClientCapabilities) {
	s.clientCapabilities.Store(clientCapabilities)
}

var (
	_ SessionWithTools             = (*streamableHttpSession)(nil)
	_ SessionWithResources         = (*streamableHttpSession)(nil)
	_ SessionWithResourceTemplates = (*streamableHttpSession)(nil)
	_ SessionWithLogging           = (*streamableHttpSession)(nil)
	_ SessionWithClientInfo        = (*streamableHttpSession)(nil)
)

func (s *streamableHttpSession) UpgradeToSSEWhenReceiveNotification() {
	s.upgradeToSSE.Store(true)
}

var _ SessionWithStreamableHTTPConfig = (*streamableHttpSession)(nil)

// RequestSampling implements SessionWithSampling interface for HTTP transport
func (s *streamableHttpSession) RequestSampling(ctx context.Context, request mcp.CreateMessageRequest) (*mcp.CreateMessageResult, error) {
	// Generate unique request ID
	requestID := s.requestIDCounter.Add(1)

	// Create response channel for this specific request
	responseChan := make(chan samplingResponseItem, 1)

	// Create the sampling request item
	samplingRequest := samplingRequestItem{
		requestID: requestID,
		request:   request,
		response:  responseChan,
	}

	// Store the pending request
	s.samplingRequests.Store(requestID, responseChan)
	defer s.samplingRequests.Delete(requestID)

	// Send the sampling request via the channel (non-blocking)
	select {
	case s.samplingRequestChan <- samplingRequest:
		// Request queued successfully
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
		return nil, fmt.Errorf("sampling request queue is full - server overloaded")
	}

	// Wait for response or context cancellation
	select {
	case response := <-responseChan:
		if response.err != nil {
			return nil, response.err
		}
		var result mcp.CreateMessageResult
		if err := json.Unmarshal(response.result, &result); err != nil {
			return nil, fmt.Errorf("failed to unmarshal sampling response: %v", err)
		}

		// Parse content from map[string]any to proper Content type (TextContent, ImageContent, AudioContent)
		// HTTP transport unmarshals Content as map[string]any, we need to convert it to the proper type
		if contentMap, ok := result.Content.(map[string]any); ok {
			content, err := mcp.ParseContent(contentMap)
			if err != nil {
				return nil, fmt.Errorf("failed to parse sampling response content: %w", err)
			}
			result.Content = content
		}

		return &result, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// ListRoots implements SessionWithRoots interface for HTTP transport.
// It sends a list roots request to the client via SSE and waits for the response.
func (s *streamableHttpSession) ListRoots(ctx context.Context, request mcp.ListRootsRequest) (*mcp.ListRootsResult, error) {
	// Generate unique request ID
	requestID := s.requestIDCounter.Add(1)

	// Create response channel for this specific request
	responseChan := make(chan samplingResponseItem, 1)

	// Create the roots request item
	rootsRequest := rootsRequestItem{
		requestID: requestID,
		request:   request,
		response:  responseChan,
	}

	// Store the pending request
	s.samplingRequests.Store(requestID, responseChan)
	defer s.samplingRequests.Delete(requestID)

	// Send the list roots request via the channel (non-blocking)
	select {
	case s.rootsRequestChan <- rootsRequest:
		// Request queued successfully
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
		return nil, fmt.Errorf("list roots request queue is full - server overloaded")
	}

	// Wait for response or context cancellation
	select {
	case response := <-responseChan:
		if response.err != nil {
			return nil, response.err
		}
		var result mcp.ListRootsResult
		if err := json.Unmarshal(response.result, &result); err != nil {
			return nil, fmt.Errorf("failed to unmarshal list roots response: %v", err)
		}
		return &result, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// RequestElicitation implements SessionWithElicitation interface for HTTP transport
func (s *streamableHttpSession) RequestElicitation(ctx context.Context, request mcp.ElicitationRequest) (*mcp.ElicitationResult, error) {
	// Generate unique request ID
	requestID := s.requestIDCounter.Add(1)

	// Create response channel for this specific request
	responseChan := make(chan samplingResponseItem, 1)

	// Create the sampling request item
	elicitationRequest := elicitationRequestItem{
		requestID: requestID,
		request:   request,
		response:  responseChan,
	}

	// Store the pending request
	s.samplingRequests.Store(requestID, responseChan)
	defer s.samplingRequests.Delete(requestID)

	// Send the sampling request via the channel (non-blocking)
	select {
	case s.elicitationRequestChan <- elicitationRequest:
		// Request queued successfully
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
		return nil, fmt.Errorf("elicitation request queue is full - server overloaded")
	}

	// Wait for response or context cancellation
	select {
	case response := <-responseChan:
		if response.err != nil {
			return nil, response.err
		}
		var result mcp.ElicitationResult
		if err := json.Unmarshal(response.result, &result); err != nil {
			return nil, fmt.Errorf("failed to unmarshal elicitation response: %v", err)
		}
		return &result, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

var _ SessionWithSampling = (*streamableHttpSession)(nil)
var _ SessionWithElicitation = (*streamableHttpSession)(nil)
var _ SessionWithRoots = (*streamableHttpSession)(nil)

// --- session id manager ---

// SessionIdManagerResolver resolves a SessionIdManager based on the HTTP request
type SessionIdManagerResolver interface {
	ResolveSessionIdManager(r *http.Request) SessionIdManager
}

type SessionIdManager interface {
	Generate() string
	// Validate checks if a session ID is valid and not terminated.
	// Returns isTerminated=true if the ID is valid but belongs to a terminated session.
	// Returns err!=nil if the ID format is invalid or lookup failed.
	Validate(sessionID string) (isTerminated bool, err error)
	// Terminate marks a session ID as terminated.
	// Returns isNotAllowed=true if the server policy prevents client termination.
	// Returns err!=nil if the ID is invalid or termination failed.
	Terminate(sessionID string) (isNotAllowed bool, err error)
}

// DefaultSessionIdManagerResolver is a simple resolver that returns the same SessionIdManager for all requests
type DefaultSessionIdManagerResolver struct {
	manager SessionIdManager
}

// NewDefaultSessionIdManagerResolver creates a new DefaultSessionIdManagerResolver with the given SessionIdManager
func NewDefaultSessionIdManagerResolver(manager SessionIdManager) *DefaultSessionIdManagerResolver {
	if manager == nil {
		manager = &StatelessSessionIdManager{}
	}
	return &DefaultSessionIdManagerResolver{manager: manager}
}

// ResolveSessionIdManager returns the configured SessionIdManager for all requests
func (r *DefaultSessionIdManagerResolver) ResolveSessionIdManager(_ *http.Request) SessionIdManager {
	return r.manager
}

// StatelessSessionIdManager does nothing, which means it has no session management, which is stateless.
type StatelessSessionIdManager struct{}

func (s *StatelessSessionIdManager) Generate() string {
	return ""
}

func (s *StatelessSessionIdManager) Validate(sessionID string) (isTerminated bool, err error) {
	// In stateless mode, ignore session IDs completely - don't validate or reject them
	return false, nil
}

func (s *StatelessSessionIdManager) Terminate(sessionID string) (isNotAllowed bool, err error) {
	return false, nil
}

// StatelessGeneratingSessionIdManager generates session IDs but doesn't validate them locally.
// This allows session IDs to be generated for clients while working across multiple instances.
type StatelessGeneratingSessionIdManager struct{}

func (s *StatelessGeneratingSessionIdManager) Generate() string {
	return idPrefix + uuid.New().String()
}

func (s *StatelessGeneratingSessionIdManager) Validate(sessionID string) (isTerminated bool, err error) {
	// Only validate format, not existence - allows cross-instance operation
	if !strings.HasPrefix(sessionID, idPrefix) {
		return false, fmt.Errorf("invalid session id: %s", sessionID)
	}
	if _, err := uuid.Parse(sessionID[len(idPrefix):]); err != nil {
		return false, fmt.Errorf("invalid session id: %s", sessionID)
	}
	return false, nil
}

func (s *StatelessGeneratingSessionIdManager) Terminate(sessionID string) (isNotAllowed bool, err error) {
	// No-op termination since we don't track sessions
	return false, nil
}

// InsecureStatefulSessionIdManager generate id with uuid and tracks active sessions.
// It validates both format and existence of session IDs.
// For more secure session id, use a more complex generator, like a JWT.
type InsecureStatefulSessionIdManager struct {
	sessions   sync.Map
	terminated sync.Map
}

const idPrefix = "mcp-session-"

func (s *InsecureStatefulSessionIdManager) Generate() string {
	sessionID := idPrefix + uuid.New().String()
	s.sessions.Store(sessionID, true)
	return sessionID
}

func (s *InsecureStatefulSessionIdManager) Validate(sessionID string) (isTerminated bool, err error) {
	if !strings.HasPrefix(sessionID, idPrefix) {
		return false, fmt.Errorf("invalid session id: %s", sessionID)
	}
	if _, err := uuid.Parse(sessionID[len(idPrefix):]); err != nil {
		return false, fmt.Errorf("invalid session id: %s", sessionID)
	}
	if _, exists := s.terminated.Load(sessionID); exists {
		return true, nil
	}
	if _, exists := s.sessions.Load(sessionID); !exists {
		return false, fmt.Errorf("session not found: %s", sessionID)
	}
	return false, nil
}

func (s *InsecureStatefulSessionIdManager) Terminate(sessionID string) (isNotAllowed bool, err error) {
	if _, exists := s.terminated.Load(sessionID); exists {
		return false, nil
	}
	if _, exists := s.sessions.Load(sessionID); !exists {
		return false, nil
	}
	s.terminated.Store(sessionID, true)
	s.sessions.Delete(sessionID)
	return false, nil
}

// NewTestStreamableHTTPServer creates a test server for testing purposes
func NewTestStreamableHTTPServer(server *MCPServer, opts ...StreamableHTTPOption) *httptest.Server {
	sseServer := NewStreamableHTTPServer(server, opts...)
	testServer := httptest.NewServer(sseServer)
	return testServer
}

// isJSONEmpty reports whether the provided JSON value is "empty":
//   - null
//   - empty object: {}
//   - empty array: []
//
// It also treats nil/whitespace-only input as empty.
// It does NOT treat 0, false, "" or non-empty composites as empty.
func isJSONEmpty(data json.RawMessage) bool {
	if len(data) == 0 {
		return true
	}

	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 {
		return true
	}

	switch trimmed[0] {
	case '{':
		if len(trimmed) == 2 && trimmed[1] == '}' {
			return true
		}
		for i := 1; i < len(trimmed); i++ {
			if !unicode.IsSpace(rune(trimmed[i])) {
				return trimmed[i] == '}'
			}
		}
	case '[':
		if len(trimmed) == 2 && trimmed[1] == ']' {
			return true
		}
		for i := 1; i < len(trimmed); i++ {
			if !unicode.IsSpace(rune(trimmed[i])) {
				return trimmed[i] == ']'
			}
		}

	case '"': // treat "" as not empty
		return false

	case 'n': // null
		return len(trimmed) == 4 &&
			trimmed[1] == 'u' &&
			trimmed[2] == 'l' &&
			trimmed[3] == 'l'
	}
	return false
}
