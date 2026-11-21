package rpc

import (
	"bytes"
	"strings"
	"testing"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/server/buildcore"
	"github.com/bishopfox/sliver/server/buildplan"
)

func TestLegacyBuildRequestCapturesFormatAndProtocols(t *testing.T) {
	config := &clientpb.ImplantConfig{
		GOOS: "windows", GOARCH: "amd64", IsBeacon: true,
		Format:                 clientpb.OutputFormat_SERVICE,
		C2:                     []*clientpb.ImplantC2{{URL: "mtls://example.test:8888"}},
		IncludeHTTP:            true,
		TrafficEncodersEnabled: true,
		Assets:                 []*commonpb.File{{Name: "encoder.wasm"}},
	}
	request, err := legacyBuildRequest("example", config)
	if err != nil {
		t.Fatal(err)
	}
	if request.Core.Name != "builtin/sliver" || request.Mode != "beacon" || request.FinalFormat != "service" {
		t.Fatalf("wrong Sliver route: %+v", request)
	}
	if len(request.C2) != 1 || request.C2[0].Protocol != "mtls" {
		t.Fatalf("wrong endpoint: %+v", request.C2)
	}
	if len(request.IncludeProtocols) != 2 || request.IncludeProtocols[0] != "http" || request.IncludeProtocols[1] != "https" {
		t.Fatalf("wrong included protocols: %+v", request.IncludeProtocols)
	}
	foundEncoder := false
	for _, option := range request.CoreOptions {
		if option.Name == "trafficEncoders" {
			foundEncoder = string(option.ValueJSON) == `["encoder.wasm"]`
		}
	}
	if !foundEncoder {
		t.Fatalf("traffic encoder selection missing from frozen request: %+v", request.CoreOptions)
	}
}

func TestLegacyBuildRequestRejectsUnknownFormat(t *testing.T) {
	_, err := legacyBuildRequest("example", &clientpb.ImplantConfig{Format: clientpb.OutputFormat_THIRD_PARTY})
	if err == nil {
		t.Fatal("unknown format must not fall back to executable")
	}
}

func TestRecordCoreBuildPreservesFinalBytesAndRevision(t *testing.T) {
	t.Setenv("SLIVER_ROOT_DIR", t.TempDir())
	rpc := &Server{}
	artifact := buildplan.Artifact{
		Kind: "executable", Container: "pe", Target: buildplan.Target{OS: "windows", Arch: "amd64"}, GoBuildMode: "default",
	}
	plan := buildplan.Plan{
		Steps:         []buildplan.Step{{ID: "core", Kind: "core", Provider: "server", Output: artifact}},
		FinalArtifact: artifact,
	}
	initial := []byte("original output")
	store, err := rpc.buildRecordStore()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := buildcore.CompleteRecord(store, plan, buildcore.Result{
		BuildID: "test-record", FileName: "example.exe", Data: initial, ToolchainIdentity: "test compiler",
	}); err != nil {
		t.Fatal(err)
	}
	got, metadata, err := store.FinalBytes("test-record")
	if err != nil || !bytes.Equal(got, initial) || metadata.Name != "example.exe" {
		t.Fatalf("recorded output mismatch: %v, %+v", err, metadata)
	}
	updated := []byte("spoofed output")
	if err := rpc.replaceRecordedFinal("test-record", updated); err != nil {
		t.Fatal(err)
	}
	got, _, err = store.FinalBytes("test-record")
	if err != nil || !bytes.Equal(got, updated) {
		t.Fatalf("recorded revision mismatch: %v", err)
	}
	record, err := store.Get("test-record")
	if err != nil || record.FinalRevision != 2 || len(record.FinalHistory) != 2 {
		t.Fatalf("final revision trail missing: %v, %+v", err, record)
	}
}

func TestBuildRecordRPCExposesFailure(t *testing.T) {
	t.Setenv("SLIVER_ROOT_DIR", t.TempDir())
	rpc := &Server{}
	store, err := rpc.buildRecordStore()
	if err != nil {
		t.Fatal(err)
	}
	plan := buildplan.Plan{
		Steps: []buildplan.Step{{ID: "core", Kind: "core"}},
	}
	record, err := store.Create(plan)
	if err != nil {
		t.Fatal(err)
	}
	_, err = store.UpdateStep(record.ID, 0, buildplan.StepUpdate{State: buildplan.StepFailed, Failure: "compiler exited with status 1"})
	if err != nil {
		t.Fatal(err)
	}
	got, err := rpc.GetBuildRecord(t.Context(), &clientpb.BuildRecordRef{ID: record.ID})
	if err != nil {
		t.Fatal(err)
	}
	if got.State != clientpb.BuildRecordState_BUILD_RECORD_FAILED || got.Failure == nil ||
		!strings.Contains(got.Failure.Message, "compiler exited") || got.Steps[0].State != clientpb.BuildStepState_BUILD_STEP_FAILED {
		t.Fatalf("failed build details missing: %+v", got)
	}
}
