// Package buildplan resolves build requests and stores their plans and results.
package buildplan

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/bishopfox/sliver/server/buildcatalog"
)

// Catalog exposes immutable spec snapshots without coupling the planner to
// catalog persistence or source YAML parsing.
type Catalog interface {
	Get(buildcatalog.Reference) (buildcatalog.Snapshot, error)
	List(buildcatalog.Query) []buildcatalog.Snapshot
}

// ToolchainCheck must verify that a provider can execute a declared tuple on
// the requested target. The exact spec lets a caller select the registered
// executor without teaching the planner about any particular core. A nil
// callback fails closed, especially for CGo routes whose compiler availability
// depends on the server host.
type ToolchainCheck func(provider string, spec buildcatalog.Snapshot, target Target, toolchain Toolchain) bool

// Planner resolves requests against immutable catalog definitions.
type Planner struct {
	catalog            Catalog
	toolchain          ToolchainCheck
	providerRank       func(string) (uint64, int)
	providerCapability func(string, buildcatalog.Snapshot, Target, *Artifact, Artifact, Toolchain, string, []string) bool
	ignoreAvailability bool // Diagnostic topology search after an unavailable route.
}

// NewPlanner constructs a planner with the given catalog and toolchain check.
func NewPlanner(catalog Catalog, toolchain ToolchainCheck) *Planner {
	return &Planner{catalog: catalog, toolchain: toolchain}
}

// SetProviderRank supplies a dynamic, non-persisted capacity ordering for
// remote providers. The resolved provider is still frozen into the plan.
func (p *Planner) SetProviderRank(rank func(string) (uint64, int)) {
	p.providerRank = rank
}

// SetProviderCapabilityCheck narrows a provider to the exact resolved input,
// output, core mode, and protocols. Installed local adapters retain their
// existing toolchain check; remote workers use this for tuple validation.
func (p *Planner) SetProviderCapabilityCheck(check func(string, buildcatalog.Snapshot, Target, *Artifact, Artifact, Toolchain, string, []string) bool) {
	p.providerCapability = check
}

type route struct {
	steps         []Step
	final         Artifact
	directFormats []string
}

// Plan resolves one deterministic, non-executing path. It never writes a
// record, generates credentials, or invokes an executor.
func (p *Planner) Plan(ctx context.Context, request Request) (Plan, error) {
	plan, err := p.plan(ctx, request)
	if err == nil || p == nil || p.ignoreAvailability {
		return plan, err
	}
	var planError *PlanError
	if !errors.As(err, &planError) || planError.Code != CodeIncompatible && planError.Code != CodeUnavailableProvider {
		return Plan{}, err
	}
	// Availability and topology are different questions. Diagnose a failed
	// route against the same immutable definitions without current providers;
	// an impossible output remains incompatible even if some other capability
	// happens to be offline.
	topology := *p
	topology.ignoreAvailability = true
	_, topologyError := topology.plan(ctx, request)
	if topologyError == nil {
		return Plan{}, &PlanError{Code: CodeUnavailableProvider, Detail: "a compatible route has no online provider or toolchain"}
	}
	var topologyPlanError *PlanError
	if errors.As(topologyError, &topologyPlanError) && topologyPlanError.Code == CodeAmbiguousRoute {
		return Plan{}, &PlanError{Code: CodeUnavailableProvider, Detail: "compatible routes have no online provider or toolchain"}
	}
	if errors.As(topologyError, &topologyPlanError) && topologyPlanError.Code == CodeIncompatible {
		return Plan{}, topologyError
	}
	return Plan{}, err
}

