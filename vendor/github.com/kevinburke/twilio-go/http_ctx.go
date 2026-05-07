//go:build go1.7
// +build go1.7

package twilio

import (
	"context"
	"net/http"
)

func withContext(r *http.Request, ctx context.Context) *http.Request {
	return r.WithContext(ctx)
}
