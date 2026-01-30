package amd64

import (
	"bytes"
	"fmt"
	"io/fs"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestXorDynamicEncoderMatchesMSFVenom(t *testing.T) {
	if _, err := exec.LookPath("msfvenom"); err != nil {
		t.Skipf("msfvenom not available: %v", err)
	}
	if !keystoneAvailable() {
		t.Skip("keystone assembler not available")
	}

	fixtures, err := fs.Glob(xorFixtures, "testdata/*.bin")
	if err != nil {
		t.Fatalf("failed to list fixtures: %v", err)
	}
	if len(fixtures) == 0 {
		t.Fatal("no xor fixtures found")
	}

	home := t.TempDir()
	primeMSFVenom(t, home)

	for _, name := range fixtures {
		name := name
		t.Run(filepath.Base(name), func(t *testing.T) {
			payload, err := xorFixtures.ReadFile(name)
			if err != nil {
				t.Fatalf("failed to read fixture %s: %v", name, err)
			}
			if len(payload) == 0 {
				t.Fatalf("fixture %s is empty", name)
			}

			platform, err := platformFromFixture(name)
			if err != nil {
				t.Fatalf("fixture %s: %v", name, err)
			}

			msfEncoded := msfvenomEncode(t, home, payload, platform, "x64/xor_dynamic")
			key, keyTerm, payloadTerm, err := extractXorDynamicParams(msfEncoded)
			if err != nil {
				t.Fatalf("fixture %s: %v", name, err)
			}

			keyWithTerms := make([]byte, 0, len(key)+3)
			keyWithTerms = append(keyWithTerms, key...)
			keyWithTerms = append(keyWithTerms, keyTerm)
			keyWithTerms = append(keyWithTerms, payloadTerm...)

			encoded, err := XorDynamic(payload, keyWithTerms)
			if err != nil {
				t.Fatalf("fixture %s: XorDynamic failed: %v", name, err)
			}

			if !bytes.Equal(encoded, msfEncoded) {
				diff := firstDiff(encoded, msfEncoded)
				t.Fatalf("fixture %s: output mismatch at byte %d (got=%d, msf=%d)", name, diff, len(encoded), len(msfEncoded))
			}
		})
	}
}

func extractXorDynamicParams(encoded []byte) ([]byte, byte, []byte, error) {
	if len(encoded) < xorDynamicStubSize+3 {
		return nil, 0, nil, fmt.Errorf("xor_dynamic output too short")
	}

	stub := encoded[:xorDynamicStubSize]
	keyTerm, payloadTerm, err := extractXorDynamicTerms(stub)
	if err != nil {
		return nil, 0, nil, err
	}

	rest := encoded[xorDynamicStubSize:]
	keyEnd := bytes.IndexByte(rest, keyTerm)
	if keyEnd == -1 {
		return nil, 0, nil, fmt.Errorf("xor_dynamic: key terminator not found in payload")
	}

	key := rest[:keyEnd]
	if len(key) == 0 {
		return nil, 0, nil, fmt.Errorf("xor_dynamic: empty key in payload")
	}

	if len(rest) < keyEnd+1+2 {
		return nil, 0, nil, fmt.Errorf("xor_dynamic: payload missing terminator")
	}

	if len(rest) < 2 || !bytes.HasSuffix(rest, payloadTerm) {
		return nil, 0, nil, fmt.Errorf("xor_dynamic: payload terminator mismatch")
	}

	return key, keyTerm, payloadTerm, nil
}

func extractXorDynamicTerms(stub []byte) (byte, []byte, error) {
	movIdx := bytes.Index(stub, []byte{0xB0})
	if movIdx == -1 || movIdx+1 >= len(stub) {
		return 0, nil, fmt.Errorf("xor_dynamic: mov al not found in stub")
	}
	keyTerm := stub[movIdx+1]

	cmpIdx := bytes.Index(stub, []byte{0x80, 0x3E})
	if cmpIdx == -1 || cmpIdx+2 >= len(stub) {
		return 0, nil, fmt.Errorf("xor_dynamic: cmp byte not found in stub")
	}
	if stub[cmpIdx+2] != keyTerm {
		return 0, nil, fmt.Errorf("xor_dynamic: stub key terminator mismatch")
	}

	termIdx := bytes.Index(stub, []byte{0x66, 0x81, 0x3F})
	if termIdx == -1 || termIdx+4 >= len(stub) {
		return 0, nil, fmt.Errorf("xor_dynamic: cmp word not found in stub")
	}

	payloadTerm := []byte{stub[termIdx+3], stub[termIdx+4]}
	return keyTerm, payloadTerm, nil
}
