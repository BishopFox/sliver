package keyfunc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

// ErrInvalidHTTPStatusCode indicates that the HTTP status code is invalid.
var ErrInvalidHTTPStatusCode = errors.New("invalid HTTP status code")

// Options represents the configuration options for a JWKS.
//
// If RefreshInterval and or RefreshUnknownKID is not nil, then a background goroutine will be launched to refresh the
// remote JWKS under the specified circumstances.
//
// When using a background refresh goroutine, make sure to use RefreshRateLimit if paired with RefreshUnknownKID. Also
// make sure to end the background refresh goroutine with the JWKS.EndBackground method when it's no longer needed.
type Options struct {
	// Client is the HTTP client used to get the JWKS via HTTP.
	Client *http.Client

	// Ctx is the context for the keyfunc's background refresh. When the context expires or is canceled, the background
	// goroutine will end.
	Ctx context.Context

	// GivenKeys is a map of JWT key IDs, `kid`, to their given keys. If the JWKS has a background refresh goroutine,
	// these values persist across JWKS refreshes. By default, if the remote JWKS resource contains a key with the same
	// `kid` any given keys with the same `kid` will be overwritten by the keys from the remote JWKS. Use the
	// GivenKIDOverride option to flip this behavior.
	GivenKeys map[string]GivenKey

	// GivenKIDOverride will make a GivenKey override any keys with the same ID (`kid`) in the remote JWKS. The is only
	// effectual if GivenKeys is provided.
	GivenKIDOverride bool

	// JWKUseWhitelist is a whitelist of JWK `use` parameter values that will restrict what keys can be returned for
	// jwt.Keyfunc. The assumption is that jwt.Keyfunc is only used for JWT signature verification.
	// The default behavior is to only return a JWK if its `use` parameter has the value `"sig"`, an empty string, or if
	// the parameter was omitted entirely.
	JWKUseWhitelist []JWKUse

	// JWKUseNoWhitelist overrides the JWKUseWhitelist field and its default behavior. If set to true, all JWKs will be
	// returned regardless of their `use` parameter value.
	JWKUseNoWhitelist bool

	// RefreshErrorHandler is a function that consumes errors that happen during a JWKS refresh. This is only effectual
	// if a background refresh goroutine is active.
	RefreshErrorHandler ErrorHandler

	// RefreshInterval is the duration to refresh the JWKS in the background via a new HTTP request. If this is not nil,
	// then a background goroutine will be used to refresh the JWKS once per the given interval. Make sure to call the
	// JWKS.EndBackground method to end this goroutine when it's no longer needed.
	RefreshInterval time.Duration

	// RefreshRateLimit limits the rate at which refresh requests are granted. Only one refresh request can be queued
	// at a time any refresh requests received while there is already a queue are ignored. It does not make sense to
	// have RefreshInterval's value shorter than this.
	RefreshRateLimit time.Duration

	// RefreshTimeout is the duration for the context timeout used to create the HTTP request for a refresh of the JWKS.
	// This defaults to one minute. This is used for the HTTP request and any background goroutine refreshes.
	RefreshTimeout time.Duration

	// RefreshUnknownKID indicates that the JWKS refresh request will occur every time a kid that isn't cached is seen.
	// This is done through a background goroutine. Without specifying a RefreshInterval a malicious client could
	// self-sign X JWTs, send them to this service, then cause potentially high network usage proportional to X. Make
	// sure to call the JWKS.EndBackground method to end this goroutine when it's no longer needed.
	//
	// It is recommended this option is not used when in MultipleJWKS. This is because KID collisions SHOULD be uncommon
	// meaning nearly any JWT SHOULD trigger a refresh for the number of JWKS in the MultipleJWKS minus one.
	RefreshUnknownKID bool

	// RequestFactory creates HTTP requests for the remote JWKS resource located at the given url. For example, an
	// HTTP header could be added to indicate a User-Agent.
	RequestFactory func(ctx context.Context, url string) (*http.Request, error)

	// ResponseExtractor consumes a *http.Response and produces the raw JSON for the JWKS. By default, the
	// ResponseExtractorStatusOK function is used. The default behavior changed in v1.4.0.
	ResponseExtractor func(ctx context.Context, resp *http.Response) (json.RawMessage, error)
}

// MultipleOptions is used to configure the behavior when multiple JWKS are used by MultipleJWKS.
type MultipleOptions struct {
	// KeySelector is a function that selects the key to use for a given token. It will be used in the implementation
	// for jwt.Keyfunc. If implementing this custom selector extract the key ID and algorithm from the token's header.
	// Use the key ID to select a token and confirm the key's algorithm before returning it.
	//
	// This value defaults to KeySelectorFirst.
	KeySelector func(multiJWKS *MultipleJWKS, token *jwt.Token) (key interface{}, err error)
}

// RefreshOptions are used to specify manual refresh behavior.
type RefreshOptions struct {
	IgnoreRateLimit bool
}

type refreshRequest struct {
	cancel          context.CancelFunc
	ignoreRateLimit bool
}

// ResponseExtractorStatusOK is meant to be used as the ResponseExtractor field for Options. It confirms that response
// status code is 200 OK and returns the raw JSON from the response body.
func ResponseExtractorStatusOK(ctx context.Context, resp *http.Response) (json.RawMessage, error) {
	//goland:noinspection GoUnhandledErrorResult
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: %d", ErrInvalidHTTPStatusCode, resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}

// ResponseExtractorStatusAny is meant to be used as the ResponseExtractor field for Options. It returns the raw JSON
// from the response body regardless of the response status code.
func ResponseExtractorStatusAny(ctx context.Context, resp *http.Response) (json.RawMessage, error) {
	//goland:noinspection GoUnhandledErrorResult
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

// applyOptions applies the given options to the given JWKS.
func applyOptions(jwks *JWKS, options Options) {
	if options.Ctx != nil {
		jwks.ctx, jwks.cancel = context.WithCancel(options.Ctx)
	}

	if options.GivenKeys != nil {
		jwks.givenKeys = make(map[string]GivenKey)
		for kid, key := range options.GivenKeys {
			jwks.givenKeys[kid] = key
		}
	}

	if !options.JWKUseNoWhitelist {
		jwks.jwkUseWhitelist = make(map[JWKUse]struct{})
		for _, use := range options.JWKUseWhitelist {
			jwks.jwkUseWhitelist[use] = struct{}{}
		}
	}

	jwks.client = options.Client
	jwks.givenKIDOverride = options.GivenKIDOverride
	jwks.refreshErrorHandler = options.RefreshErrorHandler
	jwks.refreshInterval = options.RefreshInterval
	jwks.refreshRateLimit = options.RefreshRateLimit
	jwks.refreshTimeout = options.RefreshTimeout
	jwks.refreshUnknownKID = options.RefreshUnknownKID
	jwks.requestFactory = options.RequestFactory
	jwks.responseExtractor = options.ResponseExtractor
}
