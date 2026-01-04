package server

import (
	"context"
	"fmt"
	"net/url"

	"github.com/mark3labs/mcp-go/mcp"
)

// ClientSession represents an active session that can be used by MCPServer to interact with client.
type ClientSession interface {
	// Initialize marks session as fully initialized and ready for notifications
	Initialize()
	// Initialized returns if session is ready to accept notifications
	Initialized() bool
	// NotificationChannel provides a channel suitable for sending notifications to client.
	NotificationChannel() chan<- mcp.JSONRPCNotification
	// SessionID is a unique identifier used to track user session.
	SessionID() string
}

// SessionWithLogging is an extension of ClientSession that can receive log message notifications and set log level
type SessionWithLogging interface {
	ClientSession
	// SetLogLevel sets the minimum log level
	SetLogLevel(level mcp.LoggingLevel)
	// GetLogLevel retrieves the minimum log level
	GetLogLevel() mcp.LoggingLevel
}

// SessionWithTools is an extension of ClientSession that can store session-specific tool data
type SessionWithTools interface {
	ClientSession
	// GetSessionTools returns the tools specific to this session, if any
	// This method must be thread-safe for concurrent access
	GetSessionTools() map[string]ServerTool
	// SetSessionTools sets tools specific to this session
	// This method must be thread-safe for concurrent access
	SetSessionTools(tools map[string]ServerTool)
}

// SessionWithResources is an extension of ClientSession that can store session-specific resource data
type SessionWithResources interface {
	ClientSession
	// GetSessionResources returns the resources specific to this session, if any
	// This method must be thread-safe for concurrent access
	GetSessionResources() map[string]ServerResource
	// SetSessionResources sets resources specific to this session
	// This method must be thread-safe for concurrent access
	SetSessionResources(resources map[string]ServerResource)
}

// SessionWithResourceTemplates is an extension of ClientSession that can store session-specific resource template data
type SessionWithResourceTemplates interface {
	ClientSession
	// GetSessionResourceTemplates returns the resource templates specific to this session, if any
	// This method must be thread-safe for concurrent access
	GetSessionResourceTemplates() map[string]ServerResourceTemplate
	// SetSessionResourceTemplates sets resource templates specific to this session
	// This method must be thread-safe for concurrent access
	SetSessionResourceTemplates(templates map[string]ServerResourceTemplate)
}

// SessionWithClientInfo is an extension of ClientSession that can store client info
type SessionWithClientInfo interface {
	ClientSession
	// GetClientInfo returns the client information for this session
	GetClientInfo() mcp.Implementation
	// SetClientInfo sets the client information for this session
	SetClientInfo(clientInfo mcp.Implementation)
	// GetClientCapabilities returns the client capabilities for this session
	GetClientCapabilities() mcp.ClientCapabilities
	// SetClientCapabilities sets the client capabilities for this session
	SetClientCapabilities(clientCapabilities mcp.ClientCapabilities)
}

// SessionWithElicitation is an extension of ClientSession that can send elicitation requests
type SessionWithElicitation interface {
	ClientSession
	// RequestElicitation sends an elicitation request to the client and waits for response
	RequestElicitation(ctx context.Context, request mcp.ElicitationRequest) (*mcp.ElicitationResult, error)
}

// SessionWithRoots is an extension of ClientSession that can send list roots requests
type SessionWithRoots interface {
	ClientSession
	// ListRoots sends an list roots request to the client and waits for response
	ListRoots(ctx context.Context, request mcp.ListRootsRequest) (*mcp.ListRootsResult, error)
}

// SessionWithStreamableHTTPConfig extends ClientSession to support streamable HTTP transport configurations
type SessionWithStreamableHTTPConfig interface {
	ClientSession
	// UpgradeToSSEWhenReceiveNotification upgrades the client-server communication to SSE stream when the server
	// sends notifications to the client
	//
	// The protocol specification:
	// - If the server response contains any JSON-RPC notifications, it MUST either:
	//   - Return Content-Type: text/event-stream to initiate an SSE stream, OR
	//   - Return Content-Type: application/json for a single JSON object
	// - The client MUST support both response types.
	//
	// Reference: https://modelcontextprotocol.io/specification/2025-03-26/basic/transports#sending-messages-to-the-server
	UpgradeToSSEWhenReceiveNotification()
}