func (p *Planner) plan(ctx context.Context, request Request) (Plan, error) {
	if p == nil || p.catalog == nil {
		return Plan{}, &PlanError{Code: CodeInvalidRequest, Detail: "catalog is unavailable"}
	}
	if err := ctx.Err(); err != nil {
		return Plan{}, err
	}
	request, protocols, err := normalizeRequest(request)
	if err != nil {
		return Plan{}, err
	}
	coreSnapshot, err := p.catalog.Get(catalogReference(request.Core))
	if err != nil {
		code := CodeUnknownSpec
		if errors.Is(err, buildcatalog.ErrAmbiguous) {
			code = CodeAmbiguousSpec
		}
		return Plan{}, &PlanError{Code: code, Detail: fmt.Sprintf("core %s: %v", request.Core.Name, err)}
	}
	if coreSnapshot.Core == nil {
		return Plan{}, &PlanError{Code: CodeIncompatible, Detail: "requested spec is not a core"}
	}
	request.Core = planReference(coreSnapshot.Reference)
	coreOptions, err := resolveOptions(request.CoreOptions, coreSnapshot.Core.Options)
	if err != nil {
		return Plan{}, &PlanError{Code: CodeInvalidRequest, Step: "core", Detail: err.Error()}
	}
	starts, compatibleCore, unavailableCore := p.coreRoutes(coreSnapshot, request, protocols, coreOptions)
	if len(starts) == 0 {
		if compatibleCore && unavailableCore {
			return Plan{}, &PlanError{Code: CodeUnavailableProvider, Step: "core", Detail: "no compatible online provider and toolchain"}
		}
		return Plan{}, &PlanError{Code: CodeIncompatible, Step: "core", Detail: "target, mode, protocol, or core output is unsupported"}
	}
	var chosen route
	if len(request.Transforms) > 0 {
		chosen, err = p.explicitRoute(ctx, request, starts)
	} else {
		chosen, err = p.automaticRoute(ctx, request, starts)
	}
	if err != nil {
		return Plan{}, err
	}
	plan := Plan{Request: request, Steps: chosen.steps, FinalArtifact: chosen.final}
	plan.SHA256, err = planDigest(plan)
	if err != nil {
		return Plan{}, err
	}
	return plan, nil
}

