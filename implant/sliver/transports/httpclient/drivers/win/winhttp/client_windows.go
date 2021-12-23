package winhttp

import (
	"encoding/binary"
	"fmt"
	"time"
)

// Client is a struct containing relevant metadata to make HTTP
// requests.
type Client struct {
	handle          uintptr
	Timeout         time.Duration
	TLSClientConfig struct {
		InsecureSkipVerify bool
	}
}

// NewClient will return a pointer to a new Client instance that
// simply wraps the net/http.Client type.
func NewClient(userAgent string) (*Client, error) {
	var c = &Client{}
	var e error

	// Create session
	c.handle, e = Open(
		userAgent,
		WinhttpAccessTypeAutomaticProxy,
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
func (c *Client) Do(req *Request) (*Response, error) {
	var buf []byte
	var err error
	var reqHandle uintptr
	var res *Response
	var tlsIgnore uintptr

	if reqHandle, err = buildRequest(c.handle, req); err != nil {
		return nil, err
	}

	if c.Timeout > 0 {
		buf = make([]byte, 4)
		binary.LittleEndian.PutUint32(
			buf,
			uint32(c.Timeout.Milliseconds()),
		)

		err = SetOption(
			reqHandle,
			WinhttpOptionConnectTimeout,
			buf,
			len(buf),
		)
		if err != nil {
			err = fmt.Errorf("failed to set connect timeout: %w", err)
			return nil, err
		}

		err = SetOption(
			reqHandle,
			WinhttpOptionReceiveResponseTimeout,
			buf,
			len(buf),
		)
		if err != nil {
			err = fmt.Errorf("failed to set response timeout: %w", err)
			return nil, err
		}

		err = SetOption(
			reqHandle,
			WinhttpOptionReceiveTimeout,
			buf,
			len(buf),
		)
		if err != nil {
			err = fmt.Errorf("failed to set receive timeout: %w", err)
			return nil, err
		}

		err = SetOption(
			reqHandle,
			WinhttpOptionResolveTimeout,
			buf,
			len(buf),
		)
		if err != nil {
			err = fmt.Errorf("failed to set resolve timeout: %w", err)
			return nil, err
		}

		err = SetOption(
			reqHandle,
			WinhttpOptionSendTimeout,
			buf,
			len(buf),
		)
		if err != nil {
			err = fmt.Errorf("failed to set send timeout: %w", err)
			return nil, err
		}
	}

	if c.TLSClientConfig.InsecureSkipVerify {
		tlsIgnore |= SecurityFlagIgnoreUnknownCa
		tlsIgnore |= SecurityFlagIgnoreCertDateInvalid
		tlsIgnore |= SecurityFlagIgnoreCertCnInvalid
		tlsIgnore |= SecurityFlagIgnoreCertWrongUsage

		buf = make([]byte, 4)
		binary.LittleEndian.PutUint32(buf, uint32(tlsIgnore))

		err = SetOption(
			reqHandle,
			WinhttpOptionSecurityFlags,
			buf,
			len(buf),
		)
		if err != nil {
			err = fmt.Errorf("failed to set security flags: %w", err)
			return nil, err
		}
	}

	if err = sendRequest(reqHandle, req); err != nil {
		return nil, err
	}

	if res, err = buildResponse(reqHandle, req); err != nil {
		return nil, err
	}

	return res, nil
}

// Get will make a GET request using dll.
func (c *Client) Get(url string) (*Response, error) {
	return c.Do(NewRequest(MethodGet, url))
}

// Head will make a HEAD request using dll.
func (c *Client) Head(url string) (*Response, error) {
	return c.Do(NewRequest(MethodHead, url))
}

// Post will make a POST request using dll.
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
