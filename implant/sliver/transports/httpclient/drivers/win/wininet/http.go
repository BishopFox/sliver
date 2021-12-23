package wininet

import "errors"

// DefaultClient is the default client similar to net/http.
var DefaultClient *Client

// ErrNoCookie is returned by Request's Cookie method when a cookie is
// not found.
var ErrNoCookie = errors.New("named cookie not present")

// Common HTTP methods.
const (
	MethodConnect string = "CONNECT"
	MethodDelete  string = "DELETE"
	MethodGet     string = "GET"
	MethodHead    string = "HEAD"
	MethodOptions string = "OPTIONS"
	MethodPatch   string = "PATCH"
	MethodPost    string = "POST"
	MethodPut     string = "PUT"
	MethodTrace   string = "TRACE"
)

// Get will make a GET request using the DefaultClient.
func Get(url string) (*Response, error) {
	return DefaultClient.Get(url)
}

// Head will make a HEAD request using the DefaultClient.
func Head(url string) (*Response, error) {
	return DefaultClient.Head(url)
}

func init() {
	DefaultClient, _ = NewClient()
}

// Post will make a POST request using the DefaultClient.
func Post(url, contentType string, body []byte) (*Response, error) {
	return DefaultClient.Post(url, contentType, body)
}
