// Package rest allows for quick and easy access any REST or REST-like API.
package rest

import (
	"bytes"
	"context"
	"io/ioutil"
	"net/http"
	"net/url"
)

// Version represents the current version of the rest library
const Version = "2.6.9"

// Method contains the supported HTTP verbs.
type Method string

// Supported HTTP verbs.
const (
	Get    Method = "GET"
	Post   Method = "POST"
	Put    Method = "PUT"
	Patch  Method = "PATCH"
	Delete Method = "DELETE"
)

// Request holds the request to an API Call.
type Request struct {
	Method      Method
	BaseURL     string // e.g. https://api.sendgrid.com
	Headers     map[string]string
	QueryParams map[string]string
	Body        []byte
}

// RestError is a struct for an error handling.
type RestError struct {
	Response *Response
}

// Error is the implementation of the error interface.
func (e *RestError) Error() string {
	return e.Response.Body
}

// DefaultClient is used if no custom HTTP client is defined
var DefaultClient = &Client{HTTPClient: &http.Client{}}

// Client allows modification of client headers, redirect policy
// and other settings
// See https://golang.org/pkg/net/http
type Client struct {
	HTTPClient *http.Client
}

// Response holds the response from an API call.
type Response struct {
	StatusCode int                 // e.g. 200
	Body       string              // e.g. {"result: success"}
	Headers    map[string][]string // e.g. map[X-Ratelimit-Limit:[600]]
}

// AddQueryParameters adds query parameters to the URL.
func AddQueryParameters(baseURL string, queryParams map[string]string) string {
	baseURL += "?"
	params := url.Values{}
	for key, value := range queryParams {
		params.Add(key, value)
	}
	return baseURL + params.Encode()
}

// BuildRequestObject creates the HTTP request object.
func BuildRequestObject(request Request) (*http.Request, error) {
	// Add any query parameters to the URL.
	if len(request.QueryParams) != 0 {
		request.BaseURL = AddQueryParameters(request.BaseURL, request.QueryParams)
	}
	req, err := http.NewRequest(string(request.Method), request.BaseURL, bytes.NewBuffer(request.Body))
	if err != nil {
		return req, err
	}
	for key, value := range request.Headers {
		req.Header.Set(key, value)
	}
	_, exists := req.Header["Content-Type"]
	if len(request.Body) > 0 && !exists {
		req.Header.Set("Content-Type", "application/json")
	}
	return req, err
}

// MakeRequest makes the API call.
func MakeRequest(req *http.Request) (*http.Response, error) {
	return DefaultClient.HTTPClient.Do(req)
}

// BuildResponse builds the response struct.
func BuildResponse(res *http.Response) (*Response, error) {
	body, err := ioutil.ReadAll(res.Body)
	response := Response{
		StatusCode: res.StatusCode,
		Body:       string(body),
		Headers:    res.Header,
	}
	res.Body.Close() // nolint
	return &response, err
}

// Deprecated: API supports old implementation
func API(request Request) (*Response, error) {
	return Send(request)
}

// Send uses the DefaultClient to send your request
func Send(request Request) (*Response, error) {
	return SendWithContext(context.Background(), request)
}

// SendWithContext uses the DefaultClient to send your request with the provided context.
func SendWithContext(ctx context.Context, request Request) (*Response, error) {
	return DefaultClient.SendWithContext(ctx, request)
}

// The following functions enable the ability to define a
// custom HTTP Client

// MakeRequest makes the API call.
func (c *Client) MakeRequest(req *http.Request) (*http.Response, error) {
	return c.HTTPClient.Do(req)
}

// Deprecated: API supports old implementation
func (c *Client) API(request Request) (*Response, error) {
	return c.Send(request)
}

// Send will build your request, make the request, and build your response.
func (c *Client) Send(request Request) (*Response, error) {
	return c.SendWithContext(context.Background(), request)
}

// SendWithContext will build your request passing in the provided context, make the request, and build your response.
func (c *Client) SendWithContext(ctx context.Context, request Request) (*Response, error) {
	// Build the HTTP request object.
	req, err := BuildRequestObject(request)
	if err != nil {
		return nil, err
	}
	// Pass in the user provided context
	req = req.WithContext(ctx)

	// Build the HTTP client and make the request.
	res, err := c.MakeRequest(req)
	if err != nil {
		return nil, err
	}

	// Build Response object.
	return BuildResponse(res)
}
