package rpc

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"sync"
	"time"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/server/assets"
	"github.com/bishopfox/sliver/server/buildcatalog"
	"github.com/bishopfox/sliver/server/buildcore"
	"github.com/bishopfox/sliver/server/buildplan"
	"github.com/bishopfox/sliver/server/core"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type buildRecordStoreEntry struct {
	once  sync.Once
	store *buildplan.Store
	err   error
}

var buildRecordStores sync.Map // absolute app root -> *buildRecordStoreEntry
var buildRecordIDPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9-]{0,127}$`)

func (rpc *Server) buildRecordStore() (*buildplan.Store, error) {
	root, err := filepath.Abs(assets.GetRootAppDir())
	if err != nil {
		return nil, err
	}
	value, _ := buildRecordStores.LoadOrStore(root, &buildRecordStoreEntry{})
	entry := value.(*buildRecordStoreEntry)
	entry.once.Do(func() { entry.store, entry.err = buildplan.OpenStore(filepath.Join(root, "build-records")) })
	return entry.store, entry.err
}

// resolveBuildPlan uses the same catalog and installed compiler checks for
// preview, neutral execution, and the legacy Generate compatibility path.
func (rpc *Server) resolveBuildPlan(ctx context.Context, req buildplan.Request) (buildplan.Plan, error) {
	catalog, err := rpc.buildCatalog()
	if err != nil {
		return buildplan.Plan{}, err
	}
	check := func(provider string, spec buildcatalog.Snapshot, target buildplan.Target, toolchain buildplan.Toolchain) bool {
		if provider == "server" {
			if toolchain.Driver == "builtin" {
				return true
			}
			if spec.Core == nil {
				return false
			}
			adapter, ok := buildcore.DefaultAdapter(spec.Core.Executor.ID)
			if !ok {
				return false
			}
			if !matchesInstalledMageBinding(spec.Core, adapter) {
				return false
			}
			availability, ok := adapter.(buildcore.Availability)
			return ok && availability.Supports(target, toolchain)
		}
		return core.GetBuilder(provider) != nil
	}
	planner := buildplan.NewPlanner(catalog, check)
	if scheduler, err := rpc.stepScheduler(); err == nil {
		inventory := scheduler.Inventory(time.Now())
		ranks := make(map[string]struct {
			mb  uint64
			cpu int
		}, len(inventory))
		for _, entry := range inventory {
			ranks[entry.Name] = struct {
				mb  uint64
				cpu int
			}{entry.RoundedAvailableMB, entry.EffectiveCPUCores}
		}
		planner.SetProviderRank(func(name string) (uint64, int) {
			rank := ranks[name]
			return rank.mb, rank.cpu
		})
		planner.SetProviderCapabilityCheck(func(provider string, snapshot buildcatalog.Snapshot, target buildplan.Target,
			input *buildplan.Artifact, output buildplan.Artifact, toolchain buildplan.Toolchain, mode string, protocols []string) bool {
			if provider == "server" {
				return true
			}
			requirement := buildSchedulerRequirement(snapshot, target, input, output, toolchain, mode, protocols)
			kind := "transform"
			if snapshot.Core != nil {
				kind = "core"
			}
			step := buildplan.Step{Kind: kind, Spec: requirement.Spec, Input: input, Output: output, Toolchain: toolchain}
			peak := rpc.stepPeakBytes(buildplan.Plan{Request: buildplan.Request{Mode: mode}}, step)
			return len(scheduler.Eligible(requirement, provider, peak, time.Now())) > 0
		})
	}
	return planner.Plan(ctx, req)
}

func (rpc *Server) PlanBuild(ctx context.Context, req *clientpb.BuildRequest) (*clientpb.BuildPlan, error) {
	neutral, err := buildRequestFromProto(req)
	if err != nil {
		return nil, err
	}
	plan, err := rpc.resolveBuildPlan(ctx, neutral)
	if err != nil {
		return nil, buildPlanRPCError(err)
	}
	return buildPlanProto(plan), nil
}

func buildPlanRPCError(err error) error {
	var planError *buildplan.PlanError
	if errors.As(err, &planError) {
		switch planError.Code {
		case buildplan.CodeInvalidRequest:
			return status.Error(codes.InvalidArgument, planError.Error())
		case buildplan.CodeUnknownSpec:
			return status.Error(codes.NotFound, planError.Error())
		case buildplan.CodeUnavailableProvider:
			return status.Error(codes.Unavailable, planError.Error())
		case buildplan.CodeUnsupportedExecution:
			return status.Error(codes.Unimplemented, planError.Error())
		default:
			return status.Error(codes.FailedPrecondition, planError.Error())
		}
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return status.FromContextError(err).Err()
	}
	return status.Error(codes.Internal, err.Error())
}

func buildRequestFromProto(req *clientpb.BuildRequest) (buildplan.Request, error) {
	if req == nil {
		return buildplan.Request{}, status.Error(codes.InvalidArgument, "missing build request")
	}
	result := buildplan.Request{
		Name: req.Name, Target: targetFromProto(req.Target), Mode: req.Mode,
		IncludeProtocols: append([]string(nil), req.IncludeProtocols...), FinalFormat: req.Format,
		Core: referenceFromProto(req.Core), Provider: providerFromProto(req.Provider),
	}
	if settings := req.C2Settings; settings != nil {
		result.C2Settings = &buildplan.C2Settings{
			ConnectionStrategy: settings.ConnectionStrategy, ReconnectSeconds: settings.ReconnectSeconds,
			PollTimeoutSeconds: settings.PollTimeoutSeconds, MaxConnectionErrors: settings.MaxConnectionErrors,
			HTTPC2Profile: settings.HTTPC2Profile, WireguardKeyExchangePort: settings.WireguardKeyExchangePort,
			WireguardTCPCommsPort: settings.WireguardTCPCommsPort,
		}
	}
	if settings := req.BeaconSettings; settings != nil {
		result.BeaconSettings = &buildplan.BeaconSettings{IntervalSeconds: settings.IntervalSeconds, JitterSeconds: settings.JitterSeconds}
	}
	if req.CoreOutput != nil {
		artifact := artifactFromProto(req.CoreOutput)
		result.CoreOutput = &artifact
	}
	for _, endpoint := range req.C2 {
		if endpoint == nil {
			return buildplan.Request{}, status.Error(codes.InvalidArgument, "nil C2 endpoint")
		}
		result.C2 = append(result.C2, buildplan.Endpoint{Protocol: endpoint.Protocol, Address: endpoint.Address})
	}
	var err error
	result.CoreOptions, err = optionsFromProto(req.CoreOptions)
	if err != nil {
		return buildplan.Request{}, err
	}
	for _, transform := range req.Transforms {
		if transform == nil || transform.Spec == nil {
			return buildplan.Request{}, status.Error(codes.InvalidArgument, "transform spec reference is required")
		}
		options, err := optionsFromProto(transform.Options)
		if err != nil {
			return buildplan.Request{}, err
		}
		result.Transforms = append(result.Transforms, buildplan.TransformRequest{Spec: referenceFromProto(transform.Spec), Options: options})
	}
	return result, nil
}

func optionsFromProto(options []*clientpb.BuildOption) ([]buildplan.Option, error) {
	result := make([]buildplan.Option, 0, len(options))
	for _, option := range options {
		if option == nil || option.Name == "" || !json.Valid(option.ValueJSON) {
			return nil, status.Error(codes.InvalidArgument, "build options require a name and valid JSON value")
		}
		result = append(result, buildplan.Option{Name: option.Name, ValueJSON: append(json.RawMessage(nil), option.ValueJSON...)})
	}
	return result, nil
}

func targetFromProto(target *clientpb.BuildTarget) buildplan.Target {
	if target == nil {
		return buildplan.Target{}
	}
	return buildplan.Target{OS: target.GOOS, Arch: target.GOARCH}
}

func referenceFromProto(ref *clientpb.BuildSpecRef) buildplan.Reference {
	if ref == nil {
		return buildplan.Reference{}
	}
	return buildplan.Reference{Name: ref.Name, Version: ref.Version, Digest: ref.Digest}
}

func artifactFromProto(artifact *clientpb.BuildArtifactType) buildplan.Artifact {
	if artifact == nil {
		return buildplan.Artifact{}
	}
	return buildplan.Artifact{
		Kind: artifact.Kind, Container: artifact.Container,
		Target:      buildplan.Target{OS: artifact.GOOS, Arch: artifact.GOARCH},
		GoBuildMode: artifact.GoBuildMode, EntrySymbols: append([]string(nil), artifact.EntrySymbols...),
	}
}

func providerFromProto(provider *clientpb.BuildProviderPolicy) buildplan.ProviderPolicy {
	if provider == nil {
		return buildplan.ProviderPolicy{}
	}
	return buildplan.ProviderPolicy{Mode: provider.Mode, Builder: provider.Builder}
}

func buildRequestProto(req buildplan.Request) *clientpb.BuildRequest {
	result := &clientpb.BuildRequest{
		Name: req.Name, Target: buildTargetProto(req.Target), Mode: req.Mode,
		IncludeProtocols: append([]string(nil), req.IncludeProtocols...), Format: req.FinalFormat,
		Core: buildReferenceProto(req.Core), Provider: &clientpb.BuildProviderPolicy{Mode: req.Provider.Mode, Builder: req.Provider.Builder},
	}
	if settings := req.C2Settings; settings != nil {
		result.C2Settings = &clientpb.BuildC2Settings{
			ConnectionStrategy: settings.ConnectionStrategy, ReconnectSeconds: settings.ReconnectSeconds,
			PollTimeoutSeconds: settings.PollTimeoutSeconds, MaxConnectionErrors: settings.MaxConnectionErrors,
			HTTPC2Profile: settings.HTTPC2Profile, WireguardKeyExchangePort: settings.WireguardKeyExchangePort,
			WireguardTCPCommsPort: settings.WireguardTCPCommsPort,
		}
	}
	if settings := req.BeaconSettings; settings != nil {
		result.BeaconSettings = &clientpb.BuildBeaconSettings{IntervalSeconds: settings.IntervalSeconds, JitterSeconds: settings.JitterSeconds}
	}
	if req.CoreOutput != nil {
		result.CoreOutput = buildArtifactProto(*req.CoreOutput)
	}
	for _, endpoint := range req.C2 {
		result.C2 = append(result.C2, &clientpb.BuildEndpoint{Protocol: endpoint.Protocol, Address: endpoint.Address})
	}
	result.CoreOptions = buildOptionsProto(req.CoreOptions)
	for _, transform := range req.Transforms {
		result.Transforms = append(result.Transforms, &clientpb.BuildTransformRequest{Spec: buildReferenceProto(transform.Spec), Options: buildOptionsProto(transform.Options)})
	}
	return result
}

func buildOptionsProto(options []buildplan.Option) []*clientpb.BuildOption {
	result := make([]*clientpb.BuildOption, 0, len(options))
	for _, option := range options {
		result = append(result, &clientpb.BuildOption{Name: option.Name, ValueJSON: append([]byte(nil), option.ValueJSON...)})
	}
	return result
}

func buildTargetProto(target buildplan.Target) *clientpb.BuildTarget {
	return &clientpb.BuildTarget{GOOS: target.OS, GOARCH: target.Arch}
}

func buildReferenceProto(ref buildplan.Reference) *clientpb.BuildSpecRef {
	return &clientpb.BuildSpecRef{Name: ref.Name, Version: ref.Version, Digest: ref.Digest}
}

func buildArtifactProto(artifact buildplan.Artifact) *clientpb.BuildArtifactType {
	return &clientpb.BuildArtifactType{
		Kind: artifact.Kind, Container: artifact.Container, GOOS: artifact.Target.OS, GOARCH: artifact.Target.Arch,
		GoBuildMode: artifact.GoBuildMode, EntrySymbols: append([]string(nil), artifact.EntrySymbols...),
	}
}

func buildPlanProto(plan buildplan.Plan) *clientpb.BuildPlan {
	result := &clientpb.BuildPlan{
		Request: buildRequestProto(plan.Request), FinalArtifact: buildArtifactProto(plan.FinalArtifact), PlanSHA256: plan.SHA256,
	}
	for _, step := range plan.Steps {
		entry := &clientpb.BuildPlanStep{
			ID: step.ID, Kind: step.Kind, Spec: buildReferenceProto(step.Spec), Provider: step.Provider,
			Toolchain: step.Toolchain.Driver, ToolchainRequirements: append([]string(nil), step.Toolchain.Requirements...),
			ToolchainIdentity: step.Toolchain.Identity, Output: buildArtifactProto(step.Output), Options: buildOptionsProto(step.Options),
		}
		if step.Input != nil {
			entry.Input = buildArtifactProto(*step.Input)
		}
		result.Steps = append(result.Steps, entry)
	}
	return result
}

func buildArtifactMetadataProto(info buildplan.ArtifactInfo) *clientpb.BuildArtifactMetadata {
	return &clientpb.BuildArtifactMetadata{Type: buildArtifactProto(info.Type), FileName: info.Name, SizeBytes: uint64(info.Size), SHA256: info.SHA256}
}

func buildRecordProto(record buildplan.Record) *clientpb.BuildRecord {
	result := &clientpb.BuildRecord{
		ID: record.ID, Plan: buildPlanProto(record.Plan), CreatedUnix: unixOrZero(record.CreatedAt), UpdatedUnix: unixOrZero(record.UpdatedAt),
		FinalRevision: uint32(record.FinalRevision),
	}
	switch record.State {
	case buildplan.BuildPending:
		result.State = clientpb.BuildRecordState_BUILD_RECORD_PENDING
	case buildplan.BuildRunning:
		result.State = clientpb.BuildRecordState_BUILD_RECORD_RUNNING
	case buildplan.BuildComplete:
		result.State = clientpb.BuildRecordState_BUILD_RECORD_SUCCEEDED
	case buildplan.BuildFailed:
		result.State = clientpb.BuildRecordState_BUILD_RECORD_FAILED
	case buildplan.BuildCancelled:
		result.State = clientpb.BuildRecordState_BUILD_RECORD_CANCELLED
	}
	if record.State == buildplan.BuildComplete || record.State == buildplan.BuildFailed || record.State == buildplan.BuildCancelled {
		for index := len(record.Steps) - 1; index >= 0; index-- {
			if record.Steps[index].FinishedAt != nil {
				result.FinishedUnix = unixOrZero(*record.Steps[index].FinishedAt)
				break
			}
		}
		if result.FinishedUnix == 0 {
			result.FinishedUnix = unixOrZero(record.UpdatedAt)
		}
	}
	if record.Failure != "" {
		result.Failure = &clientpb.BuildFailure{Code: "execution-failed", Message: record.Failure}
	}
	if record.Final != nil {
		result.FinalArtifact = buildArtifactMetadataProto(*record.Final)
	}
	for _, revision := range record.FinalHistory {
		result.FinalHistory = append(result.FinalHistory, &clientpb.BuildArtifactRevision{
			Number: uint32(revision.Number), UpdatedUnix: unixOrZero(revision.UpdatedAt),
			Artifact: buildArtifactMetadataProto(revision.Artifact),
		})
	}
	for _, step := range record.Steps {
		entry := &clientpb.BuildStepProgress{
			StepID: step.ID, ProgressPercent: uint32(step.Progress), ToolchainIdentity: step.ToolchainIdentity,
		}
		if step.Selection != nil {
			entry.Selection = &clientpb.BuildStepSelection{
				Builder: step.Selection.Builder, ReportedUnix: unixOrZero(step.Selection.ReportedAt),
				AvailableMemoryBytes:    step.Selection.AvailableMemoryBytes,
				ReservedMemoryBytes:     step.Selection.ReservedMemoryBytes,
				RoundedAvailableMB:      step.Selection.RoundedAvailableMB,
				EffectiveCPUCores:       step.Selection.EffectiveCPUCores,
				RequiredPeakMemoryBytes: step.Selection.RequiredPeakMemoryBytes,
				Decision:                step.Selection.Decision,
			}
			for _, candidate := range step.Selection.Candidates {
				entry.Selection.Candidates = append(entry.Selection.Candidates, &clientpb.BuildStepCandidate{
					Name: candidate.Name, ReportedUnix: unixOrZero(candidate.ReportedAt),
					AvailableMemoryBytes: candidate.AvailableMemoryBytes, ReservedMemoryBytes: candidate.ReservedMemoryBytes,
					RoundedAvailableMB: candidate.RoundedAvailableMB, EffectiveCPUCores: candidate.EffectiveCPUCores,
					ToolchainIdentity: candidate.ToolchainIdentity,
				})
			}
		}
		switch step.State {
		case buildplan.StepPending:
			entry.State = clientpb.BuildStepState_BUILD_STEP_PENDING
		case buildplan.StepRunning:
			entry.State = clientpb.BuildStepState_BUILD_STEP_RUNNING
		case buildplan.StepComplete:
			entry.State = clientpb.BuildStepState_BUILD_STEP_SUCCEEDED
		case buildplan.StepFailed:
			entry.State = clientpb.BuildStepState_BUILD_STEP_FAILED
		case buildplan.StepCancelled:
			entry.State = clientpb.BuildStepState_BUILD_STEP_CANCELLED
		}
		if step.StartedAt != nil {
			entry.StartedUnix = unixOrZero(*step.StartedAt)
		}
		if step.FinishedAt != nil {
			entry.FinishedUnix = unixOrZero(*step.FinishedAt)
		}
		if step.Failure != "" {
			entry.Failure = &clientpb.BuildFailure{Code: "execution-failed", Message: step.Failure, StepID: step.ID}
		}
		if step.Artifact != nil {
			entry.Artifact = buildArtifactMetadataProto(*step.Artifact)
		}
		result.Steps = append(result.Steps, entry)
	}
	return result
}

func (rpc *Server) GetBuildRecord(_ context.Context, req *clientpb.BuildRecordRef) (*clientpb.BuildRecord, error) {
	if req == nil || !buildRecordIDPattern.MatchString(req.ID) {
		return nil, status.Error(codes.InvalidArgument, "valid build record ID is required")
	}
	store, err := rpc.buildRecordStore()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "open build record store: %v", err)
	}
	record, err := store.Get(req.ID)
	if err != nil {
		return nil, buildRecordLookupError(err)
	}
	return buildRecordProto(record), nil
}

func (rpc *Server) GetBuildArtifact(_ context.Context, req *clientpb.BuildRecordRef) (*clientpb.BuildArtifactResponse, error) {
	if req == nil || !buildRecordIDPattern.MatchString(req.ID) {
		return nil, status.Error(codes.InvalidArgument, "valid build record ID is required")
	}
	store, err := rpc.buildRecordStore()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "open build record store: %v", err)
	}
	data, info, err := store.FinalBytes(req.ID)
	if err != nil {
		return nil, buildRecordLookupError(err)
	}
	return &clientpb.BuildArtifactResponse{
		File: &commonpb.File{Name: info.Name, Data: data}, Metadata: buildArtifactMetadataProto(info),
	}, nil
}

func buildRecordLookupError(err error) error {
	if errors.Is(err, os.ErrNotExist) {
		return status.Error(codes.NotFound, "build record not found")
	}
	if errors.Is(err, buildplan.ErrArtifactNotReady) {
		return status.Error(codes.FailedPrecondition, "build has no completed final artifact")
	}
	return status.Error(codes.Internal, err.Error())
}
