package keyfunc

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"errors"
	"fmt"
	"math/big"
)

const (
	// ktyEC is the key type (kty) in the JWT header for ECDSA.
	ktyEC = "EC"

	// p256 represents a 256-bit cryptographic elliptical curve type.
	p256 = "P-256"

	// p384 represents a 384-bit cryptographic elliptical curve type.
	p384 = "P-384"

	// p521 represents a 521-bit cryptographic elliptical curve type.
	p521 = "P-521"
)

var (
	// ErrECDSACurve indicates an error with the ECDSA curve.
	ErrECDSACurve = errors.New("invalid ECDSA curve")
)

// ECDSA parses a jsonWebKey and turns it into an ECDSA public key.
func (j *jsonWebKey) ECDSA() (publicKey *ecdsa.PublicKey, err error) {
	if j.X == "" || j.Y == "" || j.Curve == "" {
		return nil, fmt.Errorf("%w: %s", ErrMissingAssets, ktyEC)
	}

	// Decode the X coordinate from Base64.
	//
	// According to RFC 7518, this is a Base64 URL unsigned integer.
	// https://tools.ietf.org/html/rfc7518#section-6.3
	xCoordinate, err := base64urlTrailingPadding(j.X)
	if err != nil {
		return nil, err
	}
	yCoordinate, err := base64urlTrailingPadding(j.Y)
	if err != nil {
		return nil, err
	}

	publicKey = &ecdsa.PublicKey{}
	switch j.Curve {
	case p256:
		publicKey.Curve = elliptic.P256()
	case p384:
		publicKey.Curve = elliptic.P384()
	case p521:
		publicKey.Curve = elliptic.P521()
	default:
		return nil, fmt.Errorf("%w: unknown curve: %s", ErrECDSACurve, j.Curve)
	}

	// Turn the X coordinate into *big.Int.
	//
	// According to RFC 7517, these numbers are in big-endian format.
	// https://tools.ietf.org/html/rfc7517#appendix-A.1
	publicKey.X = big.NewInt(0).SetBytes(xCoordinate)
	publicKey.Y = big.NewInt(0).SetBytes(yCoordinate)

	return publicKey, nil
}
