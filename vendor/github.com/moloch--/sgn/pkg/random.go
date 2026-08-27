package sgn

import (
	cryptorand "crypto/rand"
	"fmt"
	"math/big"
)

// secureRandomByte returns one byte from the operating system's
// cryptographically secure random source.
func secureRandomByte() (byte, error) {
	var value [1]byte
	if _, err := cryptorand.Read(value[:]); err != nil {
		return 0, fmt.Errorf("read cryptographic randomness: %w", err)
	}
	return value[0], nil
}

// secureIntn returns a cryptographically random integer in [0, n). The legacy
// helper API cannot return entropy errors, so failures are surfaced as panics
// instead of silently falling back to a predictable PRNG.
func secureIntn(n int) int {
	if n <= 0 {
		panic("secureIntn called with non-positive bound")
	}

	value, err := cryptorand.Int(cryptorand.Reader, big.NewInt(int64(n)))
	if err != nil {
		panic(fmt.Errorf("read cryptographic randomness: %w", err))
	}
	return int(value.Int64())
}
