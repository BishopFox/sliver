package rpc

import (
	"bytes"
	"context"
	"crypto/subtle"
	"path/filepath"
	"slices"
	"sync"
	"time"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/server/assets"
	"github.com/bishopfox/sliver/server/buildcatalog"
	"github.com/bishopfox/sliver/server/buildcore"
	"github.com/bishopfox/sliver/server/buildplan"
	"github.com/bishopfox/sliver/server/buildscheduler"
	"github.com/bishopfox/sliver/server/core"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

var stepSchedulers sync.Map // absolute app root -> *buildscheduler.Scheduler

func (rpc *Server) stepScheduler() (*buildscheduler.Scheduler, error) {
	root, err := filepath.Abs(assets.GetRootAppDir())
	if err != nil {
		return nil, err
	}
	value, _ := stepSchedulers.LoadOrStore(root, buildscheduler.New(buildscheduler.DefaultStatusTTL))
	return value.(*buildscheduler.Scheduler), nil
}

func builderStatusFromProto(cpu uint32, memory uint64, tools []*clientpb.BuilderToolchainStatus) buildscheduler.Status {
	result := buildscheduler.Status{EffectiveCPUCores: int(cpu), AvailableMemoryBytes: memory}
	for _, tool := range tools {
		if tool == nil {
			continue
		}
		result.Toolchains = append(result.Toolchains, buildscheduler.ToolchainStatus{
			Driver: tool.Driver, Target: targetFromProto(tool.Target),
			Requirements: slices.Clone(tool.Requirements), Identity: tool.Identity, Available: tool.Available,
		})
	}
	return result
}

// registerStepBuilderBeforeCatalog checks all authenticated claims without
// publishing new YAML. It makes the builder temporarily unavailable, commits
// the exact definitions, and only then publishes its toolchain status. A
// rejected registration never leaves a new offline catalog definition.
func (rpc *Server) registerStepBuilderBeforeCatalog(req *clientpb.Builder, catalog *buildcatalog.Catalog, raws [][]byte) error {
	preview, err := catalog.PreviewRemoteBatch(req.Name, raws)
	if err != nil {
		return status.Error(codes.InvalidArgument, err.Error())
	}
	registration, builderStatus, required, err := prepareStepRegistration(req, func(ref buildcatalog.Reference) (buildcatalog.Snapshot, error) {
		if candidate, ok := preview[ref]; ok {
			return candidate, nil
		}
		return buildcatalog.Snapshot{}, buildcatalog.ErrNotFound
	})
	if err != nil {
		return err
	}
	if !required {
		return status.Error(codes.InvalidArgument, "assigned-step builder capabilities are required")
	}
	scheduler, err := rpc.stepScheduler()
	if err != nil {
		return status.Errorf(codes.Internal, "open build scheduler: %v", err)
	}
	if err := registerStepSignals(req.Name); err != nil {
		return status.Error(codes.InvalidArgument, err.Error())
	}
	dormant := builderStatus
	dormant.Toolchains = slices.Clone(builderStatus.Toolchains)
	for index := range dormant.Toolchains {
		dormant.Toolchains[index].Available = false
	}
	if err := scheduler.Register(registration, dormant, time.Now()); err != nil {
		unregisterStepSignals(req.Name)
		return status.Error(codes.InvalidArgument, err.Error())
	}
	if err := catalog.AdvertiseRemoteBatch(req.Name, raws, time.Now().UTC()); err != nil {
		rpc.disconnectStepBuilder(req.Name)
		return status.Error(codes.InvalidArgument, err.Error())
	}
	if err := scheduler.UpdateStatus(req.Name, builderStatus, time.Now()); err != nil {
		rpc.disconnectStepBuilder(req.Name)
		_ = catalog.SetRemoteOnline(req.Name, false, time.Now().UTC())
		return status.Error(codes.InvalidArgument, err.Error())
	}
	return nil
}

func stepRegistrationRequested(req *clientpb.Builder) bool {
	return len(req.GetSessionToken()) != 0 || len(req.GetSpecRefs()) != 0 || len(req.GetCapabilities()) != 0 || req.GetStatus() != nil
}

//nolint:gocyclo // Authentication, capability validation, and registration share one state transition.
func prepareStepRegistration(req *clientpb.Builder, lookup func(buildcatalog.Reference) (buildcatalog.Snapshot, error)) (buildscheduler.Registration, buildscheduler.Status, bool, error) {
	if !stepRegistrationRequested(req) {
		return buildscheduler.Registration{}, buildscheduler.Status{}, false, nil
	}
	if len(req.GetSessionToken()) != 32 || req.GetOperatorName() == "" || req.GetStatus() == nil ||
		len(req.GetSpecRefs()) == 0 || len(req.GetCapabilities()) == 0 || len(req.GetSpecYAML()) == 0 {
		return buildscheduler.Registration{}, buildscheduler.Status{}, true, status.Error(codes.InvalidArgument, "incomplete authenticated step builder registration")
	}
	refs := make(map[buildplan.Reference]*buildscheduler.SpecSupport, len(req.SpecRefs))
	registration := buildscheduler.Registration{Name: req.Name}
	for _, advertised := range req.SpecRefs {
		if advertised == nil || advertised.Name == "" || advertised.Version == "" || advertised.Digest == "" {
			return buildscheduler.Registration{}, buildscheduler.Status{}, true, status.Error(codes.InvalidArgument, "builder spec reference must be exact")
		}
		ref := referenceFromProto(advertised)
		if _, duplicate := refs[ref]; duplicate {
			return buildscheduler.Registration{}, buildscheduler.Status{}, true, status.Error(codes.InvalidArgument, "duplicate builder spec reference")
		}
		snapshot, lookupErr := lookup(buildcatalog.Reference{Name: ref.Name, Version: ref.Version, Digest: ref.Digest})
		if lookupErr != nil || !builderSentSpecYAML(req.SpecYAML, snapshot.RawYAML) {
			return buildscheduler.Registration{}, buildscheduler.Status{}, true, status.Error(codes.InvalidArgument, "builder spec is absent from its exact YAML advertisement")
		}
		if snapshot.Core != nil && !remotelyIntegratedCore(snapshot) ||
			snapshot.Transform != nil && !builtinSnapshot(snapshot) {
			return buildscheduler.Registration{}, buildscheduler.Status{}, true, status.Error(codes.InvalidArgument, "remote spec has no installed assigned-step material or transform integration")
		}
		registration.Specs = append(registration.Specs, buildscheduler.SpecSupport{Reference: ref})
		refs[ref] = &registration.Specs[len(registration.Specs)-1]
	}
	// A map of slice element pointers would become stale after append above.
	refs = make(map[buildplan.Reference]*buildscheduler.SpecSupport, len(registration.Specs))
	for index := range registration.Specs {
		refs[registration.Specs[index].Reference] = &registration.Specs[index]
	}
	for _, advertised := range req.Capabilities {
		if advertised == nil || advertised.Spec == nil || advertised.Capability == nil {
			return buildscheduler.Registration{}, buildscheduler.Status{}, true, status.Error(codes.InvalidArgument, "incomplete builder capability")
		}
		ref := referenceFromProto(advertised.Spec)
		support := refs[ref]
		if support == nil {
			return buildscheduler.Registration{}, buildscheduler.Status{}, true, status.Error(codes.InvalidArgument, "capability references an unadvertised spec")
		}
		snapshot, _ := lookup(buildcatalog.Reference{Name: ref.Name, Version: ref.Version, Digest: ref.Digest})
		if !matchesCatalogCapability(snapshot, advertised.Capability) {
			return buildscheduler.Registration{}, buildscheduler.Status{}, true, status.Error(codes.InvalidArgument, "capability differs from exact advertised spec")
		}
		capability := advertised.Capability
		if capability.Output == nil || capability.ToolchainDriver == "" || len(capability.Targets) == 0 {
			return buildscheduler.Registration{}, buildscheduler.Status{}, true, status.Error(codes.InvalidArgument, "builder capability has no output, target, or toolchain")
		}
		for _, target := range capability.Targets {
			if target == nil {
				return buildscheduler.Registration{}, buildscheduler.Status{}, true, status.Error(codes.InvalidArgument, "builder capability has nil target")
			}
			resolved := targetFromProto(target)
			var input *buildplan.Artifact
			if capability.Input != nil {
				value := artifactFromProto(capability.Input)
				value.Target = resolved
				input = &value
			}
			output := artifactFromProto(capability.Output)
			output.Target = resolved
			support.Capabilities = append(support.Capabilities, buildscheduler.Capability{
				Target: resolved, Input: input, Output: output,
				Mode: capability.Mode, Protocols: slices.Clone(capability.Protocols),
				Toolchain: buildscheduler.Toolchain{Driver: capability.ToolchainDriver, Requirements: slices.Clone(capability.ToolchainRequirements)},
			})
		}
	}
	builderStatus := builderStatusFromProto(req.Status.EffectiveCPUCores, req.Status.AvailableMemoryBytes, req.Status.Toolchains)
	validation := buildscheduler.New(buildscheduler.DefaultStatusTTL)
	if err := validation.Register(registration, builderStatus, time.Now()); err != nil {
		return buildscheduler.Registration{}, buildscheduler.Status{}, true, status.Error(codes.InvalidArgument, err.Error())
	}
	return registration, builderStatus, true, nil
}

func builtinSnapshot(snapshot buildcatalog.Snapshot) bool {
	for _, origin := range snapshot.Sources {
		if origin.Source == "builtin" {
			return true
		}
	}
	return false
}

func remotelyIntegratedCore(snapshot buildcatalog.Snapshot) bool {
	if snapshot.Core == nil {
		return false
	}
	if snapshot.Reference.Name == "builtin/sliver" {
		return true // Existing Sliver preparation retains its compatibility contract.
	}
	adapter, installed := buildcore.DefaultAdapter(snapshot.Core.Executor.ID)
	if !installed || !matchesInstalledMageBinding(snapshot.Core, adapter) {
		return false
	}
	_, remote := adapter.(buildcore.RemoteCoreAdapter)
	return remote
}

func builderSentSpecYAML(advertised []string, raw []byte) bool {
	for _, value := range advertised {
		if bytes.Equal([]byte(value), raw) {
			return true
		}
	}
	return false
}

func matchesCatalogCapability(snapshot buildcatalog.Snapshot, advertised *clientpb.BuildSpecCapability) bool {
	for _, allowed := range buildSpecProto(snapshot).Capabilities {
		if len(advertised.Targets) == 0 {
			continue
		}
		allTargetsAllowed := true
		for _, target := range advertised.Targets {
			if target == nil {
				allTargetsAllowed = false
				break
			}
			found := false
			for _, candidate := range allowed.Targets {
				if proto.Equal(target, candidate) {
					found = true
					break
				}
			}
			if !found {
				allTargetsAllowed = false
				break
			}
		}
		if !allTargetsAllowed {
			continue
		}
		allowedCopy := proto.Clone(allowed).(*clientpb.BuildSpecCapability)
		advertisedCopy := proto.Clone(advertised).(*clientpb.BuildSpecCapability)
		if !normalizeClaimArtifactTarget(advertisedCopy.Output, advertised.Targets) ||
			!normalizeClaimArtifactTarget(advertisedCopy.Input, advertised.Targets) {
			continue
		}
		allowedCopy.Targets, advertisedCopy.Targets = nil, nil
		if proto.Equal(allowedCopy, advertisedCopy) {
			return true
		}
	}
	return false
}

func normalizeClaimArtifactTarget(artifact *clientpb.BuildArtifactType, targets []*clientpb.BuildTarget) bool {
	if artifact == nil {
		return true
	}
	if artifact.GOOS == "" && artifact.GOARCH == "" {
		return true
	}
	for _, target := range targets {
		if target == nil || target.GOOS != artifact.GOOS || target.GOARCH != artifact.GOARCH {
			return false
		}
	}
	artifact.GOOS, artifact.GOARCH = "", ""
	return true
}

func (rpc *Server) disconnectStepBuilder(name string) {
	if scheduler, err := rpc.stepScheduler(); err == nil {
		scheduler.Disconnect(name)
	}
	cancelStepAssignments(name)
}

func (rpc *Server) authorizeStepBuilder(ctx context.Context, name string, token []byte) error {
	if name == "" || len(token) != 32 {
		return status.Error(codes.Unauthenticated, "invalid builder credentials")
	}
	operator := rpc.getClientCommonName(ctx)
	if operator == "" {
		return status.Error(codes.Unauthenticated, "Authentication failure")
	}
	builder := core.GetBuilder(name)
	if builder == nil || builder.OperatorName != operator || len(builder.SessionToken) != 32 ||
		subtle.ConstantTimeCompare(builder.SessionToken, token) != 1 {
		return status.Error(codes.PermissionDenied, "builder identity mismatch")
	}
	return nil
}

// BuilderUpdateStatus records a status update for an authenticated builder session.
func (rpc *Server) BuilderUpdateStatus(ctx context.Context, req *clientpb.BuilderStatusReport) (*commonpb.Empty, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "missing builder status report")
	}
	if err := rpc.authorizeStepBuilder(ctx, req.Name, req.SessionToken); err != nil {
		return nil, err
	}
	scheduler, err := rpc.stepScheduler()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "open build scheduler: %v", err)
	}
	if err := scheduler.UpdateStatus(req.Name, builderStatusFromProto(req.EffectiveCPUCores, req.AvailableMemoryBytes, req.Toolchains), time.Now()); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	return &commonpb.Empty{}, nil
}

