package rpc

import (
	"context"
	"errors"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/server/buildcatalog"
	"github.com/bishopfox/sliver/server/buildcore"
	"github.com/bishopfox/sliver/server/buildplan"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ExecuteBuild resolves a neutral route and runs each local step in order.
// Successful records contain observed metadata for every intermediate and the
// exact final bytes returned by GetBuildArtifact.
func (rpc *Server) ExecuteBuild(ctx context.Context, req *clientpb.BuildRequest) (*clientpb.BuildRecord, error) {
	neutral, err := buildRequestFromProto(req)
	if err != nil {
		return nil, err
	}
	resolveInput, releaseInputs, err := buildInputResolver(req.GetInputs())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	defer releaseInputs()
	plan, err := rpc.resolveBuildPlan(ctx, neutral)
	if err != nil {
		return nil, buildPlanRPCError(err)
	}
	var adapter buildcore.Adapter
	if plan.Steps[0].Provider == "server" {
		adapter, err = rpc.coreAdapterForPlan(plan)
		if err != nil {
			return nil, buildPlanRPCError(err)
		}
	}
	_, record, err := rpc.executeResolvedPlan(ctx, plan, adapter, nil, resolveInput)
	if err != nil {
		if ctx.Err() != nil {
			return nil, status.FromContextError(ctx.Err()).Err()
		}
		var planErr *buildplan.PlanError
		if errors.As(err, &planErr) {
			return nil, buildPlanRPCError(err)
		}
		if record.ID == "" {
			return nil, status.Errorf(codes.Internal, "record build execution: %v", err)
		}
		return buildRecordProto(record), nil
	}
	return buildRecordProto(record), nil
}

func (rpc *Server) coreAdapterForPlan(plan buildplan.Plan) (buildcore.Adapter, error) {
	unsupported := func(detail string) error {
		return &buildplan.PlanError{Code: buildplan.CodeUnsupportedExecution, Step: "core", Detail: detail}
	}
	if len(plan.Steps) == 0 || plan.Steps[0].Kind != "core" || plan.Steps[0].Provider != "server" {
		return nil, unsupported("the selected core provider has no local execution adapter")
	}
	catalog, err := rpc.buildCatalog()
	if err != nil {
		return nil, err
	}
	ref := plan.Steps[0].Spec
	snapshot, err := catalog.Get(buildcatalog.Reference{Name: ref.Name, Version: ref.Version, Digest: ref.Digest})
	if err != nil {
		return nil, err
	}
	if snapshot.Core == nil {
		return nil, unsupported("the selected spec is not a core")
	}
	adapter, ok := buildcore.DefaultAdapter(snapshot.Core.Executor.ID)
	if !ok {
		return nil, unsupported("the selected core has no installed execution adapter")
	}
	if !matchesInstalledMageBinding(snapshot.Core, adapter) {
		return nil, unsupported("the selected core's Magefile or renderer differs from its installed integration")
	}
	return adapter, nil
}

func matchesInstalledMageBinding(spec *buildcatalog.CoreSpec, adapter buildcore.Adapter) bool {
	if spec == nil {
		return false
	}
	if spec.Source == nil && spec.Builder == nil {
		return true // immutable pre-Mage specs use their installed legacy adapter.
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
