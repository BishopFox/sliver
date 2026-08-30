package core

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/bishopfox/sliver/protobuf/rpcpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"google.golang.org/grpc"
)

type tunnelLoopTestRPC struct {
	rpcpb.SliverRPCClient
	stream *tunnelLoopTestStream
	opened chan struct{}
}

func (r *tunnelLoopTestRPC) TunnelData(ctx context.Context, _ ...grpc.CallOption) (grpc.BidiStreamingClient[sliverpb.TunnelData, sliverpb.TunnelData], error) {
	r.stream.ctx = ctx
	close(r.opened)
	return r.stream, nil
}

type tunnelLoopTestStream struct {
	grpc.BidiStreamingClient[sliverpb.TunnelData, sliverpb.TunnelData]
	ctx       context.Context
	incoming  chan *sliverpb.TunnelData
	sent      chan *sliverpb.TunnelData
	recvCalls chan struct{}
	recvErr   chan error
	sendErr   error
}

func (s *tunnelLoopTestStream) Send(data *sliverpb.TunnelData) error {
	if s.sendErr != nil {
		return s.sendErr
	}
	s.sent <- data
	return nil
}

func TestTunnelBindSendFailureClosesTunnel(t *testing.T) {
	tunnels := GetTunnels()
	tunnels.Reset()
	stream := &tunnelLoopTestStream{
		incoming:  make(chan *sliverpb.TunnelData),
		sent:      make(chan *sliverpb.TunnelData, 1),
		recvCalls: make(chan struct{}, 2),
		sendErr:   errors.New("test bind send failure"),
	}
	rpc := &tunnelLoopTestRPC{stream: stream, opened: make(chan struct{})}
	ctx, cancel := context.WithCancel(context.Background())
	loopDone := make(chan error, 1)
	go func() { loopDone <- TunnelLoop(ctx, rpc) }()
	t.Cleanup(func() {
		cancel()
		select {
		case <-loopDone:
		case <-time.After(time.Second):
			t.Error("TunnelLoop did not stop after cancellation")
		}
		tunnels.Reset()
	})
	waitForTestSignal(t, rpc.opened, "TunnelLoop stream startup")
	waitForTestSignal(t, stream.recvCalls, "initial TunnelData receive")

	tunnel := tunnels.Start(9, "session")
	select {
	case <-tunnel.Done():
	case <-time.After(time.Second):
		t.Fatal("tunnel remained open after its bind frame failed to send")
	}
	if got := tunnels.Get(tunnel.ID); got != nil {
		t.Fatal("failed bind tunnel remained registered")
	}
}

func (s *tunnelLoopTestStream) Recv() (*sliverpb.TunnelData, error) {
	s.recvCalls <- struct{}{}
	select {
	case data := <-s.incoming:
		return data, nil
	case err := <-s.recvErr:
		return nil, err
	case <-s.ctx.Done():
		return nil, s.ctx.Err()
	}
}

