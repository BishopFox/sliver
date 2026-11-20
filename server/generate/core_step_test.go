package generate

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/server/buildplan"
)

func TestSliverCoreAdapterCoreStepRejectsBeforeCredentialGeneration(t *testing.T) {
	adapter := SliverCoreAdapter{}
	target := buildplan.Target{OS: WINDOWS, Arch: "amd64"}
	plan := buildplan.Plan{Request: buildplan.Request{Name: "core-step-test", Target: target},
		Steps: []buildplan.Step{{Kind: "core", Provider: "server", Spec: buildplan.Reference{Name: "builtin/sliver"},
			Output: buildplan.Artifact{Kind: "executable", Container: "pe", Target: target, GoBuildMode: "default"}}}}
	config := &clientpb.ImplantConfig{GOOS: WINDOWS, GOARCH: "amd64", Format: clientpb.OutputFormat_EXECUTABLE}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := adapter.BuildCoreWithConfig(ctx, plan, config); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled build should stop before preparing credentials: %v", err)
	}
	if _, err := adapter.BuildCoreWithConfig(nil, plan, config); err == nil {
		t.Fatal("nil context should be rejected")
	}
	if _, err := adapter.BuildCoreWithConfig(context.Background(), plan, nil); err == nil {
		t.Fatal("nil config should be rejected")
	}
	plan.Steps[0].Output.Target.Arch = "386"
	if _, err := adapter.BuildCoreWithConfig(context.Background(), plan, config); err == nil {
		t.Fatal("config target must match the frozen core target")
	}
}

func TestSliverCoreAdapterCoreStepArtifactShapes(t *testing.T) {
	for _, test := range []struct {
		name     string
		artifact buildplan.Artifact
		valid    bool
	}{
		{"windows-pie", buildplan.Artifact{Kind: "executable", Container: "pe", GoBuildMode: "pie", Target: buildplan.Target{OS: WINDOWS, Arch: "amd64"}}, true},
		{"windows-no-default", buildplan.Artifact{Kind: "executable", Container: "pe", GoBuildMode: "default", Target: buildplan.Target{OS: WINDOWS, Arch: "amd64"}}, false},
		{"linux-shared", buildplan.Artifact{Kind: "shared-library", Container: "elf", GoBuildMode: "c-shared", Target: buildplan.Target{OS: LINUX, Arch: "arm64"}, EntrySymbols: []string{"StartW"}}, true},
		{"linux-no-export", buildplan.Artifact{Kind: "shared-library", Container: "elf", GoBuildMode: "c-shared", Target: buildplan.Target{OS: LINUX, Arch: "arm64"}}, false},
		{"darwin-shared", buildplan.Artifact{Kind: "shared-library", Container: "macho", GoBuildMode: "c-shared", Target: buildplan.Target{OS: DARWIN, Arch: "arm64"}, EntrySymbols: []string{"StartW"}}, true},
		{"darwin-unsupported", buildplan.Artifact{Kind: "shared-library", Container: "macho", GoBuildMode: "c-shared", Target: buildplan.Target{OS: DARWIN, Arch: "amd64"}, EntrySymbols: []string{"StartW"}}, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			if err := validateShellcodeCoreArtifact(test.artifact); (err == nil) != test.valid {
				t.Fatalf("artifact validation: valid=%t err=%v", test.valid, err)
			}
		})
	}
}

func TestSliverCoreAdapterCoreStepStageType(t *testing.T) {
	plan := buildplan.Plan{
		Steps: []buildplan.Step{
			{Kind: "core"},
			{Kind: "transform", Spec: buildplan.Reference{Name: "builtin/stage-frame"}},
			{Kind: "transform", Spec: buildplan.Reference{Name: "builtin/stage-compress"}, Options: []buildplan.Option{{Name: "algorithm", ValueJSON: json.RawMessage(`"gzip"`)}}},
			{Kind: "transform", Spec: buildplan.Reference{Name: "builtin/stage-aes"}},
		},
		FinalArtifact: buildplan.Artifact{Kind: "stage"},
	}
	if label, err := sliverStageType(plan); err != nil || label != "gzip - AES - " {
		t.Fatalf("legacy stage label: label=%q err=%v", label, err)
	}
	plan.Steps[3].Spec.Name = "builtin/stage-rc4"
	if label, err := sliverStageType(plan); err != nil || label != "gzip - RC4 - " {
		t.Fatalf("legacy RC4 stage label: label=%q err=%v", label, err)
	}
	plan.FinalArtifact.Kind = "shellcode"
	if _, err := sliverStageType(plan); err == nil {
		t.Fatal("non-stage output must not receive a stage label")
	}
}
