package server

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"

	"github.com/mark3labs/mcp-go/mcp"
)

// StdioContextFunc is a function that takes an existing context and returns
// a potentially modified context.
// This can be used to inject context values from environment variables,
// for example.
type StdioContextFunc func(ctx context.Context) context.Context

// StdioServer wraps a MCPServer and handles stdio communication.
// It provides a simple way to create command-line MCP servers that
// communicate via standard input/output streams using JSON-RPC messages.
type StdioServer struct {
	server      *MCPServer
	errLogger   *log.Logger
	contextFunc StdioContextFunc

	// Thread-safe tool call processing
	toolCallQueue  chan *toolCallWork
	workerWg       sync.WaitGroup
	workerPoolSize int
	queueSize      int
	writeMu        sync.Mutex // Protects concurrent writes
}

// toolCallWork represents a queued tool call request
type toolCallWork struct {
	ctx     context.Context
	message json.RawMessage
	writer  io.Writer
}

// StdioOption defines a function type for configuring StdioServer
type StdioOption func(*StdioServer)

// WithErrorLogger sets the error logger for the server
func WithErrorLogger(logger *log.Logger) StdioOption {
	return func(s *StdioServer) {
		s.errLogger = logger
	}
}

// WithStdioContextFunc sets a function that will be called to customise the context
// to the server. Note that the stdio server uses the same context for all requests,
// so this function will only be called once per server instance.
func WithStdioContextFunc(fn StdioContextFunc) StdioOption {
	return func(s *StdioServer) {
		s.contextFunc = fn
	}
}

// WithWorkerPoolSize sets the number of workers for processing tool calls
func WithWorkerPoolSize(size int) StdioOption {
	return func(s *StdioServer) {
		const maxWorkerPoolSize = 100
		if size > 0 && size <= maxWorkerPoolSize {
			s.workerPoolSize = size
		} else if size > maxWorkerPoolSize {
			s.errLogger.Printf("Worker pool size %d exceeds maximum (%d), using maximum", size, maxWorkerPoolSize)
			s.workerPoolSize = maxWorkerPoolSize
		}
	}
}

// WithQueueSize sets the size of the tool call queue
func WithQueueSize(size int) StdioOption {
	return func(s *StdioServer) {
		const maxQueueSize = 10000
		if size > 0 && size <= maxQueueSize {
			s.queueSize = size
		} else if size > maxQueueSize {
			s.errLogger.Printf("Queue size %d exceeds maximum (%d), using maximum", size, maxQueueSize)
			s.queueSize = maxQueueSize
		}
	}
}

// stdioSession is a static client session, since stdio has only one client.
type stdioSession struct {
	notifications       chan mcp.JSONRPCNotification
	initialized         atomic.Bool
	loggingLevel        atomic.Value
	clientInfo          atomic.Value                        // stores session-specific client info
	clientCapabilities  atomic.Value                        // stores session-specific client capabilities
	writer              io.Writer                           // for sending requests to client
	requestID           atomic.Int64                        // for generating unique request IDs
	mu                  sync.RWMutex                        // protects writer
	pendingRequests     map[int64]chan *samplingResponse    // for tracking pending sampling requests
	pendingElicitations map[int64]chan *elicitationResponse // for tracking pending elicitation requests
	pendingRoots        map[int64]chan *rootsResponse       // for tracking pending list roots requests
	pendingMu           sync.RWMutex                        // protects pendingRequests and pendingElicitations
}

// samplingResponse represents a response to a sampling request
type samplingResponse struct {
	result *mcp.CreateMessageResult
	err    error
}

// elicitationResponse represents a response to an elicitation request
type elicitationResponse struct {
	result *mcp.ElicitationResult
	err    error
}

// rootsResponse represents a response to an list root request
type rootsResponse struct {
	result *mcp.ListRootsResult
	err    error
}

func (s *stdioSession) SessionID() string {
	return "stdio"
}

func (s *stdioSession) NotificationChannel() chan<- mcp.JSONRPCNotification {
	return s.notifications
}

func (s *stdioSession) Initialize() {
	// set default logging level
	s.loggingLevel.Store(mcp.LoggingLevelError)
	s.initialized.Store(true)
}

func (s *stdioSession) Initialized() bool {
	return s.initialized.Load()
}

