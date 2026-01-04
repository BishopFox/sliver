package util

import (
	"crypto/rand"
	"encoding/binary"
	"io"
	"math"
)

// -----------------------
// Secure Random Utilities
// -----------------------

// Intn returns, like math/rand.Intn, a uniform int in [0, n).
// Panics if n <= 0 or if the OS CSPRNG fails.
func Intn(n int) int {
	if n <= 0 {
		panic("secure.Intn: non-positive n")
	}
	un := uint64(n)

	// Rejection sampling to remove modulo bias.
	// limit is the highest uint64 such that limit+1 is a multiple of n.
	limit := (math.MaxUint64 / un) * un

	for {
		x := mustRandUint64()
		if x < limit {
			return int(x % un)
		}
	}
}

// Shuffle does an in-place Fisher–Yates using secure.Intn.
// Same semantics as math/rand.Shuffle.
func Shuffle(n int, swap func(i, j int)) {
	if n < 0 {
		panic("secure.Shuffle: negative n")
	}
	for i := n - 1; i > 0; i-- {
		j := Intn(i + 1)
		if i != j {
			swap(i, j)
		}
	}
}

func mustRandUint64() uint64 {
	var b [8]byte
	if _, err := io.ReadFull(rand.Reader, b[:]); err != nil {
		panic("secure: crypto/rand failure: " + err.Error())
	}
	return binary.LittleEndian.Uint64(b[:])
}

// Int63n returns a uniform int64 in [0, n).
// Panics if n <= 0 or if crypto/rand fails.
func Int63n(n int64) int64 {
	if n <= 0 {
		panic("secure.Int63n: non-positive n")
	}
	un := uint64(n)

	const max63 = uint64(1<<63 - 1)
	// Largest acceptable value so that (limit+1) is a multiple of n.
	limit := max63 - (max63+1)%un

	for {
		x := randUint63()
		if x <= limit {
			return int64(x % un)
		}
	}
}

func randUint63() uint64 {
	var b [8]byte
	if _, err := io.ReadFull(rand.Reader, b[:]); err != nil {
		panic("secure: crypto/rand failure: " + err.Error())
	}
	// Uniform over [0, 2^63-1]
	return binary.LittleEndian.Uint64(b[:]) & (1<<63 - 1)
}

// Float64 returns a uniform float64 in [0.0, 1.0).
// Panics if crypto/rand fails.
func Float64() float64 {
	u := randUint53()
	const inv1p53 = 1.0 / (1 << 53) // 1 / 2^53
	return float64(u) * inv1p53
}

func randUint53() uint64 {
	var b [8]byte
	if _, err := io.ReadFull(rand.Reader, b[:]); err != nil {
		panic("secure: crypto/rand failure: " + err.Error())
	}
	// Take top 53 bits of a uniform 64-bit value. Uniform over [0, 2^53).
	return binary.LittleEndian.Uint64(b[:]) >> 11
}
