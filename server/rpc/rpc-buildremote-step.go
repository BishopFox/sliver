package rpc

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"slices"
	"sync"
	"time"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/server/buildcore/artifactaccept"
	"github.com/bishopfox/sliver/server/buildplan"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

const remoteStepTimeout = 30 * time.Minute

type assignedStepResult struct {
	data     []byte
	info     buildplan.ArtifactInfo
	identity string
	prepared *clientpb.ExternalImplantConfig
	err      error
}

type assignedStep struct {
	builder  string
	identity string
	request  *clientpb.BuilderStepRequest
	result   chan assignedStepResult
	done     bool
}

type stepSignalStream struct {
	events      chan *clientpb.Event
	stop        chan struct{}
	quarantined bool
}

var stepAssignments = struct {
	sync.Mutex
	items   map[string]*assignedStep
	signals map[string]*stepSignalStream
}{items: make(map[string]*assignedStep), signals: make(map[string]*stepSignalStream)}

const stepSignalQueueSize = 64

// Each assigned-step notification has a private, bounded queue. Unlike the
// shared event broker, a full queue rejects dispatch rather than silently
// dropping an assignment that already holds a capacity reservation.
func registerStepSignals(builder string) error {
	stepAssignments.Lock()
	defer stepAssignments.Unlock()
	if stepAssignments.signals[builder] != nil {
		return errors.New("builder already has an assigned-step stream")
	}
	stepAssignments.signals[builder] = &stepSignalStream{
		events: make(chan *clientpb.Event, stepSignalQueueSize), stop: make(chan struct{}),
	}
	return nil
}

func stepSignals(builder string) *stepSignalStream {
	stepAssignments.Lock()
	defer stepAssignments.Unlock()
	return stepAssignments.signals[builder]
}

func unregisterStepSignals(builder string) {
	stepAssignments.Lock()
	if stream := stepAssignments.signals[builder]; stream != nil && !stream.quarantined {
		stream.quarantined = true
		close(stream.stop)
	}
	delete(stepAssignments.signals, builder)
	stepAssignments.Unlock()
}

// A timed-out worker may still be compiling. Close its registration stream so
// it cannot accept more steps until it reconnects with a fresh status report.
func quarantineStepBuilder(builder string) {
	stepAssignments.Lock()
	if stream := stepAssignments.signals[builder]; stream != nil && !stream.quarantined {
		stream.quarantined = true
		close(stream.stop)
	}
	stepAssignments.Unlock()
}

