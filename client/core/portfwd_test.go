package core

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/bishopfox/sliver/client/tcpproxy"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/rpcpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"google.golang.org/grpc"
)

type portfwdCloseObservation struct {
	request           *sliverpb.Tunnel
	hasDeadline       bool
	deadlineRemaining time.Duration
}

type portfwdLifecycleTestStream struct {
	grpc.BidiStreamingClient[sliverpb.TunnelData, sliverpb.TunnelData]
	sent            chan *sliverpb.TunnelData
	dataEntered     chan struct{}
	releaseData     chan struct{}
	dataEnteredOnce sync.Once
	releaseDataOnce sync.Once
}

type portfwdBlockingAddr string

func (address portfwdBlockingAddr) Network() string { return "test" }
func (address portfwdBlockingAddr) String() string  { return string(address) }

type portfwdBlockingConn struct {
	readStarted chan struct{}
	releaseRead chan struct{}
	closed      chan struct{}
	readOnce    sync.Once
	releaseOnce sync.Once
	closeOnce   sync.Once
}

func newPortfwdBlockingConn() *portfwdBlockingConn {
	return &portfwdBlockingConn{
		readStarted: make(chan struct{}),
		releaseRead: make(chan struct{}),
		closed:      make(chan struct{}),
	}
}

func (connection *portfwdBlockingConn) Read([]byte) (int, error) {
	connection.readOnce.Do(func() { close(connection.readStarted) })
	<-connection.releaseRead
	return 0, io.EOF
}

func (connection *portfwdBlockingConn) Write(data []byte) (int, error) {
	select {
	case <-connection.closed:
		return 0, net.ErrClosed
	default:
		return len(data), nil
	}
}

func (connection *portfwdBlockingConn) Close() error {
	connection.closeOnce.Do(func() { close(connection.closed) })
	return nil
}

func (connection *portfwdBlockingConn) LocalAddr() net.Addr {
	return portfwdBlockingAddr("local")
}

func (connection *portfwdBlockingConn) RemoteAddr() net.Addr {
	return portfwdBlockingAddr("remote")
}

func (connection *portfwdBlockingConn) SetDeadline(time.Time) error      { return nil }
func (connection *portfwdBlockingConn) SetReadDeadline(time.Time) error  { return nil }
func (connection *portfwdBlockingConn) SetWriteDeadline(time.Time) error { return nil }

func (connection *portfwdBlockingConn) release() {
	connection.releaseOnce.Do(func() { close(connection.releaseRead) })
}

func (stream *portfwdLifecycleTestStream) Send(data *sliverpb.TunnelData) error {
	if len(data.GetData()) > 0 && stream.dataEntered != nil {
		stream.dataEnteredOnce.Do(func() { close(stream.dataEntered) })
		<-stream.releaseData
	}
	stream.sent <- data
	return nil
}

func (stream *portfwdLifecycleTestStream) releaseBlockedData() {
	if stream.releaseData != nil {
		stream.releaseDataOnce.Do(func() { close(stream.releaseData) })
	}
}

type portfwdLifecycleTestRPC struct {
	rpcpb.SliverRPCClient
	createResponse  *sliverpb.Tunnel
	createErr       error
	portfwdResponse *sliverpb.Portfwd
	portfwdErr      error
	portfwdCalls    chan *sliverpb.PortfwdReq
	closeCalls      chan portfwdCloseObservation
	createCalls     chan *sliverpb.Tunnel
	blockCreate     bool
	createEntered   chan struct{}
	createCanceled  chan struct{}
	createOnce      sync.Once
	cancelOnce      sync.Once
}

func newPortfwdLifecycleTestRPC(tunnelID uint64, sessionID string) *portfwdLifecycleTestRPC {
	return &portfwdLifecycleTestRPC{
		createResponse:  &sliverpb.Tunnel{TunnelID: tunnelID, SessionID: sessionID},
		portfwdResponse: &sliverpb.Portfwd{Response: &commonpb.Response{}},
		portfwdCalls:    make(chan *sliverpb.PortfwdReq, 4),
		closeCalls:      make(chan portfwdCloseObservation, 4),
		createCalls:     make(chan *sliverpb.Tunnel, 4),
	}
}

