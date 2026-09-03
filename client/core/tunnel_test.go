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
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type tunnelLoopTestRPC struct {
	rpcpb.SliverRPCClient
	stream   *tunnelLoopTestStream
	opened   chan struct{}
	startErr error
}

func (r *tunnelLoopTestRPC) TunnelData(ctx context.Context, _ ...grpc.CallOption) (grpc.BidiStreamingClient[sliverpb.TunnelData, sliverpb.TunnelData], error) {
	close(r.opened)
	if r.startErr != nil {
		return nil, r.startErr
	}
	r.stream.ctx = ctx
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

func TestTunnelLoopWithReadySignalsAfterStreamInstallation(t *testing.T) {
	tunnels := GetTunnels()
	tunnels.Reset()
	stream := &tunnelLoopTestStream{
		incoming:  make(chan *sliverpb.TunnelData),
		sent:      make(chan *sliverpb.TunnelData, 1),
		recvCalls: make(chan struct{}, 2),
	}
	rpc := &tunnelLoopTestRPC{stream: stream, opened: make(chan struct{})}
	ctx, cancel := context.WithCancel(context.Background())
	ready := make(chan struct{})
	loopDone := make(chan error, 1)
	go func() { loopDone <- TunnelLoopWithReady(ctx, rpc, ready) }()
	t.Cleanup(func() {
		cancel()
		select {
		case <-loopDone:
		case <-time.After(time.Second):
			t.Error("TunnelLoopWithReady did not stop after cancellation")
		}
		tunnels.Reset()
	})

	waitForTestSignal(t, ready, "TunnelLoop readiness")
	tunnel := tunnels.Start(10, "session")
	select {
	case bind := <-stream.sent:
		if bind.TunnelID != tunnel.ID {
			t.Fatalf("bind tunnel ID = %d, want %d", bind.TunnelID, tunnel.ID)
		}
	case <-time.After(time.Second):
		t.Fatal("ready tunnel loop did not send a bind frame")
	}
}

func TestTunnelLoopWithReadyDoesNotSignalOnStartupFailure(t *testing.T) {
	startErr := errors.New("test stream startup failure")
	rpc := &tunnelLoopTestRPC{opened: make(chan struct{}), startErr: startErr}
	ready := make(chan struct{})

	err := TunnelLoopWithReady(context.Background(), rpc, ready)
	if !errors.Is(err, startErr) {
		t.Fatalf("TunnelLoopWithReady error = %v, want %v", err, startErr)
	}
	select {
	case <-ready:
		t.Fatal("readiness was signaled after stream startup failed")
	default:
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

//nolint:gocyclo // The overload and sibling-isolation assertions share one stream lifecycle.
func TestTunnelLoopOverloadedTunnelDoesNotBlockHealthyTunnel(t *testing.T) {
	tunnels := GetTunnels()
	tunnels.Reset()

	stream := &tunnelLoopTestStream{
		incoming:  make(chan *sliverpb.TunnelData, tunnelRecvBufferSize+4),
		sent:      make(chan *sliverpb.TunnelData, 2),
		recvCalls: make(chan struct{}, tunnelRecvBufferSize+8),
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
	stalled := tunnels.Start(11, "session")
	healthy := tunnels.Start(12, "session")
	bound := map[uint64]bool{}
	for len(bound) < 2 {
		select {
		case frame := <-stream.sent:
			bound[frame.TunnelID] = true
		case <-time.After(time.Second):
			t.Fatal("client did not send both tunnel bind frames")
		}
	}

	for sequence := 0; sequence < tunnelRecvBufferSize; sequence++ {
		stream.incoming <- &sliverpb.TunnelData{
			TunnelID:  stalled.ID,
			SessionID: stalled.SessionID,
			Data:      []byte{byte(sequence)},
		}
		waitForTestSignal(t, stream.recvCalls, "receive after stalled tunnel frame")
	}
	stream.incoming <- &sliverpb.TunnelData{
		TunnelID:  stalled.ID,
		SessionID: stalled.SessionID,
		Data:      []byte("overflow"),
	}
	waitForTestSignal(t, stream.recvCalls, "receive after stalled tunnel overflow")

	healthyPayload := []byte("healthy tunnel data")
	stream.incoming <- &sliverpb.TunnelData{
		TunnelID:  healthy.ID,
		SessionID: healthy.SessionID,
		Data:      healthyPayload,
	}
	waitForTestSignal(t, stream.recvCalls, "receive after healthy tunnel frame")

	select {
	case <-stalled.Done():
	case <-time.After(time.Second):
		t.Fatal("overloaded tunnel was not closed")
	}
	if got := tunnels.Get(stalled.ID); got != nil {
		t.Fatal("overloaded tunnel remained registered")
	}
	type healthyReadResult struct {
		data []byte
		n    int
		err  error
	}
	healthyRead := make(chan healthyReadResult, 1)
	go func() {
		buffer := make([]byte, len(healthyPayload))
		n, err := healthy.Read(buffer)
		healthyRead <- healthyReadResult{data: buffer[:n], n: n, err: err}
	}()
	select {
	case result := <-healthyRead:
		if result.n != len(healthyPayload) || result.err != nil || !bytes.Equal(result.data, healthyPayload) {
			t.Fatalf("healthy tunnel read = (%d, %q, %v), want (%d, %q, nil)", result.n, result.data, result.err, len(healthyPayload), healthyPayload)
		}
	case <-time.After(time.Second):
		t.Fatal("healthy tunnel read blocked behind overloaded tunnel")
	}
	if !healthy.IsOpen() {
		t.Fatal("healthy tunnel closed after unrelated tunnel overload")
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

func TestTunnelLoopPropagatesRemoteGRPCCanceled(t *testing.T) {
	tunnels := GetTunnels()
	tunnels.Reset()
	t.Cleanup(tunnels.Reset)

	stream := &tunnelLoopTestStream{
		incoming:  make(chan *sliverpb.TunnelData),
		sent:      make(chan *sliverpb.TunnelData, 1),
		recvCalls: make(chan struct{}, 2),
		recvErr:   make(chan error, 1),
	}
	rpc := &tunnelLoopTestRPC{stream: stream, opened: make(chan struct{})}
	loopDone := make(chan error, 1)
	go func() {
		loopDone <- TunnelLoop(context.Background(), rpc)
	}()

	waitForTestSignal(t, rpc.opened, "TunnelLoop stream startup")
	waitForTestSignal(t, stream.recvCalls, "initial TunnelData receive")
	remoteErr := status.Error(codes.Canceled, "test remote shutdown")
	stream.recvErr <- remoteErr

	select {
	case err := <-loopDone:
		if status.Code(err) != codes.Canceled {
			t.Fatalf("TunnelLoop remote canceled status result = %v, want %v", err, remoteErr)
		}
	case <-time.After(time.Second):
		t.Fatal("TunnelLoop did not return after remote canceled gRPC status")
	}
}

func TestTunnelLoopTreatsCallerCancellationAsCleanShutdown(t *testing.T) {
	tunnels := GetTunnels()
	tunnels.Reset()
	t.Cleanup(tunnels.Reset)

	stream := &tunnelLoopTestStream{
		incoming:  make(chan *sliverpb.TunnelData),
		sent:      make(chan *sliverpb.TunnelData, 1),
		recvCalls: make(chan struct{}, 2),
		recvErr:   make(chan error, 1),
	}
	rpc := &tunnelLoopTestRPC{stream: stream, opened: make(chan struct{})}
	ctx, cancel := context.WithCancel(context.Background())
	loopDone := make(chan error, 1)
	go func() {
		loopDone <- TunnelLoop(ctx, rpc)
	}()

	waitForTestSignal(t, rpc.opened, "TunnelLoop stream startup")
	waitForTestSignal(t, stream.recvCalls, "initial TunnelData receive")
	cancel()

	select {
	case err := <-loopDone:
		if err != nil {
			t.Fatalf("TunnelLoop caller cancellation result = %v, want nil", err)
		}
	case <-time.After(time.Second):
		t.Fatal("TunnelLoop did not return after caller cancellation")
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
