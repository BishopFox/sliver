package rpc

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/server/buildcatalog"
	"github.com/bishopfox/sliver/server/buildplan"
	"github.com/bishopfox/sliver/server/buildscheduler"
	"github.com/bishopfox/sliver/server/core"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"gopkg.in/yaml.v3"
)

func TestBuilderRegisterRejectsStepClaimsBeforeCatalogPublish(t *testing.T) {
	for _, invalid := range []string{"short token", "wrong capability"} {
		t.Run(invalid, func(t *testing.T) {
			t.Setenv("SLIVER_ROOT_DIR", t.TempDir())
			rpc := &Server{}
			catalog, err := rpc.buildCatalog()
			if err != nil {
				t.Fatal(err)
			}
			builtin, err := catalog.Get(buildcatalog.Reference{Name: "fixture/minimal-sliver-v1"})
			if err != nil {
				t.Fatal(err)
			}
			name := "test/rejected-step"
			raw := bytes.Replace(builtin.RawYAML, []byte("name: fixture/minimal-sliver-v1"), []byte("name: "+name), 1)
			var document map[string]any
			if err := yaml.Unmarshal(raw, &document); err != nil {
				t.Fatal(err)
			}
			metadata := document["metadata"].(map[string]any)
			oldDigest := metadata["digest"].(string)
			delete(metadata, "digest")
			canonical, err := json.Marshal(document)
			if err != nil {
				t.Fatal(err)
			}
			digest := sha256.Sum256(canonical)
			raw = bytes.Replace(raw, []byte(oldDigest), []byte("sha256:"+hex.EncodeToString(digest[:])), 1)
			ref := &clientpb.BuildSpecRef{Name: name, Version: builtin.Reference.Version,
				Digest: "sha256:" + hex.EncodeToString(digest[:])}
			capability := proto.Clone(buildSpecProto(builtin).Capabilities[0]).(*clientpb.BuildSpecCapability)
			capability.Targets = []*clientpb.BuildTarget{{GOOS: "darwin", GOARCH: "arm64"}}
			worker := &clientpb.Builder{
				Name: "rejected-step-builder", SpecYAML: []string{string(raw)}, SpecRefs: []*clientpb.BuildSpecRef{ref},
				SessionToken: make([]byte, 32),
				Capabilities: []*clientpb.BuilderCapability{{Spec: ref, Capability: capability}},
				Status: &clientpb.BuilderStatus{EffectiveCPUCores: 4, AvailableMemoryBytes: 100_000_000,
					Toolchains: []*clientpb.BuilderToolchainStatus{{Driver: "go", Target: &clientpb.BuildTarget{GOOS: "darwin", GOARCH: "arm64"}, Requirements: []string{"go"}, Identity: "test-go", Available: true}}},
			}
			switch invalid {
			case "short token":
				worker.SessionToken = []byte("short")
			case "wrong capability":
				worker.Capabilities[0].Capability.ToolchainDriver = "wrong-driver"
			}
			stream := &stepRegisterStream{ctx: contextWithCommonName("builder-operator"), events: make(chan *clientpb.Event, 1)}
			if err := rpc.BuilderRegister(worker, stream); status.Code(err) != codes.InvalidArgument {
				t.Fatalf("invalid registration returned %v, want InvalidArgument", err)
			}
			if core.GetBuilder(worker.Name) != nil {
				t.Fatal("rejected builder remained connected")
			}
			if _, err := catalog.Get(buildcatalog.Reference{Name: name}); !errors.Is(err, buildcatalog.ErrNotFound) {
				t.Fatalf("rejected registration published a spec: %v", err)
			}
			if _, err := catalog.ImportAdministrator(raw); err != nil {
				t.Fatalf("rejected builder reserved the admin spec name: %v", err)
			}
		})
	}
}