func (rpc *portfwdLifecycleTestRPC) CreateTunnel(ctx context.Context, request *sliverpb.Tunnel, _ ...grpc.CallOption) (*sliverpb.Tunnel, error) {
	rpc.createCalls <- request
	if rpc.blockCreate {
		rpc.createOnce.Do(func() { close(rpc.createEntered) })
		<-ctx.Done()
		rpc.cancelOnce.Do(func() { close(rpc.createCanceled) })
		return nil, ctx.Err()
	}
	return rpc.createResponse, rpc.createErr
}

func (rpc *portfwdLifecycleTestRPC) Portfwd(_ context.Context, request *sliverpb.PortfwdReq, _ ...grpc.CallOption) (*sliverpb.Portfwd, error) {
	rpc.portfwdCalls <- request
	return rpc.portfwdResponse, rpc.portfwdErr
}

func (rpc *portfwdLifecycleTestRPC) CloseTunnel(ctx context.Context, request *sliverpb.Tunnel, _ ...grpc.CallOption) (*commonpb.Empty, error) {
	deadline, hasDeadline := ctx.Deadline()
	observation := portfwdCloseObservation{
		request:     request,
		hasDeadline: hasDeadline,
	}
	if hasDeadline {
		observation.deadlineRemaining = time.Until(deadline)
	}
	rpc.closeCalls <- observation
	return &commonpb.Empty{}, nil
}

