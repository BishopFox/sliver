package assets

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestVerifyGarbleSHA256(t *testing.T) {
	artifact := filepath.Join(t.TempDir(), "garble")
	data := []byte("garble-control-flow")
	if err := os.WriteFile(artifact, data, 0o600); err != nil {
		t.Fatalf("write test artifact: %v", err)
	}
	digest := sha256.Sum256(data)
	expected := hex.EncodeToString(digest[:])

	if err := verifyGarbleSHA256(artifact, strings.ToUpper(expected)); err != nil {
		t.Fatalf("verifyGarbleSHA256() error = %v", err)
	}
	if err := verifyGarbleSHA256(artifact, strings.Repeat("0", sha256.Size*2)); err == nil {
		t.Fatal("verifyGarbleSHA256() accepted an incorrect digest")
	}
}

func TestGarblePlatformManifest(t *testing.T) {
	platforms := garblePlatforms()
	if len(platforms) != garbleTotal {
		t.Fatalf("garble platform count = %d, want %d", len(platforms), garbleTotal)
	}

	seen := map[string]struct{}{}
	for _, platform := range platforms {
		key := platform.os + "/" + platform.arch
		if _, duplicate := seen[key]; duplicate {
			t.Fatalf("duplicate Garble platform %s", key)
		}
		seen[key] = struct{}{}

		digest, err := hex.DecodeString(platform.sha256)
		if err != nil || len(digest) != sha256.Size {
			t.Fatalf("Garble platform %s has invalid sha256 %q", key, platform.sha256)
		}
		if !strings.Contains(platform.url, "/v"+garbleVersion+"/") {
			t.Fatalf("Garble platform %s URL %q does not use pinned version %s", key, platform.url, garbleVersion)
		}
		if platform.os == "windows" && platform.filename != "garble.exe" {
			t.Fatalf("Garble platform %s filename = %q, want garble.exe", key, platform.filename)
		}
		if platform.os != "windows" && platform.filename != "garble" {
			t.Fatalf("Garble platform %s filename = %q, want garble", key, platform.filename)
		}
		expected, ok := ExpectedGarbleSHA256(platform.os, platform.arch)
		if !ok || expected != platform.sha256 {
			t.Fatalf("ExpectedGarbleSHA256(%q, %q) = %q, %t; want %q, true", platform.os, platform.arch, expected, ok, platform.sha256)
		}
	}

	if _, ok := ExpectedGarbleSHA256("plan9", "mips"); ok {
		t.Fatal("ExpectedGarbleSHA256() accepted an unsupported platform")
	}
}
