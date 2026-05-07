package oauth1

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
)

// Noncer provides random nonce strings.
type Noncer interface {
	Nonce() string
}

// Base64Noncer reads 32 bytes from crypto/rand and
// returns those bytes as a base64 encoded string.
type Base64Noncer struct{}

// Nonce provides a random nonce string.
func (n Base64Noncer) Nonce() string {
	b := make([]byte, 32)
	rand.Read(b)
	return base64.StdEncoding.EncodeToString(b)
}

// HexNoncer reads 32 bytes from crypto/rand and
// returns those bytes as a base64 encoded string.
type HexNoncer struct{}

// Nonce provides a random nonce string.
func (n HexNoncer) Nonce() string {
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b)
}