func (rpc *Server) stepBuilderInventory() *clientpb.Builders {
	response := &clientpb.Builders{}
	byName := make(map[string]*clientpb.Builder)
	for _, registered := range core.AllBuilders() {
		builder := proto.Clone(registered).(*clientpb.Builder)
		builder.SessionToken = nil
		builder.Status = &clientpb.BuilderStatus{Online: true}
		response.Builders = append(response.Builders, builder)
		byName[builder.Name] = builder
	}
	if scheduler, err := rpc.stepScheduler(); err == nil {
		for _, entry := range scheduler.Inventory(time.Now()) {
			builder := byName[entry.Name]
			if builder == nil {
				builder = &clientpb.Builder{Name: entry.Name}
				for _, spec := range entry.Specs {
					ref := buildReferenceProto(spec.Reference)
					builder.SpecRefs = append(builder.SpecRefs, ref)
					for _, cap := range spec.Capabilities {
						wire := &clientpb.BuildSpecCapability{
							Targets: []*clientpb.BuildTarget{buildTargetProto(cap.Target)},
							Mode:    cap.Mode, Protocols: slices.Clone(cap.Protocols),
							Output: buildArtifactProto(cap.Output), ToolchainDriver: cap.Toolchain.Driver,
							ToolchainRequirements: slices.Clone(cap.Toolchain.Requirements),
						}
						if cap.Input != nil {
							wire.Input = buildArtifactProto(*cap.Input)
						}
						builder.Capabilities = append(builder.Capabilities, &clientpb.BuilderCapability{Spec: ref, Capability: wire})
					}
				}
				response.Builders = append(response.Builders, builder)
			}
			toolchainAvailable := false
			for _, tool := range entry.Toolchains {
				toolchainAvailable = toolchainAvailable || tool.Available
			}
			builder.Status = &clientpb.BuilderStatus{
				EffectiveCPUCores: uint32(entry.EffectiveCPUCores), AvailableMemoryBytes: entry.AvailableMemoryBytes,
				ReportedUnix: entry.ReportedAt.Unix(), Online: entry.Connected,
				Available:           entry.Connected && entry.Fresh && !entry.Exclusive && entry.AvailableMemoryBytes > 0 && toolchainAvailable,
				ReservedMemoryBytes: entry.ReservedMemoryBytes,
			}
			for _, tool := range entry.Toolchains {
				builder.Status.Toolchains = append(builder.Status.Toolchains, &clientpb.BuilderToolchainStatus{
					Driver: tool.Driver, Target: buildTargetProto(tool.Target), Requirements: slices.Clone(tool.Requirements),
					Identity: tool.Identity, Available: tool.Available,
				})
			}
		}
	}
	slices.SortFunc(response.Builders, func(a, b *clientpb.Builder) int { return bytes.Compare([]byte(a.Name), []byte(b.Name)) })
	return response
}

