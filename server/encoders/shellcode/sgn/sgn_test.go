package sgn

import (
	"bytes"
	"embed"
	"errors"
	"io"
	"strings"
	"sync"
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

func TestEncodeShellcodeWithConfigDeterministicReplay(t *testing.T) {
	payload := []byte("Sliver SGN deterministic replay")
	original := append([]byte(nil), payload...)
	entropy := make([]byte, 1+sgnpkg.RandomSeedSize)
	for index := range entropy {
		entropy[index] = byte(index)
	}

	for _, architecture := range []string{"386", "amd64"} {
		t.Run(architecture, func(t *testing.T) {
			cfg := SGNConfig{
				Architecture:   architecture,
				Iterations:     2,
				MaxObfuscation: 100,
				PlainDecoder:   false,
				Safe:           true,
			}
			encode := func(source []byte) []byte {
				t.Helper()
				encoded, err := encodeShellcodeWithConfig(payload, cfg, bytes.NewReader(source))
				if err != nil {
					t.Fatalf("encodeShellcodeWithConfig: %v", err)
				}
				if len(encoded) == 0 {
					t.Fatal("encodeShellcodeWithConfig returned an empty payload")
				}
				return encoded
			}

			first := encode(entropy)
			second := encode(entropy)
			if !bytes.Equal(first, second) {
				t.Fatal("fixed SGN entropy did not produce byte-identical output")
			}
			if !bytes.Equal(payload, original) {
				t.Fatal("SGN encoding mutated the input payload")
			}

			differentEntropy := append([]byte(nil), entropy...)
			differentEntropy[len(differentEntropy)-1] ^= 0xff
			if bytes.Equal(first, encode(differentEntropy)) {
				t.Fatal("different SGN entropy produced identical output")
			}
		})
	}
}

func TestEncodeShellcodeWithConfigConcurrentReplay(t *testing.T) {
	const workers = 8
	payload := []byte("Sliver SGN concurrent replay")
	cfg := SGNConfig{
		Architecture:   "amd64",
		Iterations:     1,
		MaxObfuscation: 100,
		PlainDecoder:   false,
		Safe:           true,
	}
	entropy := make([]byte, 1+sgnpkg.RandomSeedSize)
	for index := range entropy {
		entropy[index] = byte(0xa5 ^ index)
	}

	type result struct {
		payload []byte
		err     error
	}
	results := make(chan result, workers)
	var wait sync.WaitGroup
	for range workers {
		wait.Add(1)
		go func() {
			defer wait.Done()
			encoded, err := encodeShellcodeWithConfig(payload, cfg, bytes.NewReader(entropy))
			results <- result{payload: encoded, err: err}
		}()
	}
	wait.Wait()
	close(results)

	var expected []byte
	for result := range results {
		if result.err != nil {
			t.Fatalf("concurrent encodeShellcodeWithConfig: %v", result.err)
		}
		if expected == nil {
			expected = result.payload
			continue
		}
		if !bytes.Equal(result.payload, expected) {
			t.Fatal("concurrent fixed-entropy SGN output did not replay byte-identically")
		}
	}
}

func TestEncodeShellcodeWithConfigEntropyFailure(t *testing.T) {
	_, err := encodeShellcodeWithConfig(
		[]byte{0x90},
		SGNConfig{Architecture: "amd64"},
		bytes.NewReader(nil),
	)
	if !errors.Is(err, io.EOF) {
		t.Fatalf("encodeShellcodeWithConfig entropy error = %v, want EOF", err)
	}
}

func TestEncodeShellcodeWithConfigConstraintAttemptsAreStrict(t *testing.T) {
	badChars := make([]byte, 256)
	for index := range badChars {
		badChars[index] = byte(index)
	}
	entropy := bytes.NewReader(make([]byte, 64*(1+sgnpkg.RandomSeedSize)))
	_, err := encodeShellcodeWithConfig(
		[]byte{0x90},
		SGNConfig{Architecture: "amd64", BadChars: badChars},
		entropy,
	)
	if err == nil || !strings.Contains(err.Error(), "unable to satisfy encoding constraints") {
		t.Fatalf("encodeShellcodeWithConfig constraint error = %v", err)
	}
	if entropy.Len() != 0 {
		t.Fatalf("constraint search left %d entropy bytes, want 0 after 64 attempts", entropy.Len())
	}
}

func TestEncodeMSFVenomFixtures(t *testing.T) {
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
			Iterations:     1,
			PlainDecoder:   false,
			Safe:           true,
			MaxObfuscation: 100,
		}
		for variant := range 16 {
			entropy := make([]byte, 1+sgnpkg.RandomSeedSize)
			for index := range entropy {
				entropy[index] = byte(variant + index)
			}
			encoded, err := encodeShellcodeWithConfig(data, cfg, bytes.NewReader(entropy))
			if err != nil {
				t.Fatalf("encode %s variant %d: %v", tc.filename, variant, err)
			}
			if len(encoded) == 0 {
				t.Fatalf("encoded payload for %s variant %d is empty", tc.filename, variant)
			}
		}
	}
}
