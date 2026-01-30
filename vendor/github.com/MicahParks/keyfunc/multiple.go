package keyfunc

import (
	"errors"
	"fmt"

	"github.com/golang-jwt/jwt/v4"
)

// ErrMultipleJWKSSize is returned when the number of JWKS given are not enough to make a MultipleJWKS.
var ErrMultipleJWKSSize = errors.New("multiple JWKS must have two or more remote JWK Set resources")

// MultipleJWKS manages multiple JWKS and has a field for jwt.Keyfunc.
type MultipleJWKS struct {
	keySelector func(multiJWKS *MultipleJWKS, token *jwt.Token) (key interface{}, err error)
	sets        map[string]*JWKS // No lock is required because this map is read-only after initialization.
}

// GetMultiple creates a new MultipleJWKS. A map of length two or more JWKS URLs to Options is required.
//
// Be careful when choosing Options for each JWKS in the map. If RefreshUnknownKID is set to true for all JWKS in the
// map then many refresh requests would take place each time a JWT is processed, this should be rate limited by
// RefreshRateLimit.
func GetMultiple(multiple map[string]Options, options MultipleOptions) (multiJWKS *MultipleJWKS, err error) {
	if multiple == nil || len(multiple) < 2 {
		return nil, fmt.Errorf("multiple JWKS must have two or more remote JWK Set resources: %w", ErrMultipleJWKSSize)
	}

	if options.KeySelector == nil {
		options.KeySelector = KeySelectorFirst
	}

	multiJWKS = &MultipleJWKS{
		sets:        make(map[string]*JWKS, len(multiple)),
		keySelector: options.KeySelector,
	}

	for u, opts := range multiple {
		jwks, err := Get(u, opts)
		if err != nil {
			return nil, fmt.Errorf("failed to get JWKS from %q: %w", u, err)
		}
		multiJWKS.sets[u] = jwks
	}

	return multiJWKS, nil
}

func (m *MultipleJWKS) JWKSets() map[string]*JWKS {
	sets := make(map[string]*JWKS, len(m.sets))
	for u, jwks := range m.sets {
		sets[u] = jwks
	}
	return sets
}

func KeySelectorFirst(multiJWKS *MultipleJWKS, token *jwt.Token) (key interface{}, err error) {
	kid, alg, err := kidAlg(token)
	if err != nil {
		return nil, err
	}
	for _, jwks := range multiJWKS.sets {
		key, err = jwks.getKey(alg, kid)
		if err == nil {
			return key, nil
		}
	}
	return nil, fmt.Errorf("failed to find key ID in multiple JWKS: %w", ErrKIDNotFound)
}
