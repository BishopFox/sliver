package buildtransform

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"compress/zlib"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/bishopfox/sliver/server/buildplan"
	"github.com/bishopfox/sliver/util"
)

func testStageStep(name string, input, output buildplan.Artifact, options ...buildplan.Option) buildplan.Step {
	digest := map[string]string{
		StageFrameSpec: StageFrameDigest, StageCompressSpec: StageCompressDigest,
		StageAESSpec: StageAESDigest, StageRC4Spec: StageRC4Digest,
		PEMetadataSpec: PEMetadataDigest,
	}[name]
	return buildplan.Step{
		Kind: "transform", Provider: "server",
		Spec:  buildplan.Reference{Name: name, Version: "1.0.0", Digest: digest},
		Input: &input, Output: output, Toolchain: buildplan.Toolchain{Driver: "builtin"}, Options: options,
	}
}

func testStageOption(name string, value string) buildplan.Option {
	return buildplan.Option{Name: name, ValueJSON: json.RawMessage(value)}
}

//nolint:gocyclo // The assertions cover each intermediate result in one transform sequence.
func TestStageFrameCompressionAndEncryption(t *testing.T) {
	target := buildplan.Target{OS: "windows", Arch: "amd64"}
	pe := buildplan.Artifact{Kind: "executable", Container: "pe", Target: target, GoBuildMode: "default"}
	stage := buildplan.Artifact{Kind: "stage", Container: "raw", Target: target}
	frame := testStageStep(StageFrameSpec, pe, stage, testStageOption("prependSize", "true"))
	input := []byte("stage-payload")
	framed, err := ApplyStageStep(context.Background(), frame, input, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(framed) != len(input)+4 || binary.LittleEndian.Uint32(framed[:4]) != uint32(len(input)) ||
		!bytes.Equal(framed[4:], input) {
		t.Fatalf("stage frame differs from legacy four-byte little-endian framing: %q", framed)
	}
	if !bytes.Equal(input, []byte("stage-payload")) {
		t.Fatal("stage frame mutated its input")
	}
	compress := testStageStep(StageCompressSpec, stage, stage, testStageOption("algorithm", `"zlib"`))
	compressed, err := ApplyStageStep(context.Background(), compress, framed, nil)
	if err != nil {
		t.Fatal(err)
	}
	reader, err := zlib.NewReader(bytes.NewReader(compressed))
	if err != nil {
		t.Fatal(err)
	}
	plain, err := io.ReadAll(reader)
	_ = reader.Close()
	if err != nil || !bytes.Equal(plain, framed) {
		t.Fatalf("stage compression changed framed bytes: %q, %v", plain, err)
	}
	key := []byte("1234567890abcdef")
	iv := []byte("0000000000000000")
	resolve := func(kind, name string) ([]byte, error) {
		if kind != "secret" {
			return nil, errors.New("unexpected material kind")
		}
		switch name {
		case "key":
			return key, nil
		case "iv":
			return iv, nil
		}
		return nil, errors.New("missing secret")
	}
	aes := testStageStep(StageAESSpec, stage, stage,
		testStageOption("key", `{"kind":"secret","name":"key"}`),
		testStageOption("iv", `{"kind":"secret","name":"iv"}`))
	encrypted, err := ApplyStageStep(context.Background(), aes, compressed, resolve)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(encrypted[:16], iv) || !bytes.Equal(util.PreludeDecrypt(append([]byte(nil), encrypted...), key), compressed) {
		t.Fatal("AES stage bytes cannot be decrypted with the legacy prelude format")
	}
}

func TestStageCompressionAlgorithmsAndRC4(t *testing.T) {
	artifact := buildplan.Artifact{Kind: "stage", Container: "raw", Target: buildplan.Target{OS: "linux", Arch: "amd64"}}
	input := bytes.Repeat([]byte("stage-deflate-"), 32)
	for _, tc := range []struct {
		algorithm string
		decode    func([]byte) ([]byte, error)
	}{
		{"gzip", func(data []byte) ([]byte, error) {
			reader, err := gzip.NewReader(bytes.NewReader(data))
			if err != nil {
				return nil, err
			}
			defer func() { _ = reader.Close() }()
			return io.ReadAll(reader)
		}},
		{"deflate", func(data []byte) ([]byte, error) {
			reader := flate.NewReader(bytes.NewReader(data))
			defer func() { _ = reader.Close() }()
			return io.ReadAll(reader)
		}},
		{"deflate9", func(data []byte) ([]byte, error) {
			reader := flate.NewReader(bytes.NewReader(data))
			defer func() { _ = reader.Close() }()
			return io.ReadAll(reader)
		}},
	} {
		t.Run(tc.algorithm, func(t *testing.T) {
			step := testStageStep(StageCompressSpec, artifact, artifact, testStageOption("algorithm", `"`+tc.algorithm+`"`))
			encoded, err := ApplyStageStep(context.Background(), step, input, nil)
			if err != nil {
				t.Fatal(err)
			}
			plain, err := tc.decode(encoded)
			if err != nil || !bytes.Equal(plain, input) {
				t.Fatalf("decode stage: %v", err)
			}
			if strings.HasPrefix(tc.algorithm, "deflate") && !bytes.Equal(encoded, util.DeflateBuf(input)) {
				t.Fatal("deflate output differs from legacy stage compressor")
			}
		})
	}
	rc4Step := testStageStep(StageRC4Spec, artifact, artifact,
		testStageOption("key", `{"kind":"secret","name":"rc4-key"}`))
	resolve := func(_, _ string) ([]byte, error) { return []byte("stage-key"), nil }
	ciphertext, err := ApplyStageStep(context.Background(), rc4Step, input, resolve)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(util.RC4EncryptUnsafe(ciphertext, []byte("stage-key")), input) {
		t.Fatal("RC4 output differs from legacy stage encryption")
	}
}

func TestStageTransformRejectsInvalidEdgesAndMaterial(t *testing.T) {
	stage := buildplan.Artifact{Kind: "stage", Container: "raw", Target: buildplan.Target{OS: "windows", Arch: "amd64"}}
	bad := testStageStep(StageCompressSpec, buildplan.Artifact{Kind: "executable", Container: "pe", Target: stage.Target}, stage)
	if _, err := ApplyStageStep(context.Background(), bad, []byte("data"), nil); err == nil {
		t.Fatal("compression accepted non-stage input")
	}
	bad = testStageStep(StageAESSpec, stage, stage,
		testStageOption("key", `{"kind":"secret","name":"key","extra":true}`),
		testStageOption("iv", `{"kind":"secret","name":"iv"}`))
	if _, err := ApplyStageStep(context.Background(), bad, []byte("data"), func(string, string) ([]byte, error) {
		return []byte("1234567890abcdef"), nil
	}); err == nil {
		t.Fatal("AES accepted malformed secret reference")
	}
	bad = testStageStep(StageRC4Spec, stage, stage, testStageOption("key", `{"kind":"secret","name":"key"}`))
	if _, err := ApplyStageStep(context.Background(), bad, []byte("data"), func(string, string) ([]byte, error) {
		return nil, errors.New("private-key-material")
	}); err == nil || strings.Contains(err.Error(), "private-key-material") {
		t.Fatalf("secret resolver error leaked private material: %v", err)
	}
	pe := buildplan.Artifact{Kind: "executable", Container: "pe", Target: stage.Target, GoBuildMode: "default"}
	bad = testStageStep(StageFrameSpec, pe, stage, testStageOption("prependSize", "null"))
	if _, err := ApplyStageStep(context.Background(), bad, []byte("data"), nil); err == nil {
		t.Fatal("frame accepted null boolean")
	}
	bad = testStageStep(StageFrameSpec, stage, stage)
	if _, err := ApplyStageStep(context.Background(), bad, []byte("data"), nil); err == nil {
		t.Fatal("frame accepted stage input absent from its catalog capability")
	}
	bad = testStageStep(StageFrameSpec, stage, stage)
	bad.Spec.Digest = "sha256:" + strings.Repeat("0", 64)
	if _, err := ApplyStageStep(context.Background(), bad, []byte("data"), nil); err == nil {
		t.Fatal("stage accepted an uninstalled spec digest")
	}
	bad = testStageStep(StageFrameSpec, stage, stage)
	bad.Toolchain.Driver = "go"
	if _, err := ApplyStageStep(context.Background(), bad, []byte("data"), nil); err == nil {
		t.Fatal("stage accepted an uninstalled toolchain")
	}
}
