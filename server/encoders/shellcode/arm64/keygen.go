package arm64

import (
	"crypto/rand"
	"fmt"
)

const xorDynamicKeySize = 8

// XorKeyGen returns a random key suitable for the XOR encoder.
func XorKeyGen() ([]byte, error) {
	return randomBytes(xorKeySize)
}

// XorDynamicKeyGen returns a random key suitable for the XOR dynamic encoder.
func XorDynamicKeyGen() ([]byte, error) {
	return randomBytesFiltered(xorDynamicKeySize, xorDynamicBadchars)
}

func randomBytes(length int) ([]byte, error) {
	if length <= 0 {
		return nil, fmt.Errorf("invalid random length %d", length)
	}
	buf := make([]byte, length)
	if _, err := rand.Read(buf); err != nil {
		return nil, err
	}
	return buf, nil
}

func randomBytesFiltered(length int, badchars map[byte]bool) ([]byte, error) {
	if length <= 0 {
		return nil, fmt.Errorf("invalid random length %d", length)
	}
	buf := make([]byte, length)
	for i := range buf {
		for {
			if _, err := rand.Read(buf[i : i+1]); err != nil {
				return nil, err
			}
			if !badchars[buf[i]] {
				break
			}
		}
	}
	return buf, nil
}
