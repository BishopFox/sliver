package winhttp

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/url"
	"strconv"
	"strings"
)

func convertFail(str string, e error) error {
	return fmt.Errorf(
		"failed to convert %s to Windows type: %w",
		str,
		e,
	)
}

func buildRequest(sessionHandle uintptr, req *Request) (uintptr, error) {
	var connHandle uintptr
	var err error
	var flags uintptr
	var port int64
	var query string
	var reqHandle uintptr
	var uri *url.URL

	// Parse URL
	if uri, err = url.Parse(req.URL); err != nil {
		return 0, fmt.Errorf("failed to parse url %s: %w", req.URL, err)
	}

	if uri.Port() != "" {
		if port, err = strconv.ParseInt(uri.Port(), 10, 64); err != nil {
			err = fmt.Errorf("port %s invalid: %w", uri.Port(), err)
			return 0, err
		}
	}

	switch uri.Scheme {
	case "https":
		flags = WinhttpFlagSecure
	}

	// Create connection
	connHandle, err = Connect(
		sessionHandle,
		uri.Hostname(),
		int(port),
	)
	if err != nil {
		return 0, fmt.Errorf("failed to create connection: %w", err)
	}

	// Send query string too
	if uri.RawQuery != "" {
		query = "?" + uri.RawQuery
	}

	// Create HTTP request
	reqHandle, err = OpenRequest(
		connHandle,
		req.Method,
		uri.Path+query,
		"",
		"",
		[]string{},
		flags,
	)
	if err != nil {
		return 0, fmt.Errorf("failed to open request: %w", err)
	}

	return reqHandle, nil
}

func buildResponse(reqHandle uintptr, req *Request) (*Response, error) {
	var buf []byte
	var body io.ReadCloser
	var code int64
	var contentLen int64
	var cookies []*Cookie
	var err error
	var headers map[string][]string
	var major int
	var minor int
	var proto string
	var resp *Response
	var status string

	// Get response
	if err = ReceiveResponse(reqHandle); err != nil {
		return nil, fmt.Errorf("failed to get response: %w", err)
	}

	// Get status code
	buf, err = queryResponse(reqHandle, WinhttpQueryStatusCode, 0)
	if err != nil {
		return nil, err
	}

	status = string(buf)
	if code, err = strconv.ParseInt(status, 10, 64); err != nil {
		return nil, fmt.Errorf("status %s invalid: %w", status, err)
	}

	// Get status text
	buf, err = queryResponse(reqHandle, WinhttpQueryStatusText, 0)
	if err != nil {
		return nil, err
	} else if len(buf) > 0 {
		status += " " + string(buf)
	}

	// Parse cookies
	cookies = getCookies(reqHandle)

	// Parse headers and proto
	if proto, major, minor, headers, err = getHeaders(reqHandle); err != nil {
		return nil, err
	}

	// Read response body
	if body, contentLen, err = readResponse(reqHandle); err != nil {
		return nil, err
	}

	resp = &Response{
		Body:          body,
		ContentLength: contentLen,
		Header:        headers,
		Proto:         proto,
		ProtoMajor:    major,
		ProtoMinor:    minor,
		Status:        status,
		StatusCode:    int(code),
	}

	// Concat all cookies
	for _, c := range req.Cookies() {
		resp.AddCookie(c)
	}

	for _, c := range cookies {
		resp.AddCookie(c)
	}

	return resp, nil
}

func getCookies(reqHandle uintptr) []*Cookie {
	var buf []byte
	var cookies []*Cookie
	var err error
	var tmp []string

	// Get cookies
	for index := 0; ; index++ {
		buf, err = queryResponse(
			reqHandle,
			WinhttpQuerySetCookie,
			index,
		)
		if err != nil {
			break
		}

		tmp = strings.SplitN(string(buf), "=", 2)
		cookies = append(
			cookies,
			&Cookie{Name: tmp[0], Value: tmp[1]},
		)
	}

	return cookies
}

