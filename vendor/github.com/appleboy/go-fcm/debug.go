package fcm

import (
	"log"
	"net/http"
	"net/http/httputil"
)

type debugTransport struct {
	t http.RoundTripper
}

func (d debugTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	reqDump, err := httputil.DumpRequest(req, true)
	if err != nil {
		// RoundTrip must always close the request body, including on errors,
		// before returning without delegating to the wrapped transport.
		if req.Body != nil {
			_ = req.Body.Close()
		}
		return nil, err
	}
	//nolint:gosec // debug-only HTTP dump
	log.Printf("%s", string(reqDump))

	resp, err := d.t.RoundTrip(req)
	if err != nil {
		return nil, err
	}

	respDump, err := httputil.DumpResponse(resp, true)
	if err != nil {
		if cerr := resp.Body.Close(); cerr != nil {
			log.Printf("error closing response body: %v", cerr)
		}
		return nil, err
	}
	//nolint:gosec // debug-only HTTP dump
	log.Printf("%s", string(respDump))
	return resp, nil
}
