package wininet

import (
	"encoding/binary"
	"fmt"
	"io/ioutil"
	"net/http"
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
	var client = &Client{}
	var err error

	// Create session
	client.handle, err = InternetOpenW(
		userAgent,
		InternetOpenTypePreconfig,
		"",
		"",
		0,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}
	return client, nil
}

// Do will send the HTTP request and return an HTTP response.
func (c *Client) Do(request *http.Request) (*http.Response, error) {
	var buf []byte
	var err error
	var reqHandle uintptr
	var resp *Response

	var rawBody []byte
	if request.Body != nil {
		rawBody, err = ioutil.ReadAll(request.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read request body: %w", err)
		}
	}

	headers := make(map[string]string)
	for headerName, headerValue := range request.Header {
		if 0 < len(headerValue) {
			headers[headerName] = headerValue[0]
		}
	}
	req := &Request{
		Method:  request.Method,
		URL:     request.URL.String(),
		Headers: headers,
		Body:    rawBody,
	}

	if reqHandle, err = buildRequest(c.handle, req); err != nil {
		return nil, err
	}

	if c.Timeout > 0 {
		buf = make([]byte, 4)
		binary.LittleEndian.PutUint32(
			buf,
			uint32(c.Timeout.Milliseconds()),
		)

		err = InternetSetOptionW(
			reqHandle,
			InternetOptionConnectTimeout,
			buf,
			len(buf),
		)
		if err != nil {
			err = fmt.Errorf("failed to set connect timeout: %w", err)
			return nil, err
		}

		err = InternetSetOptionW(
			reqHandle,
			InternetOptionReceiveTimeout,
			buf,
			len(buf),
		)
		if err != nil {
			err = fmt.Errorf("failed to set receive timeout: %w", err)
			return nil, err
		}

		err = InternetSetOptionW(
			reqHandle,
			InternetOptionSendTimeout,
			buf,
			len(buf),
		)
		if err != nil {
			err = fmt.Errorf("failed to set send timeout: %w", err)
			return nil, err
		}
	}

	if c.TLSClientConfig.InsecureSkipVerify {
		buf = make([]byte, 4)
		binary.LittleEndian.PutUint32(
			buf,
			uint32(SecuritySetMask),
		)

		err = InternetSetOptionW(
			reqHandle,
			InternetOptionSecurityFlags,
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

	if resp, err = buildResponse(reqHandle, req); err != nil {
		return nil, err
	}

	return &http.Response{
		Status:        resp.Status,
		StatusCode:    resp.StatusCode,
		Proto:         "HTTP/1.1",
		ProtoMajor:    1,
		ProtoMinor:    1,
		Body:          resp.Body,
		ContentLength: resp.ContentLength,
		Request:       request,
		Header:        make(http.Header),
	}, nil
}
