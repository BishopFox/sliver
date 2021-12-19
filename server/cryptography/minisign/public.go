package minisign

import (
	"crypto"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"io/ioutil"
	"strconv"
	"strings"
)

// PublicKeyFromFile reads a new PublicKey from the
// given file.
func PublicKeyFromFile(path string) (PublicKey, error) {
	bytes, err := ioutil.ReadFile(path)
	if err != nil {
		return PublicKey{}, err
	}

	var key PublicKey
	if err = key.UnmarshalText(bytes); err != nil {
		return PublicKey{}, err
	}
	return key, nil
}

// PublicKey is a minisign public key.
//
// A public key is used to verify whether messages
// have been signed with the corresponding private
// key.
type PublicKey struct {
	_ [0]func() // prevent direct comparison: p1 == p2.

	id    uint64
	bytes [ed25519.PublicKeySize]byte
}

// ID returns the 64 bit key ID.
func (p PublicKey) ID() uint64 { return p.id }

// Equal returns true if and only if p and x have equivalent values.
func (p PublicKey) Equal(x crypto.PublicKey) bool {
	xx, ok := x.(PublicKey)
	if !ok {
		return false
	}
	return p.id == xx.id && p.bytes == xx.bytes
}

// String returns a base64 string representation of the PublicKey p.
func (p PublicKey) String() string {
	var bytes [2 + 8 + ed25519.PublicKeySize]byte
	binary.LittleEndian.PutUint16(bytes[:2], EdDSA)
	binary.LittleEndian.PutUint64(bytes[2:10], p.ID())
	copy(bytes[10:], p.bytes[:])

	return base64.StdEncoding.EncodeToString(bytes[:])
}

// MarshalText returns a textual representation of the PublicKey p.
//
// It never returns an error.
func (p PublicKey) MarshalText() ([]byte, error) {
	var comment = "untrusted comment: minisign public key: " + strings.ToUpper(strconv.FormatUint(p.ID(), 16)) + "\n"
	return []byte(comment + p.String()), nil
}

// UnmarshalText parses text as textual-encoded public key.
// It returns an error if text is not a well-formed public key.
func (p *PublicKey) UnmarshalText(text []byte) error {
	text = trimUntrustedComment(text)
	bytes := make([]byte, base64.StdEncoding.DecodedLen(len(text)))
	n, err := base64.StdEncoding.Decode(bytes, text)
	if err != nil {
		return fmt.Errorf("minisign: invalid public key: %v", err)
	}
	bytes = bytes[:n] // Adjust b/c text may contain '\r' or '\n' which would have been ignored during decoding.

	if n = len(bytes); n != 2+8+ed25519.PublicKeySize {
		return errors.New("minisign: invalid public key length " + strconv.Itoa(n))
	}
	if a := binary.LittleEndian.Uint16(bytes[:2]); a != EdDSA {
		return errors.New("minisign: invalid public key algorithm " + strconv.Itoa(int(a)))
	}

	p.id = binary.LittleEndian.Uint64(bytes[2:10])
	copy(p.bytes[:], bytes[10:])
	return nil
}
