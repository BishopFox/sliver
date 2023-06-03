package wininet

import (
	"io"
)

// Response is a struct containing common HTTP response data.
type Response struct {
	Body          io.ReadCloser
	cookies       []*Cookie
	ContentLength int64
	Header        map[string][]string
	Proto         string
	ProtoMajor    int
	ProtoMinor    int
	Status        string
	StatusCode    int
}

// AddCookie will add a Cookie to the Request.
func (r *Response) AddCookie(cookie *Cookie) {
	for i, c := range r.cookies {
		if cookie.Name == c.Name {
			r.cookies = append(r.cookies[:i], r.cookies[i+1:]...)
			break
		}
	}

	r.cookies = append(r.cookies, cookie)
}

// Cookie will return the named Cookie provided in the Response or
// ErrNoCookie, if not found.
func (r *Response) Cookie(name string) (*Cookie, error) {
	for _, c := range r.cookies {
		if name == c.Name {
			return c, nil
		}
	}

	return nil, ErrNoCookie
}

// Cookies will return the HTTP cookies provided in the Response.
func (r *Response) Cookies() []*Cookie {
	return r.cookies
}
