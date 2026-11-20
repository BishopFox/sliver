package buildplan

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func artifactInfo(artifact Artifact, name string, data []byte) ArtifactInfo {
	hash := sha256.Sum256(data)
	return ArtifactInfo{Type: artifact, Name: name, Size: int64(len(data)), SHA256: hex.EncodeToString(hash[:])}
}

func TestStoreFinalBytesAndRevision(t *testing.T) {
	root := filepath.Join(t.TempDir(), "records")
	store, err := OpenStore(root)
	if err != nil {
		t.Fatal(err)
	}
	plan, err := builtinPlanner(t, acceptToolchain).Plan(context.Background(), baseRequest("windows", "amd64", "exe"))
	if err != nil {
		t.Fatal(err)
	}
	record, err := store.CreateWithID("legacy-build", plan)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := store.FinalBytes(record.ID); !errors.Is(err, ErrArtifactNotReady) {
		t.Fatalf("expected artifact-not-ready, got %v", err)
	}
	if _, err := store.Complete(record.ID, artifactInfo(plan.FinalArtifact, "sliver.exe", []byte("first")), []byte("first")); err == nil {
		t.Fatal("completed a record with pending steps")
	}
	if _, err := store.UpdateStep(record.ID, 0, StepUpdate{State: StepRunning, Progress: 25, ToolchainIdentity: "go-test"}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.UpdateStep(record.ID, 0, StepUpdate{State: StepComplete, Progress: 100}); err != nil {
		t.Fatal(err)
	}
	first := []byte("first")
	if _, err := store.Complete(record.ID, artifactInfo(plan.FinalArtifact, "sliver.exe", first), first); err != nil {
		t.Fatal(err)
	}
	// Reopening proves the record and final bytes survive process state.
	store, err = OpenStore(root)
	if err != nil {
		t.Fatal(err)
	}
	got, info, err := store.FinalBytes(record.ID)
	if err != nil || !bytes.Equal(got, first) || info.Name != "sliver.exe" {
		t.Fatalf("reopened final artifact: %q %+v %v", got, info, err)
	}
	second := []byte("spoofed-final")
	updated, err := store.ReplaceFinal(record.ID, artifactInfo(plan.FinalArtifact, "renamed.exe", second), second)
	if err != nil {
		t.Fatal(err)
	}
	if updated.FinalRevision != 2 || len(updated.FinalHistory) != 2 || updated.Plan.SHA256 != plan.SHA256 ||
		updated.Steps[0].ToolchainIdentity != "go-test" {
		t.Fatalf("revision damaged immutable record data: %+v", updated)
	}
	got, info, err = store.FinalBytes(record.ID)
	if err != nil || !bytes.Equal(got, second) || info.Name != "renamed.exe" {
		t.Fatalf("replaced final artifact: %q %+v %v", got, info, err)
	}
	prior, err := os.ReadFile(store.finalPath(record.ID, 1))
	if err != nil || !bytes.Equal(prior, first) {
		t.Fatalf("prior final revision was lost: %q %v", prior, err)
	}
	if _, err := store.ReplaceFinal(record.ID, artifactInfo(plan.FinalArtifact, "bad.exe", []byte("different")), second); err == nil {
		t.Fatal("accepted incorrect replacement checksum")
	}
	if err := os.WriteFile(store.finalPath(record.ID, 2), []byte("tampered"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := store.FinalBytes(record.ID); err == nil {
		t.Fatal("returned tampered final artifact")
	}
}