//nolint:gocyclo // Preserve validation order while normalizing the request and its derived protocol set.
func normalizeRequest(request Request) (Request, []string, error) {
	// Planning must not reorder or retain mutable slices owned by its caller.
	request.C2 = slices.Clone(request.C2)
	request.IncludeProtocols = slices.Clone(request.IncludeProtocols)
	request.CoreOptions = cloneOptions(request.CoreOptions)
	request.Transforms = slices.Clone(request.Transforms)
	if request.C2Settings != nil {
		settings := *request.C2Settings
		request.C2Settings = &settings
	}
	if request.BeaconSettings != nil {
		settings := *request.BeaconSettings
		request.BeaconSettings = &settings
	}
	if request.ProfileSnapshot != nil {
		snapshot := *request.ProfileSnapshot
		snapshot.YAML = slices.Clone(snapshot.YAML)
		request.ProfileSnapshot = &snapshot
	}
	for i := range request.Transforms {
		request.Transforms[i].Options = cloneOptions(request.Transforms[i].Options)
	}
	if request.CoreOutput != nil {
		coreOutput := *request.CoreOutput
		coreOutput.EntrySymbols = slices.Clone(coreOutput.EntrySymbols)
		request.CoreOutput = &coreOutput
	}
	if request.Target.OS == "" || request.Target.Arch == "" {
		return Request{}, nil, &PlanError{Code: CodeInvalidRequest, Detail: "target OS and architecture are required"}
	}
	if request.Mode != "session" && request.Mode != "beacon" {
		return Request{}, nil, &PlanError{Code: CodeInvalidRequest, Detail: "mode must be session or beacon"}
	}
	const maxDurationSeconds = uint64(9223372036)
	if settings := request.C2Settings; settings != nil {
		if settings.ConnectionStrategy != "sequential" && settings.ConnectionStrategy != "random" && settings.ConnectionStrategy != "random-domain" ||
			settings.HTTPC2Profile == "" || settings.ReconnectSeconds > maxDurationSeconds || settings.PollTimeoutSeconds > maxDurationSeconds ||
			settings.WireguardKeyExchangePort > 65535 || settings.WireguardTCPCommsPort > 65535 {
			return Request{}, nil, &PlanError{Code: CodeInvalidRequest, Detail: "invalid C2 runtime settings"}
		}
	}
	if settings := request.BeaconSettings; settings != nil {
		if request.Mode != "beacon" || settings.IntervalSeconds == 0 || settings.IntervalSeconds > maxDurationSeconds ||
			settings.JitterSeconds > maxDurationSeconds {
			return Request{}, nil, &PlanError{Code: CodeInvalidRequest, Detail: "invalid beacon runtime settings"}
		}
	}
	if _, ok := finalKinds[request.FinalFormat]; !ok {
		return Request{}, nil, &PlanError{Code: CodeIncompatible, Detail: fmt.Sprintf("unknown output format %q", request.FinalFormat)}
	}
	if request.Core.Name == "" {
		request.Core.Name = "builtin/sliver"
	}
	if request.Provider.Mode == "" {
		request.Provider.Mode = "automatic"
	}
	switch request.Provider.Mode {
	case "automatic", "remote":
		if request.Provider.Builder != "" {
			return Request{}, nil, &PlanError{Code: CodeInvalidRequest, Detail: "builder pin requires pinned provider policy"}
		}
	case "pinned":
		if request.Provider.Builder == "" {
			return Request{}, nil, &PlanError{Code: CodeInvalidRequest, Detail: "pinned provider requires builder name"}
		}
	default:
		return Request{}, nil, &PlanError{Code: CodeInvalidRequest, Detail: "unknown provider policy"}
	}
	if len(request.C2) == 0 {
		return Request{}, nil, &PlanError{Code: CodeInvalidRequest, Detail: "at least one C2 endpoint is required"}
	}
	protocolSet := map[string]bool{}
	for i := range request.C2 {
		request.C2[i].Protocol = normalizeProtocol(request.C2[i].Protocol)
		if !validProtocol(request.C2[i].Protocol) || !validEndpoint(request.C2[i]) {
			return Request{}, nil, &PlanError{Code: CodeInvalidRequest, Detail: "invalid C2 endpoint"}
		}
		protocolSet[request.C2[i].Protocol] = true
	}
	for i, protocol := range request.IncludeProtocols {
		protocol = normalizeProtocol(protocol)
		if !validProtocol(protocol) {
			return Request{}, nil, &PlanError{Code: CodeInvalidRequest, Detail: "invalid included protocol"}
		}
		request.IncludeProtocols[i] = protocol
		protocolSet[protocol] = true
	}
	protocols := make([]string, 0, len(protocolSet))
	for protocol := range protocolSet {
		if (protocol == "named-pipe" || protocol == "tcp-pivot") && request.Mode == "beacon" {
			return Request{}, nil, &PlanError{Code: CodeIncompatible, Detail: "pivot protocols require a session"}
		}
		if protocol == "named-pipe" && request.Target.OS != "windows" {
			return Request{}, nil, &PlanError{Code: CodeIncompatible, Detail: "named pipe requires Windows"}
		}
		protocols = append(protocols, protocol)
	}
	sort.Strings(protocols)
	sort.Strings(request.IncludeProtocols)
	request.IncludeProtocols = slices.Compact(request.IncludeProtocols)
	if request.CoreOutput != nil {
		if request.CoreOutput.Target == (Target{}) {
			request.CoreOutput.Target = request.Target
		}
		if request.CoreOutput.Target != request.Target {
			return Request{}, nil, &PlanError{Code: CodeIncompatible, Detail: "core output target differs from request target"}
		}
	}
	return request, protocols, nil
}

var finalKinds = map[string]string{
	"exe":       "executable",
	"service":   "service-executable",
	"shared":    "shared-library",
	"archive":   "c-archive-bundle",
	"shellcode": "shellcode",
	"stage":     "stage",
}

func normalizeProtocol(protocol string) string {
	switch strings.ToLower(protocol) {
	case "wg":
		return "wireguard"
	case "namedpipe":
		return "named-pipe"
	case "tcppivot":
		return "tcp-pivot"
	default:
		return strings.ToLower(protocol)
	}
}

func validProtocol(protocol string) bool {
	switch protocol {
	case "mtls", "http", "https", "dns", "wireguard", "named-pipe", "tcp-pivot":
		return true
	default:
		return false
	}
}

func validEndpoint(endpoint Endpoint) bool {
	parsed, err := url.Parse(endpoint.Address)
	if err != nil || parsed.Host == "" {
		return false
	}
	expectedScheme := endpoint.Protocol
	switch endpoint.Protocol {
	case "wireguard":
		expectedScheme = "wg"
	case "named-pipe":
		expectedScheme = "namedpipe"
	case "tcp-pivot":
		expectedScheme = "tcppivot"
	}
	return parsed.Scheme == expectedScheme
}

