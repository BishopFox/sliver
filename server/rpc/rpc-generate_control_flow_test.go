package rpc

import (
	"context"
	"strings"
	"testing"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/server/core"
	"github.com/bishopfox/sliver/server/generate"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestGenerateExternalRejectsBuilderWithoutControlFlowCapability(t *testing.T) {
	builderName := "control-flow-incompatible-builder"
	if err := core.AddBuilder(&clientpb.Builder{
		Name:         builderName,
		OperatorName: "control-flow-test-operator",
	}); err != nil {
		t.Fatalf("register test builder: %v", err)
	}
	t.Cleanup(func() {
		core.RemoveBuilder(builderName)
	})

	rpc := &Server{}
	_, err := rpc.GenerateExternal(context.Background(), &clientpb.ExternalGenerateReq{
		Name:        "control-flow-capability-test",
		BuilderName: builderName,
		Config: &clientpb.ImplantConfig{
			ObfuscateSymbols: true,
			ControlFlow:      clientpb.ControlFlowPolicy_CONTROL_FLOW_BALANCED_V1,
			TemplateName:     "sliver",
		},
	})
	if status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("GenerateExternal() error = %v, code = %s; want %s", err, status.Code(err), codes.FailedPrecondition)
	}
	if !strings.Contains(err.Error(), "does not support garble.control-flow/balanced-v1") {
		t.Fatalf("GenerateExternal() error = %v, want control-flow capability detail", err)
	}
}

func TestGenerateRejectsInvalidControlFlowConfigAsInvalidArgument(t *testing.T) {
	_, err := (&Server{}).Generate(context.Background(), &clientpb.GenerateReq{
		Name: "control-flow-invalid-config",
		Config: &clientpb.ImplantConfig{
			Debug:            true,
			ObfuscateSymbols: true,
			ControlFlow:      clientpb.ControlFlowPolicy_CONTROL_FLOW_BALANCED_V1,
		},
	})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("Generate() error = %v, code = %s; want %s", err, status.Code(err), codes.InvalidArgument)
	}
}

func TestSaveImplantProfileRejectsInvalidControlFlowConfigAsInvalidArgument(t *testing.T) {
	_, err := (&Server{}).SaveImplantProfile(context.Background(), &clientpb.ImplantProfile{
		Name: "control-flow-invalid-profile",
		Config: &clientpb.ImplantConfig{
			ObfuscateSymbols: true,
			ControlFlow:      clientpb.ControlFlowPolicy(99),
		},
	})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("SaveImplantProfile() error = %v, code = %s; want %s", err, status.Code(err), codes.InvalidArgument)
	}
}

func TestGenerateRejectsConcurrentControlFlowBuild(t *testing.T) {
	config := &clientpb.ImplantConfig{
		ObfuscateSymbols: true,
		ControlFlow:      clientpb.ControlFlowPolicy_CONTROL_FLOW_BALANCED_V1,
		TemplateName:     "sliver",
	}
	release, err := generate.AcquireControlFlowBuildSlot(config)
	if err != nil {
		t.Fatalf("AcquireControlFlowBuildSlot() error = %v", err)
	}
	defer release()

	_, err = (&Server{}).Generate(context.Background(), &clientpb.GenerateReq{
		Name:   "control-flow-contended-build",
		Config: config,
	})
	if status.Code(err) != codes.ResourceExhausted {
		t.Fatalf("Generate() error = %v, code = %s; want %s", err, status.Code(err), codes.ResourceExhausted)
	}
}

func TestGetCompilerAdvertisesProtocolAndVerifiedLocalControlFlowCapabilities(t *testing.T) {
	compiler, err := (&Server{}).GetCompiler(context.Background(), &commonpb.Empty{})
	if err != nil {
		t.Fatalf("GetCompiler() error = %v", err)
	}
	if !generate.HasControlFlowCapability(compiler.Capabilities) {
		t.Fatalf("GetCompiler() capabilities = %q, want %q", compiler.Capabilities, generate.ControlFlowCapability)
	}
	if !generate.HasLocalControlFlowCapability(compiler.Capabilities) {
		t.Fatalf("GetCompiler() capabilities = %q, want %q", compiler.Capabilities, generate.LocalControlFlowCapability)
	}
}
