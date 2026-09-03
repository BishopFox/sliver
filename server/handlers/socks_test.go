package handlers

import (
	"sync"
	"testing"
	"time"

	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/bishopfox/sliver/server/core"
	"google.golang.org/protobuf/proto"
)

func marshalSocksData(t *testing.T, data *sliverpb.SocksData) []byte {
	t.Helper()
	encoded, err := proto.Marshal(data)
	if err != nil {
		t.Fatalf("marshal SOCKS data: %v", err)
	}
	return encoded
}

func TestSocksDataHandlerEnforcesTunnelSessionOwnership(t *testing.T) {
	ownerConnection, ownerSession := addTestSession(t)
	attackerConnection, _ := addTestSession(t)
	tunnel, err := core.SocksTunnels.Create(ownerSession.ID)
	if err != nil {
		t.Fatalf("create SOCKS tunnel: %v", err)
	}
	t.Cleanup(func() { core.SocksTunnels.CloseIf(tunnel) })
	frame := &sliverpb.SocksData{TunnelID: tunnel.ID, Sequence: 0, Data: []byte("owned")}
	encoded := marshalSocksData(t, frame)

	socksDataHandler(attackerConnection, encoded)
	select {
	case unexpected := <-tunnel.FromImplant():
		t.Fatalf("non-owning session delivered SOCKS frame: %+v", unexpected)
	default:
	}

	socksDataHandler(ownerConnection, encoded)
	select {
	case delivered := <-tunnel.FromImplant():
		tunnel.CompleteFromImplant(delivered)
		if delivered.TunnelID != tunnel.ID || string(delivered.Data) != "owned" {
			t.Fatalf("owned SOCKS frame = %+v", delivered)
		}
	case <-time.After(time.Second):
		t.Fatal("owning session SOCKS frame was not delivered")
	}
}

func TestSocksDataHandlerDeliveryCloseRace(t *testing.T) {
	const iterations = 500
	connection, session := addTestSession(t)
	for iteration := 0; iteration < iterations; iteration++ {
		tunnel, err := core.SocksTunnels.Create(session.ID)
		if err != nil {
			t.Fatalf("iteration %d create SOCKS tunnel: %v", iteration, err)
		}
		encoded := marshalSocksData(t, &sliverpb.SocksData{
			TunnelID: tunnel.ID,
			Sequence: 0,
			Data:     []byte("race"),
		})
		start := make(chan struct{})
		var workers sync.WaitGroup
		workers.Add(2)
		go func() {
			defer workers.Done()
			<-start
			socksDataHandler(connection, encoded)
		}()
		go func() {
			defer workers.Done()
			<-start
			core.SocksTunnels.CloseIf(tunnel)
		}()
		close(start)
		finished := make(chan struct{})
		go func() {
			workers.Wait()
			close(finished)
		}()
		select {
		case <-finished:
		case <-time.After(time.Second):
			core.SocksTunnels.CloseIf(tunnel)
			t.Fatalf("iteration %d SOCKS delivery/close race did not finish", iteration)
		}
		core.SocksTunnels.CloseIf(tunnel)
	}
}

func TestSocksDataHandlerRejectsMalformedFrame(t *testing.T) {
	connection, _ := addTestSession(t)
	assertNoPanic(t, func() {
		socksDataHandler(connection, []byte{0xff, 0xff, 0xff})
	})
}

