package builder

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/server/assets"
	"github.com/bishopfox/sliver/server/buildcatalog"
	"github.com/bishopfox/sliver/server/buildcore"
	"github.com/bishopfox/sliver/server/buildcore/artifactaccept"
	"github.com/bishopfox/sliver/server/buildplan"
	"github.com/bishopfox/sliver/server/buildtransform"
	"github.com/bishopfox/sliver/server/generate"
)

const assignedStepEvent = "build-step-assigned"
const maxStepInputBytes = 512 << 20
const maxAssignedCoreMaterialBytes = 16 << 20

// handleAssignedStep treats the event as a notification only. The assignment
// and its per-build material are fetched through the authenticated RPC before
// anything is compiled. The server also authenticates the result upload.
func (b *Builder) handleAssignedStep(ctx context.Context, event *clientpb.Event) {
	name, assignmentID, err := splitAssignedEvent(event.GetData())
	if err != nil || name != b.externalBuilder.Name {
		builderLog.Warnf("Ignoring invalid or foreign step assignment event")
		return
	}
	ref := &clientpb.BuilderStepRef{
		BuilderName: name, SessionToken: b.externalBuilder.SessionToken, AssignmentID: assignmentID,
	}
	request, err := b.rpc.BuilderGetStep(ctx, ref)
	if err != nil {
		builderLog.Warnf("Failed to fetch assigned step %s: %s", assignmentID, err)
		return
	}
	result := &clientpb.BuilderStepResult{
		BuilderName: name, SessionToken: b.externalBuilder.SessionToken, AssignmentID: assignmentID,
	}
	if request != nil && request.Step != nil {
		result.Spec = request.Step.Spec
		if request.Step.Output != nil {
			result.Target = &clientpb.BuildTarget{GOOS: request.Step.Output.GOOS, GOARCH: request.Step.Output.GOARCH}
		}
	}
	b.mutex.Lock()
	output, identity, stepErr := b.executeAssignedStep(ctx, assignmentID, request)
	b.mutex.Unlock()
	if stepErr != nil {
		stepID := ""
		if request != nil && request.Step != nil {
			stepID = request.Step.ID
		}
		result.Failure = &clientpb.BuildFailure{Code: "builder-step-failed", Message: stepErr.Error(), StepID: stepID}
	} else {
		result.Output = output
		result.ToolchainIdentity = identity
	}
	if _, err := b.rpc.BuilderSubmitStep(ctx, result); err != nil {
		builderLog.Errorf("Failed to submit assigned step %s: %s", assignmentID, err)
		return
	}
	if stepErr != nil {
		builderLog.Errorf("Assigned step %s failed: %s", assignmentID, stepErr)
	} else {
		builderLog.Infof("Assigned step %s completed", assignmentID)
	}
}

func splitAssignedEvent(data []byte) (name, assignmentID string, err error) {
	index := bytes.LastIndexByte(data, ':')
	if index < 1 || index == len(data)-1 || len(data) > 512 {
		return "", "", errors.New("invalid step assignment event")
	}
	name, assignmentID = string(data[:index]), string(data[index+1:])
	if strings.ContainsAny(assignmentID, "/\\\x00\r\n:") || !utf8.ValidString(name) || !utf8.ValidString(assignmentID) {
		return "", "", errors.New("invalid step assignment identity")
	}
	return name, assignmentID, nil
}