// clientSessionKey is the context key for storing current client notification channel.
type clientSessionKey struct{}

// ClientSessionFromContext retrieves current client notification context from context.
func ClientSessionFromContext(ctx context.Context) ClientSession {
	if session, ok := ctx.Value(clientSessionKey{}).(ClientSession); ok {
		return session
	}
	return nil
}

// WithContext sets the current client session and returns the provided context
func (s *MCPServer) WithContext(
	ctx context.Context,
	session ClientSession,
) context.Context {
	return context.WithValue(ctx, clientSessionKey{}, session)
}

// RegisterSession saves session that should be notified in case if some server attributes changed.
func (s *MCPServer) RegisterSession(
	ctx context.Context,
	session ClientSession,
) error {
	sessionID := session.SessionID()
	if _, exists := s.sessions.LoadOrStore(sessionID, session); exists {
		return ErrSessionExists
	}
	s.hooks.RegisterSession(ctx, session)
	return nil
}

func (s *MCPServer) buildLogNotification(notification mcp.LoggingMessageNotification) mcp.JSONRPCNotification {
	return mcp.JSONRPCNotification{
		JSONRPC: mcp.JSONRPC_VERSION,
		Notification: mcp.Notification{
			Method: notification.Method,
			Params: mcp.NotificationParams{
				AdditionalFields: map[string]any{
					"level":  notification.Params.Level,
					"logger": notification.Params.Logger,
					"data":   notification.Params.Data,
				},
			},
		},
	}
}

func (s *MCPServer) SendLogMessageToClient(ctx context.Context, notification mcp.LoggingMessageNotification) error {
	session := ClientSessionFromContext(ctx)
	if session == nil || !session.Initialized() {
		return ErrNotificationNotInitialized
	}
	sessionLogging, ok := session.(SessionWithLogging)
	if !ok {
		return ErrSessionDoesNotSupportLogging
	}
	if !notification.Params.Level.ShouldSendTo(sessionLogging.GetLogLevel()) {
		return nil
	}
	return s.sendNotificationCore(ctx, session, s.buildLogNotification(notification))
}

func (s *MCPServer) sendNotificationToAllClients(notification mcp.JSONRPCNotification) {
	s.sessions.Range(func(k, v any) bool {
		if session, ok := v.(ClientSession); ok && session.Initialized() {
			if sessionWithStreamableHTTPConfig, ok := session.(SessionWithStreamableHTTPConfig); ok {
				sessionWithStreamableHTTPConfig.UpgradeToSSEWhenReceiveNotification()
			}
			select {
			case session.NotificationChannel() <- notification:
				// Successfully sent notification
			default:
				// Channel is blocked, if there's an error hook, use it
				if s.hooks != nil && len(s.hooks.OnError) > 0 {
					err := ErrNotificationChannelBlocked
					// Copy hooks pointer to local variable to avoid race condition
					hooks := s.hooks
					go func(sessionID string, hooks *Hooks) {
						ctx := context.Background()
						// Use the error hook to report the blocked channel
						hooks.onError(ctx, nil, "notification", map[string]any{
							"method":    notification.Method,
							"sessionID": sessionID,
						}, fmt.Errorf("notification channel blocked for session %s: %w", sessionID, err))
					}(session.SessionID(), hooks)
				}
			}
		}
		return true
	})
}

