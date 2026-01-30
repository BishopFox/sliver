package reddit

import "net/http"

// cloneRequest returns a clone of the provided *http.Request.
// The clone is a shallow copy of the struct and its Header map,
// since we'll only be modify the headers.
// Per the specification of http.RoundTripper, we should not directly modify a request.
func cloneRequest(r *http.Request) *http.Request {
	r2 := new(http.Request)
	*r2 = *r
	// deep copy of the Header
	r2.Header = make(http.Header, len(r.Header))
	for k, s := range r.Header {
		r2.Header[k] = append([]string(nil), s...)
	}
	return r2
}

// Sets the User-Agent header for requests.
// We need to set a custom user agent because using the one set by the
// stdlib gives us 429 Too Many Requests responses from the Reddit API.
type userAgentTransport struct {
	userAgent string
	Base      http.RoundTripper
}

func (t *userAgentTransport) setUserAgent(req *http.Request) *http.Request {
	req2 := cloneRequest(req)
	req2.Header.Set(headerUserAgent, t.userAgent)
	return req2
}

func (t *userAgentTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req2 := t.setUserAgent(req)
	return t.base().RoundTrip(req2)
}

func (t *userAgentTransport) base() http.RoundTripper {
	if t.Base != nil {
		return t.Base
	}
	return http.DefaultTransport
}