func (b *Builder) executeAssignedStep(ctx context.Context, assignmentID string,
	request *clientpb.BuilderStepRequest) (*clientpb.BuildArtifactPayload, string, error) {
	if request != nil {
		defer clear(request.CoreMaterial)
	}
	if ctx == nil || request == nil || request.Step == nil || request.Request == nil ||
		request.AssignmentID != assignmentID || assignmentID == "" || request.BuildID == "" || request.Step.ID == "" {
		return nil, "", errors.New("missing or mismatched assigned step")
	}
	step, err := stepFromWire(request.Step)
	if err != nil {
		return nil, "", err
	}
	if step.Provider != b.externalBuilder.Name || step.Spec.Name == "" || step.Spec.Version == "" || step.Spec.Digest == "" ||
		step.Output.Target != targetFromWire(request.Request.Target) ||
		(step.Kind != "core" && step.Kind != "transform") {
		return nil, "", errors.New("assigned step has an incompatible builder, spec, or target")
	}
	if !b.claimsStep(step, request.Request) {
		return nil, "", errors.New("assigned step is outside the registered capability tuple")
	}
	catalog, err := buildcatalog.Open(assets.GetRootAppDir(), buildcore.InstalledExecutorIDs())
	if err != nil {
		return nil, "", err
	}
	snapshot, err := catalog.Get(buildcatalog.Reference{Name: step.Spec.Name, Version: step.Spec.Version, Digest: step.Spec.Digest})
	if err != nil || snapshot.Kind != map[string]string{"core": "CoreSpec", "transform": "TransformSpec"}[step.Kind] {
		return nil, "", errors.New("assigned step differs from an installed immutable spec")
	}
	var data []byte
	var name, identity string
	if step.Kind == "core" {
		if step.Input != nil || request.Input != nil || len(request.Inputs) != 0 || snapshot.Core == nil {
			return nil, "", errors.New("core assignment has unexpected input or spec")
		}
		data, name, identity, err = b.executeCore(ctx, request, step, snapshot)
	} else {
		if step.Input == nil || request.Input == nil || request.SliverConfig != nil || len(request.CoreMaterial) != 0 || snapshot.Transform == nil {
			return nil, "", errors.New("transform assignment is missing typed input or has unexpected core material")
		}
		data, name, identity, err = executeTransform(ctx, request, step)
	}
	if err != nil {
		return nil, "", err
	}
	info, err := artifactaccept.InspectBytes(data, step.Output, name)
	if err != nil {
		return nil, "", fmt.Errorf("assigned step output: %w", err)
	}
	return &clientpb.BuildArtifactPayload{
		Type: artifactToWire(info.Type), FileName: info.Name, Data: data, SHA256: info.SHA256,
	}, identity, nil
}

func (b *Builder) claimsStep(step buildplan.Step, request *clientpb.BuildRequest) bool {
	for _, claim := range b.externalBuilder.Capabilities {
		if claim == nil || claim.Spec == nil || claim.Capability == nil ||
			claim.Spec.Name != step.Spec.Name || claim.Spec.Version != step.Spec.Version || claim.Spec.Digest != step.Spec.Digest {
			continue
		}
		capability := claim.Capability
		if capability.ToolchainDriver != step.Toolchain.Driver ||
			!slices.Equal(capability.ToolchainRequirements, step.Toolchain.Requirements) ||
			!sameClaimArtifact(capability.Output, step.Output) ||
			step.Input != nil && !sameClaimArtifact(capability.Input, *step.Input) ||
			step.Input == nil && capability.Input != nil {
			continue
		}
		if capability.RequiresEntrySymbol && (step.Input == nil || len(step.Input.EntrySymbols) == 0) {
			continue
		}
		for _, target := range capability.Targets {
			if target == nil || target.GOOS != step.Output.Target.OS || target.GOARCH != step.Output.Target.Arch {
				continue
			}
			if step.Kind == "core" && (capability.Mode != request.Mode || !claimProtocols(capability.Protocols, request)) {
				continue
			}
			return true
		}
	}
	return false
}

func sameClaimArtifact(claim *clientpb.BuildArtifactType, artifact buildplan.Artifact) bool {
	return claim != nil && claim.Kind == artifact.Kind && claim.Container == artifact.Container &&
		claim.GoBuildMode == artifact.GoBuildMode &&
		(claim.GOOS == "" || claim.GOOS == artifact.Target.OS) &&
		(claim.GOARCH == "" || claim.GOARCH == artifact.Target.Arch) &&
		(len(claim.EntrySymbols) == 0 || slices.Equal(claim.EntrySymbols, artifact.EntrySymbols))
}

func claimProtocols(allowed []string, request *clientpb.BuildRequest) bool {
	for _, protocol := range request.IncludeProtocols {
		if !slices.Contains(allowed, protocol) {
			return false
		}
	}
	for _, endpoint := range request.C2 {
		if endpoint == nil || !slices.Contains(allowed, endpoint.Protocol) {
			return false
		}
	}
	return true
}