func (p *Planner) coreRoutes(snapshot buildcatalog.Snapshot, request Request, protocols []string, options []Option) ([]route, bool, bool) {
	routes := []route{}
	compatible, unavailable := false, false
	for _, capability := range snapshot.Core.Capabilities {
		if capability.Mode != request.Mode || !catalogTargetsContain(capability.Targets, request.Target) ||
			!containsProtocols(capability.Protocols, protocols) {
			continue
		}
		artifact := fromCatalogArtifact(capability.Output.Artifact, request.Target)
		artifact.EntrySymbols = resolvedEntrySymbols(artifact, options)
		if request.CoreOutput != nil && !artifactSelectorMatches(*request.CoreOutput, artifact) {
			continue
		}
		if request.FinalFormat == "service" && artifact.Kind != "service-executable" {
			continue
		}
		compatible = true
		toolchain := fromCatalogToolchain(capability.Toolchain)
		provider, ok := p.chooseProvider(snapshot, request.Provider, request.Target, toolchain, nil, artifact, request.Mode, protocols)
		if !ok {
			unavailable = true
			continue
		}
		step := Step{ID: "core", Kind: "core", Spec: planReference(snapshot.Reference), Provider: provider,
			Output: artifact, Toolchain: toolchain, Options: cloneOptions(options)}
		routes = append(routes, route{steps: []Step{step}, final: artifact, directFormats: slices.Clone(capability.Output.OperatorFormats)})
	}
	return dedupeRoutes(routes), compatible, unavailable
}

func resolvedEntrySymbols(artifact Artifact, options []Option) []string {
	if artifact.Kind != "shared-library" && artifact.Kind != "c-archive-bundle" {
		return artifact.EntrySymbols
	}
	for _, option := range options {
		if option.Name == "exports" {
			var exports []string
			if json.Unmarshal(option.ValueJSON, &exports) == nil {
				return exports
			}
		}
	}
	return artifact.EntrySymbols
}

//nolint:gocyclo // Shortest-path selection, deduplication, and unavailable-provider errors share the same search state.
func (p *Planner) explicitRoute(ctx context.Context, request Request, starts []route) (route, error) {
	resolved := make([]buildcatalog.Snapshot, len(request.Transforms))
	resolvedOptions := make([][]Option, len(request.Transforms))
	for i, transform := range request.Transforms {
		snapshot, err := p.catalog.Get(catalogReference(transform.Spec))
		if err != nil {
			code := CodeUnknownSpec
			if errors.Is(err, buildcatalog.ErrAmbiguous) {
				code = CodeAmbiguousSpec
			}
			return route{}, &PlanError{Code: code, Step: fmt.Sprintf("transform-%d", i+1), Detail: err.Error()}
		}
		if snapshot.Transform == nil {
			return route{}, &PlanError{Code: CodeIncompatible, Step: fmt.Sprintf("transform-%d", i+1), Detail: "spec is not a transform"}
		}
		options, err := resolveOptions(transform.Options, snapshot.Transform.Options)
		if err != nil {
			return route{}, &PlanError{Code: CodeInvalidRequest, Step: fmt.Sprintf("transform-%d", i+1), Detail: err.Error()}
		}
		resolved[i], resolvedOptions[i] = snapshot, options
		request.Transforms[i].Spec = planReference(snapshot.Reference)
	}
	// Explicit transforms are an ordered suffix. A converter may be needed to
	// reach its first input (for example, PE -> Donut -> shellcode encoder).
	transforms := p.sortedTransforms()
	frontier := dedupeRoutes(starts)
	unavailable := false
	maxDepth := maximumTransformDepth(transforms)
	for depth := 0; depth <= maxDepth; depth++ {
		winners := []route{}
		for _, current := range frontier {
			suffixes, suffixUnavailable := p.applyExplicitSuffix(current, request, resolved, resolvedOptions)
			unavailable = unavailable || suffixUnavailable
			for _, suffix := range suffixes {
				if routeMatchesFormat(suffix, request.FinalFormat) {
					winners = append(winners, suffix)
				}
			}
		}
		winners = dedupeRoutes(winners)
		if len(winners) == 1 {
			return winners[0], nil
		}
		if len(winners) > 1 {
			return route{}, &PlanError{Code: CodeAmbiguousRoute, Detail: "multiple shortest paths reach explicit transforms"}
		}
		if depth == maxDepth {
			break
		}
		var nextUnavailable bool
		frontier, nextUnavailable = p.expandAutomatic(frontier, transforms, request)
		unavailable = unavailable || nextUnavailable
		if len(frontier) == 0 {
			break
		}
		if err := ctx.Err(); err != nil {
			return route{}, err
		}
	}
	if unavailable {
		return route{}, &PlanError{Code: CodeUnavailableProvider, Detail: "an explicit path requires an unavailable provider or toolchain"}
	}
	return route{}, &PlanError{Code: CodeIncompatible, Detail: "explicit transform path does not reach requested output"}
}

