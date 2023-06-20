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
	"os"
	"sync"
	"testing"

	implantCrypto "github.com/bishopfox/sliver/implant/sliver/cryptography"
	"github.com/bishopfox/sliver/server/cryptography/minisign"
)

var (
	sample1 = randomData()
	sample2 = randomData()

	serverECCKeyPair  *AgeKeyPair
	implantECCKeyPair *AgeKeyPair
)

func randomData() []byte {
	buf := make([]byte, insecureRand.Intn(256))
	rand.Read(buf)
	return buf
}

func TestMain(m *testing.M) {
	setup()
	os.Exit(m.Run())
}

func setup() {
	var err error
	serverECCKeyPair, err = RandomAgeKeyPair()
	if err != nil {
		panic(err)
	}
	implantECCKeyPair, err = RandomAgeKeyPair()
	if err != nil {
		panic(err)
	}
	totpSecret, err := TOTPServerSecret()
	if err != nil {
		panic(err)
	}

	implantCrypto.SetSecrets(
		implantECCKeyPair.Public,
		implantECCKeyPair.Private,
		MinisignServerSign([]byte(implantECCKeyPair.Public)),
		serverECCKeyPair.Public,
		totpSecret,
		MinisignServerPublicKey(),
	)
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

	receiver, err := RandomAgeKeyPair()
	if err != nil {
		t.Fatal(err)
	}
	ciphertext, err := AgeEncrypt(receiver.Public, sample)
	if err != nil {
		t.Fatal(err)
	}
	plaintext, err := AgeDecrypt(receiver.Private, ciphertext)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(plaintext, sample) {
		t.Fatalf("Sample does not match decrypted data")
	}
}

// TestEncryptDecrypt - Test AEAD functions
func TestImplantEncryptDecrypt(t *testing.T) {
	key := RandomKey()
	cipher1, err := Encrypt(key, sample1)
	if err != nil {
		t.Fatal(err)
	}
	data1, err := implantCrypto.Decrypt(key, cipher1)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(sample1, data1) {
		t.Fatalf("Sample does not match decrypted data")
	}

	key = RandomKey()
	cipher2, err := implantCrypto.Encrypt(key, sample2)
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

func TestImplantECCEncryptDecrypt(t *testing.T) {
	sample := randomData()
	ciphertext, err := implantCrypto.AgeKeyExToServer(sample)
	if err != nil {
		t.Fatalf("encrypt to server failed: %s", err)
	}
	if len(ciphertext) < 33 {
		t.Fatalf("ciphertext too short (%d)", len(ciphertext))
	}

	// Ciphertext has sender public key digest prepended [:32]
	plaintext, err := AgeKeyExFromImplant(serverECCKeyPair.Private, implantECCKeyPair.Private, ciphertext[32:])
	if err != nil {
		t.Fatalf("failed to decrypt implant ciphertext: %s", err)
	}
	if !bytes.Equal(plaintext, sample) {
		t.Fatalf("Sample does not match decrypted data")
	}
}

func TestImplantECCEncryptDecryptTamperData(t *testing.T) {
	sample := randomData()
	ciphertext, err := implantCrypto.AgeKeyExToServer(sample)
	if err != nil {
		t.Fatal(err)
	}
	if len(ciphertext) < 34 {
		t.Fatal("ciphertext too short")
	}
	ciphertext[33]++ // Change a byte in the ciphertext
	_, err = AgeDecrypt(serverECCKeyPair.Private, ciphertext[32:])
	if err == nil {
		t.Fatal("ecc decrypted tampered data without error")
	}
}

func TestServerMinisign(t *testing.T) {
	message := randomData()
	privateKey := MinisignServerPrivateKey()
	signature := minisign.Sign(*privateKey, message)
	if !minisign.Verify(privateKey.Public().(minisign.PublicKey), message, signature) {
		t.Fatalf("Failed to very message with server minisign")
	}
	message[0]++
	if minisign.Verify(privateKey.Public().(minisign.PublicKey), message, signature) {
		t.Fatalf("Minisign verified tampered message")
	}
}

func TestImplantMinisign(t *testing.T) {
	message := randomData()
	privateKey := MinisignServerPrivateKey()
	signature := minisign.Sign(*privateKey, message)

	publicKey := privateKey.Public().(minisign.PublicKey)
	publicKeyTxt, err := publicKey.MarshalText()
	if err != nil {
		t.Fatal(err)
	}

	implantPublicKey, err := implantCrypto.DecodeMinisignPublicKey(string(publicKeyTxt))
	if err != nil {
		t.Fatal(err)
	}
	implantSig, err := implantCrypto.DecodeMinisignSignature(string(signature))
	if err != nil {
		t.Fatal(err)
	}
	valid, err := implantPublicKey.Verify(message, implantSig)
	if err != nil {
		t.Fatal(err)
	}

	if !valid {
		t.Fatal("Implant failed to verify minisign signature")
	}
	message[0]++
	valid, err = implantPublicKey.Verify(message, implantSig)
	if err == nil {
		t.Fatal("Expected invalid signature error")
	}
	if valid {
		t.Fatal("Implant verified tampered message")
	}

}
