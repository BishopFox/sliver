package crypto

import (
	"encoding/base64"

	"maunium.net/go/mautrix/crypto/ed25519"
	"maunium.net/go/mautrix/crypto/goolm/libolmpickle"
	"maunium.net/go/mautrix/id"
)

const (
	Ed25519SignatureSize = ed25519.SignatureSize //The length of a signature
)

// Ed25519GenerateKey creates a new ed25519 key pair.
func Ed25519GenerateKey() (Ed25519KeyPair, error) {
	publicKey, privateKey, err := ed25519.GenerateKey(nil)
	return Ed25519KeyPair{
		PrivateKey: Ed25519PrivateKey(privateKey),
		PublicKey:  Ed25519PublicKey(publicKey),
	}, err
}

// Ed25519GenerateFromPrivate creates a new ed25519 key pair with the private key given.
func Ed25519GenerateFromPrivate(privKey Ed25519PrivateKey) Ed25519KeyPair {
	return Ed25519KeyPair{
		PrivateKey: privKey,
		PublicKey:  privKey.PubKey(),
	}
}

// Ed25519GenerateFromSeed creates a new ed25519 key pair with a given seed.
func Ed25519GenerateFromSeed(seed []byte) Ed25519KeyPair {
	privKey := Ed25519PrivateKey(ed25519.NewKeyFromSeed(seed))
	return Ed25519KeyPair{
		PrivateKey: privKey,
		PublicKey:  privKey.PubKey(),
	}
}

// Ed25519KeyPair stores both parts of a ed25519 key.
type Ed25519KeyPair struct {
	PrivateKey Ed25519PrivateKey `json:"private,omitempty"`
	PublicKey  Ed25519PublicKey  `json:"public,omitempty"`
}

// B64Encoded returns a base64 encoded string of the public key.
func (c Ed25519KeyPair) B64Encoded() id.Ed25519 {
	return id.Ed25519(base64.RawStdEncoding.EncodeToString(c.PublicKey))
}

// Sign returns the signature for the message.
func (c Ed25519KeyPair) Sign(message []byte) ([]byte, error) {
	return c.PrivateKey.Sign(message)
}

// Verify checks the signature of the message against the givenSignature
func (c Ed25519KeyPair) Verify(message, givenSignature []byte) bool {
	return c.PublicKey.Verify(message, givenSignature)
}

// PickleLibOlm pickles the key pair into the encoder.
func (c Ed25519KeyPair) PickleLibOlm(encoder *libolmpickle.Encoder) {
	c.PublicKey.PickleLibOlm(encoder)
	if len(c.PrivateKey) == ed25519.PrivateKeySize {
		encoder.Write(c.PrivateKey)
	} else {
		encoder.WriteEmptyBytes(ed25519.PrivateKeySize)
	}
}

// UnpickleLibOlm unpickles the unencryted value and populates the key pair accordingly.
func (c *Ed25519KeyPair) UnpickleLibOlm(decoder *libolmpickle.Decoder) error {
	if err := c.PublicKey.UnpickleLibOlm(decoder); err != nil {
		return err
	} else if privKey, err := decoder.ReadBytes(ed25519.PrivateKeySize); err != nil {
		return err
	} else {
		c.PrivateKey = privKey
		return nil
	}
}

// Curve25519PrivateKey represents the private key for ed25519 usage. This is just a wrapper.
type Ed25519PrivateKey ed25519.PrivateKey

// Equal compares the private key to the given private key.
func (c Ed25519PrivateKey) Equal(x Ed25519PrivateKey) bool {
	return ed25519.PrivateKey(c).Equal(ed25519.PrivateKey(x))
}

// PubKey returns the public key derived from the private key.
func (c Ed25519PrivateKey) PubKey() Ed25519PublicKey {
	publicKey := ed25519.PrivateKey(c).Public()
	return Ed25519PublicKey(publicKey.([]byte))
}

// Sign returns the signature for the message.
func (c Ed25519PrivateKey) Sign(message []byte) ([]byte, error) {
	return ed25519.PrivateKey(c).Sign(nil, message, &ed25519.Options{})
}

// Ed25519PublicKey represents the public key for ed25519 usage. This is just a wrapper.
type Ed25519PublicKey ed25519.PublicKey

// Equal compares the public key to the given public key.
func (c Ed25519PublicKey) Equal(x Ed25519PublicKey) bool {
	return ed25519.PublicKey(c).Equal(ed25519.PublicKey(x))
}

// B64Encoded returns a base64 encoded string of the public key.
func (c Ed25519PublicKey) B64Encoded() id.Curve25519 {
	return id.Curve25519(base64.RawStdEncoding.EncodeToString(c))
}

// Verify checks the signature of the message against the givenSignature
func (c Ed25519PublicKey) Verify(message, givenSignature []byte) bool {
	return ed25519.Verify(ed25519.PublicKey(c), message, givenSignature)
}

// PickleLibOlm pickles the public key into the encoder.
func (c Ed25519PublicKey) PickleLibOlm(encoder *libolmpickle.Encoder) {
	if len(c) == ed25519.PublicKeySize {
		encoder.Write(c)
	} else {
		encoder.WriteEmptyBytes(ed25519.PublicKeySize)
	}
}

// UnpickleLibOlm unpickles the unencryted value and populates the public key
// accordingly.
func (c *Ed25519PublicKey) UnpickleLibOlm(decoder *libolmpickle.Decoder) error {
	key, err := decoder.ReadBytes(ed25519.PublicKeySize)
	*c = key
	return err
}
