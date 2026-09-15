package goolmbase64

import (
	"encoding/base64"
)

// These methods should only be used for raw byte operations, never with string conversion

func Decode(input []byte) ([]byte, error) {
	decoded := make([]byte, base64.RawStdEncoding.DecodedLen(len(input)))
	writtenBytes, err := base64.RawStdEncoding.Decode(decoded, input)
	if err != nil {
		return nil, err
	}
	return decoded[:writtenBytes], nil
}

func Encode(input []byte) []byte {
	encoded := make([]byte, base64.RawStdEncoding.EncodedLen(len(input)))
	base64.RawStdEncoding.Encode(encoded, input)
	return encoded
}
