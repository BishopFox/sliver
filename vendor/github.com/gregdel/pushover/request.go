package pushover

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
)

// do is a generic function to send a request to the API.
func do(req *http.Request, resType interface{}, returnHeaders bool) error {
	client := http.DefaultClient

	// Send request
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Only 500 errors will not respond a readable result
	if resp.StatusCode >= http.StatusInternalServerError {
		return ErrHTTPPushover
	}

	// Decode the JSON response
	if err := json.NewDecoder(resp.Body).Decode(&resType); err != nil {
		return err
	}

	// Check if the unmarshaled data is a response
	r, ok := resType.(*Response)
	if !ok {
		return nil
	}

	// Check response status
	if r.Status != 1 {
		return r.Errors
	}

	// The headers are only returned when posting a new notification
	if returnHeaders {
		// Get app limits from headers
		appLimits, err := newLimit(resp.Header)
		if err != nil {
			return err
		}
		r.Limit = appLimits
	}

	return nil
}

// urlEncodedRequest returns a new url encoded request.
func newURLEncodedRequest(method, endpoint string, params map[string]string) (*http.Request, error) {
	urlValues := url.Values{}
	for k, v := range params {
		urlValues.Add(k, v)
	}

	req, err := http.NewRequest(method, endpoint, strings.NewReader(urlValues.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	return req, nil
}
