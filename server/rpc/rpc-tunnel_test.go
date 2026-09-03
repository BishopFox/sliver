package rpc

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"sync"
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
	ctx                  context.Context
	recv                 chan tunnelStreamRecv
	sent                 chan *sliverpb.TunnelData
	sendDelay            time.Duration
	failAfter            int32
	failOnCall           int32
	blockOnCall          int32
	sendStarted          chan struct{}
	releaseSend          <-chan struct{}
	handlerReturned      *atomic.Bool
	sendAfterHandlerExit atomic.Bool
	sendCalls            atomic.Int32
	inSend               atomic.Int32
	overlap              atomic.Bool
}

func (s *testTunnelStream) Context() context.Context {
	if s.ctx != nil {
		return s.ctx
	}
	return context.Background()
}

func (s *testTunnelStream) Recv() (*sliverpb.TunnelData, error) {
	result := <-s.recv
	return result.data, result.err
}

func (s *testTunnelStream) Send(data *sliverpb.TunnelData) error {
	if s.handlerReturned != nil && s.handlerReturned.Load() {
		s.sendAfterHandlerExit.Store(true)
	}
	if s.inSend.Add(1) > 1 {
		s.overlap.Store(true)
	}
	defer s.inSend.Add(-1)
	call := s.sendCalls.Add(1)
	if s.blockOnCall > 0 && call == s.blockOnCall {
		select {
		case s.sendStarted <- struct{}{}:
		default:
		}
		<-s.releaseSend
	}
	if s.sendDelay > 0 {
		time.Sleep(s.sendDelay)
	}
	if (s.failAfter > 0 && call > s.failAfter) || (s.failOnCall > 0 && call == s.failOnCall) {
		return errors.New("test stream send failure")
	}
	s.sent <- proto.Clone(data).(*sliverpb.TunnelData)
	return nil
}

