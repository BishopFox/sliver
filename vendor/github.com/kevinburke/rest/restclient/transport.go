package restclient

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httputil"
	"os"
)

// DefaultTransport is like http.DefaultTransport, but prints the contents of
// HTTP requests to os.Stderr if the DEBUG_HTTP_TRAFFIC environment variable is
// set to true.
var DefaultTransport *Transport
var defaultHttpClient *http.Client

func init() {
	DefaultTransport = &Transport{
		RoundTripper: http.DefaultTransport,
		Debug:        os.Getenv("DEBUG_HTTP_TRAFFIC") == "true",
		Output:       os.Stderr,
	}
	defaultHttpClient = &http.Client{
		Transport: DefaultTransport,
	}
}

// Transport implements HTTP round trips, but adds hooks for debugging the HTTP
// request.
type Transport struct {
	// The underlying RoundTripper.
	RoundTripper http.RoundTripper
	// Whether to write the HTTP request and response contents to Output.
	Debug bool
	// If Debug is true, write the HTTP request and response contents here. If
	// Output is nil, os.Stderr will be used.
	Output io.Writer
}

func (t *Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	if t == nil {
		panic("nil Transport")
	}
	var w io.ReadWriter = nil
	if t.Debug {
		w = new(bytes.Buffer)
		bits, err := httputil.DumpRequestOut(req, true)
		if err != nil {
			return nil, err
		}
		if len(bits) > 0 && bits[len(bits)-1] != '\n' {
			bits = append(bits, '\n')
		}
		w.Write(bits)
	}
	var res *http.Response
	var err error
	if t.RoundTripper == nil {
		res, err = http.DefaultTransport.RoundTrip(req)
	} else {
		res, err = t.RoundTripper.RoundTrip(req)
	}
	if err != nil {
		return res, err
	}
	if t.Debug {
		bits, err := httputil.DumpResponse(res, true)
		if err != nil {
			return res, err
		}
		if len(bits) > 0 && bits[len(bits)-1] != '\n' {
			bits = append(bits, '\n')
		}
		w.Write(bits)
		if t.Output == nil {
			t.Output = os.Stderr
		}
		_, err = io.Copy(t.Output, w)
		if err != nil {
			return res, err
		}
	}
	return res, err
}