func (p *Planner) applyExplicitSuffix(start route, request Request, transforms []buildcatalog.Snapshot,
	options [][]Option) ([]route, bool) {
	candidates := []route{start}
	unavailable := false
	for i, snapshot := range transforms {
		nextCandidates := []route{}
		for _, current := range candidates {
			if routeContainsTransformInput(current, snapshot.Reference, current.final) {
				continue
			}
			next, missingProvider := p.applyTransform(snapshot, request.Provider, request.Target,
				current, options[i], fmt.Sprintf("transform-%d", len(current.steps)))
			unavailable = unavailable || missingProvider
			for _, candidate := range next {
				if !artifactEqual(candidate.final, current.final) && routeContainsArtifact(current, candidate.final) {
					continue
				}
				nextCandidates = append(nextCandidates, candidate)
			}
		}
		candidates = dedupeRoutes(nextCandidates)
		if len(candidates) == 0 {
			break
		}
	}
	return candidates, unavailable
}

func (p *Planner) automaticRoute(ctx context.Context, request Request, starts []route) (route, error) {
	transforms := p.sortedTransforms()
	frontier := dedupeRoutes(starts)
	unavailable := false
	// Every automatic edge changes artifact type. A spec may contain several
	// different edges, so bound the simple path by capability count.
	maxDepth := maximumTransformDepth(transforms)
	for depth := 0; depth <= maxDepth; depth++ {
		winners := []route{}
		for _, current := range frontier {
			if routeMatchesFormat(current, request.FinalFormat) {
				winners = append(winners, current)
			}
		}
		winners = dedupeRoutes(winners)
		if len(winners) == 1 {
			return winners[0], nil
		}
		if len(winners) > 1 {
			return route{}, &PlanError{Code: CodeAmbiguousRoute, Detail: "multiple shortest transform paths reach requested output"}
		}
		if depth == maxDepth {
			break
		}
		var nextUnavailable bool
		frontier, nextUnavailable = p.expandAutomatic(frontier, transforms, request)
		unavailable = unavailable || nextUnavailable
		if len(frontier) == 0 {
			break
		}
		if err := ctx.Err(); err != nil {
			return route{}, err
		}
	}
	if unavailable {
		return route{}, &PlanError{Code: CodeUnavailableProvider, Detail: "a required transform has no compatible online provider or toolchain"}
	}
	return route{}, &PlanError{Code: CodeIncompatible, Detail: "no transform path reaches requested output"}
}

func (p *Planner) sortedTransforms() []buildcatalog.Snapshot {
	transforms := p.catalog.List(buildcatalog.Query{Kind: "TransformSpec"})
	sort.Slice(transforms, func(i, j int) bool {
		return referenceKey(transforms[i].Reference) < referenceKey(transforms[j].Reference)
	})
	return transforms
}

func maximumTransformDepth(transforms []buildcatalog.Snapshot) int {
	depth := 0
	for _, snapshot := range transforms {
		if snapshot.Transform != nil {
			depth += len(snapshot.Transform.Capabilities)
		}
	}
	return depth
}

