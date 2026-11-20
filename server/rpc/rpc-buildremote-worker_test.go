package rpc

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/rpcpb"
	"github.com/bishopfox/sliver/server/buildcatalog"
	"github.com/bishopfox/sliver/server/builder"
	"github.com/bishopfox/sliver/server/buildplan"
	"github.com/bishopfox/sliver/server/buildscheduler"
	"google.golang.org/grpc"
)

// workerStepClient exercises the real builder event loop and step executor
// against the server RPC methods. The adapter supplies the same authenticated
// operator identity to each call without opening a network listener.
type workerStepClient struct {
	rpcpb.SliverRPCClient
	server      *Server
	operator    string
	submitGate  <-chan struct{}
	registered  chan error
	stepFetched chan struct{}
}

type workerStepEvents struct {
	grpc.ClientStream
	ctx    context.Context
	events <-chan *clientpb.Event
}

func (stream *workerStepEvents) Recv() (*clientpb.Event, error) {
	select {
	case event := <-stream.events:
		return event, nil
	case <-stream.ctx.Done():
		return nil, io.EOF
	}
}

func (client *workerStepClient) BuilderRegister(ctx context.Context, config *clientpb.Builder,
	_ ...grpc.CallOption) (rpcpb.SliverRPC_BuilderRegisterClient, error) {
	serverCtx, cancel := context.WithCancel(contextWithCommonName(client.operator))
	stream := &stepRegisterStream{ctx: serverCtx, events: make(chan *clientpb.Event, 16)}
	go func() {
		select {
		case <-ctx.Done():
			cancel()
		case <-serverCtx.Done():
		}
	}()
	go func() {
		client.registered <- client.server.BuilderRegister(config, stream)
		cancel()
	}()
	return &workerStepEvents{ctx: serverCtx, events: stream.events}, nil
}

func (client *workerStepClient) BuilderGetStep(_ context.Context, ref *clientpb.BuilderStepRef,
	_ ...grpc.CallOption) (*clientpb.BuilderStepRequest, error) {
	request, err := client.server.BuilderGetStep(contextWithCommonName(client.operator), ref)
	if err == nil && client.stepFetched != nil {
		select {
		case client.stepFetched <- struct{}{}:
		default:
		}
	}
	return request, err
}

