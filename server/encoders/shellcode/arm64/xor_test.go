package arm64

import (
	"bytes"
	"fmt"
	"testing"
)

func TestXorRoundTrip(t *testing.T) {
	key := []byte{0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88}

	tests := [][]byte{
		{0x90},
		{0x90, 0x90, 0x90, 0x90, 0x90, 0x90, 0x90},
		{0x90, 0x90, 0x90, 0x90, 0x90, 0x90, 0x90, 0x90},
		{0x90, 0x90, 0x90, 0x90, 0x90, 0x90, 0x90, 0x90, 0x90},
		bytes.Repeat([]byte{0xAB}, 31),
		bytes.Repeat([]byte{0xAB}, 32),
		bytes.Repeat([]byte{0xAB}, 33),
	}

	for _, payload := range tests {
		payload := payload
		t.Run(fmt.Sprintf("len=%d", len(payload)), func(t *testing.T) {
			encoded, err := Xor(payload, key)
			if err != nil {
				t.Fatalf("Xor failed: %v", err)
			}

			paddedLen := ((len(payload) + xorBlockSize - 1) / xorBlockSize) * xorBlockSize
			if len(encoded) < paddedLen {
				t.Fatalf("encoded output too short: got=%d want>=%d", len(encoded), paddedLen)
			}

			stubLen := len(encoded) - paddedLen
			if stubLen <= 0 {
				t.Fatalf("unexpected stub length %d", stubLen)
			}

			encodedPayload := encoded[stubLen:]
			decoded := make([]byte, len(encodedPayload))
			copy(decoded, encodedPayload)
			for i := 0; i < len(decoded); i += xorBlockSize {
				for j := 0; j < xorKeySize; j++ {
					decoded[i+j] ^= key[j]
				}
			}

			if !bytes.Equal(decoded[:len(payload)], payload) {
				t.Fatalf("decoded payload mismatch")
			}
			if !bytes.Equal(decoded[len(payload):], make([]byte, len(decoded)-len(payload))) {
				t.Fatalf("decoded padding not zeroed")
			}
		})
	}
}

func TestXorRejectsEmptyPayload(t *testing.T) {
	_, err := Xor(nil, []byte{0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestXorRejectsInvalidKeyLength(t *testing.T) {
	_, err := Xor([]byte{0x90}, []byte{0x01})
	if err == nil {
		t.Fatal("expected error")
	}
}