func (s *stdioSession) GetClientInfo() mcp.Implementation {
	if value := s.clientInfo.Load(); value != nil {
		if clientInfo, ok := value.(mcp.Implementation); ok {
			return clientInfo
		}
	}
	return mcp.Implementation{}
}

func (s *stdioSession) SetClientInfo(clientInfo mcp.Implementation) {
	s.clientInfo.Store(clientInfo)
}

func (s *stdioSession) GetClientCapabilities() mcp.ClientCapabilities {
	if value := s.clientCapabilities.Load(); value != nil {
		if clientCapabilities, ok := value.(mcp.ClientCapabilities); ok {
			return clientCapabilities
		}
	}
	return mcp.ClientCapabilities{}
}

func (s *stdioSession) SetClientCapabilities(clientCapabilities mcp.ClientCapabilities) {
	s.clientCapabilities.Store(clientCapabilities)
}

func (s *stdioSession) SetLogLevel(level mcp.LoggingLevel) {
	s.loggingLevel.Store(level)
}

func (s *stdioSession) GetLogLevel() mcp.LoggingLevel {
	level := s.loggingLevel.Load()
	if level == nil {
		return mcp.LoggingLevelError
	}
	return level.(mcp.LoggingLevel)
}

// RequestSampling sends a sampling request to the client and waits for the response.
func (s *stdioSession) RequestSampling(ctx context.Context, request mcp.CreateMessageRequest) (*mcp.CreateMessageResult, error) {
	s.mu.RLock()
	writer := s.writer
	s.mu.RUnlock()

	if writer == nil {
		return nil, fmt.Errorf("no writer available for sending requests")
	}

	// Generate a unique request ID
	id := s.requestID.Add(1)

	// Create a response channel for this request
	responseChan := make(chan *samplingResponse, 1)
	s.pendingMu.Lock()
	s.pendingRequests[id] = responseChan
	s.pendingMu.Unlock()

	// Cleanup function to remove the pending request
	cleanup := func() {
		s.pendingMu.Lock()
		delete(s.pendingRequests, id)
		s.pendingMu.Unlock()
	}
	defer cleanup()

	// Create the JSON-RPC request
	jsonRPCRequest := struct {
		JSONRPC string                  `json:"jsonrpc"`
		ID      int64                   `json:"id"`
		Method  string                  `json:"method"`
		Params  mcp.CreateMessageParams `json:"params"`
	}{
		JSONRPC: mcp.JSONRPC_VERSION,
		ID:      id,
		Method:  string(mcp.MethodSamplingCreateMessage),
		Params:  request.CreateMessageParams,
	}

	// Marshal and send the request
	requestBytes, err := json.Marshal(jsonRPCRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal sampling request: %w", err)
	}
	requestBytes = append(requestBytes, '\n')

	if _, err := writer.Write(requestBytes); err != nil {
		return nil, fmt.Errorf("failed to write sampling request: %w", err)
	}

	// Wait for the response or context cancellation
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case response := <-responseChan:
		if response.err != nil {
			return nil, response.err
		}
		return response.result, nil
	}
}

// ListRoots sends an list roots request to the client and waits for the response.
func (s *stdioSession) ListRoots(ctx context.Context, request mcp.ListRootsRequest) (*mcp.ListRootsResult, error) {
	s.mu.RLock()
	writer := s.writer
	s.mu.RUnlock()

	if writer == nil {
		return nil, fmt.Errorf("no writer available for sending requests")
	}

	// Generate a unique request ID
	id := s.requestID.Add(1)

	// Create a response channel for this request
	responseChan := make(chan *rootsResponse, 1)
	s.pendingMu.Lock()
	s.pendingRoots[id] = responseChan
	s.pendingMu.Unlock()

	// Cleanup function to remove the pending request
	cleanup := func() {
		s.pendingMu.Lock()
		delete(s.pendingRoots, id)
		s.pendingMu.Unlock()
	}
	defer cleanup()

	// Create the JSON-RPC request
	jsonRPCRequest := struct {
		JSONRPC string `json:"jsonrpc"`
		ID      int64  `json:"id"`
		Method  string `json:"method"`
	}{
		JSONRPC: mcp.JSONRPC_VERSION,
		ID:      id,
		Method:  string(mcp.MethodListRoots),
	}

	// Marshal and send the request
	requestBytes, err := json.Marshal(jsonRPCRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal list roots request: %w", err)
	}
	requestBytes = append(requestBytes, '\n')

	if _, err := writer.Write(requestBytes); err != nil {
		return nil, fmt.Errorf("failed to write list roots request: %w", err)
	}

	// Wait for the response or context cancellation
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case response := <-responseChan:
		if response.err != nil {
			return nil, response.err
		}
		return response.result, nil
	}
}

