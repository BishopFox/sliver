package minisign

import (
	"crypto/rand"
	insecureRand "math/rand"
	"testing"
)

func TestRawSigValid(t *testing.T) {
	publicKey, privateKey, err := GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}
	for i := 0; i < 100; i++ {
		message := randomBuf()
		signature := SignRawBuf(privateKey, message)
		rawMsg := append(signature[:], message...)
		if !VerifyRawBuf(publicKey, rawMsg) {
			t.Fatalf("Verification failed: signature %q - public key %q", signature, publicKey)
		}
	}
}

func TestRawSigInvalidKey(t *testing.T) {
	_, privateKeyA, err := GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}
	publicKeyB, _, err := GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}
	for i := 0; i < 100; i++ {
		message := randomBuf()
		signature := SignRawBuf(privateKeyA, message)
		rawMsg := append(signature[:], message...)
		if VerifyRawBuf(publicKeyB, rawMsg) {
			t.Fatalf("Verification expected to fail, but didn't: signature %q - public key %q", signature, publicKeyB)
		}
	}
}

func TestRawSigInvalidTamper(t *testing.T) {
	publicKey, privateKey, err := GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}
	for i := 0; i < 100; i++ {
		message := randomBuf()
		signature := SignRawBuf(privateKey, message)
		message[insecureRand.Intn(len(message))] ^= 0xFF
		message[insecureRand.Intn(len(message))] ^= 0xFF
		message[insecureRand.Intn(len(message))] ^= 0xFF
		message[insecureRand.Intn(len(message))] ^= 0xFF
		message[insecureRand.Intn(len(message))] ^= 0xFF
		rawMsg := append(signature[:], message...)
		if VerifyRawBuf(publicKey, rawMsg) {
			t.Fatalf("Verification expected to fail, but didn't: signature %q - public key %q", signature, publicKey)
		}
	}
}

func randomBuf() []byte {
	buf := make([]byte, insecureRand.Intn(4096)+1)
	rand.Read(buf)
	return buf
}
