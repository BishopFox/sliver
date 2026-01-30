package oauth1

import (
	"context"
	"net/http"
)

type contextKey struct{}

// HTTPClient is the context key to associate an *http.Client value with
// a context.
var HTTPClient contextKey

// NoContext is the default context to use in most cases.
var NoContext = context.TODO()

// contextTransport gets the Transport from the context client or nil.
func contextTransport(ctx context.Context) http.RoundTripper {
	if client, ok := ctx.Value(HTTPClient).(*http.Client); ok {
		return client.Transport
	}
	return nil
}
