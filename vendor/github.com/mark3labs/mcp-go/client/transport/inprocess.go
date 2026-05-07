package transport

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

type InProcessTransport struct {
	server             *server.MCPServer
	samplingHandler    server.SamplingHandler
	elicitationHandler server.ElicitationHandler
	rootsHandler       server.RootsHandler
	session            *server.InProcessSession
	sessionID          string

	onNotification func(mcp.JSONRPCNotification)
	notifyMu       sync.RWMutex
	started        bool
	startedMu      sync.Mutex
}

type InProcessOption func(*InProcessTransport)

func WithSamplingHandler(handler server.SamplingHandler) InProcessOption {
	return func(t *InProcessTransport) {
		t.samplingHandler = handler
	}
}

func WithElicitationHandler(handler server.ElicitationHandler) InProcessOption {
	return func(t *InProcessTransport) {
		t.elicitationHandler = handler
	}
}

func WithRootsHandler(handler server.RootsHandler) InProcessOption {
	return func(t *InProcessTransport) {
		t.rootsHandler = handler
	}
}

func NewInProcessTransport(server *server.MCPServer) *InProcessTransport {
	return &InProcessTransport{
		server: server,
	}
}

func NewInProcessTransportWithOptions(server *server.MCPServer, opts ...InProcessOption) *InProcessTransport {
	t := &InProcessTransport{
		server:    server,
		sessionID: server.GenerateInProcessSessionID(),
	}

	for _, opt := range opts {
		opt(t)
	}

	return t
}

func (c *InProcessTransport) Start(ctx context.Context) error {
	c.startedMu.Lock()
	if c.started {
		c.startedMu.Unlock()
		return nil
	}
	c.started = true
	c.startedMu.Unlock()

	// Create and register session if we have handlers
	if c.samplingHandler != nil || c.elicitationHandler != nil || c.rootsHandler != nil {
		c.session = server.NewInProcessSessionWithHandlers(c.sessionID, c.samplingHandler, c.elicitationHandler, c.rootsHandler)
		if err := c.server.RegisterSession(ctx, c.session); err != nil {
			c.startedMu.Lock()
			c.started = false
			c.startedMu.Unlock()
			return fmt.Errorf("failed to register session: %w", err)
		}
	}
	return nil
}

func (c *InProcessTransport) SendRequest(ctx context.Context, request JSONRPCRequest) (*JSONRPCResponse, error) {
	requestBytes, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}
	requestBytes = append(requestBytes, '\n')

	// Add session to context if available
	if c.session != nil {
		ctx = c.server.WithContext(ctx, c.session)
	}

	respMessage := c.server.HandleMessage(ctx, requestBytes)
	respByte, err := json.Marshal(respMessage)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal response message: %w", err)
	}
	var rpcResp JSONRPCResponse
	err = json.Unmarshal(respByte, &rpcResp)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal response message: %w", err)
	}

	return &rpcResp, nil
}

func (c *InProcessTransport) SendNotification(ctx context.Context, notification mcp.JSONRPCNotification) error {
	notificationBytes, err := json.Marshal(notification)
	if err != nil {
		return fmt.Errorf("failed to marshal notification: %w", err)
	}
	notificationBytes = append(notificationBytes, '\n')
	c.server.HandleMessage(ctx, notificationBytes)

	return nil
}

func (c *InProcessTransport) SetNotificationHandler(handler func(notification mcp.JSONRPCNotification)) {
	c.notifyMu.Lock()
	defer c.notifyMu.Unlock()
	c.onNotification = handler
}

func (c *InProcessTransport) Close() error {
	if c.session != nil {
		c.server.UnregisterSession(context.Background(), c.sessionID)
	}
	return nil
}

func (c *InProcessTransport) GetSessionId() string {
	return ""
}