func TestChannelProxyWaitsForTunnelBindBeforePortfwd(t *testing.T) {
	resetPortfwdLifecycleTestState(t)
	stream := installPortfwdLifecycleTestStream()
	const (
		tunnelID  = uint64(301)
		sessionID = "portfwd-bind-session"
	)
	rpc := newPortfwdLifecycleTestRPC(tunnelID, sessionID)
	proxy := &ChannelProxy{
		Rpc:        rpc,
		Session:    &clientpb.Session{ID: sessionID},
		RemoteAddr: "127.0.0.1:8080",
	}

	type dialResult struct {
		tunnel *TunnelIO
		err    error
	}
	result := make(chan dialResult, 1)
	go func() {
		tunnel, err := proxy.dialImplant(context.Background())
		result <- dialResult{tunnel: tunnel, err: err}
	}()
	tunnel := waitForPortfwdLifecycleTunnel(t, tunnelID)
	waitForPortfwdLifecycleBindFrame(t, stream, tunnelID)
	select {
	case <-rpc.portfwdCalls:
		t.Fatal("Portfwd RPC ran before the tunnel bind acknowledgement")
	case <-time.After(50 * time.Millisecond):
	}

	tunnel.markBound()
	select {
	case request := <-rpc.portfwdCalls:
		if request.TunnelID != tunnelID {
			t.Fatalf("Portfwd tunnel ID = %d, want %d", request.TunnelID, tunnelID)
		}
	case <-time.After(time.Second):
		t.Fatal("Portfwd RPC did not run after tunnel bind acknowledgement")
	}
	select {
	case outcome := <-result:
		if outcome.err != nil {
			t.Fatalf("dialImplant: %v", outcome.err)
		}
		if outcome.tunnel != tunnel {
			t.Fatal("dialImplant returned a different tunnel generation")
		}
		if err := proxy.closeTunnel(outcome.tunnel); err != nil {
			t.Fatalf("close test tunnel: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("dialImplant did not return after Portfwd response")
	}
}

//nolint:gocyclo // Final payload ordering and both local/remote cleanup form one relay lifecycle.
func TestChannelProxyForwardsFinalPayloadBeforeCloseTunnel(t *testing.T) {
	resetPortfwdLifecycleTestState(t)
	stream := &portfwdLifecycleTestStream{
		sent:        make(chan *sliverpb.TunnelData, 4),
		dataEntered: make(chan struct{}),
		releaseData: make(chan struct{}),
	}
	t.Cleanup(stream.releaseBlockedData)
	GetTunnels().SetStream(stream)
	const (
		tunnelID  = uint64(311)
		sessionID = "portfwd-final-payload-session"
	)
	rpc := newPortfwdLifecycleTestRPC(tunnelID, sessionID)
	proxy := &ChannelProxy{
		Rpc:        rpc,
		Session:    &clientpb.Session{ID: sessionID},
		RemoteAddr: "127.0.0.1:8080",
	}
	serverConnection, clientConnection := net.Pipe()
	t.Cleanup(func() {
		_ = serverConnection.Close()
		_ = clientConnection.Close()
	})
	handlerDone := make(chan struct{})
	go func() {
		proxy.HandleConn(serverConnection)
		close(handlerDone)
	}()

	tunnel := waitForPortfwdLifecycleTunnel(t, tunnelID)
	waitForPortfwdLifecycleBindFrame(t, stream, tunnelID)
	tunnel.markBound()
	select {
	case <-rpc.portfwdCalls:
	case <-time.After(time.Second):
		t.Fatal("final-payload handler did not establish its port forward")
	}

	payload := []byte("last local bytes before EOF")
	localWriteDone := make(chan error, 1)
	go func() {
		_, err := clientConnection.Write(payload)
		if closeErr := clientConnection.Close(); err == nil {
			err = closeErr
		}
		localWriteDone <- err
	}()
	select {
	case <-stream.dataEntered:
	case <-time.After(time.Second):
		t.Fatal("final payload did not enter the tunnel stream Send")
	}
	select {
	case err := <-localWriteDone:
		if err != nil {
			t.Fatalf("write final local payload: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("local writer did not finish")
	}
	select {
	case observation := <-rpc.closeCalls:
		t.Fatalf("CloseTunnel overtook the blocked final stream Send: %+v", observation.request)
	case <-time.After(50 * time.Millisecond):
	}
	select {
	case <-handlerDone:
		t.Fatal("port-forward handler returned before the final stream Send completed")
	default:
	}

	stream.releaseBlockedData()
	select {
	case frame := <-stream.sent:
		if frame.GetTunnelID() != tunnelID || !bytes.Equal(frame.GetData(), payload) {
			t.Fatalf("final stream frame = %+v, want tunnel=%d data=%q", frame, tunnelID, payload)
		}
	case <-time.After(time.Second):
		t.Fatal("final payload was not sent after the stream was released")
	}
	assertBoundedPortfwdClose(t, rpc.closeCalls, tunnelID, sessionID)
	select {
	case <-handlerDone:
	case <-time.After(time.Second):
		t.Fatal("port-forward handler did not finish after sending the final payload")
	}
}

func TestChannelProxyRejectsInvalidRemoteBeforeCreateTunnel(t *testing.T) {
	resetPortfwdLifecycleTestState(t)
	rpc := newPortfwdLifecycleTestRPC(308, "invalid-remote-session")
	proxy := &ChannelProxy{
		Rpc:        rpc,
		Session:    &clientpb.Session{ID: "invalid-remote-session"},
		RemoteAddr: "target.example:not-a-port",
	}

	tunnel, err := proxy.dialImplant(context.Background())
	if err == nil {
		t.Fatal("dialImplant accepted an invalid remote target")
	}
	if tunnel != nil {
		t.Fatal("dialImplant returned a tunnel for an invalid remote target")
	}
	select {
	case request := <-rpc.createCalls:
		t.Fatalf("invalid remote target invoked CreateTunnel with %+v", request)
	default:
	}
	if active := GetTunnels().Get(308); active != nil {
		t.Fatal("invalid remote target registered a client tunnel")
	}
}

//nolint:gocyclo // The compatibility table verifies both legacy and strict result contracts.
func TestChannelProxyHostPortCompatibilityAndValidatedTargets(t *testing.T) {
	tests := []struct {
		name       string
		remoteAddr string
		wantHost   string
		wantPort   uint32
	}{
		{name: "hostname", remoteAddr: "target.example:443", wantHost: "target.example", wantPort: 443},
		{name: "IPv4", remoteAddr: "192.0.2.10:8080", wantHost: "192.0.2.10", wantPort: 8080},
		{name: "IPv6", remoteAddr: "[2001:db8::10]:8443", wantHost: "2001:db8::10", wantPort: 8443},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			proxy := &ChannelProxy{RemoteAddr: test.remoteAddr}
			host, port := proxy.HostPort()
			if host != test.wantHost || port != test.wantPort {
				t.Fatalf("HostPort(%q) = (%q, %d), want (%q, %d)", test.remoteAddr, host, port, test.wantHost, test.wantPort)
			}
			host, port, err := proxy.ValidatedHostPort()
			if err != nil {
				t.Fatalf("ValidatedHostPort(%q): %v", test.remoteAddr, err)
			}
			if host != test.wantHost || port != test.wantPort {
				t.Fatalf("ValidatedHostPort(%q) = (%q, %d), want (%q, %d)", test.remoteAddr, host, port, test.wantHost, test.wantPort)
			}
		})
	}

	proxy := &ChannelProxy{RemoteAddr: "target.example:not-a-port"}
	if host, port := proxy.HostPort(); host != "" || port != 8080 {
		t.Fatalf("legacy HostPort invalid fallback = (%q, %d), want (\"\", 8080)", host, port)
	}
	if host, port, err := proxy.ValidatedHostPort(); err == nil || host != "" || port != 0 {
		t.Fatalf("ValidatedHostPort invalid result = (%q, %d, %v), want empty/zero/error", host, port, err)
	}

	// HostPort historically allowed an empty host and returned its parsed port.
	// Validation-sensitive setup rejects the same value through the new API.
	proxy.RemoteAddr = ":80"
	if host, port := proxy.HostPort(); host != "" || port != 80 {
		t.Fatalf("legacy HostPort empty-host result = (%q, %d), want (\"\", 80)", host, port)
	}
	if host, port, err := proxy.ValidatedHostPort(); err == nil || host != "" || port != 0 {
		t.Fatalf("ValidatedHostPort empty-host result = (%q, %d, %v), want empty/zero/error", host, port, err)
	}

	for _, remoteAddr := range []string{
		"bad host:80",
		"bad\thost:80",
		"bad\nhost:80",
		"[fe80::1%bad zone]:80",
	} {
		proxy.RemoteAddr = remoteAddr
		if host, port, err := proxy.ValidatedHostPort(); err == nil || host != "" || port != 0 {
			t.Errorf("ValidatedHostPort whitespace/control result for %q = (%q, %d, %v), want empty/zero/error", remoteAddr, host, port, err)
		}
	}
}

func TestChannelProxySetupFailureClosesClientAndServerTunnel(t *testing.T) {
	resetPortfwdLifecycleTestState(t)
	stream := installPortfwdLifecycleTestStream()
	const (
		tunnelID  = uint64(302)
		sessionID = "portfwd-failure-session"
	)
	portfwdErr := errors.New("test Portfwd RPC failure")
	rpc := newPortfwdLifecycleTestRPC(tunnelID, sessionID)
	rpc.portfwdErr = portfwdErr
	proxy := &ChannelProxy{
		Rpc:        rpc,
		Session:    &clientpb.Session{ID: sessionID},
		RemoteAddr: "127.0.0.1:8080",
	}

	result := make(chan error, 1)
	go func() {
		_, err := proxy.dialImplant(context.Background())
		result <- err
	}()
	tunnel := waitForPortfwdLifecycleTunnel(t, tunnelID)
	waitForPortfwdLifecycleBindFrame(t, stream, tunnelID)
	tunnel.markBound()
	select {
	case err := <-result:
		if !errors.Is(err, portfwdErr) {
			t.Fatalf("dialImplant error = %v, want %v", err, portfwdErr)
		}
	case <-time.After(time.Second):
		t.Fatal("dialImplant setup failure did not return")
	}
	if active := GetTunnels().Get(tunnelID); active != nil {
		t.Fatal("failed setup retained the client tunnel")
	}
	assertBoundedPortfwdClose(t, rpc.closeCalls, tunnelID, sessionID)
}

func TestChannelProxyImplantSetupErrorClosesClientAndServerTunnel(t *testing.T) {
	resetPortfwdLifecycleTestState(t)
	stream := installPortfwdLifecycleTestStream()
	const (
		tunnelID  = uint64(305)
		sessionID = "portfwd-implant-failure-session"
	)
	rpc := newPortfwdLifecycleTestRPC(tunnelID, sessionID)
	rpc.portfwdResponse.Response.Err = "target dial refused"
	proxy := &ChannelProxy{
		Rpc:        rpc,
		Session:    &clientpb.Session{ID: sessionID},
		RemoteAddr: "127.0.0.1:8080",
	}

	result := make(chan error, 1)
	go func() {
		_, err := proxy.dialImplant(context.Background())
		result <- err
	}()
	tunnel := waitForPortfwdLifecycleTunnel(t, tunnelID)
	waitForPortfwdLifecycleBindFrame(t, stream, tunnelID)
	tunnel.markBound()
	select {
	case err := <-result:
		if err == nil || err.Error() != "target dial refused" {
			t.Fatalf("dialImplant error = %v, want target dial refused", err)
		}
	case <-time.After(time.Second):
		t.Fatal("dialImplant implant failure did not return")
	}
	if active := GetTunnels().Get(tunnelID); active != nil {
		t.Fatal("implant setup failure retained the client tunnel")
	}
	assertBoundedPortfwdClose(t, rpc.closeCalls, tunnelID, sessionID)
}

func TestChannelProxyClosesAcceptedConnectionWhenCreateTunnelFails(t *testing.T) {
	resetPortfwdLifecycleTestState(t)
	rpc := newPortfwdLifecycleTestRPC(303, "portfwd-create-failure")
	rpc.createErr = errors.New("test CreateTunnel failure")
	proxy := &ChannelProxy{
		Rpc:        rpc,
		Session:    &clientpb.Session{ID: "portfwd-create-failure"},
		RemoteAddr: "127.0.0.1:8080",
	}
	serverConnection, clientConnection := net.Pipe()
	defer func() { _ = clientConnection.Close() }()
	handlerDone := make(chan struct{})
	go func() {
		proxy.HandleConn(serverConnection)
		close(handlerDone)
	}()

	if err := clientConnection.SetReadDeadline(time.Now().Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	buffer := make([]byte, 1)
	count, err := clientConnection.Read(buffer)
	if count != 0 || !errors.Is(err, io.EOF) {
		t.Fatalf("accepted connection read = (%d, %v), want (0, EOF)", count, err)
	}
	select {
	case <-handlerDone:
	case <-time.After(time.Second):
		t.Fatal("HandleConn did not return after CreateTunnel failure")
	}
}

func TestChannelProxyStopCancelsCreateTunnelWithoutDialTimeout(t *testing.T) {
	resetPortfwdLifecycleTestState(t)
	rpc := newPortfwdLifecycleTestRPC(306, "portfwd-cancel-create")
	rpc.blockCreate = true
	rpc.createEntered = make(chan struct{})
	rpc.createCanceled = make(chan struct{})
	proxy := &ChannelProxy{
		Rpc:         rpc,
		Session:     &clientpb.Session{ID: "portfwd-cancel-create"},
		RemoteAddr:  "127.0.0.1:8080",
		DialTimeout: -1,
	}
	serverConnection, clientConnection := net.Pipe()
	defer func() { _ = clientConnection.Close() }()
	handlerDone := make(chan struct{})
	go func() {
		proxy.HandleConn(serverConnection)
		close(handlerDone)
	}()

	select {
	case <-rpc.createEntered:
	case <-time.After(time.Second):
		t.Fatal("CreateTunnel did not start")
	}
	proxy.Stop()
	select {
	case <-rpc.createCanceled:
	case <-time.After(time.Second):
		t.Fatal("Stop did not cancel the in-flight CreateTunnel RPC")
	}
	select {
	case <-handlerDone:
	case <-time.After(time.Second):
		t.Fatal("HandleConn did not return after Stop canceled CreateTunnel")
	}
	if err := clientConnection.SetReadDeadline(time.Now().Add(time.Second)); err == nil {
		buffer := make([]byte, 1)
		count, readErr := clientConnection.Read(buffer)
		if count != 0 || (!errors.Is(readErr, io.EOF) && !errors.Is(readErr, io.ErrClosedPipe)) {
			t.Fatalf("connection read after Stop = (%d, %v), want (0, EOF or closed pipe)", count, readErr)
		}
	} else if !errors.Is(err, io.ErrClosedPipe) {
		t.Fatalf("set read deadline after Stop: %v", err)
	}
}

func TestChannelProxyStopWaitsForBlockedHandler(t *testing.T) {
	resetPortfwdLifecycleTestState(t)
	stream := installPortfwdLifecycleTestStream()
	const (
		tunnelID  = uint64(309)
		sessionID = "portfwd-stop-join-session"
	)
	rpc := newPortfwdLifecycleTestRPC(tunnelID, sessionID)
	proxy := &ChannelProxy{
		Rpc:        rpc,
		Session:    &clientpb.Session{ID: sessionID},
		RemoteAddr: "127.0.0.1:8080",
	}
	connection := newPortfwdBlockingConn()
	t.Cleanup(connection.release)
	handlerDone := make(chan struct{})
	go func() {
		proxy.HandleConn(connection)
		close(handlerDone)
	}()

	tunnel := waitForPortfwdLifecycleTunnel(t, tunnelID)
	waitForPortfwdLifecycleBindFrame(t, stream, tunnelID)
	tunnel.markBound()
	select {
	case <-rpc.portfwdCalls:
	case <-time.After(time.Second):
		t.Fatal("blocked handler did not establish its port forward")
	}
	select {
	case <-connection.readStarted:
	case <-time.After(time.Second):
		t.Fatal("handler copy loop did not enter the blocking connection read")
	}

	stopDone := make(chan struct{})
	go func() {
		proxy.Stop()
		close(stopDone)
	}()
	select {
	case <-connection.closed:
	case <-time.After(time.Second):
		t.Fatal("Stop did not close the tracked connection")
	}
	select {
	case <-stopDone:
		t.Fatal("Stop returned before the blocked handler copy loop joined")
	case <-time.After(50 * time.Millisecond):
	}

	connection.release()
	select {
	case <-stopDone:
	case <-time.After(time.Second):
		t.Fatal("Stop did not return after the blocked handler was released")
	}
	select {
	case <-handlerDone:
	case <-time.After(time.Second):
		t.Fatal("HandleConn did not return before Stop completed")
	}
	if active := GetTunnels().Get(tunnelID); active != nil {
		t.Fatal("joined handler retained its client tunnel")
	}
	assertBoundedPortfwdClose(t, rpc.closeCalls, tunnelID, sessionID)
}

func TestChannelProxyReceiveOverflowClosesLocalAndRemoteTunnel(t *testing.T) {
	resetPortfwdLifecycleTestState(t)
	stream := installPortfwdLifecycleTestStream()
	const (
		tunnelID  = uint64(310)
		sessionID = "portfwd-overflow-session"
	)
	rpc := newPortfwdLifecycleTestRPC(tunnelID, sessionID)
	proxy := &ChannelProxy{
		Rpc:        rpc,
		Session:    &clientpb.Session{ID: sessionID},
		RemoteAddr: "127.0.0.1:8080",
	}
	serverConnection, clientConnection := net.Pipe()
	t.Cleanup(func() { _ = clientConnection.Close() })
	handlerDone := make(chan struct{})
	go func() {
		proxy.HandleConn(serverConnection)
		close(handlerDone)
	}()

	tunnel := waitForPortfwdLifecycleTunnel(t, tunnelID)
	waitForPortfwdLifecycleBindFrame(t, stream, tunnelID)
	tunnel.markBound()
	select {
	case <-rpc.portfwdCalls:
	case <-time.After(time.Second):
		t.Fatal("overflow test did not establish its port forward")
	}

	var admissionErr error
	for index := 0; index < tunnelRecvBufferSize+2; index++ {
		admissionErr = tunnel.RecvData([]byte{byte(index)})
		if admissionErr != nil {
			break
		}
	}
	if !errors.Is(admissionErr, errTunnelReceiveQueueFull) {
		t.Fatalf("receive overflow error = %v, want %v", admissionErr, errTunnelReceiveQueueFull)
	}
	tunnel.failReceive()
	GetTunnels().CloseIf(tunnel)

	if err := clientConnection.SetReadDeadline(time.Now().Add(time.Second)); err == nil {
		for {
			if _, err := clientConnection.Read(make([]byte, 1)); err != nil {
				break
			}
		}
	} else if !errors.Is(err, io.ErrClosedPipe) {
		t.Fatalf("set read deadline after receive overflow: %v", err)
	}
	select {
	case <-handlerDone:
	case <-time.After(time.Second):
		t.Fatal("receive overflow did not stop the port-forward handler")
	}
	assertBoundedPortfwdClose(t, rpc.closeCalls, tunnelID, sessionID)
}

func TestPortfwdsRemoveClosesEstablishedConnectionAndTunnel(t *testing.T) {
	resetPortfwdLifecycleTestState(t)
	stream := installPortfwdLifecycleTestStream()
	const (
		tunnelID  = uint64(304)
		sessionID = "portfwd-remove-session"
	)
	rpc := newPortfwdLifecycleTestRPC(tunnelID, sessionID)
	channelProxy := &ChannelProxy{
		Rpc:        rpc,
		Session:    &clientpb.Session{ID: sessionID},
		RemoteAddr: "127.0.0.1:8080",
	}
	registered := Portfwds.Add(&tcpproxy.Proxy{}, channelProxy)
	serverConnection, clientConnection := net.Pipe()
	defer func() { _ = clientConnection.Close() }()
	handlerDone := make(chan struct{})
	go func() {
		channelProxy.HandleConn(serverConnection)
		close(handlerDone)
	}()
	tunnel := waitForPortfwdLifecycleTunnel(t, tunnelID)
	waitForPortfwdLifecycleBindFrame(t, stream, tunnelID)
	tunnel.markBound()
	select {
	case <-rpc.portfwdCalls:
	case <-time.After(time.Second):
		t.Fatal("established connection did not invoke Portfwd")
	}

	if !Portfwds.Remove(registered.ID) {
		t.Fatal("Portfwds.Remove did not remove the registered proxy")
	}
	if err := clientConnection.SetReadDeadline(time.Now().Add(time.Second)); err != nil {
		// net.Pipe may report the peer teardown while setting the deadline,
		// before a subsequent Read can observe EOF.
		if !errors.Is(err, io.ErrClosedPipe) {
			t.Fatalf("set read deadline after remove: %v", err)
		}
	} else {
		buffer := make([]byte, 1)
		count, err := clientConnection.Read(buffer)
		if count != 0 || (!errors.Is(err, io.EOF) && !errors.Is(err, io.ErrClosedPipe)) {
			t.Fatalf("established connection read after remove = (%d, %v), want (0, EOF or closed pipe)", count, err)
		}
	}
	select {
	case <-handlerDone:
	case <-time.After(time.Second):
		t.Fatal("established handler did not stop after Portfwds.Remove")
	}
	if active := GetTunnels().Get(tunnelID); active != nil {
		t.Fatal("Portfwds.Remove retained the client tunnel")
	}
	assertBoundedPortfwdClose(t, rpc.closeCalls, tunnelID, sessionID)
}

func TestChannelProxyStaleCloseDoesNotCloseReplacementTunnel(t *testing.T) {
	resetPortfwdLifecycleTestState(t)
	stream := installPortfwdLifecycleTestStream()
	const tunnelID = uint64(307)
	rpc := newPortfwdLifecycleTestRPC(tunnelID, "old-session")
	proxy := &ChannelProxy{Rpc: rpc}

	old := GetTunnels().Start(tunnelID, "old-session")
	waitForPortfwdLifecycleBindFrame(t, stream, tunnelID)
	replacement := GetTunnels().Start(tunnelID, "new-session")
	waitForPortfwdLifecycleBindFrame(t, stream, tunnelID)

	if err := proxy.closeTunnel(old); err != nil {
		t.Fatalf("close stale port forward tunnel: %v", err)
	}
	if current := GetTunnels().Get(tunnelID); current != replacement {
		t.Fatal("stale port forward cleanup removed the replacement tunnel")
	}
	select {
	case <-replacement.Done():
		t.Fatal("stale port forward cleanup closed the replacement tunnel")
	default:
	}
	assertBoundedPortfwdClose(t, rpc.closeCalls, tunnelID, "old-session")
}

func resetPortfwdLifecycleTestState(t *testing.T) {
	t.Helper()
	for _, metadata := range Portfwds.List() {
		Portfwds.Remove(metadata.ID)
	}
	GetTunnels().Reset()
	t.Cleanup(func() {
		for _, metadata := range Portfwds.List() {
			Portfwds.Remove(metadata.ID)
		}
		GetTunnels().Reset()
	})
}

func waitForPortfwdLifecycleTunnel(t *testing.T, tunnelID uint64) *TunnelIO {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if tunnel := GetTunnels().Get(tunnelID); tunnel != nil {
			return tunnel
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("timed out waiting for client tunnel %d", tunnelID)
	return nil
}

func installPortfwdLifecycleTestStream() *portfwdLifecycleTestStream {
	stream := &portfwdLifecycleTestStream{sent: make(chan *sliverpb.TunnelData, 4)}
	GetTunnels().SetStream(stream)
	return stream
}

func waitForPortfwdLifecycleBindFrame(t *testing.T, stream *portfwdLifecycleTestStream, tunnelID uint64) {
	t.Helper()
	select {
	case frame := <-stream.sent:
		if frame.GetTunnelID() != tunnelID || len(frame.GetData()) != 0 {
			t.Fatalf("bind frame = %+v, want empty data for tunnel %d", frame, tunnelID)
		}
	case <-time.After(time.Second):
		t.Fatalf("timed out waiting for tunnel %d bind frame", tunnelID)
	}
}

func assertBoundedPortfwdClose(t *testing.T, observations <-chan portfwdCloseObservation, tunnelID uint64, sessionID string) {
	t.Helper()
	select {
	case observation := <-observations:
		if observation.request.GetTunnelID() != tunnelID || observation.request.GetSessionID() != sessionID {
			t.Fatalf("CloseTunnel request = %+v, want tunnel=%d session=%q", observation.request, tunnelID, sessionID)
		}
		if !observation.hasDeadline {
			t.Fatal("CloseTunnel RPC context has no deadline")
		}
		if observation.deadlineRemaining <= 0 || observation.deadlineRemaining > portfwdCloseTimeout {
			t.Fatalf("CloseTunnel deadline remaining = %v, want within (0, %v]", observation.deadlineRemaining, portfwdCloseTimeout)
		}
	case <-time.After(time.Second):
		t.Fatal("CloseTunnel RPC was not called")
	}
}
