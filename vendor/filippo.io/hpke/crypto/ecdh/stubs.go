package ecdh

import "crypto/ecdh"

// This file contains stubs to allow importing only this package instead of
// crypto/ecdh, to minimize the diff.

type Curve = ecdh.Curve
type PrivateKey = ecdh.PrivateKey
type PublicKey = ecdh.PublicKey

func X25519() Curve { return ecdh.X25519() }
func P256() Curve   { return ecdh.P256() }
func P384() Curve   { return ecdh.P384() }
func P521() Curve   { return ecdh.P521() }