func requirementForStep(plan buildplan.Plan, step buildplan.Step) buildscheduler.Requirement {
	requirement := buildscheduler.Requirement{
		Spec: step.Spec, Target: step.Output.Target, Input: step.Input, Output: step.Output,
		Toolchain: buildscheduler.Toolchain{Driver: step.Toolchain.Driver, Requirements: slices.Clone(step.Toolchain.Requirements)},
	}
	if step.Kind == "core" {
		requirement.Mode = plan.Request.Mode
		for _, endpoint := range plan.Request.C2 {
			requirement.Protocols = append(requirement.Protocols, endpoint.Protocol)
		}
		requirement.Protocols = append(requirement.Protocols, plan.Request.IncludeProtocols...)
		slices.Sort(requirement.Protocols)
		requirement.Protocols = slices.Compact(requirement.Protocols)
	}
	return requirement
}

func buildSchedulerRequirement(snapshot buildcatalog.Snapshot, target buildplan.Target, input *buildplan.Artifact,
	output buildplan.Artifact, toolchain buildplan.Toolchain, mode string, protocols []string) buildscheduler.Requirement {
	return buildscheduler.Requirement{
		Spec:   buildplan.Reference{Name: snapshot.Reference.Name, Version: snapshot.Reference.Version, Digest: snapshot.Reference.Digest},
		Target: target, Input: input, Output: output, Mode: mode, Protocols: slices.Clone(protocols),
		Toolchain: buildscheduler.Toolchain{Driver: toolchain.Driver, Requirements: slices.Clone(toolchain.Requirements)},
	}
}