func (s *MCPServer) sendNotificationToSpecificClient(session ClientSession, notification mcp.JSONRPCNotification) error {
	// upgrades the client-server communication to SSE stream when the server sends notifications to the client
	if sessionWithStreamableHTTPConfig, ok := session.(SessionWithStreamableHTTPConfig); ok {
		sessionWithStreamableHTTPConfig.UpgradeToSSEWhenReceiveNotification()
	}
	select {
	case session.NotificationChannel() <- notification:
		return nil
	default:
		// Channel is blocked, if there's an error hook, use it
		if s.hooks != nil && len(s.hooks.OnError) > 0 {
			err := ErrNotificationChannelBlocked
			ctx := context.Background()
			// Copy hooks pointer to local variable to avoid race condition
			hooks := s.hooks
			go func(sID string, hooks *Hooks) {
				// Use the error hook to report the blocked channel
				hooks.onError(ctx, nil, "notification", map[string]any{
					"method":    notification.Method,
					"sessionID": sID,
				}, fmt.Errorf("notification channel blocked for session %s: %w", sID, err))
			}(session.SessionID(), hooks)
		}
		return ErrNotificationChannelBlocked
	}
}

func (s *MCPServer) SendLogMessageToSpecificClient(sessionID string, notification mcp.LoggingMessageNotification) error {
	sessionValue, ok := s.sessions.Load(sessionID)
	if !ok {
		return ErrSessionNotFound
	}
	session, ok := sessionValue.(ClientSession)
	if !ok || !session.Initialized() {
		return ErrSessionNotInitialized
	}
	sessionLogging, ok := session.(SessionWithLogging)
	if !ok {
		return ErrSessionDoesNotSupportLogging
	}
	if !notification.Params.Level.ShouldSendTo(sessionLogging.GetLogLevel()) {
		return nil
	}
	return s.sendNotificationToSpecificClient(session, s.buildLogNotification(notification))
}

// UnregisterSession removes from storage session that is shut down.
func (s *MCPServer) UnregisterSession(
	ctx context.Context,
	sessionID string,
) {
	sessionValue, ok := s.sessions.LoadAndDelete(sessionID)
	if !ok {
		return
	}
	if session, ok := sessionValue.(ClientSession); ok {
		s.hooks.UnregisterSession(ctx, session)
	}
}

// SendNotificationToAllClients sends a notification to all the currently active clients.
func (s *MCPServer) SendNotificationToAllClients(
	method string,
	params map[string]any,
) {
	notification := mcp.JSONRPCNotification{
		JSONRPC: mcp.JSONRPC_VERSION,
		Notification: mcp.Notification{
			Method: method,
			Params: mcp.NotificationParams{
				AdditionalFields: params,
			},
		},
	}
	s.sendNotificationToAllClients(notification)
}

// SendNotificationToClient sends a notification to the current client
func (s *MCPServer) sendNotificationCore(
	ctx context.Context,
	session ClientSession,
	notification mcp.JSONRPCNotification,
) error {
	// upgrades the client-server communication to SSE stream when the server sends notifications to the client
	if sessionWithStreamableHTTPConfig, ok := session.(SessionWithStreamableHTTPConfig); ok {
		sessionWithStreamableHTTPConfig.UpgradeToSSEWhenReceiveNotification()
	}
	select {
	case session.NotificationChannel() <- notification:
		return nil
	default:
		// Channel is blocked, if there's an error hook, use it
		if s.hooks != nil && len(s.hooks.OnError) > 0 {
			method := notification.Method
			err := ErrNotificationChannelBlocked
			// Copy hooks pointer to local variable to avoid race condition
			hooks := s.hooks
			go func(sessionID string, hooks *Hooks) {
				// Use the error hook to report the blocked channel
				hooks.onError(ctx, nil, "notification", map[string]any{
					"method":    method,
					"sessionID": sessionID,
				}, fmt.Errorf("notification channel blocked for session %s: %w", sessionID, err))
			}(session.SessionID(), hooks)
		}
		return ErrNotificationChannelBlocked
	}
}

