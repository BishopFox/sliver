package rpc

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"
	"time"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/server/buildplan"
	"github.com/bishopfox/sliver/server/core"
)

func signalTestTransform(worker *clientpb.Builder) (buildplan.Plan, *clientpb.BuildArtifactPayload) {
	target := buildplan.Target{OS: "windows", Arch: "amd64"}
	artifact := buildplan.Artifact{Kind: "shellcode", Container: "raw", Target: target}
	ref := worker.SpecRefs[0]
	data := []byte{0x90, 0xc3}
	hash := sha256.Sum256(data)
	return buildplan.Plan{
		Request: buildplan.Request{Name: "signal-test", Target: target,
			Provider: buildplan.ProviderPolicy{Mode: "pinned", Builder: worker.Name}},
		Steps: []buildplan.Step{{
			ID: "transform-1", Kind: "transform", Provider: worker.Name,
			Spec:  buildplan.Reference{Name: ref.Name, Version: ref.Version, Digest: ref.Digest},
			Input: &artifact, Output: artifact, Toolchain: buildplan.Toolchain{Driver: "builtin"},
		}},
	}, &clientpb.BuildArtifactPayload{
		Type: buildArtifactProto(artifact), FileName: "input.bin", Data: data,
		SHA256: hex.EncodeToString(hash[:]),
	}
}

// A saturated subscriber of the shared, lossy event broker cannot consume or
// discard an assigned-step notification from the builder's private queue.
// Disconnecting that builder must then cancel its outstanding assignment.

//nolint:gocyclo // The test checks signal delivery through broker pressure and disconnect in one run.
func TestAssignedStepSignalSurvivesBrokerPressureAndDisconnect(t *testing.T) {
	t.Setenv("SLIVER_ROOT_DIR", t.TempDir())
	rpc := &Server{}
	worker := registerTestStepBuilder(t, rpc, "signal-worker", 100_000_000)
	brokerEvents := core.EventBroker.Subscribe()
	defer core.EventBroker.Unsubscribe(brokerEvents)
	for range cap(brokerEvents) + 12 {
		core.EventBroker.Publish(core.Event{EventType: "unrelated-test-event"})
	}
	deadline := time.Now().Add(2 * time.Second)
	for len(brokerEvents) != cap(brokerEvents) && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if len(brokerEvents) != cap(brokerEvents) {
		t.Fatal("shared broker subscriber was not saturated")
	}
	plan, input := signalTestTransform(worker)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	done := make(chan error, 1)
	go func() {
		_, _, err := rpc.dispatchRemoteStep(ctx, &plan, 0, "signal-build", input, nil, nil)
		done <- err
	}()
	select {
	case event := <-stepSignals(worker.Name).events:
		if event == nil || event.EventType != "build-step-assigned" || !strings.HasPrefix(string(event.Data), worker.Name+":") {
			t.Fatalf("private queue delivered wrong assignment: %+v", event)
		}
	case <-ctx.Done():
		t.Fatal("assigned step was lost under shared broker pressure")
	}
	rpc.disconnectStepBuilder(worker.Name)
	select {
	case err := <-done:
		if err == nil || !strings.Contains(err.Error(), "disconnected") {
			t.Fatalf("disconnect did not cancel assigned step: %v", err)
		}
	case <-ctx.Done():
		t.Fatal("disconnect left assigned step waiting")
	}
	stepAssignments.Lock()
	defer stepAssignments.Unlock()
	for _, assignment := range stepAssignments.items {
		if assignment.builder == worker.Name {
			t.Fatal("disconnected assignment remained in registry")
		}
	}
}

// Full queues fail before a request can wait for a worker that will never see
// the notification, and the atomic capacity reservation is released.
func TestAssignedStepSignalQueueFullReleasesReservation(t *testing.T) {
	t.Setenv("SLIVER_ROOT_DIR", t.TempDir())
	rpc := &Server{}
	worker := registerTestStepBuilder(t, rpc, "full-queue-worker", 100_000_000)
	stepAssignments.Lock()
	signals := stepAssignments.signals[worker.Name]
	if signals == nil {
		stepAssignments.Unlock()
		t.Fatal("registered builder has no assigned-step queue")
	}
	for range cap(signals.events) {
		signals.events <- &clientpb.Event{EventType: "test-queued-step"}
	}
	stepAssignments.Unlock()
	plan, input := signalTestTransform(worker)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	start := time.Now()
	_, _, err := rpc.dispatchRemoteStep(ctx, &plan, 0, "full-queue-build", input, nil, nil)
	if err == nil || !strings.Contains(err.Error(), "queue is full") || time.Since(start) >= 2*time.Second {
		t.Fatalf("full queue did not reject dispatch promptly: %v", err)
	}
	scheduler, err := rpc.stepScheduler()
	if err != nil {
		t.Fatal(err)
	}
	inventory := scheduler.Inventory(time.Now())
	if len(inventory) != 1 || inventory[0].ReservedMemoryBytes != 0 || inventory[0].Exclusive {
		t.Fatalf("queue failure stranded builder capacity: %+v", inventory)
	}
	stepAssignments.Lock()
	defer stepAssignments.Unlock()
	for _, assignment := range stepAssignments.items {
		if assignment.builder == worker.Name {
			t.Fatal("rejected assignment remained in registry")
		}
	}
}
