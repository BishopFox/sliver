package rpc

import (
	"context"
	"path/filepath"
	"sync"
	"time"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/server/assets"
	"github.com/bishopfox/sliver/server/buildcatalog"
	"github.com/bishopfox/sliver/server/buildcore"
	_ "github.com/bishopfox/sliver/server/buildcore/minimal" // Register the installed alternate core on the server.
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type buildCatalogEntry struct {
	once    sync.Once
	catalog *buildcatalog.Catalog
	err     error
}

var buildCatalogs sync.Map // absolute app root -> *buildCatalogEntry

// buildCatalog provides one mutable catalog per server app root. This also
// lets tests use an isolated SLIVER_ROOT_DIR without sharing catalog state.
func (rpc *Server) buildCatalog() (*buildcatalog.Catalog, error) {
	root, err := filepath.Abs(assets.GetRootAppDir())
	if err != nil {
		return nil, err
	}
	value, _ := buildCatalogs.LoadOrStore(root, &buildCatalogEntry{})
	entry := value.(*buildCatalogEntry)
	entry.once.Do(func() { entry.catalog, entry.err = buildcatalog.Open(root, buildcore.InstalledExecutorIDs()) })
	return entry.catalog, entry.err
}

func (rpc *Server) ListBuildSpecs(_ context.Context, req *clientpb.BuildSpecQuery) (*clientpb.BuildSpecList, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "missing build spec query")
	}
	kind, err := catalogKind(req.Kind)
	if err != nil {
		return nil, err
	}
	if req.Availability < clientpb.BuildSpecAvailability_BUILD_SPEC_ALL || req.Availability > clientpb.BuildSpecAvailability_BUILD_SPEC_UNAVAILABLE {
		return nil, status.Error(codes.InvalidArgument, "invalid availability filter")
	}
	if (req.TargetGOOS == "") != (req.TargetGOARCH == "") {
		return nil, status.Error(codes.InvalidArgument, "target filter requires both GOOS and GOARCH")
	}
	catalog, err := rpc.buildCatalog()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "open build catalog: %v", err)
	}
	query := buildcatalog.Query{
		Kind: kind, Name: req.Name, TargetOS: req.TargetGOOS, TargetArch: req.TargetGOARCH,
		AvailableOnly: req.Availability == clientpb.BuildSpecAvailability_BUILD_SPEC_AVAILABLE,
	}
	response := &clientpb.BuildSpecList{}
	for _, snapshot := range catalog.List(query) {
		if req.Availability == clientpb.BuildSpecAvailability_BUILD_SPEC_UNAVAILABLE && snapshot.Available {
			continue
		}
		response.Specs = append(response.Specs, buildSpecProto(snapshot))
	}
	return response, nil
}

func (rpc *Server) GetBuildSpec(_ context.Context, req *clientpb.BuildSpecRequest) (*clientpb.BuildSpec, error) {
	if req == nil || req.Ref == nil || req.Ref.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "build spec name is required")
	}
	kind, err := catalogKind(req.Kind)
	if err != nil {
		return nil, err
	}
	if kind == "" {
		return nil, status.Error(codes.InvalidArgument, "build spec kind is required")
	}
	catalog, err := rpc.buildCatalog()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "open build catalog: %v", err)
	}
	var match *buildcatalog.Snapshot
	for _, snapshot := range catalog.List(buildcatalog.Query{Kind: kind, Name: req.Ref.Name}) {
		if req.Ref.Version != "" && snapshot.Reference.Version != req.Ref.Version {
			continue
		}
		if req.Ref.Digest != "" && snapshot.Reference.Digest != req.Ref.Digest {
			continue
		}
		if match != nil {
			return nil, status.Error(codes.FailedPrecondition, "build spec reference is ambiguous; specify version and digest")
		}
		copy := snapshot
		match = &copy
	}
	if match == nil {
		return nil, status.Error(codes.NotFound, "build spec not found")
	}
	return buildSpecProto(*match), nil
}