// RequestElicitation sends an elicitation request to the client and waits for the response.
func (s *stdioSession) RequestElicitation(ctx context.Context, request mcp.ElicitationRequest) (*mcp.ElicitationResult, error) {
	s.mu.RLock()
	writer := s.writer
	s.mu.RUnlock()

	if writer == nil {
		return nil, fmt.Errorf("no writer available for sending requests")
	}

	// Generate a unique request ID
	id := s.requestID.Add(1)

	// Create a response channel for this request
	responseChan := make(chan *elicitationResponse, 1)
	s.pendingMu.Lock()
	s.pendingElicitations[id] = responseChan
	s.pendingMu.Unlock()

	// Cleanup function to remove the pending request
	cleanup := func() {
		s.pendingMu.Lock()
		delete(s.pendingElicitations, id)
		s.pendingMu.Unlock()
	}
	defer cleanup()

	// Create the JSON-RPC request
	jsonRPCRequest := struct {
		JSONRPC string                `json:"jsonrpc"`
		ID      int64                 `json:"id"`
		Method  string                `json:"method"`
		Params  mcp.ElicitationParams `json:"params"`
	}{
		JSONRPC: mcp.JSONRPC_VERSION,
		ID:      id,
		Method:  string(mcp.MethodElicitationCreate),
		Params:  request.Params,
	}

	// Marshal and send the request
	requestBytes, err := json.Marshal(jsonRPCRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal elicitation request: %w", err)
	}
	requestBytes = append(requestBytes, '\n')

	if _, err := writer.Write(requestBytes); err != nil {
		return nil, fmt.Errorf("failed to write elicitation request: %w", err)
	}

	// Wait for the response or context cancellation
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case response := <-responseChan:
		if response.err != nil {
			return nil, response.err
		}
		return response.result, nil
	}
}

// SetWriter sets the writer for sending requests to the client.
func (s *stdioSession) SetWriter(writer io.Writer) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.writer = writer
}

var (
	_ ClientSession          = (*stdioSession)(nil)
	_ SessionWithLogging     = (*stdioSession)(nil)
	_ SessionWithClientInfo  = (*stdioSession)(nil)
	_ SessionWithSampling    = (*stdioSession)(nil)
	_ SessionWithElicitation = (*stdioSession)(nil)
	_ SessionWithRoots       = (*stdioSession)(nil)
)

var stdioSessionInstance = stdioSession{
	notifications:       make(chan mcp.JSONRPCNotification, 100),
	pendingRequests:     make(map[int64]chan *samplingResponse),
	pendingElicitations: make(map[int64]chan *elicitationResponse),
	pendingRoots:        make(map[int64]chan *rootsResponse),
}

// NewStdioServer creates a new stdio server wrapper around an MCPServer.
// It initializes the server with a default error logger that discards all output.
func NewStdioServer(server *MCPServer) *StdioServer {
	return &StdioServer{
		server: server,
		errLogger: log.New(
			os.Stderr,
			"",
			log.LstdFlags,
		), // Default to discarding logs
		workerPoolSize: 5,   // Default worker pool size
		queueSize:      100, // Default queue size
	}
}

// SetErrorLogger configures where error messages from the StdioServer are logged.
// The provided logger will receive all error messages generated during server operation.
func (s *StdioServer) SetErrorLogger(logger *log.Logger) {
	s.errLogger = logger
}

// SetContextFunc sets a function that will be called to customise the context
// to the server. Note that the stdio server uses the same context for all requests,
// so this function will only be called once per server instance.
func (s *StdioServer) SetContextFunc(fn StdioContextFunc) {
	s.contextFunc = fn
}

