// Versions older than Go 1.7 are no longer supported, though this _may_ work
// for you.

//go:build !go1.7
// +build !go1.7

package twilio

import (
	"net/http"

	"golang.org/x/net/context"
)

func withContext(r *http.Request, ctx context.Context) *http.Request {
	return r
}
