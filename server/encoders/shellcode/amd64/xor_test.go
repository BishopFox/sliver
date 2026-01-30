package amd64

import (
	"bytes"
	"embed"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/moloch--/go-keystone"
)

//go:embed testdata/*.bin
var xorFixtures embed.FS

func TestXorEncoderMatchesMSFVenom(t *testing.T) {
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

			msfEncoded := msfvenomEncode(t, home, payload, platform)
			key, err := extractMSFKey(msfEncoded)
			if err != nil {
				t.Fatalf("fixture %s: %v", name, err)
			}

			encoded, err := Xor(payload, key)
			if err != nil {
				t.Fatalf("fixture %s: Xor failed: %v", name, err)
			}

			if !bytes.Equal(encoded, msfEncoded) {
				diff := firstDiff(encoded, msfEncoded)
				t.Fatalf("fixture %s: output mismatch at byte %d (got=%d, msf=%d)", name, diff, len(encoded), len(msfEncoded))
			}
		})
	}
}

func keystoneAvailable() bool {
	engine, err := keystone.NewEngine(keystone.ARCH_X86, keystone.MODE_64)
	if err != nil {
		return false
	}
	_ = engine.Close()
	return true
}

func msfvenomEncode(t *testing.T, home string, payload []byte, platform string) []byte {
	t.Helper()

	outFile, err := os.CreateTemp(home, "msf-xor-*.bin")
	if err != nil {
		t.Fatalf("msfvenom temp file: %v", err)
	}
	outPath := outFile.Name()
	_ = outFile.Close()
	defer func() { _ = os.Remove(outPath) }()

	cmd := exec.Command("msfvenom",
		"-p", "-",
		"-a", "x64",
		"--platform", platform,
		"-e", "x64/xor",
		"-i", "1",
		"-f", "raw",
		"-o", outPath,
	)
	cmd.Env = append(os.Environ(), "HOME="+home)
	cmd.Stdin = bytes.NewReader(payload)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		t.Fatalf("msfvenom failed: %v (stderr=%s)", err, strings.TrimSpace(stderr.String()))
	}

	encoded, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("msfvenom read output: %v", err)
	}
	if len(encoded) == 0 {
		t.Fatalf("msfvenom produced empty output (stderr=%s)", strings.TrimSpace(stderr.String()))
	}
	return encoded
}

func primeMSFVenom(t *testing.T, home string) {
	t.Helper()

	cmd := exec.Command("msfvenom", "-l", "encoders")
	cmd.Env = append(os.Environ(), "HOME="+home)
	cmd.Stdin = bytes.NewReader(nil)

	var stderr bytes.Buffer
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("msfvenom init failed: %v (stderr=%s)", err, strings.TrimSpace(stderr.String()))
	}
}

func extractMSFKey(encoded []byte) ([]byte, error) {
	keyOffset := bytes.Index(encoded, []byte{0x48, 0xBB})
	if keyOffset == -1 || keyOffset+10 > len(encoded) {
		return nil, fmt.Errorf("failed to locate xor key in msf output")
	}
	key := make([]byte, xorKeySize)
	copy(key, encoded[keyOffset+2:keyOffset+10])
	return key, nil
}

func platformFromFixture(name string) (string, error) {
	base := filepath.Base(name)
	switch {
	case strings.HasPrefix(base, "linux-"):
		return "linux", nil
	case strings.HasPrefix(base, "windows-"):
		return "windows", nil
	default:
		return "", fmt.Errorf("unknown platform for fixture %s", base)
	}
}

func firstDiff(a, b []byte) int {
	min := len(a)
	if len(b) < min {
		min = len(b)
	}
	for i := 0; i < min; i++ {
		if a[i] != b[i] {
			return i
		}
	}
	return min
}
