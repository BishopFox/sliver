package wininet

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/url"
	"strconv"
	"strings"
)

func convertFail(str string, err error) error {
	return fmt.Errorf(
		"failed to convert %s to Windows type: %w",
		str,
		err,
	)
}

func buildRequest(sessionHndl uintptr, r *Request) (uintptr, error) {
	var connHndl uintptr
	var err error
	var flags uintptr
	var passwd string
	var port int64
	var query string
	var reqHndl uintptr
	var uri *url.URL

	// Parse URL
	if uri, err = url.Parse(r.URL); err != nil {
		return 0, fmt.Errorf("failed to parse url %s: %w", r.URL, err)
	}

	passwd, _ = uri.User.Password()

	if uri.Port() != "" {
		if port, err = strconv.ParseInt(uri.Port(), 10, 64); err != nil {
			err = fmt.Errorf("port %s invalid: %w", uri.Port(), err)
			return 0, err
		}
	}

	switch uri.Scheme {
	case "https":
		flags = InternetFlagSecure
	}

	// Create connection
	connHndl, err = InternetConnectW(
		sessionHndl,
		uri.Hostname(),
		int(port),
		uri.User.Username(),
		passwd,
		InternetServiceHTTP,
		flags,
		0,
	)
	if err != nil {
		return 0, fmt.Errorf("failed to create connection: %w", err)
	}

	// Send query string too
	if uri.RawQuery != "" {
		query = "?" + uri.RawQuery
	}

	// Allow NTLM auth
	flags |= InternetFlagKeepConnection
	flags |= InternetFlagNoCookies //we're responsible for cookie management

	// Create HTTP request
	reqHndl, err = HTTPOpenRequestW(
		connHndl,
		r.Method,
		uri.Path+query,
		"",
		"",
		[]string{},
		flags,
		0,
	)
	if err != nil {
		return 0, fmt.Errorf("failed to open request: %w", err)
	}

	return reqHndl, nil
}

var cookies []*Cookie

func buildResponse(reqHndl uintptr, req *Request) (*Response, error) {
	var b []byte
	var body io.ReadCloser
	var code int64
	var contentLen int64
	var err error
	var hdrs map[string][]string
	var major int
	var minor int
	var proto string
	var res *Response
	var status string

	// Get status code
	b, err = queryResponse(reqHndl, HTTPQueryStatusCode, 0)
	if err != nil {
		return nil, err
	}

	status = string(b)
	if code, err = strconv.ParseInt(status, 10, 64); err != nil {
		return nil, fmt.Errorf("status %s invalid: %w", status, err)
	}

	// Get status text
	b, err = queryResponse(reqHndl, HTTPQueryStatusText, 0)
	if err != nil {
		return nil, err
	} else if len(b) > 0 {
		status += " " + string(b)
	}

	// Parse cookies
	cookies = getCookies(reqHndl)

	// Parse headers and proto
	if proto, major, minor, hdrs, err = getHeaders(reqHndl); err != nil {
		return nil, err
	}

	// Read response body
	if body, contentLen, err = readResponse(reqHndl); err != nil {
		return nil, err
	}

	res = &Response{
		Body:          body,
		ContentLength: contentLen,
		Header:        hdrs,
		Proto:         proto,
		ProtoMajor:    major,
		ProtoMinor:    minor,
		Status:        status,
		StatusCode:    int(code),
	}

	// Concat all cookies
	for _, c := range req.Cookies() {
		res.AddCookie(c)
	}

	for _, c := range cookies {
		res.AddCookie(c)
	}

	return res, nil
}

func getCookies(reqHndl uintptr) []*Cookie {
	var b []byte
	var cookies []*Cookie
	var err error
	var tmp []string

	// Get cookies
	for i := 0; ; i++ {
		b, err = queryResponse(
			reqHndl,
			HTTPQuerySetCookie,
			i,
		)
		if err != nil {
			break
		}

		tmp = strings.SplitN(string(b), "=", 2)
		cookies = append(
			cookies,
			&Cookie{Name: tmp[0], Value: tmp[1]},
		)
	}

	return cookies
}

func getHeaders(
	reqHndl uintptr,
) (string, int, int, map[string][]string, error) {
	var b []byte
	var err error
	var hdrs = map[string][]string{}
	var major int64
	var minor int64
	var proto string
	var tmp []string

	// Get headers
	b, err = queryResponse(reqHndl, HTTPQueryRawHeadersCRLF, 0)
	if err != nil {
		return "", 0, 0, nil, err
	}

	for _, hdr := range strings.Split(string(b), "\r\n") {
		tmp = strings.SplitN(hdr, ": ", 2)

		if len(tmp) == 2 {
			if _, ok := hdrs[tmp[0]]; !ok {
				hdrs[tmp[0]] = []string{}
			}

			hdrs[tmp[0]] = append(hdrs[tmp[0]], tmp[1])
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

	return proto, int(major), int(minor), hdrs, nil
}

func queryResponse(reqHndl, info uintptr, idx int) ([]byte, error) {
	var buffer []byte
	var err error
	var size int

	if idx < 0 {
		idx = 0
	}

	err = HTTPQueryInfoW(reqHndl, info, &buffer, &size, &idx)
	if err != nil {
		buffer = make([]byte, size)

		err = HTTPQueryInfoW(
			reqHndl,
			info,
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

func readResponse(reqHndl uintptr) (io.ReadCloser, int64, error) {
	var b []byte
	var chunk []byte
	var chunkLen int64
	var contentLen int64
	var err error
	var n int64

	// Get Content-Length and body of response
	for {
		// Get next chunk size
		err = InternetQueryDataAvailable(reqHndl, &chunkLen)
		if err != nil {
			err = fmt.Errorf("failed to query data available: %w", err)
			break
		}

		// Stop, if finished
		if chunkLen == 0 {
			break
		}

		// Read next chunk
		err = InternetReadFile(reqHndl, &chunk, chunkLen, &n)
		if err != nil {
			err = fmt.Errorf("failed to read data: %w", err)
			break
		}

		// Update fields
		contentLen += chunkLen
		b = append(b, chunk...)
	}

	if err != nil {
		return nil, 0, err
	}

	return ioutil.NopCloser(bytes.NewReader(b)), contentLen, nil
}

func sendRequest(reqHndl uintptr, r *Request) error {
	var err error
	var method uintptr

	// Process cookies
	method = HTTPAddreqFlagAdd
	method |= HTTPAddreqFlagCoalesceWithSemicolon

	// // FIXME This is a dumb hack
	// HTTPAddRequestHeadersW(
	// 	reqHndl,
	// 	"Cookie: ignore=ignore",
	// 	HTTPAddreqFlagAddIfNew,
	// )
	// End dumb hack

	for _, c := range r.Cookies() {
		err = HTTPAddRequestHeadersW(
			reqHndl,
			"Cookie: "+c.Name+"="+c.Value,
			method,
		)
		if err != nil {
			return fmt.Errorf("failed to add cookies: %w", err)
		}
	}

	// Process headers
	method = HTTPAddreqFlagAdd
	method |= HTTPAddreqFlagReplace

	for k, v := range r.Headers {
		err = HTTPAddRequestHeadersW(
			reqHndl,
			k+": "+v,
			method,
		)
		if err != nil {
			return fmt.Errorf("failed to add request headers: %w", err)
		}
	}

	// Send HTTP request
	err = HTTPSendRequestW(
		reqHndl,
		"",
		0,
		r.Body,
		len(r.Body),
	)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}

	return nil
}
