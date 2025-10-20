package wininet

import (
	"encoding/binary"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"sync"
	"time"
)

const (
	FLAGS_ERROR_UI_FILTER_FOR_ERRORS    uint32 = 0x01
	FLAGS_ERROR_UI_FLAGS_CHANGE_OPTIONS uint32 = 0x02
	FLAGS_ERROR_UI_FLAGS_GENERATE_DATA  uint32 = 0x04
	INTERNET_ERROR_BASE                 uint32 = 12000
	ERROR_INTERNET_INCORRECT_PASSWORD   uint32 = INTERNET_ERROR_BASE + 14
	ERROR_INTERNET_FORCE_RETRY          uint32 = INTERNET_ERROR_BASE + 32
)

// Client is a struct containing relevant metadata to make HTTP
// requests.
type Client struct {
	handle          uintptr
	Timeout         time.Duration
	TLSClientConfig struct {
		InsecureSkipVerify bool
	}
	CookieJar     *Jar
	AskProxyCreds bool
}

// NewClient will return a pointer to a new Client instance that
// simply wraps the net/http.Client type.
func NewClient(userAgent string) (*Client, error) {
	var client = &Client{
		CookieJar: cookieJar(),
	}
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

	for _, cookie := range c.CookieJar.Cookies(request.URL) {
		req.AddCookie(cookie)
	}

	if err = sendRequest(reqHandle, req); err != nil {
		return nil, err
	}

	if resp, err = buildResponse(reqHandle, req); err != nil {
		return nil, err
	}

	if c.AskProxyCreds {
		if resp.StatusCode == 407 {
			err := promptUserPassword(reqHandle)
			if err != nil {
				return nil, err
			}
			if err = sendRequest(reqHandle, req); err != nil {
				return nil, err
			}
			if resp, err = buildResponse(reqHandle, req); err != nil {
				return nil, err
			}
		}
	}

	c.CookieJar.cookies = resp.Cookies()

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

func promptUserPassword(reqHandle uintptr) error {
	var lppvData []byte
	dwError := ERROR_INTERNET_INCORRECT_PASSWORD
	dwFlags := FLAGS_ERROR_UI_FILTER_FOR_ERRORS | FLAGS_ERROR_UI_FLAGS_CHANGE_OPTIONS | FLAGS_ERROR_UI_FLAGS_GENERATE_DATA
	_, err := InternetErrorDlg(GetDesktopWindow(), reqHandle, dwError, dwFlags, &lppvData)
	if err != nil {
		return err
	}
	return nil
}

// Jar - CookieJar implementation that ignores domains/origins
type Jar struct {
	lk      sync.Mutex
	cookies []*Cookie
}

func cookieJar() *Jar {
	return &Jar{
		lk:      sync.Mutex{},
		cookies: []*Cookie{},
	}
}

// NewJar - Get a new instance of a cookie jar
func NewJar() *Jar {
	jar := new(Jar)
	jar.cookies = make([]*Cookie, 0)
	return jar
}

// SetCookies handles the receipt of the cookies in a reply for the
// given URL (which is ignored).
func (jar *Jar) SetCookies(u *url.URL, cookies []*Cookie) {
	jar.lk.Lock()
	jar.cookies = append(jar.cookies, cookies...)
	jar.lk.Unlock()
}

// Cookies returns the cookies to send in a request for the given URL.
// It is up to the implementation to honor the standard cookie use
// restrictions such as in RFC 6265 (which we do not).
func (jar *Jar) Cookies(u *url.URL) []*Cookie {
	return jar.cookies
}
