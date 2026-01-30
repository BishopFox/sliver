package mcp

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/bishopfox/sliver/protobuf/rpcpb"
	mcpserver "github.com/mark3labs/mcp-go/server"
)

var (
	ErrAlreadyRunning = errors.New("mcp server already running")
	ErrNotRunning     = errors.New("mcp server not running")
)

// Status describes the running state of the MCP server.
type Status struct {
	Running   bool
	StartedAt time.Time
	Config    Config
	LastError string
}

type serverTransport interface {
	Start(addr string) error
	Shutdown(ctx context.Context) error
}

type mcpManager struct {
	mu        sync.Mutex
	cfg       Config
	running   bool
	startedAt time.Time
	lastErr   string
	server    *SliverMCPServer
	transport serverTransport
	done      chan struct{}
	logger    *log.Logger
	logFile   *os.File
}

func newManager() *mcpManager {
	return &mcpManager{
		cfg: DefaultConfig(),
	}
}

var defaultManager = newManager()

// GetStatus returns the current MCP server state.
func GetStatus() Status {
	return defaultManager.status()
}

// Start launches the MCP server using the provided configuration.
func Start(cfg Config, rpc rpcpb.SliverRPCClient) error {
	return defaultManager.start(cfg, rpc)
}

// ServeStdio runs the MCP server over stdio using the provided configuration.
func ServeStdio(cfg Config, rpc rpcpb.SliverRPCClient) error {
	cfg = cfg.WithDefaults()
	if err := cfg.Validate(); err != nil {
		return err
	}
	logger, logFile, err := newMCPLogger()
	if err != nil {
		return err
	}
	defer logFile.Close()
	logger.Printf("starting mcp stdio server")
	srv := newServer(cfg, rpc, logger)
	err = mcpserver.ServeStdio(srv.server)
	if err != nil {
		logger.Printf("mcp stdio server error: %v", err)
	} else {
		logger.Printf("mcp stdio server stopped")
	}
	return err
}

// Stop shuts down the MCP server.
func Stop(ctx context.Context) error {
	return defaultManager.stop(ctx)
}

func (m *mcpManager) status() Status {
	m.mu.Lock()
	defer m.mu.Unlock()
	return Status{
		Running:   m.running,
		StartedAt: m.startedAt,
		Config:    m.cfg,
		LastError: m.lastErr,
	}
}

func (m *mcpManager) start(cfg Config, rpc rpcpb.SliverRPCClient) error {
	cfg = cfg.WithDefaults()
	if err := cfg.Validate(); err != nil {
		return err
	}

	m.mu.Lock()
	if m.running {
		m.mu.Unlock()
		return ErrAlreadyRunning
	}
	logger, logFile, err := newMCPLogger()
	if err != nil {
		m.mu.Unlock()
		return err
	}
	mcpServer := newServer(cfg, rpc, logger)
	var transport serverTransport
	switch cfg.Transport {
	case TransportHTTP:
		transport = mcpserver.NewStreamableHTTPServer(mcpServer.server)
	case TransportSSE:
		transport = mcpserver.NewSSEServer(mcpServer.server)
	default:
		m.mu.Unlock()
		logFile.Close()
		return errors.New("unsupported transport")
	}

	m.cfg = cfg
	m.running = true
	m.startedAt = time.Now()
	m.lastErr = ""
	m.server = mcpServer
	m.transport = transport
	m.done = make(chan struct{})
	m.logger = logger
	m.logFile = logFile
	addr := cfg.ListenAddress
	done := m.done
	m.mu.Unlock()

	logger.Printf("starting mcp server transport=%s listen=%s", cfg.Transport, cfg.ListenAddress)
	go m.run(addr, transport, done)
	return nil
}

func (m *mcpManager) run(addr string, transport serverTransport, done chan struct{}) {
	err := transport.Start(addr)
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		m.logf("mcp server error: %v", err)
		m.mu.Lock()
		m.lastErr = err.Error()
		m.mu.Unlock()
	}
	m.logf("mcp server stopped")

	m.mu.Lock()
	m.running = false
	m.transport = nil
	m.server = nil
	logFile := m.logFile
	m.logFile = nil
	m.logger = nil
	m.mu.Unlock()
	if logFile != nil {
		logFile.Close()
	}
	close(done)
}

func (m *mcpManager) stop(ctx context.Context) error {
	m.mu.Lock()
	if !m.running {
		m.mu.Unlock()
		return ErrNotRunning
	}
	transport := m.transport
	done := m.done
	m.mu.Unlock()

	if transport == nil {
		return ErrNotRunning
	}
	m.logf("mcp server shutdown requested")
	if err := transport.Shutdown(ctx); err != nil {
		return err
	}
	if done == nil {
		return nil
	}
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