func TestSocksDataHandlerClosesConnectionBeforeOversizedUnmarshal(t *testing.T) {
	connection := core.NewImplantConnection("oversized-socks-wire", "test")
	session := core.NewSession(connection)
	if !connection.SetCleanup(func() { core.Sessions.Remove(session.ID) }) {
		t.Fatal("install oversized-frame session cleanup")
	}
	core.Sessions.Add(session)
	t.Cleanup(func() {
		core.Sessions.Remove(session.ID)
		connection.Close()
	})
	tunnel, err := core.SocksTunnels.Create(session.ID)
	if err != nil {
		t.Fatalf("create SOCKS tunnel: %v", err)
	}
	socksDataHandler(connection, make([]byte, maxTunnelDataMessageBytes+1))
	select {
	case <-connection.Done():
	case <-time.After(time.Second):
		t.Fatal("oversized raw SOCKS message did not close implant connection")
	}
	select {
	case <-tunnel.Done():
	case <-time.After(time.Second):
		t.Fatal("oversized raw SOCKS message did not finalize session tunnel")
	}
	if got := core.SocksTunnels.Get(tunnel.ID); got != nil {
		t.Fatalf("oversized raw SOCKS message retained tunnel: %p", got)
	}
}

func TestSocksDataHandlerRejectsOversizedPayload(t *testing.T) {
	connection, session := addTestSession(t)
	tunnel, err := core.SocksTunnels.Create(session.ID)
	if err != nil {
		t.Fatalf("create SOCKS tunnel: %v", err)
	}
	t.Cleanup(func() { core.SocksTunnels.CloseIf(tunnel) })
	encoded := marshalSocksData(t, &sliverpb.SocksData{
		TunnelID: tunnel.ID,
		Sequence: 0,
		Data:     make([]byte, core.MaxSocksFrameBytes+1),
	})
	if len(encoded) > maxTunnelDataMessageBytes {
		t.Fatalf("test payload encoded to %d bytes, raw limit is %d", len(encoded), maxTunnelDataMessageBytes)
	}

	socksDataHandler(connection, encoded)
	select {
	case <-connection.Done():
	case <-time.After(time.Second):
		t.Fatal("oversized SOCKS payload did not close implant connection")
	}
	select {
	case <-tunnel.Done():
	case <-time.After(time.Second):
		t.Fatal("oversized SOCKS payload did not close tunnel")
	}
	if got := core.SocksTunnels.Get(tunnel.ID); got != nil {
		t.Fatalf("oversized SOCKS payload retained tunnel: %p", got)
	}
}

func TestSocksDataHandlerRejectsConflictingDuplicate(t *testing.T) {
	connection, session := addTestSession(t)
	tunnel, err := core.SocksTunnels.Create(session.ID)
	if err != nil {
		t.Fatalf("create SOCKS tunnel: %v", err)
	}
	t.Cleanup(func() { core.SocksTunnels.CloseIf(tunnel) })
	socksDataHandler(connection, marshalSocksData(t, &sliverpb.SocksData{
		TunnelID: tunnel.ID,
		Sequence: 1,
		Data:     []byte("first"),
	}))
	socksDataHandler(connection, marshalSocksData(t, &sliverpb.SocksData{
		TunnelID: tunnel.ID,
		Sequence: 1,
		Data:     []byte("conflict"),
	}))
	select {
	case <-connection.Done():
	case <-time.After(time.Second):
		t.Fatal("conflicting SOCKS duplicate did not close implant connection")
	}
	if got := core.SocksTunnels.Get(tunnel.ID); got != nil {
		t.Fatalf("conflicting SOCKS duplicate retained tunnel: %p", got)
	}
}

