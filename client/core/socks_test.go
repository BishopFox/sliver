package core

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/rpcpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

type socksLifecycleTestAddr string

func (address socksLifecycleTestAddr) Network() string { return "test" }
func (address socksLifecycleTestAddr) String() string  { return string(address) }

type socksLifecycleTestListener struct {
	closed atomic.Bool
}

func (listener *socksLifecycleTestListener) Accept() (net.Conn, error) {
	return nil, net.ErrClosed
}

func (listener *socksLifecycleTestListener) Close() error {
	listener.closed.Store(true)
	return nil
}

func (listener *socksLifecycleTestListener) Addr() net.Addr {
	return socksLifecycleTestAddr("listener")
}

type socksLifecycleTestConn struct {
	closed atomic.Bool
}

func (connection *socksLifecycleTestConn) Read([]byte) (int, error) {
	if connection.closed.Load() {
		return 0, net.ErrClosed
	}
	return 0, io.EOF
}

func (connection *socksLifecycleTestConn) Write(data []byte) (int, error) {
	if connection.closed.Load() {
		return 0, net.ErrClosed
	}
	return len(data), nil
}

func (connection *socksLifecycleTestConn) Close() error {
	connection.closed.Store(true)
	return nil
}

func (connection *socksLifecycleTestConn) LocalAddr() net.Addr {
	return socksLifecycleTestAddr("local")
}

func (connection *socksLifecycleTestConn) RemoteAddr() net.Addr {
	return socksLifecycleTestAddr("remote")
}

func (connection *socksLifecycleTestConn) SetDeadline(time.Time) error      { return nil }
func (connection *socksLifecycleTestConn) SetReadDeadline(time.Time) error  { return nil }
func (connection *socksLifecycleTestConn) SetWriteDeadline(time.Time) error { return nil }

type socksFinalReadConn struct {
	socksLifecycleTestConn
	payload []byte
	read    atomic.Bool
}

type socksFinalReadDeadlineErrorConn struct {
	socksFinalReadConn
}

func (*socksFinalReadDeadlineErrorConn) SetReadDeadline(deadline time.Time) error {
	if deadline.IsZero() {
		return errors.New("test deadline clear failure")
	}
	return nil
}

type socksScriptedReadConn struct {
	socksLifecycleTestConn
	payloads [][]byte
	next     int
}

type socksBurstReadConn struct {
	socksLifecycleTestConn
	remaining int
}

type socksFullBufferReadConn struct {
	socksLifecycleTestConn
	read atomic.Bool
}

type socksQueuedListener struct {
	connection net.Conn
	accepted   atomic.Bool
	closed     chan struct{}
	closeOnce  sync.Once
}

func (listener *socksQueuedListener) Accept() (net.Conn, error) {
	if !listener.accepted.Swap(true) {
		return listener.connection, nil
	}
	<-listener.closed
	return nil, net.ErrClosed
}

func (listener *socksQueuedListener) Close() error {
	listener.closeOnce.Do(func() { close(listener.closed) })
	return nil
}

func (listener *socksQueuedListener) Addr() net.Addr {
	return socksLifecycleTestAddr("queued-listener")
}

type socksSequenceListener struct {
	connections chan net.Conn
	closed      chan struct{}
	closeOnce   sync.Once
}

func (listener *socksSequenceListener) Accept() (net.Conn, error) {
	select {
	case connection := <-listener.connections:
		return connection, nil
	case <-listener.closed:
		return nil, net.ErrClosed
	}
}

func (listener *socksSequenceListener) Close() error {
	listener.closeOnce.Do(func() { close(listener.closed) })
	return nil
}

func (listener *socksSequenceListener) Addr() net.Addr {
	return socksLifecycleTestAddr("sequence-listener")
}

type socksStartTestStream struct {
	grpc.BidiStreamingClient[sliverpb.SocksData, sliverpb.SocksData]
	ctx     context.Context
	recvErr error
	sent    chan *sliverpb.SocksData
}

func (stream *socksStartTestStream) Send(frame *sliverpb.SocksData) error {
	if stream.sent != nil {
		stream.sent <- proto.Clone(frame).(*sliverpb.SocksData)
	}
	return nil
}
func (stream *socksStartTestStream) Recv() (*sliverpb.SocksData, error) {
	if stream.recvErr != nil {
		err := stream.recvErr
		stream.recvErr = nil
		return nil, err
	}
	<-stream.ctx.Done()
	return nil, stream.ctx.Err()
}
func (*socksStartTestStream) CloseSend() error { return nil }

type socksStartLifecycleRPC struct {
	rpcpb.SliverRPCClient
	streamErr          error
	recvErr            error
	createEntered      chan struct{}
	createCanceled     chan struct{}
	createResponse     *sliverpb.Socks
	beforeCreateReturn func()
	closeCalls         chan socksCloseObservation
	createOnce         sync.Once
	cancelOnce         sync.Once
}

type socksStartCreateResult struct {
	response *sliverpb.Socks
	err      error
}

type socksStartSequenceRPC struct {
	rpcpb.SliverRPCClient
	results        chan socksStartCreateResult
	closeCalls     chan socksCloseObservation
	createRequests chan *sliverpb.Socks
	streamSent     chan *sliverpb.SocksData
}

func (rpc *socksStartSequenceRPC) SocksProxy(ctx context.Context, _ ...grpc.CallOption) (grpc.BidiStreamingClient[sliverpb.SocksData, sliverpb.SocksData], error) {
	return &socksStartTestStream{ctx: ctx, sent: rpc.streamSent}, nil
}

