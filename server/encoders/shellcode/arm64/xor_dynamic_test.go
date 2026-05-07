package arm64

import (
	"bytes"
	"testing"
)

func TestXorDynamicRoundTripAutoTerms(t *testing.T) {
	// Ensure splitExplicitTerms() returns false by making the would-be key terminator
	// (key[len-3]) present in candidateKey.
	coreKey := []byte{0x11, 0x22, 0x33, 0x44, 0x55, 0x11, 0x77, 0x88}
	payload := bytes.Repeat([]byte{0x00}, 64)

	out, err := XorDynamic(payload, coreKey)
	if err != nil {
		t.Fatalf("XorDynamic failed: %v", err)
	}
	if len(out) <= len(coreKey)+1+len(payload)+2 {
		t.Fatalf("xor_dynamic output too short: %d", len(out))
	}

	payloadTerm := out[len(out)-2:]
	encodedStart := len(out) - 2 - len(payload)
	if encodedStart <= 0 {
		t.Fatalf("bad encodedStart: %d", encodedStart)
	}
	keyTerm := out[encodedStart-1]
	encoded := out[encodedStart : len(out)-2]

	if bytes.IndexByte(coreKey, keyTerm) != -1 {
		t.Fatalf("keyTerm present in key")
	}
	if bytes.Contains(encoded, payloadTerm) {
		t.Fatalf("payloadTerm present in encoded payload")
	}

	stubLen := encodedStart - 1 - len(coreKey)
	if stubLen <= 0 {
		t.Fatalf("unexpected stub length: %d", stubLen)
	}
	if !bytes.Equal(out[stubLen:stubLen+len(coreKey)], coreKey) {
		t.Fatalf("key region mismatch")
	}

	decoded := make([]byte, len(encoded))
	for i := range encoded {
		decoded[i] = encoded[i] ^ coreKey[i%len(coreKey)]
	}
	if !bytes.Equal(decoded, payload) {
		t.Fatalf("decoded payload mismatch")
	}
}

func TestXorDynamicRoundTripExplicitTerms(t *testing.T) {
	coreKey := []byte{0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88}
	payload := bytes.Repeat([]byte{0x00}, 32)

	keyTerm := byte(0xFD)
	payloadTerm := []byte{0xFF, 0xFE}

	keyWithTerms := make([]byte, 0, len(coreKey)+3)
	keyWithTerms = append(keyWithTerms, coreKey...)
	keyWithTerms = append(keyWithTerms, keyTerm)
	keyWithTerms = append(keyWithTerms, payloadTerm...)

	out, err := XorDynamic(payload, keyWithTerms)
	if err != nil {
		t.Fatalf("XorDynamic failed: %v", err)
	}
	if len(out) <= len(coreKey)+1+len(payload)+2 {
		t.Fatalf("xor_dynamic output too short: %d", len(out))
	}

	if !bytes.Equal(out[len(out)-2:], payloadTerm) {
		t.Fatalf("payloadTerm mismatch")
	}
	encodedStart := len(out) - 2 - len(payload)
	if encodedStart <= 0 {
		t.Fatalf("bad encodedStart: %d", encodedStart)
	}
	if out[encodedStart-1] != keyTerm {
		t.Fatalf("keyTerm mismatch")
	}

	stubLen := encodedStart - 1 - len(coreKey)
	if stubLen <= 0 {
		t.Fatalf("unexpected stub length: %d", stubLen)
	}
	if !bytes.Equal(out[stubLen:stubLen+len(coreKey)], coreKey) {
		t.Fatalf("key region mismatch")
	}

	encoded := out[encodedStart : len(out)-2]
	decoded := make([]byte, len(encoded))
	for i := range encoded {
		decoded[i] = encoded[i] ^ coreKey[i%len(coreKey)]
	}
	if !bytes.Equal(decoded, payload) {
		t.Fatalf("decoded payload mismatch")
	}
}

func TestXorDynamicRejectsEmptyPayload(t *testing.T) {
	_, err := XorDynamic(nil, []byte{0x11})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestXorDynamicRejectsEmptyKey(t *testing.T) {
	_, err := XorDynamic([]byte{0x90}, nil)
	if err == nil {
		t.Fatal("expected error")
	}
}