func (p *Planner) expandAutomatic(frontier []route, transforms []buildcatalog.Snapshot, request Request) ([]route, bool) {
	nextFrontier := []route{}
	unavailable := false
	for _, current := range frontier {
		for _, snapshot := range transforms {
			if snapshot.Transform == nil {
				continue
			}
			options, err := resolveOptions(nil, snapshot.Transform.Options)
			if err != nil { // Required options make this an explicit-only step.
				continue
			}
			next, missingProvider := p.applyTransform(snapshot, request.Provider, request.Target,
				current, options, fmt.Sprintf("transform-%d", len(current.steps)))
			unavailable = unavailable || missingProvider
			for _, candidate := range next {
				if artifactEqual(candidate.final, current.final) || routeContainsArtifact(current, candidate.final) {
					continue // automatic self-loops and cycles cannot reach a new type
				}
				nextFrontier = append(nextFrontier, candidate)
			}
		}
	}
	return dedupeRoutes(nextFrontier), unavailable
}

func (p *Planner) applyTransform(snapshot buildcatalog.Snapshot, policy ProviderPolicy, target Target,
	current route, options []Option, stepID string) ([]route, bool) {
	results := []route{}
	unavailable := false
	for _, capability := range snapshot.Transform.Capabilities {
		if !catalogTargetsContain(capability.Targets, target) || !artifactSelectorMatches(
			fromCatalogArtifact(capability.Input, target), current.final) {
			continue
		}
		if capability.RequiresEntrySymbol && len(current.final.EntrySymbols) == 0 {
			continue
		}
		toolchain := fromCatalogToolchain(capability.Toolchain)
		output := fromCatalogArtifact(capability.Output, target)
		if output.Kind == current.final.Kind && output.Container == current.final.Container &&
			output.GoBuildMode == current.final.GoBuildMode && len(output.EntrySymbols) == 0 {
			output.EntrySymbols = slices.Clone(current.final.EntrySymbols)
		}
		provider, ok := p.chooseProvider(snapshot, policy, target, toolchain, artifactPointer(current.final), output, "", nil)
		if !ok {
			unavailable = true
			continue
		}
		step := Step{ID: stepID, Kind: "transform", Spec: planReference(snapshot.Reference), Provider: provider,
			Input: artifactPointer(current.final), Output: output, Toolchain: toolchain, Options: cloneOptions(options)}
		nextSteps := append(slices.Clone(current.steps), step)
		results = append(results, route{steps: nextSteps, final: output})
	}
	return dedupeRoutes(results), unavailable
}

//nolint:gocyclo // Keep diagnostic and live provider choices tied to the same provider policy.
func (p *Planner) chooseProvider(snapshot buildcatalog.Snapshot, policy ProviderPolicy, target Target,
	toolchain Toolchain, input *Artifact, output Artifact, mode string, protocols []string) (string, bool) {
	if p.ignoreAvailability {
		switch policy.Mode {
		case "pinned":
			return policy.Builder, true
		case "remote":
			return "unavailable-remote", true
		default:
			return "unavailable", true
		}
	}
	if !snapshot.Available || p.toolchain == nil {
		return "", false
	}
	providers := slices.Clone(snapshot.Providers)
	sort.Slice(providers, func(i, j int) bool {
		if providers[i].Name == "server" {
			return true
		}
		if providers[j].Name == "server" {
			return false
		}
		if p.providerRank != nil {
			leftMB, leftCPU := p.providerRank(providers[i].Name)
			rightMB, rightCPU := p.providerRank(providers[j].Name)
			if leftMB != rightMB {
				return leftMB > rightMB
			}
			if leftCPU != rightCPU {
				return leftCPU > rightCPU
			}
		}
		return providers[i].Name < providers[j].Name
	})
	for _, provider := range providers {
		if !provider.Online {
			continue
		}
		if policy.Mode == "remote" && provider.Name == "server" {
			continue
		}
		if policy.Mode == "pinned" && provider.Name != policy.Builder {
			continue
		}
		if p.toolchain(provider.Name, snapshot, target, toolchain) &&
			(p.providerCapability == nil || p.providerCapability(provider.Name, snapshot, target, input, output, toolchain, mode, protocols)) {
			return provider.Name, true
		}
	}
	return "", false
}