func TestCloseTunnelClaimsOneSchedulerPerGeneration(t *testing.T) {
	session, tunnels := newTunnelStreamTestSession(t, 1)
	tunnel := tunnels[0]
	request := &sliverpb.Tunnel{TunnelID: tunnel.ID, SessionID: session.ID}
	server := &Server{}

	if _, err := server.CloseTunnel(context.Background(), request); err != nil {
		t.Fatalf("first CloseTunnel: %v", err)
	}
	claimedAt := tunnel.LastToImplantTime()
	time.Sleep(time.Millisecond)
	for index := range 100 {
		if _, err := server.CloseTunnel(context.Background(), request); err != nil {
			t.Fatalf("duplicate CloseTunnel %d: %v", index, err)
		}
	}
	if got := tunnel.LastToImplantTime(); !got.Equal(claimedAt) {
		t.Fatalf("duplicate CloseTunnel refreshed generic close grace: got=%v want=%v", got, claimedAt)
	}
	if tunnel.ClaimToImplantClose() {
		t.Fatal("CloseTunnel left client close ownership unclaimed")
	}
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

func TestTunnelDataRejectsNoncanonicalBindFramesWithoutClaimingTunnel(t *testing.T) {
	session, tunnels := newTunnelStreamTestSession(t, 1)
	session.Connection.Send = make(chan *sliverpb.Envelope, 2)
	tunnel := tunnels[0]
	stream := &testTunnelStream{
		recv: make(chan tunnelStreamRecv, 12),
		sent: make(chan *sliverpb.TunnelData, 4),
	}
	var acceptedFrames atomic.Int32
	server := &Server{tunnelDataBeforeBindClient: func(*core.Tunnel, *sliverpb.TunnelData) {
		acceptedFrames.Add(1)
	}}

	invalid := []*sliverpb.TunnelData{
		{TunnelID: tunnel.ID},
		{TunnelID: tunnel.ID, SessionID: "wrong-session"},
		{TunnelID: tunnel.ID, SessionID: session.ID, Data: []byte("not-control")},
		{TunnelID: tunnel.ID, SessionID: session.ID, Closed: true},
		{TunnelID: tunnel.ID, SessionID: session.ID, Sequence: 1},
		{TunnelID: tunnel.ID, SessionID: session.ID, Ack: 1},
		{TunnelID: tunnel.ID, SessionID: session.ID, Resend: true},
		{TunnelID: tunnel.ID, SessionID: session.ID, CreateReverse: true},
		{TunnelID: tunnel.ID, SessionID: session.ID, Rportfwd: &sliverpb.RPortfwd{}},
	}
	for _, frame := range invalid {
		stream.recv <- tunnelStreamRecv{data: frame}
	}
	stream.recv <- tunnelStreamRecv{data: &sliverpb.TunnelData{
		TunnelID:  tunnel.ID,
		SessionID: session.ID,
	}}

	handlerDone := make(chan error, 1)
	go func() { handlerDone <- server.TunnelData(stream) }()
	bind := waitForTunnelStreamSend(t, stream.sent)
	if bind.TunnelID != tunnel.ID || bind.SessionID != session.ID || bind.Closed {
		t.Fatalf("bind acknowledgement = %+v", bind)
	}
	if got := acceptedFrames.Load(); got != 1 {
		t.Fatalf("frames admitted to bind path = %d, want only the canonical frame", got)
	}
	if !tunnel.IsClient(stream) {
		t.Fatal("canonical frame did not claim tunnel after malformed attempts")
	}
	select {
	case envelope := <-session.Connection.Send:
		t.Fatalf("malformed bind reached implant: %+v", envelope)
	default:
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

//nolint:gocyclo // The assertions cover bind, ordered delivery, terminal, and connection isolation.
func TestTunnelDataForwardsSequencedFramesBeforeOvertakingTerminal(t *testing.T) {
	session, tunnels := newTunnelStreamTestSession(t, 1)
	session.Capabilities = sliverpb.CapabilityTunnelTerminalV1
	tunnel := tunnels[0]
	drainImplantCloseEnvelopes(session, 1)
	stream := &testTunnelStream{
		recv: make(chan tunnelStreamRecv, 2),
		sent: make(chan *sliverpb.TunnelData, 4),
	}
	stream.recv <- tunnelStreamRecv{data: &sliverpb.TunnelData{TunnelID: tunnel.ID, SessionID: session.ID}}

	handlerDone := make(chan error, 1)
	go func() { handlerDone <- (&Server{}).TunnelData(stream) }()
	waitForTunnelStreamSend(t, stream.sent)

	ready, err := tunnel.MarkFromImplantTerminal(&sliverpb.TunnelData{
		TunnelID: tunnel.ID,
		Sequence: 2,
		Closed:   true,
	})
	if err != nil {
		t.Fatalf("mark overtaking terminal: %v", err)
	}
	if ready {
		t.Fatal("overtaking terminal was ready before preceding frames")
	}
	core.Tunnels.ArmFromImplantTerminalClose(tunnel)
	if err := tunnel.ProcessDataFromImplant(&sliverpb.TunnelData{
		TunnelID: tunnel.ID,
		Sequence: 1,
		Data:     []byte("second"),
	}); err != nil {
		t.Fatalf("queue second frame: %v", err)
	}
	deliveryDone := make(chan error, 1)
	go func() {
		deliveryDone <- tunnel.ProcessDataFromImplant(&sliverpb.TunnelData{
			TunnelID: tunnel.ID,
			Sequence: 0,
			Data:     []byte("first"),
		})
	}()

	first := waitForTunnelStreamSend(t, stream.sent)
	if first.Closed || string(first.Data) != "first" {
		t.Fatalf("first frame after overtaking terminal = %+v", first)
	}
	second := waitForTunnelStreamSend(t, stream.sent)
	if second.Closed || string(second.Data) != "second" {
		t.Fatalf("second frame after overtaking terminal = %+v", second)
	}
	terminal := waitForTunnelStreamSend(t, stream.sent)
	if !terminal.Closed || len(terminal.Data) != 0 || terminal.TunnelID != tunnel.ID || terminal.SessionID != session.ID {
		t.Fatalf("terminal after sequenced frames = %+v", terminal)
	}
	select {
	case err := <-deliveryDone:
		if err != nil {
			t.Fatalf("deliver preceding frames: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("preceding frame delivery did not complete")
	}
	select {
	case <-tunnel.Done():
	case <-time.After(time.Second):
		t.Fatal("sequenced terminal did not close after successful stream sends")
	}
	select {
	case <-session.Connection.Done():
		t.Fatal("valid sequenced terminal closed the implant connection")
	default:
	}

	stream.recv <- tunnelStreamRecv{err: io.EOF}
	waitForTunnelDataHandler(t, handlerDone)
}

func TestTunnelDataWorkerSendFailureTerminatesBlockedReceive(t *testing.T) {
	session, tunnels := newTunnelStreamTestSession(t, 1)
	drainImplantCloseEnvelopes(session, 1)
	tunnel := tunnels[0]
	stream := &testTunnelStream{
		recv:      make(chan tunnelStreamRecv, 2),
		sent:      make(chan *sliverpb.TunnelData, 4),
		failAfter: 1, // Bind acknowledgement succeeds; first data frame fails.
	}
	t.Cleanup(func() {
		select {
		case stream.recv <- tunnelStreamRecv{err: io.EOF}:
		default:
		}
	})
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
	select {
	case err := <-handlerDone:
		if err == nil {
			t.Fatal("TunnelData returned nil after relay-worker Send failed")
		}
	case <-time.After(time.Second):
		t.Fatal("TunnelData remained blocked in Recv after relay-worker Send failed")
	}
	if got := stream.sendCalls.Load(); got != 2 {
		t.Fatalf("actual stream Send calls = %d, want bind plus one failed data Send", got)
	}
}

func TestTunnelDataWaitsForTerminalSendBeforeReturning(t *testing.T) {
	session, tunnels := newTunnelStreamTestSession(t, 1)
	drainImplantCloseEnvelopes(session, 1)
	tunnel := tunnels[0]
	releaseSend := make(chan struct{}, 1)
	var handlerReturned atomic.Bool
	stream := &testTunnelStream{
		recv:            make(chan tunnelStreamRecv, 2),
		sent:            make(chan *sliverpb.TunnelData, 2),
		blockOnCall:     2,
		sendStarted:     make(chan struct{}, 1),
		releaseSend:     releaseSend,
		handlerReturned: &handlerReturned,
	}
	t.Cleanup(func() {
		select {
		case releaseSend <- struct{}{}:
		default:
		}
	})
	stream.recv <- tunnelStreamRecv{data: &sliverpb.TunnelData{TunnelID: tunnel.ID, SessionID: session.ID}}
	handlerDone := make(chan error, 1)
	go func() {
		err := (&Server{}).TunnelData(stream)
		handlerReturned.Store(true)
		handlerDone <- err
	}()
	bind := waitForTunnelStreamSend(t, stream.sent)
	if bind.Closed || bind.TunnelID != tunnel.ID || bind.SessionID != session.ID {
		t.Fatalf("bind acknowledgement = %+v", bind)
	}

	stream.recv <- tunnelStreamRecv{err: io.EOF}
	select {
	case <-stream.sendStarted:
	case <-time.After(time.Second):
		t.Fatal("terminal Send did not start during TunnelData cleanup")
	}
	select {
	case err := <-handlerDone:
		t.Fatalf("TunnelData returned before terminal Send completed: %v", err)
	default:
	}
	releaseSend <- struct{}{}
	terminal := waitForTunnelStreamSend(t, stream.sent)
	if !terminal.Closed || terminal.TunnelID != tunnel.ID || terminal.SessionID != session.ID {
		t.Fatalf("cleanup terminal = %+v", terminal)
	}
	select {
	case err := <-handlerDone:
		if err != nil {
			t.Fatalf("TunnelData EOF result = %v, want nil", err)
		}
	case <-time.After(time.Second):
		t.Fatal("TunnelData did not return after terminal Send completed")
	}
	if stream.sendAfterHandlerExit.Load() {
		t.Fatal("stream Send ran after TunnelData returned")
	}
}

func TestTunnelDataCleanupTerminalSendFailureReturnsError(t *testing.T) {
	session, tunnels := newTunnelStreamTestSession(t, 1)
	drainImplantCloseEnvelopes(session, 1)
	tunnel := tunnels[0]
	stream := &testTunnelStream{
		recv:       make(chan tunnelStreamRecv, 2),
		sent:       make(chan *sliverpb.TunnelData, 1),
		failOnCall: 2,
	}
	stream.recv <- tunnelStreamRecv{data: &sliverpb.TunnelData{TunnelID: tunnel.ID, SessionID: session.ID}}
	stream.recv <- tunnelStreamRecv{err: io.EOF}

	if err := (&Server{}).TunnelData(stream); err == nil {
		t.Fatal("TunnelData returned nil after cleanup terminal Send failed")
	}
	if got := stream.sendCalls.Load(); got != 2 {
		t.Fatalf("actual stream Send calls = %d, want bind plus one failed terminal Send", got)
	}
}

//nolint:gocyclo // The table exercises all bind/close interleavings in one state-machine test.
func TestTunnelDataBindCloseRaceKeepsSharedStreamAlive(t *testing.T) {
	for _, testCase := range []struct {
		name              string
		closeAfterBindAck bool
	}{
		{name: "before-bind-client"},
		{name: "after-bind-ack", closeAfterBindAck: true},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			survivorSession, survivorTunnels := newTunnelStreamTestSession(t, 1)
			survivor := survivorTunnels[0]
			victimSession, victimTunnels := newTunnelStreamTestSession(t, 1)
			victim := victimTunnels[0]
			survivorSession.Connection.Send = make(chan *sliverpb.Envelope, 4)
			victimSession.Connection.Send = make(chan *sliverpb.Envelope, 2)
			stream := &testTunnelStream{
				recv: make(chan tunnelStreamRecv, 4),
				sent: make(chan *sliverpb.TunnelData, 6),
			}
			hookResult := make(chan error, 1)
			closeVictim := func(receivedTunnel *core.Tunnel, frame *sliverpb.TunnelData) {
				if receivedTunnel != victim {
					return
				}
				if frame.TunnelID != victim.ID || frame.SessionID != victimSession.ID {
					hookResult <- fmt.Errorf("bind frame = %+v", frame)
					return
				}
				if !core.Tunnels.CloseIf(victim) {
					hookResult <- errors.New("bind-race close did not own victim generation")
					return
				}
				hookResult <- nil
			}
			server := &Server{}
			if testCase.closeAfterBindAck {
				server.tunnelDataAfterBindAcknowledgment = closeVictim
			} else {
				server.tunnelDataBeforeBindClient = closeVictim
			}

			stream.recv <- tunnelStreamRecv{data: &sliverpb.TunnelData{
				TunnelID:  survivor.ID,
				SessionID: survivorSession.ID,
			}}
			handlerDone := make(chan error, 1)
			go func() { handlerDone <- server.TunnelData(stream) }()
			survivorBind := waitForTunnelStreamSend(t, stream.sent)
			if survivorBind.TunnelID != survivor.ID || survivorBind.SessionID != survivorSession.ID || survivorBind.Closed {
				t.Fatalf("survivor bind acknowledgement = %+v", survivorBind)
			}

			stream.recv <- tunnelStreamRecv{data: &sliverpb.TunnelData{
				TunnelID:  victim.ID,
				SessionID: victimSession.ID,
			}}
			if testCase.closeAfterBindAck {
				victimBind := waitForTunnelStreamSend(t, stream.sent)
				if victimBind.TunnelID != victim.ID || victimBind.SessionID != victimSession.ID || victimBind.Closed {
					t.Fatalf("victim bind acknowledgement = %+v", victimBind)
				}
			}
			select {
			case err := <-hookResult:
				if err != nil {
					t.Fatal(err)
				}
			case <-time.After(time.Second):
				t.Fatal("bind-race close hook did not run")
			}
			terminal := waitForTunnelStreamSend(t, stream.sent)
			if terminal.TunnelID != victim.ID || terminal.SessionID != victimSession.ID || !terminal.Closed {
				t.Fatalf("bind-race terminal = %+v", terminal)
			}
			victimImplantClose := waitForImplantTunnelEnvelope(t, victimSession.Connection.Send)
			if !victimImplantClose.Closed || victimImplantClose.TunnelID != victim.ID || victimImplantClose.SessionID != victimSession.ID {
				t.Fatalf("victim implant terminal = %+v", victimImplantClose)
			}

			if core.Tunnels.Get(survivor.ID) != survivor || tunnelIsClosed(survivor) {
				t.Fatal("bind race closed unrelated tunnel on shared stream")
			}
			if core.Sessions.Get(survivorSession.ID) != survivorSession || core.Sessions.Get(victimSession.ID) != victimSession {
				t.Fatal("bind race removed an unrelated or victim session")
			}
			for name, connection := range map[string]*core.ImplantConnection{
				"survivor": survivorSession.Connection,
				"victim":   victimSession.Connection,
			} {
				select {
				case <-connection.Done():
					t.Fatalf("bind race closed %s implant connection", name)
				default:
				}
			}

			stream.recv <- tunnelStreamRecv{data: &sliverpb.TunnelData{
				TunnelID:  survivor.ID,
				SessionID: survivorSession.ID,
				Data:      []byte("still-live"),
			}}
			survivorFrame := waitForImplantTunnelEnvelope(t, survivorSession.Connection.Send)
			if survivorFrame.Closed || survivorFrame.TunnelID != survivor.ID || string(survivorFrame.Data) != "still-live" {
				t.Fatalf("survivor implant frame = %+v", survivorFrame)
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
		})
	}
}

//nolint:gocyclo // The test distinguishes a retained racing stream from the bound terminal owner.
func TestTunnelDataClosedRacingBinderCannotConsumeBoundClientTerminal(t *testing.T) {
	session, tunnels := newTunnelStreamTestSession(t, 1)
	tunnel := tunnels[0]
	session.Connection.Send = make(chan *sliverpb.Envelope, 2)
	bound := &testTunnelStream{
		recv: make(chan tunnelStreamRecv, 1),
		sent: make(chan *sliverpb.TunnelData, 2),
	}
	if !tunnel.BindClient(bound) || !tunnel.MarkClientBound(bound) {
		t.Fatal("could not install the original bound client")
	}

	racing := &testTunnelStream{
		recv: make(chan tunnelStreamRecv, 2),
		sent: make(chan *sliverpb.TunnelData, 2),
	}
	hookEntered := make(chan struct{}, 1)
	releaseHook := make(chan struct{})
	var releaseOnce sync.Once
	release := func() { releaseOnce.Do(func() { close(releaseHook) }) }
	t.Cleanup(release)
	server := &Server{tunnelDataBeforeBindClient: func(receivedTunnel *core.Tunnel, frame *sliverpb.TunnelData) {
		if receivedTunnel != tunnel || frame == nil {
			return
		}
		hookEntered <- struct{}{}
		<-releaseHook
	}}
	racing.recv <- tunnelStreamRecv{data: &sliverpb.TunnelData{
		TunnelID:  tunnel.ID,
		SessionID: session.ID,
	}}
	handlerDone := make(chan error, 1)
	go func() { handlerDone <- server.TunnelData(racing) }()
	select {
	case <-hookEntered:
	case <-time.After(time.Second):
		t.Fatal("racing binder did not retain the tunnel before bind")
	}
	if !core.Tunnels.CloseIf(tunnel) {
		t.Fatal("exact close did not own the bound tunnel generation")
	}
	release()
	implantTerminal := waitForImplantTunnelEnvelope(t, session.Connection.Send)
	if !implantTerminal.Closed || implantTerminal.TunnelID != tunnel.ID {
		t.Fatalf("implant terminal after bind race = %+v", implantTerminal)
	}
	racing.recv <- tunnelStreamRecv{err: io.EOF}
	waitForTunnelDataHandler(t, handlerDone)
	select {
	case terminal := <-racing.sent:
		t.Fatalf("non-owner racing stream consumed client terminal: %+v", terminal)
	default:
	}

	if err := notifyTunnelClosedToClient(tunnel, bound, bound.Send); err != nil {
		t.Fatalf("notify original bound client: %v", err)
	}
	terminal := waitForTunnelStreamSend(t, bound.sent)
	if !terminal.Closed || terminal.TunnelID != tunnel.ID || terminal.SessionID != session.ID {
		t.Fatalf("bound client terminal = %+v", terminal)
	}
	if err := notifyTunnelClosedToClient(tunnel, bound, bound.Send); err != nil {
		t.Fatalf("repeat bound client terminal: %v", err)
	}
	select {
	case duplicate := <-bound.sent:
		t.Fatalf("duplicate bound client terminal = %+v", duplicate)
	default:
	}
	select {
	case <-session.Connection.Done():
		t.Fatal("client terminal ownership race closed the healthy implant connection")
	default:
	}
}

func TestTunnelDataBindAcknowledgmentFailureNotifiesImplant(t *testing.T) {
	session, tunnels := newTunnelStreamTestSession(t, 1)
	tunnel := tunnels[0]
	session.Connection.Send = make(chan *sliverpb.Envelope, 1)
	stream := &testTunnelStream{
		recv:       make(chan tunnelStreamRecv, 1),
		sent:       make(chan *sliverpb.TunnelData, 1),
		failOnCall: 1,
	}
	t.Cleanup(func() {
		select {
		case stream.recv <- tunnelStreamRecv{err: io.EOF}:
		default:
		}
	})
	stream.recv <- tunnelStreamRecv{data: &sliverpb.TunnelData{TunnelID: tunnel.ID, SessionID: session.ID}}

	if err := (&Server{}).TunnelData(stream); err == nil {
		t.Fatal("bind acknowledgement failure returned nil")
	}
	select {
	case <-tunnel.Done():
	default:
		t.Fatal("bind acknowledgement failure left tunnel open")
	}
	implantClose := waitForImplantTunnelEnvelope(t, session.Connection.Send)
	if !implantClose.Closed || implantClose.TunnelID != tunnel.ID || implantClose.SessionID != session.ID {
		t.Fatalf("implant terminal after bind acknowledgement failure = %+v", implantClose)
	}
	select {
	case <-session.Connection.Done():
		t.Fatal("successful implant terminal delivery closed connection")
	default:
	}
}

func TestTunnelDataUsesCreatingConnectionAfterSessionReplacement(t *testing.T) {
	creatingSession, tunnels := newTunnelStreamTestSession(t, 1)
	tunnel := tunnels[0]
	creatingSession.Connection.Send = make(chan *sliverpb.Envelope, 4)
	t.Cleanup(creatingSession.Connection.Close)
	replacement := replaceTunnelStreamTestSession(t, creatingSession)

	stream := &testTunnelStream{
		recv: make(chan tunnelStreamRecv, 3),
		sent: make(chan *sliverpb.TunnelData, 3),
	}
	stream.recv <- tunnelStreamRecv{data: &sliverpb.TunnelData{TunnelID: tunnel.ID, SessionID: creatingSession.ID}}
	handlerDone := make(chan error, 1)
	go func() { handlerDone <- (&Server{}).TunnelData(stream) }()
	waitForTunnelStreamSend(t, stream.sent)

	stream.recv <- tunnelStreamRecv{data: &sliverpb.TunnelData{
		TunnelID:  tunnel.ID,
		SessionID: creatingSession.ID,
		Data:      []byte("creating-connection"),
	}}
	data := waitForExactImplantTunnelEnvelope(t, creatingSession.Connection.Send, replacement.Connection.Send, sliverpb.MsgTunnelData)
	if data.Closed || data.TunnelID != tunnel.ID || data.SessionID != creatingSession.ID || string(data.Data) != "creating-connection" {
		t.Fatalf("to-implant frame after session replacement = %+v", data)
	}

	if !core.Tunnels.CloseIf(tunnel) {
		t.Fatal("close exact tunnel after session replacement failed")
	}
	terminal := waitForExactImplantTunnelEnvelope(t, creatingSession.Connection.Send, replacement.Connection.Send, sliverpb.MsgTunnelClose)
	if !terminal.Closed || terminal.TunnelID != tunnel.ID || terminal.SessionID != creatingSession.ID {
		t.Fatalf("implant terminal after session replacement = %+v", terminal)
	}
	clientTerminal := waitForTunnelStreamSend(t, stream.sent)
	if !clientTerminal.Closed || clientTerminal.TunnelID != tunnel.ID || clientTerminal.SessionID != creatingSession.ID {
		t.Fatalf("client terminal after session replacement = %+v", clientTerminal)
	}
	assertReplacementConnectionUntouched(t, replacement)

	stream.recv <- tunnelStreamRecv{err: io.EOF}
	waitForTunnelDataHandler(t, handlerDone)
	assertReplacementConnectionUntouched(t, replacement)
}

func TestTunnelDataResendAndProtocolFailureUseCreatingConnectionAfterSessionReplacement(t *testing.T) {
	creatingSession, tunnels := newTunnelStreamTestSession(t, 1)
	tunnel := tunnels[0]
	creatingSession.Connection.Send = make(chan *sliverpb.Envelope, 4)
	t.Cleanup(creatingSession.Connection.Close)
	stream := &testTunnelStream{
		recv: make(chan tunnelStreamRecv, 3),
		sent: make(chan *sliverpb.TunnelData, 3),
	}
	stream.recv <- tunnelStreamRecv{data: &sliverpb.TunnelData{TunnelID: tunnel.ID, SessionID: creatingSession.ID}}
	handlerDone := make(chan error, 1)
	go func() { handlerDone <- (&Server{}).TunnelData(stream) }()
	waitForTunnelStreamSend(t, stream.sent)

	replacement := replaceTunnelStreamTestSession(t, creatingSession)
	stream.recv <- tunnelStreamRecv{data: &sliverpb.TunnelData{
		TunnelID:  tunnel.ID,
		SessionID: creatingSession.ID,
		Data:      []byte("cached-for-resend"),
	}}
	original := waitForExactImplantTunnelEnvelope(t, creatingSession.Connection.Send, replacement.Connection.Send, sliverpb.MsgTunnelData)
	if original.Closed || original.Sequence != 0 || string(original.Data) != "cached-for-resend" {
		t.Fatalf("original to-implant frame = %+v", original)
	}

	requireImplantTunnelDelivery(t, tunnel, &sliverpb.TunnelData{TunnelID: tunnel.ID, Resend: true, Ack: 0})
	resent := waitForExactImplantTunnelEnvelope(t, creatingSession.Connection.Send, replacement.Connection.Send, sliverpb.MsgTunnelData)
	if !proto.Equal(resent, original) {
		t.Fatalf("resent to-implant frame = %+v, want %+v", resent, original)
	}

	requireImplantTunnelDelivery(t, tunnel, &sliverpb.TunnelData{
		TunnelID: tunnel.ID,
		Sequence: 0,
		Ack:      2,
		Data:     []byte("invalid-ack"),
	})
	select {
	case <-creatingSession.Connection.Done():
	case <-time.After(time.Second):
		t.Fatal("protocol failure did not close the creating implant connection")
	}
	select {
	case <-tunnel.Done():
	case <-time.After(time.Second):
		t.Fatal("protocol failure did not close the exact tunnel")
	}
	clientTerminal := waitForTunnelStreamSend(t, stream.sent)
	if !clientTerminal.Closed || clientTerminal.TunnelID != tunnel.ID || clientTerminal.SessionID != creatingSession.ID {
		t.Fatalf("client terminal after protocol failure = %+v", clientTerminal)
	}
	assertReplacementConnectionUntouched(t, replacement)

	stream.recv <- tunnelStreamRecv{err: io.EOF}
	waitForTunnelDataHandler(t, handlerDone)
	assertReplacementConnectionUntouched(t, replacement)
}

//nolint:gocyclo // The test keeps the selected-data close interleaving deterministic.
func TestTunnelDataExternalCloseAfterToImplantReceiveSendsTerminal(t *testing.T) {
	session, tunnels := newTunnelStreamTestSession(t, 1)
	tunnel := tunnels[0]
	session.Connection.Send = make(chan *sliverpb.Envelope, 2)
	stream := &testTunnelStream{
		recv: make(chan tunnelStreamRecv, 3),
		sent: make(chan *sliverpb.TunnelData, 3),
	}
	hookResult := make(chan error, 1)
	server := &Server{tunnelDataBeforeNextToImplant: func(receivedTunnel *core.Tunnel, data []byte) {
		if receivedTunnel != tunnel || string(data) != "in-flight" {
			hookResult <- fmt.Errorf("pre-send tunnel/data = %p/%q, want %p/%q", receivedTunnel, data, tunnel, "in-flight")
			return
		}
		if !core.Tunnels.CloseIf(tunnel) {
			hookResult <- errors.New("external close did not own to-implant tunnel generation")
			return
		}
		hookResult <- nil
	}}
	stream.recv <- tunnelStreamRecv{data: &sliverpb.TunnelData{TunnelID: tunnel.ID, SessionID: session.ID}}
	handlerDone := make(chan error, 1)
	go func() { handlerDone <- server.TunnelData(stream) }()
	waitForTunnelStreamSend(t, stream.sent)

	stream.recv <- tunnelStreamRecv{data: &sliverpb.TunnelData{
		TunnelID:  tunnel.ID,
		SessionID: session.ID,
		Data:      []byte("in-flight"),
	}}
	select {
	case err := <-hookResult:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("to-implant close race hook did not run")
	}
	implantClose := waitForImplantTunnelEnvelope(t, session.Connection.Send)
	if !implantClose.Closed || implantClose.TunnelID != tunnel.ID || implantClose.SessionID != session.ID {
		t.Fatalf("implant terminal after selected-data close = %+v", implantClose)
	}
	clientClose := waitForTunnelStreamSend(t, stream.sent)
	if !clientClose.Closed || clientClose.TunnelID != tunnel.ID || clientClose.SessionID != session.ID {
		t.Fatalf("client terminal after selected-data close = %+v", clientClose)
	}
	select {
	case <-session.Connection.Done():
		t.Fatal("successful selected-data terminal delivery closed implant connection")
	default:
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
}

//nolint:gocyclo // The test keeps the final-frame quiesce and terminal ordering deterministic.
func TestTunnelDataQuiesceSendsFinalImplantFrameBeforeCapableTerminal(t *testing.T) {
	connection := core.NewImplantConnection("mtls", "quiesced-final-frame")
	connection.Send = make(chan *sliverpb.Envelope)
	session := core.NewSession(connection)
	session.Capabilities = sliverpb.CapabilityTunnelTerminalV1
	core.Sessions.Add(session)
	t.Cleanup(func() {
		core.Sessions.RemoveIf(session)
		connection.Close()
	})
	tunnel, err := core.Tunnels.Create(session.ID)
	if err != nil {
		t.Fatalf("create capable tunnel: %v", err)
	}
	t.Cleanup(func() { core.Tunnels.CloseIf(tunnel) })

	stream := &testTunnelStream{
		recv: make(chan tunnelStreamRecv, 3),
		sent: make(chan *sliverpb.TunnelData, 3),
	}
	forwardingStarted := make(chan struct{}, 1)
	server := &Server{tunnelDataBeforeNextToImplant: func(receivedTunnel *core.Tunnel, data []byte) {
		if receivedTunnel == tunnel && string(data) == "final" {
			forwardingStarted <- struct{}{}
		}
	}}
	stream.recv <- tunnelStreamRecv{data: &sliverpb.TunnelData{
		TunnelID:  tunnel.ID,
		SessionID: session.ID,
	}}
	handlerDone := make(chan error, 1)
	go func() { handlerDone <- server.TunnelData(stream) }()
	waitForTunnelStreamSend(t, stream.sent)

	stream.recv <- tunnelStreamRecv{data: &sliverpb.TunnelData{
		TunnelID:  tunnel.ID,
		SessionID: session.ID,
		Data:      []byte("final"),
	}}
	select {
	case <-forwardingStarted:
	case <-time.After(time.Second):
		t.Fatal("final payload did not reach the implant forwarding worker")
	}

	closeDone := make(chan struct{})
	go func() {
		tunnel.QuiesceDataToImplant()
		core.Tunnels.CloseIf(tunnel)
		close(closeDone)
	}()
	select {
	case <-closeDone:
		t.Fatal("graceful close completed while the final implant send was blocked")
	case <-time.After(25 * time.Millisecond):
	}

	finalFrame := waitForImplantTunnelEnvelope(t, connection.Send)
	if finalFrame.Closed || finalFrame.Sequence != 0 || string(finalFrame.Data) != "final" {
		t.Fatalf("first implant frame = %+v, want final data at sequence 0", finalFrame)
	}
	select {
	case <-closeDone:
	case <-time.After(time.Second):
		t.Fatal("graceful close did not resume after the final implant send")
	}
	terminal := waitForImplantTunnelEnvelope(t, connection.Send)
	if !terminal.Closed || terminal.Sequence != 1 || len(terminal.Data) != 0 {
		t.Fatalf("implant terminal = %+v, want exclusive sequence 1 after final data", terminal)
	}
	clientTerminal := waitForTunnelStreamSend(t, stream.sent)
	if !clientTerminal.Closed || clientTerminal.TunnelID != tunnel.ID || clientTerminal.SessionID != session.ID {
		t.Fatalf("client terminal after quiesced close = %+v", clientTerminal)
	}

	stream.recv <- tunnelStreamRecv{err: io.EOF}
	waitForTunnelDataHandler(t, handlerDone)
}

//nolint:gocyclo // The assertions cover the failed data send and both peer lifecycles.
func TestTunnelDataFailedImplantSendDoesNotInflateCapableTerminal(t *testing.T) {
	connection := core.NewImplantConnection("mtls", "failed-data-send")
	connection.Send = make(chan *sliverpb.Envelope, 2)
	session := core.NewSession(connection)
	session.Capabilities = sliverpb.CapabilityTunnelTerminalV1
	core.Sessions.Add(session)
	t.Cleanup(func() {
		core.Sessions.RemoveIf(session)
		connection.Close()
	})
	tunnel, err := core.Tunnels.Create(session.ID)
	if err != nil {
		t.Fatalf("create capable tunnel: %v", err)
	}
	t.Cleanup(func() { core.Tunnels.CloseIf(tunnel) })

	dataSendAttempted := make(chan struct{}, 1)
	server := &Server{tunnelDataSendToImplant: func(_ *core.ImplantConnection, envelope *sliverpb.Envelope, _ <-chan struct{}, _ time.Duration) error {
		if envelope == nil || envelope.Type != sliverpb.MsgTunnelData {
			return fmt.Errorf("unexpected injected envelope: %+v", envelope)
		}
		dataSendAttempted <- struct{}{}
		return errors.New("injected data send failure")
	}}
	stream := &testTunnelStream{
		recv: make(chan tunnelStreamRecv, 3),
		sent: make(chan *sliverpb.TunnelData, 3),
	}
	stream.recv <- tunnelStreamRecv{data: &sliverpb.TunnelData{
		TunnelID:  tunnel.ID,
		SessionID: session.ID,
	}}
	handlerDone := make(chan error, 1)
	go func() { handlerDone <- server.TunnelData(stream) }()
	waitForTunnelStreamSend(t, stream.sent)

	stream.recv <- tunnelStreamRecv{data: &sliverpb.TunnelData{
		TunnelID:  tunnel.ID,
		SessionID: session.ID,
		Data:      []byte("never enqueued"),
	}}
	select {
	case <-dataSendAttempted:
	case <-time.After(time.Second):
		t.Fatal("implant data send was not attempted")
	}
	terminal := waitForImplantTunnelEnvelope(t, connection.Send)
	if !terminal.Closed || terminal.Sequence != 0 || len(terminal.Data) != 0 {
		t.Fatalf("terminal after failed data send = %+v, want empty sequence 0", terminal)
	}
	clientTerminal := waitForTunnelStreamSend(t, stream.sent)
	if !clientTerminal.Closed || clientTerminal.TunnelID != tunnel.ID || clientTerminal.SessionID != session.ID {
		t.Fatalf("client terminal after failed data send = %+v", clientTerminal)
	}
	select {
	case <-connection.Done():
		t.Fatal("finite data-send failure closed the healthy implant connection")
	default:
	}
	if current := core.Sessions.Get(session.ID); current != session {
		t.Fatalf("session after finite data-send failure = %p, want %p", current, session)
	}

	stream.recv <- tunnelStreamRecv{err: io.EOF}
	waitForTunnelDataHandler(t, handlerDone)
}

//nolint:gocyclo // The test freezes the successful-send/completion interval and terminal owner.
func TestTunnelDataTerminalWaitsForSuccessfulSendCompletionPrefix(t *testing.T) {
	connection := core.NewImplantConnection("mtls", "successful-send-prefix")
	connection.Send = make(chan *sliverpb.Envelope, 2)
	session := core.NewSession(connection)
	session.Capabilities = sliverpb.CapabilityTunnelTerminalV1
	core.Sessions.Add(session)
	t.Cleanup(func() {
		core.Sessions.RemoveIf(session)
		connection.Close()
	})
	tunnel, err := core.Tunnels.Create(session.ID)
	if err != nil {
		t.Fatalf("create capable tunnel: %v", err)
	}
	t.Cleanup(func() { core.Tunnels.CloseIf(tunnel) })

	sendCompleted := make(chan struct{}, 1)
	releaseCompletion := make(chan struct{})
	var releaseOnce sync.Once
	release := func() { releaseOnce.Do(func() { close(releaseCompletion) }) }
	t.Cleanup(release)
	server := &Server{
		tunnelDataSendToImplant: func(_ *core.ImplantConnection, envelope *sliverpb.Envelope, _ <-chan struct{}, _ time.Duration) error {
			if envelope == nil || envelope.Type != sliverpb.MsgTunnelData {
				return fmt.Errorf("unexpected injected envelope: %+v", envelope)
			}
			return nil
		},
		tunnelDataAfterImplantSend: func(receivedTunnel *core.Tunnel, frame *sliverpb.TunnelData) {
			if receivedTunnel != tunnel || frame.Sequence != 0 || string(frame.Data) != "successfully enqueued" {
				t.Errorf("post-send tunnel/frame = %p/%+v", receivedTunnel, frame)
			}
			sendCompleted <- struct{}{}
			<-releaseCompletion
		},
	}
	stream := &testTunnelStream{
		recv: make(chan tunnelStreamRecv, 3),
		sent: make(chan *sliverpb.TunnelData, 3),
	}
	stream.recv <- tunnelStreamRecv{data: &sliverpb.TunnelData{
		TunnelID:  tunnel.ID,
		SessionID: session.ID,
	}}
	handlerDone := make(chan error, 1)
	go func() { handlerDone <- server.TunnelData(stream) }()
	waitForTunnelStreamSend(t, stream.sent)
	stream.recv <- tunnelStreamRecv{data: &sliverpb.TunnelData{
		TunnelID:  tunnel.ID,
		SessionID: session.ID,
		Data:      []byte("successfully enqueued"),
	}}
	select {
	case <-sendCompleted:
	case <-time.After(time.Second):
		t.Fatal("data send did not reach the pre-completion interval")
	}

	if !core.Tunnels.CloseIf(tunnel) {
		t.Fatal("failed to close exact tunnel during pre-completion interval")
	}
	terminalDone := make(chan error, 1)
	go func() { terminalDone <- sendTunnelCloseToImplant(tunnel, time.Second) }()
	select {
	case err := <-terminalDone:
		t.Fatalf("terminal completed before successful-send prefix publication: %v", err)
	case terminal := <-connection.Send:
		t.Fatalf("terminal enqueued before successful-send prefix publication: %+v", terminal)
	case <-time.After(25 * time.Millisecond):
	}

	release()
	select {
	case err := <-terminalDone:
		if err != nil {
			t.Fatalf("send terminal after prefix completion: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("terminal did not resume after successful-send prefix publication")
	}
	terminal := waitForImplantTunnelEnvelope(t, connection.Send)
	if !terminal.Closed || terminal.Sequence != 1 || len(terminal.Data) != 0 {
		t.Fatalf("terminal after successful send = %+v, want exclusive sequence 1", terminal)
	}
	clientTerminal := waitForTunnelStreamSend(t, stream.sent)
	if !clientTerminal.Closed || clientTerminal.TunnelID != tunnel.ID {
		t.Fatalf("client terminal after successful-send race = %+v", clientTerminal)
	}
	select {
	case <-connection.Done():
		t.Fatal("successful-send completion race closed the healthy implant connection")
	default:
	}

	stream.recv <- tunnelStreamRecv{err: io.EOF}
	waitForTunnelDataHandler(t, handlerDone)
}

//nolint:gocyclo // The test drives the retained-frame branch against both bound workers.
func TestTunnelDataCloseRacePublishesEachPeerTerminalOnce(t *testing.T) {
	session, tunnels := newTunnelStreamTestSession(t, 1)
	tunnel := tunnels[0]
	session.Connection.Send = make(chan *sliverpb.Envelope, 4)
	stream := &testTunnelStream{
		recv: make(chan tunnelStreamRecv, 3),
		sent: make(chan *sliverpb.TunnelData, 4),
	}
	closeHookDone := make(chan struct{}, 1)
	server := &Server{tunnelDataBeforeBindClient: func(receivedTunnel *core.Tunnel, frame *sliverpb.TunnelData) {
		if receivedTunnel != tunnel || frame == nil || len(frame.Data) == 0 {
			return
		}
		if !core.Tunnels.CloseIf(tunnel) {
			t.Errorf("close-race hook did not own tunnel generation")
		}
		closeHookDone <- struct{}{}
	}}
	stream.recv <- tunnelStreamRecv{data: &sliverpb.TunnelData{
		TunnelID:  tunnel.ID,
		SessionID: session.ID,
	}}
	handlerDone := make(chan error, 1)
	go func() { handlerDone <- server.TunnelData(stream) }()
	waitForTunnelStreamSend(t, stream.sent)

	stream.recv <- tunnelStreamRecv{data: &sliverpb.TunnelData{
		TunnelID:  tunnel.ID,
		SessionID: session.ID,
		Data:      []byte("racing frame"),
	}}
	select {
	case <-closeHookDone:
	case <-time.After(time.Second):
		t.Fatal("retained-frame close hook did not run")
	}
	implantTerminal := waitForImplantTunnelEnvelope(t, session.Connection.Send)
	if !implantTerminal.Closed || implantTerminal.TunnelID != tunnel.ID {
		t.Fatalf("implant terminal = %+v", implantTerminal)
	}
	clientTerminal := waitForTunnelStreamSend(t, stream.sent)
	if !clientTerminal.Closed || clientTerminal.TunnelID != tunnel.ID {
		t.Fatalf("client terminal = %+v", clientTerminal)
	}

	stream.recv <- tunnelStreamRecv{err: io.EOF}
	waitForTunnelDataHandler(t, handlerDone)
	select {
	case duplicate := <-session.Connection.Send:
		t.Fatalf("duplicate implant terminal = %+v", duplicate)
	default:
	}
	select {
	case duplicate := <-stream.sent:
		t.Fatalf("duplicate client terminal = %+v", duplicate)
	default:
	}
	select {
	case <-session.Connection.Done():
		t.Fatal("duplicate terminal race closed the healthy implant connection")
	default:
	}
}

func TestSendTunnelCloseToImplantFailsUndeliverableConnectionClosed(t *testing.T) {
	for _, testCase := range []struct {
		name    string
		prepare func(*core.ImplantConnection)
	}{
		{name: "invalid-queue", prepare: func(connection *core.ImplantConnection) { connection.Send = nil }},
		{name: "send-timeout"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			session, tunnels := newTunnelStreamTestSession(t, 1)
			connection := session.Connection
			tunnel := tunnels[0]
			t.Cleanup(connection.Close)
			if testCase.prepare != nil {
				testCase.prepare(connection)
			}
			err := sendTunnelCloseToImplant(tunnel, 25*time.Millisecond)
			if err == nil || errors.Is(err, core.ErrImplantConnectionClosed) {
				t.Fatalf("undeliverable terminal error = %v", err)
			}
			select {
			case <-connection.Done():
			case <-time.After(time.Second):
				t.Fatal("undeliverable implant terminal left connection live")
			}
		})
	}
}

func TestSendTunnelCloseToImplantUsesExclusiveSequenceForCapableImplant(t *testing.T) {
	for _, testCase := range []struct {
		name         string
		capabilities uint64
		wantSequence uint64
	}{
		{name: "legacy implant", capabilities: 0, wantSequence: 0},
		{name: "terminal capable implant", capabilities: sliverpb.CapabilityTunnelTerminalV1, wantSequence: 2},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			connection := core.NewImplantConnection("mtls", "terminal-sequence")
			connection.Send = make(chan *sliverpb.Envelope, 2)
			session := core.NewSession(connection)
			session.Capabilities = testCase.capabilities
			core.Sessions.Add(session)
			t.Cleanup(func() {
				core.Sessions.RemoveIf(session)
				connection.Close()
			})
			tunnel, err := core.Tunnels.Create(session.ID)
			if err != nil {
				t.Fatalf("create tunnel: %v", err)
			}
			t.Cleanup(func() { core.Tunnels.CloseIf(tunnel) })

			for _, payload := range []string{"first", "final"} {
				frame, err := tunnel.NextDataToImplant([]byte(payload))
				if err != nil {
					t.Fatalf("assign implant data sequence: %v", err)
				}
				if err := tunnel.CompleteDataToImplantForward(frame.Sequence); err != nil {
					t.Fatalf("complete implant data sequence %d: %v", frame.Sequence, err)
				}
			}
			if err := sendTunnelCloseToImplant(tunnel, time.Second); err != nil {
				t.Fatalf("send implant terminal: %v", err)
			}
			terminal := waitForImplantTunnelEnvelope(t, connection.Send)
			if !terminal.Closed || len(terminal.Data) != 0 || terminal.Sequence != testCase.wantSequence {
				t.Fatalf("implant terminal = %+v, want closed sequence %d", terminal, testCase.wantSequence)
			}
		})
	}
}

func TestTunnelDataSessionRemovalNotifiesBoundClient(t *testing.T) {
	session, tunnels := newTunnelStreamTestSession(t, 1)
	tunnel := tunnels[0]
	drainImplantCloseEnvelopes(session, 1)
	stream := &testTunnelStream{
		recv: make(chan tunnelStreamRecv, 2),
		sent: make(chan *sliverpb.TunnelData, 2),
	}
	stream.recv <- tunnelStreamRecv{data: &sliverpb.TunnelData{TunnelID: tunnel.ID, SessionID: session.ID}}
	handlerDone := make(chan error, 1)
	go func() { handlerDone <- (&Server{}).TunnelData(stream) }()
	bind := waitForTunnelStreamSend(t, stream.sent)
	if bind.TunnelID != tunnel.ID || bind.SessionID != session.ID || bind.Closed {
		t.Fatalf("bind acknowledgement = %+v", bind)
	}

	core.Sessions.Remove(session.ID)
	select {
	case <-tunnel.Done():
	case <-time.After(time.Second):
		t.Fatal("session removal did not close bound generic tunnel")
	}
	if core.Tunnels.Get(tunnel.ID) != nil {
		t.Fatal("session removal retained bound generic tunnel")
	}
	terminal := waitForTunnelStreamSend(t, stream.sent)
	if terminal.TunnelID != tunnel.ID || terminal.SessionID != session.ID || !terminal.Closed {
		t.Fatalf("session-removal terminal frame = %+v", terminal)
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
}

func TestTunnelDataInFlightCloseNotifiesBoundClient(t *testing.T) {
	session, tunnels := newTunnelStreamTestSession(t, 1)
	tunnel := tunnels[0]
	drainImplantCloseEnvelopes(session, 1)
	stream := &testTunnelStream{
		recv: make(chan tunnelStreamRecv, 2),
		sent: make(chan *sliverpb.TunnelData, 3),
	}
	stream.recv <- tunnelStreamRecv{data: &sliverpb.TunnelData{TunnelID: tunnel.ID, SessionID: session.ID}}
	handlerDone := make(chan error, 1)
	go func() { handlerDone <- (&Server{}).TunnelData(stream) }()
	waitForTunnelStreamSend(t, stream.sent)

	// A future ACK is a protocol violation that deterministically closes the
	// tunnel from the ready-frame path. That exit must still notify the already-
	// bound client with a terminal frame.
	deliveryDone := make(chan bool, 1)
	go func() {
		deliveryDone <- tunnel.SendDataFromImplant(&sliverpb.TunnelData{
			TunnelID: tunnel.ID,
			Sequence: 0,
			Ack:      1,
			Data:     []byte("in-flight"),
		})
	}()
	select {
	case delivered := <-deliveryDone:
		if !delivered {
			t.Fatal("in-flight implant frame was not accepted before close")
		}
	case <-time.After(time.Second):
		t.Fatal("in-flight implant frame delivery blocked")
	}
	select {
	case <-tunnel.Done():
	case <-time.After(time.Second):
		t.Fatal("invalid in-flight ACK did not close tunnel")
	}
	terminal := waitForTunnelStreamSend(t, stream.sent)
	if terminal.TunnelID != tunnel.ID || terminal.SessionID != session.ID || !terminal.Closed {
		t.Fatalf("in-flight-close terminal frame = %+v", terminal)
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
}

//nolint:gocyclo // The test asserts the complete receive/close/session race lifecycle.
func TestTunnelDataExternalCloseAfterReceiveDoesNotCloseSession(t *testing.T) {
	session, tunnels := newTunnelStreamTestSession(t, 1)
	tunnel := tunnels[0]
	drainImplantCloseEnvelopes(session, 1)
	stream := &testTunnelStream{
		recv: make(chan tunnelStreamRecv, 2),
		sent: make(chan *sliverpb.TunnelData, 3),
	}
	hookResult := make(chan error, 1)
	server := &Server{tunnelDataBeforeImplantControl: func(receivedTunnel *core.Tunnel, frame *sliverpb.TunnelData) {
		if receivedTunnel != tunnel {
			hookResult <- fmt.Errorf("pre-ack tunnel = %p, want %p", receivedTunnel, tunnel)
			return
		}
		if frame.Ack != 0 || frame.Sequence != 0 || string(frame.Data) != "in-flight" {
			hookResult <- fmt.Errorf("pre-ack frame = %+v", frame)
			return
		}
		if !core.Tunnels.CloseIf(tunnel) {
			hookResult <- errors.New("external close did not own the tunnel generation")
			return
		}
		hookResult <- nil
	}}

	stream.recv <- tunnelStreamRecv{data: &sliverpb.TunnelData{TunnelID: tunnel.ID, SessionID: session.ID}}
	handlerDone := make(chan error, 1)
	go func() { handlerDone <- server.TunnelData(stream) }()
	waitForTunnelStreamSend(t, stream.sent)

	deliveryDone := make(chan bool, 1)
	go func() {
		deliveryDone <- tunnel.SendDataFromImplant(&sliverpb.TunnelData{
			TunnelID: tunnel.ID,
			Sequence: 0,
			Data:     []byte("in-flight"),
		})
	}()
	select {
	case delivered := <-deliveryDone:
		if !delivered {
			t.Fatal("in-flight implant frame was not accepted before external close")
		}
	case <-time.After(time.Second):
		t.Fatal("in-flight implant frame delivery blocked")
	}
	select {
	case err := <-hookResult:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("pre-ack external close hook did not run")
	}

	terminal := waitForTunnelStreamSend(t, stream.sent)
	if terminal.TunnelID != tunnel.ID || terminal.SessionID != session.ID || !terminal.Closed {
		t.Fatalf("external-close terminal frame = %+v", terminal)
	}
	select {
	case <-session.Connection.Done():
		t.Fatal("benign external tunnel close tore down the implant connection")
	default:
	}
	if current := core.Sessions.Get(session.ID); current != session {
		t.Fatalf("session after benign external close = %p, want %p", current, session)
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
}

//nolint:gocyclo // The test asserts the complete resend/close/session race lifecycle.
func TestTunnelDataExternalCloseBeforeResendDoesNotCloseSession(t *testing.T) {
	session, tunnels := newTunnelStreamTestSession(t, 1)
	tunnel := tunnels[0]
	drainImplantCloseEnvelopes(session, 1)
	stream := &testTunnelStream{
		recv: make(chan tunnelStreamRecv, 2),
		sent: make(chan *sliverpb.TunnelData, 3),
	}
	hookResult := make(chan error, 1)
	server := &Server{tunnelDataBeforeImplantControl: func(receivedTunnel *core.Tunnel, frame *sliverpb.TunnelData) {
		if receivedTunnel != tunnel {
			hookResult <- fmt.Errorf("pre-resend tunnel = %p, want %p", receivedTunnel, tunnel)
			return
		}
		if !frame.Resend || frame.Ack != 0 || frame.Sequence != 0 || len(frame.Data) != 0 {
			hookResult <- fmt.Errorf("pre-resend frame = %+v", frame)
			return
		}
		if !core.Tunnels.CloseIf(tunnel) {
			hookResult <- errors.New("external close did not own the tunnel generation")
			return
		}
		hookResult <- nil
	}}

	stream.recv <- tunnelStreamRecv{data: &sliverpb.TunnelData{TunnelID: tunnel.ID, SessionID: session.ID}}
	handlerDone := make(chan error, 1)
	go func() { handlerDone <- server.TunnelData(stream) }()
	waitForTunnelStreamSend(t, stream.sent)

	deliveryDone := make(chan bool, 1)
	go func() {
		deliveryDone <- tunnel.SendDataFromImplant(&sliverpb.TunnelData{
			TunnelID: tunnel.ID,
			Resend:   true,
			Ack:      0,
		})
	}()
	select {
	case delivered := <-deliveryDone:
		if !delivered {
			t.Fatal("in-flight implant resend was not accepted before external close")
		}
	case <-time.After(time.Second):
		t.Fatal("in-flight implant resend delivery blocked")
	}
	select {
	case err := <-hookResult:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("pre-resend external close hook did not run")
	}

	terminal := waitForTunnelStreamSend(t, stream.sent)
	if terminal.TunnelID != tunnel.ID || terminal.SessionID != session.ID || !terminal.Closed {
		t.Fatalf("external-close terminal frame = %+v", terminal)
	}
	select {
	case <-session.Connection.Done():
		t.Fatal("benign external tunnel close before resend tore down the implant connection")
	default:
	}
	if current := core.Sessions.Get(session.ID); current != session {
		t.Fatalf("session after benign external close = %p, want %p", current, session)
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
}

func TestTunnelDataInvalidResendClosesImplantConnection(t *testing.T) {
	session, tunnels := newTunnelStreamTestSession(t, 1)
	tunnel := tunnels[0]
	t.Cleanup(session.Connection.Close)
	stream := &testTunnelStream{
		recv: make(chan tunnelStreamRecv, 2),
		sent: make(chan *sliverpb.TunnelData, 3),
	}
	stream.recv <- tunnelStreamRecv{data: &sliverpb.TunnelData{TunnelID: tunnel.ID, SessionID: session.ID}}
	handlerDone := make(chan error, 1)
	go func() { handlerDone <- (&Server{}).TunnelData(stream) }()
	waitForTunnelStreamSend(t, stream.sent)
	replacement := replaceTunnelStreamTestSession(t, session)

	requireImplantTunnelDelivery(t, tunnel, &sliverpb.TunnelData{
		TunnelID: tunnel.ID,
		Resend:   true,
		Ack:      0,
	})
	select {
	case <-session.Connection.Done():
	case <-time.After(time.Second):
		t.Fatal("invalid resend did not fail the implant connection closed")
	}
	select {
	case <-tunnel.Done():
	case <-time.After(time.Second):
		t.Fatal("invalid resend did not close the tunnel")
	}
	terminal := waitForTunnelStreamSend(t, stream.sent)
	if terminal.TunnelID != tunnel.ID || terminal.SessionID != session.ID || !terminal.Closed {
		t.Fatalf("invalid-resend terminal frame = %+v", terminal)
	}
	assertReplacementConnectionUntouched(t, replacement)

	stream.recv <- tunnelStreamRecv{err: io.EOF}
	select {
	case err := <-handlerDone:
		if err != nil {
			t.Fatalf("TunnelData EOF result = %v, want nil", err)
		}
	case <-time.After(time.Second):
		t.Fatal("TunnelData did not return after EOF")
	}
	assertReplacementConnectionUntouched(t, replacement)
}

func TestForEachTunnelPayloadFrame(t *testing.T) {
	testCases := []int{
		0,
		core.MaxTunnelFrameBytes,
		core.MaxTunnelFrameBytes + 1,
		96 * 1024,
		2*core.MaxTunnelFrameBytes + 1,
	}
	for _, size := range testCases {
		t.Run(fmt.Sprintf("bytes-%d", size), func(t *testing.T) {
			payload := make([]byte, size)
			for index := range payload {
				payload[index] = byte(index)
			}

			frames := [][]byte{}
			if err := forEachTunnelPayloadFrame(payload, func(frame []byte) error {
				frames = append(frames, append([]byte(nil), frame...))
				return nil
			}); err != nil {
				t.Fatalf("segment payload: %v", err)
			}
			wantFrames := 1
			if size > 0 {
				wantFrames = 1 + (size-1)/core.MaxTunnelFrameBytes
			}
			if len(frames) != wantFrames {
				t.Fatalf("frame count = %d, want %d", len(frames), wantFrames)
			}
			for index, frame := range frames {
				if len(frame) > core.MaxTunnelFrameBytes {
					t.Fatalf("frame %d length = %d, limit %d", index, len(frame), core.MaxTunnelFrameBytes)
				}
			}
			if got := bytes.Join(frames, nil); !bytes.Equal(got, payload) {
				t.Fatal("segmented payload did not reassemble byte-for-byte")
			}
		})
	}

	stopErr := errors.New("stop iteration")
	frames := 0
	err := forEachTunnelPayloadFrame(make([]byte, 2*core.MaxTunnelFrameBytes+1), func([]byte) error {
		frames++
		return stopErr
	})
	if !errors.Is(err, stopErr) || frames != 1 {
		t.Fatalf("callback stop = (%v, %d frames), want (%v, 1 frame)", err, frames, stopErr)
	}
}

func TestTunnelDataSegmentsOversizedClientPayload(t *testing.T) {
	session, tunnels := newTunnelStreamTestSession(t, 1)
	session.Connection.Send = make(chan *sliverpb.Envelope, 4)
	tunnel := tunnels[0]
	stream := &testTunnelStream{
		recv: make(chan tunnelStreamRecv, 3),
		sent: make(chan *sliverpb.TunnelData, 2),
	}

	stream.recv <- tunnelStreamRecv{data: &sliverpb.TunnelData{
		TunnelID:  tunnel.ID,
		SessionID: session.ID,
	}}
	handlerDone := make(chan error, 1)
	go func() { handlerDone <- (&Server{}).TunnelData(stream) }()
	waitForTunnelStreamSend(t, stream.sent)

	payload := make([]byte, 96*1024)
	for index := range payload {
		payload[index] = byte(index)
	}
	stream.recv <- tunnelStreamRecv{data: &sliverpb.TunnelData{
		TunnelID:  tunnel.ID,
		SessionID: session.ID,
		Data:      payload,
	}}

	reassembled := make([]byte, 0, len(payload))
	for wantSequence := uint64(0); wantSequence < 2; wantSequence++ {
		var envelope *sliverpb.Envelope
		select {
		case envelope = <-session.Connection.Send:
		case <-time.After(time.Second):
			t.Fatalf("timed out waiting for implant frame %d", wantSequence)
		}
		if envelope.Type != sliverpb.MsgTunnelData {
			t.Fatalf("envelope type = %d, want MsgTunnelData", envelope.Type)
		}
		frame := &sliverpb.TunnelData{}
		if err := proto.Unmarshal(envelope.Data, frame); err != nil {
			t.Fatalf("unmarshal frame %d: %v", wantSequence, err)
		}
		if frame.Sequence != wantSequence {
			t.Fatalf("frame sequence = %d, want %d", frame.Sequence, wantSequence)
		}
		if len(frame.Data) > core.MaxTunnelFrameBytes {
			t.Fatalf("frame %d length = %d, limit %d", wantSequence, len(frame.Data), core.MaxTunnelFrameBytes)
		}
		reassembled = append(reassembled, frame.Data...)
	}
	if !bytes.Equal(reassembled, payload) {
		t.Fatal("implant frames did not reassemble to the client payload")
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
		t.Cleanup(func() { core.Tunnels.CloseIf(tunnel) })
	}
	return session, tunnels
}

func replaceTunnelStreamTestSession(t *testing.T, creatingSession *core.Session) *core.Session {
	t.Helper()
	replacementConnection := core.NewImplantConnection("mtls", "replacement")
	replacementConnection.Send = make(chan *sliverpb.Envelope, 4)
	replacement := core.NewSession(replacementConnection)
	replacement.ID = creatingSession.ID
	core.Sessions.Add(replacement)
	t.Cleanup(replacementConnection.Close)
	return replacement
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

func waitForImplantTunnelEnvelope(t *testing.T, sent <-chan *sliverpb.Envelope) *sliverpb.TunnelData {
	t.Helper()
	select {
	case envelope := <-sent:
		return decodeImplantTunnelEnvelope(t, envelope)
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for implant tunnel envelope")
		return nil
	}
}

func waitForExactImplantTunnelEnvelope(t *testing.T, creating <-chan *sliverpb.Envelope, replacement <-chan *sliverpb.Envelope, expectedType uint32) *sliverpb.TunnelData {
	t.Helper()
	select {
	case envelope := <-creating:
		if envelope == nil || envelope.Type != expectedType {
			t.Fatalf("creating-connection envelope type = %v, want %v", envelope, expectedType)
		}
		return decodeImplantTunnelEnvelope(t, envelope)
	case envelope := <-replacement:
		t.Fatalf("tunnel envelope was routed to replacement implant connection: %+v", envelope)
		return nil
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for exact creating-connection tunnel envelope")
		return nil
	}
}

func decodeImplantTunnelEnvelope(t *testing.T, envelope *sliverpb.Envelope) *sliverpb.TunnelData {
	t.Helper()
	if envelope == nil || (envelope.Type != sliverpb.MsgTunnelData && envelope.Type != sliverpb.MsgTunnelClose) {
		t.Fatalf("implant tunnel envelope = %+v", envelope)
	}
	frame := &sliverpb.TunnelData{}
	if err := proto.Unmarshal(envelope.Data, frame); err != nil {
		t.Fatalf("decode implant tunnel envelope: %v", err)
	}
	return frame
}

func assertReplacementConnectionUntouched(t *testing.T, replacement *core.Session) {
	t.Helper()
	if current := core.Sessions.Get(replacement.ID); current != replacement {
		t.Fatalf("replacement session was removed or superseded: got %p, want %p", current, replacement)
	}
	select {
	case <-replacement.Connection.Done():
		t.Fatal("tunnel lifecycle closed replacement implant connection")
	default:
	}
	select {
	case envelope := <-replacement.Connection.Send:
		t.Fatalf("tunnel lifecycle sent envelope to replacement implant connection: %+v", envelope)
	default:
	}
}

func requireImplantTunnelDelivery(t *testing.T, tunnel *core.Tunnel, frame *sliverpb.TunnelData) {
	t.Helper()
	deliveryDone := make(chan bool, 1)
	go func() {
		deliveryDone <- tunnel.SendDataFromImplant(frame)
	}()
	select {
	case delivered := <-deliveryDone:
		if !delivered {
			t.Fatalf("implant frame was not delivered to tunnel %d", tunnel.ID)
		}
	case <-time.After(time.Second):
		t.Fatalf("implant frame delivery to tunnel %d blocked", tunnel.ID)
	}
}

func waitForTunnelDataHandler(t *testing.T, handlerDone <-chan error) {
	t.Helper()
	select {
	case err := <-handlerDone:
		if err != nil {
			t.Fatalf("TunnelData EOF result = %v, want nil", err)
		}
	case <-time.After(time.Second):
		t.Fatal("TunnelData did not return after EOF")
	}
}