//nolint:gocyclo // Core and transform estimates are compared with overrides in one calculation.
func (rpc *Server) stepPeakBytes(plan buildplan.Plan, step buildplan.Step) uint64 {
	var specEstimate, executorEstimate uint64
	catalog, err := rpc.buildCatalog()
	if err == nil {
		snapshot, lookupErr := catalog.Get(buildcatalog.Reference{Name: step.Spec.Name, Version: step.Spec.Version, Digest: step.Spec.Digest})
		if lookupErr == nil {
			if snapshot.Core != nil {
				if adapter, installed := buildcore.DefaultAdapter(snapshot.Core.Executor.ID); installed {
					if estimator, ok := adapter.(buildcore.PeakMemoryEstimator); ok {
						executorEstimate = estimator.EstimatedPeakMemoryBytes(step.Output.Target, step.Toolchain)
					}
				}
				for _, cap := range snapshot.Core.Capabilities {
					if cap.Resources != nil && cap.Resources.EstimatedPeakMemoryBytes > 0 && cap.Mode == plan.Request.Mode &&
						catalogCapTarget(cap.Targets, step.Output.Target) && catalogCapArtifact(cap.Output.Artifact, step.Output) &&
						catalogCapToolchain(cap.Toolchain, step.Toolchain) && catalogCapProtocols(cap.Protocols, plan.Request) {
						specEstimate = max(specEstimate, cap.Resources.EstimatedPeakMemoryBytes)
					}
				}
			} else if snapshot.Transform != nil {
				for _, cap := range snapshot.Transform.Capabilities {
					if cap.Resources != nil && cap.Resources.EstimatedPeakMemoryBytes > 0 && step.Input != nil &&
						catalogCapTarget(cap.Targets, step.Output.Target) && catalogCapArtifact(cap.Input, *step.Input) &&
						catalogCapArtifact(cap.Output, step.Output) && catalogCapToolchain(cap.Toolchain, step.Toolchain) {
						specEstimate = max(specEstimate, cap.Resources.EstimatedPeakMemoryBytes)
					}
				}
			}
		}
	}
	// Zero means neither source knows a safe estimate. The scheduler then
	// reserves the selected builder exclusively for this step.
	return max(specEstimate, executorEstimate)
}

