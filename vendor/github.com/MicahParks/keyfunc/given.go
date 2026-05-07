package keyfunc

import (
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/rsa"
	"encoding/json"
)

// GivenKey represents a cryptographic key that resides in a JWKS. In conjuncture with Options.
type GivenKey struct {
	algorithm string
	inter     interface{}
}

// GivenKeyOptions represents the configuration options for a GivenKey.
type GivenKeyOptions struct {
	// Algorithm is the given key's signing algorithm. Its value will be compared to unverified tokens' "alg" header.
	//
	// See RFC 8725 Section 3.1 for details.
	// https://www.rfc-editor.org/rfc/rfc8725#section-3.1
	//
	// For a list of possible values, please see:
	// https://www.rfc-editor.org/rfc/rfc7518#section-3.1
	// https://www.iana.org/assignments/jose/jose.xhtml#web-signature-encryption-algorithms
	Algorithm string
}

// NewGiven creates a JWKS from a map of given keys.
func NewGiven(givenKeys map[string]GivenKey) (jwks *JWKS) {
	keys := make(map[string]parsedJWK)

	for kid, given := range givenKeys {
		keys[kid] = parsedJWK{
			algorithm: given.algorithm,
			public:    given.inter,
		}
	}

	return &JWKS{
		keys: keys,
	}
}

// NewGivenCustom creates a new GivenKey given an untyped variable. The key argument is expected to be a supported
// by the jwt package used.
//
// See the https://pkg.go.dev/github.com/golang-jwt/jwt/v4#RegisterSigningMethod function for registering an unsupported
// signing method.
//
// Deprecated: This function does not allow the user to specify the JWT's signing algorithm. Use
// NewGivenCustomWithOptions instead.
func NewGivenCustom(key interface{}) (givenKey GivenKey) {
	return GivenKey{
		inter: key,
	}
}

// NewGivenCustomWithOptions creates a new GivenKey given an untyped variable. The key argument is expected to be a type
// supported by the jwt package used.
//
// Consider the options carefully as each field may have a security implication.
//
// See the https://pkg.go.dev/github.com/golang-jwt/jwt/v4#RegisterSigningMethod function for registering an unsupported
// signing method.
func NewGivenCustomWithOptions(key interface{}, options GivenKeyOptions) (givenKey GivenKey) {
	return GivenKey{
		algorithm: options.Algorithm,
		inter:     key,
	}
}

// NewGivenECDSA creates a new GivenKey given an ECDSA public key.
//
// Deprecated: This function does not allow the user to specify the JWT's signing algorithm. Use
// NewGivenECDSACustomWithOptions instead.
func NewGivenECDSA(key *ecdsa.PublicKey) (givenKey GivenKey) {
	return GivenKey{
		inter: key,
	}
}

// NewGivenECDSACustomWithOptions creates a new GivenKey given an ECDSA public key.
//
// Consider the options carefully as each field may have a security implication.
func NewGivenECDSACustomWithOptions(key *ecdsa.PublicKey, options GivenKeyOptions) (givenKey GivenKey) {
	return GivenKey{
		algorithm: options.Algorithm,
		inter:     key,
	}
}

// NewGivenEdDSA creates a new GivenKey given an EdDSA public key.
//
// Deprecated: This function does not allow the user to specify the JWT's signing algorithm. Use
// NewGivenEdDSACustomWithOptions instead.
func NewGivenEdDSA(key ed25519.PublicKey) (givenKey GivenKey) {
	return GivenKey{
		inter: key,
	}
}

// NewGivenEdDSACustomWithOptions creates a new GivenKey given an EdDSA public key.
//
// Consider the options carefully as each field may have a security implication.
func NewGivenEdDSACustomWithOptions(key ed25519.PublicKey, options GivenKeyOptions) (givenKey GivenKey) {
	return GivenKey{
		algorithm: options.Algorithm,
		inter:     key,
	}
}

// NewGivenHMAC creates a new GivenKey given an HMAC key in a byte slice.
//
// Deprecated: This function does not allow the user to specify the JWT's signing algorithm. Use
// NewGivenHMACCustomWithOptions instead.
func NewGivenHMAC(key []byte) (givenKey GivenKey) {
	return GivenKey{
		inter: key,
	}
}

// NewGivenHMACCustomWithOptions creates a new GivenKey given an HMAC key in a byte slice.
//
// Consider the options carefully as each field may have a security implication.
func NewGivenHMACCustomWithOptions(key []byte, options GivenKeyOptions) (givenKey GivenKey) {
	return GivenKey{
		algorithm: options.Algorithm,
		inter:     key,
	}
}

// NewGivenRSA creates a new GivenKey given an RSA public key.
//
// Deprecated: This function does not allow the user to specify the JWT's signing algorithm. Use
// NewGivenRSACustomWithOptions instead.
func NewGivenRSA(key *rsa.PublicKey) (givenKey GivenKey) {
	return GivenKey{
		inter: key,
	}
}

// NewGivenRSACustomWithOptions creates a new GivenKey given an RSA public key.
//
// Consider the options carefully as each field may have a security implication.
func NewGivenRSACustomWithOptions(key *rsa.PublicKey, options GivenKeyOptions) (givenKey GivenKey) {
	return GivenKey{
		algorithm: options.Algorithm,
		inter:     key,
	}
}

// NewGivenKeysFromJSON parses a raw JSON message into a map of key IDs (`kid`) to GivenKeys. The returned map is
// suitable for passing to `NewGiven()` or as `Options.GivenKeys` to `Get()`
func NewGivenKeysFromJSON(jwksBytes json.RawMessage) (map[string]GivenKey, error) {
	// Parse by making a temporary JWKS instance. No need to lock its map since it doesn't escape this function.
	j, err := NewJSON(jwksBytes)
	if err != nil {
		return nil, err
	}
	keys := make(map[string]GivenKey, len(j.keys))
	for kid, cryptoKey := range j.keys {
		keys[kid] = GivenKey{
			algorithm: cryptoKey.algorithm,
			inter:     cryptoKey.public,
		}
	}
	return keys, nil
}
