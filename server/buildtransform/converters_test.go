package buildtransform

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bishopfox/sliver/server/buildplan"
)

func converterTestStep(name, osName, arch, kind, container, mode string) buildplan.Step {
	target := buildplan.Target{OS: osName, Arch: arch}
	return buildplan.Step{
		Kind: "transform", Provider: "server", Spec: buildplan.Reference{
			Name: name, Version: "1.0.0", Digest: converterDigests[name],
		},
		Input: &buildplan.Artifact{Kind: kind, Container: container, Target: target, GoBuildMode: mode,
			EntrySymbols: []string{"StartW"}},
		Output:    buildplan.Artifact{Kind: "shellcode", Container: "raw", Target: target},
		Toolchain: buildplan.Toolchain{Driver: "builtin"},
	}
}

func TestConverterRejectsWrongFrozenArtifactOrTarget(t *testing.T) {
	valid := []buildplan.Step{
		converterTestStep("builtin/donut", "windows", "amd64", "executable", "pe", "pie"),
		converterTestStep("builtin/beignet", "darwin", "arm64", "shared-library", "macho", "c-shared"),
		converterTestStep("builtin/malasada", "linux", "amd64", "shared-library", "elf", "c-shared"),
	}
	for _, step := range valid {
		if err := validateConverterStep(step); err != nil {
			t.Fatalf("valid %s step: %v", step.Spec.Name, err)
		}
	}
	for _, tc := range []struct {
		name string
		step buildplan.Step
	}{
		{name: "Donut arm64", step: converterTestStep("builtin/donut", "windows", "arm64", "executable", "pe", "pie")},
		{name: "Beignet amd64", step: converterTestStep("builtin/beignet", "darwin", "amd64", "shared-library", "macho", "c-shared")},
		{name: "Malasada archive", step: converterTestStep("builtin/malasada", "linux", "amd64", "c-archive", "ar", "c-archive")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := validateConverterStep(tc.step); err == nil {
				t.Fatal("incompatible converter step was accepted")
			}
		})
	}
	wrongProvider := valid[0]
	wrongProvider.Provider = "builder-a"
	if err := validateConverterStep(wrongProvider); err == nil {
		t.Fatal("remote converter was accepted as local")
	}
	wrongDigest := valid[0]
	wrongDigest.Spec.Digest = "sha256:" + strings.Repeat("0", 64)
	if err := validateConverterStep(wrongDigest); err == nil {
		t.Fatal("uninstalled converter digest was accepted")
	}
	missingEntry := valid[2]
	missingEntry.Input.EntrySymbols = nil
	if err := validateConverterStep(missingEntry); err == nil {
		t.Fatal("Malasada accepted a missing entry symbol")
	}
}

func TestConverterOptionsPreserveLegacyDonutDefaultsAndRejectInvalidValues(t *testing.T) {
	step := converterTestStep("builtin/donut", "windows", "386", "executable", "pe", "pie")
	options, err := converterOptions(step)
	defaults := parsedConverterOptions{entropy: 1, exitOption: 1, bypass: 3, headers: 1}
	if err != nil || options != defaults {
		t.Fatalf("legacy Donut defaults changed: %+v, %v", options, err)
	}
	step.Options = []buildplan.Option{
		{Name: "compress", ValueJSON: json.RawMessage(`true`)},
		{Name: "entropy", ValueJSON: json.RawMessage(`2`)},
		{Name: "exitOption", ValueJSON: json.RawMessage(`3`)},
		{Name: "bypass", ValueJSON: json.RawMessage(`1`)},
		{Name: "headers", ValueJSON: json.RawMessage(`2`)},
		{Name: "thread", ValueJSON: json.RawMessage(`true`)},
		{Name: "unicode", ValueJSON: json.RawMessage(`true`)},
		{Name: "originalEntryPoint", ValueJSON: json.RawMessage(`4294967295`)},
	}
	options, err = converterOptions(step)
	explicit := parsedConverterOptions{
		compress: true, entropy: 2, exitOption: 3, bypass: 1, headers: 2,
		thread: true, unicode: true, originalEntryPoint: ^uint32(0),
	}
	if err != nil || options != explicit {
		t.Fatalf("Donut options changed: %+v, %v", options, err)
	}
	for _, tc := range []buildplan.Option{
		{Name: "entropy", ValueJSON: json.RawMessage(`4`)},
		{Name: "compress", ValueJSON: json.RawMessage(`2`)},
		{Name: "originalEntryPoint", ValueJSON: json.RawMessage(`4294967296`)},
		{Name: "unknown", ValueJSON: json.RawMessage(`true`)},
	} {
		invalid := step
		invalid.Options = []buildplan.Option{tc}
		if _, err := converterOptions(invalid); err == nil {
			t.Fatalf("invalid Donut option %s=%s was accepted", tc.Name, tc.ValueJSON)
		}
	}
	linux := converterTestStep("builtin/malasada", "linux", "amd64", "shared-library", "elf", "c-shared")
	linux.Options = []buildplan.Option{{Name: "thread", ValueJSON: json.RawMessage(`true`)}}
	if _, err := converterOptions(linux); err == nil {
		t.Fatal("Windows-only option was accepted for Malasada")
	}
}

func TestConverterRejectsMalformedInputAndRemovesTemporarySharedObject(t *testing.T) {
	step := converterTestStep("builtin/donut", "windows", "amd64", "executable", "pe", "pie")
	if _, err := Convert(context.Background(), step, []byte("not a PE"), t.TempDir()); err == nil ||
		!strings.Contains(err.Error(), "invalid Donut input") {
		t.Fatalf("malformed PE was accepted: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := Convert(ctx, step, []byte("not a PE"), t.TempDir()); err != context.Canceled {
		t.Fatalf("canceled conversion returned %v", err)
	}
	workDir := t.TempDir()
	if _, err := convertMalasada(context.Background(), []byte("not an ELF"), "StartW", false, workDir); err == nil {
		t.Fatal("malformed ELF was accepted")
	}
	files, err := filepath.Glob(filepath.Join(workDir, "malasada-*.so"))
	if err != nil || len(files) != 0 {
		t.Fatalf("temporary shared object remained: %v, %v", files, err)
	}
	if _, err := os.Stat(workDir); err != nil {
		t.Fatalf("test work directory disappeared: %v", err)
	}
}
