package builder

import (
	"context"
	"strings"
	"testing"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/server/assets"
	"github.com/bishopfox/sliver/server/buildcatalog"
	"github.com/bishopfox/sliver/server/buildcore"
	"github.com/bishopfox/sliver/server/buildplan"
	"google.golang.org/protobuf/proto"
)

func TestSplitAssignedEvent(t *testing.T) {
	name, assignment, err := splitAssignedEvent([]byte("builder:with:colon:step-123"))
	if err != nil || name != "builder:with:colon" || assignment != "step-123" {
		t.Fatalf("split assigned event: %q %q %v", name, assignment, err)
	}
	for _, value := range []string{"", "missing-separator", ":step", "builder:", "builder:step:bad/path", "builder:step\n"} {
		if _, _, err := splitAssignedEvent([]byte(value)); err == nil {
			t.Errorf("accepted malformed event %q", value)
		}
	}
}

func TestM7AssignedAlternateCoreRejectsForeignMaterialAndBinding(t *testing.T) {
	t.Setenv("SLIVER_ROOT_DIR", t.TempDir())
	catalog, err := buildcatalog.Open(assets.GetRootAppDir(), buildcore.InstalledExecutorIDs())
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err := catalog.Get(buildcatalog.Reference{Name: "fixture/minimal-sliver-v1"})
	if err != nil || snapshot.Core == nil {
		t.Fatalf("installed alternate core spec is missing: %v", err)
	}
	target := buildplan.Target{OS: "linux", Arch: "arm64"}
	var artifact buildplan.Artifact
	var toolchain buildplan.Toolchain
	for _, capability := range snapshot.Core.Capabilities {
		for _, candidate := range capability.Targets {
			if candidate.OS == target.OS && candidate.Arch == target.Arch {
				artifact = buildplan.Artifact{Kind: capability.Output.Artifact.Kind,
					Container: capability.Output.Artifact.Container, GoBuildMode: capability.Output.Artifact.GoBuildMode,
					EntrySymbols: capability.Output.Artifact.EntrySymbols, Target: target}
				toolchain = buildplan.Toolchain{Driver: capability.Toolchain.Driver, Requirements: capability.Toolchain.Requirements}
				break
			}
		}
	}
	if artifact.Kind != "executable" || toolchain.Driver == "" {
		t.Fatal("alternate core lacks its declared Linux/arm64 executable tuple")
	}
	ref := &clientpb.BuildSpecRef{Name: snapshot.Reference.Name, Version: snapshot.Reference.Version, Digest: snapshot.Reference.Digest}
	wiredArtifact := artifactToWire(artifact)
	workerName := "m7-negative-worker"
	b := &Builder{externalBuilder: &clientpb.Builder{Name: workerName, Capabilities: []*clientpb.BuilderCapability{{
		Spec: ref, Capability: &clientpb.BuildSpecCapability{
			Targets: []*clientpb.BuildTarget{{GOOS: target.OS, GOARCH: target.Arch}},
			Mode:    "session", Protocols: []string{"mtls"}, Output: wiredArtifact,
			ToolchainDriver: toolchain.Driver, ToolchainRequirements: toolchain.Requirements,
		},
	}}}}
	base := &clientpb.BuilderStepRequest{
		AssignmentID: "m7-negative-assignment", BuildID: "m7-negative-build",
		Step: &clientpb.BuildPlanStep{ID: "core", Kind: "core", Spec: ref, Provider: workerName,
			Output: wiredArtifact, Toolchain: toolchain.Driver, ToolchainRequirements: toolchain.Requirements},
		Request: &clientpb.BuildRequest{Name: "m7-negative", Target: &clientpb.BuildTarget{GOOS: target.OS, GOARCH: target.Arch},
			Mode: "session", C2: []*clientpb.BuildEndpoint{{Protocol: "mtls", Address: "mtls://127.0.0.1:8888"}}},
		CoreMaterial: []byte("opaque-test-material"),
	}
	for _, test := range []struct {
		name    string
		change  func(*clientpb.BuilderStepRequest)
		wantErr string
	}{
		{"Sliver config on alternate core", func(r *clientpb.BuilderStepRequest) {
			r.SliverConfig = &clientpb.ExternalImplantConfig{}
		}, "core has no valid prepared remote material"},
		{"wrong immutable spec", func(r *clientpb.BuilderStepRequest) {
			r.Step.Spec.Digest = "sha256:" + strings.Repeat("0", 64)
		}, "outside the registered capability tuple"},
		{"oversized material", func(r *clientpb.BuilderStepRequest) {
			r.CoreMaterial = make([]byte, maxAssignedCoreMaterialBytes+1)
			r.CoreMaterial[0], r.CoreMaterial[len(r.CoreMaterial)-1] = 1, 1
		}, "core has no valid prepared remote material"},
	} {
		t.Run(test.name, func(t *testing.T) {
			request := proto.Clone(base).(*clientpb.BuilderStepRequest)
			test.change(request)
			if _, _, err := b.executeAssignedStep(context.Background(), request.AssignmentID, request); err == nil || !strings.Contains(err.Error(), test.wantErr) {
				t.Fatalf("unsafe alternate core assignment returned wrong error: %v, want %q", err, test.wantErr)
			}
			for _, value := range request.CoreMaterial {
				if value != 0 {
					t.Fatal("rejected assigned core material was not cleared")
				}
			}
		})
	}
	assertAlternateCoreBinding(t, snapshot.Core)
}