// SendNotificationToClient sends a notification to the current client
func (s *MCPServer) SendNotificationToClient(
	ctx context.Context,
	method string,
	params map[string]any,
) error {
	session := ClientSessionFromContext(ctx)
	if session == nil || !session.Initialized() {
		return ErrNotificationNotInitialized
	}
	notification := mcp.JSONRPCNotification{
		JSONRPC: mcp.JSONRPC_VERSION,
		Notification: mcp.Notification{
			Method: method,
			Params: mcp.NotificationParams{
				AdditionalFields: params,
			},
		},
	}
	return s.sendNotificationCore(ctx, session, notification)
}

// SendNotificationToSpecificClient sends a notification to a specific client by session ID
func (s *MCPServer) SendNotificationToSpecificClient(
	sessionID string,
	method string,
	params map[string]any,
) error {
	sessionValue, ok := s.sessions.Load(sessionID)
	if !ok {
		return ErrSessionNotFound
	}
	session, ok := sessionValue.(ClientSession)
	if !ok || !session.Initialized() {
		return ErrSessionNotInitialized
	}
	notification := mcp.JSONRPCNotification{
		JSONRPC: mcp.JSONRPC_VERSION,
		Notification: mcp.Notification{
			Method: method,
			Params: mcp.NotificationParams{
				AdditionalFields: params,
			},
		},
	}
	return s.sendNotificationToSpecificClient(session, notification)
}

// AddSessionTool adds a tool for a specific session
func (s *MCPServer) AddSessionTool(sessionID string, tool mcp.Tool, handler ToolHandlerFunc) error {
	return s.AddSessionTools(sessionID, ServerTool{Tool: tool, Handler: handler})
}

// AddSessionTools adds tools for a specific session
func (s *MCPServer) AddSessionTools(sessionID string, tools ...ServerTool) error {
	sessionValue, ok := s.sessions.Load(sessionID)
	if !ok {
		return ErrSessionNotFound
	}

	session, ok := sessionValue.(SessionWithTools)
	if !ok {
		return ErrSessionDoesNotSupportTools
	}

	s.implicitlyRegisterToolCapabilities()

	// Get existing tools (this should return a thread-safe copy)
	sessionTools := session.GetSessionTools()

	// Create a new map to avoid concurrent modification issues
	newSessionTools := make(map[string]ServerTool, len(sessionTools)+len(tools))

	// Copy existing tools
	for k, v := range sessionTools {
		newSessionTools[k] = v
	}

	// Add new tools
	for _, tool := range tools {
		newSessionTools[tool.Tool.Name] = tool
	}

	// Set the tools (this should be thread-safe)
	session.SetSessionTools(newSessionTools)

	// It only makes sense to send tool notifications to initialized sessions --
	// if we're not initialized yet the client can't possibly have sent their
	// initial tools/list message.
	//
	// For initialized sessions, honor tools.listChanged, which is specifically
	// about whether notifications will be sent or not.
	// see <https://modelcontextprotocol.io/specification/2025-03-26/server/tools#capabilities>
	if session.Initialized() && s.capabilities.tools != nil && s.capabilities.tools.listChanged {
		// Send notification only to this session
		if err := s.SendNotificationToSpecificClient(sessionID, "notifications/tools/list_changed", nil); err != nil {
			// Log the error but don't fail the operation
			// The tools were successfully added, but notification failed
			if s.hooks != nil && len(s.hooks.OnError) > 0 {
				hooks := s.hooks
				go func(sID string, hooks *Hooks) {
					ctx := context.Background()
					hooks.onError(ctx, nil, "notification", map[string]any{
						"method":    "notifications/tools/list_changed",
						"sessionID": sID,
					}, fmt.Errorf("failed to send notification after adding tools: %w", err))
				}(sessionID, hooks)
			}
		}
	}

	return nil
}

