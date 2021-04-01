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

// TestAESEncryptDecrypt - Test AES functions
func TestAESEncryptDecrypt(t *testing.T) {
	key := RandomAESKey()
	cipher1, err := GCMEncrypt(key, sample1)
	if err != nil {
		t.Error(err)
	}
	data1, err := GCMDecrypt(key, cipher1)
	if err != nil {
		t.Error(err)
	}
	if !bytes.Equal(sample1, data1) {
		t.Errorf("Sample does not match decrypted data")
	}

	key = RandomAESKey()
	cipher2, err := GCMEncrypt(key, sample2)
	if err != nil {
		t.Error(err)
	}
	data2, err := GCMDecrypt(key, cipher2)
	if err != nil {
		t.Error(err)
	}
	if !bytes.Equal(sample2, data2) {
		t.Errorf("Sample does not match decrypted data")
	}
}

// TestAESTamperData - Detect tampered ciphertext
func TestAESTamperData(t *testing.T) {
	key := RandomAESKey()
	cipher1, err := GCMEncrypt(key, sample1)
	if err != nil {
		t.Error(err)
	}

	index := insecureRand.Intn(len(cipher1))
	cipher1[index]++

	_, err = GCMDecrypt(key, cipher1)
	if err == nil {
		t.Errorf("Decrypted tampered data, should have resulted in error")
	}
}

// TestAESWrongKey - Attempt to decrypt with wrong key
func TestAESWrongKey(t *testing.T) {
	key := RandomAESKey()
	cipher1, err := GCMEncrypt(key, sample1)
	if err != nil {
		t.Error(err)
	}

	key2 := RandomAESKey()
	_, err = GCMDecrypt(key2, cipher1)
	if err == nil {
		t.Errorf("Decrypted with wrong key, should have resulted in error")
	}
}

// TestRSAEncryptDecrypt - Test RSA functions
func TestRSAEncryptDecrypt(t *testing.T) {

	rsaKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Error(err)
	}
	cipher1, err := RSAEncrypt(sample1, &rsaKey.PublicKey)
	if err != nil {
		t.Error(err)
	}
	data1, err := RSADecrypt(cipher1, rsaKey)
	if err != nil {
		t.Error(err)
	}
	if !bytes.Equal(sample1, data1) {
		t.Errorf("Sample does not match decrypted data")
	}

	rsaKey, err = rsa.GenerateKey(rand.Reader, 2048)
	cipher2, err := RSAEncrypt(sample2, &rsaKey.PublicKey)
	if err != nil {
		t.Error(err)
	}
	data2, err := RSADecrypt(cipher2, rsaKey)
	if err != nil {
		t.Error(err)
	}
	if !bytes.Equal(sample2, data2) {
		t.Errorf("Sample does not match decrypted data")
	}
}

// TestRSATamperData - Test RSA with tampered data
func TestRSATamperData(t *testing.T) {
	rsaKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Error(err)
	}
	cipher1, err := RSAEncrypt(sample1, &rsaKey.PublicKey)
	if err != nil {
		t.Error(err)
	}

	index := insecureRand.Intn(len(cipher1))
	cipher1[index]++

	_, err = RSADecrypt(cipher1, rsaKey)
	if err == nil {
		t.Errorf("Decrypted tampered data, should have resulted in error")
	}
}

// TestRSAWrongKey - Test RSA with wrong key
func TestRSAWrongKey(t *testing.T) {
	rsaKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Error(err)
	}
	cipher1, err := RSAEncrypt(sample1, &rsaKey.PublicKey)
	if err != nil {
		t.Error(err)
	}

	rsaKey2, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Error(err)
	}
	_, err = RSADecrypt(cipher1, rsaKey2)
	if err == nil {
		t.Errorf("Decrypted with wrong key, should have resulted in error")
	}
}
