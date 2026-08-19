package privilege

import (
	"testing"

	"github.com/bishopfox/sliver/protobuf/clientpb"
)

func TestApplyGetSystemObfuscationPolicy(t *testing.T) {
	config := &clientpb.ImplantConfig{Debug: true}
	builds := &clientpb.ImplantBuilds{Configs: map[string]*clientpb.ImplantConfig{
		"source-build": {
			Debug:            false,
			ObfuscateSymbols: true,
			ControlFlow:      clientpb.ControlFlowPolicy_CONTROL_FLOW_BALANCED_V1,
		},
	}}

	if !applyGetSystemObfuscationPolicy(config, "source-build", builds) {
		t.Fatal("applyGetSystemObfuscationPolicy() = false, want true")
	}
	if config.Debug || !config.ObfuscateSymbols || config.ControlFlow != clientpb.ControlFlowPolicy_CONTROL_FLOW_BALANCED_V1 {
		t.Fatalf("obfuscation policy = %+v, want originating build values", config)
	}
}

func TestApplyGetSystemObfuscationPolicyMissingBuildPreservesLegacyConfig(t *testing.T) {
	config := &clientpb.ImplantConfig{Debug: true}
	if applyGetSystemObfuscationPolicy(config, "missing-build", &clientpb.ImplantBuilds{}) {
		t.Fatal("applyGetSystemObfuscationPolicy() = true for missing build, want false")
	}
	if !config.Debug || config.ObfuscateSymbols || config.ControlFlow != clientpb.ControlFlowPolicy_CONTROL_FLOW_DISABLED {
		t.Fatalf("missing build changed legacy config: %+v", config)
	}
}
