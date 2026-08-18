package generate

import (
	"testing"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/gofrs/uuid"
)

func TestNewImplantConfigSnapshotDetachesAndPreservesPolicy(t *testing.T) {
	profileID := uuid.Must(uuid.NewV4()).String()
	configID := uuid.Must(uuid.NewV4()).String()
	c2ID := uuid.Must(uuid.NewV4()).String()
	source := &clientpb.ImplantConfig{
		ID:               configID,
		ImplantProfileID: profileID,
		ImplantBuilds:    []*clientpb.ImplantBuild{{Name: "old-build"}},
		ObfuscateSymbols: true,
		ControlFlow:      clientpb.ControlFlowPolicy_CONTROL_FLOW_BALANCED_V1,
		C2: []*clientpb.ImplantC2{
			{ID: c2ID, URL: "mtls://127.0.0.1:8888"},
			nil,
		},
	}

	snapshot, sourceProfileID, err := NewImplantConfigSnapshot(source)
	if err != nil {
		t.Fatalf("NewImplantConfigSnapshot() error = %v", err)
	}
	if sourceProfileID != profileID {
		t.Fatalf("source profile ID = %q, want %q", sourceProfileID, profileID)
	}
	if snapshot.ID == "" || snapshot.ID == configID {
		t.Fatalf("snapshot ID = %q, want a fresh UUID", snapshot.ID)
	}
	if uuid.FromStringOrNil(snapshot.ID) == uuid.Nil {
		t.Fatalf("snapshot ID = %q, want a valid UUID", snapshot.ID)
	}
	if snapshot.ImplantProfileID != "" {
		t.Fatalf("snapshot profile ID = %q, want detached config", snapshot.ImplantProfileID)
	}
	if snapshot.ImplantBuilds != nil {
		t.Fatalf("snapshot builds = %v, want nil", snapshot.ImplantBuilds)
	}
	if snapshot.C2[0].ID != "" {
		t.Fatalf("snapshot C2 ID = %q, want fresh persistence ID", snapshot.C2[0].ID)
	}
	if snapshot.ControlFlow != clientpb.ControlFlowPolicy_CONTROL_FLOW_BALANCED_V1 {
		t.Fatalf("snapshot control-flow policy = %v, want balanced-v1", snapshot.ControlFlow)
	}

	source.ControlFlow = clientpb.ControlFlowPolicy_CONTROL_FLOW_DISABLED
	source.C2[0].URL = "mtls://127.0.0.1:9999"
	if snapshot.ControlFlow != clientpb.ControlFlowPolicy_CONTROL_FLOW_BALANCED_V1 {
		t.Fatal("mutating the source profile changed the build snapshot policy")
	}
	if snapshot.C2[0].URL != "mtls://127.0.0.1:8888" {
		t.Fatal("mutating the source profile changed the build snapshot C2")
	}
	if source.ID != configID || source.ImplantProfileID != profileID || source.C2[0].ID != c2ID {
		t.Fatal("snapshot creation mutated the source profile config")
	}

	second, _, err := NewImplantConfigSnapshot(source)
	if err != nil {
		t.Fatalf("second NewImplantConfigSnapshot() error = %v", err)
	}
	if second.ID == snapshot.ID {
		t.Fatalf("two build snapshots reused config ID %q", snapshot.ID)
	}
}

func TestNewImplantConfigSnapshotRejectsInvalidSourceProfileID(t *testing.T) {
	_, _, err := NewImplantConfigSnapshot(&clientpb.ImplantConfig{ImplantProfileID: "not-a-uuid"})
	if err == nil {
		t.Fatal("NewImplantConfigSnapshot() accepted an invalid source profile ID")
	}
}
