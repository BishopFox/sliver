package mcp

import (
	"fmt"
	"net"
	"net/url"
	"strings"

	"github.com/bishopfox/sliver/client/version"
)

// Transport identifies the MCP server transport to expose.
type Transport string

const (
	TransportHTTP Transport = "http"
	TransportSSE  Transport = "sse"
)

// Config controls the local MCP server settings.
type Config struct {
	Transport     Transport
	ListenAddress string
	ServerName    string
	ServerVersion string
}

// DefaultConfig returns the baseline MCP configuration.
func DefaultConfig() Config {
	return Config{
		Transport:     TransportSSE,
		ListenAddress: "127.0.0.1:8080",
		ServerName:    "Sliver MCP",
		ServerVersion: version.Version,
	}
}

// WithDefaults fills any empty fields with defaults.
func (c Config) WithDefaults() Config {
	defaults := DefaultConfig()
	if c.Transport == "" {
		c.Transport = defaults.Transport
	}
	if c.ListenAddress == "" {
		c.ListenAddress = defaults.ListenAddress
	}
	if c.ServerName == "" {
		c.ServerName = defaults.ServerName
	}
	if c.ServerVersion == "" {
		c.ServerVersion = defaults.ServerVersion
	}
	return c
}

// ParseTransport validates an input transport string.
func ParseTransport(raw string) (Transport, error) {
	value := strings.ToLower(strings.TrimSpace(raw))
	if value == "" {
		return DefaultConfig().Transport, nil
	}
	switch Transport(value) {
	case TransportHTTP, TransportSSE:
		return Transport(value), nil
	default:
		return "", fmt.Errorf("unsupported transport %q", raw)
	}
}

// Validate ensures the configuration can be used for a server start.
func (c Config) Validate() error {
	if c.Transport != TransportHTTP && c.Transport != TransportSSE {
		return fmt.Errorf("unsupported transport %q", c.Transport)
	}
	if c.ListenAddress == "" {
		return fmt.Errorf("listen address is required")
	}
	if _, _, err := net.SplitHostPort(c.ListenAddress); err != nil {
		return fmt.Errorf("invalid listen address %q: %w", c.ListenAddress, err)
	}
	if c.ServerName == "" {
		return fmt.Errorf("server name is required")
	}
	if c.ServerVersion == "" {
		return fmt.Errorf("server version is required")
	}
	return nil
}

// EndpointURL returns the base URL for clients to connect.
func (c Config) EndpointURL() (string, error) {
	cfg := c.WithDefaults()
	if err := cfg.Validate(); err != nil {
		return "", err
	}

	host, port, err := net.SplitHostPort(cfg.ListenAddress)
	if err != nil {
		return "", err
	}
	if host == "" {
		host = "127.0.0.1"
	}

	endpoint := url.URL{
		Scheme: "http",
		Host:   net.JoinHostPort(host, port),
	}
	switch cfg.Transport {
	case TransportHTTP:
		endpoint.Path = "/mcp"
	case TransportSSE:
		endpoint.Path = "/sse"
	default:
		return "", fmt.Errorf("unsupported transport %q", cfg.Transport)
	}
	return endpoint.String(), nil
}