func TestSocksDataHandlerScopesIngressPressureToExactTunnel(t *testing.T) {
	const receiveWindow = 128

	connection, session := addTestSession(t)
	tunnel, err := core.SocksTunnels.Create(session.ID)
	if err != nil {
		t.Fatalf("create saturated SOCKS tunnel: %v", err)
	}
	sibling, err := core.SocksTunnels.Create(session.ID)
	if err != nil {
		core.SocksTunnels.CloseIf(tunnel)
		t.Fatalf("create sibling SOCKS tunnel: %v", err)
	}
	t.Cleanup(func() {
		core.SocksTunnels.CloseIf(tunnel)
		core.SocksTunnels.CloseIf(sibling)
	})

	for sequence := range receiveWindow {
		socksDataHandler(connection, marshalSocksData(t, &sliverpb.SocksData{
			TunnelID: tunnel.ID,
			Sequence: uint64(sequence),
			Data:     []byte("queued"),
		}))
	}
	socksDataHandler(connection, marshalSocksData(t, &sliverpb.SocksData{
		TunnelID: tunnel.ID,
		Sequence: receiveWindow,
		Data:     []byte("overflow"),
	}))

	select {
	case <-tunnel.Done():
	case <-time.After(time.Second):
		t.Fatal("resource-pressure frame did not close its exact SOCKS tunnel")
	}
	if got := core.SocksTunnels.Get(tunnel.ID); got != nil {
		t.Fatalf("resource-pressure frame retained SOCKS tunnel: %p", got)
	}
	select {
	case <-connection.Done():
		t.Fatal("SOCKS tunnel resource pressure closed the implant connection")
	default:
	}
	if got := core.SocksTunnels.Get(sibling.ID); got != sibling {
		t.Fatalf("SOCKS tunnel resource pressure replaced sibling: got=%p want=%p", got, sibling)
	}
	select {
	case <-sibling.Done():
		t.Fatal("SOCKS tunnel resource pressure closed sibling tunnel")
	default:
	}
}

func TestSocksDataHandlerRequiresExactConnectionGeneration(t *testing.T) {
	ownerConnection, ownerSession := addTestSession(t)
	tunnel, err := core.SocksTunnels.Create(ownerSession.ID)
	if err != nil {
		t.Fatalf("create SOCKS tunnel: %v", err)
	}
	t.Cleanup(func() { core.SocksTunnels.CloseIf(tunnel) })

	replacementConnection := core.NewImplantConnection("test", "replacement")
	replacement := core.NewSession(replacementConnection)
	replacement.ID = ownerSession.ID
	core.Sessions.Add(replacement)
	t.Cleanup(func() {
		core.Sessions.RemoveIf(replacement)
		replacementConnection.Close()
	})

	oversized := marshalSocksData(t, &sliverpb.SocksData{
		TunnelID: tunnel.ID,
		Sequence: 0,
		Data:     make([]byte, core.MaxSocksFrameBytes+1),
	})
	if len(oversized) > maxTunnelDataMessageBytes {
		t.Fatalf("test frame encoded to %d bytes, raw limit is %d", len(oversized), maxTunnelDataMessageBytes)
	}
	socksDataHandler(replacementConnection, oversized)
	if got := core.SocksTunnels.Get(tunnel.ID); got != tunnel {
		t.Fatalf("replacement connection mutated SOCKS tunnel: got=%p want=%p", got, tunnel)
	}
	select {
	case <-tunnel.Done():
		t.Fatal("replacement connection closed old-generation SOCKS tunnel")
	default:
	}
	select {
	case <-replacementConnection.Done():
		t.Fatal("wrong-generation SOCKS data closed replacement connection")
	default:
	}
	select {
	case <-ownerConnection.Done():
		t.Fatal("wrong-generation SOCKS ingress closed owner connection")
	default:
	}
}

func TestSocksDataHandlerDoesNotUseGlobalTunnelMutex(t *testing.T) {
	connection, session := addTestSession(t)
	tunnel, err := core.SocksTunnels.Create(session.ID)
	if err != nil {
		t.Fatalf("create SOCKS tunnel: %v", err)
	}
	t.Cleanup(func() { core.SocksTunnels.CloseIf(tunnel) })
	encoded := marshalSocksData(t, &sliverpb.SocksData{
		TunnelID: tunnel.ID,
		Sequence: 0,
		Data:     []byte("independent"),
	})

	tunnelHandlerMutex.Lock()
	finished := make(chan struct{})
	go func() {
		defer close(finished)
		socksDataHandler(connection, encoded)
	}()
	select {
	case <-finished:
	case <-time.After(time.Second):
		tunnelHandlerMutex.Unlock()
		t.Fatal("SOCKS handler blocked on global tunnel mutex")
	}
	tunnelHandlerMutex.Unlock()
}
