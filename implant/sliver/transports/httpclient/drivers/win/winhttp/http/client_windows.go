package http

import (
	"encoding/binary"
	"fmt"
	"time"

	"github.com/bishopfox/sliver/implant/sliver/transports/httpclient/drivers/win/winhttp"
)

// Client is a struct containing relevant metadata to make HTTP
// requests.
type Client struct {
	hndl            uintptr
	Timeout         time.Duration
	TLSClientConfig struct {
		InsecureSkipVerify bool
	}
}

// NewClient will return a pointer to a new Client instance that
// simply wraps the net/http.Client type.
func NewClient() (*Client, error) {
	var c = &Client{}
	var e error

	// Create session
	c.hndl, e = winhttp.Open(
		"Go-http-client/1.1",
		winhttp.WinhttpAccessTypeAutomaticProxy,
		"",
		"",
		0,
	)
	if e != nil {
		return nil, fmt.Errorf("failed to create session: %w", e)
	}

	return c, nil
}

// Do will send the HTTP request and return an HTTP response.
func (c *Client) Do(r *Request) (*Response, error) {
	var b []byte
	var e error
	var reqHndl uintptr
	var res *Response
	var tlsIgnore uintptr

	if reqHndl, e = buildRequest(c.hndl, r); e != nil {
		return nil, e
	}

	if c.Timeout > 0 {
		b = make([]byte, 4)
		binary.LittleEndian.PutUint32(
			b,
			uint32(c.Timeout.Milliseconds()),
		)

		e = winhttp.SetOption(
			reqHndl,
			winhttp.WinhttpOptionConnectTimeout,
			b,
			len(b),
		)
		if e != nil {
			e = fmt.Errorf("failed to set connect timeout: %w", e)
			return nil, e
		}

		e = winhttp.SetOption(
			reqHndl,
			winhttp.WinhttpOptionReceiveResponseTimeout,
			b,
			len(b),
		)
		if e != nil {
			e = fmt.Errorf("failed to set response timeout: %w", e)
			return nil, e
		}

		e = winhttp.SetOption(
			reqHndl,
			winhttp.WinhttpOptionReceiveTimeout,
			b,
			len(b),
		)
		if e != nil {
			e = fmt.Errorf("failed to set receive timeout: %w", e)
			return nil, e
		}

		e = winhttp.SetOption(
			reqHndl,
			winhttp.WinhttpOptionResolveTimeout,
			b,
			len(b),
		)
		if e != nil {
			e = fmt.Errorf("failed to set resolve timeout: %w", e)
			return nil, e
		}

		e = winhttp.SetOption(
			reqHndl,
			winhttp.WinhttpOptionSendTimeout,
			b,
			len(b),
		)
		if e != nil {
			e = fmt.Errorf("failed to set send timeout: %w", e)
			return nil, e
		}
	}

	if c.TLSClientConfig.InsecureSkipVerify {
		tlsIgnore |= winhttp.SecurityFlagIgnoreUnknownCa
		tlsIgnore |= winhttp.SecurityFlagIgnoreCertDateInvalid
		tlsIgnore |= winhttp.SecurityFlagIgnoreCertCnInvalid
		tlsIgnore |= winhttp.SecurityFlagIgnoreCertWrongUsage

		b = make([]byte, 4)
		binary.LittleEndian.PutUint32(b, uint32(tlsIgnore))

		e = winhttp.SetOption(
			reqHndl,
			winhttp.WinhttpOptionSecurityFlags,
			b,
			len(b),
		)
		if e != nil {
			e = fmt.Errorf("failed to set security flags: %w", e)
			return nil, e
		}
	}

	if e = sendRequest(reqHndl, r); e != nil {
		return nil, e
	}

	if res, e = buildResponse(reqHndl, r); e != nil {
		return nil, e
	}

	return res, nil
}

// Get will make a GET request using Winhttp.dll.
func (c *Client) Get(url string) (*Response, error) {
	return c.Do(NewRequest(MethodGet, url))
}

// Head will make a HEAD request using Winhttp.dll.
func (c *Client) Head(url string) (*Response, error) {
	return c.Do(NewRequest(MethodHead, url))
}

// Post will make a POST request using Winhttp.dll.
func (c *Client) Post(
	url string,
	contentType string,
	body []byte,
) (*Response, error) {
	var r *Request = NewRequest(MethodPost, url, body)

	if contentType != "" {
		r.Headers["Content-Type"] = contentType
	}

	return c.Do(r)
}
