package rpc

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/bishopfox/sliver/server/buildplan"
	"github.com/bishopfox/sliver/server/buildtransform"
)

//nolint:gocyclo // Every step's observed artifact and the resulting final bytes come from one execution.
func TestResolvedPipelineRecordsEachObservedArtifactAndExactFinalBytes(t *testing.T) {
	t.Setenv("SLIVER_ROOT_DIR", t.TempDir())
	rpc := &Server{}
	core := buildplan.Artifact{Kind: "executable", Container: "pe", Target: buildplan.Target{OS: "windows", Arch: "amd64"}, GoBuildMode: "default"}
	stage := buildplan.Artifact{Kind: "stage", Container: "raw", Target: core.Target}
	plan := buildplan.Plan{
		Request: buildplan.Request{Name: "example", Target: core.Target, Mode: "session", FinalFormat: "stage"},
		Steps: []buildplan.Step{
			{ID: "core", Kind: "core", Spec: buildplan.Reference{Name: "test/core"}, Provider: "server", Output: core},
			{ID: "transform-1", Kind: "transform", Provider: "server", Spec: buildplan.Reference{
				Name: buildtransform.StageFrameSpec, Version: "1.0.0", Digest: buildtransform.StageFrameDigest,
			}, Input: &core, Output: stage, Toolchain: buildplan.Toolchain{Driver: "builtin"}, Options: []buildplan.Option{{
				Name: "prependSize", ValueJSON: json.RawMessage(`true`),
			}}},
		},
		FinalArtifact: stage,
	}
	result, record, err := rpc.executeResolvedPlan(context.Background(), plan, testRegisteredCoreAdapter{}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if record.State != buildplan.BuildComplete || len(record.Steps) != 2 || record.Final == nil {
		t.Fatalf("incomplete pipeline record: %+v", record)
	}
	if result.BuildID != record.ID || result.FileName != "example.bin" || !bytes.Equal(result.Data[4:], testCorePE()) ||
		binary.LittleEndian.Uint32(result.Data[:4]) != uint32(len(testCorePE())) {
		t.Fatalf("unexpected framed final artifact: %+v", result)
	}
	coreHash := sha256.Sum256(testCorePE())
	finalHash := sha256.Sum256(result.Data)
	if record.Steps[0].Artifact == nil || record.Steps[0].Artifact.SHA256 != hex.EncodeToString(coreHash[:]) ||
		record.Steps[1].Artifact == nil || record.Steps[1].Artifact.SHA256 != hex.EncodeToString(finalHash[:]) ||
		record.Final.SHA256 != hex.EncodeToString(finalHash[:]) {
		t.Fatalf("step/final checksums do not describe observed bytes: %+v", record)
	}
	store, err := rpc.buildRecordStore()
	if err != nil {
		t.Fatal(err)
	}
	stored, _, err := store.FinalBytes(record.ID)
	if err != nil || !bytes.Equal(stored, result.Data) {
		t.Fatalf("stored final differs from returned bytes: %v", err)
	}
	entries, err := os.ReadDir(filepath.Join(os.Getenv("SLIVER_ROOT_DIR"), "build-records", record.ID))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Fatalf("intermediate bytes unexpectedly retained on disk: %+v", entries)
	}
}
