package keyfunc

import (
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"github.com/golang-jwt/jwt/v4"
)

var (
	// ErrKID indicates that the JWT had an invalid kid.
	ErrKID = errors.New("the JWT has an invalid kid")
)

// Keyfunc matches the signature of github.com/golang-jwt/jwt/v4's jwt.Keyfunc function.
func (j *JWKS) Keyfunc(token *jwt.Token) (interface{}, error) {
	kid, alg, err := kidAlg(token)
	if err != nil {
		return nil, err
	}
	return j.getKey(alg, kid)
}

func (m *MultipleJWKS) Keyfunc(token *jwt.Token) (interface{}, error) {
	return m.keySelector(m, token)
}

func kidAlg(token *jwt.Token) (kid, alg string, err error) {
	kidInter, ok := token.Header["kid"]
	if !ok {
		return "", "", fmt.Errorf("%w: could not find kid in JWT header", ErrKID)
	}
	kid, ok = kidInter.(string)
	if !ok {
		return "", "", fmt.Errorf("%w: could not convert kid in JWT header to string", ErrKID)
	}
	alg, ok = token.Header["alg"].(string)
	if !ok {
		// For test coverage purposes, this should be impossible to reach because the JWT package rejects a token
		// without an alg parameter in the header before calling jwt.Keyfunc.
		return "", "", fmt.Errorf(`%w: the JWT header did not contain the "alg" parameter, which is required by RFC 7515 section 4.1.1`, ErrJWKAlgMismatch)
	}
	return kid, alg, nil
}

// base64urlTrailingPadding removes trailing padding before decoding a string from base64url. Some non-RFC compliant
// JWKS contain padding at the end values for base64url encoded public keys.
//
// Trailing padding is required to be removed from base64url encoded keys.
// RFC 7517 defines base64url the same as RFC 7515 Section 2:
// https://datatracker.ietf.org/doc/html/rfc7517#section-1.1
// https://datatracker.ietf.org/doc/html/rfc7515#section-2
func base64urlTrailingPadding(s string) ([]byte, error) {
	s = strings.TrimRight(s, "=")
	return base64.RawURLEncoding.DecodeString(s)
}
