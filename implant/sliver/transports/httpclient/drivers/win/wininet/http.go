package wininet

import "errors"

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
