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
	"crypto/rsa"
	insecureRand "math/rand"
	"sync"
	"testing"
)

var (
	sample1 = randomData()
	sample2 = randomData()
)

func randomData() []byte {
	buf := make([]byte, insecureRand.Intn(8*1024))
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

// TestAESEncryptDecrypt - Test AES functions
func TestAESEncryptDecrypt(t *testing.T) {
	key := RandomAESKey()
	cipher1, err := AESEncrypt(key, sample1)
	if err != nil {
		t.Fatal(err)
	}
	data1, err := AESDecrypt(key, cipher1)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(sample1, data1) {
		t.Fatalf("Sample does not match decrypted data")
	}

	key = RandomAESKey()
	cipher2, err := AESEncrypt(key, sample2)
	if err != nil {
		t.Fatal(err)
	}
	data2, err := AESDecrypt(key, cipher2)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(sample2, data2) {
		t.Fatalf("Sample does not match decrypted data")
	}
}

// TestAESTamperData - Detect tampered ciphertext
func TestAESTamperData(t *testing.T) {
	key := RandomAESKey()
	cipher1, err := AESEncrypt(key, sample1)
	if err != nil {
		t.Fatal(err)
	}

	index := insecureRand.Intn(len(cipher1))
	cipher1[index]++

	_, err = AESDecrypt(key, cipher1)
	if err == nil {
		t.Fatalf("Decrypted tampered data, should have resulted in Fatal")
	}
}

// TestAESWrongKey - Attempt to decrypt with wrong key
func TestAESWrongKey(t *testing.T) {
	key := RandomAESKey()
	cipher1, err := AESEncrypt(key, sample1)
	if err != nil {
		t.Fatal(err)
	}

	key2 := RandomAESKey()
	_, err = AESDecrypt(key2, cipher1)
	if err == nil {
		t.Fatalf("Decrypted with wrong key, should have resulted in Fatal")
	}
}

// TestRSAEncryptDecrypt - Test RSA functions
func TestRSAEncryptDecrypt(t *testing.T) {

	rsaKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	cipher1, err := RSAEncrypt(sample1, &rsaKey.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	data1, err := RSADecrypt(cipher1, rsaKey)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(sample1, data1) {
		t.Fatalf("Sample does not match decrypted data")
	}

	rsaKey, err = rsa.GenerateKey(rand.Reader, 2048)
	cipher2, err := RSAEncrypt(sample2, &rsaKey.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	data2, err := RSADecrypt(cipher2, rsaKey)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(sample2, data2) {
		t.Fatalf("Sample does not match decrypted data")
	}
}

// TestRSATamperData - Test RSA with tampered data
func TestRSATamperData(t *testing.T) {
	rsaKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	cipher1, err := RSAEncrypt(sample1, &rsaKey.PublicKey)
	if err != nil {
		t.Fatal(err)
	}

	index := insecureRand.Intn(len(cipher1))
	cipher1[index]++

	_, err = RSADecrypt(cipher1, rsaKey)
	if err == nil {
		t.Fatalf("Decrypted tampered data, should have resulted in Fatal")
	}
}

// TestRSAWrongKey - Test RSA with wrong key
func TestRSAWrongKey(t *testing.T) {
	rsaKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	cipher1, err := RSAEncrypt(sample1, &rsaKey.PublicKey)
	if err != nil {
		t.Fatal(err)
	}

	rsaKey2, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	_, err = RSADecrypt(cipher1, rsaKey2)
	if err == nil {
		t.Fatalf("Decrypted with wrong key, should have resulted in Fatal")
	}
}

// TestCipherContext - Test CipherContext
func TestCipherContext(t *testing.T) {
	testKey := RandomAESKey()
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
	if err == nil {
		t.Fatalf("Failed to encrypt sample")
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
