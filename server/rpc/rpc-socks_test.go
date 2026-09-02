package rpc

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/rpcpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/bishopfox/sliver/server/core"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

func isCanceledSocksProxyError(err error) bool {
	return errors.Is(err, context.Canceled) || status.Code(err) == codes.Canceled
}

type controllableSocksProxyServer struct {
	rpcpb.SliverRPC_SocksProxyServer
	ctx  context.Context
	recv chan *sliverpb.SocksData
	sent chan *sliverpb.SocksData
}

func (stream *controllableSocksProxyServer) Context() context.Context {
	return stream.ctx
}

func (stream *controllableSocksProxyServer) Recv() (*sliverpb.SocksData, error) {
	select {
	case <-stream.ctx.Done():
		return nil, stream.ctx.Err()
	case frame := <-stream.recv:
		return frame, nil
	}
}

func (stream *controllableSocksProxyServer) Send(frame *sliverpb.SocksData) error {
	frameCopy := proto.Clone(frame).(*sliverpb.SocksData)
	select {
	case <-stream.ctx.Done():
		return stream.ctx.Err()
	case stream.sent <- frameCopy:
		return nil
	}
}

type blockingSocksProxyServer struct {
	rpcpb.SliverRPC_SocksProxyServer
	ctx         context.Context
	recvStarted chan struct{}
	releaseRecv chan struct{}
	startOnce   sync.Once
}

func (stream *blockingSocksProxyServer) Context() context.Context {
	return stream.ctx
}

func (stream *blockingSocksProxyServer) Recv() (*sliverpb.SocksData, error) {
	stream.startOnce.Do(func() { close(stream.recvStarted) })
	<-stream.releaseRecv
	return nil, context.Canceled
}

func TestSocksProxyCancellationDoesNotWaitForBlockedReceive(t *testing.T) {
	streamContext, cancelStream := context.WithCancel(context.Background())
	stream := &blockingSocksProxyServer{
		ctx:         streamContext,
		recvStarted: make(chan struct{}),
		releaseRecv: make(chan struct{}),
	}
	result := make(chan error, 1)
	go func() {
		result <- (&Server{}).SocksProxy(stream)
	}()

	select {
	case <-stream.recvStarted:
	case <-time.After(time.Second):
		t.Fatal("SocksProxy did not start receiving")
	}
	cancelStream()
	select {
	case err := <-result:
		if !isCanceledSocksProxyError(err) {
			t.Fatalf("SocksProxy cancellation error = %v, want %v", err, context.Canceled)
		}
	case <-time.After(time.Second):
		t.Fatal("SocksProxy waited for a blocked Recv after cancellation")
	}
	close(stream.releaseRecv)
}

func TestSocksProxyLifecycleMarkerOwnsTunnelWithoutForwarding(t *testing.T) {
	connection := core.NewImplantConnection("idle-bind", "test")
	connection.Send = make(chan *sliverpb.Envelope, 8)
	session := core.NewSession(connection)
	core.Sessions.Add(session)
	t.Cleanup(func() {
		core.Sessions.Remove(session.ID)
		connection.Close()
	})

	created, err := (&Server{}).CreateSocks(context.Background(), &sliverpb.Socks{SessionID: session.ID})
	if err != nil {
		t.Fatalf("create SOCKS tunnel: %v", err)
	}
	tunnel := core.SocksTunnels.Get(created.TunnelID)
	if tunnel == nil {
		t.Fatal("created SOCKS tunnel was not registered")
	}
	if tunnel.ImplantConnection() != connection {
		t.Fatal("created SOCKS tunnel did not retain its exact implant connection")
	}

	streamContext, cancelStream := context.WithCancel(context.Background())
	stream := &controllableSocksProxyServer{
		ctx:  streamContext,
		recv: make(chan *sliverpb.SocksData, 1),
		sent: make(chan *sliverpb.SocksData, 2),
	}
	result := make(chan error, 1)
	go func() { result <- (&Server{}).SocksProxy(stream) }()
	stream.recv <- &sliverpb.SocksData{
		TunnelID: created.TunnelID,
		Sequence: socksLifecycleBindSequence,
		Request:  &commonpb.Request{SessionID: session.ID},
	}

	waitForRPCSocksCondition(t, func() bool { return tunnel.Client() != nil }, "lifecycle marker did not claim the SOCKS tunnel")
	if lifecycle := tunnel.ClientLifecycle(); !lifecycle.SendsTerminal || lifecycle.ReceivedPayload {
		t.Fatalf("lifecycle marker state = %+v", lifecycle)
	}
	select {
	case frame := <-tunnel.ToImplant():
		tunnel.CompleteToImplant(frame)
		t.Fatal("lifecycle marker was queued as implant payload")
	default:
	}

	cancelStream()
	select {
	case err := <-result:
		if !isCanceledSocksProxyError(err) {
			t.Fatalf("SocksProxy cancellation error = %v, want %v", err, context.Canceled)
		}
	case <-time.After(time.Second):
		t.Fatal("SocksProxy did not exit after stream cancellation")
	}
	if got := core.SocksTunnels.Get(tunnel.ID); got != nil {
		t.Fatalf("idle bound tunnel remained after proxy stream failure: %p", got)
	}
}

func TestCreateSocksNegotiatesExactFlowControlCapability(t *testing.T) {
	tests := []struct {
		name                string
		requestCapabilities uint64
		sessionCapabilities uint64
		want                uint64
	}{
		{
			name:                "both endpoints support flow control",
			requestCapabilities: sliverpb.CapabilitySocksFlowControlV1 | uint64(1<<20),
			sessionCapabilities: sliverpb.CapabilitySocksFlowControlV1 | sliverpb.CapabilityTunnelTerminalV1,
			want:                sliverpb.CapabilitySocksFlowControlV1,
		},
		{
			name:                "implant lacks flow control",
			requestCapabilities: sliverpb.CapabilitySocksFlowControlV1,
			sessionCapabilities: sliverpb.CapabilityTunnelTerminalV1,
			want:                0,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			connection := core.NewImplantConnection("capability-negotiation", "test")
			connection.Send = make(chan *sliverpb.Envelope, 2)
			session := core.NewSession(connection)
			session.Capabilities = test.sessionCapabilities
			core.Sessions.Add(session)
			t.Cleanup(func() {
				core.Sessions.RemoveIf(session)
				connection.Close()
			})

			created, err := (&Server{}).CreateSocks(context.Background(), &sliverpb.Socks{
				SessionID:    session.ID,
				Capabilities: test.requestCapabilities,
			})
			if err != nil {
				t.Fatalf("CreateSocks: %v", err)
			}
			if created.SessionID != session.ID || created.Capabilities != test.want {
				t.Fatalf("CreateSocks response = %+v, want session %s capabilities %d", created, session.ID, test.want)
			}
			tunnel := core.SocksTunnels.Get(created.TunnelID)
			if tunnel == nil {
				t.Fatal("created SOCKS tunnel was not registered")
			}
			if tunnel.ImplantConnection() != connection || tunnel.Capabilities() != test.want {
				t.Fatalf("created tunnel = %p owner:%p capabilities:%d", tunnel, tunnel.ImplantConnection(), tunnel.Capabilities())
			}
		})
	}
}