func newStepAssignmentID() (string, error) {
	var random [24]byte
	if _, err := rand.Read(random[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(random[:]), nil
}

func cancelStepAssignments(builder string) {
	stepAssignments.Lock()
	defer stepAssignments.Unlock()
	if stream := stepAssignments.signals[builder]; stream != nil && !stream.quarantined {
		stream.quarantined = true
		close(stream.stop)
	}
	delete(stepAssignments.signals, builder)
	for _, assignment := range stepAssignments.items {
		if assignment.builder == builder && !assignment.done {
			assignment.done = true
			assignment.result <- assignedStepResult{err: errors.New("assigned builder disconnected")}
		}
	}
}

func (rpc *Server) BuilderGetStep(ctx context.Context, ref *clientpb.BuilderStepRef) (*clientpb.BuilderStepRequest, error) {
	if ref == nil || ref.AssignmentID == "" {
		return nil, status.Error(codes.InvalidArgument, "missing step assignment")
	}
	if err := rpc.authorizeStepBuilder(ctx, ref.BuilderName, ref.SessionToken); err != nil {
		return nil, err
	}
	stepAssignments.Lock()
	assignment := stepAssignments.items[ref.AssignmentID]
	if assignment == nil || assignment.builder != ref.BuilderName || assignment.done {
		stepAssignments.Unlock()
		return nil, status.Error(codes.PermissionDenied, "step is not assigned to this builder")
	}
	response := proto.Clone(assignment.request).(*clientpb.BuilderStepRequest)
	stepAssignments.Unlock()
	return response, nil
}

func (rpc *Server) BuilderSubmitStep(ctx context.Context, submitted *clientpb.BuilderStepResult) (*commonpb.Empty, error) {
	if submitted == nil || submitted.AssignmentID == "" {
		return nil, status.Error(codes.InvalidArgument, "missing step result")
	}
	if err := rpc.authorizeStepBuilder(ctx, submitted.BuilderName, submitted.SessionToken); err != nil {
		return nil, err
	}
	stepAssignments.Lock()
	assignment := stepAssignments.items[submitted.AssignmentID]
	if assignment == nil || assignment.builder != submitted.BuilderName || assignment.done {
		stepAssignments.Unlock()
		return nil, status.Error(codes.PermissionDenied, "step is not assigned to this builder")
	}
	if assignment.request == nil || assignment.request.Step == nil || assignment.request.Request == nil || assignment.request.Request.Target == nil {
		stepAssignments.Unlock()
		return nil, status.Error(codes.Internal, "assigned step is incomplete")
	}
	step := proto.Clone(assignment.request.Step).(*clientpb.BuildPlanStep)
	target := proto.Clone(assignment.request.Request.Target).(*clientpb.BuildTarget)
	identity := assignment.identity
	stepAssignments.Unlock()
	if !proto.Equal(submitted.Spec, step.Spec) || !proto.Equal(submitted.Target, target) {
		return nil, status.Error(codes.InvalidArgument, "step result spec or target differs from assignment")
	}
	if submitted.Failure != nil {
		if submitted.Output != nil || submitted.Failure.Message == "" || submitted.Failure.StepID != step.ID {
			return nil, status.Error(codes.InvalidArgument, "invalid assigned step failure")
		}
		stepAssignments.Lock()
		if stepAssignments.items[submitted.AssignmentID] != assignment || assignment.done {
			stepAssignments.Unlock()
			return nil, status.Error(codes.PermissionDenied, "step assignment is no longer active")
		}
		assignment.done = true
		assignment.result <- assignedStepResult{err: fmt.Errorf("remote step %s: %s", step.ID, submitted.Failure.Message)}
		stepAssignments.Unlock()
		return &commonpb.Empty{}, nil
	}
	output := submitted.Output
	if output == nil || !proto.Equal(output.Type, step.Output) || submitted.ToolchainIdentity != identity {
		return nil, status.Error(codes.InvalidArgument, "step output type or toolchain identity differs from assignment")
	}
	if len(output.Data) == 0 || len(output.Data) > 512<<20 {
		return nil, status.Error(codes.InvalidArgument, "step output exceeds artifact size limit")
	}
	data := bytes.Clone(output.Data)
	info, err := artifactaccept.InspectBytes(data, artifactFromProto(step.Output), output.FileName)
	if err != nil || output.SHA256 != info.SHA256 {
		return nil, status.Error(codes.InvalidArgument, "step output artifact or checksum is invalid")
	}
	stepAssignments.Lock()
	if stepAssignments.items[submitted.AssignmentID] != assignment || assignment.done {
		stepAssignments.Unlock()
		return nil, status.Error(codes.PermissionDenied, "step assignment is no longer active")
	}
	assignment.done = true
	assignment.result <- assignedStepResult{data: data, info: info, identity: submitted.ToolchainIdentity}
	stepAssignments.Unlock()
	return &commonpb.Empty{}, nil
}

// dispatchRemoteStep reserves the reported capacity before publishing a
// private assignment. The worker must authenticate both its operator TLS
// certificate and its connection token to fetch or submit this step.
func (rpc *Server) dispatchRemoteStep(ctx context.Context, plan *buildplan.Plan, index int,
	buildID string, input *clientpb.BuildArtifactPayload, prepared *clientpb.ExternalImplantConfig,
	inputs []*clientpb.BuildInput) (assignedStepResult, *buildplan.StepSelection, error) {
	if plan == nil || index < 0 || index >= len(plan.Steps) {
		return assignedStepResult{}, nil, errors.New("invalid remote step plan or index")
	}
	step := plan.Steps[index]
	scheduler, err := rpc.stepScheduler()
	if err != nil {
		return assignedStepResult{}, nil, err
	}
	pin := ""
	if plan.Request.Provider.Mode == "pinned" {
		pin = plan.Request.Provider.Builder
	}
	reservation, err := scheduler.SelectAndReserve(requirementForStep(*plan, step), pin, rpc.stepPeakBytes(*plan, step), time.Now())
	if err != nil {
		return assignedStepResult{}, nil, &buildplan.PlanError{Code: buildplan.CodeUnavailableProvider, Step: step.ID, Detail: err.Error()}
	}
	var assignmentID string
	var request *clientpb.BuilderStepRequest
	cleanupOnReturn := true
	cleanup := func() {
		if assignmentID != "" {
			stepAssignments.Lock()
			delete(stepAssignments.items, assignmentID)
			stepAssignments.Unlock()
		}
		if request != nil {
			if request.Input != nil {
				clear(request.Input.Data)
			}
			for _, material := range request.Inputs {
				if material != nil {
					clear(material.Data)
				}
			}
		}
		scheduler.Release(reservation.ID)
	}
	defer func() {
		if cleanupOnReturn {
			cleanup()
		}
	}()
	if reservation.Builder != step.Provider {
		plan.Steps[index].Provider = reservation.Builder
		if err := buildplan.Rehash(plan); err != nil {
			return assignedStepResult{}, nil, fmt.Errorf("freeze selected remote provider: %w", err)
		}
		step = plan.Steps[index]
	}
	selection := selectionFromReservation(reservation)
	if step.Kind == "core" && prepared == nil {
		prepared, err = rpc.prepareRemoteCore(*plan)
		if err != nil {
			return assignedStepResult{}, selection, fmt.Errorf("prepare assigned remote core: %w", err)
		}
		buildID = prepared.Build.ID
	}
	assignmentID, err = newStepAssignmentID()
	if err != nil {
		return assignedStepResult{prepared: prepared}, selection, err
	}
	request = &clientpb.BuilderStepRequest{
		AssignmentID: assignmentID, BuildID: buildID,
		Step: buildPlanProto(*plan).Steps[index], Request: buildRequestProto(plan.Request),
	}
	if prepared != nil {
		request.SliverConfig = proto.Clone(prepared).(*clientpb.ExternalImplantConfig)
		if prepared.Build != nil {
			request.Request.Name = prepared.Build.Name
		}
	}
	for _, material := range inputs {
		if material != nil {
			request.Inputs = append(request.Inputs, proto.Clone(material).(*clientpb.BuildInput))
		}
	}
	if input != nil {
		request.Input = proto.Clone(input).(*clientpb.BuildArtifactPayload)
	}
	identity := ""
	for _, candidate := range reservation.Decision.Candidates {
		if candidate.Name == reservation.Builder {
			identity = candidate.ToolchainIdentity
			break
		}
	}
	if identity == "" {
		return assignedStepResult{prepared: prepared}, selection, errors.New("selected builder has no reported toolchain identity")
	}
	assignment := &assignedStep{builder: step.Provider, identity: identity, request: request, result: make(chan assignedStepResult, 1)}
	stepAssignments.Lock()
	signals := stepAssignments.signals[step.Provider]
	if signals == nil || signals.quarantined {
		stepAssignments.Unlock()
		return assignedStepResult{prepared: prepared}, selection, errors.New("selected builder has no active assigned-step stream")
	}
	select {
	case signals.events <- &clientpb.Event{EventType: "build-step-assigned", Data: []byte(step.Provider + ":" + assignmentID)}:
		stepAssignments.items[assignmentID] = assignment
	default:
		stepAssignments.Unlock()
		return assignedStepResult{prepared: prepared}, selection, errors.New("selected builder's assignment queue is full")
	}
	stepAssignments.Unlock()
	deadline := time.NewTimer(remoteStepTimeout)
	defer deadline.Stop()
	select {
	case result := <-assignment.result:
		result.prepared = prepared
		return result, selection, result.err
	case <-ctx.Done():
		// The worker may already be compiling. Keep its capacity committed
		// until its result arrives or the assignment lease expires.
		cleanupOnReturn = false
		go func() {
			lease := time.NewTimer(remoteStepTimeout)
			defer lease.Stop()
			select {
			case <-assignment.result:
			case <-lease.C:
				scheduler.Disconnect(step.Provider)
				quarantineStepBuilder(step.Provider)
			}
			cleanup()
		}()
		return assignedStepResult{prepared: prepared}, selection, ctx.Err()
	case <-deadline.C:
		scheduler.Disconnect(step.Provider)
		quarantineStepBuilder(step.Provider)
		return assignedStepResult{prepared: prepared}, selection, errors.New("assigned remote step timed out")
	}
}

func payloadForArtifact(data []byte, info buildplan.ArtifactInfo) *clientpb.BuildArtifactPayload {
	return &clientpb.BuildArtifactPayload{
		Type: buildArtifactProto(info.Type), FileName: info.Name, Data: slices.Clone(data), SHA256: info.SHA256,
	}
}