// handleNotifications continuously processes notifications from the session's notification channel
// and writes them to the provided output. It runs until the context is cancelled.
// Any errors encountered while writing notifications are logged but do not stop the handler.
func (s *StdioServer) handleNotifications(ctx context.Context, stdout io.Writer) {
	for {
		select {
		case notification := <-stdioSessionInstance.notifications:
			if err := s.writeResponse(notification, stdout); err != nil {
				s.errLogger.Printf("Error writing notification: %v", err)
			}
		case <-ctx.Done():
			return
		}
	}
}

// processInputStream continuously reads and processes messages from the input stream.
// It handles EOF gracefully as a normal termination condition.
// The function returns when either:
// - The context is cancelled (returns context.Err())
// - EOF is encountered (returns nil)
// - An error occurs while reading or processing messages (returns the error)
func (s *StdioServer) processInputStream(ctx context.Context, reader *bufio.Reader, stdout io.Writer) error {
	for {
		if err := ctx.Err(); err != nil {
			return err
		}

		line, err := s.readNextLine(ctx, reader)
		if err != nil {
			if err == io.EOF {
				return nil
			}
			s.errLogger.Printf("Error reading input: %v", err)
			return err
		}

		if err := s.processMessage(ctx, line, stdout); err != nil {
			if err == io.EOF {
				return nil
			}
			s.errLogger.Printf("Error handling message: %v", err)
			return err
		}
	}
}

// toolCallWorker processes tool calls from the queue
func (s *StdioServer) toolCallWorker(ctx context.Context) {
	defer s.workerWg.Done()

	for {
		select {
		case work, ok := <-s.toolCallQueue:
			if !ok {
				// Channel closed, exit worker
				return
			}
			// Process the tool call
			response := s.server.HandleMessage(work.ctx, work.message)
			if response != nil {
				if err := s.writeResponse(response, work.writer); err != nil {
					s.errLogger.Printf("Error writing tool response: %v", err)
				}
			}
		case <-ctx.Done():
			return
		}
	}
}

// readNextLine reads a single line from the input reader in a context-aware manner.
// It uses channels to make the read operation cancellable via context.
// Returns the read line and any error encountered. If the context is cancelled,
// returns an empty string and the context's error. EOF is returned when the input
// stream is closed.
func (s *StdioServer) readNextLine(ctx context.Context, reader *bufio.Reader) (string, error) {
	type result struct {
		line string
		err  error
	}

	resultCh := make(chan result, 1)

	go func() {
		line, err := reader.ReadString('\n')
		resultCh <- result{line: line, err: err}
	}()

	select {
	case <-ctx.Done():
		return "", nil
	case res := <-resultCh:
		return res.line, res.err
	}
}

// Listen starts listening for JSON-RPC messages on the provided input and writes responses to the provided output.
// It runs until the context is cancelled or an error occurs.
// Returns an error if there are issues with reading input or writing output.
func (s *StdioServer) Listen(
	ctx context.Context,
	stdin io.Reader,
	stdout io.Writer,
) error {
	// Initialize the tool call queue
	s.toolCallQueue = make(chan *toolCallWork, s.queueSize)

	// Set a static client context since stdio only has one client
	if err := s.server.RegisterSession(ctx, &stdioSessionInstance); err != nil {
		return fmt.Errorf("register session: %w", err)
	}
	defer s.server.UnregisterSession(ctx, stdioSessionInstance.SessionID())
	ctx = s.server.WithContext(ctx, &stdioSessionInstance)

	// Set the writer for sending requests to the client
	stdioSessionInstance.SetWriter(stdout)

	// Add in any custom context.
	if s.contextFunc != nil {
		ctx = s.contextFunc(ctx)
	}

	reader := bufio.NewReader(stdin)

	// Start worker pool for tool calls
	for i := 0; i < s.workerPoolSize; i++ {
		s.workerWg.Add(1)
		go s.toolCallWorker(ctx)
	}

	// Start notification handler
	go s.handleNotifications(ctx, stdout)

	// Process input stream
	err := s.processInputStream(ctx, reader, stdout)

	// Shutdown workers gracefully
	close(s.toolCallQueue)
	s.workerWg.Wait()

	return err
}

