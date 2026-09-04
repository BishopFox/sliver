package models

import (
	"testing"

	"github.com/bishopfox/sliver/protobuf/clientpb"
)

func TestImplantConfigControlFlowRoundTrip(t *testing.T) {
	policies := []clientpb.ControlFlowPolicy{
		clientpb.ControlFlowPolicy_CONTROL_FLOW_DISABLED,
		clientpb.ControlFlowPolicy_CONTROL_FLOW_BALANCED_V1,
	}

	for _, policy := range policies {
		t.Run(policy.String(), func(t *testing.T) {
			model := ImplantConfigFromProtobuf(&clientpb.ImplantConfig{ControlFlow: policy})
			if model.ControlFlow != policy {
				t.Fatalf("model ControlFlow = %v, want %v", model.ControlFlow, policy)
			}

			got := model.ToProtobuf().ControlFlow
			if got != policy {
				t.Fatalf("protobuf ControlFlow = %v, want %v", got, policy)
			}
		})
	}
}

func TestImplantConfigBeforeCreatePreservesAssignedID(t *testing.T) {
	assignedID := NewUUID()
	config := &ImplantConfig{ID: assignedID}
	if err := config.BeforeCreate(nil); err != nil {
		t.Fatalf("BeforeCreate() error = %v", err)
	}
	if config.ID != assignedID {
		t.Fatalf("BeforeCreate() ID = %s, want assigned ID %s", config.ID, assignedID)
	}
}