func catalogKind(kind clientpb.BuildSpecKind) (string, error) {
	switch kind {
	case clientpb.BuildSpecKind_BUILD_SPEC_UNSPECIFIED:
		return "", nil
	case clientpb.BuildSpecKind_BUILD_SPEC_CORE:
		return "CoreSpec", nil
	case clientpb.BuildSpecKind_BUILD_SPEC_TRANSFORM:
		return "TransformSpec", nil
	default:
		return "", status.Error(codes.InvalidArgument, "invalid build spec kind")
	}
}

func buildSpecProto(snapshot buildcatalog.Snapshot) *clientpb.BuildSpec {
	spec := &clientpb.BuildSpec{
		Ref:       &clientpb.BuildSpecRef{Name: snapshot.Reference.Name, Version: snapshot.Reference.Version, Digest: snapshot.Reference.Digest},
		Available: snapshot.Available,
		YAML:      append([]byte(nil), snapshot.RawYAML...),
	}
	switch snapshot.Kind {
	case "CoreSpec":
		spec.Kind = clientpb.BuildSpecKind_BUILD_SPEC_CORE
		if snapshot.Core != nil {
			spec.ExecutorID = snapshot.Core.Executor.ID
			spec.RuntimeProtocol = snapshot.Core.Runtime.Protocol
			spec.CommandSource = snapshot.Core.Runtime.CommandSource
			spec.RuntimeCommands = append([]string(nil), snapshot.Core.Runtime.Commands...)
			for _, capability := range snapshot.Core.Capabilities {
				result := &clientpb.BuildSpecCapability{
					Mode:                  capability.Mode,
					Protocols:             append([]string(nil), capability.Protocols...),
					Output:                catalogArtifactProto(capability.Output.Artifact),
					OperatorFormats:       append([]string(nil), capability.Output.OperatorFormats...),
					ToolchainDriver:       capability.Toolchain.Driver,
					ToolchainRequirements: append([]string(nil), capability.Toolchain.Requirements...),
				}
				for _, target := range capability.Targets {
					result.Targets = append(result.Targets, &clientpb.BuildTarget{GOOS: target.OS, GOARCH: target.Arch})
				}
				if capability.Resources != nil {
					result.EstimatedPeakMemoryBytes = capability.Resources.EstimatedPeakMemoryBytes
				}
				spec.Capabilities = append(spec.Capabilities, result)
			}
		}
	case "TransformSpec":
		spec.Kind = clientpb.BuildSpecKind_BUILD_SPEC_TRANSFORM
		if snapshot.Transform != nil {
			spec.ExecutorID = snapshot.Transform.Executor.ID
			for _, capability := range snapshot.Transform.Capabilities {
				result := &clientpb.BuildSpecCapability{
					Input:                 catalogArtifactProto(capability.Input),
					Output:                catalogArtifactProto(capability.Output),
					ToolchainDriver:       capability.Toolchain.Driver,
					ToolchainRequirements: append([]string(nil), capability.Toolchain.Requirements...),
					RequiresEntrySymbol:   capability.RequiresEntrySymbol,
				}
				for _, target := range capability.Targets {
					result.Targets = append(result.Targets, &clientpb.BuildTarget{GOOS: target.OS, GOARCH: target.Arch})
				}
				if capability.Resources != nil {
					result.EstimatedPeakMemoryBytes = capability.Resources.EstimatedPeakMemoryBytes
				}
				spec.Capabilities = append(spec.Capabilities, result)
			}
		}
	}
	for _, origin := range snapshot.Sources {
		spec.Origins = append(spec.Origins, &clientpb.BuildSpecOrigin{Source: origin.Source, Provider: origin.Provider, LastSeenUnix: unixOrZero(origin.LastSeen)})
	}
	for _, provider := range snapshot.Providers {
		spec.Providers = append(spec.Providers, &clientpb.BuildSpecProvider{Name: provider.Name, Online: provider.Online, LastSeenUnix: unixOrZero(provider.LastSeen)})
	}
	return spec
}

func catalogArtifactProto(artifact buildcatalog.Artifact) *clientpb.BuildArtifactType {
	return &clientpb.BuildArtifactType{
		Kind: artifact.Kind, Container: artifact.Container, GoBuildMode: artifact.GoBuildMode,
		EntrySymbols: append([]string(nil), artifact.EntrySymbols...),
	}
}

func unixOrZero(value time.Time) int64 {
	if value.IsZero() {
		return 0
	}
	return value.Unix()
}
