// Package crypto defines additional interfaces that will be added to the
// crypto package in Go 1.26+.
package crypto

import (
	"crypto/ecdh"
	"crypto/mlkem"
)

// KeyExchanger is an interface for an opaque private key that can be used for
// key exchange operations. For example, an ECDH key kept in a hardware module.
//
// It is implemented by [ecdh.PrivateKey].
type KeyExchanger interface {
	PublicKey() *ecdh.PublicKey
	Curve() ecdh.Curve
	ECDH(*ecdh.PublicKey) ([]byte, error)
}

// Encapsulator is an interface for a public KEM key that can be used for
// encapsulation operations.
//
// It is implemented, for example, by [crypto/mlkem.EncapsulationKey768].
type Encapsulator interface {
	Bytes() []byte
	Encapsulate() (sharedKey, ciphertext []byte)
}

// Decapsulator is an interface for an opaque private KEM key that can be used for
// decapsulation operations. For example, an ML-KEM key kept in a hardware module.
//
// It will be implemented by [crypto/mlkem.DecapsulationKey768] in Go 1.26+.
// In the meantime, use [DecapsulatorFromDecapsulationKey768] and
// [DecapsulatorFromDecapsulationKey1024].
type Decapsulator interface {
	Encapsulator() Encapsulator
	Decapsulate(ciphertext []byte) (sharedKey []byte, err error)
}

// DecapsulatorFromDecapsulationKey768 wraps an ML-KEM-768 decapsulation key
// into a [Decapsulator], until Go 1.26+ where [crypto/mlkem.DecapsulationKey768]
// implements it natively.
func DecapsulatorFromDecapsulationKey768(dk *mlkem.DecapsulationKey768) Decapsulator {
	return &mlkem768Decapsulator{dk}
}

type mlkem768Decapsulator struct {
	*mlkem.DecapsulationKey768
}

func (d *mlkem768Decapsulator) Encapsulator() Encapsulator {
	return d.EncapsulationKey()
}

// DecapsulatorFromDecapsulationKey1024 wraps an ML-KEM-1024 decapsulation key
// into a [Decapsulator], until Go 1.26+ where [crypto/mlkem.DecapsulationKey1024]
// implements it natively.
func DecapsulatorFromDecapsulationKey1024(dk *mlkem.DecapsulationKey1024) Decapsulator {
	return &mlkem1024Decapsulator{dk}
}

type mlkem1024Decapsulator struct {
	*mlkem.DecapsulationKey1024
}

func (d *mlkem1024Decapsulator) Encapsulator() Encapsulator {
	return d.EncapsulationKey()
}