func registerTestStepBuilder(t *testing.T, rpc *Server, name string, memory uint64) *clientpb.Builder {
	t.Helper()
	catalog, err := rpc.buildCatalog()
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err := catalog.Get(buildcatalog.Reference{Name: "builtin/xor"})
	if err != nil {
		t.Fatal(err)
	}
	ref := &clientpb.BuildSpecRef{Name: snapshot.Reference.Name, Version: snapshot.Reference.Version, Digest: snapshot.Reference.Digest}
	capability := proto.Clone(buildSpecProto(snapshot).Capabilities[0]).(*clientpb.BuildSpecCapability)
	capability.Targets = []*clientpb.BuildTarget{{GOOS: "windows", GOARCH: "amd64"}}
	token := sha256.Sum256([]byte(name))
	worker := &clientpb.Builder{
		Name: name, OperatorName: "step-operator", SessionToken: token[:],
		SpecYAML: []string{string(snapshot.RawYAML)}, SpecRefs: []*clientpb.BuildSpecRef{ref},
		Capabilities: []*clientpb.BuilderCapability{{Spec: ref, Capability: capability}},
		Status: &clientpb.BuilderStatus{
			EffectiveCPUCores: 4, AvailableMemoryBytes: memory,
			Toolchains: []*clientpb.BuilderToolchainStatus{{
				Driver: "builtin", Target: &clientpb.BuildTarget{GOOS: "windows", GOARCH: "amd64"},
				Identity: "builtin", Available: true,
			}},
		},
	}
	if err := core.AddBuilder(worker); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		rpc.disconnectStepBuilder(name)
		core.RemoveBuilder(name)
		_ = catalog.SetRemoteOnline(name, false, time.Now().UTC())
	})
	if err := rpc.registerStepBuilderBeforeCatalog(worker, catalog, [][]byte{snapshot.RawYAML}); err != nil {
		t.Fatalf("register exact step builder: %v", err)
	}
	return worker
}

type stepRegisterStream struct {
	ctx    context.Context
	events chan *clientpb.Event
}

func (stream *stepRegisterStream) Send(event *clientpb.Event) error {
	select {
	case stream.events <- event:
		return nil
	case <-stream.ctx.Done():
		return stream.ctx.Err()
	}
}
func (*stepRegisterStream) SetHeader(metadata.MD) error     { return nil }
func (*stepRegisterStream) SendHeader(metadata.MD) error    { return nil }
func (*stepRegisterStream) SetTrailer(metadata.MD)          {}
func (stream *stepRegisterStream) Context() context.Context { return stream.ctx }
func (stream *stepRegisterStream) SendMsg(value any) error {
	event, ok := value.(*clientpb.Event)
	if !ok {
		return status.Error(codes.Internal, "unexpected builder stream message")
	}
	return stream.Send(event)
}
func (stream *stepRegisterStream) RecvMsg(any) error {
	<-stream.ctx.Done()
	return stream.ctx.Err()
}

