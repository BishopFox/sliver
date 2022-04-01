package wininet

// Request is a struct containing common HTTP request data.
type Request struct {
	Body    []byte
	cookies []*Cookie
	Headers map[string]string
	Method  string
	URL     string
}

// NewRequest will return a pointer to a new Request instasnce.
func NewRequest(method, url string, body ...[]byte) *Request {
	var b []byte

	if len(body) > 0 {
		b = body[0]
	}

	return &Request{
		Body:    b,
		Headers: map[string]string{},
		Method:  method,
		URL:     url,
	}
}

// AddCookie will add a Cookie to the Request.
func (r *Request) AddCookie(cookie *Cookie) {
	for i, c := range r.cookies {
		if cookie.Name == c.Name {
			r.cookies = append(r.cookies[:i], r.cookies[i+1:]...)
			break
		}
	}

	r.cookies = append(r.cookies, cookie)
}

// Cookie will return the named Cookie provided in the Request or
// ErrNoCookie, if not found.
func (r *Request) Cookie(name string) (*Cookie, error) {
	for _, c := range r.cookies {
		if name == c.Name {
			return c, nil
		}
	}

	return nil, ErrNoCookie
}

// Cookies will return the HTTP cookies provided in the Request.
func (r *Request) Cookies() []*Cookie {
	return r.cookies
}
