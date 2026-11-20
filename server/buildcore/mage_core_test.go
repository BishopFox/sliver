package buildcore

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/bishopfox/sliver/server/buildcore/magebuild"
	"github.com/bishopfox/sliver/server/buildcore/workspace"
	"github.com/bishopfox/sliver/server/buildplan"
)

const fixtureMagefile = `//go:build mage

package main

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

func Build(ctx context.Context, manifestPath string) error {
	if err := ctx.Err(); err != nil { return err }
	data, err := os.ReadFile(manifestPath)
	if err != nil { return err }
	var input struct { SourceDir, BinDir, MaterialPath string }
	if err := json.Unmarshal(data, &input); err != nil { return err }
	material, err := os.ReadFile(input.MaterialPath)
	if err != nil { return err }
	if string(material) != "crypto-material" { return errors.New("wrong material") }
	source, err := os.ReadFile(filepath.Join(input.SourceDir, "payload.txt"))
	if err != nil { return err }
	if err := os.WriteFile(filepath.Join(input.BinDir, "payload.bin"), source, 0600); err != nil { return err }
	return os.WriteFile(filepath.Join(input.BinDir, "result.json"), []byte("{\"primary\":\"payload.bin\",\"displayName\":\"payload.bin\"}"), 0600)
}
`

func TestMageCoreAdapterPreservesLiteralSourceAndMaterial(t *testing.T) {
	goCmd, err := exec.LookPath("go")
	if err != nil {
		t.Skip("host Go compiler unavailable")
	}
	root := t.TempDir()
	installedMage := filepath.Join(root, "installed", "magefiles")
	if err := os.MkdirAll(installedMage, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(installedMage, "magefile.go"), []byte(fixtureMagefile), 0600); err != nil {
		t.Fatal(err)
	}
	digest, err := magebuild.DigestBundle(installedMage)
	if err != nil {
		t.Fatal(err)
	}
	artifact := buildplan.Artifact{Kind: "shellcode", Container: "raw", Target: buildplan.Target{OS: "linux", Arch: "amd64"}}
	plan := buildplan.Plan{
		Request:       buildplan.Request{Name: "example", Target: artifact.Target},
		Steps:         []buildplan.Step{{ID: "core", Kind: "core", Provider: "server", Output: artifact, Toolchain: buildplan.Toolchain{Driver: "fixture"}}},
		FinalArtifact: artifact,
	}
	adapter := MageCoreAdapter{
		ID: "fixture-mage-v1", AppDir: root,
		BindingSpec:  MageBinding{SourceMode: "copy", Magefile: "magefiles", Target: "Build"},
		Source:       fstest.MapFS{"payload.txt": &fstest.MapFile{Data: []byte("literal {{ .Value }}")}},
		Magefiles:    fstest.MapFS{"magefiles/magefile.go": &fstest.MapFile{Data: []byte(fixtureMagefile)}},
		BundleDigest: digest, GoCmd: goCmd,
		Available: func(target buildplan.Target, toolchain buildplan.Toolchain) bool {
			return target == artifact.Target && toolchain.Driver == "fixture"
		},
		Prepare: func(context.Context, buildplan.Plan) ([]byte, error) {
			return []byte("crypto-material"), nil
		},
		Manifest: func(_ context.Context, _ buildplan.Plan, work *workspace.Workspace, materialPath string) ([]byte, error) {
			return json.Marshal(struct {
				SourceDir, BinDir, MaterialPath string
			}{work.SourceDir, work.BinDir, materialPath})
		},
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	result, err := adapter.Build(ctx, plan)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(result.Data, []byte("literal {{ .Value }}")) || result.FileName != "payload.bin" || result.BuildID == "" {
		t.Fatalf("literal source or artifact was changed: %+v", result)
	}
	builds, err := filepath.Glob(filepath.Join(root, "slivers", "linux", "amd64", "example", "*"))
	if err != nil || len(builds) != 0 {
		t.Fatalf("private source/material workspace remained after acceptance: %v, %v", builds, err)
	}
	checkMageCoreBinding(t, adapter, artifact.Target)
}

func checkMageCoreBinding(t *testing.T, adapter MageCoreAdapter, target buildplan.Target) {
	t.Helper()
	if got := adapter.Binding(); got.Magefile != "magefiles" || got.SourceMode != "copy" {
		t.Fatalf("wrong installed Mage binding: %+v", got)
	}
	if !adapter.Supports(target, buildplan.Toolchain{Driver: "fixture"}) ||
		adapter.Supports(target, buildplan.Toolchain{Driver: "other"}) {
		t.Fatal("Mage core advertised an unavailable toolchain")
	}
}

func TestMageCoreAdapterRejectsUnboundOrTransformedBuild(t *testing.T) {
	adapter := MageCoreAdapter{ID: "fixture", AppDir: t.TempDir(), BindingSpec: MageBinding{
		SourceMode: "none", Magefile: "../outside", Target: "Build",
	}, Magefiles: fstest.MapFS{"magefiles/magefile.go": &fstest.MapFile{Data: []byte(fixtureMagefile)}},
		BundleDigest: "sha256:" + strings.Repeat("0", 64),
		Manifest: func(context.Context, buildplan.Plan, *workspace.Workspace, string) ([]byte, error) {
			return []byte(`{}`), nil
		},
	}
	if err := adapter.validate(); err == nil || !strings.Contains(err.Error(), "incomplete") {
		t.Fatalf("unsafe Magefile path was accepted: %v", err)
	}
	adapter.BindingSpec.Magefile = "magefiles"
	adapter.Available = func(buildplan.Target, buildplan.Toolchain) bool { return true }
	if err := adapter.validate(); err != nil {
		t.Fatalf("safe Magefile path was rejected: %v", err)
	}
	artifact := buildplan.Artifact{Kind: "shellcode", Container: "raw", Target: buildplan.Target{OS: "linux", Arch: "amd64"}}
	plan := buildplan.Plan{
		Request:       buildplan.Request{Target: artifact.Target},
		Steps:         []buildplan.Step{{Kind: "core", Provider: "server", Output: artifact}, {Kind: "transform"}},
		FinalArtifact: artifact,
	}
	if _, err := adapter.Build(context.Background(), plan); err == nil || !strings.Contains(err.Error(), "one direct") {
		t.Fatalf("unimplemented transform path was accepted: %v", err)
	}
}
