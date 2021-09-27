package cryptography

/*
	Sliver Implant Framework
	Copyright (C) 2020  Bishop Fox

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
	"crypto/rand"
	insecureRand "math/rand"
	"sync"
	"testing"
)

var (
	sample1 = randomData()
	sample2 = randomData()
)

func randomData() []byte {
	buf := make([]byte, insecureRand.Intn(256))
	rand.Read(buf)
	return buf
}

// TestEncryptDecrypt - Test AEAD functions
func TestEncryptDecrypt(t *testing.T) {
	key := RandomKey()
	cipher1, err := Encrypt(key, sample1)
	if err != nil {
		t.Fatal(err)
	}
	data1, err := Decrypt(key, cipher1)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(sample1, data1) {
		t.Fatalf("Sample does not match decrypted data")
	}

	key = RandomKey()
	cipher2, err := Encrypt(key, sample2)
	if err != nil {
		t.Fatal(err)
	}
	data2, err := Decrypt(key, cipher2)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(sample2, data2) {
		t.Fatalf("Sample does not match decrypted data")
	}
}

// TestTamperData - Detect tampered ciphertext
func TestTamperData(t *testing.T) {
	key := RandomKey()
	cipher1, err := Encrypt(key, sample1)
	if err != nil {
		t.Fatal(err)
	}

	index := insecureRand.Intn(len(cipher1))
	cipher1[index]++

	_, err = Decrypt(key, cipher1)
	if err == nil {
		t.Fatalf("Decrypted tampered data, should have resulted in Fatal")
	}
}

// TestWrongKey - Attempt to decrypt with wrong key
func TestWrongKey(t *testing.T) {
	key := RandomKey()
	cipher1, err := Encrypt(key, sample1)
	if err != nil {
		t.Fatal(err)
	}
	key2 := RandomKey()
	_, err = Decrypt(key2, cipher1)
	if err == nil {
		t.Fatalf("Decrypted with wrong key, should have resulted in Fatal")
	}
}

// TestCipherContext - Test CipherContext
func TestCipherContext(t *testing.T) {
	testKey := RandomKey()
	cipherCtx1 := &CipherContext{
		Key:    testKey,
		replay: &sync.Map{},
	}
	cipherCtx2 := &CipherContext{
		Key:    testKey,
		replay: &sync.Map{},
	}

	sample := randomData()

	ciphertext, err := cipherCtx1.Encrypt(sample)
	if err != nil {
		t.Fatalf("Failed to encrypt sample: %s", err)
	}
	_, err = cipherCtx1.Decrypt(ciphertext)
	if err != ErrReplayAttack {
		t.Fatal("Failed to detect replay attack")
	}
	_, err = cipherCtx1.Decrypt(ciphertext)
	if err != ErrReplayAttack {
		t.Fatal("Failed to detect replay attack")
	}

	plaintext, err := cipherCtx2.Decrypt(ciphertext)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(sample, plaintext) {
		t.Fatalf("Sample does not match decrypted data")
	}
	_, err = cipherCtx2.Decrypt(ciphertext)
	if err != ErrReplayAttack {
		t.Fatal("Failed to detect replay attack")
	}
}

func TestECCEncryptDecrypt(t *testing.T) {
	sample := randomData()
	sender, err := RandomECCKeyPair()
	if err != nil {
		t.Fatal(err)
	}
	receiver, err := RandomECCKeyPair()
	if err != nil {
		t.Fatal(err)
	}
	ciphertext, err := ECCEncrypt(receiver.Public, sender.Private, sample)
	if err != nil {
		t.Fatal(err)
	}
	plaintext, err := ECCDecrypt(sender.Public, receiver.Private, ciphertext)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(plaintext, sample) {
		t.Fatalf("Sample does not match decrypted data")
	}
}
