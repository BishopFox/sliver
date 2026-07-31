package core

import "testing"

func TestSessionToProtobufNilConnection(t *testing.T) {
	s := &Session{Name: "test-canary"}

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("ToProtobuf panicked with nil Connection: %v", r)
		}
	}()

	pb := s.ToProtobuf()
	if pb.Name != "test-canary" {
		t.Fatalf("expected Name=%q, got %q", "test-canary", pb.Name)
	}
	if pb.Transport != "" {
		t.Fatalf("expected empty Transport, got %q", pb.Transport)
	}
}