// Two live registration streams exercise authenticated assignment delivery,
// capacity debit between concurrent steps, typed fetch/upload, and teardown.
func TestTwoBuilderAssignedStepStreams(t *testing.T) {
	t.Setenv("SLIVER_ROOT_DIR", t.TempDir())
	rpc := &Server{}
	catalog, err := rpc.buildCatalog()
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err := catalog.Get(buildcatalog.Reference{Name: "builtin/xor"})
	if err != nil {
		t.Fatal(err)
	}
	ref := &clientpb.BuildSpecRef{Name: snapshot.Reference.Name, Version: snapshot.Reference.Version, Digest: snapshot.Reference.Digest}
	cap := proto.Clone(buildSpecProto(snapshot).Capabilities[0]).(*clientpb.BuildSpecCapability)
	cap.Targets = []*clientpb.BuildTarget{{GOOS: "windows", GOARCH: "amd64"}}
	makeWorker := func(name string, memory uint64) *clientpb.Builder {
		token := sha256.Sum256([]byte(name))
		return &clientpb.Builder{
			Name: name, SessionToken: token[:],
			SpecYAML: []string{string(snapshot.RawYAML)}, SpecRefs: []*clientpb.BuildSpecRef{ref},
			Capabilities: []*clientpb.BuilderCapability{{Spec: ref, Capability: cap}},
			Status: &clientpb.BuilderStatus{EffectiveCPUCores: 4, AvailableMemoryBytes: memory,
				Toolchains: []*clientpb.BuilderToolchainStatus{{Driver: "builtin", Target: &clientpb.BuildTarget{GOOS: "windows", GOARCH: "amd64"}, Identity: "builtin", Available: true}}},
		}
	}
	first, second := makeWorker("worker-first", 120_000_000), makeWorker("worker-second", 100_000_000)
	ctxA, cancelA := context.WithCancel(contextWithCommonName("stream-operator"))
	ctxB, cancelB := context.WithCancel(contextWithCommonName("stream-operator"))
	defer cancelA()
	defer cancelB()
	streamA := &stepRegisterStream{ctx: ctxA, events: make(chan *clientpb.Event, 8)}
	streamB := &stepRegisterStream{ctx: ctxB, events: make(chan *clientpb.Event, 8)}
	registerDone := make(chan error, 2)
	go func() { registerDone <- rpc.BuilderRegister(first, streamA) }()
	go func() { registerDone <- rpc.BuilderRegister(second, streamB) }()
	for _, stream := range []*stepRegisterStream{streamA, streamB} {
		select {
		case event := <-stream.events:
			if event.EventType != "builder-ready" {
				t.Fatalf("stream was not ready: %s", event.EventType)
			}
		case <-time.After(3 * time.Second):
			t.Fatal("builder registration stream did not become ready")
		}
	}
	target := buildplan.Target{OS: "windows", Arch: "amd64"}
	artifact := buildplan.Artifact{Kind: "shellcode", Container: "raw", Target: target}
	requirement := buildscheduler.Requirement{Spec: buildplan.Reference{Name: ref.Name, Version: ref.Version, Digest: ref.Digest},
		Target: target, Input: &artifact, Output: artifact, Toolchain: buildscheduler.Toolchain{Driver: "builtin"}}
	scheduler, err := rpc.stepScheduler()
	if err != nil {
		t.Fatal(err)
	}
	if candidates := scheduler.Candidates(requirement, "", time.Now().UTC()); len(candidates) != 2 || candidates[0].Name != first.Name {
		t.Fatalf("initial memory ordering: %+v", candidates)
	}
	plan := func(provider string) buildplan.Plan {
		return buildplan.Plan{Request: buildplan.Request{Name: "example", Target: target}, Steps: []buildplan.Step{{
			ID: "transform-1", Kind: "transform", Spec: requirement.Spec, Provider: provider,
			Input: &artifact, Output: artifact, Toolchain: buildplan.Toolchain{Driver: "builtin"},
		}}}
	}
	inputData := []byte{0x90, 0xc3}
	inputHash := sha256.Sum256(inputData)
	input := &clientpb.BuildArtifactPayload{Type: buildArtifactProto(artifact), FileName: "example.bin", Data: inputData, SHA256: hex.EncodeToString(inputHash[:])}
	type completed struct {
		result    assignedStepResult
		selection *buildplan.StepSelection
		err       error
	}
	firstDone := make(chan completed, 1)
	secondDone := make(chan completed, 1)
	firstPlan := plan(first.Name)
	secondPlan := plan(first.Name) // Both previewed A before its capacity was reserved.
	buildCtx, cancelBuild := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelBuild()
	go func() {
		result, selection, err := rpc.dispatchRemoteStep(buildCtx, &firstPlan, 0, "build-a", input, nil, nil)
		firstDone <- completed{result, selection, err}
	}()
	assignedA := waitStepEvent(t, streamA)
	if candidates := scheduler.Candidates(requirement, "", time.Now().UTC()); len(candidates) != 1 || candidates[0].Name != second.Name {
		t.Fatalf("reservation did not move next selection to second builder: %+v", candidates)
	}
	go func() {
		result, selection, err := rpc.dispatchRemoteStep(buildCtx, &secondPlan, 0, "build-b", input, nil, nil)
		secondDone <- completed{result, selection, err}
	}()
	assignedB := waitStepEvent(t, streamB)
	for _, item := range []struct {
		worker     *clientpb.Builder
		assignment string
	}{{first, assignedA}, {second, assignedB}} {
		fetched, err := rpc.BuilderGetStep(contextWithCommonName("stream-operator"), &clientpb.BuilderStepRef{
			BuilderName: item.worker.Name, SessionToken: item.worker.SessionToken, AssignmentID: item.assignment,
		})
		if err != nil || fetched.Step.Provider != item.worker.Name || fetched.Input.SHA256 != input.SHA256 {
			t.Fatalf("typed assigned request was not delivered: %+v, %v", fetched, err)
		}
		_, err = rpc.BuilderSubmitStep(contextWithCommonName("stream-operator"), &clientpb.BuilderStepResult{
			BuilderName: item.worker.Name, SessionToken: item.worker.SessionToken, AssignmentID: item.assignment,
			Spec: ref, Target: &clientpb.BuildTarget{GOOS: "windows", GOARCH: "amd64"}, ToolchainIdentity: "builtin",
			Output: &clientpb.BuildArtifactPayload{Type: buildArtifactProto(artifact), FileName: "example.bin", Data: inputData, SHA256: input.SHA256},
		})
		if err != nil {
			t.Fatalf("authenticated typed upload failed: %v", err)
		}
	}
	for index, done := range []chan completed{firstDone, secondDone} {
		select {
		case result := <-done:
			wantPeak := first.Status.AvailableMemoryBytes
			wantCandidates := []string{first.Name, second.Name}
			if index == 1 {
				wantPeak = second.Status.AvailableMemoryBytes
				wantCandidates = []string{second.Name}
			}
			if result.err != nil || result.selection == nil || result.selection.Builder == "" || result.selection.RequiredPeakMemoryBytes != wantPeak {
				t.Fatalf("wrong completed remote step or capacity decision: %+v", result)
			}
			if len(result.selection.Candidates) != len(wantCandidates) {
				t.Fatalf("capacity decision omitted ordered candidates: %+v", result.selection)
			}
			for candidateIndex, name := range wantCandidates {
				candidate := result.selection.Candidates[candidateIndex]
				if candidate.Name != name || candidate.AvailableMemoryBytes == 0 || candidate.ReportedAt.IsZero() || candidate.ToolchainIdentity != "builtin" {
					t.Fatalf("wrong candidate %d in capacity decision: %+v", candidateIndex, candidate)
				}
			}
			wire := buildRecordProto(buildplan.Record{
				Plan:  plan(result.selection.Builder),
				Steps: []buildplan.StepRecord{{ID: "transform-1", Selection: result.selection}},
			})
			if len(wire.Steps) != 1 || wire.Steps[0].Selection == nil || len(wire.Steps[0].Selection.Candidates) != len(wantCandidates) {
				t.Fatalf("build record omitted capacity candidates: %+v", wire.Steps)
			}
			for candidateIndex, name := range wantCandidates {
				if wire.Steps[0].Selection.Candidates[candidateIndex].Name != name {
					t.Fatalf("build record changed candidate order: %+v", wire.Steps[0].Selection.Candidates)
				}
			}
		case <-time.After(5 * time.Second):
			t.Fatal("remote assignment did not finish")
		}
	}
	if secondPlan.Steps[0].Provider != second.Name || secondPlan.SHA256 == "" {
		t.Fatalf("concurrent dispatch did not freeze the newly selected builder: %+v", secondPlan)
	}
	cancelA()
	cancelB()
	for range 2 {
		select {
		case err := <-registerDone:
			if err != nil {
				t.Fatalf("builder stream teardown: %v", err)
			}
		case <-time.After(3 * time.Second):
			t.Fatal("builder stream did not stop")
		}
	}
	if candidates := scheduler.Candidates(requirement, "", time.Now().UTC()); len(candidates) != 0 {
		t.Fatalf("disconnected builders remained schedulable: %+v", candidates)
	}
}

