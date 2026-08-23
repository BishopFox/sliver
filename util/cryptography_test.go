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
	"bytes"
	"crypto/aes"
	"testing"
)

func TestIntnRange(t *testing.T) {
	for _, n := range []int{1, 2, 7, 100, 1024} {
		for i := 0; i < 2000; i++ {
			v := Intn(n)
			if v < 0 || v >= n {
				t.Fatalf("Intn(%d) returned %d, out of range [0, %d)", n, v, n)
			}
		}
	}
}

func TestIntnCoversRange(t *testing.T) {
	// Over enough samples every bucket in a small range should be hit.
	const n = 6
	seen := make([]bool, n)
	for i := 0; i < 10000; i++ {
		seen[Intn(n)] = true
	}
	for bucket, hit := range seen {
		if !hit {
			t.Fatalf("Intn(%d) never returned %d over 10000 samples", n, bucket)
		}
	}
}

func TestIntnSingleValue(t *testing.T) {
	for i := 0; i < 100; i++ {
		if v := Intn(1); v != 0 {
			t.Fatalf("Intn(1) = %d, want 0", v)
		}
	}
}

func TestIntnPanicsOnNonPositive(t *testing.T) {
	for _, n := range []int{0, -1, -100} {
		assertPanics(t, func() { Intn(n) })
	}
}

func TestInt63nRange(t *testing.T) {
	for _, n := range []int64{1, 3, 255, 1 << 40} {
		for i := 0; i < 2000; i++ {
			v := Int63n(n)
			if v < 0 || v >= n {
				t.Fatalf("Int63n(%d) returned %d, out of range [0, %d)", n, v, n)
			}
		}
	}
}

func TestInt63nPanicsOnNonPositive(t *testing.T) {
	for _, n := range []int64{0, -1, -100} {
		assertPanics(t, func() { Int63n(n) })
	}
}

func TestFloat64Range(t *testing.T) {
	for i := 0; i < 10000; i++ {
		v := Float64()
		if v < 0.0 || v >= 1.0 {
			t.Fatalf("Float64() returned %v, out of range [0.0, 1.0)", v)
		}
	}
}

func TestShuffleIsPermutation(t *testing.T) {
	const n = 500
	data := make([]int, n)
	for i := range data {
		data[i] = i
	}
	Shuffle(n, func(i, j int) { data[i], data[j] = data[j], data[i] })

	seen := make([]bool, n)
	for _, v := range data {
		if v < 0 || v >= n {
			t.Fatalf("shuffle produced an out-of-range value: %d", v)
		}
		if seen[v] {
			t.Fatalf("shuffle produced a duplicate value: %d", v)
		}
		seen[v] = true
	}
}

func TestShuffleZeroAndOneAreNoOps(t *testing.T) {
	for _, n := range []int{0, 1} {
		swaps := 0
		Shuffle(n, func(i, j int) { swaps++ })
		if swaps != 0 {
			t.Fatalf("Shuffle(%d) performed %d swaps, want 0", n, swaps)
		}
	}
}

func TestShufflePanicsOnNegative(t *testing.T) {
	assertPanics(t, func() { Shuffle(-1, func(i, j int) {}) })
}

func TestPreludeEncryptDecryptRoundTrip(t *testing.T) {
	key := []byte("0123456789abcdef0123456789abcdef") // 32 bytes -> AES-256
	iv := []byte("fedcba9876543210")                  // 16 bytes

	plaintexts := [][]byte{
		[]byte(""),
		[]byte("a"),
		[]byte("exactly-16-bytes"),
		[]byte("a longer message that spans multiple AES blocks for good measure"),
	}

	for _, plaintext := range plaintexts {
		// With an explicit IV.
		cipherText := PreludeEncrypt(plaintext, key, iv)
		if len(cipherText) < aes.BlockSize {
			t.Fatalf("ciphertext shorter than one block for input %q", plaintext)
		}
		decrypted := PreludeDecrypt(cipherText, key)
		if !bytes.Equal(decrypted, plaintext) {
			t.Fatalf("explicit-IV round-trip mismatch: got %q, want %q", decrypted, plaintext)
		}

		// With a randomly generated IV (nil IV argument).
		cipherText = PreludeEncrypt(plaintext, key, nil)
		decrypted = PreludeDecrypt(cipherText, key)
		if !bytes.Equal(decrypted, plaintext) {
			t.Fatalf("random-IV round-trip mismatch: got %q, want %q", decrypted, plaintext)
		}
	}
}

func TestPreludeEncryptRandomIVIsUnique(t *testing.T) {
	key := []byte("0123456789abcdef")
	plaintext := []byte("same plaintext every time")
	first := PreludeEncrypt(plaintext, key, nil)
	second := PreludeEncrypt(plaintext, key, nil)
	if bytes.Equal(first, second) {
		t.Fatalf("expected distinct ciphertexts from independently random IVs")
	}
}

func TestRC4EncryptUnsafeRoundTrip(t *testing.T) {
	key := []byte("super-secret-key")
	plaintext := []byte("shellcode obfuscation round-trip check")

	cipherText := RC4EncryptUnsafe(plaintext, key)
	if bytes.Equal(cipherText, plaintext) {
		t.Fatalf("RC4 output should differ from the plaintext")
	}

	// RC4 is symmetric: applying the same keystream again restores the input.
	restored := RC4EncryptUnsafe(cipherText, key)
	if !bytes.Equal(restored, plaintext) {
		t.Fatalf("RC4 round-trip mismatch: got %q, want %q", restored, plaintext)
	}
}

func TestRC4EncryptUnsafeEmptyKey(t *testing.T) {
	// An empty key is invalid for RC4; the helper returns an empty slice.
	if out := RC4EncryptUnsafe([]byte("data"), []byte{}); len(out) != 0 {
		t.Fatalf("expected empty output for an invalid (empty) key, got %d bytes", len(out))
	}
}

func assertPanics(t *testing.T, fn func()) {
	t.Helper()
	defer func() {
		if recover() == nil {
			t.Fatalf("expected the function to panic, but it did not")
		}
	}()
	fn()
}
