package arm64

import (
	"bytes"
	"testing"
)

func TestSelectPayloadTermFallsBackTo4Bytes(t *testing.T) {
	allowed := allowedDynamicChars()

	// Populate the encoded payload with every allowed *distinct* 2-byte sequence
	// so no 2-byte terminator can be selected.
	encoded := make([]byte, 0, len(allowed)*len(allowed)*2)
	for _, first := range allowed {
		for _, second := range allowed {
			if first == second {
				continue
			}
			encoded = append(encoded, first, second)
		}
	}

	term, err := selectPayloadTerm(encoded)
	if err != nil {
		t.Fatalf("selectPayloadTerm failed: %v", err)
	}
	if len(term) != 4 {
		t.Fatalf("expected 4-byte terminator, got %d bytes", len(term))
	}
	if bytes.Contains(encoded, term) {
		t.Fatalf("4-byte terminator unexpectedly present in encoded payload")
	}

	seen := map[byte]bool{}
	for _, b := range term {
		if seen[b] {
			t.Fatalf("terminator bytes are not distinct: %x", term)
		}
		seen[b] = true
	}
}
