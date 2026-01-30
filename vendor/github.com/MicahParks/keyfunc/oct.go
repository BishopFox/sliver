package keyfunc

import (
	"fmt"
)

const (
	// ktyOct is the key type (kty) in the JWT header for oct.
	ktyOct = "oct"
)

// Oct parses a jsonWebKey and turns it into a raw byte slice (octet). This includes HMAC keys.
func (j *jsonWebKey) Oct() (publicKey []byte, err error) {
	if j.K == "" {
		return nil, fmt.Errorf("%w: %s", ErrMissingAssets, ktyOct)
	}

	// Decode the octet key from Base64.
	//
	// According to RFC 7517, this is Base64 URL bytes.
	// https://datatracker.ietf.org/doc/html/rfc7517#section-1.1
	publicKey, err = base64urlTrailingPadding(j.K)
	if err != nil {
		return nil, err
	}

	return publicKey, nil
}
