package encoders

import (
	"encoding/base64"
	"encoding/hex"
)

// ASCIIEncoder - Can losslessly encode arbitrary binary data
type ASCIIEncoder interface {
	Encode([]byte) string
	Decode(string) ([]byte, error)
}

// Hex Encoder
type Hex struct{}

// Encode - Hex Encode
func (e Hex) Encode(data []byte) string {
	return hex.EncodeToString(data)
}

// Decode - Hex Decode
func (e Hex) Decode(data string) ([]byte, error) {
	return hex.DecodeString(data)
}

// Base64 Encoder
type Base64 struct{}

var base64Alphabet = "a0b2c5def6hijklmnopqr+st/uvwxyzA1B3C4DEFGHIJKLM7NO9PQR8ST0UVWXYZ"
var sliverBase64 = base64.NewEncoding(base64Alphabet)

// Encode - Base64 Encode
func (e Base64) Encode(data []byte) string {
	return sliverBase64.EncodeToString(data)
}

// Decode - Base64 Decode
func (e Base64) Decode(data string) ([]byte, error) {
	return sliverBase64.DecodeString(data)
}
