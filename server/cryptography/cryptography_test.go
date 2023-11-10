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
	"github.com/bishopfox/sliver/util/minisign"
)

var (
	sample1 = randomData()
	sample2 = randomData()

	serverAgeKeyPair      *AgeKeyPair
	implantPeerAgeKeyPair *AgeKeyPair
)

func randomData() []byte {
	buf := make([]byte, insecureRand.Intn(256)+10)
	rand.Read(buf)
	return buf
}

func TestMain(m *testing.M) {
	setup()
	os.Exit(m.Run())
}

func setup() {
	var err error
	serverAgeKeyPair, err = RandomAgeKeyPair()
	if err != nil {
		panic(err)
	}
	implantPeerAgeKeyPair, err = RandomAgeKeyPair()
	if err != nil {
		panic(err)
	}
	implantCrypto.SetSecrets(
		implantPeerAgeKeyPair.Public,
		implantPeerAgeKeyPair.Private,
		MinisignServerSign([]byte(implantPeerAgeKeyPair.Public)),
		serverAgeKeyPair.Public,
		MinisignServerPublicKey(),
	)
}

func TestAgeEncryptDecrypt(t *testing.T) {
	encrypted, err := AgeEncrypt(serverAgeKeyPair.Public, sample1)
	if err != nil {
		t.Fatal(err)
	}
	decrypted, err := AgeDecrypt(serverAgeKeyPair.Private, encrypted)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(sample1, decrypted) {
		t.Fatalf("Sample does not match decrypted data")
	}
}

func TestAgeTamperEncryptDecrypt(t *testing.T) {
	encrypted, err := AgeEncrypt(serverAgeKeyPair.Public, sample1)
	if err != nil {
		t.Fatal(err)
	}
	encrypted[insecureRand.Intn(len(encrypted))] ^= 0xFF
	_, err = AgeDecrypt(serverAgeKeyPair.Private, encrypted)
	if err == nil {
		t.Fatal(err)
	}
}

func TestAgeWrongKeyEncryptDecrypt(t *testing.T) {
	encrypted, err := AgeEncrypt(serverAgeKeyPair.Public, sample1)
	if err != nil {
		t.Fatal(err)
	}
	keyPair, _ := RandomAgeKeyPair()
	_, err = AgeDecrypt(keyPair.Private, encrypted)
	if err == nil {
		t.Fatal(err)
	}
}

func TestAgeKeyEx(t *testing.T) {
	sessionKey := RandomSymmetricKey()
	plaintext := sessionKey[:]
	ciphertext, err := implantCrypto.AgeKeyExToServer(plaintext)
	if err != nil {
		t.Fatal(err)
	}
	decrypted, err := AgeKeyExFromImplant(
		serverAgeKeyPair.Private,
		implantPeerAgeKeyPair.Private,
		ciphertext[32:], // Remove prepended public key hash
	)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(plaintext, decrypted) {
		t.Fatalf("Session key does not match")
	}
}

func TestAgeKeyExTamper(t *testing.T) {
	sessionKey := RandomSymmetricKey()
	plaintext := sessionKey[:]
	allCiphertext, err := implantCrypto.AgeKeyExToServer(plaintext)
	if err != nil {
		t.Fatal(err)
	}

	// Tamper with the ciphertext
	ciphertext := allCiphertext[32:]
	ciphertext[insecureRand.Intn(len(ciphertext))] ^= 0xFF
	_, err = AgeKeyExFromImplant(
		serverAgeKeyPair.Private,
		implantPeerAgeKeyPair.Private,
		ciphertext,
	)
	if err == nil {
		t.Fatal(err)
	}

	// Leave an invalid header with valid ciphertext
	_, err = AgeKeyExFromImplant(
		serverAgeKeyPair.Private,
		implantPeerAgeKeyPair.Private,
		allCiphertext,
	)
	if err == nil {
		t.Fatal(err)
	}
}

func TestAgeKeyExReplay(t *testing.T) {
	sessionKey := RandomSymmetricKey()
	plaintext := sessionKey[:]
	allCiphertext, err := implantCrypto.AgeKeyExToServer(plaintext)
	if err != nil {
		t.Fatal(err)
	}

	ciphertext := allCiphertext[32:]
	_, err = AgeKeyExFromImplant(
		serverAgeKeyPair.Private,
		implantPeerAgeKeyPair.Private,
		ciphertext,
	)
	if err != nil {
		t.Fatal(err)
	}
	_, err = AgeKeyExFromImplant(
		serverAgeKeyPair.Private,
		implantPeerAgeKeyPair.Private,
		ciphertext,
	)
	if err == nil {
		t.Fatal(err)
	}
}

// TestEncryptDecrypt - Test AEAD functions
func TestEncryptDecrypt(t *testing.T) {
	key := RandomSymmetricKey()
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

	key = RandomSymmetricKey()
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
	key := RandomSymmetricKey()
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
	key := RandomSymmetricKey()
	cipher1, err := Encrypt(key, sample1)
	if err != nil {
		t.Fatal(err)
	}
	key2 := RandomSymmetricKey()
	_, err = Decrypt(key2, cipher1)
	if err == nil {
		t.Fatalf("Decrypted with wrong key, should have resulted in Fatal")
	}
}

// TestCipherContext - Test CipherContext
func TestCipherContext(t *testing.T) {
	testKey := RandomSymmetricKey()
	cipherCtx1 := &CipherContext{
		Key:    testKey,
		replay: &sync.Map{},
	}
	cipherCtx2 := implantCrypto.NewCipherContext(testKey)

	sample := randomData()

	ciphertext, err := cipherCtx1.Encrypt(sample)
	if err != nil {
		t.Fatalf("Failed to encrypt sample: %s", err)
	}
	_, err = cipherCtx1.Decrypt(ciphertext[minisign.RawSigSize:])
	if err != ErrReplayAttack {
		t.Fatal("Failed to detect replay attack (1)")
	}
	_, err = cipherCtx1.Decrypt(ciphertext[minisign.RawSigSize:])
	if err != ErrReplayAttack {
		t.Fatal("Failed to detect replay attack (2)")
	}

	plaintext, err := cipherCtx2.Decrypt(ciphertext)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(sample, plaintext) {
		t.Fatalf("Sample does not match decrypted data")
	}
	_, err = cipherCtx2.Decrypt(ciphertext)
	if err != implantCrypto.ErrReplayAttack {
		t.Fatal("Failed to detect replay attack (3)")
	}
}

// TestEncryptDecrypt - Test AEAD functions
func TestImplantEncryptDecrypt(t *testing.T) {
	key := RandomSymmetricKey()
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

	key = RandomSymmetricKey()
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
