package restclient

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"runtime"
	"strings"
	"sync/atomic"
	"time"

	"github.com/kevinburke/rest/resterror"
)

type UploadType string

// JSON specifies you'd like to upload JSON data.
var JSON UploadType = "application/json"

// FormURLEncoded specifies you'd like to upload form-urlencoded data.
var FormURLEncoded UploadType = "application/x-www-form-urlencoded"

const Version = "2.12.0"

var ua string

func init() {
	gv := strings.Replace(runtime.Version(), "go", "", 1)
	ua = fmt.Sprintf("rest-client/%s (https://github.com/kevinburke/rest) go/%s (%s/%s)",
		Version, gv, runtime.GOOS, runtime.GOARCH)
}

// Client is a generic Rest client for making HTTP requests.
type Client struct {
	// Username for use in HTTP Basic Auth
	ID string
	// HTTP Client to use for making requests
	Client *http.Client
	// The base URL for all requests to this API, for example,
	// "https://fax.twilio.com/v1"
	Base string
	// Set UploadType to JSON or FormURLEncoded to control how data is sent to
	// the server. Defaults to FormURLEncoded.
	UploadType UploadType
	// ErrorParser is invoked when the client gets a 400-or-higher status code
	// from the server. Defaults to rest.DefaultErrorParser.
	ErrorParser func(*http.Response) error

	useBearerAuth bool

	// Password for use in HTTP Basic Auth, or single token in Bearer auth
	token atomic.Value
}

func (c *Client) Token() string {
	l := c.token.Load()
	s, _ := l.(string)
	return s
}

// New returns a new Client with HTTP Basic Auth with the given user and
// password. Base is the scheme+domain to hit for all requests.
func New(user, pass, base string) *Client {

	c := &Client{
		ID:          user,
		Client:      defaultHttpClient,
		Base:        base,
		UploadType:  JSON,
		ErrorParser: DefaultErrorParser,
	}
	c.token.Store(pass)
	return c
}

// NewBearerClient returns a new Client configured to use Bearer authentication.
func NewBearerClient(token, base string) *Client {
	c := &Client{
		ID:            "",
		Client:        defaultHttpClient,
		Base:          base,
		UploadType:    JSON,
		ErrorParser:   DefaultErrorParser,
		useBearerAuth: true,
	}
	c.token.Store(token)
	return c
}

func (c *Client) UpdateToken(newToken string) {
	c.token.Store(newToken)
}

var defaultDialer = &net.Dialer{
	Timeout:   30 * time.Second,
	KeepAlive: 30 * time.Second,
	DualStack: true,
}

// DialSocket configures c to use the provided socket and http.Transport to
// dial a Unix socket instead of a TCP port.
//
// If transport is nil, the settings from DefaultTransport are used.
func (c *Client) DialSocket(socket string, transport *http.Transport) {
	dialSock := func(ctx context.Context, proto, addr string) (conn net.Conn, err error) {
		return defaultDialer.DialContext(ctx, "unix", socket)
	}
	if transport == nil {
		ht := http.DefaultTransport.(*http.Transport)
		transport = &http.Transport{
			Proxy:                 ht.Proxy,
			MaxIdleConns:          ht.MaxIdleConns,
			IdleConnTimeout:       ht.IdleConnTimeout,
			TLSHandshakeTimeout:   ht.TLSHandshakeTimeout,
			ExpectContinueTimeout: ht.ExpectContinueTimeout,
			DialContext:           dialSock,
		}
	}
	if c.Client == nil {
		// need to copy this so we don't modify the default client
		c.Client = &http.Client{
			Timeout: defaultHttpClient.Timeout,
		}
	}
	switch tp := c.Client.Transport.(type) {
	// TODO both of these cases clobbber the existing transport which isn't
	// ideal.
	case nil, *Transport:
		c.Client.Transport = &Transport{
			RoundTripper: transport,
			Debug:        DefaultTransport.Debug,
			Output:       DefaultTransport.Output,
		}
	case *http.Transport:
		c.Client.Transport = transport
	default:
		panic(fmt.Sprintf("could not set DialSocket on unknown transport: %#v", tp))
	}
}

func (c *Client) NewRequestWithContext(ctx context.Context, method, path string, body io.Reader) (*http.Request, error) {
	if c == nil {
		panic("cannot call NewRequestWithContext on nil *Client")
	}
	// see for example https://github.com/meterup/github-release/issues/1 - if
	// the path contains the full URL including the base, strip it out
	path = strings.TrimPrefix(path, c.Base)
	req, err := http.NewRequestWithContext(ctx, method, c.Base+path, body)
	if err != nil {
		return nil, err
	}
	token := c.Token()
	switch {
	case c.useBearerAuth && token != "":
		req.Header.Add("Authorization", "Bearer "+token)
	case !c.useBearerAuth && (c.ID != "" || token != ""):
		req.SetBasicAuth(c.ID, token)
	}
	req.Header.Add("User-Agent", ua)
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Accept-Charset", "utf-8")
	if method == "POST" || method == "PUT" {
		uploadType := c.UploadType
		if uploadType == "" {
			uploadType = JSON
		}
		req.Header.Add("Content-Type", string(uploadType)+"; charset=utf-8")
	}
	return req, nil
}

// NewRequest creates a new Request and sets basic auth based on the client's
// authentication information.
func (c *Client) NewRequest(method, path string, body io.Reader) (*http.Request, error) {
	return c.NewRequestWithContext(context.Background(), method, path, body)
}

// Do performs the HTTP request. If the HTTP response is in the 2xx range,
// Unmarshal the response body into v. If the response status code is 400 or
// above, attempt to Unmarshal the response into an Error. Otherwise return
// a generic http error.
func (c *Client) Do(r *http.Request, v interface{}) error {
	var res *http.Response
	var err error
	if c.Client == nil {
		res, err = defaultHttpClient.Do(r)
	} else {
		res, err = c.Client.Do(r)
	}
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode >= 400 {
		if c.ErrorParser != nil {
			return c.ErrorParser(res)
		}
		return DefaultErrorParser(res)
	}

	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}
	if v == nil || res.StatusCode == http.StatusNoContent {
		return nil
	} else {
		return json.Unmarshal(resBody, v)
	}
}

// DefaultErrorParser attempts to parse the response body as a rest.Error. If
// it cannot do so, return an error containing the entire response body.
func DefaultErrorParser(resp *http.Response) error {
	resBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	rerr := new(resterror.Error)
	err = json.Unmarshal(resBody, rerr)
	if err != nil {
		return fmt.Errorf("invalid response body: %s", string(resBody))
	}
	if rerr.Title == "" {
		return fmt.Errorf("invalid response body: %s", string(resBody))
	} else {
		rerr.Status = resp.StatusCode
		return rerr
	}
}
