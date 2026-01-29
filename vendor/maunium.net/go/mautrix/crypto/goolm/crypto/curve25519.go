package crypto

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"

	"golang.org/x/crypto/curve25519"

	"maunium.net/go/mautrix/crypto/goolm/libolmpickle"
	"maunium.net/go/mautrix/id"
)

const (
	Curve25519PrivateKeyLength = curve25519.ScalarSize //The length of the private key.
	Curve25519PublicKeyLength  = 32
)

// Curve25519KeyPair stores both parts of a curve25519 key.
type Curve25519KeyPair struct {
	PrivateKey Curve25519PrivateKey `json:"private,omitempty"`
	PublicKey  Curve25519PublicKey  `json:"public,omitempty"`
}

// Curve25519GenerateKey creates a new curve25519 key pair.
func Curve25519GenerateKey() (Curve25519KeyPair, error) {
	privateKeyByte := make([]byte, Curve25519PrivateKeyLength)
	if _, err := rand.Read(privateKeyByte); err != nil {
		return Curve25519KeyPair{}, err
	}

	privateKey := Curve25519PrivateKey(privateKeyByte)
	publicKey, err := privateKey.PubKey()
	return Curve25519KeyPair{
		PrivateKey: Curve25519PrivateKey(privateKey),
		PublicKey:  Curve25519PublicKey(publicKey),
	}, err
}

// Curve25519GenerateFromPrivate creates a new curve25519 key pair with the private key given.
func Curve25519GenerateFromPrivate(private Curve25519PrivateKey) (Curve25519KeyPair, error) {
	publicKey, err := private.PubKey()
	return Curve25519KeyPair{
		PrivateKey: private,
		PublicKey:  Curve25519PublicKey(publicKey),
	}, err
}

// B64Encoded returns a base64 encoded string of the public key.
func (c Curve25519KeyPair) B64Encoded() id.Curve25519 {
	return c.PublicKey.B64Encoded()
}

// SharedSecret returns the shared secret between the key pair and the given public key.
func (c Curve25519KeyPair) SharedSecret(pubKey Curve25519PublicKey) ([]byte, error) {
	return c.PrivateKey.SharedSecret(pubKey)
}

// PickleLibOlm pickles the key pair into the encoder.
func (c Curve25519KeyPair) PickleLibOlm(encoder *libolmpickle.Encoder) {
	c.PublicKey.PickleLibOlm(encoder)
	if len(c.PrivateKey) == Curve25519PrivateKeyLength {
		encoder.Write(c.PrivateKey)
	} else {
		encoder.WriteEmptyBytes(Curve25519PrivateKeyLength)
	}
}

// UnpickleLibOlm decodes the unencryted value and populates the key pair accordingly. It returns the number of bytes read.
func (c *Curve25519KeyPair) UnpickleLibOlm(decoder *libolmpickle.Decoder) error {
	if err := c.PublicKey.UnpickleLibOlm(decoder); err != nil {
		return err
	} else if privKey, err := decoder.ReadBytes(Curve25519PrivateKeyLength); err != nil {
		return err
	} else {
		c.PrivateKey = privKey
		return nil
	}
}

// Curve25519PrivateKey represents the private key for curve25519 usage
type Curve25519PrivateKey []byte

// Equal compares the private key to the given private key.
func (c Curve25519PrivateKey) Equal(x Curve25519PrivateKey) bool {
	return subtle.ConstantTimeCompare(c, x) == 1
}

// PubKey returns the public key derived from the private key.
func (c Curve25519PrivateKey) PubKey() (Curve25519PublicKey, error) {
	return curve25519.X25519(c, curve25519.Basepoint)
}

// SharedSecret returns the shared secret between the private key and the given public key.
func (c Curve25519PrivateKey) SharedSecret(pubKey Curve25519PublicKey) ([]byte, error) {
	return curve25519.X25519(c, pubKey)
}

// Curve25519PublicKey represents the public key for curve25519 usage
type Curve25519PublicKey []byte

// Equal compares the public key to the given public key.
func (c Curve25519PublicKey) Equal(x Curve25519PublicKey) bool {
	return subtle.ConstantTimeCompare(c, x) == 1
}

// B64Encoded returns a base64 encoded string of the public key.
func (c Curve25519PublicKey) B64Encoded() id.Curve25519 {
	return id.Curve25519(base64.RawStdEncoding.EncodeToString(c))
}

// PickleLibOlm pickles the public key into the encoder.
func (c Curve25519PublicKey) PickleLibOlm(encoder *libolmpickle.Encoder) {
	if len(c) == Curve25519PublicKeyLength {
		encoder.Write(c)
	} else {
		encoder.WriteEmptyBytes(Curve25519PublicKeyLength)
	}
}

// UnpickleLibOlm decodes the unencryted value and populates the public key accordingly. It returns the number of bytes read.
func (c *Curve25519PublicKey) UnpickleLibOlm(decoder *libolmpickle.Decoder) error {
	pubkey, err := decoder.ReadBytes(Curve25519PublicKeyLength)
	*c = pubkey
	return err
}
