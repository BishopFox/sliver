package handlers

import (
	"testing"

	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/bishopfox/sliver/server/core"
	"github.com/bishopfox/sliver/server/core/rtunnels"
	"google.golang.org/protobuf/proto"
)

type testWriteCloser struct {
	closed bool
}

func (t *testWriteCloser) Write(data []byte) (int, error) {
	return len(data), nil
}

func (t *testWriteCloser) Close() error {
	t.closed = true
	return nil
}

func addTestSession(t *testing.T) (*core.ImplantConnection, *core.Session) {
	t.Helper()

	conn := core.NewImplantConnection("test", "n/a")
	session := core.NewSession(conn)
	core.Sessions.Add(session)
	t.Cleanup(func() {
		core.Sessions.Remove(session.ID)
	})

	return conn, session
}

func marshalTunnelCloseData(t *testing.T, tunnelID uint64) []byte {
	t.Helper()

	data, err := proto.Marshal(&sliverpb.TunnelData{
		TunnelID: tunnelID,
		Closed:   true,
	})
	if err != nil {
		t.Fatalf("marshal tunnel close data: %v", err)
	}

	return data
}

func assertNoPanic(t *testing.T, fn func()) {
	t.Helper()

	defer func() {
		if recovered := recover(); recovered != nil {
			t.Fatalf("unexpected panic: %v", recovered)
		}
	}()

	fn()
}

func TestTunnelCloseHandlerClosesOwnedReverseTunnel(t *testing.T) {
	conn, session := addTestSession(t)
	tunnelID := core.NewTunnelID()
	writer := &testWriteCloser{}

	rtunnels.AddRTunnel(rtunnels.NewRTunnel(tunnelID, session.ID, writer))
	t.Cleanup(func() {
		rtunnels.RemoveRTunnel(tunnelID)
	})

	assertNoPanic(t, func() {
		tunnelCloseHandler(conn, marshalTunnelCloseData(t, tunnelID))
	})

	if rtunnel := rtunnels.GetRTunnel(tunnelID); rtunnel != nil {
		t.Fatalf("expected reverse tunnel %d to be removed", tunnelID)
	}
	if !writer.closed {
		t.Fatal("expected reverse tunnel writer to be closed")
	}
}

func TestTunnelCloseHandlerKeepsUnownedReverseTunnel(t *testing.T) {
	ownerConn, ownerSession := addTestSession(t)
	_ = ownerConn
	attackerConn, attackerSession := addTestSession(t)
	tunnelID := core.NewTunnelID()
	writer := &testWriteCloser{}

	rtunnels.AddRTunnel(rtunnels.NewRTunnel(tunnelID, ownerSession.ID, writer))
	t.Cleanup(func() {
		rtunnels.RemoveRTunnel(tunnelID)
	})

	assertNoPanic(t, func() {
		tunnelCloseHandler(attackerConn, marshalTunnelCloseData(t, tunnelID))
	})

	rtunnel := rtunnels.GetRTunnel(tunnelID)
	if rtunnel == nil {
		t.Fatalf("expected reverse tunnel %d to remain for owner %s", tunnelID, ownerSession.ID)
	}
	if rtunnel.SessionID != ownerSession.ID {
		t.Fatalf("expected reverse tunnel owner %s, got %s", ownerSession.ID, rtunnel.SessionID)
	}
	if attackerSession.ID == ownerSession.ID {
		t.Fatal("expected distinct owner and attacker sessions")
	}
	if writer.closed {
		t.Fatal("expected unowned reverse tunnel writer to remain open")
	}
}
