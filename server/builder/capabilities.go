package builder

import (
	"crypto/rand"
	"errors"
	"fmt"
	"slices"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/server/assets"
	"github.com/bishopfox/sliver/server/buildcatalog"
	"github.com/bishopfox/sliver/server/buildcore"
	_ "github.com/bishopfox/sliver/server/buildcore/minimal" // Register the installed alternate core on workers.
	"github.com/bishopfox/sliver/server/buildplan"
	"github.com/bishopfox/sliver/server/generate"
)

// ConfigureInstalledCapabilities freezes the builder's supported spec and
// capability claims at registration. Capacity is reported separately and can
// change without importing or modifying a spec definition.
func ConfigureInstalledCapabilities(config *clientpb.Builder) error {
	if config == nil {
		return errors.New("missing builder configuration")
	}
	if len(config.SessionToken) != 0 {
		return errors.New("builder session token is already configured")
	}
	token := make([]byte, 32)
	if _, err := rand.Read(token); err != nil {
		return fmt.Errorf("create builder session token: %w", err)
	}
	catalog, err := buildcatalog.Open(assets.GetRootAppDir(), buildcore.InstalledExecutorIDs())
	if err != nil {
		return fmt.Errorf("open installed build catalog: %w", err)
	}
	var refs []*clientpb.BuildSpecRef
	var capabilities []*clientpb.BuilderCapability
	var yaml []string
	var toolchains []*clientpb.BuilderToolchainStatus
	for _, snapshot := range catalog.List(buildcatalog.Query{}) {
		if !locallyInstalledSpec(snapshot) {
			continue
		}
		ref := &clientpb.BuildSpecRef{
			Name: snapshot.Reference.Name, Version: snapshot.Reference.Version, Digest: snapshot.Reference.Digest,
		}
		claim := installedCapabilities(config, snapshot, ref)
		if len(claim) == 0 {
			continue
		}
		refs = append(refs, ref)
		yaml = append(yaml, string(snapshot.RawYAML))
		capabilities = append(capabilities, claim...)
		for _, advertised := range claim {
			capability := advertised.Capability
			for _, target := range capability.Targets {
				toolchains = append(toolchains, &clientpb.BuilderToolchainStatus{
					Driver: capability.ToolchainDriver, Target: target,
					Requirements: slices.Clone(capability.ToolchainRequirements),
					Identity:     toolchainIdentity(capability.ToolchainDriver), Available: true,
				})
			}
		}
	}
	if len(refs) == 0 {
		return errors.New("no installed build specs are executable by this builder")
	}
	config.SessionToken = token
	config.SpecRefs = refs
	config.SpecYAML = yaml
	config.Capabilities = capabilities
	config.Status = &clientpb.BuilderStatus{
		EffectiveCPUCores: effectiveCPUCores(), AvailableMemoryBytes: availableMemoryBytes(),
		Toolchains: currentToolchainStatuses(toolchains),
	}
	return nil
}

func locallyInstalledSpec(snapshot buildcatalog.Snapshot) bool {
	for _, origin := range snapshot.Sources {
		if origin.Source == "builtin" || origin.Source == "administrator" && origin.Provider == "server" {
			return true
		}
	}
	return false
}

//nolint:gocyclo // Keep each executor's eligibility checks beside the tuples it advertises.
func installedCapabilities(config *clientpb.Builder, snapshot buildcatalog.Snapshot,
	ref *clientpb.BuildSpecRef) []*clientpb.BuilderCapability {
	var result []*clientpb.BuilderCapability
	if snapshot.Core != nil {
		adapter, exists := buildcore.DefaultAdapter(snapshot.Core.Executor.ID)
		if !exists || !matchesBinding(snapshot.Core, adapter) {
			return nil
		}
		if snapshot.Reference.Name != "builtin/sliver" {
			if _, remote := adapter.(buildcore.RemoteCoreAdapter); !remote {
				return nil
			}
		}
		availability, exists := adapter.(buildcore.Availability)
		if !exists {
			return nil
		}
		for _, capability := range snapshot.Core.Capabilities {
			for _, target := range capability.Targets {
				toolchain := buildplan.Toolchain{Driver: capability.Toolchain.Driver, Requirements: capability.Toolchain.Requirements}
				if !availability.Supports(buildplan.Target{OS: target.OS, Arch: target.Arch}, toolchain) ||
					(snapshot.Reference.Name == "builtin/sliver" && !configuredCoreTarget(config.Targets, target, capability.Output.Artifact)) {
					continue
				}
				output := capability.Output.Artifact
				result = append(result, &clientpb.BuilderCapability{Spec: ref, Capability: &clientpb.BuildSpecCapability{
					Targets: []*clientpb.BuildTarget{{GOOS: target.OS, GOARCH: target.Arch}},
					Mode:    capability.Mode, Protocols: slices.Clone(capability.Protocols),
					Output: catalogArtifactType(output), OperatorFormats: slices.Clone(capability.Output.OperatorFormats),
					ToolchainDriver: toolchain.Driver, ToolchainRequirements: slices.Clone(toolchain.Requirements),
					EstimatedPeakMemoryBytes: peakMemory(capability.Resources),
				}})
			}
		}
		return result
	}
	if snapshot.Transform == nil || !installedTransform(snapshot.Reference.Name) {
		return nil
	}
	for _, capability := range snapshot.Transform.Capabilities {
		if capability.Toolchain.Driver != "builtin" || len(capability.Toolchain.Requirements) != 0 {
			continue
		}
		for _, target := range capability.Targets {
			result = append(result, &clientpb.BuilderCapability{Spec: ref, Capability: &clientpb.BuildSpecCapability{
				Targets: []*clientpb.BuildTarget{{GOOS: target.OS, GOARCH: target.Arch}},
				Input:   catalogArtifactType(capability.Input), Output: catalogArtifactType(capability.Output),
				ToolchainDriver:          capability.Toolchain.Driver,
				EstimatedPeakMemoryBytes: peakMemory(capability.Resources),
				RequiresEntrySymbol:      capability.RequiresEntrySymbol,
			}})
		}
	}
	return result
}