// DeleteSessionTools removes tools from a specific session
func (s *MCPServer) DeleteSessionTools(sessionID string, names ...string) error {
	sessionValue, ok := s.sessions.Load(sessionID)
	if !ok {
		return ErrSessionNotFound
	}

	session, ok := sessionValue.(SessionWithTools)
	if !ok {
		return ErrSessionDoesNotSupportTools
	}

	// Get existing tools (this should return a thread-safe copy)
	sessionTools := session.GetSessionTools()
	if sessionTools == nil {
		return nil
	}

	// Create a new map to avoid concurrent modification issues
	newSessionTools := make(map[string]ServerTool, len(sessionTools))

	// Copy existing tools except those being deleted
	for k, v := range sessionTools {
		newSessionTools[k] = v
	}

	// Remove specified tools
	for _, name := range names {
		delete(newSessionTools, name)
	}

	// Set the tools (this should be thread-safe)
	session.SetSessionTools(newSessionTools)

	// It only makes sense to send tool notifications to initialized sessions --
	// if we're not initialized yet the client can't possibly have sent their
	// initial tools/list message.
	//
	// For initialized sessions, honor tools.listChanged, which is specifically
	// about whether notifications will be sent or not.
	// see <https://modelcontextprotocol.io/specification/2025-03-26/server/tools#capabilities>
	if session.Initialized() && s.capabilities.tools != nil && s.capabilities.tools.listChanged {
		// Send notification only to this session
		if err := s.SendNotificationToSpecificClient(sessionID, "notifications/tools/list_changed", nil); err != nil {
			// Log the error but don't fail the operation
			// The tools were successfully deleted, but notification failed
			if s.hooks != nil && len(s.hooks.OnError) > 0 {
				hooks := s.hooks
				go func(sID string, hooks *Hooks) {
					ctx := context.Background()
					hooks.onError(ctx, nil, "notification", map[string]any{
						"method":    "notifications/tools/list_changed",
						"sessionID": sID,
					}, fmt.Errorf("failed to send notification after deleting tools: %w", err))
				}(sessionID, hooks)
			}
		}
	}

	return nil
}

// AddSessionResource adds a resource for a specific session
func (s *MCPServer) AddSessionResource(sessionID string, resource mcp.Resource, handler ResourceHandlerFunc) error {
	return s.AddSessionResources(sessionID, ServerResource{Resource: resource, Handler: handler})
}

// AddSessionResources adds resources for a specific session
func (s *MCPServer) AddSessionResources(sessionID string, resources ...ServerResource) error {
	sessionValue, ok := s.sessions.Load(sessionID)
	if !ok {
		return ErrSessionNotFound
	}

	session, ok := sessionValue.(SessionWithResources)
	if !ok {
		return ErrSessionDoesNotSupportResources
	}

	// For session resources, we want listChanged enabled by default
	s.implicitlyRegisterCapabilities(
		func() bool { return s.capabilities.resources != nil },
		func() { s.capabilities.resources = &resourceCapabilities{listChanged: true} },
	)

	// Get existing resources (this should return a thread-safe copy)
	sessionResources := session.GetSessionResources()

	// Create a new map to avoid concurrent modification issues
	newSessionResources := make(map[string]ServerResource, len(sessionResources)+len(resources))

	// Copy existing resources
	for k, v := range sessionResources {
		newSessionResources[k] = v
	}

	// Add new resources with validation
	for _, resource := range resources {
		// Validate that URI is non-empty
		if resource.Resource.URI == "" {
			return fmt.Errorf("resource URI cannot be empty")
		}

		// Validate that URI conforms to RFC 3986
		if _, err := url.ParseRequestURI(resource.Resource.URI); err != nil {
			return fmt.Errorf("invalid resource URI: %w", err)
		}

		newSessionResources[resource.Resource.URI] = resource
	}

	// Set the resources (this should be thread-safe)
	session.SetSessionResources(newSessionResources)

	// It only makes sense to send resource notifications to initialized sessions --
	// if we're not initialized yet the client can't possibly have sent their
	// initial resources/list message.
	//
	// For initialized sessions, honor resources.listChanged, which is specifically
	// about whether notifications will be sent or not.
	// see <https://modelcontextprotocol.io/specification/2025-03-26/server/resources#capabilities>
	if session.Initialized() && s.capabilities.resources != nil && s.capabilities.resources.listChanged {
		// Send notification only to this session
		if err := s.SendNotificationToSpecificClient(sessionID, "notifications/resources/list_changed", nil); err != nil {
			// Log the error but don't fail the operation
			// The resources were successfully added, but notification failed
			if s.hooks != nil && len(s.hooks.OnError) > 0 {
				hooks := s.hooks
				go func(sID string, hooks *Hooks) {
					ctx := context.Background()
					hooks.onError(ctx, nil, "notification", map[string]any{
						"method":    "notifications/resources/list_changed",
						"sessionID": sID,
					}, fmt.Errorf("failed to send notification after adding resources: %w", err))
				}(sessionID, hooks)
			}
		}
	}

	return nil
}

