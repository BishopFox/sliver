package rpc

import (
	"context"
	"strings"
	"testing"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestGetSystemRejectsMissingImplantConfig(t *testing.T) {
	_, err := (&Server{}).GetSystem(context.Background(), &clientpb.GetSystemReq{
		Request: &commonpb.Request{SessionID: "unused"},
	})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("GetSystem() error = %v, code = %s; want %s", err, status.Code(err), codes.InvalidArgument)
	}
	if !strings.Contains(err.Error(), "missing implant config") {
		t.Fatalf("GetSystem() error = %v, want missing implant config detail", err)
	}
}

func TestPrepareGetSystemFallbackConfigRejectsControlFlowWithoutDowngrade(t *testing.T) {
	config := &clientpb.ImplantConfig{
		ObfuscateSymbols: true,
		ControlFlow:      clientpb.ControlFlowPolicy_CONTROL_FLOW_BALANCED_V1,
		Format:           clientpb.OutputFormat_EXECUTABLE,
		IsSharedLib:      true,
		TemplateName:     "custom",
	}

	err := prepareGetSystemFallbackConfig(config)
	if status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("prepareGetSystemFallbackConfig() error = %v, code = %s; want %s", err, status.Code(err), codes.FailedPrecondition)
	}
	if !strings.Contains(err.Error(), "does not support control-flow-obfuscated helper builds") {
		t.Fatalf("prepareGetSystemFallbackConfig() error = %v, want control-flow detail", err)
	}
	if config.ControlFlow != clientpb.ControlFlowPolicy_CONTROL_FLOW_BALANCED_V1 || !config.ObfuscateSymbols {
		t.Fatalf("control-flow policy was downgraded: %+v", config)
	}
	if config.Format != clientpb.OutputFormat_EXECUTABLE || !config.IsSharedLib || config.TemplateName != "custom" {
		t.Fatalf("config was mutated before fallback rejection: %+v", config)
	}
}

func TestPrepareGetSystemFallbackConfigPreservesDisabledBehavior(t *testing.T) {
	config := &clientpb.ImplantConfig{
		ObfuscateSymbols: true,
		Format:           clientpb.OutputFormat_EXECUTABLE,
		IsSharedLib:      true,
	}

	if err := prepareGetSystemFallbackConfig(config); err != nil {
		t.Fatalf("prepareGetSystemFallbackConfig() error = %v", err)
	}
	if config.Format != clientpb.OutputFormat_SHELLCODE || config.ObfuscateSymbols || !config.IsShellcode || config.IsSharedLib {
		t.Fatalf("fallback config = %+v, want legacy shellcode settings", config)
	}
	if config.TemplateName != "sliver" {
		t.Fatalf("TemplateName = %q, want sliver", config.TemplateName)
	}
	if len(config.Exports) != 1 || config.Exports[0] != "StartW" {
		t.Fatalf("Exports = %q, want [StartW]", config.Exports)
	}
}