func TestTunnelLoopFiltersEmptyBindAcknowledgement(t *testing.T) {
	tunnels := GetTunnels()
	tunnels.Reset()

	stream := &tunnelLoopTestStream{
		incoming:  make(chan *sliverpb.TunnelData, 2),
		sent:      make(chan *sliverpb.TunnelData, 1),
		recvCalls: make(chan struct{}, 4),
	}
	rpc := &tunnelLoopTestRPC{
		stream: stream,
		opened: make(chan struct{}),
	}
	ctx, cancel := context.WithCancel(context.Background())
	loopDone := make(chan error, 1)
	go func() {
		loopDone <- TunnelLoop(ctx, rpc)
	}()
	t.Cleanup(func() {
		cancel()
		select {
		case <-loopDone:
		case <-time.After(time.Second):
			t.Error("TunnelLoop did not stop after cancellation")
		}
		tunnels.Reset()
	})

	waitForTestSignal(t, rpc.opened, "TunnelLoop stream startup")
	waitForTestSignal(t, stream.recvCalls, "initial TunnelData receive")

	tunnel := tunnels.Start(7, "session")
	select {
	case bind := <-stream.sent:
		if bind.TunnelID != tunnel.ID || len(bind.Data) != 0 {
			t.Fatalf("bind frame = %+v, want empty data for tunnel %d", bind, tunnel.ID)
		}
	case <-time.After(time.Second):
		t.Fatal("client did not send the tunnel bind frame")
	}

	stream.incoming <- &sliverpb.TunnelData{
		TunnelID:  tunnel.ID,
		SessionID: tunnel.SessionID,
	}
	waitForTestSignal(t, stream.recvCalls, "receive after bind acknowledgement")
	select {
	case <-tunnel.Bound():
	default:
		t.Fatal("bind acknowledgement did not mark the tunnel bound")
	}
	select {
	case data := <-tunnel.Recv:
		t.Fatalf("bind acknowledgement leaked to the tunnel reader: %q", data)
	default:
	}

	payload := []byte("shell prompt")
	stream.incoming <- &sliverpb.TunnelData{
		TunnelID:  tunnel.ID,
		SessionID: tunnel.SessionID,
		Data:      payload,
	}
	waitForTestSignal(t, stream.recvCalls, "receive after shell payload")
	select {
	case got := <-tunnel.Recv:
		if !bytes.Equal(got, payload) {
			t.Fatalf("tunnel payload = %q, want %q", got, payload)
		}
	default:
		t.Fatal("non-empty tunnel payload was not delivered")
	}

	stream.incoming <- &sliverpb.TunnelData{
		TunnelID:  tunnel.ID,
		SessionID: tunnel.SessionID,
		Closed:    true,
	}
	waitForTestSignal(t, stream.recvCalls, "receive after tunnel close")
	if got := tunnels.Get(tunnel.ID); got != nil {
		t.Fatal("empty close frame was filtered as a bind acknowledgement")
	}
	select {
	case tunnel.Send <- []byte("after close"):
		t.Fatal("tunnel sender still consumed data after Close")
	case <-time.After(50 * time.Millisecond):
	}
}

func TestTunnelLoopClosesActiveTunnelsOnStreamEOF(t *testing.T) {
	tunnels := GetTunnels()
	tunnels.Reset()
	t.Cleanup(tunnels.Reset)

	stream := &tunnelLoopTestStream{
		incoming:  make(chan *sliverpb.TunnelData),
		sent:      make(chan *sliverpb.TunnelData, 1),
		recvCalls: make(chan struct{}, 4),
		recvErr:   make(chan error, 1),
	}
	rpc := &tunnelLoopTestRPC{stream: stream, opened: make(chan struct{})}
	loopDone := make(chan error, 1)
	go func() {
		loopDone <- TunnelLoop(context.Background(), rpc)
	}()

	waitForTestSignal(t, rpc.opened, "TunnelLoop stream startup")
	waitForTestSignal(t, stream.recvCalls, "initial TunnelData receive")
	tunnel := tunnels.Start(8, "session")
	select {
	case <-stream.sent:
	case <-time.After(time.Second):
		t.Fatal("client did not send the tunnel bind frame")
	}

	stream.recvErr <- io.EOF
	select {
	case err := <-loopDone:
		if err != nil {
			t.Fatalf("TunnelLoop EOF result = %v, want nil", err)
		}
	case <-time.After(time.Second):
		t.Fatal("TunnelLoop did not return after stream EOF")
	}
	select {
	case <-tunnel.Done():
	case <-time.After(time.Second):
		t.Fatal("active tunnel did not close after stream EOF")
	}
	if got := tunnels.Get(tunnel.ID); got != nil {
		t.Fatal("active tunnel remained registered after stream EOF")
	}
}

func waitForTestSignal(t *testing.T, signal <-chan struct{}, description string) {
	t.Helper()
	select {
	case <-signal:
	case <-time.After(time.Second):
		t.Fatalf("timed out waiting for %s", description)
	}
}
