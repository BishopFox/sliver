//go:build sliver_generate

package rpc

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"runtime"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/server/buildcatalog"
	"github.com/bishopfox/sliver/server/builder"
	"github.com/bishopfox/sliver/server/buildplan"
	"github.com/bishopfox/sliver/server/buildscheduler"
	"github.com/bishopfox/sliver/server/db"
)

// TestM7RemoteMinimalCoreLiveWorker runs the installed alternate core through
// an authenticated assigned-step worker, then executes the returned binary
// against the ordinary Sliver mTLS listener and Ping RPC.

//nolint:gocyclo // Worker startup, assignment, and artifact checks are one integration scenario.
func TestM7RemoteMinimalCoreLiveWorker(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" ||
		runtime.GOOS == "darwin" && runtime.GOARCH != "arm64" ||
		runtime.GOOS == "linux" && runtime.GOARCH != "amd64" && runtime.GOARCH != "arm64" {
		t.Skip("minimal core fixture does not support this native target")
	}
	setupGenerateSpoofTest(t)
	port := startM7MinimalCoreLoopbackMTLS(t)
	rpc := &Server{}
	workerName := fmt.Sprintf("m7-minimal-worker-%d", time.Now().UnixNano())
	config := &clientpb.Builder{Name: workerName}
	if err := builder.ConfigureInstalledCapabilities(config); err != nil {
		t.Fatalf("configure installed builder: %v", err)
	}
	config.Status.AvailableMemoryBytes = 4_000_000_000
	client := &workerStepClient{
		server: rpc, operator: "m7-minimal-worker-operator", registered: make(chan error, 1),
		coreFetched: make(chan coreAssignmentObservation, 1),
	}
	worker := builder.NewBuilder(config, &sync.Mutex{}, client, nil)
	done := make(chan struct{})
	go func() {
		worker.Start()
		close(done)
	}()
	t.Cleanup(func() {
		worker.Stop()
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Error("minimal core worker did not stop")
		}
		select {
		case err := <-client.registered:
			if err != nil {
				t.Errorf("minimal core registration ended: %v", err)
			}
		case <-time.After(5 * time.Second):
			t.Error("minimal core registration did not stop")
		}
	})

	catalog, err := rpc.buildCatalog()
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err := catalog.Get(buildcatalog.Reference{Name: "fixture/minimal-sliver-v1"})
	if err != nil || snapshot.Core == nil {
		t.Fatalf("installed minimal core spec is missing: %v", err)
	}
	target := buildplan.Target{OS: runtime.GOOS, Arch: runtime.GOARCH}
	var artifact buildplan.Artifact
	var toolchain buildplan.Toolchain
	for _, capability := range snapshot.Core.Capabilities {
		if capability.Mode != "session" || !slices.Contains(capability.Protocols, "mtls") {
			continue
		}
		for _, candidate := range capability.Targets {
			if candidate.OS == target.OS && candidate.Arch == target.Arch {
				artifact = buildplan.Artifact{Kind: capability.Output.Artifact.Kind,
					Container: capability.Output.Artifact.Container, GoBuildMode: capability.Output.Artifact.GoBuildMode,
					EntrySymbols: slices.Clone(capability.Output.Artifact.EntrySymbols), Target: target}
				toolchain = buildplan.Toolchain{Driver: capability.Toolchain.Driver,
					Requirements: slices.Clone(capability.Toolchain.Requirements)}
				break
			}
		}
	}
	if artifact.Kind != "executable" || toolchain.Driver == "" {
		t.Fatalf("minimal core does not advertise native executable for %s/%s", target.OS, target.Arch)
	}
	ref := buildplan.Reference{Name: snapshot.Reference.Name, Version: snapshot.Reference.Version, Digest: snapshot.Reference.Digest}
	requirement := buildscheduler.Requirement{Spec: ref, Target: target, Output: artifact,
		Mode: "session", Protocols: []string{"mtls"}, Toolchain: buildscheduler.Toolchain{
			Driver: toolchain.Driver, Requirements: slices.Clone(toolchain.Requirements)}}
	scheduler, err := rpc.stepScheduler()
	if err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(10 * time.Second)
	for {
		if candidates := scheduler.Candidates(requirement, workerName, time.Now()); len(candidates) == 1 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("installed worker did not advertise the exact minimal core tuple: %+v",
				scheduler.Candidates(requirement, workerName, time.Now()))
		}
		time.Sleep(10 * time.Millisecond)
	}

	name := fmt.Sprintf("m7-minimal-remote-%d", time.Now().UnixNano())
	t.Cleanup(func() { _ = RemoveBuildByName(name) })
	ctx, cancel := context.WithTimeout(t.Context(), 20*time.Minute)
	defer cancel()
	record, err := rpc.ExecuteBuild(ctx, &clientpb.BuildRequest{
		Name: name, Target: &clientpb.BuildTarget{GOOS: target.OS, GOARCH: target.Arch},
		Mode: "session", Format: "exe", Core: &clientpb.BuildSpecRef{Name: ref.Name, Version: ref.Version, Digest: ref.Digest},
		C2:       []*clientpb.BuildEndpoint{{Protocol: "mtls", Address: fmt.Sprintf("mtls://127.0.0.1:%d", port)}},
		Provider: &clientpb.BuildProviderPolicy{Mode: "pinned", Builder: workerName},
	})
	if err != nil {
		t.Fatalf("execute remotely assigned minimal core: %v", err)
	}
	if record.State != clientpb.BuildRecordState_BUILD_RECORD_SUCCEEDED || record.Plan == nil || record.FinalArtifact == nil ||
		len(record.Plan.Steps) != 1 || record.Plan.Steps[0].Provider != workerName ||
		len(record.Steps) != 1 || record.Steps[0].Selection == nil ||
		record.Steps[0].Selection.Builder != workerName {
		t.Fatalf("minimal core was not built by the selected worker: %+v", record)
	}
	select {
	case observed := <-client.coreFetched:
		if observed.specName != ref.Name || observed.materialBytes == 0 ||
			observed.materialBytes > maxRemoteCoreMaterialBytes || observed.hasSliver {
			t.Fatalf("unsafe remote core assignment metadata: %+v", observed)
		}
	case <-ctx.Done():
		t.Fatal("worker did not fetch the alternate core assignment")
	}
	artifactResult, err := rpc.GetBuildArtifact(ctx, &clientpb.BuildRecordRef{ID: record.ID})
	if err != nil || artifactResult.File == nil || artifactResult.Metadata == nil || len(artifactResult.File.Data) == 0 {
		t.Fatalf("remote minimal core artifact is missing: %+v, %v", artifactResult, err)
	}
	hash := sha256.Sum256(artifactResult.File.Data)
	if want := hex.EncodeToString(hash[:]); artifactResult.Metadata.SHA256 != want || record.FinalArtifact.SHA256 != want {
		t.Fatalf("remote artifact checksum differs from record: got %s, want %s", record.FinalArtifact.SHA256, want)
	}
	build, err := db.ImplantBuildByName(name)
	if err != nil || build.PeerPrivateKey == "" || build.ID != record.ID {
		t.Fatalf("remote core did not register server-side runtime authentication: %+v, %v", build, err)
	}
	assertM7MinimalCoreLivePing(t, name, artifactResult.File.Name, artifactResult.File.Data, artifactResult.Metadata.SHA256)
}
