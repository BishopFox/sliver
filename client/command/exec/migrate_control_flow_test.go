package exec

import (
	"testing"

	"github.com/bishopfox/sliver/protobuf/clientpb"
)

func TestApplyMigrateObfuscationPolicy(t *testing.T) {
	config := &clientpb.ImplantConfig{
		Debug:            true,
		ObfuscateSymbols: false,
		ControlFlow:      clientpb.ControlFlowPolicy_CONTROL_FLOW_DISABLED,
	}
	builds := &clientpb.ImplantBuilds{Configs: map[string]*clientpb.ImplantConfig{
		"source-build": {
			Debug:            false,
			ObfuscateSymbols: true,
			ControlFlow:      clientpb.ControlFlowPolicy_CONTROL_FLOW_BALANCED_V1,
		},
	}}

	if !applyMigrateObfuscationPolicy(config, "source-build", builds) {
		t.Fatal("applyMigrateObfuscationPolicy() = false, want true")
	}
	if config.Debug {
		t.Fatal("Debug = true, want originating build value false")
	}
	if !config.ObfuscateSymbols {
		t.Fatal("ObfuscateSymbols = false, want originating build value true")
	}
	if config.ControlFlow != clientpb.ControlFlowPolicy_CONTROL_FLOW_BALANCED_V1 {
		t.Fatalf("ControlFlow = %s, want %s", config.ControlFlow, clientpb.ControlFlowPolicy_CONTROL_FLOW_BALANCED_V1)
	}
}

func TestApplyMigrateObfuscationPolicyMissingBuildPreservesLegacyConfig(t *testing.T) {
	config := &clientpb.ImplantConfig{
		Debug:            true,
		ObfuscateSymbols: false,
		ControlFlow:      clientpb.ControlFlowPolicy_CONTROL_FLOW_DISABLED,
	}
	if applyMigrateObfuscationPolicy(
		config,
		"missing-build",
		&clientpb.ImplantBuilds{Configs: map[string]*clientpb.ImplantConfig{}},
	) {
		t.Fatal("applyMigrateObfuscationPolicy() = true for missing build, want false")
	}
	if !config.Debug || config.ObfuscateSymbols || config.ControlFlow != clientpb.ControlFlowPolicy_CONTROL_FLOW_DISABLED {
		t.Fatalf("missing build changed legacy config: %+v", config)
	}
}
