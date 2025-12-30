package util

/*
	Sliver Implant Framework
	Copyright (C) 2021  Bishop Fox

	This program is free software: you can redistribute it and/or modify
	it under the terms of the GNU General Public License as published by
	the Free Software Foundation, either version 3 of the License, or
	(at your option) any later version.

	This program is distributed in the hope that it will be useful,
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	GNU General Public License for more details.

	You should have received a copy of the GNU General Public License
	along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rc4"
	"encoding/binary"
	"errors"
	"io"
	"math"
)

// RC4 encryption - Cryptographically insecure!
// Added for stage-listener shellcode obfuscation
// Dont use for anything else!
func RC4EncryptUnsafe(data []byte, key []byte) []byte {
	cipher, err := rc4.NewCipher(key)
	if err != nil {
		return make([]byte, 0)
	}
	cipherText := make([]byte, len(data))
	cipher.XORKeyStream(cipherText, data)
	return cipherText
}

// PreludeEncrypt the results
func PreludeEncrypt(data []byte, key []byte, iv []byte) []byte {
	plainText, err := pad(data, aes.BlockSize)
	if err != nil {
		return make([]byte, 0)
	}
	block, _ := aes.NewCipher(key)
	cipherText := make([]byte, aes.BlockSize+len(plainText))
	// Create a random IV if none was provided
	// len(nil) returns 0
	if len(iv) == 0 {
		iv = cipherText[:aes.BlockSize]
		if _, err := io.ReadFull(rand.Reader, iv); err != nil {
			return make([]byte, 0)
		}
	} else {
		// make sure we copy the IV
		copy(cipherText[:aes.BlockSize], iv)
	}
	mode := cipher.NewCBCEncrypter(block, iv)
	mode.CryptBlocks(cipherText[aes.BlockSize:], plainText)
	return cipherText
}

// PreludeDecrypt a command
func PreludeDecrypt(data []byte, key []byte) []byte {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil
	}
	iv := data[:aes.BlockSize]
	data = data[aes.BlockSize:]
	mode := cipher.NewCBCDecrypter(block, iv)
	mode.CryptBlocks(data, data)
	data, _ = unpad(data, aes.BlockSize)
	return data
}

func pad(buf []byte, size int) ([]byte, error) {
	bufLen := len(buf)
	padLen := size - bufLen%size
	padded := make([]byte, bufLen+padLen)
	copy(padded, buf)
	for i := 0; i < padLen; i++ {
		padded[bufLen+i] = byte(padLen)
	}
	return padded, nil
}

func unpad(padded []byte, size int) ([]byte, error) {
	if len(padded)%size != 0 {
		return nil, errors.New("pkcs7: Padded value wasn't in correct size")
	}
	bufLen := len(padded) - int(padded[len(padded)-1])
	buf := make([]byte, bufLen)
	copy(buf, padded[:bufLen])
	return buf, nil
}

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