// processMessage handles a single JSON-RPC message and writes the response.
// It parses the message, processes it through the wrapped MCPServer, and writes any response.
// Returns an error if there are issues with message processing or response writing.
func (s *StdioServer) processMessage(
	ctx context.Context,
	line string,
	writer io.Writer,
) error {
	// If line is empty, likely due to ctx cancellation
	if len(line) == 0 {
		return nil
	}

	// Parse the message as raw JSON
	var rawMessage json.RawMessage
	if err := json.Unmarshal([]byte(line), &rawMessage); err != nil {
		response := createErrorResponse(nil, mcp.PARSE_ERROR, "Parse error")
		return s.writeResponse(response, writer)
	}

	// Check if this is a response to a sampling request
	if s.handleSamplingResponse(rawMessage) {
		return nil
	}

	// Check if this is a response to an elicitation request
	if s.handleElicitationResponse(rawMessage) {
		return nil
	}

	// Check if this is a response to an list roots request
	if s.handleListRootsResponse(rawMessage) {
		return nil
	}

	// Check if this is a tool call that might need sampling (and thus should be processed concurrently)
	var baseMessage struct {
		Method string `json:"method"`
	}
	if json.Unmarshal(rawMessage, &baseMessage) == nil && baseMessage.Method == "tools/call" {
		// Queue tool calls for processing by workers
		select {
		case s.toolCallQueue <- &toolCallWork{
			ctx:     ctx,
			message: rawMessage,
			writer:  writer,
		}:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		default:
			// Queue is full, process synchronously as fallback
			s.errLogger.Printf("Tool call queue full, processing synchronously")
			response := s.server.HandleMessage(ctx, rawMessage)
			if response != nil {
				return s.writeResponse(response, writer)
			}
			return nil
		}
	}

	// Handle other messages synchronously
	response := s.server.HandleMessage(ctx, rawMessage)

	// Only write response if there is one (not for notifications)
	if response != nil {
		if err := s.writeResponse(response, writer); err != nil {
			return fmt.Errorf("failed to write response: %w", err)
		}
	}

	return nil
}

// handleSamplingResponse checks if the message is a response to a sampling request
// and routes it to the appropriate pending request channel.
func (s *StdioServer) handleSamplingResponse(rawMessage json.RawMessage) bool {
	return stdioSessionInstance.handleSamplingResponse(rawMessage)
}

// handleSamplingResponse handles incoming sampling responses for this session
func (s *stdioSession) handleSamplingResponse(rawMessage json.RawMessage) bool {
	// Try to parse as a JSON-RPC response
	var response struct {
		JSONRPC string          `json:"jsonrpc"`
		ID      json.Number     `json:"id"`
		Result  json.RawMessage `json:"result,omitempty"`
		Error   *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error,omitempty"`
	}

	if err := json.Unmarshal(rawMessage, &response); err != nil {
		return false
	}
	// Parse the ID as int64
	idInt64, err := response.ID.Int64()
	if err != nil || (response.Result == nil && response.Error == nil) {
		return false
	}

	// Look for a pending request with this ID
	s.pendingMu.RLock()
	responseChan, exists := s.pendingRequests[idInt64]
	s.pendingMu.RUnlock()

	if !exists {
		return false
	} // Parse and send the response
	samplingResp := &samplingResponse{}

	if response.Error != nil {
		samplingResp.err = fmt.Errorf("sampling request failed: %s", response.Error.Message)
	} else {
		var result mcp.CreateMessageResult
		if err := json.Unmarshal(response.Result, &result); err != nil {
			samplingResp.err = fmt.Errorf("failed to unmarshal sampling response: %w", err)
		} else {
			// Parse content from map[string]any to proper Content type (TextContent, ImageContent, AudioContent)
			if contentMap, ok := result.Content.(map[string]any); ok {
				content, err := mcp.ParseContent(contentMap)
				if err != nil {
					samplingResp.err = fmt.Errorf("failed to parse sampling response content: %w", err)
				} else {
					result.Content = content
					samplingResp.result = &result
				}
			} else {
				samplingResp.result = &result
			}
		}
	}

	// Send the response (non-blocking)
	select {
	case responseChan <- samplingResp:
	default:
		// Channel is full or closed, ignore
	}

	return true
}

// handleElicitationResponse checks if the message is a response to an elicitation request
// and routes it to the appropriate pending request channel.
func (s *StdioServer) handleElicitationResponse(rawMessage json.RawMessage) bool {
	return stdioSessionInstance.handleElicitationResponse(rawMessage)
}

