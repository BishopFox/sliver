package sgn

import (
	"embed"
	"errors"
	"testing"

	sgnpkg "github.com/moloch--/sgn/pkg"
)

//go:embed testdata/*.bin
var msfFixtures embed.FS

func TestParseArchitecture(t *testing.T) {
	cases := map[string]int{
		"amd64":  64,
		"x86_64": 64,
		"64":     64,
		"386":    32,
		"x86":    32,
		"32":     32,
		"":       64,
	}

	for input, expected := range cases {
		actual, err := parseArchitecture(input)
		if err != nil {
			t.Fatalf("parseArchitecture(%q) returned error: %v", input, err)
		}
		if expected != actual {
			t.Fatalf("parseArchitecture(%q) = %d, expected %d", input, actual, expected)
		}
	}
}

func TestParseArchitectureInvalid(t *testing.T) {
	if _, err := parseArchitecture("arm"); err == nil {
		t.Fatal("parseArchitecture should fail for unsupported architecture")
	}
}

func TestNewEncoderWithConfigAppliesOptions(t *testing.T) {
	cfg := SGNConfig{
		MaxObfuscation: 16,
		PlainDecoder:   true,
		Safe:           true,
		Iterations:     3,
	}

	encoder, err := newEncoderWithConfig(64, cfg)
	if err != nil {
		t.Fatalf("newEncoderWithConfig returned error: %v", err)
	}

	if encoder.ObfuscationLimit != cfg.MaxObfuscation {
		t.Fatalf("ObfuscationLimit = %d, expected %d", encoder.ObfuscationLimit, cfg.MaxObfuscation)
	}
	if !encoder.PlainDecoder {
		t.Fatal("PlainDecoder not applied")
	}
	if !encoder.SaveRegisters {
		t.Fatal("SaveRegisters not applied")
	}
	if encoder.EncodingCount != cfg.Iterations {
		t.Fatalf("EncodingCount = %d, expected %d", encoder.EncodingCount, cfg.Iterations)
	}
}

func TestNewEncoderWithConfigDefaults(t *testing.T) {
	cfg := SGNConfig{}
	expected, err := sgnpkg.NewEncoder(32)
	if err != nil {
		t.Fatalf("sgnpkg.NewEncoder: %v", err)
	}
	encoder, err := newEncoderWithConfig(32, cfg)
	if err != nil {
		t.Fatalf("newEncoderWithConfig returned error: %v", err)
	}

	if encoder.ObfuscationLimit != expected.ObfuscationLimit {
		t.Fatalf("expected default ObfuscationLimit %d, got %d", expected.ObfuscationLimit, encoder.ObfuscationLimit)
	}
	if encoder.EncodingCount != expected.EncodingCount {
		t.Fatalf("expected default EncodingCount %d, got %d", expected.EncodingCount, encoder.EncodingCount)
	}
}

func TestMeetsConstraintsASCII(t *testing.T) {
	cfg := SGNConfig{Asci: true}
	if !meetsConstraints([]byte("OK"), cfg) {
		t.Fatal("expected ASCII payload to satisfy constraints")
	}
	if meetsConstraints([]byte{0x01}, cfg) {
		t.Fatal("expected non-printable payload to fail constraints")
	}
}

func TestMeetsConstraintsBadChars(t *testing.T) {
	cfg := SGNConfig{BadChars: []byte{0x00, 0xff}}
	if meetsConstraints([]byte{0x41, 0x00}, cfg) {
		t.Fatal("expected payload containing bad char to fail")
	}
	if !meetsConstraints([]byte{0x41, 0x42}, cfg) {
		t.Fatal("expected payload without bad chars to pass")
	}
}

func TestIsASCIIPrintable(t *testing.T) {
	if !isASCIIPrintable([]byte("Hello, world!")) {
		t.Fatal("expected printable string to be ASCII printable")
	}
	if isASCIIPrintable([]byte{0x1b}) {
		t.Fatal("expected escape byte to be non printable")
	}
}

func TestNextSeed(t *testing.T) {
	if nextSeed(0) != 1 {
		t.Fatal("expected nextSeed(0) == 1")
	}
	if nextSeed(254) != 0 {
		t.Fatalf("expected nextSeed(254) == 0, got %d", nextSeed(254))
	}
}

