package buildtransform

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/bishopfox/sliver/server/buildplan"
)

func encoderStep(name string, target buildplan.Target) buildplan.Step {
	artifact := buildplan.Artifact{Kind: "shellcode", Container: "raw", Target: target}
	return buildplan.Step{
		Kind: "transform", Provider: "server",
		Spec:  buildplan.Reference{Name: name, Version: "1.0.0", Digest: encoderBindings[name].digest},
		Input: &artifact, Output: artifact, Toolchain: buildplan.Toolchain{Driver: "builtin"},
	}
}

func TestEncodeShellcodeInstalledMatrix(t *testing.T) {
	for _, tc := range []struct {
		name   string
		target buildplan.Target
	}{
		{"builtin/shikata-ga-nai", buildplan.Target{OS: "linux", Arch: "386"}},
		{"builtin/shikata-ga-nai", buildplan.Target{OS: "darwin", Arch: "amd64"}},
		{"builtin/xor", buildplan.Target{OS: "linux", Arch: "amd64"}},
		{"builtin/xor", buildplan.Target{OS: "windows", Arch: "arm64"}},
		{"builtin/xor-dynamic", buildplan.Target{OS: "darwin", Arch: "amd64"}},
		{"builtin/xor-dynamic", buildplan.Target{OS: "linux", Arch: "arm64"}},
	} {
		t.Run(tc.name+"/"+tc.target.OS+"/"+tc.target.Arch, func(t *testing.T) {
			if !SupportsEncoder(tc.name, tc.target) {
				t.Fatal("catalog-declared encoder target is unavailable")
			}
			input := bytes.Repeat([]byte{0x90}, 96)
			original := bytes.Clone(input)
			output, err := EncodeShellcode(context.Background(), encoderStep(tc.name, tc.target), input)
			if err != nil {
				t.Fatalf("encode shellcode: %v", err)
			}
			if len(output) == 0 || bytes.Equal(output, input) || !bytes.Equal(input, original) {
				t.Fatal("encoder returned empty or unchanged bytes, or mutated the prior step")
			}
		})
	}
}

func TestEncodeShellcodeRejectsUnsupportedEdge(t *testing.T) {
	base := encoderStep("builtin/xor", buildplan.Target{OS: "linux", Arch: "amd64"})
	input := []byte{0x90, 0xc3}
	for _, tc := range []struct {
		name string
		edit func(*buildplan.Step)
	}{
		{"wrong digest", func(step *buildplan.Step) { step.Spec.Digest = "sha256:" + strings.Repeat("0", 64) }},
		{"wrong version", func(step *buildplan.Step) { step.Spec.Version = "2.0.0" }},
		{"unexpected option", func(step *buildplan.Step) {
			step.Options = []buildplan.Option{{Name: "iterations", ValueJSON: []byte("2")}}
		}},
		{"wrong input", func(step *buildplan.Step) { step.Input.Kind = "executable" }},
		{"target change", func(step *buildplan.Step) { step.Output.Target.Arch = "arm64" }},
		{"unsupported architecture", func(step *buildplan.Step) {
			step.Output.Target.Arch = "386"
			step.Input.Target.Arch = "386"
		}},
		{"remote provider", func(step *buildplan.Step) { step.Provider = "remote" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			step := base
			artifact := *base.Input
			step.Input = &artifact
			tc.edit(&step)
			if _, err := EncodeShellcode(context.Background(), step, input); err == nil {
				t.Fatal("invalid encoder edge was accepted")
			}
		})
	}
	if SupportsEncoder("builtin/shikata-ga-nai", buildplan.Target{OS: "darwin", Arch: "386"}) ||
		SupportsEncoder("builtin/xor", buildplan.Target{OS: "linux", Arch: "386"}) ||
		SupportsEncoder("builtin/xor-dynamic", buildplan.Target{OS: "freebsd", Arch: "amd64"}) {
		t.Fatal("encoder advertised an unsupported target")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := EncodeShellcode(ctx, base, input); err != context.Canceled {
		t.Fatalf("canceled encoder returned %v", err)
	}
}

func TestEncodeShellcodeCompatUsesInstalledEncoder(t *testing.T) {
	input := bytes.Repeat([]byte{0x90}, 32)
	original := bytes.Clone(input)
	defaultEncoded, err := EncodeShellcodeCompat(context.Background(), "xor", "amd64", 0, nil, input)
	if err != nil || len(defaultEncoded) <= len(input) || !bytes.Equal(input, original) {
		t.Fatalf("compat one-pass encoding failed or changed input: output=%d err=%v", len(defaultEncoded), err)
	}
	twiceEncoded, err := EncodeShellcodeCompat(context.Background(), "xor", "amd64", 2, nil, input)
	if err != nil || len(twiceEncoded) <= len(defaultEncoded) {
		t.Fatalf("compat iterations were not applied: one=%d two=%d err=%v", len(defaultEncoded), len(twiceEncoded), err)
	}
	if _, err := EncodeShellcodeCompat(context.Background(), "xor", "386", 1, nil, input); err == nil {
		t.Fatal("unsupported compatibility encoder architecture was accepted")
	}
}