// DeleteSessionResources removes resources from a specific session
func (s *MCPServer) DeleteSessionResources(sessionID string, uris ...string) error {
	sessionValue, ok := s.sessions.Load(sessionID)
	if !ok {
		return ErrSessionNotFound
	}

	session, ok := sessionValue.(SessionWithResources)
	if !ok {
		return ErrSessionDoesNotSupportResources
	}

	// Get existing resources (this should return a thread-safe copy)
	sessionResources := session.GetSessionResources()
	if sessionResources == nil {
		return nil
	}

	// Create a new map to avoid concurrent modification issues
	newSessionResources := make(map[string]ServerResource, len(sessionResources))

	// Copy existing resources except those being deleted
	for k, v := range sessionResources {
		newSessionResources[k] = v
	}

	// Remove specified resources and track if anything was actually deleted
	actuallyDeleted := false
	for _, uri := range uris {
		if _, exists := newSessionResources[uri]; exists {
			delete(newSessionResources, uri)
			actuallyDeleted = true
		}
	}

	// Skip no-op write if nothing was actually deleted
	if !actuallyDeleted {
		return nil
	}

	// Set the resources (this should be thread-safe)
	session.SetSessionResources(newSessionResources)

	// It only makes sense to send resource notifications to initialized sessions --
	// if we're not initialized yet the client can't possibly have sent their
	// initial resources/list message.
	//
	// For initialized sessions, honor resources.listChanged, which is specifically
	// about whether notifications will be sent or not.
	// see <https://modelcontextprotocol.io/specification/2025-03-26/server/resources#capabilities>
	// Only send notification if something was actually deleted
	if actuallyDeleted && session.Initialized() && s.capabilities.resources != nil && s.capabilities.resources.listChanged {
		// Send notification only to this session
		if err := s.SendNotificationToSpecificClient(sessionID, "notifications/resources/list_changed", nil); err != nil {
			// Log the error but don't fail the operation
			// The resources were successfully deleted, but notification failed
			if s.hooks != nil && len(s.hooks.OnError) > 0 {
				hooks := s.hooks
				go func(sID string, hooks *Hooks) {
					ctx := context.Background()
					hooks.onError(ctx, nil, "notification", map[string]any{
						"method":    "notifications/resources/list_changed",
						"sessionID": sID,
					}, fmt.Errorf("failed to send notification after deleting resources: %w", err))
				}(sessionID, hooks)
			}
		}
	}

	return nil
}

// AddSessionResourceTemplate adds a resource template for a specific session
func (s *MCPServer) AddSessionResourceTemplate(sessionID string, template mcp.ResourceTemplate, handler ResourceTemplateHandlerFunc) error {
	return s.AddSessionResourceTemplates(sessionID, ServerResourceTemplate{
		Template: template,
		Handler:  handler,
	})
}