func (rpc *socksStartSequenceRPC) CreateSocks(ctx context.Context, request *sliverpb.Socks, _ ...grpc.CallOption) (*sliverpb.Socks, error) {
	if rpc.createRequests != nil {
		select {
		case rpc.createRequests <- proto.Clone(request).(*sliverpb.Socks):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	select {
	case result := <-rpc.results:
		if result.response == nil {
			return nil, result.err
		}
		return proto.Clone(result.response).(*sliverpb.Socks), result.err
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (rpc *socksStartSequenceRPC) CloseSocks(ctx context.Context, request *sliverpb.Socks, _ ...grpc.CallOption) (*commonpb.Empty, error) {
	observation := socksCloseObservation{tunnelID: request.TunnelID, sessionID: request.SessionID}
	if deadline, ok := ctx.Deadline(); ok {
		observation.hasDeadline = true
		observation.deadlineRemaining = time.Until(deadline)
	}
	select {
	case rpc.closeCalls <- observation:
		return &commonpb.Empty{}, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (rpc *socksStartLifecycleRPC) SocksProxy(ctx context.Context, _ ...grpc.CallOption) (grpc.BidiStreamingClient[sliverpb.SocksData, sliverpb.SocksData], error) {
	if rpc.streamErr != nil {
		return nil, rpc.streamErr
	}
	return &socksStartTestStream{ctx: ctx, recvErr: rpc.recvErr}, nil
}

func (rpc *socksStartLifecycleRPC) CreateSocks(ctx context.Context, _ *sliverpb.Socks, _ ...grpc.CallOption) (*sliverpb.Socks, error) {
	rpc.createOnce.Do(func() {
		if rpc.createEntered != nil {
			close(rpc.createEntered)
		}
	})
	if rpc.createResponse != nil {
		if rpc.beforeCreateReturn != nil {
			rpc.beforeCreateReturn()
		}
		return proto.Clone(rpc.createResponse).(*sliverpb.Socks), nil
	}
	<-ctx.Done()
	rpc.cancelOnce.Do(func() {
		if rpc.createCanceled != nil {
			close(rpc.createCanceled)
		}
	})
	return nil, ctx.Err()
}

func (rpc *socksStartLifecycleRPC) CloseSocks(ctx context.Context, request *sliverpb.Socks, _ ...grpc.CallOption) (*commonpb.Empty, error) {
	deadline, hasDeadline := ctx.Deadline()
	observation := socksCloseObservation{
		tunnelID:    request.TunnelID,
		sessionID:   request.SessionID,
		hasDeadline: hasDeadline,
	}
	if hasDeadline {
		observation.deadlineRemaining = time.Until(deadline)
	}
	select {
	case rpc.closeCalls <- observation:
		return &commonpb.Empty{}, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func TestSocksProxyStartupFailureStopsListenerAndRemovesInventory(t *testing.T) {
	resetSocksLifecycleTestState(t)
	wantErr := errors.New("test SOCKS stream startup failure")
	listener := &socksLifecycleTestListener{}
	proxy := &TcpProxy{
		Rpc:      &socksStartLifecycleRPC{streamErr: wantErr},
		Session:  &clientpb.Session{ID: "socks-startup-failure"},
		Listener: listener,
	}
	registered := SocksProxies.Add(proxy)
	if err := SocksProxies.Start(proxy); !errors.Is(err, wantErr) {
		t.Fatalf("Start error = %v, want %v", err, wantErr)
	}
	if !listener.closed.Load() {
		t.Fatal("startup failure did not close the SOCKS listener")
	}
	if hasSocksProxy(registered.ID) {
		t.Fatalf("startup failure retained SOCKS proxy %d in inventory", registered.ID)
	}
}

func TestTcpProxyStopCancelsInFlightCreateSocks(t *testing.T) {
	resetSocksLifecycleTestState(t)
	connection := &socksLifecycleTestConn{}
	listener := &socksQueuedListener{
		connection: connection,
		closed:     make(chan struct{}),
	}
	rpc := &socksStartLifecycleRPC{
		createEntered:  make(chan struct{}),
		createCanceled: make(chan struct{}),
	}
	proxy := &TcpProxy{
		Rpc:      rpc,
		Session:  &clientpb.Session{ID: "socks-cancel-create"},
		Listener: listener,
	}
	registered := SocksProxies.Add(proxy)
	startDone := make(chan error, 1)
	go func() { startDone <- SocksProxies.Start(proxy) }()

	select {
	case <-rpc.createEntered:
	case <-time.After(time.Second):
		t.Fatal("CreateSocks did not start")
	}
	if err := proxy.Stop(); err != nil {
		t.Fatalf("stop SOCKS proxy: %v", err)
	}
	select {
	case <-rpc.createCanceled:
	case <-time.After(time.Second):
		t.Fatal("Stop did not cancel the in-flight CreateSocks RPC")
	}
	select {
	case err := <-startDone:
		if err != nil {
			t.Fatalf("Start after explicit Stop = %v, want nil", err)
		}
	case <-time.After(time.Second):
		t.Fatal("Start did not return after Stop canceled CreateSocks")
	}
	if !connection.closed.Load() {
		t.Fatal("accepted connection was not closed after CreateSocks cancellation")
	}
	if hasSocksProxy(registered.ID) {
		t.Fatalf("stopped SOCKS proxy %d remained in inventory", registered.ID)
	}
}

func TestSocksProxyCreateFailureClosesOnlyAcceptedConnection(t *testing.T) {
	resetSocksLifecycleTestState(t)
	const (
		tunnelID  = uint64(7330)
		sessionID = "socks-create-isolation"
	)
	failedConnection := &socksLifecycleTestConn{}
	siblingConnection := &socksBlockingReadConn{
		readStarted:  make(chan struct{}),
		closedSignal: make(chan struct{}),
	}
	listener := &socksSequenceListener{
		connections: make(chan net.Conn, 2),
		closed:      make(chan struct{}),
	}
	listener.connections <- failedConnection
	listener.connections <- siblingConnection
	rpc := &socksStartSequenceRPC{
		results:    make(chan socksStartCreateResult, 2),
		closeCalls: make(chan socksCloseObservation, 1),
	}
	rpc.results <- socksStartCreateResult{err: status.Error(codes.ResourceExhausted, "SOCKS tunnel quota reached")}
	rpc.results <- socksStartCreateResult{response: &sliverpb.Socks{TunnelID: tunnelID, SessionID: sessionID}}
	proxy := &TcpProxy{
		Rpc:      rpc,
		Session:  &clientpb.Session{ID: sessionID},
		Listener: listener,
	}
	registered := SocksProxies.Add(proxy)
	startDone := make(chan error, 1)
	go func() { startDone <- SocksProxies.Start(proxy) }()

	deadline := time.Now().Add(time.Second)
	for !failedConnection.closed.Load() && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if !failedConnection.closed.Load() {
		t.Fatal("CreateSocks quota failure did not close its accepted connection")
	}
	waitForSocksConnection(t, proxy, tunnelID)
	select {
	case <-siblingConnection.readStarted:
	case <-time.After(time.Second):
		t.Fatal("SOCKS proxy did not establish the next accepted connection")
	}
	if !hasSocksProxy(registered.ID) {
		t.Fatal("one CreateSocks failure removed the shared SOCKS proxy")
	}
	select {
	case err := <-startDone:
		t.Fatalf("one CreateSocks failure stopped shared proxy: %v", err)
	default:
	}
	if siblingConnection.closed.Load() {
		t.Fatal("one CreateSocks failure closed an established sibling connection")
	}

	if err := proxy.Stop(); err != nil {
		t.Fatalf("stop SOCKS proxy: %v", err)
	}
	select {
	case err := <-startDone:
		if err != nil {
			t.Fatalf("Start after explicit Stop = %v, want nil", err)
		}
	case <-time.After(time.Second):
		t.Fatal("SOCKS proxy did not stop after isolation test")
	}
}

type socksFlowNegotiationTestCase struct {
	name                 string
	serverCapabilities   uint64
	wantBindCapabilities uint64
	wantFlowEnabled      bool
}

func requireSocksCreateRequest(t *testing.T, rpc *socksStartSequenceRPC, sessionID string) {
	t.Helper()
	select {
	case request := <-rpc.createRequests:
		if request.SessionID != sessionID || request.Capabilities != sliverpb.CapabilitySocksFlowControlV1 {
			t.Fatalf("CreateSocks request = %+v", request)
		}
	case <-time.After(time.Second):
		t.Fatal("SOCKS proxy did not request flow-control capability")
	}
}

func requireSocksOwnershipBind(t *testing.T, rpc *socksStartSequenceRPC, tunnelID uint64, capabilities uint64) {
	t.Helper()
	select {
	case bind := <-rpc.streamSent:
		if bind.TunnelID != tunnelID || bind.Sequence != socksLifecycleBindSequence || bind.Capabilities != capabilities {
			t.Fatalf("SOCKS ownership bind = %+v", bind)
		}
	case <-time.After(time.Second):
		t.Fatal("SOCKS proxy did not echo negotiated capability in ownership bind")
	}
}

func requireSocksFlowEnabled(t *testing.T, proxy *TcpProxy, tunnelID uint64, want bool) {
	t.Helper()
	waitForSocksConnection(t, proxy, tunnelID)
	flow, ok := proxy.getSendFlow(tunnelID)
	if !ok {
		t.Fatal("negotiated SOCKS send flow was not registered")
	}
	flow.mu.Lock()
	flowEnabled := flow.enabled
	flow.mu.Unlock()
	if flowEnabled != want {
		t.Fatalf("SOCKS flow enabled = %t, want %t", flowEnabled, want)
	}
}

func stopSocksNegotiationProxy(t *testing.T, proxy *TcpProxy, startDone <-chan error) {
	t.Helper()
	if err := proxy.Stop(); err != nil {
		t.Fatalf("stop negotiated SOCKS proxy: %v", err)
	}
	select {
	case err := <-startDone:
		if err != nil {
			t.Fatalf("SOCKS proxy after stop = %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("negotiated SOCKS proxy did not stop")
	}
}

func runSocksFlowNegotiationTest(
	t *testing.T,
	test socksFlowNegotiationTestCase,
	tunnelID uint64,
	sessionID string,
) {
	t.Helper()
	resetSocksLifecycleTestState(t)
	connection := &socksBlockingReadConn{
		readStarted:  make(chan struct{}),
		closedSignal: make(chan struct{}),
	}
	listener := &socksQueuedListener{
		connection: connection,
		closed:     make(chan struct{}),
	}
	rpc := &socksStartSequenceRPC{
		results:        make(chan socksStartCreateResult, 1),
		closeCalls:     make(chan socksCloseObservation, 1),
		createRequests: make(chan *sliverpb.Socks, 1),
		streamSent:     make(chan *sliverpb.SocksData, 2),
	}
	rpc.results <- socksStartCreateResult{response: &sliverpb.Socks{
		TunnelID:     tunnelID,
		SessionID:    sessionID,
		Capabilities: test.serverCapabilities,
	}}
	proxy := &TcpProxy{
		Rpc:      rpc,
		Session:  &clientpb.Session{ID: sessionID},
		Listener: listener,
	}
	SocksProxies.Add(proxy)
	startDone := make(chan error, 1)
	go func() { startDone <- SocksProxies.Start(proxy) }()

	requireSocksCreateRequest(t, rpc, sessionID)
	requireSocksOwnershipBind(t, rpc, tunnelID, test.wantBindCapabilities)
	requireSocksFlowEnabled(t, proxy, tunnelID, test.wantFlowEnabled)
	stopSocksNegotiationProxy(t, proxy, startDone)
}

func TestSocksProxyNegotiatesAndEchoesFlowControlCapability(t *testing.T) {
	const (
		tunnelID          = uint64(7331)
		sessionID         = "socks-flow-negotiation"
		unknownCapability = uint64(1) << 63
	)
	tests := []socksFlowNegotiationTestCase{
		{
			name:                 "negotiated",
			serverCapabilities:   sliverpb.CapabilitySocksFlowControlV1 | unknownCapability,
			wantBindCapabilities: sliverpb.CapabilitySocksFlowControlV1,
			wantFlowEnabled:      true,
		},
		{
			name:                 "downgraded",
			serverCapabilities:   unknownCapability,
			wantBindCapabilities: 0,
			wantFlowEnabled:      false,
		},
	}

	for testIndex, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			responseTunnelID := tunnelID + uint64(testIndex)
			runSocksFlowNegotiationTest(t, test, responseTunnelID, sessionID)
		})
	}
}

func TestTcpProxyStopAfterCreateSocksClosesUnboundServerTunnel(t *testing.T) {
	resetSocksLifecycleTestState(t)
	const (
		tunnelID  = uint64(7331)
		sessionID = "socks-stop-after-create"
	)
	connection := &socksLifecycleTestConn{}
	listener := &socksQueuedListener{
		connection: connection,
		closed:     make(chan struct{}),
	}
	rpc := &socksStartLifecycleRPC{
		createResponse: &sliverpb.Socks{
			TunnelID:  tunnelID,
			SessionID: sessionID,
		},
		closeCalls: make(chan socksCloseObservation, 1),
	}
	proxy := &TcpProxy{
		Rpc:          rpc,
		Session:      &clientpb.Session{ID: sessionID},
		Listener:     listener,
		closeTimeout: time.Second,
	}
	rpc.beforeCreateReturn = func() {
		_ = proxy.Stop()
	}
	registered := SocksProxies.Add(proxy)

	if err := SocksProxies.Start(proxy); err != nil {
		t.Fatalf("Start after explicit Stop = %v, want nil", err)
	}
	select {
	case observation := <-rpc.closeCalls:
		if observation.tunnelID != tunnelID || observation.sessionID != sessionID {
			t.Fatalf("CloseSocks request = tunnel %d session %q, want tunnel %d session %q", observation.tunnelID, observation.sessionID, tunnelID, sessionID)
		}
		if !observation.hasDeadline {
			t.Fatal("CloseSocks fallback did not use a bounded context")
		}
		if observation.deadlineRemaining <= 0 || observation.deadlineRemaining > proxy.closeTimeout {
			t.Fatalf("CloseSocks deadline remaining = %s, want within (0, %s]", observation.deadlineRemaining, proxy.closeTimeout)
		}
	case <-time.After(time.Second):
		t.Fatal("Stop-after-CreateSocks race did not close the unbound server tunnel")
	}
	if !connection.closed.Load() {
		t.Fatal("accepted connection was not closed after Stop won the CreateSocks race")
	}
	if hasSocksProxy(registered.ID) {
		t.Fatalf("stopped SOCKS proxy %d remained in inventory", registered.ID)
	}
}

func TestSocksProxyReturnsUnexpectedReceiveFailure(t *testing.T) {
	resetSocksLifecycleTestState(t)
	wantErr := errors.New("test SOCKS receive failure")
	listener := &socksQueuedListener{closed: make(chan struct{})}
	listener.accepted.Store(true)
	proxy := &TcpProxy{
		Rpc:      &socksStartLifecycleRPC{recvErr: wantErr},
		Session:  &clientpb.Session{ID: "socks-receive-failure"},
		Listener: listener,
	}
	registered := SocksProxies.Add(proxy)
	if err := SocksProxies.Start(proxy); !errors.Is(err, wantErr) {
		t.Fatalf("Start receive error = %v, want %v", err, wantErr)
	}
	if hasSocksProxy(registered.ID) {
		t.Fatalf("receive failure retained SOCKS proxy %d in inventory", registered.ID)
	}
}

func resetSocksLifecycleTestState(t *testing.T) {
	t.Helper()
	for _, metadata := range SocksProxies.List() {
		SocksProxies.Remove(metadata.ID)
	}
	t.Cleanup(func() {
		for _, metadata := range SocksProxies.List() {
			SocksProxies.Remove(metadata.ID)
		}
	})
}

func hasSocksProxy(id uint64) bool {
	for _, metadata := range SocksProxies.List() {
		if metadata.ID == id {
			return true
		}
	}
	return false
}

func (connection *socksFinalReadConn) Read(buffer []byte) (int, error) {
	if connection.read.Swap(true) {
		return 0, io.EOF
	}
	return copy(buffer, connection.payload), io.EOF
}

func (connection *socksScriptedReadConn) Read(buffer []byte) (int, error) {
	if connection.next >= len(connection.payloads) {
		return 0, io.EOF
	}
	payload := connection.payloads[connection.next]
	connection.next++
	if connection.next == len(connection.payloads) {
		return copy(buffer, payload), io.EOF
	}
	return copy(buffer, payload), nil
}

func (connection *socksBurstReadConn) Read(buffer []byte) (int, error) {
	if connection.remaining == 0 {
		return 0, io.EOF
	}
	buffer[0] = byte(connection.remaining)
	connection.remaining--
	return 1, nil
}

func (connection *socksFullBufferReadConn) Read(buffer []byte) (int, error) {
	if connection.read.Swap(true) {
		return 0, io.EOF
	}
	for index := range buffer {
		buffer[index] = byte(index)
	}
	return len(buffer), nil
}

func TestTcpProxyStopOnlyClosesOwnedConnections(t *testing.T) {
	const (
		firstTunnelID  = uint64(701)
		secondTunnelID = uint64(702)
	)
	firstListener := &socksLifecycleTestListener{}
	secondListener := &socksLifecycleTestListener{}
	firstConnection := &socksLifecycleTestConn{}
	secondConnection := &socksLifecycleTestConn{}
	firstProxy := &TcpProxy{Listener: firstListener}
	secondProxy := &TcpProxy{Listener: secondListener}
	if !firstProxy.addConnection(firstTunnelID, firstConnection) {
		t.Fatal("failed to register first proxy connection")
	}
	if !secondProxy.addConnection(secondTunnelID, secondConnection) {
		t.Fatal("failed to register second proxy connection")
	}
	t.Cleanup(func() {
		_ = firstProxy.Stop()
		_ = secondProxy.Stop()
		SocksConnPool.Delete(firstTunnelID)
		SocksConnPool.Delete(secondTunnelID)
	})

	if err := firstProxy.Stop(); err != nil {
		t.Fatalf("stop first proxy: %v", err)
	}
	if !firstListener.closed.Load() {
		t.Fatal("first proxy listener was not closed")
	}
	if !firstConnection.closed.Load() {
		t.Fatal("first proxy connection was not closed")
	}
	if secondListener.closed.Load() {
		t.Fatal("stopping first proxy closed the second proxy listener")
	}
	if secondConnection.closed.Load() {
		t.Fatal("stopping first proxy closed the second proxy connection")
	}
	if _, ok := SocksConnPool.Load(firstTunnelID); ok {
		t.Fatal("stopped proxy connection remained in the compatibility pool")
	}
	if connection, ok := SocksConnPool.Load(secondTunnelID); !ok || connection != secondConnection {
		t.Fatal("unrelated proxy connection was removed from the compatibility pool")
	}
}

func TestTcpProxyRejectsConnectionAfterStop(t *testing.T) {
	proxy := &TcpProxy{Listener: &socksLifecycleTestListener{}}
	if err := proxy.Stop(); err != nil {
		t.Fatalf("stop proxy: %v", err)
	}
	connection := &socksLifecycleTestConn{}
	if proxy.addConnection(703, connection) {
		t.Fatal("stopped proxy accepted a new connection")
	}
	if !connection.closed.Load() {
		t.Fatal("connection rejected after stop was not closed")
	}
	if _, ok := SocksConnPool.Load(uint64(703)); ok {
		SocksConnPool.Delete(uint64(703))
		t.Fatal("connection rejected after stop was added to the compatibility pool")
	}
}

type socksSerializationTestStream struct {
	release  <-chan struct{}
	active   atomic.Int32
	maximum  atomic.Int32
	sends    atomic.Int32
	closes   atomic.Int32
	entered  chan struct{}
	enterOne sync.Once
}

func (stream *socksSerializationTestStream) Send(*sliverpb.SocksData) error {
	stream.call(&stream.sends)
	return nil
}

func (stream *socksSerializationTestStream) CloseSend() error {
	stream.call(&stream.closes)
	return nil
}

func (stream *socksSerializationTestStream) call(counter *atomic.Int32) {
	active := stream.active.Add(1)
	for {
		maximum := stream.maximum.Load()
		if active <= maximum || stream.maximum.CompareAndSwap(maximum, active) {
			break
		}
	}
	stream.enterOne.Do(func() { close(stream.entered) })
	<-stream.release
	counter.Add(1)
	stream.active.Add(-1)
}

func TestSerializedSocksStreamSerializesSendAndClose(t *testing.T) {
	release := make(chan struct{})
	underlying := &socksSerializationTestStream{
		release: release,
		entered: make(chan struct{}),
	}
	stream := &serializedSocksStream{stream: underlying}
	const sends = 24
	var waitGroup sync.WaitGroup
	waitGroup.Add(sends + 1)
	for index := 0; index < sends; index++ {
		go func(sequence int) {
			defer waitGroup.Done()
			if err := stream.Send(&sliverpb.SocksData{Sequence: uint64(sequence)}); err != nil {
				t.Errorf("send frame %d: %v", sequence, err)
			}
		}(index)
	}
	go func() {
		defer waitGroup.Done()
		if err := stream.CloseSend(); err != nil {
			t.Errorf("close stream: %v", err)
		}
	}()

	select {
	case <-underlying.entered:
	case <-time.After(time.Second):
		t.Fatal("no stream operation started")
	}
	time.Sleep(20 * time.Millisecond)
	if maximum := underlying.maximum.Load(); maximum != 1 {
		t.Fatalf("concurrent underlying stream operations = %d, want 1", maximum)
	}
	close(release)
	waitDone := make(chan struct{})
	go func() {
		waitGroup.Wait()
		close(waitDone)
	}()
	select {
	case <-waitDone:
	case <-time.After(time.Second):
		t.Fatal("serialized stream operations did not finish")
	}
	if sends := underlying.sends.Load(); sends != 24 {
		t.Fatalf("underlying sends = %d, want 24", sends)
	}
	if closes := underlying.closes.Load(); closes != 1 {
		t.Fatalf("underlying closes = %d, want 1", closes)
	}
	if maximum := underlying.maximum.Load(); maximum != 1 {
		t.Fatalf("maximum concurrent underlying operations = %d, want 1", maximum)
	}
}

type socksCloseObservation struct {
	tunnelID          uint64
	sessionID         string
	hasDeadline       bool
	deadlineRemaining time.Duration
}

type socksLifecycleTestRPC struct {
	rpcpb.SliverRPCClient
	closeCalls chan socksCloseObservation
	blockClose bool
}

func (rpc *socksLifecycleTestRPC) CloseSocks(ctx context.Context, request *sliverpb.Socks, _ ...grpc.CallOption) (*commonpb.Empty, error) {
	deadline, hasDeadline := ctx.Deadline()
	observation := socksCloseObservation{
		tunnelID:    request.TunnelID,
		sessionID:   request.SessionID,
		hasDeadline: hasDeadline,
	}
	if hasDeadline {
		observation.deadlineRemaining = time.Until(deadline)
	}
	rpc.closeCalls <- observation
	if rpc.blockClose {
		<-ctx.Done()
		return nil, ctx.Err()
	}
	return &commonpb.Empty{}, nil
}

type socksLifecycleTestStream struct {
	sent      chan *sliverpb.SocksData
	sendCount int
	failAt    int
}

func (stream *socksLifecycleTestStream) Send(frame *sliverpb.SocksData) error {
	stream.sendCount++
	if stream.sent != nil {
		stream.sent <- proto.Clone(frame).(*sliverpb.SocksData)
	}
	if stream.failAt == stream.sendCount {
		return errors.New("test stream send failure")
	}
	return nil
}
func (*socksLifecycleTestStream) CloseSend() error { return nil }

type socksRetainingTestStream struct {
	sent []*sliverpb.SocksData
}

func (stream *socksRetainingTestStream) Send(frame *sliverpb.SocksData) error {
	stream.sent = append(stream.sent, frame)
	return nil
}

func (*socksRetainingTestStream) CloseSend() error { return nil }

//nolint:gocyclo // This test covers the complete failed-terminal fallback lifecycle.
func TestConnectFallsBackToUnaryCloseWhenTerminalSendFails(t *testing.T) {
	const (
		tunnelID  = uint64(704)
		sessionID = "socks-lifecycle-session"
	)
	rpc := &socksLifecycleTestRPC{closeCalls: make(chan socksCloseObservation, 1)}
	proxy := &TcpProxy{Rpc: rpc}
	underlyingStream := &socksLifecycleTestStream{
		sent:   make(chan *sliverpb.SocksData, 2),
		failAt: 2,
	}
	stream := &serializedSocksStream{stream: underlyingStream}
	proxyConnection, userConnection := net.Pipe()
	done := make(chan struct{})
	go func() {
		connect(proxy, proxyConnection, stream, &sliverpb.SocksData{
			TunnelID: tunnelID,
			Request:  &commonpb.Request{SessionID: sessionID},
		})
		close(done)
	}()
	waitForSocksConnection(t, proxy, tunnelID)
	select {
	case bind := <-underlyingStream.sent:
		if bind.TunnelID != tunnelID || bind.Sequence != socksLifecycleBindSequence || len(bind.Data) != 0 || bind.CloseConn {
			t.Fatalf("SOCKS bind frame = %+v", bind)
		}
		if bind.Request == nil || bind.Request.SessionID != sessionID {
			t.Fatalf("SOCKS bind request = %+v, want session %q", bind.Request, sessionID)
		}
	case <-time.After(time.Second):
		t.Fatal("connection handler did not bind the tunnel before reading a SOCKS greeting")
	}
	if err := userConnection.Close(); err != nil {
		t.Fatalf("close user connection: %v", err)
	}
	select {
	case terminal := <-underlyingStream.sent:
		if !terminal.CloseConn || terminal.Sequence != 0 || terminal.TunnelID != tunnelID {
			t.Fatalf("failed terminal frame = %+v, want tunnel %d sequence 0", terminal, tunnelID)
		}
	case <-time.After(time.Second):
		t.Fatal("connection teardown did not attempt a same-stream terminal")
	}

	select {
	case observation := <-rpc.closeCalls:
		if observation.tunnelID != tunnelID {
			t.Fatalf("closed tunnel ID = %d, want %d", observation.tunnelID, tunnelID)
		}
		if observation.sessionID != sessionID {
			t.Fatalf("closed tunnel session ID = %q, want %q", observation.sessionID, sessionID)
		}
		if !observation.hasDeadline {
			t.Fatal("CloseSocks RPC did not have a deadline")
		}
		if observation.deadlineRemaining <= 0 || observation.deadlineRemaining > socksTunnelCloseTimeout {
			t.Fatalf("CloseSocks deadline remaining = %s", observation.deadlineRemaining)
		}
	case <-time.After(time.Second):
		t.Fatal("connection teardown did not close the server SOCKS tunnel")
	}
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("connection handler did not finish")
	}
	if _, ok := proxy.getConnection(tunnelID); ok {
		t.Fatal("closed connection remained owned by its proxy")
	}
	if _, ok := SocksConnPool.Load(tunnelID); ok {
		SocksConnPool.Delete(tunnelID)
		t.Fatal("closed connection remained in the compatibility pool")
	}
}

//nolint:gocyclo // This test verifies timeout, terminal, RPC fallback, and socket cleanup together.
func TestConnectNoGreetingExpiresLocallyAndSendsTerminal(t *testing.T) {
	const (
		tunnelID  = uint64(705)
		sessionID = "socks-no-greeting-session"
	)
	rpc := &socksLifecycleTestRPC{closeCalls: make(chan socksCloseObservation, 1)}
	proxy := &TcpProxy{Rpc: rpc, firstReadLease: 25 * time.Millisecond}
	underlyingStream := &socksLifecycleTestStream{sent: make(chan *sliverpb.SocksData, 2)}
	proxyConnection, userConnection := net.Pipe()
	t.Cleanup(func() { _ = userConnection.Close() })
	done := make(chan struct{})
	go func() {
		connect(proxy, proxyConnection, &serializedSocksStream{stream: underlyingStream}, &sliverpb.SocksData{
			TunnelID: tunnelID,
			Request:  &commonpb.Request{SessionID: sessionID},
		})
		close(done)
	}()

	select {
	case marker := <-underlyingStream.sent:
		if marker.Sequence != socksLifecycleBindSequence || marker.CloseConn || len(marker.Data) != 0 {
			t.Fatalf("lifecycle marker = %+v", marker)
		}
	case <-time.After(time.Second):
		t.Fatal("connection handler did not send lifecycle marker")
	}
	select {
	case terminal := <-underlyingStream.sent:
		if !terminal.CloseConn || terminal.Sequence != 0 || terminal.TunnelID != tunnelID {
			t.Fatalf("no-greeting terminal = %+v", terminal)
		}
	case <-time.After(time.Second):
		t.Fatal("no-greeting connection did not expire with a terminal")
	}
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("no-greeting connection handler did not terminate")
	}
	// net.Pipe does not expose close state; a peer read is the observable proof
	// that local teardown completed.
	if err := userConnection.SetReadDeadline(time.Now().Add(time.Second)); err == nil {
		if _, err := userConnection.Read(make([]byte, 1)); err == nil {
			t.Fatal("no-greeting peer remained open after first-payload lease")
		}
	}
	select {
	case observation := <-rpc.closeCalls:
		t.Fatalf("successful no-greeting terminal unexpectedly used unary close: %+v", observation)
	default:
	}
}

//nolint:gocyclo // The mixed-version contract requires one connected stream lifecycle assertion.
func TestConnectLifecycleMarkerIsLegacyServerCompatible(t *testing.T) {
	const tunnelID = uint64(7060)
	payload := []byte("legacy-server-sequence-zero")
	rpc := &socksLifecycleTestRPC{closeCalls: make(chan socksCloseObservation, 1)}
	proxy := &TcpProxy{Rpc: rpc}
	stream := &socksLifecycleTestStream{sent: make(chan *sliverpb.SocksData, 3)}
	connect(proxy, &socksFinalReadConn{payload: payload}, &serializedSocksStream{stream: stream}, &sliverpb.SocksData{
		TunnelID: tunnelID,
		Username: "legacy-user",
		Password: "legacy-password",
		Request:  &commonpb.Request{SessionID: "legacy-server-session"},
	})

	// Model the legacy server's sequence cache exactly: it retained arbitrary
	// future frames and drained only contiguous frames from sequence zero.
	pending := map[uint64]*sliverpb.SocksData{}
	expected := uint64(0)
	delivered := []*sliverpb.SocksData{}
	for index := 0; index < 3; index++ {
		frame := <-stream.sent
		if frame.Request != nil && frame.Request.Async {
			t.Fatal("lifecycle capability overloaded Request.Async")
		}
		pending[frame.Sequence] = frame
		for {
			next := pending[expected]
			if next == nil {
				break
			}
			delete(pending, expected)
			delivered = append(delivered, next)
			expected++
		}
	}
	if marker := pending[socksLifecycleBindSequence]; marker == nil || marker.CloseConn || len(marker.Data) != 0 {
		t.Fatalf("legacy sequence cache marker = %+v", marker)
	}
	if len(delivered) != 2 {
		t.Fatalf("legacy server delivered %d frames, want payload and terminal", len(delivered))
	}
	if delivered[0].Sequence != 0 || delivered[0].CloseConn || string(delivered[0].Data) != string(payload) ||
		delivered[0].Username != "legacy-user" || delivered[0].Password != "legacy-password" {
		t.Fatalf("legacy server first frame = %+v", delivered[0])
	}
	if delivered[1].Sequence != 1 || !delivered[1].CloseConn || len(delivered[1].Data) != 0 {
		t.Fatalf("legacy server terminal = %+v", delivered[1])
	}
}

//nolint:gocyclo // Final-byte ordering and cleanup are intentionally asserted as one lifecycle.
func TestConnectSendsFinalDataThenTerminalWithoutUnaryClose(t *testing.T) {
	const (
		tunnelID  = uint64(706)
		sessionID = "socks-ordered-terminal-session"
	)
	rpc := &socksLifecycleTestRPC{closeCalls: make(chan socksCloseObservation, 1)}
	proxy := &TcpProxy{Rpc: rpc}
	underlyingStream := &socksLifecycleTestStream{sent: make(chan *sliverpb.SocksData, 3)}
	stream := &serializedSocksStream{stream: underlyingStream}
	proxyConnection, userConnection := net.Pipe()
	done := make(chan struct{})
	go func() {
		connect(proxy, proxyConnection, stream, &sliverpb.SocksData{
			TunnelID: tunnelID,
			Username: "one-shot-user",
			Password: "one-shot-password",
			Request:  &commonpb.Request{SessionID: sessionID},
		})
		close(done)
	}()
	waitForSocksConnection(t, proxy, tunnelID)
	select {
	case bind := <-underlyingStream.sent:
		if bind.CloseConn || bind.Sequence != socksLifecycleBindSequence || len(bind.Data) != 0 || bind.Username != "one-shot-user" || bind.Password != "one-shot-password" {
			t.Fatalf("SOCKS bind frame = %+v", bind)
		}
	case <-time.After(time.Second):
		t.Fatal("connection handler did not send ownership bind")
	}

	payload := []byte("final SOCKS payload")
	if _, err := userConnection.Write(payload); err != nil {
		t.Fatalf("write final SOCKS payload: %v", err)
	}
	if err := userConnection.Close(); err != nil {
		t.Fatalf("close user connection: %v", err)
	}
	select {
	case data := <-underlyingStream.sent:
		if data.CloseConn || data.Sequence != 0 || string(data.Data) != string(payload) || data.Username != "one-shot-user" || data.Password != "one-shot-password" {
			t.Fatalf("final data frame = %+v, want sequence 0 payload %q", data, payload)
		}
	case <-time.After(time.Second):
		t.Fatal("connection handler did not send final SOCKS payload")
	}
	select {
	case terminal := <-underlyingStream.sent:
		if !terminal.CloseConn || terminal.Sequence != 1 || len(terminal.Data) != 0 || terminal.Username != "" || terminal.Password != "" {
			t.Fatalf("terminal frame = %+v, want sequence 1 after final payload", terminal)
		}
	case <-time.After(time.Second):
		t.Fatal("connection handler did not send terminal after final payload")
	}
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("connection handler did not finish after ordered terminal")
	}
	select {
	case observation := <-rpc.closeCalls:
		t.Fatalf("successful terminal unexpectedly used unary CloseSocks: %+v", observation)
	default:
	}
	if _, ok := proxy.getConnection(tunnelID); ok {
		t.Fatal("closed connection remained owned by its proxy")
	}
	if _, ok := SocksConnPool.Load(tunnelID); ok {
		SocksConnPool.Delete(tunnelID)
		t.Fatal("closed connection remained in the compatibility pool")
	}
}

func TestConnectPreservesReadBytesWhenDeadlineClearFails(t *testing.T) {
	const tunnelID = uint64(707)
	payload := []byte("bytes-returned-with-eof")
	rpc := &socksLifecycleTestRPC{closeCalls: make(chan socksCloseObservation, 1)}
	proxy := &TcpProxy{Rpc: rpc}
	underlyingStream := &socksLifecycleTestStream{sent: make(chan *sliverpb.SocksData, 4)}
	connection := &socksFinalReadDeadlineErrorConn{socksFinalReadConn: socksFinalReadConn{payload: payload}}

	connect(proxy, connection, &serializedSocksStream{stream: underlyingStream}, &sliverpb.SocksData{
		TunnelID: tunnelID,
		Request:  &commonpb.Request{SessionID: "final-read-session"},
	})
	receive := func(label string) *sliverpb.SocksData {
		t.Helper()
		select {
		case frame := <-underlyingStream.sent:
			return frame
		case <-time.After(time.Second):
			t.Fatalf("timed out waiting for %s frame", label)
			return nil
		}
	}
	bind := receive("bind")
	data := receive("final data")
	terminal := receive("terminal")
	if bind.CloseConn || bind.Sequence != socksLifecycleBindSequence || len(bind.Data) != 0 {
		t.Fatalf("bind frame = %+v", bind)
	}
	if data.CloseConn || data.Sequence != 0 || string(data.Data) != string(payload) {
		t.Fatalf("final data frame = %+v, want sequence 0 payload %q", data, payload)
	}
	if !terminal.CloseConn || terminal.Sequence != 1 {
		t.Fatalf("terminal frame = %+v, want sequence 1", terminal)
	}
	select {
	case extra := <-underlyingStream.sent:
		t.Fatalf("unexpected extra SOCKS frame after terminal: %+v", extra)
	default:
	}
	select {
	case observation := <-rpc.closeCalls:
		t.Fatalf("successful terminal unexpectedly used unary close: %+v", observation)
	default:
	}
}

//nolint:gocyclo // Message identity, payload ownership, credentials, ordering, and terminal state form one send contract.
func TestConnectDoesNotMutateSentFramesOrPooledPayloads(t *testing.T) {
	const (
		tunnelID  = uint64(7071)
		sessionID = "immutable-send-session"
	)
	initial := &sliverpb.SocksData{
		Username: "immutable-user",
		Password: "immutable-password",
		TunnelID: tunnelID,
		Request:  &commonpb.Request{SessionID: sessionID},
	}
	underlyingStream := &socksRetainingTestStream{}
	connection := &socksScriptedReadConn{payloads: [][]byte{[]byte("first"), []byte("later")}}

	connect(
		&TcpProxy{Rpc: &socksLifecycleTestRPC{closeCalls: make(chan socksCloseObservation, 1)}},
		connection,
		&serializedSocksStream{stream: underlyingStream},
		initial,
	)
	if len(underlyingStream.sent) != 4 {
		t.Fatalf("sent %d frames, want bind, two data frames, and terminal", len(underlyingStream.sent))
	}
	bind, first, second, terminal := underlyingStream.sent[0], underlyingStream.sent[1], underlyingStream.sent[2], underlyingStream.sent[3]
	if bind == first || first == second || second == terminal {
		t.Fatal("SOCKS sends reused a protobuf message")
	}
	if string(first.Data) != "first" || first.Sequence != 0 || first.Username != initial.Username || first.Password != initial.Password {
		t.Fatalf("first retained data frame mutated: %+v", first)
	}
	if string(second.Data) != "later" || second.Sequence != 1 || second.Username != "" || second.Password != "" {
		t.Fatalf("second retained data frame = %+v", second)
	}
	if !terminal.CloseConn || terminal.Sequence != 2 || len(terminal.Data) != 0 {
		t.Fatalf("terminal retained frame = %+v", terminal)
	}
	if len(initial.Data) != 0 || initial.Sequence != 0 || initial.Username != "immutable-user" || initial.Password != "immutable-password" {
		t.Fatalf("input frame mutated after sends: %+v", initial)
	}
}

func TestConnectReliesOnStreamBackpressureWithoutArtificialPacing(t *testing.T) {
	const (
		tunnelID = uint64(708)
		frames   = socksFlowControlWindow + 1
	)
	rpc := &socksLifecycleTestRPC{closeCalls: make(chan socksCloseObservation, 1)}
	proxy := &TcpProxy{Rpc: rpc}
	underlyingStream := &socksLifecycleTestStream{sent: make(chan *sliverpb.SocksData, frames+2)}
	connection := &socksBurstReadConn{remaining: frames}

	started := time.Now()
	connect(proxy, connection, &serializedSocksStream{stream: underlyingStream}, &sliverpb.SocksData{
		TunnelID: tunnelID,
		Request:  &commonpb.Request{SessionID: "unpaced-read-session"},
	})
	if elapsed := time.Since(started); elapsed >= time.Second {
		t.Fatalf("%d immediately readable legacy SOCKS frames took %s; an unnegotiated flow window must not block", frames, elapsed)
	}

	bind := <-underlyingStream.sent
	if bind.CloseConn || bind.Sequence != socksLifecycleBindSequence || len(bind.Data) != 0 {
		t.Fatalf("bind frame = %+v", bind)
	}
	for sequence := 0; sequence < frames; sequence++ {
		data := <-underlyingStream.sent
		if data.CloseConn || data.Sequence != uint64(sequence) || len(data.Data) != 1 {
			t.Fatalf("data frame %d = %+v", sequence, data)
		}
	}
	terminal := <-underlyingStream.sent
	if !terminal.CloseConn || terminal.Sequence != frames {
		t.Fatalf("terminal frame = %+v, want sequence %d", terminal, frames)
	}
	select {
	case observation := <-rpc.closeCalls:
		t.Fatalf("successful terminal unexpectedly used unary close: %+v", observation)
	default:
	}
}

func TestConnectReadsUpToMaximumTunnelFrame(t *testing.T) {
	const tunnelID = uint64(709)
	rpc := &socksLifecycleTestRPC{closeCalls: make(chan socksCloseObservation, 1)}
	proxy := &TcpProxy{Rpc: rpc}
	underlyingStream := &socksLifecycleTestStream{sent: make(chan *sliverpb.SocksData, 3)}
	connection := &socksFullBufferReadConn{}

	connect(proxy, connection, &serializedSocksStream{stream: underlyingStream}, &sliverpb.SocksData{
		TunnelID: tunnelID,
		Request:  &commonpb.Request{SessionID: "maximum-frame-session"},
	})
	bind := <-underlyingStream.sent
	data := <-underlyingStream.sent
	terminal := <-underlyingStream.sent
	if bind.CloseConn || bind.Sequence != socksLifecycleBindSequence || len(bind.Data) != 0 {
		t.Fatalf("bind frame = %+v", bind)
	}
	if data.CloseConn || data.Sequence != 0 || len(data.Data) != sliverpb.MaxTunnelFrameBytes {
		t.Fatalf("data frame = sequence %d, close %t, length %d; want sequence 0, open, length %d", data.Sequence, data.CloseConn, len(data.Data), sliverpb.MaxTunnelFrameBytes)
	}
	if !terminal.CloseConn || terminal.Sequence != 1 {
		t.Fatalf("terminal frame = %+v, want sequence 1", terminal)
	}
}

func TestCloseSocksTunnelIsBounded(t *testing.T) {
	rpc := &socksLifecycleTestRPC{
		closeCalls: make(chan socksCloseObservation, 1),
		blockClose: true,
	}
	const timeout = 25 * time.Millisecond
	started := time.Now()
	closeSocksTunnel(rpc, 705, "bounded-close-session", timeout)
	elapsed := time.Since(started)
	if elapsed < timeout {
		t.Fatalf("CloseSocks returned in %s before its %s timeout", elapsed, timeout)
	}
	if elapsed > 500*time.Millisecond {
		t.Fatalf("CloseSocks exceeded bounded cleanup window: %s", elapsed)
	}
	select {
	case observation := <-rpc.closeCalls:
		if observation.tunnelID != 705 {
			t.Fatalf("closed tunnel ID = %d, want 705", observation.tunnelID)
		}
		if observation.sessionID != "bounded-close-session" {
			t.Fatalf("closed tunnel session ID = %q, want bounded-close-session", observation.sessionID)
		}
		if !observation.hasDeadline {
			t.Fatal("CloseSocks RPC did not have a deadline")
		}
	case <-time.After(time.Second):
		t.Fatal("CloseSocks RPC was not called")
	}
}

func waitForSocksConnection(t *testing.T, proxy *TcpProxy, tunnelID uint64) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if _, ok := proxy.getConnection(tunnelID); ok {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal(errors.New("SOCKS connection was not registered"))
}

type socksReceiveTestResult struct {
	data *sliverpb.SocksData
	err  error
}

type socksReceiveTestStream struct {
	results   chan socksReceiveTestResult
	recvCalls chan struct{}
}

func (stream *socksReceiveTestStream) Recv() (*sliverpb.SocksData, error) {
	stream.recvCalls <- struct{}{}
	result := <-stream.results
	return result.data, result.err
}

type socksBlockingWriteConn struct {
	socksLifecycleTestConn
	writeStarted chan struct{}
	closedSignal chan struct{}
	writeOnce    sync.Once
	closeOnce    sync.Once
}

func (connection *socksBlockingWriteConn) Write([]byte) (int, error) {
	connection.writeOnce.Do(func() { close(connection.writeStarted) })
	<-connection.closedSignal
	return 0, net.ErrClosed
}

func (connection *socksBlockingWriteConn) Close() error {
	connection.closed.Store(true)
	connection.closeOnce.Do(func() { close(connection.closedSignal) })
	return nil
}

type socksRecordingWriteConn struct {
	socksLifecycleTestConn
	writes       chan []byte
	closedSignal chan struct{}
	closeOnce    sync.Once
}

type socksGatedWriteConn struct {
	socksLifecycleTestConn
	writes       chan []byte
	writeStarted chan struct{}
	writeGate    chan struct{}
	closedSignal chan struct{}
	writeOnce    sync.Once
	closeOnce    sync.Once
}

type socksPartialWriteConn struct {
	socksLifecycleTestConn
	mu                  sync.Mutex
	writes              int
	finalWriteStarted   chan struct{}
	finalWriteRelease   chan struct{}
	finalWriteStartOnce sync.Once
}

func (connection *socksGatedWriteConn) Write(data []byte) (int, error) {
	connection.writeOnce.Do(func() { close(connection.writeStarted) })
	select {
	case <-connection.writeGate:
	case <-connection.closedSignal:
		return 0, net.ErrClosed
	}
	if connection.closed.Load() {
		return 0, net.ErrClosed
	}
	payloadCopy := append([]byte(nil), data...)
	connection.writes <- payloadCopy
	return len(data), nil
}

func (connection *socksGatedWriteConn) Close() error {
	connection.closed.Store(true)
	connection.closeOnce.Do(func() { close(connection.closedSignal) })
	return nil
}

func (connection *socksPartialWriteConn) Write(data []byte) (int, error) {
	connection.mu.Lock()
	connection.writes++
	write := connection.writes
	connection.mu.Unlock()
	if write == socksFlowAcknowledgementGap && len(data) > 1 {
		return 1, nil
	}
	if write == socksFlowAcknowledgementGap+1 {
		connection.finalWriteStartOnce.Do(func() { close(connection.finalWriteStarted) })
		<-connection.finalWriteRelease
	}
	return len(data), nil
}

type socksBlockingReadConn struct {
	socksLifecycleTestConn
	readStarted  chan struct{}
	closedSignal chan struct{}
	readOnce     sync.Once
	closeOnce    sync.Once
}

func (connection *socksBlockingReadConn) Read([]byte) (int, error) {
	connection.readOnce.Do(func() { close(connection.readStarted) })
	<-connection.closedSignal
	return 0, net.ErrClosed
}

func (connection *socksBlockingReadConn) Close() error {
	connection.closed.Store(true)
	connection.closeOnce.Do(func() { close(connection.closedSignal) })
	return nil
}

func (connection *socksRecordingWriteConn) Write(data []byte) (int, error) {
	if connection.closed.Load() {
		return 0, net.ErrClosed
	}
	payloadCopy := append([]byte(nil), data...)
	connection.writes <- payloadCopy
	return len(data), nil
}

func (connection *socksRecordingWriteConn) Close() error {
	connection.closed.Store(true)
	connection.closeOnce.Do(func() { close(connection.closedSignal) })
	return nil
}

//nolint:gocyclo // The overload and sibling-isolation assertions share one deterministic setup.
func TestSocksReceiveLoopOverloadedConnectionDoesNotBlockHealthyConnection(t *testing.T) {
	const (
		stalledTunnelID = uint64(801)
		healthyTunnelID = uint64(802)
	)
	stalledConnection := &socksBlockingWriteConn{
		writeStarted: make(chan struct{}),
		closedSignal: make(chan struct{}),
	}
	healthyConnection := &socksRecordingWriteConn{
		writes:       make(chan []byte, 2),
		closedSignal: make(chan struct{}),
	}
	proxy := &TcpProxy{}
	if !proxy.addConnection(stalledTunnelID, stalledConnection) {
		t.Fatal("failed to add stalled SOCKS connection")
	}
	if !proxy.addConnection(healthyTunnelID, healthyConnection) {
		t.Fatal("failed to add healthy SOCKS connection")
	}
	_, stalledReceiver, ok := proxy.getReceiveQueue(stalledTunnelID)
	if !ok {
		t.Fatal("stalled SOCKS receive actor was not registered")
	}
	_, healthyReceiver, ok := proxy.getReceiveQueue(healthyTunnelID)
	if !ok {
		t.Fatal("healthy SOCKS receive actor was not registered")
	}

	stream := &socksReceiveTestStream{
		results:   make(chan socksReceiveTestResult, socksReceiveFrameLimit+8),
		recvCalls: make(chan struct{}, socksReceiveFrameLimit+8),
	}
	if !proxy.startReceiveLoop(stream) {
		t.Fatal("failed to start SOCKS receive loop")
	}
	waitForTestSignal(t, stream.recvCalls, "initial SOCKS stream receive")

	stream.results <- socksReceiveTestResult{data: &sliverpb.SocksData{
		TunnelID: stalledTunnelID,
		Sequence: 0,
		Data:     []byte{0},
	}}
	waitForTestSignal(t, stalledConnection.writeStarted, "stalled SOCKS connection write")
	waitForTestSignal(t, stream.recvCalls, "SOCKS receive after first stalled frame")
	for sequence := 1; sequence < socksReceiveFrameLimit; sequence++ {
		stream.results <- socksReceiveTestResult{data: &sliverpb.SocksData{
			TunnelID: stalledTunnelID,
			Sequence: uint64(sequence),
			Data:     []byte{byte(sequence)},
		}}
		waitForTestSignal(t, stream.recvCalls, "SOCKS receive after queued stalled frame")
	}

	stream.results <- socksReceiveTestResult{data: &sliverpb.SocksData{
		TunnelID: stalledTunnelID,
		Sequence: socksReceiveFrameLimit,
		Data:     []byte("overflow"),
	}}
	waitForTestSignal(t, stream.recvCalls, "SOCKS receive after stalled overflow")
	select {
	case <-stalledConnection.closedSignal:
	case <-time.After(time.Second):
		t.Fatal("overloaded SOCKS connection was not closed")
	}
	select {
	case <-stalledReceiver.finished:
	default:
		t.Fatal("overloaded SOCKS delivery actor was not joined before dispatch resumed")
	}

	for sequence, payload := range [][]byte{[]byte("healthy-1"), []byte("healthy-2")} {
		stream.results <- socksReceiveTestResult{data: &sliverpb.SocksData{
			TunnelID: healthyTunnelID,
			Sequence: uint64(sequence),
			Data:     payload,
		}}
		waitForTestSignal(t, stream.recvCalls, "SOCKS receive after healthy frame")
	}
	stream.results <- socksReceiveTestResult{data: &sliverpb.SocksData{
		TunnelID:  healthyTunnelID,
		Sequence:  2,
		CloseConn: true,
	}}
	waitForTestSignal(t, stream.recvCalls, "SOCKS receive after healthy terminal")

	for _, want := range []string{"healthy-1", "healthy-2"} {
		select {
		case got := <-healthyConnection.writes:
			if string(got) != want {
				t.Fatalf("healthy SOCKS delivery = %q, want %q", got, want)
			}
		case <-time.After(time.Second):
			t.Fatalf("healthy SOCKS connection did not receive %q", want)
		}
	}
	select {
	case <-healthyConnection.closedSignal:
	case <-time.After(time.Second):
		t.Fatal("healthy SOCKS terminal did not close after ordered data")
	}

	stream.results <- socksReceiveTestResult{err: io.EOF}
	if err := proxy.Stop(); err != nil {
		t.Fatalf("stop SOCKS receive test proxy: %v", err)
	}
	select {
	case <-healthyReceiver.finished:
	default:
		t.Fatal("SOCKS proxy shutdown returned before healthy delivery actor exited")
	}
}

func TestSocksReceiveLoopRejectsInvalidAcknowledgementAtExactTunnelScope(t *testing.T) {
	tests := []struct {
		name       string
		negotiate  bool
		sentFrames uint64
		frame      *sliverpb.SocksData
	}{
		{
			name:       "future acknowledgement",
			negotiate:  true,
			sentFrames: 1,
			frame:      &sliverpb.SocksData{Ack: 2},
		},
		{
			name:       "mixed data and acknowledgement",
			negotiate:  true,
			sentFrames: 1,
			frame:      &sliverpb.SocksData{Ack: 1, Data: []byte("invalid")},
		},
		{
			name:       "acknowledgement with request metadata",
			negotiate:  true,
			sentFrames: 1,
			frame:      &sliverpb.SocksData{Ack: 1, Request: &commonpb.Request{SessionID: "unexpected"}},
		},
		{
			name:  "unnegotiated acknowledgement",
			frame: &sliverpb.SocksData{Ack: 1},
		},
	}

	for testIndex, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			invalidTunnelID := uint64(0x8100 + testIndex*2)
			healthyTunnelID := invalidTunnelID + 1
			invalidConnection := &socksLifecycleTestConn{}
			healthyConnection := &socksLifecycleTestConn{}
			proxy := &TcpProxy{}
			if !proxy.addConnection(invalidTunnelID, invalidConnection) || !proxy.addConnection(healthyTunnelID, healthyConnection) {
				t.Fatal("failed to register SOCKS acknowledgement test connections")
			}
			flow, ok := proxy.getSendFlow(invalidTunnelID)
			if !ok {
				t.Fatal("invalid tunnel send flow was not registered")
			}
			if test.negotiate {
				flow.enable()
			}
			for sequence := uint64(0); sequence < test.sentFrames; sequence++ {
				if err := flow.recordSent(sequence); err != nil {
					t.Fatalf("record sent sequence %d: %v", sequence, err)
				}
			}

			stream := &socksReceiveTestStream{
				results:   make(chan socksReceiveTestResult, 2),
				recvCalls: make(chan struct{}, 2),
			}
			if !proxy.startReceiveLoop(stream) {
				t.Fatal("failed to start SOCKS acknowledgement receive loop")
			}
			waitForTestSignal(t, stream.recvCalls, "initial acknowledgement receive")
			frame := proto.Clone(test.frame).(*sliverpb.SocksData)
			frame.TunnelID = invalidTunnelID
			stream.results <- socksReceiveTestResult{data: frame}
			waitForTestSignal(t, stream.recvCalls, "receive after invalid acknowledgement")

			if _, ok := proxy.getConnection(invalidTunnelID); ok {
				t.Fatal("invalid acknowledgement retained its exact SOCKS connection")
			}
			if !invalidConnection.closed.Load() {
				t.Fatal("invalid acknowledgement did not close its exact local connection")
			}
			if got, ok := proxy.getConnection(healthyTunnelID); !ok || got != healthyConnection {
				t.Fatalf("invalid acknowledgement disturbed healthy sibling: got=%p present=%t", got, ok)
			}
			if healthyConnection.closed.Load() {
				t.Fatal("invalid acknowledgement closed healthy sibling")
			}

			stream.results <- socksReceiveTestResult{err: io.EOF}
			if err := proxy.Stop(); err != nil {
				t.Fatalf("stop acknowledgement test proxy: %v", err)
			}
		})
	}
}

func TestSocksReceiveLoopAppliesValidAcknowledgementOutOfBand(t *testing.T) {
	const tunnelID = uint64(0x8200)
	connection := &socksLifecycleTestConn{}
	proxy := &TcpProxy{}
	if !proxy.addConnection(tunnelID, connection) {
		t.Fatal("failed to register SOCKS acknowledgement connection")
	}
	flow, ok := proxy.getSendFlow(tunnelID)
	if !ok {
		t.Fatal("SOCKS send flow was not registered")
	}
	flow.enable()
	for sequence := uint64(0); sequence < socksFlowControlWindow; sequence++ {
		if err := flow.recordSent(sequence); err != nil {
			t.Fatalf("record sent sequence %d: %v", sequence, err)
		}
	}
	waitDone := make(chan error, 1)
	go func() { waitDone <- flow.wait() }()
	select {
	case err := <-waitDone:
		t.Fatalf("full flow window returned before acknowledgement: %v", err)
	case <-time.After(25 * time.Millisecond):
	}

	stream := &socksReceiveTestStream{
		results:   make(chan socksReceiveTestResult, 2),
		recvCalls: make(chan struct{}, 2),
	}
	if !proxy.startReceiveLoop(stream) {
		t.Fatal("failed to start SOCKS acknowledgement receive loop")
	}
	waitForTestSignal(t, stream.recvCalls, "initial acknowledgement receive")
	stream.results <- socksReceiveTestResult{data: &sliverpb.SocksData{
		TunnelID: tunnelID,
		Ack:      socksFlowAcknowledgementGap,
	}}
	waitForTestSignal(t, stream.recvCalls, "receive after valid acknowledgement")
	select {
	case err := <-waitDone:
		if err != nil {
			t.Fatalf("flow wait after valid acknowledgement: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("valid acknowledgement did not release per-tunnel send credit")
	}
	if _, receiver, ok := proxy.getReceiveQueue(tunnelID); !ok {
		t.Fatal("valid acknowledgement removed its SOCKS connection")
	} else {
		receiver.mu.Lock()
		pendingFrames := receiver.pendingFrames
		receiver.mu.Unlock()
		if pendingFrames != 0 {
			t.Fatalf("acknowledgement entered payload receive queue: pending=%d", pendingFrames)
		}
	}

	stream.results <- socksReceiveTestResult{err: io.EOF}
	if err := proxy.Stop(); err != nil {
		t.Fatalf("stop valid acknowledgement test proxy: %v", err)
	}
}

func TestSocksReceiveQueueAdmitsHTTPBurstAndDrainsInOrder(t *testing.T) {
	const (
		httpConcurrentPosts = 64
		httpRelayFrameBytes = 32 * 1024
	)
	if socksReceiveFrameLimit != 128 {
		t.Fatalf("SOCKS receive frame limit = %d, want 128", socksReceiveFrameLimit)
	}
	if socksReceiveByteLimit != 8*1024*1024 {
		t.Fatalf("SOCKS receive byte limit = %d, want 8 MiB", socksReceiveByteLimit)
	}

	connection := &socksGatedWriteConn{
		writes:       make(chan []byte, httpConcurrentPosts),
		writeStarted: make(chan struct{}),
		writeGate:    make(chan struct{}),
		closedSignal: make(chan struct{}),
	}
	receiver := newSocksReceiveQueue(connection, socksReceiveFrameLimit, socksReceiveByteLimit)
	runResult := make(chan error, 1)
	go func() {
		defer receiver.finish()
		runResult <- receiver.run()
	}()
	t.Cleanup(func() {
		_ = connection.Close()
		receiver.stop()
		receiver.wait()
	})

	frames := make([][]byte, httpConcurrentPosts)
	for sequence := range frames {
		frame := make([]byte, httpRelayFrameBytes)
		frame[0] = byte(sequence)
		frame[len(frame)-1] = byte(^sequence)
		frames[sequence] = frame
		if err := receiver.admit(&sliverpb.SocksData{
			Sequence: uint64(sequence),
			Data:     frame,
		}); err != nil {
			t.Fatalf("admit HTTP-style SOCKS burst frame %d: %v", sequence, err)
		}
		if sequence == 0 {
			waitForTestSignal(t, connection.writeStarted, "blocked HTTP-style SOCKS delivery")
		}
	}

	select {
	case unexpected := <-connection.writes:
		t.Fatalf("blocked SOCKS writer delivered data before release: %d bytes", len(unexpected))
	default:
	}
	close(connection.writeGate)
	for sequence, want := range frames {
		select {
		case got := <-connection.writes:
			if !bytes.Equal(got, want) {
				t.Fatalf("HTTP-style SOCKS burst frame %d was not delivered in order", sequence)
			}
		case <-time.After(time.Second):
			t.Fatalf("timed out draining HTTP-style SOCKS burst frame %d", sequence)
		}
	}

	if err := receiver.admit(&sliverpb.SocksData{
		Sequence:  httpConcurrentPosts,
		CloseConn: true,
	}); err != nil {
		t.Fatalf("admit terminal after HTTP-style SOCKS burst: %v", err)
	}
	select {
	case err := <-runResult:
		if err != nil {
			t.Fatalf("drain HTTP-style SOCKS burst: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("SOCKS receive actor did not stop after terminal")
	}
}

func TestSocksSendFlowWaitsForCumulativeAcknowledgement(t *testing.T) {
	flow := newSocksSendFlow()
	flow.enable()
	for sequence := uint64(0); sequence < socksFlowControlWindow; sequence++ {
		if err := flow.wait(); err != nil {
			t.Fatalf("wait for sequence %d: %v", sequence, err)
		}
		if err := flow.recordSent(sequence); err != nil {
			t.Fatalf("record sequence %d: %v", sequence, err)
		}
	}

	waitDone := make(chan error, 1)
	go func() { waitDone <- flow.wait() }()
	select {
	case err := <-waitDone:
		t.Fatalf("full SOCKS flow window returned early: %v", err)
	case <-time.After(25 * time.Millisecond):
	}
	if err := flow.acknowledge(socksFlowAcknowledgementGap); err != nil {
		t.Fatalf("acknowledge first batch: %v", err)
	}
	select {
	case err := <-waitDone:
		if err != nil {
			t.Fatalf("wait after acknowledgement: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("cumulative acknowledgement did not release SOCKS send credit")
	}
	if err := flow.acknowledge(socksFlowControlWindow + 1); err == nil {
		t.Fatal("future SOCKS acknowledgement was accepted")
	}
}

func TestSocksReceiveFlowAcknowledgesConsumedFramesAndFlushesAtTerminal(t *testing.T) {
	connection := &socksLifecycleTestConn{}
	receiver := newSocksReceiveQueue(connection, socksReceiveFrameLimit, socksReceiveByteLimit)
	acknowledgements := make(chan uint64, 2)
	if err := receiver.enableFlowControl(func(ack uint64) error {
		acknowledgements <- ack
		return nil
	}); err != nil {
		t.Fatalf("enable receive flow control: %v", err)
	}
	runResult := make(chan error, 1)
	go func() {
		defer receiver.finish()
		runResult <- receiver.run()
	}()

	const frameCount = 17
	for sequence := uint64(0); sequence < frameCount; sequence++ {
		if err := receiver.admit(&sliverpb.SocksData{Sequence: sequence, Data: []byte{byte(sequence)}}); err != nil {
			t.Fatalf("admit flow-controlled frame %d: %v", sequence, err)
		}
	}
	if err := receiver.admit(&sliverpb.SocksData{Sequence: frameCount, CloseConn: true}); err != nil {
		t.Fatalf("admit flow-controlled terminal: %v", err)
	}

	for index, want := range []uint64{socksFlowAcknowledgementGap, frameCount} {
		select {
		case got := <-acknowledgements:
			if got != want {
				t.Fatalf("acknowledgement %d = %d, want %d", index, got, want)
			}
		case <-time.After(time.Second):
			t.Fatalf("timed out waiting for acknowledgement %d", want)
		}
	}
	select {
	case err := <-runResult:
		if err != nil {
			t.Fatalf("flow-controlled receiver: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("flow-controlled receiver did not stop after terminal")
	}
	if err := receiver.admit(&sliverpb.SocksData{Sequence: frameCount - 1, Data: []byte("stale")}); err == nil {
		t.Fatal("flow-controlled receiver accepted data after terminal")
	}
}

func TestSocksReceiveFlowRejectsDuplicateAndGapSequences(t *testing.T) {
	newReceiver := func(t *testing.T) *socksReceiveQueue {
		t.Helper()
		receiver := newSocksReceiveQueue(&socksLifecycleTestConn{}, socksReceiveFrameLimit, socksReceiveByteLimit)
		if err := receiver.enableFlowControl(func(uint64) error { return nil }); err != nil {
			t.Fatalf("enable receive flow control: %v", err)
		}
		return receiver
	}

	t.Run("duplicate", func(t *testing.T) {
		receiver := newReceiver(t)
		if err := receiver.admit(&sliverpb.SocksData{Sequence: 0, Data: []byte("first")}); err != nil {
			t.Fatalf("admit first frame: %v", err)
		}
		if err := receiver.admit(&sliverpb.SocksData{Sequence: 0, Data: []byte("duplicate")}); err == nil {
			t.Fatal("flow-controlled receiver accepted duplicate sequence")
		}
	})

	t.Run("gap", func(t *testing.T) {
		receiver := newReceiver(t)
		if err := receiver.admit(&sliverpb.SocksData{Sequence: 1, Data: []byte("future")}); err == nil {
			t.Fatal("flow-controlled receiver accepted a sequence gap")
		}
	})
}

func TestSocksReceiveFlowAcknowledgesOnlyAfterCompleteLocalWrite(t *testing.T) {
	connection := &socksPartialWriteConn{
		finalWriteStarted: make(chan struct{}),
		finalWriteRelease: make(chan struct{}),
	}
	receiver := newSocksReceiveQueue(connection, socksReceiveFrameLimit, socksReceiveByteLimit)
	acknowledgements := make(chan uint64, 1)
	if err := receiver.enableFlowControl(func(ack uint64) error {
		acknowledgements <- ack
		return nil
	}); err != nil {
		t.Fatalf("enable receive flow control: %v", err)
	}
	runResult := make(chan error, 1)
	go func() {
		defer receiver.finish()
		runResult <- receiver.run()
	}()

	for sequence := uint64(0); sequence < socksFlowAcknowledgementGap; sequence++ {
		payload := []byte{byte(sequence)}
		if sequence == socksFlowAcknowledgementGap-1 {
			payload = append(payload, 0xff)
		}
		if err := receiver.admit(&sliverpb.SocksData{Sequence: sequence, Data: payload}); err != nil {
			t.Fatalf("admit flow-controlled frame %d: %v", sequence, err)
		}
	}
	waitForTestSignal(t, connection.finalWriteStarted, "final partial local write")
	select {
	case ack := <-acknowledgements:
		t.Fatalf("receiver acknowledged partially written frame at %d", ack)
	default:
	}
	close(connection.finalWriteRelease)
	select {
	case ack := <-acknowledgements:
		if ack != socksFlowAcknowledgementGap {
			t.Fatalf("complete-write acknowledgement = %d, want %d", ack, socksFlowAcknowledgementGap)
		}
	case <-time.After(time.Second):
		t.Fatal("receiver did not acknowledge after the complete local write")
	}
	if err := receiver.admit(&sliverpb.SocksData{Sequence: socksFlowAcknowledgementGap, CloseConn: true}); err != nil {
		t.Fatalf("admit terminal after complete local write: %v", err)
	}
	select {
	case err := <-runResult:
		if err != nil {
			t.Fatalf("flow-controlled receiver: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("flow-controlled receiver did not stop after terminal")
	}
}

func TestSocksReceiveTerminalWakesFullOutboundFlowWindow(t *testing.T) {
	connection := &socksLifecycleTestConn{}
	receiver := newSocksReceiveQueue(connection, socksReceiveFrameLimit, socksReceiveByteLimit)
	flow := newSocksSendFlow()
	flow.enable()
	for sequence := uint64(0); sequence < socksFlowControlWindow; sequence++ {
		if err := flow.recordSent(sequence); err != nil {
			t.Fatalf("record outbound sequence %d: %v", sequence, err)
		}
	}

	waitDone := make(chan error, 1)
	go func() { waitDone <- flow.wait() }()
	select {
	case err := <-waitDone:
		t.Fatalf("full outbound window returned before terminal: %v", err)
	case <-time.After(25 * time.Millisecond):
	}

	proxy := &TcpProxy{}
	proxy.deliveryWG.Add(1)
	go proxy.runReceiveQueue(0x51ee, socksConnectionState{
		conn:     connection,
		receiver: receiver,
		flow:     flow,
	})
	if err := receiver.admit(&sliverpb.SocksData{CloseConn: true}); err != nil {
		t.Fatalf("admit inbound terminal: %v", err)
	}
	proxy.deliveryWG.Wait()

	select {
	case err := <-waitDone:
		if !errors.Is(err, net.ErrClosed) {
			t.Fatalf("full-window waiter after inbound terminal = %v, want %v", err, net.ErrClosed)
		}
	case <-time.After(time.Second):
		t.Fatal("inbound terminal did not wake full-window outbound sender")
	}
	if !connection.closed.Load() {
		t.Fatal("receive actor terminal did not close the exact local connection")
	}
}

func TestSocksReceiveQueueBoundsFramesBytesAndPayloadSize(t *testing.T) {
	connection := &socksLifecycleTestConn{}
	frameLimited := newSocksReceiveQueue(connection, 2, 100)
	if err := frameLimited.admit(&sliverpb.SocksData{Data: []byte("a")}); err != nil {
		t.Fatalf("admit first frame: %v", err)
	}
	if err := frameLimited.admit(&sliverpb.SocksData{Data: []byte("b")}); err != nil {
		t.Fatalf("admit second frame: %v", err)
	}
	if err := frameLimited.admit(&sliverpb.SocksData{Data: []byte("c")}); !errors.Is(err, errSocksReceiveQueueFull) {
		t.Fatalf("frame beyond SOCKS receive frame budget = %v, want %v", err, errSocksReceiveQueueFull)
	}

	byteLimited := newSocksReceiveQueue(connection, 3, 3)
	if err := byteLimited.admit(&sliverpb.SocksData{Data: []byte("abc")}); err != nil {
		t.Fatalf("fill SOCKS receive byte budget: %v", err)
	}
	if err := byteLimited.admit(&sliverpb.SocksData{Data: []byte("d")}); !errors.Is(err, errSocksReceiveQueueFull) {
		t.Fatalf("frame beyond SOCKS receive byte budget = %v, want %v", err, errSocksReceiveQueueFull)
	}
	if err := byteLimited.admit(&sliverpb.SocksData{Data: make([]byte, sliverpb.MaxTunnelFrameBytes+1)}); !errors.Is(err, errSocksReceiveFrameTooLarge) {
		t.Fatalf("oversized SOCKS receive frame = %v, want %v", err, errSocksReceiveFrameTooLarge)
	}
	if err := byteLimited.admit(&sliverpb.SocksData{CloseConn: true, Data: []byte("payload")}); !errors.Is(err, errSocksTerminalPayload) {
		t.Fatalf("SOCKS terminal payload = %v, want %v", err, errSocksTerminalPayload)
	}
}

//nolint:gocyclo // This test joins listener, stream, worker, and unary fallback lifecycles.
func TestTcpProxyStopJoinsConnectionWorkerThroughBoundedCloseFallback(t *testing.T) {
	const (
		tunnelID     = uint64(803)
		closeTimeout = 100 * time.Millisecond
	)
	rpc := &socksLifecycleTestRPC{
		closeCalls: make(chan socksCloseObservation, 1),
		blockClose: true,
	}
	proxy := &TcpProxy{Rpc: rpc, closeTimeout: closeTimeout}
	underlyingStream := &socksLifecycleTestStream{
		sent:   make(chan *sliverpb.SocksData, 2),
		failAt: 2,
	}
	connection := &socksBlockingReadConn{
		readStarted:  make(chan struct{}),
		closedSignal: make(chan struct{}),
	}
	if !proxy.startConnectionWorker(
		connection,
		&serializedSocksStream{stream: underlyingStream},
		&sliverpb.SocksData{
			TunnelID: tunnelID,
			Request:  &commonpb.Request{SessionID: "joined-worker-session"},
		},
	) {
		t.Fatal("failed to start SOCKS connection worker")
	}
	select {
	case bind := <-underlyingStream.sent:
		if bind.CloseConn || bind.Sequence != socksLifecycleBindSequence || bind.TunnelID != tunnelID {
			t.Fatalf("SOCKS bind frame = %+v", bind)
		}
	case <-time.After(time.Second):
		t.Fatal("SOCKS connection worker did not send bind frame")
	}
	waitForTestSignal(t, connection.readStarted, "SOCKS connection worker read")

	stopDone := make(chan error, 1)
	go func() { stopDone <- proxy.Stop() }()
	select {
	case observation := <-rpc.closeCalls:
		if observation.tunnelID != tunnelID || observation.sessionID != "joined-worker-session" {
			t.Fatalf("SOCKS close fallback = %+v", observation)
		}
	case <-time.After(time.Second):
		t.Fatal("SOCKS connection worker did not enter close fallback")
	}
	select {
	case err := <-stopDone:
		t.Fatalf("Stop returned before the connection worker's close fallback ended: %v", err)
	case <-time.After(10 * time.Millisecond):
	}
	select {
	case err := <-stopDone:
		if err != nil {
			t.Fatalf("Stop SOCKS proxy: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("Stop did not join the SOCKS connection worker after bounded close fallback")
	}
	if _, ok := proxy.getConnection(tunnelID); ok {
		t.Fatal("joined SOCKS connection worker remained registered")
	}
	if _, ok := SocksConnPool.Load(tunnelID); ok {
		SocksConnPool.Delete(tunnelID)
		t.Fatal("joined SOCKS connection worker remained in compatibility pool")
	}
}

func TestSocksProxyRemoveReleasesInventoryLockBeforeStopping(t *testing.T) {
	resetSocksLifecycleTestState(t)
	const (
		tunnelID     = uint64(804)
		closeTimeout = 100 * time.Millisecond
	)
	rpc := &socksLifecycleTestRPC{
		closeCalls: make(chan socksCloseObservation, 1),
		blockClose: true,
	}
	proxy := &TcpProxy{Rpc: rpc, closeTimeout: closeTimeout}
	registered := SocksProxies.Add(proxy)
	underlyingStream := &socksLifecycleTestStream{
		sent:   make(chan *sliverpb.SocksData, 2),
		failAt: 2,
	}
	connection := &socksBlockingReadConn{
		readStarted:  make(chan struct{}),
		closedSignal: make(chan struct{}),
	}
	if !proxy.startConnectionWorker(
		connection,
		&serializedSocksStream{stream: underlyingStream},
		&sliverpb.SocksData{
			TunnelID: tunnelID,
			Request:  &commonpb.Request{SessionID: "remove-lock-session"},
		},
	) {
		t.Fatal("failed to start SOCKS connection worker")
	}
	<-underlyingStream.sent // ownership bind
	waitForTestSignal(t, connection.readStarted, "SOCKS connection worker read")

	removeDone := make(chan bool, 1)
	go func() { removeDone <- SocksProxies.Remove(registered.ID) }()
	select {
	case <-rpc.closeCalls:
	case <-time.After(time.Second):
		t.Fatal("Remove did not enter the bounded connection close fallback")
	}

	listDone := make(chan []*SocksProxyMeta, 1)
	go func() { listDone <- SocksProxies.List() }()
	select {
	case proxies := <-listDone:
		for _, candidate := range proxies {
			if candidate.ID == registered.ID {
				t.Fatal("removed proxy remained visible while its workers drained")
			}
		}
	case <-time.After(25 * time.Millisecond):
		t.Fatal("Remove held the inventory lock while its workers drained")
	}

	select {
	case removed := <-removeDone:
		if !removed {
			t.Fatal("Remove returned false for a registered proxy")
		}
	case <-time.After(time.Second):
		t.Fatal("Remove did not finish after its bounded close fallback")
	}
}