func (client *workerStepClient) BuilderSubmitStep(ctx context.Context, result *clientpb.BuilderStepResult,
	_ ...grpc.CallOption) (*commonpb.Empty, error) {
	if client.submitGate != nil {
		select {
		case <-client.submitGate:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	return client.server.BuilderSubmitStep(contextWithCommonName(client.operator), result)
}

func (client *workerStepClient) BuilderUpdateStatus(_ context.Context, report *clientpb.BuilderStatusReport,
	_ ...grpc.CallOption) (*commonpb.Empty, error) {
	return client.server.BuilderUpdateStatus(contextWithCommonName(client.operator), report)
}

// This test starts two actual builder workers. Their registration streams,
// authenticated fetches, builtin XOR execution, and result uploads all run
// through the assigned-step protocol. Holding A's upload proves B can be
// selected while A's unknown-peak reservation is exclusive.
func TestTwoBuilderLiveWorkersExecuteAssignedTransforms(t *testing.T) {
	t.Setenv("SLIVER_ROOT_DIR", t.TempDir())
	rpc := &Server{}
	gate := make(chan struct{})
	releaseFirst := sync.OnceFunc(func() { close(gate) })
	defer releaseFirst()
	makeWorker := func(name string, memory uint64, submitGate <-chan struct{}) (*builder.Builder, *workerStepClient, chan struct{}) {
		t.Helper()
		config := &clientpb.Builder{Name: name}
		if err := builder.ConfigureInstalledCapabilities(config); err != nil {
			t.Fatal(err)
		}
		config.Status.AvailableMemoryBytes = memory
		client := &workerStepClient{
			server: rpc, operator: "live-worker-operator", submitGate: submitGate,
			registered: make(chan error, 1), stepFetched: make(chan struct{}, 1),
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
				t.Errorf("worker %s did not stop", name)
			}
			select {
			case err := <-client.registered:
				if err != nil {
					t.Errorf("worker %s registration ended: %v", name, err)
				}
			case <-time.After(5 * time.Second):
				t.Errorf("worker %s registration did not stop", name)
			}
		})
		return worker, client, done
	}
	_, firstClient, _ := makeWorker("live-worker-first", 120_000_000, gate)
	_, secondClient, _ := makeWorker("live-worker-second", 100_000_000, nil)

	catalog, err := rpc.buildCatalog()
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err := catalog.Get(buildcatalog.Reference{Name: "builtin/xor"})
	if err != nil {
		t.Fatal(err)
	}
	ref := buildplan.Reference{Name: snapshot.Reference.Name, Version: snapshot.Reference.Version, Digest: snapshot.Reference.Digest}
	target := buildplan.Target{OS: "windows", Arch: "amd64"}
	artifact := buildplan.Artifact{Kind: "shellcode", Container: "raw", Target: target}
	requirement := buildscheduler.Requirement{
		Spec: ref, Target: target, Input: &artifact, Output: artifact,
		Toolchain: buildscheduler.Toolchain{Driver: "builtin"},
	}
	scheduler, err := rpc.stepScheduler()
	if err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(10 * time.Second)
	for {
		candidates := scheduler.Candidates(requirement, "", time.Now().UTC())
		if len(candidates) == 2 {
			if candidates[0].Name != "live-worker-first" {
				t.Fatalf("wrong initial worker ranking: %+v", candidates)
			}
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("live workers did not register capability: %+v", candidates)
		}
		time.Sleep(10 * time.Millisecond)
	}
	plan := func() buildplan.Plan {
		return buildplan.Plan{Request: buildplan.Request{Name: "live", Target: target}, Steps: []buildplan.Step{{
			ID: "xor", Kind: "transform", Spec: ref, Provider: "live-worker-first",
			Input: &artifact, Output: artifact, Toolchain: buildplan.Toolchain{Driver: "builtin"},
		}}}
	}
	input := []byte{0x90, 0x90, 0xc3}
	sha := sha256.Sum256(input)
	payload := &clientpb.BuildArtifactPayload{
		Type: buildArtifactProto(artifact), FileName: "input.bin", Data: input, SHA256: hex.EncodeToString(sha[:]),
	}
	type outcome struct {
		result    assignedStepResult
		selection *buildplan.StepSelection
		err       error
	}
	buildCtx, cancelBuild := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancelBuild()
	firstPlan, secondPlan := plan(), plan()
	firstDone, secondDone := make(chan outcome, 1), make(chan outcome, 1)
	go func() {
		result, selection, err := rpc.dispatchRemoteStep(buildCtx, &firstPlan, 0, "first-build", payload, nil, nil)
		firstDone <- outcome{result, selection, err}
	}()
	select {
	case <-firstClient.stepFetched:
	case <-buildCtx.Done():
		t.Fatal("first live worker did not fetch its assignment")
	}
	if candidates := scheduler.Candidates(requirement, "", time.Now().UTC()); len(candidates) != 1 || candidates[0].Name != "live-worker-second" {
		t.Fatalf("exclusive reservation did not select second worker: %+v", candidates)
	}
	go func() {
		result, selection, err := rpc.dispatchRemoteStep(buildCtx, &secondPlan, 0, "second-build", payload, nil, nil)
		secondDone <- outcome{result, selection, err}
	}()
	select {
	case <-secondClient.stepFetched:
	case <-buildCtx.Done():
		t.Fatal("second live worker did not fetch its assignment")
	}
	releaseFirst()
	for index, done := range []chan outcome{firstDone, secondDone} {
		select {
		case completed := <-done:
			wantBuilder, wantPeak := "live-worker-first", uint64(120_000_000)
			if index == 1 {
				wantBuilder, wantPeak = "live-worker-second", 100_000_000
			}
			if completed.err != nil || completed.selection == nil || completed.selection.Builder != wantBuilder ||
				completed.selection.RequiredPeakMemoryBytes != wantPeak || completed.result.info.SHA256 == "" ||
				bytes.Equal(completed.result.data, input) {
				t.Fatalf("live worker %s did not execute and submit XOR: %+v", wantBuilder, completed)
			}
			if completed.result.info.SHA256 != payloadSHA(completed.result.data) {
				t.Fatalf("live worker %s result checksum differs from bytes", wantBuilder)
			}
		case <-buildCtx.Done():
			t.Fatal("live worker did not complete assigned transform")
		}
	}
	if secondPlan.Steps[0].Provider != "live-worker-second" || secondPlan.SHA256 == "" {
		t.Fatalf("second live worker was not frozen into the plan: %+v", secondPlan)
	}
}

func payloadSHA(data []byte) string {
	value := sha256.Sum256(data)
	return hex.EncodeToString(value[:])
}

// A timed-out builder is quarantined by closing its registration stream.
// The worker reconnects and sends a fresh status report before it can receive
// another assignment.
func TestAssignedStepQuarantineReconnectsWorker(t *testing.T) {
	t.Setenv("SLIVER_ROOT_DIR", t.TempDir())
	rpc := &Server{}
	config := &clientpb.Builder{Name: "reconnecting-worker"}
	if err := builder.ConfigureInstalledCapabilities(config); err != nil {
		t.Fatal(err)
	}
	config.Status.AvailableMemoryBytes = 100_000_000
	client := &workerStepClient{
		server: rpc, operator: "reconnect-operator", registered: make(chan error, 2),
	}
	worker := builder.NewBuilder(config, &sync.Mutex{}, client, nil)
	done := make(chan struct{})
	go func() {
		worker.Start()
		close(done)
	}()
	t.Cleanup(worker.Stop)
	scheduler, err := rpc.stepScheduler()
	if err != nil {
		t.Fatal(err)
	}
	connected := func() bool {
		for _, entry := range scheduler.Inventory(time.Now()) {
			if entry.Name == config.Name && entry.Connected {
				return true
			}
		}
		return false
	}
	deadline := time.Now().Add(8 * time.Second)
	for !connected() && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if !connected() {
		t.Fatal("worker did not register its first step stream")
	}
	quarantineStepBuilder(config.Name)
	select {
	case err := <-client.registered:
		if err == nil {
			t.Fatal("quarantined registration ended without a reconnect signal")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("quarantined registration did not close")
	}
	for !connected() && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if !connected() {
		t.Fatal("worker did not reconnect with a fresh status report")
	}
	worker.Stop()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("reconnected worker did not stop")
	}
	select {
	case err := <-client.registered:
		if err != nil {
			t.Fatalf("reconnected registration did not close cleanly: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("reconnected registration did not close")
	}
}