// handleElicitationResponse handles incoming elicitation responses for this session
func (s *stdioSession) handleElicitationResponse(rawMessage json.RawMessage) bool {
	// Try to parse as a JSON-RPC response
	var response struct {
		JSONRPC string          `json:"jsonrpc"`
		ID      json.Number     `json:"id"`
		Result  json.RawMessage `json:"result,omitempty"`
		Error   *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error,omitempty"`
	}

	if err := json.Unmarshal(rawMessage, &response); err != nil {
		return false
	}
	// Parse the ID as int64
	id, err := response.ID.Int64()
	if err != nil || (response.Result == nil && response.Error == nil) {
		return false
	}

	// Check if we have a pending elicitation request with this ID
	s.pendingMu.RLock()
	responseChan, exists := s.pendingElicitations[id]
	s.pendingMu.RUnlock()

	if !exists {
		return false
	}

	// Parse and send the response
	elicitationResp := &elicitationResponse{}

	if response.Error != nil {
		elicitationResp.err = fmt.Errorf("elicitation request failed: %s", response.Error.Message)
	} else {
		var result mcp.ElicitationResult
		if err := json.Unmarshal(response.Result, &result); err != nil {
			elicitationResp.err = fmt.Errorf("failed to unmarshal elicitation response: %w", err)
		} else {
			elicitationResp.result = &result
		}
	}

	// Send the response (non-blocking)
	select {
	case responseChan <- elicitationResp:
	default:
		// Channel is full or closed, ignore
	}

	return true
}

// handleListRootsResponse checks if the message is a response to an list roots request
// and routes it to the appropriate pending request channel.
func (s *StdioServer) handleListRootsResponse(rawMessage json.RawMessage) bool {
	return stdioSessionInstance.handleListRootsResponse(rawMessage)
}

// handleListRootsResponse handles incoming list root responses for this session
func (s *stdioSession) handleListRootsResponse(rawMessage json.RawMessage) bool {
	// Try to parse as a JSON-RPC response
	var response struct {
		JSONRPC string          `json:"jsonrpc"`
		ID      json.Number     `json:"id"`
		Result  json.RawMessage `json:"result,omitempty"`
		Error   *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error,omitempty"`
	}

	if err := json.Unmarshal(rawMessage, &response); err != nil {
		return false
	}
	// Parse the ID as int64
	id, err := response.ID.Int64()
	if err != nil || (response.Result == nil && response.Error == nil) {
		return false
	}

	// Check if we have a pending list root request with this ID
	s.pendingMu.RLock()
	responseChan, exists := s.pendingRoots[id]
	s.pendingMu.RUnlock()

	if !exists {
		return false
	}

	// Parse and send the response
	rootsResp := &rootsResponse{}

	if response.Error != nil {
		rootsResp.err = fmt.Errorf("list root request failed: %s", response.Error.Message)
	} else {
		var result mcp.ListRootsResult
		if err := json.Unmarshal(response.Result, &result); err != nil {
			rootsResp.err = fmt.Errorf("failed to unmarshal list root response: %w", err)
		} else {
			rootsResp.result = &result
		}
	}

	// Send the response (non-blocking)
	select {
	case responseChan <- rootsResp:
	default:
		// Channel is full or closed, ignore
	}

	return true
}

// writeResponse marshals and writes a JSON-RPC response message followed by a newline.
// Returns an error if marshaling or writing fails.
func (s *StdioServer) writeResponse(
	response mcp.JSONRPCMessage,
	writer io.Writer,
) error {
	responseBytes, err := json.Marshal(response)
	if err != nil {
		return err
	}

	// Protect concurrent writes
	s.writeMu.Lock()
	defer s.writeMu.Unlock()

	// Write response followed by newline
	if _, err := fmt.Fprintf(writer, "%s\n", responseBytes); err != nil {
		return err
	}

	return nil
}

// ServeStdio is a convenience function that creates and starts a StdioServer with os.Stdin and os.Stdout.
// It sets up signal handling for graceful shutdown on SIGTERM and SIGINT.
// Returns an error if the server encounters any issues during operation.
func ServeStdio(server *MCPServer, opts ...StdioOption) error {
	s := NewStdioServer(server)

	for _, opt := range opts {
		opt(s)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Set up signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)

	go func() {
		<-sigChan
		cancel()
	}()

	return s.Listen(ctx, os.Stdin, os.Stdout)
}
