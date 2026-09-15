package models

import "testing"

func TestBeaconToProtobufCapabilities(t *testing.T) {
	beacon := &Beacon{Capabilities: 42}

	pb := beacon.ToProtobuf()
	if pb.Capabilities != beacon.Capabilities {
		t.Fatalf("expected Capabilities=%d, got %d", beacon.Capabilities, pb.Capabilities)
	}
}
