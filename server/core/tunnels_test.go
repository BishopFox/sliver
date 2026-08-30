package core

import (
	"sync"
	"testing"
	"time"

	"github.com/bishopfox/sliver/protobuf/rpcpb"
)

type testTunnelDataServer struct {
	rpcpb.SliverRPC_TunnelDataServer
}

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

func TestTunnelsCloseUnboundTunnelDoesNotBlock(t *testing.T) {
	tunnel := NewTunnel(1, "test-session")
	tunnels := &tunnels{
		tunnels: map[uint64]*Tunnel{tunnel.ID: tunnel},
		mutex:   &sync.Mutex{},
	}

	result := make(chan error, 1)
	go func() {
		result <- tunnels.Close(tunnel.ID)
	}()

	select {
	case err := <-result:
		if err != nil {
			t.Fatalf("unexpected close error: %v", err)
		}
	case <-time.After(time.Second):
		// Unblock the legacy in-band close send so a failing test does not
		// leave a goroutine holding the local tunnel registry mutex.
		go func() {
			<-tunnel.ToImplant
		}()
		select {
		case <-result:
		case <-time.After(time.Second):
		}
		t.Fatal("closing an unbound tunnel blocked")
	}

	if got := tunnels.Get(tunnel.ID); got != nil {
		t.Fatalf("closed tunnel remains registered: %+v", got)
	}
	select {
	case <-tunnel.Done():
	default:
		t.Fatal("closed tunnel did not signal completion")
	}
}

func TestTunnelSignalsClientBoundAfterAcknowledgement(t *testing.T) {
	tunnel := NewTunnel(1, "test-session")
	client := &testTunnelDataServer{}
	otherClient := &testTunnelDataServer{}

	if !tunnel.BindClient(client) {
		t.Fatal("first client failed to reserve tunnel")
	}
	if tunnel.BindClient(otherClient) {
		t.Fatal("second client replaced tunnel reservation")
	}
	select {
	case <-tunnel.ClientBound():
		t.Fatal("tunnel reported bound before acknowledgement")
	default:
	}
	if tunnel.MarkClientBound(otherClient) {
		t.Fatal("non-owning client acknowledged tunnel")
	}
	if !tunnel.MarkClientBound(client) {
		t.Fatal("owning client failed to acknowledge tunnel")
	}
	select {
	case <-tunnel.ClientBound():
	default:
		t.Fatal("tunnel did not report acknowledged client binding")
	}
}

func TestTunnelCloseUnblocksSenders(t *testing.T) {
	tunnel := NewTunnel(1, "test-session")

	fromStarted := make(chan struct{})
	fromResult := make(chan bool, 1)
	go func() {
		close(fromStarted)
		fromResult <- tunnel.SendDataFromImplant(nil)
	}()

	toStarted := make(chan struct{})
	toResult := make(chan bool, 1)
	go func() {
		close(toStarted)
		toResult <- tunnel.SendDataToImplant(nil)
	}()

	<-fromStarted
	<-toStarted
	tunnel.Close()

	for name, result := range map[string]<-chan bool{
		"from implant": fromResult,
		"to implant":   toResult,
	} {
		select {
		case sent := <-result:
			if sent {
				t.Errorf("%s sender reported success after close", name)
			}
		case <-time.After(time.Second):
			t.Errorf("%s sender remained blocked after close", name)
		}
	}
}

func TestTunnelsCloseForClientOnlyClosesOwnedTunnels(t *testing.T) {
	client := &testTunnelDataServer{}
	otherClient := &testTunnelDataServer{}
	owned := NewTunnel(11, "session")
	other := NewTunnel(12, "session")
	if !owned.BindClient(client) || !other.BindClient(otherClient) {
		t.Fatal("bind test clients")
	}
	tunnels := &tunnels{
		tunnels: map[uint64]*Tunnel{owned.ID: owned, other.ID: other},
		mutex:   &sync.Mutex{},
	}

	tunnels.CloseForClient(client)
	if got := tunnels.Get(owned.ID); got != nil {
		t.Fatal("stream-owned tunnel remained registered")
	}
	select {
	case <-owned.Done():
	default:
		t.Fatal("stream-owned tunnel was not closed")
	}
	if got := tunnels.Get(other.ID); got != other {
		t.Fatal("unrelated client's tunnel was removed")
	}
	select {
	case <-other.Done():
		t.Fatal("unrelated client's tunnel was closed")
	default:
	}
}

func TestTunnelClientDataRefreshesCloseActivity(t *testing.T) {
	tunnel := NewTunnel(13, "session")
	before := tunnel.GetLastMessageTime()
	time.Sleep(time.Millisecond)
	received := make(chan []byte, 1)
	go func() { received <- <-tunnel.ToImplant }()
	payload := []byte("exit\n")
	if !tunnel.SendDataToImplant(payload) {
		t.Fatal("client data was not queued")
	}
	if got := <-received; string(got) != string(payload) {
		t.Fatalf("queued data = %q, want %q", got, payload)
	}
	if after := tunnel.GetLastMessageTime(); !after.After(before) {
		t.Fatalf("client data did not refresh close activity: before=%v after=%v", before, after)
	}
}

func TestTunnelTouchRefreshesCloseActivity(t *testing.T) {
	tunnel := NewTunnel(14, "session")
	tunnel.mutex.Lock()
	tunnel.lastDataMessageTime = time.Now().Add(-2 * delayBeforeClose)
	tunnel.mutex.Unlock()

	tunnel.Touch()
	if age := time.Since(tunnel.GetLastMessageTime()); age < 0 || age >= delayBeforeClose {
		t.Fatalf("Touch did not establish a fresh close grace period: age=%v", age)
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