func routeMatchesFormat(candidate route, format string) bool {
	if candidate.final.Kind != finalKinds[format] {
		return false
	}
	if len(candidate.steps) == 1 {
		return slices.Contains(candidate.directFormats, format)
	}
	return true
}

func routeContainsArtifact(candidate route, artifact Artifact) bool {
	for _, step := range candidate.steps {
		if artifactEqual(step.Output, artifact) {
			return true
		}
	}
	return false
}

func routeContainsTransformInput(candidate route, reference buildcatalog.Reference, input Artifact) bool {
	for _, step := range candidate.steps {
		if step.Kind == "transform" && step.Spec == planReference(reference) && step.Input != nil &&
			artifactEqual(*step.Input, input) {
			return true
		}
	}
	return false
}

func dedupeRoutes(routes []route) []route {
	seen := map[string]int{}
	result := make([]route, 0, len(routes))
	for _, candidate := range routes {
		key := routeKey(candidate)
		if index, exists := seen[key]; exists {
			for _, format := range candidate.directFormats {
				if !slices.Contains(result[index].directFormats, format) {
					result[index].directFormats = append(result[index].directFormats, format)
				}
			}
			continue
		}
		seen[key] = len(result)
		result = append(result, candidate)
	}
	return result
}

func routeKey(candidate route) string {
	var parts []string
	for _, step := range candidate.steps {
		parts = append(parts, step.Kind+":"+referenceKey(catalogReference(step.Spec))+":"+step.Provider+":"+
			artifactKey(step.Output)+":"+step.Toolchain.Driver+":"+strings.Join(step.Toolchain.Requirements, ","))
	}
	return strings.Join(parts, "|")
}

func artifactKey(artifact Artifact) string {
	return artifact.Kind + "/" + artifact.Container + "/" + artifact.Target.OS + "/" + artifact.Target.Arch +
		"/" + artifact.GoBuildMode + "/" + strings.Join(artifact.EntrySymbols, ",")
}

func referenceKey(reference buildcatalog.Reference) string {
	return reference.Name + "/" + reference.Version + "/" + reference.Digest
}

func catalogTargetsContain(targets []buildcatalog.Target, target Target) bool {
	for _, candidate := range targets {
		if candidate.OS == target.OS && candidate.Arch == target.Arch {
			return true
		}
	}
	return false
}

func containsProtocols(supported, requested []string) bool {
	for _, protocol := range requested {
		if !slices.Contains(supported, protocol) {
			return false
		}
	}
	return true
}

func artifactSelectorMatches(selector, actual Artifact) bool {
	if selector.Kind != actual.Kind || selector.Container != "" && selector.Container != actual.Container ||
		selector.Target != actual.Target {
		return false
	}
	if selector.GoBuildMode != "" && selector.GoBuildMode != actual.GoBuildMode {
		return false
	}
	for _, symbol := range selector.EntrySymbols {
		if !slices.Contains(actual.EntrySymbols, symbol) {
			return false
		}
	}
	return true
}

func fromCatalogArtifact(artifact buildcatalog.Artifact, target Target) Artifact {
	return Artifact{Kind: artifact.Kind, Container: artifact.Container, Target: target,
		GoBuildMode: artifact.GoBuildMode, EntrySymbols: slices.Clone(artifact.EntrySymbols)}
}

func fromCatalogToolchain(toolchain buildcatalog.Toolchain) Toolchain {
	return Toolchain{Driver: toolchain.Driver, Requirements: slices.Clone(toolchain.Requirements)}
}

func planReference(reference buildcatalog.Reference) Reference {
	return Reference{Name: reference.Name, Version: reference.Version, Digest: reference.Digest}
}

func catalogReference(reference Reference) buildcatalog.Reference {
	return buildcatalog.Reference{Name: reference.Name, Version: reference.Version, Digest: reference.Digest}
}

func artifactPointer(value Artifact) *Artifact { return &value }

func cloneOptions(options []Option) []Option {
	result := make([]Option, len(options))
	for i, option := range options {
		result[i] = Option{Name: option.Name, ValueJSON: bytes.Clone(option.ValueJSON)}
	}
	return result
}