func assertAlternateCoreBinding(t *testing.T, core *buildcatalog.CoreSpec) {
	t.Helper()
	adapter, installed := buildcore.DefaultAdapter(core.Executor.ID)
	if !installed || !matchesBinding(core, adapter) {
		t.Fatal("declared alternate core binding is unavailable")
	}
	altered := *core
	binding := *altered.Builder
	binding.Magefile = "magefiles/other"
	altered.Builder = &binding
	if matchesBinding(&altered, adapter) {
		t.Fatal("alternate Magefile binding change was accepted")
	}
}

func TestAssignedTransformValidatesBuilderSpecTargetAndTypedInput(t *testing.T) {
	t.Setenv("SLIVER_ROOT_DIR", t.TempDir())
	config := &clientpb.Builder{Name: "worker-a"}
	if err := ConfigureInstalledCapabilities(config); err != nil {
		t.Fatal(err)
	}
	b := &Builder{externalBuilder: config}
	ref := &clientpb.BuildSpecRef{
		Name: "builtin/xor", Version: "1.0.0",
		Digest: "sha256:40caa733df52db78843b43abd479e5e818fd61257557f0bc0c67bd4307f72268",
	}
	artifact := &clientpb.BuildArtifactType{Kind: "shellcode", Container: "raw", GOOS: "windows", GOARCH: "amd64"}
	input := []byte{0x90, 0x90, 0xc3}
	request := &clientpb.BuilderStepRequest{
		AssignmentID: "assignment-1",
		BuildID:      "build-1",
		Request:      &clientpb.BuildRequest{Target: &clientpb.BuildTarget{GOOS: "windows", GOARCH: "amd64"}},
		Step: &clientpb.BuildPlanStep{
			ID: "xor", Kind: "transform", Spec: ref, Provider: config.Name,
			Toolchain: "builtin", Input: artifact, Output: artifact,
		},
		Input: &clientpb.BuildArtifactPayload{
			Type: artifact, FileName: "input.bin", Data: input, SHA256: payloadSHA256(input),
		},
	}
	output, _, err := b.executeAssignedStep(context.Background(), request.AssignmentID, request)
	if err != nil {
		t.Fatalf("valid assigned transform: %v", err)
	}
	if output.Type.Kind != "shellcode" || output.Type.GOOS != "windows" || output.SHA256 != payloadSHA256(output.Data) {
		t.Fatalf("wrong typed transform output: %+v", output)
	}
	for _, test := range []struct {
		name   string
		change func(*clientpb.BuilderStepRequest)
	}{
		{"wrong builder", func(r *clientpb.BuilderStepRequest) { r.Step.Provider = "worker-b" }},
		{"wrong spec", func(r *clientpb.BuilderStepRequest) {
			r.Step.Spec.Digest = "sha256:" + strings.Repeat("0", 64)
		}},
		{"wrong target", func(r *clientpb.BuilderStepRequest) {
			r.Request = &clientpb.BuildRequest{Target: &clientpb.BuildTarget{GOOS: "linux", GOARCH: "amd64"}}
		}},
		{"wrong artifact", func(r *clientpb.BuilderStepRequest) {
			r.Input.Type = &clientpb.BuildArtifactType{Kind: "stage", Container: "raw", GOOS: "windows", GOARCH: "amd64"}
		}},
		{"wrong checksum", func(r *clientpb.BuilderStepRequest) { r.Input.SHA256 = strings.Repeat("0", 64) }},
	} {
		t.Run(test.name, func(t *testing.T) {
			bad := proto.CloneOf(request)
			test.change(bad)
			if _, _, err := b.executeAssignedStep(context.Background(), bad.AssignmentID, bad); err == nil {
				t.Fatal("accepted invalid assigned step")
			}
		})
	}
}