func (b *Builder) executeCore(ctx context.Context, request *clientpb.BuilderStepRequest,
	step buildplan.Step, snapshot buildcatalog.Snapshot) ([]byte, string, string, error) {
	adapter, exists := buildcore.DefaultAdapter(snapshot.Core.Executor.ID)
	if !exists || !matchesBinding(snapshot.Core, adapter) {
		return nil, "", "", errors.New("core executor is unavailable or changed")
	}
	var remote buildcore.RemoteCoreAdapter
	if step.Spec.Name == "builtin/sliver" {
		if len(request.CoreMaterial) != 0 || request.SliverConfig == nil || request.SliverConfig.Build == nil ||
			request.SliverConfig.Build.ID != request.BuildID {
			return nil, "", "", errors.New("Sliver assignment has no valid prepared cryptographic material")
		}
	} else {
		remote, exists = adapter.(buildcore.RemoteCoreAdapter)
		if !exists || request.SliverConfig != nil || len(request.CoreMaterial) == 0 ||
			len(request.CoreMaterial) > maxAssignedCoreMaterialBytes {
			return nil, "", "", errors.New("core has no valid prepared remote material or installed integration")
		}
	}
	availability, exists := adapter.(buildcore.Availability)
	if !exists || !availability.Supports(step.Output.Target, step.Toolchain) {
		return nil, "", "", errors.New("core toolchain is no longer available")
	}
	local := step
	local.Provider = "server" // Installed adapters validate their local binding.
	buildRequest, err := requestFromWire(request.Request)
	if err != nil {
		return nil, "", "", err
	}
	plan := buildplan.Plan{Request: buildRequest, Steps: []buildplan.Step{local}, FinalArtifact: local.Output}
	if step.Spec.Name == "builtin/sliver" {
		result, err := generate.CompilePreparedSliverCore(ctx, plan, request.SliverConfig)
		return result.Data, result.FileName, result.ToolchainIdentity, err
	}
	material := bytes.Clone(request.CoreMaterial)
	defer clear(material)
	result, err := remote.BuildRemote(ctx, plan, material)
	return result.Data, result.FileName, result.ToolchainIdentity, err
}

func executeTransform(ctx context.Context, request *clientpb.BuilderStepRequest,
	step buildplan.Step) ([]byte, string, string, error) {
	input, err := inspectInputPayload(request.Input, *step.Input)
	if err != nil {
		return nil, "", "", err
	}
	local := step
	local.Provider = "server" // Installed transform functions validate this binding.
	if err := buildtransform.ValidateInstalledStep(local); err != nil {
		return nil, "", "", err
	}
	resolve, cleanup, err := stepInputResolver(request.Inputs)
	if err != nil {
		return nil, "", "", err
	}
	defer cleanup()
	var output []byte
	switch local.Spec.Name {
	case "builtin/donut", "builtin/beignet", "builtin/malasada":
		workDir := ""
		if local.Spec.Name == "builtin/malasada" {
			workDir, err = os.MkdirTemp(assets.GetRootAppDir(), "remote-transform-*")
			if err != nil {
				return nil, "", "", err
			}
		}
		output, err = buildtransform.Convert(ctx, local, input, workDir)
		if workDir != "" {
			if removeErr := os.RemoveAll(workDir); removeErr != nil {
				return nil, "", "", fmt.Errorf("remove private transform workspace: %w", removeErr)
			}
		}
	case "builtin/shikata-ga-nai", "builtin/xor", "builtin/xor-dynamic":
		output, err = buildtransform.EncodeShellcode(ctx, local, input)
	case buildtransform.StageFrameSpec, buildtransform.StageCompressSpec, buildtransform.StageAESSpec, buildtransform.StageRC4Spec:
		output, err = buildtransform.ApplyStageStep(ctx, local, input, resolve)
	case buildtransform.PEMetadataSpec:
		output, err = applyRemotePEMetadata(ctx, local, input, resolve)
	default:
		return nil, "", "", errors.New("transform has no installed executor")
	}
	if err != nil {
		return nil, "", "", err
	}
	name := request.Input.FileName
	if local.Output.Kind == "shellcode" || local.Output.Kind == "stage" {
		name = "output.bin"
	}
	return output, name, "builtin", nil
}

