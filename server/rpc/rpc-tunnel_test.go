package rpc

import (
	"errors"
	"io"
	"sync/atomic"
	"testing"
	"time"

	"github.com/bishopfox/sliver/protobuf/rpcpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/bishopfox/sliver/server/core"
	"google.golang.org/protobuf/proto"
)

type tunnelStreamRecv struct {
	data *sliverpb.TunnelData
	err  error
}

type testTunnelStream struct {
	rpcpb.SliverRPC_TunnelDataServer
	recv      chan tunnelStreamRecv
	sent      chan *sliverpb.TunnelData
	sendDelay time.Duration
	failAfter int32
	sendCalls atomic.Int32
	inSend    atomic.Int32
	overlap   atomic.Bool
}

func (s *testTunnelStream) Recv() (*sliverpb.TunnelData, error) {
	result := <-s.recv
	return result.data, result.err
}

func (s *testTunnelStream) Send(data *sliverpb.TunnelData) error {
	if s.inSend.Add(1) > 1 {
		s.overlap.Store(true)
	}
	defer s.inSend.Add(-1)
	if s.sendDelay > 0 {
		time.Sleep(s.sendDelay)
	}
	call := s.sendCalls.Add(1)
	if s.failAfter > 0 && call > s.failAfter {
		return errors.New("test stream send failure")
	}
	s.sent <- proto.Clone(data).(*sliverpb.TunnelData)
	return nil
}

func TestTunnelDataDropsUnknownFrameWithoutClosingStream(t *testing.T) {
	stream := &testTunnelStream{
		recv: make(chan tunnelStreamRecv, 2),
		sent: make(chan *sliverpb.TunnelData, 1),
	}
	stream.recv <- tunnelStreamRecv{data: &sliverpb.TunnelData{TunnelID: 0xdeadbeef, Data: []byte("late")}}
	stream.recv <- tunnelStreamRecv{err: io.EOF}

	if err := (&Server{}).TunnelData(stream); err != nil {
		t.Fatalf("TunnelData returned for stale frame: %v", err)
	}
}

func TestTunnelDataClosesOwnedTunnelsAndSerializesSends(t *testing.T) {
	session, tunnels := newTunnelStreamTestSession(t, 2)
	drainImplantCloseEnvelopes(session, len(tunnels))
	stream := &testTunnelStream{
		recv:      make(chan tunnelStreamRecv, 3),
		sent:      make(chan *sliverpb.TunnelData, 16),
		sendDelay: 20 * time.Millisecond,
	}
	for _, tunnel := range tunnels {
		stream.recv <- tunnelStreamRecv{data: &sliverpb.TunnelData{TunnelID: tunnel.ID, SessionID: session.ID}}
	}

	handlerDone := make(chan error, 1)
	go func() { handlerDone <- (&Server{}).TunnelData(stream) }()
	for range tunnels {
		waitForTunnelStreamSend(t, stream.sent)
	}

	for index, tunnel := range tunnels {
		index := index
		tunnel := tunnel
		go tunnel.SendDataFromImplant(&sliverpb.TunnelData{
			TunnelID: tunnel.ID,
			Sequence: 0,
			Data:     []byte{byte(index + 1)},
		})
	}
	for range tunnels {
		waitForTunnelStreamSend(t, stream.sent)
	}
	if stream.overlap.Load() {
		t.Fatal("server called Send concurrently on one TunnelData stream")
	}

	stream.recv <- tunnelStreamRecv{err: io.EOF}
	select {
	case err := <-handlerDone:
		if err != nil {
			t.Fatalf("TunnelData EOF result = %v, want nil", err)
		}
	case <-time.After(time.Second):
		t.Fatal("TunnelData did not return after EOF")
	}
	for _, tunnel := range tunnels {
		select {
		case <-tunnel.Done():
		case <-time.After(time.Second):
			t.Fatalf("tunnel %d did not close after its client stream ended", tunnel.ID)
		}
	}
}

func TestTunnelDataSendFailureClosesTunnel(t *testing.T) {
	session, tunnels := newTunnelStreamTestSession(t, 1)
	drainImplantCloseEnvelopes(session, 1)
	tunnel := tunnels[0]
	stream := &testTunnelStream{
		recv:      make(chan tunnelStreamRecv, 2),
		sent:      make(chan *sliverpb.TunnelData, 4),
		failAfter: 1, // Bind acknowledgement succeeds; first data frame fails.
	}
	stream.recv <- tunnelStreamRecv{data: &sliverpb.TunnelData{TunnelID: tunnel.ID, SessionID: session.ID}}
	handlerDone := make(chan error, 1)
	go func() { handlerDone <- (&Server{}).TunnelData(stream) }()
	waitForTunnelStreamSend(t, stream.sent)

	go tunnel.SendDataFromImplant(&sliverpb.TunnelData{TunnelID: tunnel.ID, Sequence: 0, Data: []byte("output")})
	select {
	case <-tunnel.Done():
	case <-time.After(time.Second):
		t.Fatal("tunnel remained open after stream Send failed")
	}
	stream.recv <- tunnelStreamRecv{err: io.EOF}
	select {
	case <-handlerDone:
	case <-time.After(time.Second):
		t.Fatal("TunnelData did not return after EOF")
	}
}

func newTunnelStreamTestSession(t *testing.T, count int) (*core.Session, []*core.Tunnel) {
	t.Helper()
	session := core.NewSession(core.NewImplantConnection("mtls", "test"))
	core.Sessions.Add(session)
	t.Cleanup(func() { core.Sessions.Remove(session.ID) })
	tunnels := make([]*core.Tunnel, 0, count)
	for range count {
		tunnel, err := core.Tunnels.Create(session.ID)
		if err != nil {
			t.Fatalf("create tunnel: %v", err)
		}
		tunnels = append(tunnels, tunnel)
		t.Cleanup(func() { _ = core.Tunnels.Close(tunnel.ID) })
	}
	return session, tunnels
}

func drainImplantCloseEnvelopes(session *core.Session, count int) {
	go func() {
		for range count {
			<-session.Connection.Send
		}
	}()
}

func waitForTunnelStreamSend(t *testing.T, sent <-chan *sliverpb.TunnelData) *sliverpb.TunnelData {
	t.Helper()
	select {
	case data := <-sent:
		return data
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for TunnelData Send")
		return nil
	}
}