//nolint:gocyclo // The assertions cover negotiation and both ACK relay directions.
func TestSocksProxyRelaysNegotiatedFlowControlWithoutQueueingAcks(t *testing.T) {
	connection := core.NewImplantConnection("flow-control-relay", "test")
	connection.Send = make(chan *sliverpb.Envelope, 8)
	session := core.NewSession(connection)
	session.Capabilities = sliverpb.CapabilitySocksFlowControlV1
	core.Sessions.Add(session)
	t.Cleanup(func() {
		core.Sessions.RemoveIf(session)
		connection.Close()
	})
	server := &Server{}
	created, err := server.CreateSocks(context.Background(), &sliverpb.Socks{
		SessionID:    session.ID,
		Capabilities: sliverpb.CapabilitySocksFlowControlV1,
	})
	if err != nil {
		t.Fatalf("create flow-controlled SOCKS tunnel: %v", err)
	}
	tunnel := core.SocksTunnels.Get(created.TunnelID)
	if tunnel == nil {
		t.Fatal("flow-controlled SOCKS tunnel was not registered")
	}

	streamContext, cancelStream := context.WithCancel(context.Background())
	defer cancelStream()
	stream := &controllableSocksProxyServer{
		ctx:  streamContext,
		recv: make(chan *sliverpb.SocksData, 4),
		sent: make(chan *sliverpb.SocksData, 8),
	}
	result := make(chan error, 1)
	go func() { result <- server.SocksProxy(stream) }()
	request := &commonpb.Request{SessionID: session.ID}
	stream.recv <- &sliverpb.SocksData{
		TunnelID:     tunnel.ID,
		Sequence:     socksLifecycleBindSequence,
		Capabilities: sliverpb.CapabilitySocksFlowControlV1,
		Username:     "flow-user",
		Password:     "flow-password",
		Request:      request,
	}
	waitForRPCSocksCondition(t, func() bool { return tunnel.Client() != nil }, "negotiated ownership bind did not claim tunnel")
	stream.recv <- &sliverpb.SocksData{
		TunnelID: tunnel.ID,
		Sequence: 0,
		Data:     []byte("operator-to-implant"),
		Request:  request,
	}

	var toImplant sliverpb.SocksData
	select {
	case envelope := <-connection.Send:
		if err := proto.Unmarshal(envelope.Data, &toImplant); err != nil {
			t.Fatalf("decode operator data: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("operator data did not reach implant connection")
	}
	if toImplant.Sequence != 0 || string(toImplant.Data) != "operator-to-implant" ||
		toImplant.Capabilities != sliverpb.CapabilitySocksFlowControlV1 ||
		toImplant.Username != "flow-user" || toImplant.Password != "flow-password" || toImplant.Ack != 0 {
		t.Fatalf("first implant data frame = %+v", &toImplant)
	}
	waitForRPCSocksCondition(t, func() bool {
		return atomic.LoadUint64(&tunnel.ToImplantSequence) == 1
	}, "to-implant sent high-water did not advance")

	if err := tunnel.ProcessDataFromImplant(&sliverpb.SocksData{
		TunnelID: tunnel.ID,
		Sequence: 0,
		Data:     []byte("implant-to-operator"),
	}); err != nil {
		t.Fatalf("admit implant data: %v", err)
	}
	select {
	case frame := <-stream.sent:
		if frame.Sequence != 0 || string(frame.Data) != "implant-to-operator" || frame.Ack != 0 || frame.Capabilities != 0 {
			t.Fatalf("client data frame = %+v", frame)
		}
	case <-time.After(time.Second):
		t.Fatal("implant data did not reach SOCKS client")
	}
	waitForRPCSocksCondition(t, func() bool {
		return atomic.LoadUint64(&tunnel.FromImplantSequence) == 1
	}, "from-implant sent high-water did not advance")

	stream.recv <- &sliverpb.SocksData{TunnelID: tunnel.ID, Ack: 1, Request: request}
	select {
	case envelope := <-connection.Send:
		ack := &sliverpb.SocksData{}
		if err := proto.Unmarshal(envelope.Data, ack); err != nil {
			t.Fatalf("decode client acknowledgement: %v", err)
		}
		if ack.TunnelID != tunnel.ID || ack.Ack != 1 || len(ack.Data) != 0 || ack.CloseConn || ack.Sequence != 0 || ack.Capabilities != 0 || ack.Request != nil {
			t.Fatalf("client acknowledgement relayed to implant = %+v", ack)
		}
	case <-time.After(time.Second):
		t.Fatal("client acknowledgement did not reach implant")
	}
	if err := tunnel.RelayImplantAcknowledgement(connection, 1); err != nil {
		t.Fatalf("queue implant acknowledgement: %v", err)
	}
	select {
	case ack := <-stream.sent:
		if ack.TunnelID != tunnel.ID || ack.Ack != 1 || len(ack.Data) != 0 || ack.CloseConn || ack.Sequence != 0 || ack.Capabilities != 0 || ack.Request != nil {
			t.Fatalf("implant acknowledgement relayed to client = %+v", ack)
		}
	case <-time.After(time.Second):
		t.Fatal("implant acknowledgement did not reach SOCKS client")
	}
	select {
	case frame := <-tunnel.ToImplant():
		tunnel.CompleteToImplant(frame)
		t.Fatalf("acknowledgement entered payload queue: %+v", frame)
	default:
	}

	cancelStream()
	select {
	case err := <-result:
		if !isCanceledSocksProxyError(err) {
			t.Fatalf("flow-control SocksProxy cancellation error = %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("flow-control SocksProxy did not stop")
	}
}

func TestSocksProxyInjectsCapabilityIntoOrderedSequenceZeroTerminal(t *testing.T) {
	connection := core.NewImplantConnection("flow-control-empty-terminal", "test")
	connection.Send = make(chan *sliverpb.Envelope, 4)
	session := core.NewSession(connection)
	session.Capabilities = sliverpb.CapabilitySocksFlowControlV1
	core.Sessions.Add(session)
	t.Cleanup(func() {
		core.Sessions.RemoveIf(session)
		connection.Close()
	})
	server := &Server{}
	created, err := server.CreateSocks(context.Background(), &sliverpb.Socks{SessionID: session.ID, Capabilities: sliverpb.CapabilitySocksFlowControlV1})
	if err != nil {
		t.Fatalf("create empty flow-controlled tunnel: %v", err)
	}
	streamContext, cancelStream := context.WithCancel(context.Background())
	defer cancelStream()
	stream := &controllableSocksProxyServer{ctx: streamContext, recv: make(chan *sliverpb.SocksData, 2), sent: make(chan *sliverpb.SocksData, 2)}
	result := make(chan error, 1)
	go func() { result <- server.SocksProxy(stream) }()
	request := &commonpb.Request{SessionID: session.ID}
	stream.recv <- &sliverpb.SocksData{TunnelID: created.TunnelID, Sequence: socksLifecycleBindSequence, Capabilities: sliverpb.CapabilitySocksFlowControlV1, Request: request}
	stream.recv <- &sliverpb.SocksData{TunnelID: created.TunnelID, Sequence: 0, CloseConn: true, Request: request}

	select {
	case envelope := <-connection.Send:
		terminal := &sliverpb.SocksData{}
		if err := proto.Unmarshal(envelope.Data, terminal); err != nil {
			t.Fatalf("decode empty terminal: %v", err)
		}
		if terminal.TunnelID != created.TunnelID || !terminal.CloseConn || terminal.Sequence != 0 || terminal.Capabilities != sliverpb.CapabilitySocksFlowControlV1 || terminal.Ack != 0 {
			t.Fatalf("empty negotiated implant terminal = %+v", terminal)
		}
	case <-time.After(time.Second):
		t.Fatal("empty negotiated terminal did not reach implant")
	}
	select {
	case duplicate := <-connection.Send:
		t.Fatalf("ordered terminal produced duplicate implant frame: %+v", duplicate)
	default:
	}

	cancelStream()
	select {
	case err := <-result:
		if !isCanceledSocksProxyError(err) {
			t.Fatalf("empty-terminal SocksProxy cancellation error = %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("empty-terminal SocksProxy did not stop")
	}
}

func TestSocksProxyLateBindAfterSessionRemovalGetsExactTerminal(t *testing.T) {
	connection := core.NewImplantConnection("late-bind", "test")
	connection.Send = make(chan *sliverpb.Envelope, 2)
	session := core.NewSession(connection)
	core.Sessions.Add(session)
	server := &Server{}
	created, err := server.CreateSocks(context.Background(), &sliverpb.Socks{SessionID: session.ID})
	if err != nil {
		t.Fatalf("create SOCKS tunnel: %v", err)
	}
	core.Sessions.Remove(session.ID)
	t.Cleanup(connection.Close)
	if got := core.SocksTunnels.Get(created.TunnelID); got != nil {
		t.Fatalf("session removal retained unbound tunnel: %p", got)
	}

	streamContext, cancelStream := context.WithCancel(context.Background())
	stream := &controllableSocksProxyServer{
		ctx:  streamContext,
		recv: make(chan *sliverpb.SocksData, 1),
		sent: make(chan *sliverpb.SocksData, 1),
	}
	result := make(chan error, 1)
	go func() { result <- server.SocksProxy(stream) }()
	stream.recv <- &sliverpb.SocksData{
		TunnelID: created.TunnelID,
		Sequence: socksLifecycleBindSequence,
		Request:  &commonpb.Request{SessionID: session.ID},
	}
	select {
	case terminal := <-stream.sent:
		if terminal.TunnelID != created.TunnelID || !terminal.CloseConn || len(terminal.Data) != 0 {
			t.Fatalf("late-bind terminal = %+v", terminal)
		}
	case <-time.After(time.Second):
		t.Fatal("late bind did not receive exact close terminal")
	}
	select {
	case duplicate := <-stream.sent:
		t.Fatalf("late bind received duplicate terminal: %+v", duplicate)
	default:
	}

	cancelStream()
	select {
	case err := <-result:
		if !isCanceledSocksProxyError(err) {
			t.Fatalf("SocksProxy cancellation error = %v, want context.Canceled", err)
		}
	case <-time.After(time.Second):
		t.Fatal("SocksProxy did not exit after late-bind test")
	}
}

func TestSocksProxyRejectsInvalidOperatorFraming(t *testing.T) {
	tests := []struct {
		name   string
		frames []*sliverpb.SocksData
	}{
		{
			name: "oversized payload",
			frames: []*sliverpb.SocksData{{
				Sequence: 0,
				Data:     make([]byte, core.MaxSocksFrameBytes+1),
			}},
		},
		{
			name: "far future sequence",
			frames: []*sliverpb.SocksData{{
				Sequence: 128,
			}},
		},
		{
			name: "conflicting duplicate",
			frames: []*sliverpb.SocksData{
				{Sequence: 1, Data: []byte("first")},
				{Sequence: 1, Data: []byte("conflict")},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			connection := core.NewImplantConnection("invalid-operator", "test")
			connection.Send = make(chan *sliverpb.Envelope, 4)
			session := core.NewSession(connection)
			core.Sessions.Add(session)
			t.Cleanup(func() {
				core.Sessions.Remove(session.ID)
				connection.Close()
			})
			server := &Server{}
			created, err := server.CreateSocks(context.Background(), &sliverpb.Socks{SessionID: session.ID})
			if err != nil {
				t.Fatalf("create SOCKS tunnel: %v", err)
			}
			tunnel := core.SocksTunnels.Get(created.TunnelID)
			if tunnel == nil {
				t.Fatal("created SOCKS tunnel was not registered")
			}

			streamContext, cancelStream := context.WithCancel(context.Background())
			defer cancelStream()
			stream := &controllableSocksProxyServer{
				ctx:  streamContext,
				recv: make(chan *sliverpb.SocksData, len(test.frames)+1),
				sent: make(chan *sliverpb.SocksData, 2),
			}
			result := make(chan error, 1)
			go func() { result <- server.SocksProxy(stream) }()
			request := &commonpb.Request{SessionID: session.ID}
			stream.recv <- &sliverpb.SocksData{TunnelID: tunnel.ID, Sequence: socksLifecycleBindSequence, Request: request}
			waitForRPCSocksCondition(t, func() bool { return tunnel.Client() != nil }, "bind did not claim SOCKS tunnel")
			for _, frame := range test.frames {
				frameCopy := proto.Clone(frame).(*sliverpb.SocksData)
				frameCopy.TunnelID = tunnel.ID
				frameCopy.Request = request
				stream.recv <- frameCopy
			}

			select {
			case err := <-result:
				if err == nil {
					t.Fatal("invalid operator framing returned nil error")
				}
			case <-time.After(time.Second):
				t.Fatal("invalid operator framing did not terminate proxy handler")
			}
			if got := core.SocksTunnels.Get(tunnel.ID); got != nil {
				t.Fatalf("invalid operator framing retained tunnel: %p", got)
			}
			select {
			case terminal := <-stream.sent:
				if terminal.TunnelID != tunnel.ID || !terminal.CloseConn {
					t.Fatalf("operator rejection terminal = %+v", terminal)
				}
			case <-time.After(time.Second):
				t.Fatal("invalid operator framing did not notify client")
			}
		})
	}
}

func TestSocksProxyClosedBindRaceDoesNotStopSibling(t *testing.T) {
	connection := core.NewImplantConnection("closed-bind-race", "test")
	connection.Send = make(chan *sliverpb.Envelope, 8)
	session := core.NewSession(connection)
	core.Sessions.Add(session)
	server := &Server{}
	closing := createSocksTunnelForRPCStreamTest(t, server, session, 0)
	sibling := createSocksTunnelForRPCStreamTest(t, server, session, 0)

	streamContext, cancelStream := context.WithCancel(context.Background())
	stream := &controllableSocksProxyServer{
		ctx:  streamContext,
		recv: make(chan *sliverpb.SocksData, 4),
		sent: make(chan *sliverpb.SocksData, 4),
	}
	t.Cleanup(func() {
		cancelStream()
		core.SocksTunnels.CloseIf(closing)
		core.SocksTunnels.CloseIf(sibling)
		core.Sessions.RemoveIf(session)
		connection.Close()
	})

	var closed atomic.Bool
	getTunnel := func(tunnelID uint64) *core.TcpTunnel {
		tunnel := core.SocksTunnels.Get(tunnelID)
		if tunnelID == closing.ID && closed.CompareAndSwap(false, true) {
			// Return the generation found by lookup after synchronously closing it,
			// reproducing the exact lookup-to-bind lifecycle race.
			server.closeSocksTunnel(tunnel)
		}
		return tunnel
	}
	result := make(chan error, 1)
	go func() {
		result <- server.socksProxy(stream, getTunnel, socksLegacyTerminalReorderGrace, nil, nil)
	}()

	request := &commonpb.Request{SessionID: session.ID}
	stream.recv <- &sliverpb.SocksData{TunnelID: closing.ID, Sequence: socksLifecycleBindSequence, Request: request}
	stream.recv <- &sliverpb.SocksData{TunnelID: sibling.ID, Sequence: socksLifecycleBindSequence, Request: request}
	stream.recv <- &sliverpb.SocksData{TunnelID: sibling.ID, Sequence: 0, Data: []byte("sibling-after-bind-race"), Request: request}

	select {
	case terminal := <-stream.sent:
		if terminal.TunnelID != closing.ID || !terminal.CloseConn {
			t.Fatalf("closed bind race terminal = %+v", terminal)
		}
	case <-time.After(time.Second):
		t.Fatal("closed bind race did not notify the exact client tunnel")
	}
	requireRPCSocksImplantPayload(t, connection, sibling.ID, "sibling-after-bind-race")
	requireRPCSocksSiblingAlive(t, session, connection, sibling, result)

	cancelStream()
	requireCanceledSocksProxyResult(t, result, "closed bind race")
}

//nolint:gocyclo // The coordinated queue saturation proves the close race remains tunnel-local.
func TestSocksProxyClosedAdmissionRaceDoesNotStopSibling(t *testing.T) {
	const receiveWindow = 128

	connection := core.NewImplantConnection("closed-admission-race", "test")
	connection.Send = make(chan *sliverpb.Envelope, 8)
	session := core.NewSession(connection)
	core.Sessions.Add(session)
	server := &Server{}
	closing := createSocksTunnelForRPCStreamTest(t, server, session, 0)
	sibling := createSocksTunnelForRPCStreamTest(t, server, session, 0)

	closing.ToImplantMux.Lock()
	muxLocked := true
	streamContext, cancelStream := context.WithCancel(context.Background())
	stream := &controllableSocksProxyServer{
		ctx:  streamContext,
		recv: make(chan *sliverpb.SocksData, receiveWindow+4),
		sent: make(chan *sliverpb.SocksData, 4),
	}
	result := make(chan error, 1)
	go func() { result <- server.SocksProxy(stream) }()
	t.Cleanup(func() {
		cancelStream()
		if muxLocked {
			closing.ToImplantMux.Unlock()
		}
		core.SocksTunnels.CloseIf(closing)
		core.SocksTunnels.CloseIf(sibling)
		core.Sessions.RemoveIf(session)
		connection.Close()
	})

	request := &commonpb.Request{SessionID: session.ID}
	for _, tunnel := range []*core.TcpTunnel{closing, sibling} {
		stream.recv <- &sliverpb.SocksData{TunnelID: tunnel.ID, Sequence: socksLifecycleBindSequence, Request: request}
	}
	waitForRPCSocksCondition(t, func() bool {
		return closing.Client() != nil && sibling.Client() != nil
	}, "admission race did not bind both SOCKS tunnels")
	for sequence := range receiveWindow {
		stream.recv <- &sliverpb.SocksData{
			TunnelID: closing.ID,
			Sequence: uint64(sequence),
			Data:     []byte("queued"),
			Request:  request,
		}
	}
	stream.recv <- &sliverpb.SocksData{TunnelID: closing.ID, Sequence: receiveWindow, Data: []byte("waiting"), Request: request}
	stream.recv <- &sliverpb.SocksData{TunnelID: sibling.ID, Sequence: 0, Data: []byte("sibling-after-admission-race"), Request: request}
	waitForRPCSocksCondition(t, func() bool { return len(stream.recv) == 0 }, "proxy receiver did not reach closed admission waiter")

	closeDone := make(chan struct{})
	go func() {
		server.closeSocksTunnel(closing)
		close(closeDone)
	}()
	select {
	case <-closing.Done():
	case <-time.After(time.Second):
		t.Fatal("admission-race tunnel did not begin closing")
	}
	closing.ToImplantMux.Unlock()
	muxLocked = false
	select {
	case <-closeDone:
	case <-time.After(time.Second):
		t.Fatal("admission-race tunnel close did not finish")
	}
	select {
	case terminal := <-stream.sent:
		if terminal.TunnelID != closing.ID || !terminal.CloseConn {
			t.Fatalf("closed admission race terminal = %+v", terminal)
		}
	case <-time.After(time.Second):
		t.Fatal("closed admission race did not notify the exact client tunnel")
	}
	requireRPCSocksImplantPayload(t, connection, sibling.ID, "sibling-after-admission-race")
	requireRPCSocksSiblingAlive(t, session, connection, sibling, result)

	cancelStream()
	requireCanceledSocksProxyResult(t, result, "closed admission race")
}

//nolint:gocyclo // The backpressure and shared-stream assertions share one coordinated lifecycle.
func TestSocksProxyWaitsForOperatorIngressCapacity(t *testing.T) {
	const receiveWindow = 128

	connection := core.NewImplantConnection("operator-pressure", "test")
	connection.Send = make(chan *sliverpb.Envelope, 4)
	session := core.NewSession(connection)
	core.Sessions.Add(session)
	t.Cleanup(func() {
		core.Sessions.Remove(session.ID)
		connection.Close()
	})
	server := &Server{}
	create := func(label string) *core.TcpTunnel {
		t.Helper()
		created, err := server.CreateSocks(context.Background(), &sliverpb.Socks{SessionID: session.ID})
		if err != nil {
			t.Fatalf("create %s SOCKS tunnel: %v", label, err)
		}
		tunnel := core.SocksTunnels.Get(created.TunnelID)
		if tunnel == nil {
			t.Fatalf("%s SOCKS tunnel was not registered", label)
		}
		return tunnel
	}
	saturated := create("saturated")
	sibling := create("sibling")

	// Prevent the saturated tunnel's implant sender from completing a frame so
	// all reservations remain charged while the shared proxy stream stays live.
	saturated.ToImplantMux.Lock()
	muxLocked := true
	defer func() {
		if muxLocked {
			saturated.ToImplantMux.Unlock()
		}
	}()

	streamContext, cancelStream := context.WithCancel(context.Background())
	defer cancelStream()
	stream := &controllableSocksProxyServer{
		ctx:  streamContext,
		recv: make(chan *sliverpb.SocksData, receiveWindow+4),
		sent: make(chan *sliverpb.SocksData, 4),
	}
	result := make(chan error, 1)
	go func() { result <- server.SocksProxy(stream) }()
	request := &commonpb.Request{SessionID: session.ID}
	for _, tunnel := range []*core.TcpTunnel{saturated, sibling} {
		stream.recv <- &sliverpb.SocksData{
			TunnelID: tunnel.ID,
			Sequence: socksLifecycleBindSequence,
			Request:  request,
		}
	}
	waitForRPCSocksCondition(t, func() bool {
		return saturated.Client() != nil && sibling.Client() != nil
	}, "proxy stream did not bind both SOCKS tunnels")

	for sequence := range receiveWindow {
		stream.recv <- &sliverpb.SocksData{
			TunnelID: saturated.ID,
			Sequence: uint64(sequence),
			Data:     []byte("queued"),
			Request:  request,
		}
	}
	stream.recv <- &sliverpb.SocksData{
		TunnelID: saturated.ID,
		Sequence: receiveWindow,
		Data:     []byte("overflow"),
		Request:  request,
	}
	// This sibling frame is queued behind the saturated frame on the shared
	// gRPC stream. The minimal bounded fix intentionally accepts this temporary
	// head-of-line backpressure rather than converting valid traffic into EOF.
	stream.recv <- &sliverpb.SocksData{
		TunnelID: sibling.ID,
		Sequence: 0,
		Data:     []byte("healthy"),
		Request:  request,
	}
	waitForRPCSocksCondition(t, func() bool {
		return len(stream.recv) == 0
	}, "proxy receiver did not reach the saturated admission waiter")
	if got := core.SocksTunnels.Get(saturated.ID); got != saturated {
		t.Fatalf("ordinary ingress pressure removed saturated tunnel: got=%p want=%p", got, saturated)
	}
	if got := core.SocksTunnels.Get(sibling.ID); got != sibling {
		t.Fatalf("ordinary ingress pressure removed sibling tunnel: got=%p want=%p", got, sibling)
	}
	select {
	case terminal := <-stream.sent:
		t.Fatalf("ordinary ingress pressure emitted operator terminal: %+v", terminal)
	default:
	}
	select {
	case err := <-result:
		t.Fatalf("ordinary ingress pressure terminated shared SOCKS stream: %v", err)
	default:
	}

	saturated.ToImplantMux.Unlock()
	muxLocked = false
	seenSaturated := map[uint64]bool{}
	siblingForwarded := false
	deadline := time.NewTimer(3 * time.Second)
	defer deadline.Stop()
	for len(seenSaturated) < receiveWindow+1 || !siblingForwarded {
		select {
		case envelope := <-connection.Send:
			if envelope.Type != sliverpb.MsgSocksData {
				t.Fatalf("implant envelope type = %d, want %d", envelope.Type, sliverpb.MsgSocksData)
			}
			frame := &sliverpb.SocksData{}
			if err := proto.Unmarshal(envelope.Data, frame); err != nil {
				t.Fatalf("decode backpressured implant frame: %v", err)
			}
			switch frame.TunnelID {
			case saturated.ID:
				if frame.CloseConn || frame.Sequence > receiveWindow || string(frame.Data) != "queued" && string(frame.Data) != "overflow" {
					t.Fatalf("saturated implant frame = %+v", frame)
				}
				if seenSaturated[frame.Sequence] {
					t.Fatalf("duplicate saturated sequence %d", frame.Sequence)
				}
				seenSaturated[frame.Sequence] = true
			case sibling.ID:
				if frame.CloseConn || frame.Sequence != 0 || string(frame.Data) != "healthy" {
					t.Fatalf("sibling implant frame = %+v", frame)
				}
				siblingForwarded = true
			default:
				t.Fatalf("unexpected tunnel frame = %+v", frame)
			}
		case <-deadline.C:
			t.Fatalf("backpressured delivery incomplete: saturated=%d/%d sibling=%t", len(seenSaturated), receiveWindow+1, siblingForwarded)
		}
	}
	if got := core.SocksTunnels.Get(saturated.ID); got != saturated {
		t.Fatalf("drained ingress removed saturated tunnel: got=%p want=%p", got, saturated)
	}
	if got := core.SocksTunnels.Get(sibling.ID); got != sibling {
		t.Fatalf("drained ingress removed sibling tunnel: got=%p want=%p", got, sibling)
	}

	cancelStream()
	select {
	case err := <-result:
		if !isCanceledSocksProxyError(err) {
			t.Fatalf("SocksProxy cancellation error = %v, want %v", err, context.Canceled)
		}
	case <-time.After(time.Second):
		t.Fatal("SocksProxy did not exit after backpressure test")
	}
}

//nolint:gocyclo // The scenario asserts the full ordered data-and-terminal lifecycle.
func TestSocksProxyForwardsFinalDataThenTerminalWithoutDuplicateClose(t *testing.T) {
	connection := core.NewImplantConnection("ordered-terminal", "test")
	connection.Send = make(chan *sliverpb.Envelope, 4)
	session := core.NewSession(connection)
	core.Sessions.Add(session)
	t.Cleanup(func() {
		core.Sessions.Remove(session.ID)
		connection.Close()
	})

	server := &Server{}
	created, err := server.CreateSocks(context.Background(), &sliverpb.Socks{SessionID: session.ID})
	if err != nil {
		t.Fatalf("create SOCKS tunnel: %v", err)
	}
	streamContext, cancelStream := context.WithCancel(context.Background())
	stream := &controllableSocksProxyServer{
		ctx:  streamContext,
		recv: make(chan *sliverpb.SocksData, 3),
		sent: make(chan *sliverpb.SocksData, 2),
	}
	result := make(chan error, 1)
	go func() { result <- server.SocksProxy(stream) }()
	request := &commonpb.Request{Async: true, SessionID: session.ID}
	stream.recv <- &sliverpb.SocksData{
		TunnelID: created.TunnelID,
		Sequence: socksLifecycleBindSequence,
		Username: "canonical-user",
		Password: "canonical-password",
		Request:  request,
	}
	stream.recv <- &sliverpb.SocksData{
		TunnelID: created.TunnelID,
		Sequence: 0,
		Data:     []byte("final-data"),
		Username: "ignored-later-user",
		Password: "ignored-later-password",
		Request:  request,
	}
	stream.recv <- &sliverpb.SocksData{
		TunnelID:  created.TunnelID,
		Sequence:  1,
		CloseConn: true,
		Request:   request,
	}

	frames := make([]*sliverpb.SocksData, 0, 2)
	for len(frames) < 2 {
		select {
		case envelope := <-connection.Send:
			frame := &sliverpb.SocksData{}
			if err := proto.Unmarshal(envelope.Data, frame); err != nil {
				t.Fatalf("decode implant frame %d: %v", len(frames), err)
			}
			frames = append(frames, frame)
		case <-time.After(time.Second):
			t.Fatalf("received %d implant frames, want final data and terminal", len(frames))
		}
	}
	if frames[0].CloseConn || frames[0].Sequence != 0 || string(frames[0].Data) != "final-data" ||
		frames[0].Username != "canonical-user" || frames[0].Password != "canonical-password" || frames[0].Request != nil {
		t.Fatalf("implant data frame = %+v", frames[0])
	}
	if !frames[1].CloseConn || frames[1].Sequence != 1 || len(frames[1].Data) != 0 {
		t.Fatalf("implant terminal frame = %+v", frames[1])
	}
	waitForRPCSocksCondition(t, func() bool {
		return core.SocksTunnels.Get(created.TunnelID) == nil
	}, "SOCKS tunnel remained registered after ordered terminal")
	select {
	case duplicate := <-connection.Send:
		t.Fatalf("ordered terminal produced duplicate implant close: %+v", duplicate)
	default:
	}
	select {
	case clientClose := <-stream.sent:
		if !clientClose.CloseConn || clientClose.TunnelID != created.TunnelID {
			t.Fatalf("client close = %+v", clientClose)
		}
	case <-time.After(time.Second):
		t.Fatal("client was not notified after ordered terminal")
	}

	cancelStream()
	select {
	case err := <-result:
		if !isCanceledSocksProxyError(err) {
			t.Fatalf("SocksProxy cancellation error = %v, want %v", err, context.Canceled)
		}
	case <-time.After(time.Second):
		t.Fatal("SocksProxy did not exit after ordered terminal test")
	}
}

//nolint:gocyclo // Each mixed-version ordering validates data, terminal, and sibling survival together.
func TestSocksProxyLegacyImplantTerminalIsUnsequencedAndTunnelLocal(t *testing.T) {
	tests := []struct {
		name          string
		dataBeforeEOF bool
		dataAfterEOF  bool
	}{
		{name: "data sequence zero then close sequence zero", dataBeforeEOF: true},
		{name: "close sequence zero then delayed data sequence zero", dataAfterEOF: true},
		{name: "close sequence zero expires after quiet window"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			connection := core.NewImplantConnection("legacy-implant-terminal", "test")
			connection.Send = make(chan *sliverpb.Envelope, 8)
			session := core.NewSession(connection)
			// A capability-zero session models an older implant while the ownership
			// marker below models a current lifecycle-aware operator client.
			core.Sessions.Add(session)
			server := &Server{}
			legacy := createSocksTunnelForRPCStreamTest(t, server, session, sliverpb.CapabilitySocksFlowControlV1)
			sibling := createSocksTunnelForRPCStreamTest(t, server, session, sliverpb.CapabilitySocksFlowControlV1)
			if legacy.FlowControlEnabled() || sibling.FlowControlEnabled() {
				t.Fatal("capability-zero implant unexpectedly negotiated SOCKS flow control")
			}

			streamContext, cancelStream := context.WithCancel(context.Background())
			stream := &controllableSocksProxyServer{
				ctx:  streamContext,
				recv: make(chan *sliverpb.SocksData, 4),
				sent: make(chan *sliverpb.SocksData, 8),
			}
			legacyTerminalExpiry := make(chan time.Time, 1)
			legacyTerminalWaitStarted := make(chan uint64, 2)
			result := make(chan error, 1)
			go func() {
				result <- server.socksProxy(
					stream,
					core.SocksTunnels.Get,
					socksLegacyTerminalReorderGrace,
					legacyTerminalExpiry,
					legacyTerminalWaitStarted,
				)
			}()
			t.Cleanup(func() {
				cancelStream()
				core.SocksTunnels.CloseIf(legacy)
				core.SocksTunnels.CloseIf(sibling)
				core.Sessions.RemoveIf(session)
				connection.Close()
			})

			request := &commonpb.Request{SessionID: session.ID}
			for _, tunnel := range []*core.TcpTunnel{legacy, sibling} {
				stream.recv <- &sliverpb.SocksData{TunnelID: tunnel.ID, Sequence: socksLifecycleBindSequence, Request: request}
			}
			waitForRPCSocksCondition(t, func() bool {
				return legacy.Client() != nil && sibling.Client() != nil
			}, "mixed-version stream did not bind both SOCKS tunnels")
			if lifecycle := legacy.ClientLifecycle(); !lifecycle.SendsTerminal {
				t.Fatalf("current operator ownership marker did not enable lifecycle terminals: %+v", lifecycle)
			}

			if test.dataBeforeEOF {
				if err := legacy.ProcessDataFromImplant(&sliverpb.SocksData{
					TunnelID: legacy.ID,
					Sequence: 0,
					Data:     []byte("legacy-data-zero"),
				}); err != nil {
					t.Fatalf("admit legacy implant data: %v", err)
				}
			}
			if err := legacy.ProcessDataFromImplant(&sliverpb.SocksData{
				TunnelID:  legacy.ID,
				Sequence:  0,
				CloseConn: true,
			}); err != nil {
				t.Fatalf("admit unsequenced legacy implant terminal: %v", err)
			}
			if got := core.SocksTunnels.Get(legacy.ID); got != legacy {
				t.Fatalf("provisional legacy terminal closed tunnel before expiry: got=%p want=%p", got, legacy)
			}
			pending, terminalGeneration, _ := legacy.LegacyImplantTerminalState()
			if !pending {
				t.Fatal("legacy terminal was not retained provisionally before expiry")
			}
			if waitingGeneration := receiveLegacyTerminalWaitGeneration(t, legacyTerminalWaitStarted); waitingGeneration != terminalGeneration {
				t.Fatalf("legacy terminal actor waiting generation = %d, want %d", waitingGeneration, terminalGeneration)
			}
			if test.dataAfterEOF {
				if err := legacy.ProcessDataFromImplant(&sliverpb.SocksData{
					TunnelID: legacy.ID,
					Sequence: 0,
					Data:     []byte("legacy-data-zero"),
				}); err != nil {
					t.Fatalf("admit legacy data overtaken by terminal: %v", err)
				}
				stillPending, refreshedGeneration, _ := legacy.LegacyImplantTerminalState()
				if !stillPending || refreshedGeneration <= terminalGeneration {
					t.Fatalf("late legacy data did not refresh terminal generation: pending=%t before=%d after=%d", stillPending, terminalGeneration, refreshedGeneration)
				}
				if waitingGeneration := receiveLegacyTerminalWaitGeneration(t, legacyTerminalWaitStarted); waitingGeneration != refreshedGeneration {
					t.Fatalf("legacy terminal actor did not reposition to refreshed generation: got=%d want=%d", waitingGeneration, refreshedGeneration)
				}
			}
			// Exactly one pulse expires the generation the actor reported above.
			// The close-first case cannot reach this point unless late data woke the
			// prior wait and repositioned the actor to its refreshed generation.
			legacyTerminalExpiry <- time.Now()

			if test.dataBeforeEOF || test.dataAfterEOF {
				dataFrame := receiveRPCSocksClientFrame(t, stream, legacy.ID)
				if dataFrame.CloseConn || dataFrame.Sequence != 0 || string(dataFrame.Data) != "legacy-data-zero" {
					t.Fatalf("legacy client data frame = %+v", dataFrame)
				}
			}
			terminal := receiveRPCSocksClientFrame(t, stream, legacy.ID)
			wantTerminalSequence := uint64(0)
			if test.dataBeforeEOF || test.dataAfterEOF {
				wantTerminalSequence = 1
			}
			if !terminal.CloseConn || len(terminal.Data) != 0 || terminal.Sequence != wantTerminalSequence {
				t.Fatalf("legacy client terminal = %+v, want sequence %d", terminal, wantTerminalSequence)
			}
			waitForRPCSocksCondition(t, func() bool {
				return core.SocksTunnels.Get(legacy.ID) == nil
			}, "legacy terminal did not retire its exact tunnel")

			if err := sibling.ProcessDataFromImplant(&sliverpb.SocksData{
				TunnelID: sibling.ID,
				Sequence: 0,
				Data:     []byte("legacy-terminal-sibling"),
			}); err != nil {
				t.Fatalf("admit sibling implant data: %v", err)
			}
			siblingFrame := receiveRPCSocksClientFrame(t, stream, sibling.ID)
			if siblingFrame.CloseConn || siblingFrame.Sequence != 0 || string(siblingFrame.Data) != "legacy-terminal-sibling" {
				t.Fatalf("sibling client frame = %+v", siblingFrame)
			}
			requireRPCSocksSiblingAlive(t, session, connection, sibling, result)

			cancelStream()
			requireCanceledSocksProxyResult(t, result, "legacy implant terminal")
		})
	}
}

func TestCreateSocksUnboundTunnelClosesWithSessionConnection(t *testing.T) {
	connection := core.NewImplantConnection("unbound-session", "test")
	connection.Send = make(chan *sliverpb.Envelope, 1)
	session := core.NewSession(connection)
	core.Sessions.Add(session)
	t.Cleanup(func() {
		core.Sessions.Remove(session.ID)
		connection.Close()
	})

	created, err := (&Server{}).CreateSocks(context.Background(), &sliverpb.Socks{SessionID: session.ID})
	if err != nil {
		t.Fatalf("create SOCKS tunnel: %v", err)
	}
	if core.SocksTunnels.Get(created.TunnelID) == nil {
		t.Fatal("created SOCKS tunnel was not registered")
	}
	connection.Close()
	waitForRPCSocksCondition(t, func() bool {
		return core.SocksTunnels.Get(created.TunnelID) == nil
	}, "unbound SOCKS tunnel remained after session connection failure")
}

//nolint:gocyclo // The scenario verifies every bound-tunnel teardown invariant.
func TestSessionRemovalFinalizesBoundSocksTunnelBeforeReturning(t *testing.T) {
	connection := core.NewImplantConnection("session-removal", "test")
	connection.Send = make(chan *sliverpb.Envelope, 2)
	session := core.NewSession(connection)
	core.Sessions.Add(session)
	server := &Server{}
	created, err := server.CreateSocks(context.Background(), &sliverpb.Socks{SessionID: session.ID})
	if err != nil {
		t.Fatalf("create SOCKS tunnel: %v", err)
	}
	tunnel := core.SocksTunnels.Get(created.TunnelID)
	if tunnel == nil {
		t.Fatal("created SOCKS tunnel was not registered")
	}
	if tunnel.ImplantConnection() != connection {
		t.Fatal("created SOCKS tunnel did not retain its exact implant connection")
	}
	sender := &recordingSocksSender{}
	client := core.NewSocksClient(sender)
	if owned, newlyBound := tunnel.BindClient(client); !owned || !newlyBound {
		t.Fatalf("bind SOCKS client = owned:%v new:%v", owned, newlyBound)
	}
	if err := tunnel.AdmitToImplant(&sliverpb.SocksData{TunnelID: tunnel.ID, Sequence: 7}); err != nil {
		t.Fatalf("admit pending operator frame: %v", err)
	}
	if err := tunnel.ProcessDataFromImplant(&sliverpb.SocksData{TunnelID: tunnel.ID, Sequence: 9}); err != nil {
		t.Fatalf("admit pending implant frame: %v", err)
	}
	atomic.StoreUint64(&tunnel.ToImplantSequence, 3)

	// Keep the transport itself open: cleanup must be driven by session removal,
	// not by the lifecycle monitor observing Connection.Done.
	core.Sessions.Remove(session.ID)
	t.Cleanup(connection.Close)
	if got := core.SocksTunnels.Get(tunnel.ID); got != nil {
		t.Fatalf("removed session retained SOCKS tunnel: %p", got)
	}
	select {
	case <-tunnel.Done():
	default:
		t.Fatal("removed session did not close SOCKS tunnel")
	}
	if err := tunnel.AdmitToImplant(&sliverpb.SocksData{TunnelID: tunnel.ID, Sequence: 0}); !errors.Is(err, core.ErrTunnelClosed) {
		t.Fatalf("closed tunnel operator admission = %v, want ErrTunnelClosed", err)
	}
	if err := tunnel.ProcessDataFromImplant(&sliverpb.SocksData{TunnelID: tunnel.ID, Sequence: 0}); !errors.Is(err, core.ErrTunnelClosed) {
		t.Fatalf("closed tunnel implant admission = %v, want ErrTunnelClosed", err)
	}
	frames := sender.snapshot()
	if len(frames) != 1 || !frames[0].CloseConn || frames[0].TunnelID != tunnel.ID {
		t.Fatalf("session removal client notifications = %+v, want one terminal", frames)
	}
	var implantTerminal sliverpb.SocksData
	select {
	case envelope := <-connection.Send:
		if envelope.Type != sliverpb.MsgSocksData {
			t.Fatalf("session removal implant envelope type = %d, want MsgSocksData", envelope.Type)
		}
		if err := proto.Unmarshal(envelope.Data, &implantTerminal); err != nil {
			t.Fatalf("decode session removal implant terminal: %v", err)
		}
	default:
		t.Fatal("session removal returned before enqueueing an implant terminal")
	}
	if !implantTerminal.CloseConn || implantTerminal.TunnelID != tunnel.ID || implantTerminal.Sequence != 3 {
		t.Fatalf("session removal implant terminal = %+v, want tunnel %d sequence 3 terminal", &implantTerminal, tunnel.ID)
	}
	select {
	case <-connection.Done():
		t.Fatal("session removal closed the implant transport")
	default:
	}
	core.Sessions.Remove(session.ID)
	if got := len(sender.snapshot()); got != 1 {
		t.Fatalf("repeated session removal produced %d client notifications, want 1", got)
	}
	select {
	case envelope := <-connection.Send:
		t.Fatalf("repeated session removal produced another implant notification: %+v", envelope)
	default:
	}
}

func TestCloseSocksUsesCreatingConnectionAfterSessionGenerationReplacement(t *testing.T) {
	creatingConnection := core.NewImplantConnection("creating-session-generation", "test")
	creatingConnection.Send = make(chan *sliverpb.Envelope, 1)
	creatingSession := core.NewSession(creatingConnection)
	core.Sessions.Add(creatingSession)

	tunnel, err := core.SocksTunnels.Create(creatingSession.ID)
	if err != nil {
		t.Fatalf("create SOCKS tunnel: %v", err)
	}
	atomic.StoreUint64(&tunnel.ToImplantSequence, 4)

	// Replace the registry entry without touching the tunnel. This makes an
	// identifier-based lookup select the wrong transport generation and
	// deterministically exercises the same lookup-independent close behavior as
	// the delete-before-CloseForSession interleaving.
	replacementConnection := core.NewImplantConnection("replacement-session-generation", "test")
	replacementConnection.Send = make(chan *sliverpb.Envelope, 1)
	replacementSession := core.NewSession(replacementConnection)
	replacementSession.ID = creatingSession.ID
	core.Sessions.Add(replacementSession)
	t.Cleanup(func() {
		core.SocksTunnels.CloseIf(tunnel)
		core.Sessions.Remove(replacementSession.ID)
		creatingConnection.Close()
		replacementConnection.Close()
	})

	if tunnel.ImplantConnection() != creatingConnection {
		t.Fatal("registry replacement changed the tunnel's creating connection")
	}
	if _, err := (&Server{}).CloseSocks(context.Background(), &sliverpb.Socks{
		SessionID: creatingSession.ID,
		TunnelID:  tunnel.ID,
	}); err != nil {
		t.Fatalf("close SOCKS tunnel: %v", err)
	}

	var terminal sliverpb.SocksData
	select {
	case envelope := <-creatingConnection.Send:
		if envelope.Type != sliverpb.MsgSocksData {
			t.Fatalf("creating connection envelope type = %d, want MsgSocksData", envelope.Type)
		}
		if err := proto.Unmarshal(envelope.Data, &terminal); err != nil {
			t.Fatalf("decode creating connection terminal: %v", err)
		}
	default:
		t.Fatal("close did not notify the tunnel's creating connection")
	}
	if !terminal.CloseConn || terminal.TunnelID != tunnel.ID || terminal.Sequence != 4 {
		t.Fatalf("creating connection terminal = %+v, want tunnel %d sequence 4 terminal", &terminal, tunnel.ID)
	}
	select {
	case envelope := <-replacementConnection.Send:
		t.Fatalf("close notified replacement session generation: %+v", envelope)
	default:
	}
}

func TestSocksPayloadUsesCreatingConnectionAfterSessionGenerationReplacement(t *testing.T) {
	creatingConnection := core.NewImplantConnection("payload-creating-generation", "test")
	creatingConnection.Send = make(chan *sliverpb.Envelope, 1)
	creatingSession := core.NewSession(creatingConnection)
	core.Sessions.Add(creatingSession)
	tunnel, err := core.SocksTunnels.Create(creatingSession.ID)
	if err != nil {
		t.Fatalf("create SOCKS tunnel: %v", err)
	}
	client := core.NewSocksClient(&recordingSocksSender{})
	if owned, newlyBound := tunnel.BindClient(client); !owned || !newlyBound {
		t.Fatalf("bind SOCKS client = owned:%v new:%v", owned, newlyBound)
	}

	replacementConnection := core.NewImplantConnection("payload-replacement-generation", "test")
	replacementConnection.Send = make(chan *sliverpb.Envelope, 1)
	replacementSession := core.NewSession(replacementConnection)
	replacementSession.ID = creatingSession.ID
	core.Sessions.Add(replacementSession)
	t.Cleanup(func() {
		core.SocksTunnels.CloseIf(tunnel)
		core.Sessions.Remove(replacementSession.ID)
		creatingConnection.Close()
		replacementConnection.Close()
	})

	if err := tunnel.AdmitToImplant(&sliverpb.SocksData{TunnelID: tunnel.ID, Sequence: 0, Data: []byte("exact-generation")}); err != nil {
		t.Fatalf("admit SOCKS payload: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	workerDone := make(chan struct{})
	reported := make(chan error, 1)
	go func() {
		(&Server{}).sendSocksDataToImplant(ctx, tunnel, func(err error) { reported <- err })
		close(workerDone)
	}()

	select {
	case envelope := <-creatingConnection.Send:
		frame := &sliverpb.SocksData{}
		if err := proto.Unmarshal(envelope.Data, frame); err != nil {
			t.Fatalf("decode creating-generation payload: %v", err)
		}
		if frame.TunnelID != tunnel.ID || frame.Sequence != 0 || string(frame.Data) != "exact-generation" {
			t.Fatalf("creating-generation payload = %+v", frame)
		}
	case err := <-reported:
		t.Fatalf("SOCKS payload sender failed: %v", err)
	case <-time.After(time.Second):
		t.Fatal("creating connection did not receive SOCKS payload")
	}
	select {
	case envelope := <-replacementConnection.Send:
		t.Fatalf("replacement session generation received stale SOCKS payload: %+v", envelope)
	default:
	}
	cancel()
	select {
	case <-workerDone:
	case <-time.After(time.Second):
		t.Fatal("SOCKS payload worker did not stop")
	}
}

func TestSocksTerminalSendFailureClosesOnlyCreatingConnection(t *testing.T) {
	creatingConnection := core.NewImplantConnection("terminal-failure-creating", "test")
	creatingConnection.Send = nil // deterministic ErrInvalidImplantSend
	creatingSession := core.NewSession(creatingConnection)
	core.Sessions.Add(creatingSession)
	tunnel, err := core.SocksTunnels.Create(creatingSession.ID)
	if err != nil {
		t.Fatalf("create SOCKS tunnel: %v", err)
	}

	replacementConnection := core.NewImplantConnection("terminal-failure-replacement", "test")
	replacementConnection.Send = make(chan *sliverpb.Envelope, 1)
	replacementSession := core.NewSession(replacementConnection)
	replacementSession.ID = creatingSession.ID
	core.Sessions.Add(replacementSession)
	t.Cleanup(func() {
		core.SocksTunnels.CloseIf(tunnel)
		core.Sessions.Remove(replacementSession.ID)
		creatingConnection.Close()
		replacementConnection.Close()
	})

	if _, err := (&Server{}).CloseSocks(context.Background(), &sliverpb.Socks{TunnelID: tunnel.ID, SessionID: creatingSession.ID}); err != nil {
		t.Fatalf("close SOCKS tunnel: %v", err)
	}
	select {
	case <-creatingConnection.Done():
	case <-time.After(time.Second):
		t.Fatal("failed terminal did not close its exact creating connection")
	}
	select {
	case <-replacementConnection.Done():
		t.Fatal("failed terminal closed the replacement session generation")
	default:
	}
}

func TestUnboundSocksTunnelLeaseExpires(t *testing.T) {
	connection := core.NewImplantConnection("unbound-lease", "test")
	connection.Send = make(chan *sliverpb.Envelope, 1)
	session := core.NewSession(connection)
	core.Sessions.Add(session)
	tunnel, err := core.SocksTunnels.Create(session.ID)
	if err != nil {
		t.Fatalf("create SOCKS tunnel: %v", err)
	}
	t.Cleanup(func() {
		core.SocksTunnels.CloseIf(tunnel)
		core.Sessions.Remove(session.ID)
		connection.Close()
	})

	go (&Server{}).monitorUnboundSocksTunnel(tunnel, session, 25*time.Millisecond)
	waitForRPCSocksCondition(t, func() bool {
		return core.SocksTunnels.Get(tunnel.ID) == nil
	}, "unbound SOCKS tunnel remained after its bind lease expired")
}

func TestBoundSocksTunnelWithoutFirstPayloadLeaseExpires(t *testing.T) {
	connection := core.NewImplantConnection("first-payload-lease", "test")
	connection.Send = make(chan *sliverpb.Envelope, 2)
	session := core.NewSession(connection)
	core.Sessions.Add(session)
	tunnel, err := core.SocksTunnels.Create(session.ID)
	if err != nil {
		t.Fatalf("create SOCKS tunnel: %v", err)
	}
	sender := &recordingSocksSender{}
	if owned, newlyBound, bindErr := tunnel.BindClientWithCapabilities(core.NewSocksClient(sender), "", "", true); bindErr != nil || !owned || !newlyBound {
		t.Fatalf("bind lifecycle-aware client = owned:%v new:%v err:%v", owned, newlyBound, bindErr)
	}
	t.Cleanup(func() {
		core.SocksTunnels.CloseIf(tunnel)
		core.Sessions.Remove(session.ID)
		connection.Close()
	})

	done := make(chan struct{})
	go func() {
		(&Server{}).monitorSocksTunnelWithTimeouts(context.Background(), tunnel, 25*time.Millisecond, time.Second, time.Millisecond)
		close(done)
	}()
	waitForRPCSocksCondition(t, func() bool {
		return core.SocksTunnels.Get(tunnel.ID) == nil
	}, "bound SOCKS tunnel remained after its first-payload lease expired")
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("first-payload lifecycle monitor did not stop")
	}
	frames := sender.snapshot()
	if len(frames) != 1 || !frames[0].CloseConn || frames[0].TunnelID != tunnel.ID {
		t.Fatalf("first-payload expiry client frames = %+v, want exact terminal", frames)
	}
}

//nolint:gocyclo // The scenario couples legacy lease and metadata-scrubbing assertions.
func TestLegacySocksClientIdleCleanupIgnoresRequestAsyncAndStripsMetadata(t *testing.T) {
	connection := core.NewImplantConnection("legacy-idle", "test")
	connection.Send = make(chan *sliverpb.Envelope, 4)
	session := core.NewSession(connection)
	core.Sessions.Add(session)
	server := &Server{}
	created, err := server.CreateSocks(context.Background(), &sliverpb.Socks{SessionID: session.ID})
	if err != nil {
		t.Fatalf("create SOCKS tunnel: %v", err)
	}
	tunnel := core.SocksTunnels.Get(created.TunnelID)
	streamContext, cancelStream := context.WithCancel(context.Background())
	stream := &controllableSocksProxyServer{
		ctx:  streamContext,
		recv: make(chan *sliverpb.SocksData, 1),
		sent: make(chan *sliverpb.SocksData, 2),
	}
	result := make(chan error, 1)
	go func() { result <- server.SocksProxy(stream) }()
	t.Cleanup(func() {
		cancelStream()
		core.SocksTunnels.CloseIf(tunnel)
		core.Sessions.Remove(session.ID)
		connection.Close()
	})

	// Request.Async has beacon-task semantics and is deliberately not a SOCKS
	// lifecycle capability. A legacy sequence-zero payload remains legacy even
	// if this unrelated metadata bit is present.
	stream.recv <- &sliverpb.SocksData{
		TunnelID: created.TunnelID,
		Sequence: 0,
		Data:     []byte("legacy-greeting"),
		Username: "legacy-user",
		Password: "legacy-password",
		Request:  &commonpb.Request{Async: true, SessionID: session.ID},
	}
	waitForRPCSocksCondition(t, func() bool {
		return tunnel.Client() != nil && tunnel.ClientLifecycle().ReceivedPayload
	}, "legacy payload did not bind the SOCKS tunnel")
	if lifecycle := tunnel.ClientLifecycle(); lifecycle.SendsTerminal {
		t.Fatalf("Request.Async incorrectly enabled SOCKS terminal capability: %+v", lifecycle)
	}

	select {
	case envelope := <-connection.Send:
		forwarded := &sliverpb.SocksData{}
		if err := proto.Unmarshal(envelope.Data, forwarded); err != nil {
			t.Fatalf("decode legacy payload: %v", err)
		}
		if forwarded.Request != nil || forwarded.Sequence != 0 || string(forwarded.Data) != "legacy-greeting" ||
			forwarded.Username != "legacy-user" || forwarded.Password != "legacy-password" {
			t.Fatalf("forwarded legacy payload = %+v", forwarded)
		}
	case <-time.After(time.Second):
		t.Fatal("legacy payload was not forwarded to implant")
	}

	monitorDone := make(chan struct{})
	go func() {
		server.monitorSocksTunnelWithTimeouts(streamContext, tunnel, time.Second, 25*time.Millisecond, time.Millisecond)
		close(monitorDone)
	}()
	waitForRPCSocksCondition(t, func() bool {
		return core.SocksTunnels.Get(tunnel.ID) == nil
	}, "legacy SOCKS tunnel remained after bounded inactivity")
	select {
	case terminal := <-stream.sent:
		if terminal.TunnelID != tunnel.ID || !terminal.CloseConn || len(terminal.Data) != 0 {
			t.Fatalf("legacy idle terminal = %+v", terminal)
		}
	case <-time.After(time.Second):
		t.Fatal("legacy idle cleanup did not notify client")
	}
	select {
	case <-monitorDone:
	case <-time.After(time.Second):
		t.Fatal("legacy lifecycle monitor did not stop")
	}
	cancelStream()
	select {
	case err := <-result:
		if !isCanceledSocksProxyError(err) {
			t.Fatalf("legacy SocksProxy cancellation error = %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("legacy SocksProxy did not stop")
	}
}

//nolint:gocyclo // The mixed-version lease assertions cover both endpoint and session lifecycles.
func TestCurrentSocksClientWithLegacyImplantRetainsIdleFallback(t *testing.T) {
	connection := core.NewImplantConnection("current-client-legacy-implant", "test")
	connection.Send = make(chan *sliverpb.Envelope, 4)
	session := core.NewSession(connection)
	core.Sessions.Add(session)
	server := &Server{}
	tunnel := createSocksTunnelForRPCStreamTest(t, server, session, sliverpb.CapabilitySocksFlowControlV1)
	if tunnel.FlowControlEnabled() {
		t.Fatal("capability-zero implant unexpectedly negotiated SOCKS flow control")
	}
	sender := &recordingSocksSender{}
	client := core.NewSocksClient(sender)
	owned, newlyBound, err := tunnel.BindClientWithNegotiatedCapabilities(client, "", "", true, 0)
	if err != nil || !owned || !newlyBound {
		t.Fatalf("bind current client to legacy implant tunnel = owned:%v new:%v err:%v", owned, newlyBound, err)
	}
	if err := tunnel.AdmitToImplant(&sliverpb.SocksData{
		TunnelID: tunnel.ID,
		Sequence: 0,
		Data:     []byte("mixed-version-greeting"),
	}); err != nil {
		t.Fatalf("admit mixed-version greeting: %v", err)
	}
	select {
	case forwarded := <-tunnel.ToImplant():
		tunnel.CompleteToImplant(forwarded)
	case <-time.After(time.Second):
		t.Fatal("mixed-version greeting was not queued to the implant")
	}
	t.Cleanup(func() {
		core.SocksTunnels.CloseIf(tunnel)
		core.Sessions.RemoveIf(session)
		connection.Close()
	})

	monitorDone := make(chan struct{})
	go func() {
		server.monitorSocksTunnelWithTimeouts(context.Background(), tunnel, time.Second, 25*time.Millisecond, time.Millisecond)
		close(monitorDone)
	}()
	waitForRPCSocksCondition(t, func() bool {
		return core.SocksTunnels.Get(tunnel.ID) == nil
	}, "current-client/legacy-implant tunnel bypassed bounded idle cleanup")
	select {
	case <-monitorDone:
	case <-time.After(time.Second):
		t.Fatal("mixed-version lifecycle monitor did not stop")
	}
	frames := sender.snapshot()
	if len(frames) != 1 || !frames[0].CloseConn || frames[0].TunnelID != tunnel.ID {
		t.Fatalf("mixed-version idle client frames = %+v, want one exact terminal", frames)
	}
	if got := core.Sessions.Get(session.ID); got != session {
		t.Fatalf("mixed-version idle cleanup removed session: got=%p want=%p", got, session)
	}
	select {
	case <-connection.Done():
		t.Fatal("mixed-version idle cleanup closed implant connection")
	default:
	}
}

func TestLifecycleAwareSocksClientMayRemainIdleAfterFirstPayload(t *testing.T) {
	connection := core.NewImplantConnection("modern-idle", "test")
	connection.Send = make(chan *sliverpb.Envelope, 4)
	session := core.NewSession(connection)
	session.Capabilities = sliverpb.CapabilitySocksFlowControlV1
	core.Sessions.Add(session)
	server := &Server{}
	created, err := server.CreateSocks(context.Background(), &sliverpb.Socks{
		SessionID:    session.ID,
		Capabilities: sliverpb.CapabilitySocksFlowControlV1,
	})
	if err != nil {
		t.Fatalf("create SOCKS tunnel: %v", err)
	}
	tunnel := core.SocksTunnels.Get(created.TunnelID)
	streamContext, cancelStream := context.WithCancel(context.Background())
	stream := &controllableSocksProxyServer{
		ctx:  streamContext,
		recv: make(chan *sliverpb.SocksData, 2),
		sent: make(chan *sliverpb.SocksData, 2),
	}
	result := make(chan error, 1)
	go func() { result <- server.SocksProxy(stream) }()
	t.Cleanup(func() {
		cancelStream()
		core.SocksTunnels.CloseIf(tunnel)
		core.Sessions.Remove(session.ID)
		connection.Close()
	})

	request := &commonpb.Request{SessionID: session.ID}
	stream.recv <- &sliverpb.SocksData{
		TunnelID:     tunnel.ID,
		Sequence:     socksLifecycleBindSequence,
		Capabilities: sliverpb.CapabilitySocksFlowControlV1,
		Request:      request,
	}
	stream.recv <- &sliverpb.SocksData{TunnelID: tunnel.ID, Sequence: 0, Data: []byte("rdp-negotiation"), Request: request}
	waitForRPCSocksCondition(t, func() bool {
		lifecycle := tunnel.ClientLifecycle()
		return lifecycle.SendsTerminal && lifecycle.ReceivedPayload
	}, "lifecycle-aware payload did not initialize tunnel state")
	select {
	case <-connection.Send:
	case <-time.After(time.Second):
		t.Fatal("lifecycle-aware payload was not forwarded")
	}

	monitorDone := make(chan struct{})
	go func() {
		server.monitorSocksTunnelWithTimeouts(streamContext, tunnel, 20*time.Millisecond, 25*time.Millisecond, time.Millisecond)
		close(monitorDone)
	}()
	time.Sleep(75 * time.Millisecond)
	if got := core.SocksTunnels.Get(tunnel.ID); got != tunnel {
		t.Fatalf("lifecycle-aware established tunnel closed while idle: got=%p want=%p", got, tunnel)
	}
	cancelStream()
	select {
	case <-monitorDone:
	case <-time.After(time.Second):
		t.Fatal("modern lifecycle monitor did not stop after cancellation")
	}
	select {
	case err := <-result:
		if !isCanceledSocksProxyError(err) {
			t.Fatalf("modern SocksProxy cancellation error = %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("modern SocksProxy did not stop")
	}
}

type recordingSocksSender struct {
	mu     sync.Mutex
	frames []*sliverpb.SocksData
}

func (sender *recordingSocksSender) Send(frame *sliverpb.SocksData) error {
	sender.mu.Lock()
	defer sender.mu.Unlock()
	sender.frames = append(sender.frames, proto.Clone(frame).(*sliverpb.SocksData))
	return nil
}

func (sender *recordingSocksSender) snapshot() []*sliverpb.SocksData {
	sender.mu.Lock()
	defer sender.mu.Unlock()
	return append([]*sliverpb.SocksData(nil), sender.frames...)
}

func newRPCSocksTunnel(t *testing.T) (*core.Session, *core.TcpTunnel, *recordingSocksSender) {
	return newRPCSocksTunnelWithCapabilities(t, 0)
}

func newRPCSocksTunnelWithCapabilities(t *testing.T, capabilities uint64) (*core.Session, *core.TcpTunnel, *recordingSocksSender) {
	t.Helper()
	connection := core.NewImplantConnection("test", "test")
	connection.Send = make(chan *sliverpb.Envelope, 8)
	session := core.NewSession(connection)
	session.Capabilities = capabilities
	core.Sessions.Add(session)
	tunnel, err := core.SocksTunnels.CreateWithCapabilities(session.ID, capabilities)
	if err != nil {
		t.Fatalf("create SOCKS tunnel: %v", err)
	}
	sender := &recordingSocksSender{}
	owned, newlyBound, err := tunnel.BindClientWithNegotiatedCapabilities(
		core.NewSocksClient(sender),
		"",
		"",
		capabilities != 0,
		capabilities,
	)
	if err != nil || !owned || !newlyBound {
		t.Fatalf("bind SOCKS test client = owned:%v new:%v err:%v", owned, newlyBound, err)
	}
	t.Cleanup(func() {
		core.SocksTunnels.CloseIf(tunnel)
		core.Sessions.Remove(session.ID)
		connection.Close()
	})
	return session, tunnel, sender
}

func TestRejectSocksTunnelUsesFromImplantTerminalSequence(t *testing.T) {
	_, tunnel, sender := newRPCSocksTunnel(t)
	atomic.StoreUint64(&tunnel.ToImplantSequence, 9)
	atomic.StoreUint64(&tunnel.FromImplantSequence, 3)
	if err := (&Server{}).rejectSocksTunnel(tunnel, tunnel.Client()); err != nil {
		t.Fatalf("reject SOCKS tunnel: %v", err)
	}
	frames := sender.snapshot()
	if len(frames) != 1 || !frames[0].CloseConn || frames[0].Sequence != 3 {
		t.Fatalf("rejection terminal = %+v, want client sequence 3", frames)
	}
}

func TestCloseSocksValidatesOptionalSessionOwnership(t *testing.T) {
	session, tunnel, _ := newRPCSocksTunnel(t)
	server := &Server{}

	if _, err := server.CloseSocks(context.Background(), &sliverpb.Socks{
		TunnelID:  tunnel.ID,
		SessionID: "not-the-owner",
	}); err != ErrInvalidSessionID {
		t.Fatalf("mismatched session close error = %v, want %v", err, ErrInvalidSessionID)
	}
	if got := core.SocksTunnels.Get(tunnel.ID); got != tunnel {
		t.Fatalf("mismatched session close changed tunnel: got=%p want=%p", got, tunnel)
	}

	if _, err := server.CloseSocks(context.Background(), &sliverpb.Socks{
		TunnelID:  tunnel.ID,
		SessionID: session.ID,
	}); err != nil {
		t.Fatalf("owner close: %v", err)
	}
	if got := core.SocksTunnels.Get(tunnel.ID); got != nil {
		t.Fatalf("owner close retained tunnel: %p", got)
	}
}

func TestCloseSocksIsIdempotentUnderConcurrentLifecycleOwners(t *testing.T) {
	session, tunnel, sender := newRPCSocksTunnel(t)
	server := &Server{}
	const closerCount = 32
	start := make(chan struct{})
	var closers sync.WaitGroup
	for index := 0; index < closerCount; index++ {
		closers.Add(1)
		go func() {
			defer closers.Done()
			<-start
			server.closeSocksTunnel(tunnel)
		}()
	}
	close(start)
	closers.Wait()

	frames := sender.snapshot()
	if len(frames) != 1 || !frames[0].CloseConn || frames[0].TunnelID != tunnel.ID {
		t.Fatalf("client close notifications = %+v, want one close for tunnel %d", frames, tunnel.ID)
	}
	select {
	case envelope := <-session.Connection.Send:
		if envelope.Type != sliverpb.MsgSocksData {
			t.Fatalf("implant close envelope type = %d, want %d", envelope.Type, sliverpb.MsgSocksData)
		}
		closeFrame := &sliverpb.SocksData{}
		if err := proto.Unmarshal(envelope.Data, closeFrame); err != nil {
			t.Fatalf("decode implant close frame: %v", err)
		}
		if !closeFrame.CloseConn || closeFrame.TunnelID != tunnel.ID {
			t.Fatalf("implant close frame = %+v", closeFrame)
		}
	case <-time.After(time.Second):
		t.Fatal("implant did not receive SOCKS close notification")
	}
	select {
	case extra := <-session.Connection.Send:
		t.Fatalf("duplicate implant close notification: %+v", extra)
	default:
	}

	if _, err := server.CloseSocks(context.Background(), &sliverpb.Socks{TunnelID: tunnel.ID}); err != nil {
		t.Fatalf("idempotent close of removed tunnel: %v", err)
	}
	if got := len(sender.snapshot()); got != 1 {
		t.Fatalf("idempotent close produced %d client notifications, want 1", got)
	}
}

//nolint:gocyclo // The test keeps out-of-order data, terminal delivery, and cleanup in one scenario.
func TestSocksImplantCloseWaitsForEarlierSequence(t *testing.T) {
	_, tunnel, sender := newRPCSocksTunnelWithCapabilities(t, sliverpb.CapabilitySocksFlowControlV1)
	if !tunnel.FlowControlEnabled() {
		t.Fatal("ordered terminal test did not negotiate SOCKS flow control")
	}
	server := &Server{}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	workerDone := make(chan struct{})
	reported := make(chan error, 1)
	go func() {
		server.sendSocksDataToClient(ctx, tunnel, tunnel.Client(), func(err error) {
			reported <- err
		})
		close(workerDone)
	}()

	if err := tunnel.ProcessDataFromImplant(&sliverpb.SocksData{
		TunnelID:  tunnel.ID,
		Sequence:  1,
		CloseConn: true,
	}); err != nil {
		t.Fatalf("deliver out-of-order implant close: %v", err)
	}
	time.Sleep(20 * time.Millisecond)
	if frames := sender.snapshot(); len(frames) != 0 {
		t.Fatalf("out-of-order close reached client before sequence 0: %+v", frames)
	}
	if core.SocksTunnels.Get(tunnel.ID) != tunnel {
		t.Fatal("out-of-order close removed tunnel before sequence 0")
	}

	if err := tunnel.ProcessDataFromImplant(&sliverpb.SocksData{
		TunnelID: tunnel.ID,
		Sequence: 0,
		Data:     []byte("before-close"),
	}); err != nil {
		t.Fatalf("deliver sequence 0 implant data: %v", err)
	}
	select {
	case <-workerDone:
	case err := <-reported:
		t.Fatalf("SOCKS client sender failed: %v", err)
	case <-time.After(time.Second):
		t.Fatal("SOCKS client sender did not process ordered close")
	}
	frames := sender.snapshot()
	if len(frames) != 2 {
		t.Fatalf("client frames = %+v, want data then close", frames)
	}
	if string(frames[0].Data) != "before-close" || frames[0].CloseConn || frames[0].Sequence != 0 {
		t.Fatalf("first client frame = %+v", frames[0])
	}
	if !frames[1].CloseConn || frames[1].TunnelID != tunnel.ID || frames[1].Sequence != 1 {
		t.Fatalf("second client frame = %+v, want close", frames[1])
	}
	if got := atomic.LoadUint64(&tunnel.FromImplantSequence); got != 1 {
		t.Fatalf("from-implant data high-water = %d, want 1", got)
	}
}

func TestFlowControlledSocksImplantSequenceZeroTerminalIsImmediate(t *testing.T) {
	session, tunnel, sender := newRPCSocksTunnelWithCapabilities(t, sliverpb.CapabilitySocksFlowControlV1)
	if !tunnel.FlowControlEnabled() {
		t.Fatal("sequence-zero terminal test did not negotiate SOCKS flow control")
	}

	server := &Server{}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	workerDone := make(chan struct{})
	reported := make(chan error, 1)
	go func() {
		server.sendSocksDataToClient(ctx, tunnel, tunnel.Client(), func(err error) {
			reported <- err
		})
		close(workerDone)
	}()

	if err := tunnel.ProcessDataFromImplant(&sliverpb.SocksData{
		TunnelID:  tunnel.ID,
		Sequence:  0,
		CloseConn: true,
	}); err != nil {
		t.Fatalf("deliver flow-controlled sequence-zero terminal: %v", err)
	}
	select {
	case <-workerDone:
	case err := <-reported:
		t.Fatalf("SOCKS client sender failed: %v", err)
	case <-time.After(time.Second):
		t.Fatal("flow-controlled sequence-zero terminal entered the legacy reorder wait")
	}
	frames := sender.snapshot()
	if len(frames) != 1 || !frames[0].CloseConn || frames[0].TunnelID != tunnel.ID || frames[0].Sequence != 0 || len(frames[0].Data) != 0 {
		t.Fatalf("flow-controlled sequence-zero client terminal = %+v, want one immediate empty terminal", frames)
	}
	if got := core.SocksTunnels.Get(tunnel.ID); got != nil {
		t.Fatalf("flow-controlled sequence-zero terminal retained tunnel: %p", got)
	}
	if got := core.Sessions.Get(session.ID); got != session {
		t.Fatalf("flow-controlled sequence-zero terminal removed session: got=%p want=%p", got, session)
	}
	select {
	case <-session.Connection.Done():
		t.Fatal("flow-controlled sequence-zero terminal closed implant connection")
	default:
	}
}

//nolint:gocyclo // The repeated race scenario validates ordering and cleanup as one unit.
func TestSocksImplantQueueAndCloseUseContiguousSequencesUnderRace(t *testing.T) {
	const iterations = 32
	for iteration := 0; iteration < iterations; iteration++ {
		connection := core.NewImplantConnection("queue-close-race", "test")
		connection.Send = make(chan *sliverpb.Envelope)
		session := core.NewSession(connection)
		core.Sessions.Add(session)
		tunnel, err := core.SocksTunnels.Create(session.ID)
		if err != nil {
			t.Fatalf("iteration %d create SOCKS tunnel: %v", iteration, err)
		}
		sender := &recordingSocksSender{}
		owned, newlyBound := tunnel.BindClient(core.NewSocksClient(sender))
		if !owned || !newlyBound {
			t.Fatalf("iteration %d bind client = owned:%v new:%v", iteration, owned, newlyBound)
		}
		if err := tunnel.AdmitToImplant(&sliverpb.SocksData{
			TunnelID: tunnel.ID,
			Sequence: 0,
			Data:     []byte("queued-before-close"),
		}); err != nil {
			t.Fatalf("iteration %d admit queued frame: %v", iteration, err)
		}

		ctx, cancel := context.WithCancel(context.Background())
		workerDone := make(chan struct{})
		reported := make(chan error, 1)
		go func() {
			(&Server{}).sendSocksDataToImplant(ctx, tunnel, func(err error) { reported <- err })
			close(workerDone)
		}()

		var dataEnvelope *sliverpb.Envelope
		select {
		case dataEnvelope = <-connection.Send:
		case err := <-reported:
			t.Fatalf("iteration %d data sender failed: %v", iteration, err)
		case <-time.After(time.Second):
			t.Fatalf("iteration %d data was not queued to implant", iteration)
		}
		closeDone := make(chan error, 1)
		go func() {
			_, closeErr := (&Server{}).CloseSocks(context.Background(), &sliverpb.Socks{
				TunnelID:  tunnel.ID,
				SessionID: session.ID,
			})
			closeDone <- closeErr
		}()
		var closeEnvelope *sliverpb.Envelope
		select {
		case closeEnvelope = <-connection.Send:
		case err := <-reported:
			t.Fatalf("iteration %d sender failed during close race: %v", iteration, err)
		case <-time.After(time.Second):
			t.Fatalf("iteration %d close was not queued to implant", iteration)
		}

		dataFrame := &sliverpb.SocksData{}
		if err := proto.Unmarshal(dataEnvelope.Data, dataFrame); err != nil {
			t.Fatalf("iteration %d decode data frame: %v", iteration, err)
		}
		closeFrame := &sliverpb.SocksData{}
		if err := proto.Unmarshal(closeEnvelope.Data, closeFrame); err != nil {
			t.Fatalf("iteration %d decode close frame: %v", iteration, err)
		}
		if dataFrame.CloseConn || dataFrame.Sequence != 0 {
			t.Fatalf("iteration %d data frame = %+v, want sequence 0", iteration, dataFrame)
		}
		if !closeFrame.CloseConn || closeFrame.Sequence != 1 {
			t.Fatalf("iteration %d close frame = %+v, want terminal sequence 1", iteration, closeFrame)
		}
		select {
		case closeErr := <-closeDone:
			if closeErr != nil {
				t.Fatalf("iteration %d unary close failed: %v", iteration, closeErr)
			}
		case <-time.After(time.Second):
			t.Fatalf("iteration %d close did not finish", iteration)
		}
		cancel()
		select {
		case <-workerDone:
		case <-time.After(time.Second):
			t.Fatalf("iteration %d data sender did not finish", iteration)
		}
		core.SocksTunnels.CloseIf(tunnel)
		core.Sessions.Remove(session.ID)
		connection.Close()
	}
}

type blockingCloseSocksSender struct {
	started chan struct{}
	release chan struct{}
	once    sync.Once
	calls   atomic.Int32
}

func (sender *blockingCloseSocksSender) Send(*sliverpb.SocksData) error {
	sender.calls.Add(1)
	sender.once.Do(func() { close(sender.started) })
	<-sender.release
	return nil
}

func TestNotifySocksClientTimeoutFailsStreamWithoutGoroutineGrowth(t *testing.T) {
	sender := &blockingCloseSocksSender{started: make(chan struct{}), release: make(chan struct{})}
	client := core.NewSocksClient(sender)
	const timeout = 25 * time.Millisecond
	if err := notifySocksClient(client, &sliverpb.SocksData{CloseConn: true}, timeout); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("first stalled notification error = %v, want context.DeadlineExceeded", err)
	}
	select {
	case <-client.Done():
	case <-time.After(time.Second):
		t.Fatal("stalled SOCKS client was not marked failed")
	}

	started := time.Now()
	for index := 0; index < 32; index++ {
		err := notifySocksClient(client, &sliverpb.SocksData{TunnelID: uint64(index), CloseConn: true}, timeout)
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("notification %d after stream failure error = %v, want context.DeadlineExceeded", index, err)
		}
	}
	if elapsed := time.Since(started); elapsed > 250*time.Millisecond {
		t.Fatalf("notifications after stream failure took %s, want immediate rejection", elapsed)
	}
	if calls := sender.calls.Load(); calls != 1 {
		t.Fatalf("underlying stalled sender calls = %d, want exactly 1", calls)
	}
	close(sender.release)
}

func TestSocksClientDataSendTimeoutStopsWorker(t *testing.T) {
	connection := core.NewImplantConnection("blocked-client-data", "test")
	connection.Send = make(chan *sliverpb.Envelope, 1)
	session := core.NewSession(connection)
	core.Sessions.Add(session)
	tunnel, err := core.SocksTunnels.Create(session.ID)
	if err != nil {
		t.Fatalf("create SOCKS tunnel: %v", err)
	}
	sender := &blockingCloseSocksSender{started: make(chan struct{}), release: make(chan struct{})}
	client := core.NewSocksClient(sender)
	owned, newlyBound := tunnel.BindClient(client)
	if !owned || !newlyBound {
		t.Fatalf("bind client = owned:%v new:%v", owned, newlyBound)
	}
	t.Cleanup(func() {
		select {
		case <-sender.release:
		default:
			close(sender.release)
		}
		core.SocksTunnels.CloseIf(tunnel)
		core.Sessions.Remove(session.ID)
		connection.Close()
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	reported := make(chan error, 1)
	workerDone := make(chan struct{})
	go func() {
		(&Server{}).sendSocksDataToClientWithTimeout(ctx, tunnel, client, func(err error) {
			reported <- err
		}, 25*time.Millisecond)
		close(workerDone)
	}()
	if err := tunnel.ProcessDataFromImplant(&sliverpb.SocksData{TunnelID: tunnel.ID, Sequence: 0, Data: []byte("blocked")}); err != nil {
		t.Fatalf("deliver SOCKS client data: %v", err)
	}
	select {
	case <-sender.started:
	case <-time.After(time.Second):
		t.Fatal("SOCKS client data send did not start")
	}
	select {
	case err := <-reported:
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("blocked SOCKS client data error = %v, want context.DeadlineExceeded", err)
		}
	case <-time.After(time.Second):
		t.Fatal("blocked SOCKS client data send did not time out")
	}
	select {
	case <-workerDone:
	case <-time.After(time.Second):
		t.Fatal("SOCKS client data worker did not stop after send timeout")
	}
	close(sender.release)
}

//nolint:gocyclo // The scenario asserts the complete cross-direction close ordering.
func TestCloseSocksQueuesSequencedImplantCloseBeforeClientNotification(t *testing.T) {
	connection := core.NewImplantConnection("close-order", "test")
	connection.Send = make(chan *sliverpb.Envelope)
	session := core.NewSession(connection)
	core.Sessions.Add(session)
	tunnel, err := core.SocksTunnels.Create(session.ID)
	if err != nil {
		t.Fatalf("create SOCKS tunnel: %v", err)
	}
	sender := &blockingCloseSocksSender{started: make(chan struct{}), release: make(chan struct{})}
	owned, newlyBound := tunnel.BindClient(core.NewSocksClient(sender))
	if !owned || !newlyBound {
		t.Fatalf("bind client = owned:%v new:%v", owned, newlyBound)
	}
	atomic.StoreUint64(&tunnel.ToImplantSequence, 7)
	t.Cleanup(func() {
		select {
		case <-sender.release:
		default:
			close(sender.release)
		}
		core.SocksTunnels.CloseIf(tunnel)
		core.Sessions.Remove(session.ID)
		connection.Close()
	})

	closed := make(chan struct{})
	go func() {
		(&Server{}).closeSocksTunnel(tunnel)
		close(closed)
	}()
	select {
	case <-sender.started:
		t.Fatal("client close notification started before implant close was queued")
	case <-time.After(25 * time.Millisecond):
	}

	var envelope *sliverpb.Envelope
	select {
	case envelope = <-connection.Send:
	case <-time.After(time.Second):
		t.Fatal("implant close was not queued")
	}
	closeFrame := &sliverpb.SocksData{}
	if err := proto.Unmarshal(envelope.Data, closeFrame); err != nil {
		t.Fatalf("decode implant close: %v", err)
	}
	if !closeFrame.CloseConn || closeFrame.TunnelID != tunnel.ID || closeFrame.Sequence != 7 {
		t.Fatalf("implant close frame = %+v, want tunnel %d sequence 7", closeFrame, tunnel.ID)
	}
	select {
	case <-sender.started:
	case <-time.After(time.Second):
		t.Fatal("client notification did not start after implant close was queued")
	}
	close(sender.release)
	select {
	case <-closed:
	case <-time.After(time.Second):
		t.Fatal("SOCKS close did not finish after client notification completed")
	}
}

func TestNotifySocksClientIsBounded(t *testing.T) {
	sender := &blockingCloseSocksSender{started: make(chan struct{}), release: make(chan struct{})}
	const timeout = 25 * time.Millisecond
	started := time.Now()
	err := notifySocksClient(core.NewSocksClient(sender), &sliverpb.SocksData{CloseConn: true}, timeout)
	elapsed := time.Since(started)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("bounded client notification error = %v, want %v", err, context.DeadlineExceeded)
	}
	if elapsed < timeout || elapsed > 500*time.Millisecond {
		t.Fatalf("bounded client notification elapsed = %s, want [%s,500ms]", elapsed, timeout)
	}
	close(sender.release)
}

func createSocksTunnelForRPCStreamTest(t *testing.T, server *Server, session *core.Session, capabilities uint64) *core.TcpTunnel {
	t.Helper()
	created, err := server.CreateSocks(context.Background(), &sliverpb.Socks{
		SessionID:    session.ID,
		Capabilities: capabilities,
	})
	if err != nil {
		t.Fatalf("create SOCKS tunnel: %v", err)
	}
	tunnel := core.SocksTunnels.Get(created.TunnelID)
	if tunnel == nil {
		t.Fatal("created SOCKS tunnel was not registered")
	}
	return tunnel
}

func requireRPCSocksImplantPayload(t *testing.T, connection *core.ImplantConnection, tunnelID uint64, want string) {
	t.Helper()
	deadline := time.NewTimer(time.Second)
	defer deadline.Stop()
	for {
		select {
		case envelope := <-connection.Send:
			frame := &sliverpb.SocksData{}
			if err := proto.Unmarshal(envelope.Data, frame); err != nil {
				t.Fatalf("decode implant SOCKS frame: %v", err)
			}
			if frame.TunnelID != tunnelID {
				continue
			}
			if frame.CloseConn || string(frame.Data) != want {
				t.Fatalf("implant SOCKS payload = %+v, want %q", frame, want)
			}
			return
		case <-deadline.C:
			t.Fatalf("implant did not receive tunnel %d payload %q", tunnelID, want)
		}
	}
}

func receiveRPCSocksClientFrame(t *testing.T, stream *controllableSocksProxyServer, tunnelID uint64) *sliverpb.SocksData {
	t.Helper()
	deadline := time.NewTimer(time.Second)
	defer deadline.Stop()
	for {
		select {
		case frame := <-stream.sent:
			if frame.TunnelID == tunnelID {
				return frame
			}
		case <-deadline.C:
			t.Fatalf("client did not receive SOCKS frame for tunnel %d", tunnelID)
		}
	}
}

func receiveLegacyTerminalWaitGeneration(t *testing.T, waitStarted <-chan uint64) uint64 {
	t.Helper()
	select {
	case generation := <-waitStarted:
		return generation
	case <-time.After(time.Second):
		t.Fatal("legacy terminal actor did not begin its generation wait")
		return 0
	}
}

func requireRPCSocksSiblingAlive(
	t *testing.T,
	session *core.Session,
	connection *core.ImplantConnection,
	sibling *core.TcpTunnel,
	result <-chan error,
) {
	t.Helper()
	if got := core.SocksTunnels.Get(sibling.ID); got != sibling {
		t.Fatalf("close race removed sibling tunnel: got=%p want=%p", got, sibling)
	}
	if got := core.Sessions.Get(session.ID); got != session {
		t.Fatalf("close race removed sibling session: got=%p want=%p", got, session)
	}
	select {
	case <-connection.Done():
		t.Fatal("close race closed the shared implant connection")
	default:
	}
	select {
	case err := <-result:
		t.Fatalf("close race terminated the shared SOCKS stream: %v", err)
	default:
	}
}

func requireCanceledSocksProxyResult(t *testing.T, result <-chan error, scenario string) {
	t.Helper()
	select {
	case err := <-result:
		if !isCanceledSocksProxyError(err) {
			t.Fatalf("%s SocksProxy cancellation error = %v", scenario, err)
		}
	case <-time.After(time.Second):
		t.Fatalf("%s SocksProxy did not stop", scenario)
	}
}

func waitForRPCSocksCondition(t *testing.T, condition func() bool, failure string) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal(failure)
}
