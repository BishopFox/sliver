//go:build go1.7
// +build go1.7

package twilio

import "net/http"

// MediaClient is used for fetching images and does not follow redirects.
var MediaClient = http.Client{
	Timeout: defaultTimeout,
	CheckRedirect: func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	},
}