func catalogCapProtocols(supported []string, request buildplan.Request) bool {
	for _, endpoint := range request.C2 {
		if !slices.Contains(supported, endpoint.Protocol) {
			return false
		}
	}
	for _, protocol := range request.IncludeProtocols {
		if !slices.Contains(supported, protocol) {
			return false
		}
	}
	return true
}

func catalogCapTarget(targets []buildcatalog.Target, target buildplan.Target) bool {
	for _, candidate := range targets {
		if candidate.OS == target.OS && candidate.Arch == target.Arch {
			return true
		}
	}
	return false
}

func catalogCapArtifact(capability buildcatalog.Artifact, artifact buildplan.Artifact) bool {
	return capability.Kind == artifact.Kind && capability.Container == artifact.Container && capability.GoBuildMode == artifact.GoBuildMode &&
		(len(capability.EntrySymbols) == 0 || slices.Equal(capability.EntrySymbols, artifact.EntrySymbols))
}

func catalogCapToolchain(capability buildcatalog.Toolchain, toolchain buildplan.Toolchain) bool {
	if capability.Driver != toolchain.Driver || len(capability.Requirements) != len(toolchain.Requirements) {
		return false
	}
	for _, requirement := range capability.Requirements {
		if !slices.Contains(toolchain.Requirements, requirement) {
			return false
		}
	}
	return true
}