func getHeaders(reqHandle uintptr) (string, int, int, map[string][]string, error) {
	var buf []byte
	var err error
	var headers = map[string][]string{}
	var major int64
	var minor int64
	var proto string
	var tmp []string

	// Get headers
	buf, err = queryResponse(
		reqHandle,
		WinhttpQueryRawHeadersCRLF,
		0,
	)
	if err != nil {
		return "", 0, 0, nil, err
	}

	for _, hdr := range strings.Split(string(buf), "\r\n") {
		tmp = strings.SplitN(hdr, ": ", 2)

		if len(tmp) == 2 {
			if _, ok := headers[tmp[0]]; !ok {
				headers[tmp[0]] = []string{}
			}

			headers[tmp[0]] = append(headers[tmp[0]], tmp[1])
		} else if strings.HasPrefix(hdr, "HTTP") {
			proto = strings.Fields(hdr)[0]
			tmp = strings.Split(proto, ".")

			if len(tmp) >= 2 {
				tmp[0] = strings.Replace(tmp[0], "HTTP/", "", 1)

				major, err = strconv.ParseInt(tmp[0], 10, 64)
				if err != nil {
					err = fmt.Errorf("invalid HTTP version: %w", err)
					return "", 0, 0, nil, err
				}

				minor, err = strconv.ParseInt(tmp[1], 10, 64)
				if err != nil {
					err = fmt.Errorf("invalid HTTP version: %w", err)
					return "", 0, 0, nil, err
				}
			}
		}
	}

	return proto, int(major), int(minor), headers, nil
}

func queryResponse(reqHandle, info uintptr, idx int) ([]byte, error) {
	var buffer []byte
	var err error
	var size int

	if idx < 0 {
		idx = 0
	}

	err = QueryHeaders(
		reqHandle,
		info,
		"",
		&buffer,
		&size,
		&idx,
	)
	if err != nil {
		buffer = make([]byte, size)

		err = QueryHeaders(
			reqHandle,
			info,
			"",
			&buffer,
			&size,
			&idx,
		)
		if err != nil {
			err = fmt.Errorf("failed to query info: %w", err)
			return []byte{}, err
		}
	}

	return buffer, nil
}

func readResponse(reqHandle uintptr) (io.ReadCloser, int64, error) {
	var buf []byte
	var chunk []byte
	var chunkLen int64
	var contentLen int64
	var err error
	var n int64

	// Get Content-Length and body of response
	for {
		// Get next chunk size
		err = QueryDataAvailable(reqHandle, &chunkLen)
		if err != nil {
			err = fmt.Errorf("failed to query data available: %w", err)
			break
		}

		// Stop, if finished
		if chunkLen == 0 {
			break
		}

		// Read next chunk
		err = ReadData(reqHandle, &chunk, chunkLen, &n)
		if err != nil {
			err = fmt.Errorf("failed to read data: %w", err)
			break
		}

		// Update fields
		contentLen += chunkLen
		buf = append(buf, chunk...)
	}

	if err != nil {
		return nil, 0, err
	}

	return ioutil.NopCloser(bytes.NewReader(buf)), contentLen, nil
}

func sendRequest(reqHandle uintptr, req *Request) error {
	var err error
	var method uintptr

	// Process cookies
	method = WinhttpAddreqFlagAdd
	method |= WinhttpAddreqFlagCoalesceWithSemicolon

	for _, c := range req.Cookies() {
		err = AddRequestHeaders(
			reqHandle,
			"Cookie: "+c.Name+"="+c.Value,
			method,
		)
		if err != nil {
			return fmt.Errorf("failed to add cookies: %w", err)
		}
	}

	// Process headers
	method = WinhttpAddreqFlagAdd
	method |= WinhttpAddreqFlagReplace

	for k, v := range req.Headers {
		err = AddRequestHeaders(
			reqHandle,
			k+": "+v,
			method,
		)
		if err != nil {
			return fmt.Errorf("failed to add request headers: %w", err)
		}
	}

	// Send HTTP request
	err = SendRequest(
		reqHandle,
		"",
		0,
		req.Body,
		len(req.Body),
	)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}

	return nil
}