func waitStepEvent(t *testing.T, stream *stepRegisterStream) string {
	t.Helper()
	select {
	case event := <-stream.events:
		if event.EventType != "build-step-assigned" {
			t.Fatalf("unexpected builder event %s", event.EventType)
		}
		parts := bytes.Split(event.Data, []byte(":"))
		return string(parts[len(parts)-1])
	case <-time.After(5 * time.Second):
		t.Fatal("builder did not receive assigned step")
		return ""
	}
}

func TestStepBuilderRegistrationAndInventoryKeepTokenPrivate(t *testing.T) {
	t.Setenv("SLIVER_ROOT_DIR", t.TempDir())
	rpc := &Server{}
	worker := registerTestStepBuilder(t, rpc, "worker-a", 2_500_000_000)
	listed, err := rpc.Builders(context.Background(), &commonpb.Empty{})
	if err != nil {
		t.Fatal(err)
	}
	if len(listed.Builders) != 1 || listed.Builders[0].Status == nil || !listed.Builders[0].Status.Available ||
		listed.Builders[0].Status.EffectiveCPUCores != 4 || listed.Builders[0].Status.ReportedUnix == 0 ||
		len(listed.Builders[0].SpecRefs) != 1 || len(listed.Builders[0].SessionToken) != 0 {
		t.Fatalf("builder inventory omitted status or exposed token: %+v", listed.Builders)
	}
	if !bytes.Equal(core.GetBuilder(worker.Name).SessionToken, worker.SessionToken) {
		t.Fatal("inventory mutated the registered builder session token")
	}
	bad := &clientpb.BuilderStatusReport{Name: worker.Name, SessionToken: bytes.Repeat([]byte("x"), 32), EffectiveCPUCores: 8, AvailableMemoryBytes: 9_000_000_000}
	if _, err := rpc.BuilderUpdateStatus(contextWithCommonName("step-operator"), bad); status.Code(err) != codes.PermissionDenied {
		t.Fatalf("wrong session token updated capacity: %v", err)
	}
	good := proto.Clone(bad).(*clientpb.BuilderStatusReport)
	good.SessionToken = worker.SessionToken
	good.Toolchains = worker.Status.Toolchains
	if _, err := rpc.BuilderUpdateStatus(contextWithCommonName("step-operator"), good); err != nil {
		t.Fatalf("authenticated capacity update: %v", err)
	}
	listed, err = rpc.Builders(context.Background(), &commonpb.Empty{})
	if err != nil || listed.Builders[0].Status.AvailableMemoryBytes != 9_000_000_000 || listed.Builders[0].Status.EffectiveCPUCores != 8 {
		t.Fatalf("dynamic capacity was not refreshed: %+v, %v", listed, err)
	}
	if len(listed.Builders[0].Capabilities) != 1 || listed.Builders[0].Capabilities[0].Spec.Digest != worker.SpecRefs[0].Digest {
		t.Fatal("status refresh changed immutable advertised support")
	}
	rpc.disconnectStepBuilder(worker.Name)
	core.RemoveBuilder(worker.Name)
	listed, err = rpc.Builders(context.Background(), &commonpb.Empty{})
	if err != nil || len(listed.Builders) != 1 || listed.Builders[0].Status.Online || listed.Builders[0].Status.Available ||
		len(listed.Builders[0].SpecRefs) != 1 || listed.Builders[0].SpecRefs[0].Digest != worker.SpecRefs[0].Digest {
		t.Fatalf("offline builder lost immutable spec support: %+v, %v", listed, err)
	}
}