func selectionFromReservation(reservation buildscheduler.Reservation) *buildplan.StepSelection {
	ordered := make([]buildplan.SelectionCandidate, 0, len(reservation.Decision.Candidates))
	var selected *buildscheduler.Candidate
	for _, candidate := range reservation.Decision.Candidates {
		ordered = append(ordered, buildplan.SelectionCandidate{
			Name: candidate.Name, ReportedAt: candidate.ReportedAt,
			AvailableMemoryBytes: candidate.ReportedMemoryBytes, ReservedMemoryBytes: candidate.ReservedMemoryBytes,
			RoundedAvailableMB: candidate.RoundedAvailableMB, EffectiveCPUCores: uint32(candidate.EffectiveCPUCores),
			ToolchainIdentity: candidate.ToolchainIdentity,
		})
		if candidate.Name == reservation.Builder {
			selectedCandidate := candidate
			selected = &selectedCandidate
		}
	}
	if selected == nil {
		return nil
	}
	decision := "ranked available memory, CPU cores, name"
	if reservation.Decision.Pin != "" {
		decision = "pinned builder after capability and capacity checks"
	}
	return &buildplan.StepSelection{
		Builder: reservation.Builder, ReportedAt: selected.ReportedAt,
		AvailableMemoryBytes: selected.ReportedMemoryBytes, ReservedMemoryBytes: selected.ReservedMemoryBytes,
		RoundedAvailableMB: selected.RoundedAvailableMB, EffectiveCPUCores: uint32(selected.EffectiveCPUCores),
		RequiredPeakMemoryBytes: reservation.Decision.PeakMemoryBytes, Decision: decision,
		Candidates: ordered,
	}
}
