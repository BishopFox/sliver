package rpc

import (
	"testing"

	"github.com/bishopfox/sliver/protobuf/clientpb"
)

func TestMigrateCachedPolicyCompatible(t *testing.T) {
	disabled := &clientpb.ImplantConfig{ControlFlow: clientpb.ControlFlowPolicy_CONTROL_FLOW_DISABLED}
	balanced := &clientpb.ImplantConfig{ControlFlow: clientpb.ControlFlowPolicy_CONTROL_FLOW_BALANCED_V1}

	if !migrateCachedPolicyCompatible(nil, disabled) {
		t.Fatal("unknown originating policy should preserve legacy cache behavior")
	}
	if !migrateCachedPolicyCompatible(balanced, balanced) {
		t.Fatal("matching control-flow policies should permit cache reuse")
	}
	if migrateCachedPolicyCompatible(balanced, disabled) {
		t.Fatal("balanced originating policy accepted a disabled cached build")
	}
	if migrateCachedPolicyCompatible(disabled, balanced) {
		t.Fatal("disabled originating policy accepted a balanced cached build")
	}
	if migrateCachedPolicyCompatible(balanced, nil) {
		t.Fatal("known originating policy accepted a cache entry without config metadata")
	}
}
