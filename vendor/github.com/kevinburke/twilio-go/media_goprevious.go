//go:build !go1.7
// +build !go1.7

package twilio

import (
	"errors"
	"net/http"
)

// MediaClient is used for fetching images and does not follow redirects.
var MediaClient = http.Client{
	Timeout: defaultTimeout,
	CheckRedirect: func(req *http.Request, via []*http.Request) error {
		// TODO not sure if this works.
		return errors.New("use last response")
	},
}
