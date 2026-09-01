package rportfwd

import (
	"bytes"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/bishopfox/sliver/implant/sliver/transports"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"google.golang.org/protobuf/proto"
)

func TestPortfwdsRemoveIfOnlyRemovesExactGeneration(t *testing.T) {
	registry := &portfwds{
		forwards: map[int]*Portfwd{},
		mutex:    &sync.RWMutex{},
	}
	stale := &Portfwd{ID: 7, done: make(chan struct{})}
	current := &Portfwd{ID: 7, done: make(chan struct{})}
	registry.forwards[current.ID] = current

	if removed := registry.RemoveIf(current.ID, stale); removed != nil {
		t.Fatalf("RemoveIf() removed stale generation %+v", removed)
	}
	if got := registry.forwards[current.ID]; got != current {
		t.Fatalf("stale cleanup replaced current generation: got=%p want=%p", got, current)
	}
	select {
	case <-current.Done():
		t.Fatal("stale cleanup closed the current generation")
	default:
	}

	if removed := registry.RemoveIf(current.ID, current); removed != current {
		t.Fatalf("RemoveIf() = %p, want %p", removed, current)
	}
	if _, ok := registry.forwards[current.ID]; ok {
		t.Fatal("exact generation remained in registry")
	}
	select {
	case <-current.Done():
	case <-time.After(time.Second):
		t.Fatal("exact generation removal did not publish completion")
	}
	select {
	case <-stale.Done():
		t.Fatal("exact generation removal closed stale generation")
	default:
	}
}

type testTunnelConnection struct {
	*transports.Connection
	send chan *sliverpb.Envelope
}

func newTestTunnelConnection() *testTunnelConnection {
	send := make(chan *sliverpb.Envelope, 4)
	return &testTunnelConnection{
		Connection: &transports.Connection{Send: send},
		send:       send,
	}
}

func TestChannelProxySourceEOFSendsSequencedCloseAndDetaches(t *testing.T) {
	connection := newTestTunnelConnection()
	proxy := &ChannelProxy{
		Conn:            connection,
		RemoteAddr:      "127.0.0.1:443",
		AuthorizationID: "server-issued",
		DialTimeout:     time.Second,
	}
	implantSide, clientSide := net.Pipe()
	t.Cleanup(func() {
		_ = implantSide.Close()
		_ = clientSide.Close()
	})
	proxy.HandleConn(implantSide)

	payload := []byte("final-payload")
	if _, err := clientSide.Write(payload); err != nil {
		t.Fatalf("write client payload: %v", err)
	}
	dataTransportEnvelope, dataEnvelope := receiveTunnelEnvelope(t, connection.send)
	if dataTransportEnvelope.Type != sliverpb.MsgTunnelData || dataEnvelope.Sequence != 0 || !dataEnvelope.CreateReverse {
		t.Fatalf("unexpected first tunnel envelope: %+v", dataEnvelope)
	}
	if !bytes.Equal(dataEnvelope.Data, payload) {
		t.Fatalf("data payload = %q, want %q", dataEnvelope.Data, payload)
	}
	if err := clientSide.Close(); err != nil {
		t.Fatalf("close client side: %v", err)
	}

	closeTransportEnvelope, closeEnvelope := receiveTunnelEnvelope(t, connection.send)
	if closeTransportEnvelope.Type != sliverpb.MsgTunnelClose || !closeEnvelope.Closed || closeEnvelope.Sequence != 1 || closeEnvelope.TunnelID != dataEnvelope.TunnelID {
		t.Fatalf("unexpected terminal close: %+v", closeEnvelope)
	}
	if connection.Tunnel(dataEnvelope.TunnelID) != nil {
		t.Fatal("source EOF retained the local tunnel generation")
	}
	select {
	case extra := <-connection.send:
		t.Fatalf("source EOF emitted an extra envelope: %+v", extra)
	case <-time.After(50 * time.Millisecond):
	}
}

func receiveTunnelEnvelope(t *testing.T, envelopes <-chan *sliverpb.Envelope) (*sliverpb.Envelope, *sliverpb.TunnelData) {
	t.Helper()
	select {
	case envelope := <-envelopes:
		decoded := &sliverpb.TunnelData{}
		if err := proto.Unmarshal(envelope.Data, decoded); err != nil {
			t.Fatalf("decode tunnel envelope: %v", err)
		}
		return envelope, decoded
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for tunnel envelope")
		return nil, nil
	}
}