func matchesBinding(spec *buildcatalog.CoreSpec, adapter buildcore.Adapter) bool {
	if spec.Source == nil && spec.Builder == nil {
		return true
	}
	if spec.Source == nil || spec.Builder == nil || spec.Builder.Engine != "mage" {
		return false
	}
	bound, ok := adapter.(buildcore.MageBound)
	if !ok {
		return false
	}
	binding := bound.Binding()
	return binding.SourceMode == spec.Source.Mode && binding.RendererID == spec.Source.Renderer &&
		binding.Magefile == spec.Builder.Magefile && binding.Target == spec.Builder.Target
}

func configuredCoreTarget(targets []*clientpb.CompilerTarget, target buildcatalog.Target, output buildcatalog.Artifact) bool {
	format := clientpb.OutputFormat_EXECUTABLE
	switch output.Kind {
	case "service-executable":
		format = clientpb.OutputFormat_SERVICE
	case "shared-library", "go-plugin", "go-shared-library":
		format = clientpb.OutputFormat_SHARED_LIB
	case "c-archive-bundle", "c-archive", "go-archive":
		format = clientpb.OutputFormat_GO_ARCHIVE
	case "shellcode":
		format = clientpb.OutputFormat_SHELLCODE
	}
	for _, available := range targets {
		if available != nil && available.GOOS == target.OS && available.GOARCH == target.Arch && available.Format == format {
			return true
		}
	}
	return false
}

func catalogArtifactType(artifact buildcatalog.Artifact) *clientpb.BuildArtifactType {
	return &clientpb.BuildArtifactType{
		Kind: artifact.Kind, Container: artifact.Container,
		GoBuildMode: artifact.GoBuildMode, EntrySymbols: slices.Clone(artifact.EntrySymbols),
	}
}

func peakMemory(resources *buildcatalog.Resources) uint64 {
	if resources == nil {
		return 0
	}
	return resources.EstimatedPeakMemoryBytes
}

func installedTransform(name string) bool {
	switch name {
	case "builtin/donut", "builtin/beignet", "builtin/malasada", "builtin/shikata-ga-nai",
		"builtin/xor", "builtin/xor-dynamic", "builtin/stage-frame", "builtin/stage-compress",
		"builtin/stage-aes", "builtin/stage-rc4", "builtin/pe-metadata":
		return true
	default:
		return false
	}
}

func toolchainIdentity(driver string) string {
	if driver == "go" {
		return generate.SliverToolchainIdentity()
	}
	return "builtin"
}

// The registered tuple list is immutable, but installed toolchains can go
// away between reports (for example when a cross-compiler is removed).
func currentToolchainStatuses(declared []*clientpb.BuilderToolchainStatus) []*clientpb.BuilderToolchainStatus {
	result := make([]*clientpb.BuilderToolchainStatus, 0, len(declared))
	adapter := generate.SliverCoreAdapter{}
	for _, tool := range declared {
		if tool == nil || tool.Target == nil {
			continue
		}
		available := tool.Available
		if tool.Driver == "go" {
			available = adapter.Supports(buildplan.Target{OS: tool.Target.GOOS, Arch: tool.Target.GOARCH},
				buildplan.Toolchain{Driver: tool.Driver, Requirements: tool.Requirements})
		}
		result = append(result, &clientpb.BuilderToolchainStatus{
			Driver: tool.Driver, Target: &clientpb.BuildTarget{GOOS: tool.Target.GOOS, GOARCH: tool.Target.GOARCH},
			Requirements: slices.Clone(tool.Requirements), Identity: toolchainIdentity(tool.Driver), Available: available,
		})
	}
	return result
}