func applyRemotePEMetadata(ctx context.Context, step buildplan.Step, input []byte,
	resolve buildtransform.ResolveInput) ([]byte, error) {
	var donorRef struct {
		Kind string `json:"kind"`
		Name string `json:"name"`
	}
	for _, option := range step.Options {
		if option.Name == "donorArtifact" {
			if err := json.Unmarshal(option.ValueJSON, &donorRef); err != nil {
				return nil, errors.New("invalid PE donor reference")
			}
			break
		}
	}
	if donorRef.Kind != "input-artifact" || donorRef.Name == "" || resolve == nil {
		return nil, errors.New("PE donor is unavailable")
	}
	donor, err := resolve(donorRef.Kind, donorRef.Name)
	if err != nil {
		return nil, errors.New("PE donor is unavailable")
	}
	if strings.HasPrefix(donorRef.Name, "metadata-sha256:") {
		config, err := resolve("metadata-config", donorRef.Name)
		if err != nil {
			return nil, errors.New("PE metadata configuration is unavailable")
		}
		return buildtransform.ApplyPEMetadataConfig(ctx, step, input, donor, config)
	}
	return buildtransform.ApplyPEMetadata(ctx, step, input, donor)
}

func inspectInputPayload(input *clientpb.BuildArtifactPayload, expected buildplan.Artifact) ([]byte, error) {
	if input == nil || input.Type == nil || len(input.Data) == 0 || len(input.Data) > maxStepInputBytes ||
		!sameArtifactType(input.Type, expected) {
		return nil, errors.New("typed step input is missing or incompatible")
	}
	info, err := artifactaccept.InspectBytes(input.Data, expected, input.FileName)
	if err != nil || info.SHA256 != input.SHA256 {
		return nil, errors.New("typed step input failed artifact or checksum validation")
	}
	return bytes.Clone(input.Data), nil
}

func sameArtifactType(actual *clientpb.BuildArtifactType, expected buildplan.Artifact) bool {
	return actual != nil && actual.Kind == expected.Kind && actual.Container == expected.Container &&
		actual.GOOS == expected.Target.OS && actual.GOARCH == expected.Target.Arch &&
		actual.GoBuildMode == expected.GoBuildMode && slices.Equal(actual.EntrySymbols, expected.EntrySymbols)
}

func stepFromWire(wire *clientpb.BuildPlanStep) (buildplan.Step, error) {
	if wire == nil || wire.Spec == nil || wire.Output == nil {
		return buildplan.Step{}, errors.New("assigned step is incomplete")
	}
	step := buildplan.Step{
		ID: wire.ID, Kind: wire.Kind, Provider: wire.Provider,
		Spec:      buildplan.Reference{Name: wire.Spec.Name, Version: wire.Spec.Version, Digest: wire.Spec.Digest},
		Output:    artifactFromWire(wire.Output),
		Toolchain: buildplan.Toolchain{Driver: wire.Toolchain, Requirements: slices.Clone(wire.ToolchainRequirements), Identity: wire.ToolchainIdentity},
	}
	if wire.Input != nil {
		input := artifactFromWire(wire.Input)
		step.Input = &input
	}
	for _, option := range wire.Options {
		if option == nil || option.Name == "" || !json.Valid(option.ValueJSON) {
			return buildplan.Step{}, errors.New("assigned step has an invalid option")
		}
		step.Options = append(step.Options, buildplan.Option{Name: option.Name, ValueJSON: bytes.Clone(option.ValueJSON)})
	}
	return step, nil
}

