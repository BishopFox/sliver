// Package ecdh defines an additional interface that will be added to the
// crypto/ecdh package in Go 1.26+.
package ecdh

import "crypto/ecdh"

// KeyExchanger is an interface for an opaque private key that can be used for
// key exchange operations. For example, an ECDH key kept in a hardware module.
//
// It is implemented by [ecdh.PrivateKey].
type KeyExchanger interface {
	PublicKey() *ecdh.PublicKey
	Curve() ecdh.Curve
	ECDH(*ecdh.PublicKey) ([]byte, error)
}
