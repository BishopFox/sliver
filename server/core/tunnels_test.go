package core

import (
	"testing"
)

func newTestSession() *Session {
	conn := NewImplantConnection("mtls", "test-conn")
	return NewSession(conn)
}

func TestTunnelsCreateUnknownSession(t *testing.T) {
	tunnel, err := Tunnels.Create("does-not-exist")
	if err == nil {
		t.Fatalf("expected error for unknown session, got tunnel %+v", tunnel)
	}
	if tunnel != nil {
		t.Fatalf("expected nil tunnel on error, got %+v", tunnel)
	}
}

func TestTunnelsCreateKnownSession(t *testing.T) {
	session := newTestSession()
	Sessions.Add(session)
	defer Sessions.Remove(session.ID)

	tunnel, err := Tunnels.Create(session.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tunnel == nil {
		t.Fatal("expected non-nil tunnel")
	}
	if tunnel.SessionID != session.ID {
		t.Fatalf("expected SessionID %q, got %q", session.ID, tunnel.SessionID)
	}
}

func TestSocksTunnelsCreateUnknownSession(t *testing.T) {
	tunnel, err := SocksTunnels.Create("does-not-exist")
	if err == nil {
		t.Fatalf("expected error for unknown session, got tunnel %+v", tunnel)
	}
	if tunnel != nil {
		t.Fatalf("expected nil tunnel on error, got %+v", tunnel)
	}
}

func TestSocksTunnelsCreateKnownSession(t *testing.T) {
	session := newTestSession()
	Sessions.Add(session)
	defer Sessions.Remove(session.ID)

	tunnel, err := SocksTunnels.Create(session.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tunnel == nil {
		t.Fatal("expected non-nil tunnel")
	}
	if tunnel.SessionID != session.ID {
		t.Fatalf("expected SessionID %q, got %q", session.ID, tunnel.SessionID)
	}
}

func TestPivotGraphEntryToProtobufUnknownSession(t *testing.T) {
	entry := &PivotGraphEntry{
		PeerID:    1,
		SessionID: "does-not-exist",
		Name:      "ghost",
		Children:  map[int64]*PivotGraphEntry{},
	}
	pb := entry.ToProtobuf()
	if pb == nil {
		t.Fatal("expected non-nil protobuf entry")
	}
	if pb.Session != nil {
		t.Fatalf("expected nil session for unknown session, got %+v", pb.Session)
	}
	if pb.PeerID != 1 || pb.Name != "ghost" {
		t.Fatalf("unexpected entry fields: %+v", pb)
	}
}
