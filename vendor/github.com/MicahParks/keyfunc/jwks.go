package keyfunc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"
)

var (
	// ErrJWKAlgMismatch indicates that the given JWK was found, but its "alg" parameter's value did not match that of
	// the JWT.
	ErrJWKAlgMismatch = errors.New(`the given JWK was found, but its "alg" parameter's value did not match the expected algorithm`)

	// ErrJWKUseWhitelist indicates that the given JWK was found, but its "use" parameter's value was not whitelisted.
	ErrJWKUseWhitelist = errors.New(`the given JWK was found, but its "use" parameter's value was not whitelisted`)

	// ErrKIDNotFound indicates that the given key ID was not found in the JWKS.
	ErrKIDNotFound = errors.New("the given key ID was not found in the JWKS")

	// ErrMissingAssets indicates there are required assets are missing to create a public key.
	ErrMissingAssets = errors.New("required assets are missing to create a public key")
)

// ErrorHandler is a function signature that consumes an error.
type ErrorHandler func(err error)

const (
	// UseEncryption is a JWK "use" parameter value indicating the JSON Web Key is to be used for encryption.
	UseEncryption JWKUse = "enc"
	// UseOmitted is a JWK "use" parameter value that was not specified or was empty.
	UseOmitted JWKUse = ""
	// UseSignature is a JWK "use" parameter value indicating the JSON Web Key is to be used for signatures.
	UseSignature JWKUse = "sig"
)

// JWKUse is a set of values for the "use" parameter of a JWK.
// See https://tools.ietf.org/html/rfc7517#section-4.2.
type JWKUse string

// jsonWebKey represents a JSON Web Key inside a JWKS.
type jsonWebKey struct {
	Algorithm string `json:"alg"`
	Curve     string `json:"crv"`
	Exponent  string `json:"e"`
	K         string `json:"k"`
	ID        string `json:"kid"`
	Modulus   string `json:"n"`
	Type      string `json:"kty"`
	Use       string `json:"use"`
	X         string `json:"x"`
	Y         string `json:"y"`
}

// parsedJWK represents a JSON Web Key parsed with fields as the correct Go types.
type parsedJWK struct {
	algorithm string
	public    interface{}
	use       JWKUse
}

// JWKS represents a JSON Web Key Set (JWK Set).
type JWKS struct {
	jwkUseWhitelist     map[JWKUse]struct{}
	cancel              context.CancelFunc
	client              *http.Client
	ctx                 context.Context
	raw                 []byte
	givenKeys           map[string]GivenKey
	givenKIDOverride    bool
	jwksURL             string
	keys                map[string]parsedJWK
	mux                 sync.RWMutex
	refreshErrorHandler ErrorHandler
	refreshInterval     time.Duration
	refreshRateLimit    time.Duration
	refreshRequests     chan refreshRequest
	refreshTimeout      time.Duration
	refreshUnknownKID   bool
	requestFactory      func(ctx context.Context, url string) (*http.Request, error)
	responseExtractor   func(ctx context.Context, resp *http.Response) (json.RawMessage, error)
}

// rawJWKS represents a JWKS in JSON format.
type rawJWKS struct {
	Keys []*jsonWebKey `json:"keys"`
}

// NewJSON creates a new JWKS from a raw JSON message.
func NewJSON(jwksBytes json.RawMessage) (jwks *JWKS, err error) {
	var rawKS rawJWKS
	err = json.Unmarshal(jwksBytes, &rawKS)
	if err != nil {
		return nil, err
	}

	// Iterate through the keys in the raw JWKS. Add them to the JWKS.
	jwks = &JWKS{
		keys: make(map[string]parsedJWK, len(rawKS.Keys)),
	}
	for _, key := range rawKS.Keys {
		var keyInter interface{}
		switch keyType := key.Type; keyType {
		case ktyEC:
			keyInter, err = key.ECDSA()
			if err != nil {
				continue
			}
		case ktyOKP:
			keyInter, err = key.EdDSA()
			if err != nil {
				continue
			}
		case ktyOct:
			keyInter, err = key.Oct()
			if err != nil {
				continue
			}
		case ktyRSA:
			keyInter, err = key.RSA()
			if err != nil {
				continue
			}
		default:
			// Ignore unknown key types silently.
			continue
		}

		jwks.keys[key.ID] = parsedJWK{
			algorithm: key.Algorithm,
			use:       JWKUse(key.Use),
			public:    keyInter,
		}
	}

	return jwks, nil
}

// EndBackground ends the background goroutine to update the JWKS. It can only happen once and is only effective if the
// JWKS has a background goroutine refreshing the JWKS keys.
func (j *JWKS) EndBackground() {
	if j.cancel != nil {
		j.cancel()
	}
}

// KIDs returns the key IDs (`kid`) for all keys in the JWKS.
func (j *JWKS) KIDs() (kids []string) {
	j.mux.RLock()
	defer j.mux.RUnlock()
	kids = make([]string, len(j.keys))
	index := 0
	for kid := range j.keys {
		kids[index] = kid
		index++
	}
	return kids
}

// Len returns the number of keys in the JWKS.
func (j *JWKS) Len() int {
	j.mux.RLock()
	defer j.mux.RUnlock()
	return len(j.keys)
}

// RawJWKS returns a copy of the raw JWKS received from the given JWKS URL.
func (j *JWKS) RawJWKS() []byte {
	j.mux.RLock()
	defer j.mux.RUnlock()
	raw := make([]byte, len(j.raw))
	copy(raw, j.raw)
	return raw
}

// ReadOnlyKeys returns a read-only copy of the mapping of key IDs (`kid`) to cryptographic keys.
func (j *JWKS) ReadOnlyKeys() map[string]interface{} {
	keys := make(map[string]interface{})
	j.mux.Lock()
	for kid, cryptoKey := range j.keys {
		keys[kid] = cryptoKey.public
	}
	j.mux.Unlock()
	return keys
}

// getKey gets the jsonWebKey from the given KID from the JWKS. It may refresh the JWKS if configured to.
func (j *JWKS) getKey(alg, kid string) (jsonKey interface{}, err error) {
	j.mux.RLock()
	pubKey, ok := j.keys[kid]
	j.mux.RUnlock()

	if !ok {
		if !j.refreshUnknownKID {
			return nil, ErrKIDNotFound
		}

		ctx, cancel := context.WithCancel(j.ctx)
		req := refreshRequest{
			cancel: cancel,
		}

		// Refresh the JWKS.
		select {
		case <-j.ctx.Done():
			return
		case j.refreshRequests <- req:
		default:
			// If the j.refreshRequests channel is full, return the error early.
			return nil, ErrKIDNotFound
		}

		// Wait for the JWKS refresh to finish.
		<-ctx.Done()

		j.mux.RLock()
		defer j.mux.RUnlock()
		if pubKey, ok = j.keys[kid]; !ok {
			return nil, ErrKIDNotFound
		}
	}

	// jwkUseWhitelist might be empty if the jwks was from keyfunc.NewJSON() or if JWKUseNoWhitelist option was true.
	if len(j.jwkUseWhitelist) > 0 {
		_, ok = j.jwkUseWhitelist[pubKey.use]
		if !ok {
			return nil, fmt.Errorf(`%w: JWK "use" parameter value %q is not whitelisted`, ErrJWKUseWhitelist, pubKey.use)
		}
	}

	if pubKey.algorithm != "" && pubKey.algorithm != alg {
		return nil, fmt.Errorf(`%w: JWK "alg" parameter value %q does not match token "alg" parameter value %q`, ErrJWKAlgMismatch, pubKey.algorithm, alg)
	}

	return pubKey.public, nil
}
