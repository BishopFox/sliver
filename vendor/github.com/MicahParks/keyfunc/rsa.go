package keyfunc

import (
	"crypto/rsa"
	"fmt"
	"math/big"
)

const (
	// ktyRSA is the key type (kty) in the JWT header for RSA.
	ktyRSA = "RSA"
)

// RSA parses a jsonWebKey and turns it into an RSA public key.
func (j *jsonWebKey) RSA() (publicKey *rsa.PublicKey, err error) {
	if j.Exponent == "" || j.Modulus == "" {
		return nil, fmt.Errorf("%w: %s", ErrMissingAssets, ktyRSA)
	}

	// Decode the exponent from Base64.
	//
	// According to RFC 7518, this is a Base64 URL unsigned integer.
	// https://tools.ietf.org/html/rfc7518#section-6.3
	exponent, err := base64urlTrailingPadding(j.Exponent)
	if err != nil {
		return nil, err
	}
	modulus, err := base64urlTrailingPadding(j.Modulus)
	if err != nil {
		return nil, err
	}

	publicKey = &rsa.PublicKey{}

	// Turn the exponent into an integer.
	//
	// According to RFC 7517, these numbers are in big-endian format.
	// https://tools.ietf.org/html/rfc7517#appendix-A.1
	publicKey.E = int(big.NewInt(0).SetBytes(exponent).Uint64())
	publicKey.N = big.NewInt(0).SetBytes(modulus)

	return publicKey, nil
}