func resolveOptions(requested []Option, definitions []buildcatalog.OptionDefinition) ([]Option, error) {
	byName := map[string]buildcatalog.OptionDefinition{}
	for _, definition := range definitions {
		if _, exists := byName[definition.Name]; exists {
			return nil, fmt.Errorf("duplicate spec option %s", definition.Name)
		}
		byName[definition.Name] = definition
	}
	provided := map[string]Option{}
	for _, option := range requested {
		definition, exists := byName[option.Name]
		if !exists {
			return nil, fmt.Errorf("unknown option %s", option.Name)
		}
		if _, duplicate := provided[option.Name]; duplicate {
			return nil, fmt.Errorf("duplicate option %s", option.Name)
		}
		if err := validateOptionValue(option.ValueJSON, definition); err != nil {
			return nil, fmt.Errorf("option %s: %w", option.Name, err)
		}
		provided[option.Name] = option
	}
	names := make([]string, 0, len(definitions))
	for _, definition := range definitions {
		names = append(names, definition.Name)
		if _, exists := provided[definition.Name]; exists {
			continue
		}
		if len(definition.Default) != 0 {
			provided[definition.Name] = Option{Name: definition.Name, ValueJSON: bytes.Clone(definition.Default)}
		} else if definition.Required {
			return nil, fmt.Errorf("required option %s is missing", definition.Name)
		}
	}
	sort.Strings(names)
	result := make([]Option, 0, len(provided))
	for _, name := range names {
		if option, exists := provided[name]; exists {
			result = append(result, Option{Name: name, ValueJSON: bytes.Clone(option.ValueJSON)})
		}
	}
	return result, nil
}

func validateOptionValue(raw json.RawMessage, definition buildcatalog.OptionDefinition) error {
	if !json.Valid(raw) {
		return errors.New("value is not valid JSON")
	}
	if bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return errors.New("null is not a valid option value")
	}
	switch definition.Type {
	case "boolean":
		var value bool
		if err := json.Unmarshal(raw, &value); err != nil {
			return err
		}
	case "string", "enum":
		var value string
		if err := json.Unmarshal(raw, &value); err != nil {
			return err
		}
		if definition.Type == "enum" && !slices.Contains(definition.Values, value) {
			return errors.New("value is outside enum")
		}
	case "integer", "number":
		return validateNumericOption(raw, definition)
	case "string-list":
		var value []string
		if err := json.Unmarshal(raw, &value); err != nil {
			return err
		}
	case "secret-ref", "input-artifact-ref":
		return validateInputReference(raw, definition.Type)
	default:
		return fmt.Errorf("unsupported option type %s", definition.Type)
	}
	return nil
}

func validateNumericOption(raw json.RawMessage, definition buildcatalog.OptionDefinition) error {
	var value json.Number
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if err := decoder.Decode(&value); err != nil {
		return err
	}
	if definition.Type == "integer" {
		integer, err := strconv.ParseInt(value.String(), 10, 64)
		if err != nil {
			return errors.New("expected integer")
		}
		if integer < -9007199254740991 || integer > 9007199254740991 {
			return errors.New("integer exceeds the safe JSON range")
		}
	}
	floatValue, err := value.Float64()
	if err != nil {
		return err
	}
	if definition.Minimum != nil && floatValue < *definition.Minimum ||
		definition.Maximum != nil && floatValue > *definition.Maximum {
		return errors.New("value is outside allowed range")
	}
	return nil
}

func validateInputReference(raw json.RawMessage, optionType string) error {
	var value struct {
		Kind string `json:"kind"`
		Name string `json:"name"`
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&value); err != nil {
		return err
	}
	want := "secret"
	if optionType == "input-artifact-ref" {
		want = "input-artifact"
	}
	if value.Kind != want || value.Name == "" {
		return errors.New("invalid per-build input reference")
	}
	return nil
}

func planDigest(plan Plan) (string, error) {
	plan.SHA256 = ""
	data, err := json.Marshal(plan)
	if err != nil {
		return "", err
	}
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:]), nil
}

// Rehash updates the digest after execution-time remote capacity selection
// changes a provider, before the plan is stored as an immutable record.
func Rehash(plan *Plan) error {
	if plan == nil {
		return errors.New("missing build plan")
	}
	digest, err := planDigest(*plan)
	if err != nil {
		return err
	}
	plan.SHA256 = digest
	return nil
}
