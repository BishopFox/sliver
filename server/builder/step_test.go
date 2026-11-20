package builder

import (
	"context"
	"strings"
	"testing"

	"github.com/bishopfox/sliver/protobuf/clientpb"
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
	mutate := func(change func(*clientpb.BuilderStepRequest)) *clientpb.BuilderStepRequest {
		copy := *request
		step := *request.Step
		inputPayload := *request.Input
		copy.Step, copy.Input = &step, &inputPayload
		change(&copy)
		return &copy
	}
	for _, test := range []struct {
		name   string
		change func(*clientpb.BuilderStepRequest)
	}{
		{"wrong builder", func(r *clientpb.BuilderStepRequest) { r.Step.Provider = "worker-b" }},
		{"wrong spec", func(r *clientpb.BuilderStepRequest) {
			refCopy := *r.Step.Spec
			refCopy.Digest = "sha256:" + strings.Repeat("0", 64)
			r.Step.Spec = &refCopy
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
			bad := mutate(test.change)
			if _, _, err := b.executeAssignedStep(context.Background(), bad.AssignmentID, bad); err == nil {
				t.Fatal("accepted invalid assigned step")
			}
		})
	}
}
