//go:build !windows

package http

import "errors"

// Client is an empty struct, if not Windows.
type Client struct{}

// NewClient is only supported on Windows.
func NewClient() (*Client, error) {
	return &Client{}, errors.New("unsupported OS")
}

// Do is only supported on Windows.
func (c *Client) Do(r *Request) (*Response, error) {
	return nil, errors.New("unsupported OS")
}

// Get is only supported on Windows.
func (c *Client) Get(url string) (*Response, error) {
	return nil, errors.New("unsupported OS")
}

// Head is only supported on Windows.
func (c *Client) Head(url string) (*Response, error) {
	return nil, errors.New("unsupported OS")
}

// Post is only supported on Windows.
func (c *Client) Post(
	url string,
	contentType string,
	body []byte,
) (*Response, error) {
	return nil, errors.New("unsupported OS")
}
