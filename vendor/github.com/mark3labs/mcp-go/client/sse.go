package client

import (
	"fmt"
	"net/http"
	"net/url"

	"github.com/mark3labs/mcp-go/client/transport"
)

func WithHeaders(headers map[string]string) transport.ClientOption {
	return transport.WithHeaders(headers)
}

func WithHeaderFunc(headerFunc transport.HTTPHeaderFunc) transport.ClientOption {
	return transport.WithHeaderFunc(headerFunc)
}

func WithHTTPClient(httpClient *http.Client) transport.ClientOption {
	return transport.WithHTTPClient(httpClient)
}

// WithHTTPHost sets a custom Host header for the SSE client, enabling manual DNS resolution.
// This allows connecting to an IP address while sending a specific Host header to the server.
// For example, connecting to "http://192.168.1.100:8080/sse" but sending Host: "api.example.com"
func WithHTTPHost(host string) transport.ClientOption {
	return transport.WithHTTPHost(host)
}

// NewSSEMCPClient creates a new SSE-based MCP client with the given base URL.
// Returns an error if the URL is invalid.
func NewSSEMCPClient(baseURL string, options ...transport.ClientOption) (*Client, error) {
	sseTransport, err := transport.NewSSE(baseURL, options...)
	if err != nil {
		return nil, fmt.Errorf("failed to create SSE transport: %w", err)
	}
	return NewClient(sseTransport), nil
}

// GetEndpoint returns the current endpoint URL for the SSE connection.
//
// Note: This method only works with SSE transport, or it will panic.
func GetEndpoint(c *Client) *url.URL {
	t := c.GetTransport()
	sse := t.(*transport.SSE)
	return sse.GetEndpoint()
}