func requestFromWire(wire *clientpb.BuildRequest) (buildplan.Request, error) {
	request := buildplan.Request{
		Name: wire.Name, Target: targetFromWire(wire.Target), Mode: wire.Mode,
		FinalFormat: wire.Format, IncludeProtocols: slices.Clone(wire.IncludeProtocols),
	}
	if settings := wire.C2Settings; settings != nil {
		request.C2Settings = &buildplan.C2Settings{
			ConnectionStrategy: settings.ConnectionStrategy, ReconnectSeconds: settings.ReconnectSeconds,
			PollTimeoutSeconds: settings.PollTimeoutSeconds, MaxConnectionErrors: settings.MaxConnectionErrors,
			HTTPC2Profile: settings.HTTPC2Profile, WireguardKeyExchangePort: settings.WireguardKeyExchangePort,
			WireguardTCPCommsPort: settings.WireguardTCPCommsPort,
		}
	}
	if settings := wire.BeaconSettings; settings != nil {
		request.BeaconSettings = &buildplan.BeaconSettings{IntervalSeconds: settings.IntervalSeconds, JitterSeconds: settings.JitterSeconds}
	}
	if wire.Core != nil {
		request.Core = buildplan.Reference{Name: wire.Core.Name, Version: wire.Core.Version, Digest: wire.Core.Digest}
	}
	if wire.CoreOutput != nil {
		value := artifactFromWire(wire.CoreOutput)
		request.CoreOutput = &value
	}
	if wire.Provider != nil {
		request.Provider = buildplan.ProviderPolicy{Mode: wire.Provider.Mode, Builder: wire.Provider.Builder}
	}
	for _, endpoint := range wire.C2 {
		if endpoint == nil {
			return buildplan.Request{}, errors.New("assigned core request has a nil C2 endpoint")
		}
		request.C2 = append(request.C2, buildplan.Endpoint{Protocol: endpoint.Protocol, Address: endpoint.Address})
	}
	for _, option := range wire.CoreOptions {
		if option == nil || option.Name == "" || !json.Valid(option.ValueJSON) {
			return buildplan.Request{}, errors.New("assigned core request has an invalid option")
		}
		request.CoreOptions = append(request.CoreOptions, buildplan.Option{Name: option.Name, ValueJSON: bytes.Clone(option.ValueJSON)})
	}
	return request, nil
}

func artifactFromWire(wire *clientpb.BuildArtifactType) buildplan.Artifact {
	return buildplan.Artifact{
		Kind: wire.Kind, Container: wire.Container, GoBuildMode: wire.GoBuildMode,
		Target: buildplan.Target{OS: wire.GOOS, Arch: wire.GOARCH}, EntrySymbols: slices.Clone(wire.EntrySymbols),
	}
}

func artifactToWire(artifact buildplan.Artifact) *clientpb.BuildArtifactType {
	return &clientpb.BuildArtifactType{
		Kind: artifact.Kind, Container: artifact.Container, GOOS: artifact.Target.OS, GOARCH: artifact.Target.Arch,
		GoBuildMode: artifact.GoBuildMode, EntrySymbols: slices.Clone(artifact.EntrySymbols),
	}
}

func targetFromWire(wire *clientpb.BuildTarget) buildplan.Target {
	if wire == nil {
		return buildplan.Target{}
	}
	return buildplan.Target{OS: wire.GOOS, Arch: wire.GOARCH}
}

type stepMaterialKey struct{ kind, name string }

func stepInputResolver(inputs []*clientpb.BuildInput) (buildtransform.ResolveInput, func(), error) {
	material := make(map[stepMaterialKey][]byte, len(inputs))
	cleanup := func() {
		for key, value := range material {
			clear(value)
			delete(material, key)
		}
	}
	var total int
	for _, input := range inputs {
		if input == nil || (input.Kind != "secret" && input.Kind != "input-artifact" && input.Kind != "metadata-config") ||
			input.Name == "" || len(input.Name) > 128 || !utf8.ValidString(input.Name) ||
			strings.TrimSpace(input.Name) != input.Name || strings.ContainsAny(input.Name, "\x00\r\n") {
			cleanup()
			return nil, nil, errors.New("assigned step has an invalid per-build input")
		}
		limit := 128 << 20
		if input.Kind == "secret" {
			limit = 256
		}
		if len(input.Data) == 0 || len(input.Data) > limit || total > 256<<20-len(input.Data) {
			cleanup()
			return nil, nil, errors.New("assigned step per-build input exceeds size limit")
		}
		total += len(input.Data)
		key := stepMaterialKey{input.Kind, input.Name}
		if _, exists := material[key]; exists {
			cleanup()
			return nil, nil, errors.New("assigned step has duplicate per-build input")
		}
		material[key] = bytes.Clone(input.Data)
	}
	return func(kind, name string) ([]byte, error) {
		value, exists := material[stepMaterialKey{kind, name}]
		if !exists {
			return nil, errors.New("per-build input is unavailable")
		}
		return bytes.Clone(value), nil
	}, cleanup, nil
}

// Keep the digest calculation in this package for tests of typed payloads.
func payloadSHA256(data []byte) string {
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:])
}