func TestEncodeShellcodeWithConfigEmptyPayload(t *testing.T) {
	cfg := SGNConfig{Architecture: "amd64"}
	if _, err := EncodeShellcodeWithConfig(nil, cfg); err == nil {
		t.Fatal("expected error for empty payload")
	}
}

func TestEncodeShellcodeWithConfigInvalidArch(t *testing.T) {
	cfg := SGNConfig{Architecture: "arm"}
	if _, err := EncodeShellcodeWithConfig([]byte{0x90}, cfg); err == nil {
		t.Fatal("expected error for invalid architecture")
	}
}

func TestNewEncoderWithConfigInvalidArch(t *testing.T) {
	cfg := SGNConfig{}
	if _, err := newEncoderWithConfig(0, cfg); err == nil {
		t.Fatal("expected newEncoderWithConfig to fail for invalid arch")
	}
}

func TestNewEncoderWithConfigNonPositiveIterations(t *testing.T) {
	defaultEncoder, err := sgnpkg.NewEncoder(64)
	if err != nil {
		t.Fatalf("sgnpkg.NewEncoder: %v", err)
	}

	cfg := SGNConfig{Iterations: 0}
	encoderZero, err := newEncoderWithConfig(64, cfg)
	if err != nil {
		t.Fatalf("newEncoderWithConfig returned error: %v", err)
	}
	if encoderZero.EncodingCount != defaultEncoder.EncodingCount {
		t.Fatalf("expected EncodingCount to remain default %d, got %d", defaultEncoder.EncodingCount, encoderZero.EncodingCount)
	}

	cfg.Iterations = -3
	encoderNegative, err := newEncoderWithConfig(64, cfg)
	if err != nil {
		t.Fatalf("newEncoderWithConfig returned error: %v", err)
	}
	if encoderNegative.EncodingCount != defaultEncoder.EncodingCount {
		t.Fatalf("expected EncodingCount to remain default %d, got %d", defaultEncoder.EncodingCount, encoderNegative.EncodingCount)
	}
}

func TestEncodeShellcodeInvalidArch(t *testing.T) {
	if _, err := EncodeShellcode([]byte{0x90}, "arm", 1, nil); err == nil {
		t.Fatal("expected error for invalid architecture")
	}
}

func TestEncodeShellcodeEmpty(t *testing.T) {
	if _, err := EncodeShellcode(nil, "amd64", 1, nil); err == nil {
		t.Fatal("expected error for empty payload")
	}
}

func TestEncodeMSFVenomFixtures(t *testing.T) {
	t.Helper()
	checkEncoder, err := newEncoderWithConfig(64, SGNConfig{})
	if err != nil {
		t.Fatalf("newEncoderWithConfig failed: %v", err)
	}
	checkEncoder.Seed = 0x42
	if _, err := simpleEncode(checkEncoder, []byte{0x90, 0x90}); err != nil {
		t.Skipf("keystone assembler not available; skipping MSF shellcode encoding tests (%v)", err)
	}

	cases := []struct {
		filename string
		arch     string
	}{
		{"windows-meterpreter-reverse-tcp.x64.bin", "amd64"},
		{"windows-meterpreter-reverse-http.x86.bin", "386"},
		{"windows-exec-calc.x64.bin", "amd64"},
	}

	for _, tc := range cases {
		data, readErr := msfFixtures.ReadFile("testdata/" + tc.filename)
		if readErr != nil {
			t.Fatalf("failed to read fixture %s: %v", tc.filename, readErr)
		}
		if len(data) == 0 {
			t.Fatalf("fixture %s is empty", tc.filename)
		}
		cfg := SGNConfig{
			Architecture:   tc.arch,
			PlainDecoder:   true,
			MaxObfuscation: 2048,
		}
		var encoded []byte
		var err error
		for attempt := 0; attempt < 5; attempt++ {
			encoded, err = EncodeShellcodeWithConfig(data, cfg)
			if err == nil {
				break
			}
			if !errors.Is(err, ErrFailedToEncode) {
				t.Fatalf("unexpected error encoding %s: %v", tc.filename, err)
			}
		}
		if err != nil {
			t.Fatalf("EncodeShellcodeWithConfig failed for %s after retries: %v", tc.filename, err)
		}
		if len(encoded) == 0 {
			t.Fatalf("encoded payload for %s is empty", tc.filename)
		}
	}
}
