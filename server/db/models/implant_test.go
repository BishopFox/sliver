package models

import "testing"

func TestImplantConfigToProtobufRunAtLoad(t *testing.T) {
	ic := &ImplantConfig{RunAtLoad: true}

	pb := ic.ToProtobuf()
	if pb.RunAtLoad != ic.RunAtLoad {
		t.Fatalf("expected RunAtLoad=%v, got %v", ic.RunAtLoad, pb.RunAtLoad)
	}
}
