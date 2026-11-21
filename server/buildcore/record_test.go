package buildcore

import (
	"bytes"
	"testing"

	"github.com/bishopfox/sliver/server/buildplan"
)

func TestCompleteRecordFreezesPlanAndBytes(t *testing.T) {
	store, err := buildplan.OpenStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	artifact := buildplan.Artifact{Kind: "executable", Container: "pe", Target: buildplan.Target{OS: "windows", Arch: "amd64"}, GoBuildMode: "default"}
	plan := buildplan.Plan{Steps: []buildplan.Step{{ID: "core", Kind: "core", Output: artifact}}, FinalArtifact: artifact}
	result := Result{BuildID: "example-build", FileName: "example.exe", Data: []byte("compiled bytes"), ToolchainIdentity: "go1.27.3"}
	record, err := CompleteRecord(store, plan, result)
	if err != nil {
		t.Fatal(err)
	}
	if record.State != buildplan.BuildComplete || record.Steps[0].ToolchainIdentity != result.ToolchainIdentity || record.Final.SHA256 == "" {
		t.Fatalf("completed record is incomplete: %+v", record)
	}
	result.Data[0] = 'X'
	data, _, err := store.FinalBytes(result.BuildID)
	if err != nil || !bytes.Equal(data, []byte("compiled bytes")) {
		t.Fatalf("final bytes changed after result reuse: %q, %v", data, err)
	}
}