func TestAssignedStepRejectsForeignAndMismatchedResults(t *testing.T) {
	t.Setenv("SLIVER_ROOT_DIR", t.TempDir())
	rpc := &Server{}
	owner := registerTestStepBuilder(t, rpc, "worker-a", 2_000_000_000)
	foreign := registerTestStepBuilder(t, rpc, "worker-b", 2_000_000_000)
	artifact := &clientpb.BuildArtifactType{Kind: "shellcode", Container: "raw", GOOS: "windows", GOARCH: "amd64"}
	ref := owner.SpecRefs[0]
	request := &clientpb.BuilderStepRequest{
		AssignmentID: "test-assignment", BuildID: "test-build",
		Request: &clientpb.BuildRequest{Target: &clientpb.BuildTarget{GOOS: "windows", GOARCH: "amd64"}},
		Step: &clientpb.BuildPlanStep{ID: "transform-1", Kind: "transform", Spec: ref, Provider: owner.Name,
			Toolchain: "builtin", Input: artifact, Output: artifact},
	}
	assigned := &assignedStep{builder: owner.Name, identity: "builtin", request: request, result: make(chan assignedStepResult, 1)}
	stepAssignments.Lock()
	stepAssignments.items[request.AssignmentID] = assigned
	stepAssignments.Unlock()
	t.Cleanup(func() {
		stepAssignments.Lock()
		delete(stepAssignments.items, request.AssignmentID)
		stepAssignments.Unlock()
	})
	ctx := contextWithCommonName("step-operator")
	if _, err := rpc.BuilderGetStep(ctx, &clientpb.BuilderStepRef{BuilderName: foreign.Name, SessionToken: foreign.SessionToken, AssignmentID: request.AssignmentID}); status.Code(err) != codes.PermissionDenied {
		t.Fatalf("foreign builder fetched step: %v", err)
	}
	if _, err := rpc.BuilderGetStep(ctx, &clientpb.BuilderStepRef{BuilderName: owner.Name, SessionToken: foreign.SessionToken, AssignmentID: request.AssignmentID}); status.Code(err) != codes.PermissionDenied {
		t.Fatalf("wrong token fetched step: %v", err)
	}
	if fetched, err := rpc.BuilderGetStep(ctx, &clientpb.BuilderStepRef{BuilderName: owner.Name, SessionToken: owner.SessionToken, AssignmentID: request.AssignmentID}); err != nil || fetched.Step.Spec.Digest != ref.Digest {
		t.Fatalf("assigned builder could not fetch step: %+v, %v", fetched, err)
	}
	data := []byte{0x90, 0x90, 0xc3}
	hash := sha256.Sum256(data)
	base := &clientpb.BuilderStepResult{
		BuilderName: owner.Name, SessionToken: owner.SessionToken, AssignmentID: request.AssignmentID,
		Spec: ref, Target: request.Request.Target, ToolchainIdentity: "builtin",
		Output: &clientpb.BuildArtifactPayload{Type: artifact, FileName: "output.bin", Data: data, SHA256: hex.EncodeToString(hash[:])},
	}
	for _, test := range []struct {
		name   string
		mutate func(*clientpb.BuilderStepResult)
	}{
		{"foreign builder", func(value *clientpb.BuilderStepResult) {
			value.BuilderName, value.SessionToken = foreign.Name, foreign.SessionToken
		}},
		{"wrong spec", func(value *clientpb.BuilderStepResult) { value.Spec.Digest = "sha256:" + strings.Repeat("0", 64) }},
		{"wrong target", func(value *clientpb.BuilderStepResult) { value.Target.GOOS = "linux" }},
		{"wrong artifact", func(value *clientpb.BuilderStepResult) { value.Output.Type.Kind = "stage" }},
		{"wrong checksum", func(value *clientpb.BuilderStepResult) { value.Output.SHA256 = strings.Repeat("0", 64) }},
		{"wrong toolchain", func(value *clientpb.BuilderStepResult) { value.ToolchainIdentity = "other" }},
	} {
		t.Run(test.name, func(t *testing.T) {
			value := proto.Clone(base).(*clientpb.BuilderStepResult)
			test.mutate(value)
			if _, err := rpc.BuilderSubmitStep(ctx, value); err == nil {
				t.Fatal("accepted mismatched assigned step result")
			}
			if assigned.done {
				t.Fatal("mismatched result completed assignment")
			}
		})
	}
	if _, err := rpc.BuilderSubmitStep(ctx, base); err != nil {
		t.Fatalf("valid result was rejected: %v", err)
	}
	select {
	case result := <-assigned.result:
		if result.err != nil || !bytes.Equal(result.data, data) || result.info.SHA256 != base.Output.SHA256 {
			t.Fatalf("wrong accepted result: %+v", result)
		}
	case <-time.After(time.Second):
		t.Fatal("accepted result did not complete assignment")
	}
	if _, err := rpc.BuilderSubmitStep(ctx, base); status.Code(err) != codes.PermissionDenied {
		t.Fatalf("duplicate result completed assignment twice: %v", err)
	}
}
