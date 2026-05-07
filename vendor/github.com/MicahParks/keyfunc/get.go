package keyfunc

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"
)

var (
	// ErrRefreshImpossible is returned when a refresh is attempted on a JWKS that was not created from a remote
	// resource.
	ErrRefreshImpossible = errors.New("refresh impossible: JWKS was not created from a remote resource")

	// defaultRefreshTimeout is the default duration for the context used to create the HTTP request for a refresh of
	// the JWKS.
	defaultRefreshTimeout = time.Minute
)

// Get loads the JWKS at the given URL.
func Get(jwksURL string, options Options) (jwks *JWKS, err error) {
	jwks = &JWKS{
		jwksURL: jwksURL,
	}

	applyOptions(jwks, options)

	if jwks.client == nil {
		jwks.client = http.DefaultClient
	}
	if jwks.requestFactory == nil {
		jwks.requestFactory = defaultRequestFactory
	}
	if jwks.responseExtractor == nil {
		jwks.responseExtractor = ResponseExtractorStatusOK
	}
	if jwks.refreshTimeout == 0 {
		jwks.refreshTimeout = defaultRefreshTimeout
	}
	if !options.JWKUseNoWhitelist && len(jwks.jwkUseWhitelist) == 0 {
		jwks.jwkUseWhitelist = map[JWKUse]struct{}{
			UseOmitted:   {},
			UseSignature: {},
		}
	}

	err = jwks.refresh()
	if err != nil {
		return nil, err
	}

	if jwks.refreshInterval != 0 || jwks.refreshUnknownKID {
		jwks.ctx, jwks.cancel = context.WithCancel(context.Background())
		jwks.refreshRequests = make(chan refreshRequest, 1)
		go jwks.backgroundRefresh()
	}

	return jwks, nil
}

// Refresh manually refreshes the JWKS with the remote resource. It can bypass the rate limit if configured to do so.
// This function will return an ErrRefreshImpossible if the JWKS was created from a static source like given keys or raw
// JSON, because there is no remote resource to refresh from.
//
// This function will block until the refresh is finished or an error occurs.
func (j *JWKS) Refresh(ctx context.Context, options RefreshOptions) error {
	if j.jwksURL == "" {
		return ErrRefreshImpossible
	}

	// Check if the background goroutine was launched.
	if j.refreshInterval != 0 || j.refreshUnknownKID {
		ctx, cancel := context.WithCancel(ctx)

		req := refreshRequest{
			cancel:          cancel,
			ignoreRateLimit: options.IgnoreRateLimit,
		}

		select {
		case <-ctx.Done():
			return fmt.Errorf("failed to send request refresh to background goroutine: %w", j.ctx.Err())
		case j.refreshRequests <- req:
		}

		<-ctx.Done()

		if !errors.Is(ctx.Err(), context.Canceled) {
			return fmt.Errorf("unexpected keyfunc background refresh context error: %w", ctx.Err())
		}
	} else {
		err := j.refresh()
		if err != nil {
			return fmt.Errorf("failed to refresh JWKS: %w", err)
		}
	}

	return nil
}

// backgroundRefresh is meant to be a separate goroutine that will update the keys in a JWKS over a given interval of
// time.
func (j *JWKS) backgroundRefresh() {
	var lastRefresh time.Time
	var queueOnce sync.Once
	var refreshMux sync.Mutex
	if j.refreshRateLimit != 0 {
		lastRefresh = time.Now().Add(-j.refreshRateLimit)
	}

	// Create a channel that will never send anything unless there is a refresh interval.
	refreshInterval := make(<-chan time.Time)

	refresh := func() {
		err := j.refresh()
		if err != nil && j.refreshErrorHandler != nil {
			j.refreshErrorHandler(err)
		}
		lastRefresh = time.Now()
	}

	// Enter an infinite loop that ends when the background ends.
	for {
		if j.refreshInterval != 0 {
			refreshInterval = time.After(j.refreshInterval)
		}

		select {
		case <-refreshInterval:
			select {
			case <-j.ctx.Done():
				return
			case j.refreshRequests <- refreshRequest{}:
			default: // If the j.refreshRequests channel is full, don't send another request.
			}

		case req := <-j.refreshRequests:
			refreshMux.Lock()
			if req.ignoreRateLimit {
				refresh()
			} else if j.refreshRateLimit != 0 && lastRefresh.Add(j.refreshRateLimit).After(time.Now()) {
				// Launch a goroutine that will get a reservation for a JWKS refresh or fail to and immediately return.
				queueOnce.Do(func() {
					go func() {
						refreshMux.Lock()
						wait := time.Until(lastRefresh.Add(j.refreshRateLimit))
						refreshMux.Unlock()
						select {
						case <-j.ctx.Done():
							return
						case <-time.After(wait):
						}

						refreshMux.Lock()
						defer refreshMux.Unlock()
						refresh()
						queueOnce = sync.Once{}
					}()
				})
			} else {
				refresh()
			}
			if req.cancel != nil {
				req.cancel()
			}
			refreshMux.Unlock()

		// Clean up this goroutine when its context expires.
		case <-j.ctx.Done():
			return
		}
	}
}

func defaultRequestFactory(ctx context.Context, url string) (*http.Request, error) {
	return http.NewRequestWithContext(ctx, http.MethodGet, url, bytes.NewReader(nil))
}

// refresh does an HTTP GET on the JWKS URL to rebuild the JWKS.
func (j *JWKS) refresh() (err error) {
	var ctx context.Context
	var cancel context.CancelFunc
	if j.ctx != nil {
		ctx, cancel = context.WithTimeout(j.ctx, j.refreshTimeout)
	} else {
		ctx, cancel = context.WithTimeout(context.Background(), j.refreshTimeout)
	}
	defer cancel()

	req, err := j.requestFactory(ctx, j.jwksURL)
	if err != nil {
		return fmt.Errorf("failed to create request via factory function: %w", err)
	}

	resp, err := j.client.Do(req)
	if err != nil {
		return err
	}

	jwksBytes, err := j.responseExtractor(ctx, resp)
	if err != nil {
		return fmt.Errorf("failed to extract response via extractor function: %w", err)
	}

	// Only reprocess if the JWKS has changed.
	if len(jwksBytes) != 0 && bytes.Equal(jwksBytes, j.raw) {
		return nil
	}
	j.raw = jwksBytes

	updated, err := NewJSON(jwksBytes)
	if err != nil {
		return err
	}

	j.mux.Lock()
	defer j.mux.Unlock()
	j.keys = updated.keys

	if j.givenKeys != nil {
		for kid, key := range j.givenKeys {
			// Only overwrite the key if configured to do so.
			if !j.givenKIDOverride {
				if _, ok := j.keys[kid]; ok {
					continue
				}
			}

			j.keys[kid] = parsedJWK{public: key.inter}
		}
	}

	return nil
}