// AddSessionResourceTemplates adds resource templates for a specific session
func (s *MCPServer) AddSessionResourceTemplates(sessionID string, templates ...ServerResourceTemplate) error {
	sessionValue, ok := s.sessions.Load(sessionID)
	if !ok {
		return ErrSessionNotFound
	}

	session, ok := sessionValue.(SessionWithResourceTemplates)
	if !ok {
		return ErrSessionDoesNotSupportResourceTemplates
	}

	// For session resource templates, enable listChanged by default
	// This is the same behavior as session resources
	s.implicitlyRegisterCapabilities(
		func() bool { return s.capabilities.resources != nil },
		func() { s.capabilities.resources = &resourceCapabilities{listChanged: true} },
	)

	// Get existing templates (this returns a thread-safe copy)
	sessionTemplates := session.GetSessionResourceTemplates()

	// Create a new map to avoid modifying the returned copy
	newTemplates := make(map[string]ServerResourceTemplate, len(sessionTemplates)+len(templates))

	// Copy existing templates
	for k, v := range sessionTemplates {
		newTemplates[k] = v
	}

	// Validate and add new templates
	for _, t := range templates {
		if t.Template.URITemplate == nil {
			return fmt.Errorf("resource template URITemplate cannot be nil")
		}
		raw := t.Template.URITemplate.Raw()
		if raw == "" {
			return fmt.Errorf("resource template URITemplate cannot be empty")
		}
		if t.Template.Name == "" {
			return fmt.Errorf("resource template name cannot be empty")
		}
		newTemplates[raw] = t
	}

	// Set the new templates (this method must handle thread-safety)
	session.SetSessionResourceTemplates(newTemplates)

	// Send notification if the session is initialized and listChanged is enabled
	if session.Initialized() && s.capabilities.resources != nil && s.capabilities.resources.listChanged {
		if err := s.SendNotificationToSpecificClient(sessionID, "notifications/resources/list_changed", nil); err != nil {
			// Log the error but don't fail the operation
			if s.hooks != nil && len(s.hooks.OnError) > 0 {
				hooks := s.hooks
				go func(sID string, hooks *Hooks) {
					ctx := context.Background()
					hooks.onError(ctx, nil, "notification", map[string]any{
						"method":    "notifications/resources/list_changed",
						"sessionID": sID,
					}, fmt.Errorf("failed to send notification after adding resource templates: %w", err))
				}(sessionID, hooks)
			}
		}
	}

	return nil
}

// DeleteSessionResourceTemplates removes resource templates from a specific session
func (s *MCPServer) DeleteSessionResourceTemplates(sessionID string, uriTemplates ...string) error {
	sessionValue, ok := s.sessions.Load(sessionID)
	if !ok {
		return ErrSessionNotFound
	}

	session, ok := sessionValue.(SessionWithResourceTemplates)
	if !ok {
		return ErrSessionDoesNotSupportResourceTemplates
	}

	// Get existing templates (this returns a thread-safe copy)
	sessionTemplates := session.GetSessionResourceTemplates()

	// Track if any were actually deleted
	deletedAny := false

	// Create a new map without the deleted templates
	newTemplates := make(map[string]ServerResourceTemplate, len(sessionTemplates))
	for k, v := range sessionTemplates {
		newTemplates[k] = v
	}

	// Delete specified templates
	for _, uriTemplate := range uriTemplates {
		if _, exists := newTemplates[uriTemplate]; exists {
			delete(newTemplates, uriTemplate)
			deletedAny = true
		}
	}

	// Only update if something was actually deleted
	if deletedAny {
		// Set the new templates (this method must handle thread-safety)
		session.SetSessionResourceTemplates(newTemplates)

		// Send notification if the session is initialized and listChanged is enabled
		if session.Initialized() && s.capabilities.resources != nil && s.capabilities.resources.listChanged {
			if err := s.SendNotificationToSpecificClient(sessionID, "notifications/resources/list_changed", nil); err != nil {
				// Log the error but don't fail the operation
				if s.hooks != nil && len(s.hooks.OnError) > 0 {
					hooks := s.hooks
					go func(sID string, hooks *Hooks) {
						ctx := context.Background()
						hooks.onError(ctx, nil, "notification", map[string]any{
							"method":    "notifications/resources/list_changed",
							"sessionID": sID,
						}, fmt.Errorf("failed to send notification after deleting resource templates: %w", err))
					}(sessionID, hooks)
				}
			}
		}
	}

	return nil
}
