package handlers

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/bishopfox/sliver/server/core"
	"github.com/bishopfox/sliver/server/core/rtunnels"
	"google.golang.org/protobuf/proto"
)

type testWriteCloser struct {
	closed bool
}

type discardSocksSender struct{}

func (discardSocksSender) Send(*sliverpb.SocksData) error { return nil }

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
		conn.Close()
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

func marshalSessionSocksData(t *testing.T, frame *sliverpb.SocksData) []byte {
	t.Helper()
	data, err := proto.Marshal(frame)
	if err != nil {
		t.Fatalf("marshal SOCKS data: %v", err)
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

func TestRegisterSessionHandlerPropagatesCapabilities(t *testing.T) {
	conn := core.NewImplantConnection("test", "n/a")
	registerData, err := proto.Marshal(&sliverpb.Register{
		Uuid:         "4e3c1713-05fd-485a-bef7-fef5df4fa8c4",
		Capabilities: sliverpb.CapabilityBOFV1,
	})
	if err != nil {
		t.Fatalf("marshal register: %v", err)
	}

	registerSessionHandler(conn, registerData)
	t.Cleanup(conn.Close)

	session := core.Sessions.FromImplantConnection(conn)
	if session == nil {
		t.Fatal("expected registered session")
	}
	if session.Capabilities != sliverpb.CapabilityBOFV1 {
		t.Fatalf("expected Capabilities=%d, got %d", sliverpb.CapabilityBOFV1, session.Capabilities)
	}
}

func TestRegisterSessionHandlerRejectsDuplicateConnectionRegistration(t *testing.T) {
	connection := core.NewImplantConnection("test", "n/a")
	t.Cleanup(connection.Close)
	registerData, err := proto.Marshal(&sliverpb.Register{Uuid: "4e3c1713-05fd-485a-bef7-fef5df4fa8c4"})
	if err != nil {
		t.Fatalf("marshal register: %v", err)
	}
	registerSessionHandler(connection, registerData)
	first := core.Sessions.FromImplantConnection(connection)
	if first == nil {
		t.Fatal("first registration did not create a session")
	}
	registerSessionHandler(connection, registerData)
	if got := core.Sessions.FromImplantConnection(connection); got != first {
		t.Fatalf("duplicate registration replaced session: got=%p want=%p", got, first)
	}
	connection.Close()
	if got := core.Sessions.FromImplantConnection(connection); got != nil {
		core.Sessions.Remove(got.ID)
		t.Fatal("registered session survived connection close")
	}
}

func TestRegisteredConnectionCleanupPreservesSameIDReplacement(t *testing.T) {
	originalConnection := core.NewImplantConnection("test", "original")
	registerData, err := proto.Marshal(&sliverpb.Register{Uuid: "4e3c1713-05fd-485a-bef7-fef5df4fa8c4"})
	if err != nil {
		t.Fatalf("marshal register: %v", err)
	}
	registerSessionHandler(originalConnection, registerData)
	original := core.Sessions.FromImplantConnection(originalConnection)
	if original == nil {
		t.Fatal("original connection did not register a session")
	}

	replacementConnection := core.NewImplantConnection("test", "replacement")
	replacement := core.NewSession(replacementConnection)
	replacement.ID = original.ID
	core.Sessions.Add(replacement)
	t.Cleanup(func() {
		core.Sessions.RemoveIf(replacement)
		originalConnection.Close()
		replacementConnection.Close()
	})

	originalConnection.Close()
	if got := core.Sessions.Get(original.ID); got != replacement {
		t.Fatalf("session after old connection cleanup = %p, want replacement %p", got, replacement)
	}
	select {
	case <-replacementConnection.Done():
		t.Fatal("old connection cleanup closed replacement connection")
	default:
	}
}

func TestRegisterSessionHandlerCloseRaceNeverPublishesStrandedSession(t *testing.T) {
	registerData, err := proto.Marshal(&sliverpb.Register{Uuid: "4e3c1713-05fd-485a-bef7-fef5df4fa8c4"})
	if err != nil {
		t.Fatalf("marshal register: %v", err)
	}
	for iteration := range 100 {
		connection := core.NewImplantConnection("test", "n/a")
		t.Cleanup(connection.Close)
		start := make(chan struct{})
		var waitGroup sync.WaitGroup
		waitGroup.Add(2)
		go func() {
			defer waitGroup.Done()
			<-start
			registerSessionHandler(connection, registerData)
		}()
		go func() {
			defer waitGroup.Done()
			<-start
			connection.Close()
		}()
		close(start)
		finished := make(chan struct{})
		go func() {
			waitGroup.Wait()
			close(finished)
		}()
		select {
		case <-finished:
		case <-time.After(time.Second):
			connection.Close()
			t.Fatalf("iteration %d registration/close race did not finish", iteration)
		}
		if session := core.Sessions.FromImplantConnection(connection); session != nil {
			core.Sessions.Remove(session.ID)
			t.Fatalf("iteration %d stranded session %s", iteration, session.ID)
		}
	}
}

func TestReverseTunnelIDCapacityDisconnectRevokesSessionAuthorization(t *testing.T) {
	registry := rtunnels.NewRegistry()
	installHandlerBroker(t, registry, nil)
	connection := core.NewImplantConnection("test", "n/a")
	t.Cleanup(connection.Close)
	registerData, err := proto.Marshal(&sliverpb.Register{Uuid: "4e3c1713-05fd-485a-bef7-fef5df4fa8c4"})
	if err != nil {
		t.Fatalf("marshal register: %v", err)
	}
	registerSessionHandler(connection, registerData)
	session := core.Sessions.FromImplantConnection(connection)
	if session == nil {
		t.Fatal("expected registered session")
	}
	authorizationID := beginActiveAuthorization(t, registry, session.ID, "127.0.0.1:4444", 1)

	exhausted := false
	for tunnelID := uint64(0); tunnelID < 100_000; tunnelID++ {
		switch result := connection.TryClaimReverseTunnelID(tunnelID); result {
		case core.ReverseTunnelIDClaimed:
			continue
		case core.ReverseTunnelIDCapacityExhausted:
			exhausted = true
		default:
			t.Fatalf("claim tunnel %d result = %v", tunnelID, result)
		}
		break
	}
	if !exhausted {
		t.Fatal("reverse tunnel claim history did not reach its bounded capacity")
	}
	select {
	case <-connection.Done():
	case <-time.After(time.Second):
		t.Fatal("capacity exhaustion did not close the server-side C2 connection")
	}
	if got := core.Sessions.FromImplantConnection(connection); got != nil {
		core.Sessions.Remove(got.ID)
		t.Fatalf("capacity exhaustion retained session %s", got.ID)
	}
	if _, ok := registry.Lookup(session.ID, authorizationID); ok {
		t.Fatal("capacity exhaustion retained the session authorization")
	}
	if got := registry.List(session.ID); len(got) != 0 {
		t.Fatalf("capacity exhaustion retained %d session authorizations", len(got))
	}
}

func TestTunnelCloseHandlerClosesOwnedReverseTunnel(t *testing.T) {
	conn, session := addTestSession(t)
	tunnelID := core.NewTunnelID()
	writer := &testWriteCloser{}

	tunnel := rtunnels.NewRTunnel(tunnelID, session.ID, writer)
	if !rtunnels.TryAddRTunnel(tunnel) {
		t.Fatalf("failed to register reverse tunnel %d", tunnelID)
	}
	t.Cleanup(func() {
		rtunnels.RemoveRTunnelIf(tunnelID, tunnel)
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

	tunnel := rtunnels.NewRTunnel(tunnelID, ownerSession.ID, writer)
	if !rtunnels.TryAddRTunnel(tunnel) {
		t.Fatalf("failed to register reverse tunnel %d", tunnelID)
	}
	t.Cleanup(func() {
		rtunnels.RemoveRTunnelIf(tunnelID, tunnel)
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

func TestReverseTunnelOpeningAdmissionBoundsSessionAndGlobalAttempts(t *testing.T) {
	admission := newReverseTunnelAdmission(2, 3)
	if !admission.acquire("first") {
		t.Fatal("expected first per-session opening slot")
	}
	if !admission.acquire("first") {
		t.Fatal("expected second per-session opening slot")
	}
	if admission.acquire("first") {
		t.Fatal("acquired opening slot beyond per-session limit")
	}
	if !admission.acquire("second") {
		t.Fatal("expected remaining global opening slot")
	}
	if admission.acquire("third") {
		t.Fatal("acquired opening slot beyond global limit")
	}

	admission.release("first")
	if !admission.acquire("third") {
		t.Fatal("released opening slot was not reusable")
	}
	admission.release("missing")
	admission.mutex.Lock()
	if admission.total != 3 {
		t.Fatalf("invalid release changed total to %d", admission.total)
	}
	admission.mutex.Unlock()
}

func TestReverseTunnelOpeningBoundsSameIDWaiters(t *testing.T) {
	workers := installHandlerAdmissions(t, newReverseTunnelAdmission(1, 1), newReverseTunnelAdmission(maxReverseTunnelOpeningWaiters+1, maxReverseTunnelOpeningWaiters+1))
	opening := newReverseTunnelOpening("session", func() {})
	var readyOnce sync.Once
	closeReady := func() { readyOnce.Do(func() { close(opening.ready) }) }
	t.Cleanup(closeReady)
	results := make(chan error, maxReverseTunnelOpeningWaiters)
	for range maxReverseTunnelOpeningWaiters {
		workers.launch(func() {
			results <- opening.wait(time.Second)
		})
	}
	deadline := time.Now().Add(time.Second)
	for len(opening.waiters) != maxReverseTunnelOpeningWaiters && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if got := len(opening.waiters); got != maxReverseTunnelOpeningWaiters {
		t.Fatalf("opening waiters = %d, want %d", got, maxReverseTunnelOpeningWaiters)
	}
	if err := opening.wait(time.Second); err != errReverseTunnelOpeningWaiterLimit {
		t.Fatalf("extra opening waiter error = %v, want %v", err, errReverseTunnelOpeningWaiterLimit)
	}
	closeReady()
	for range maxReverseTunnelOpeningWaiters {
		select {
		case err := <-results:
			if err != nil {
				t.Fatalf("admitted opening waiter error = %v", err)
			}
		case <-time.After(time.Second):
			t.Fatal("opening waiter did not finish")
		}
	}
}

func TestReverseTunnelOpeningAcceptsLegitimateConcurrentFramesDuringDial(t *testing.T) {
	const concurrentFrames = maxReverseTunnelOpeningWaiters

	registry := rtunnels.NewRegistry()
	dialer := newGatedHandlerDialer()
	installHandlerBroker(t, registry, dialer)
	attempts := newReverseTunnelAdmission(2, 2)
	waiters := newReverseTunnelAdmission(maxReverseTunnelOpeningWaiters, maxReverseTunnelOpeningWaiters)
	workers := installHandlerAdmissions(t, attempts, waiters)
	connection, session := addBufferedTestSession(t)
	connection.Send = make(chan *sliverpb.Envelope, concurrentFrames+4)
	authorizationID := beginActiveAuthorization(t, registry, session.ID, "127.0.0.1:4450", 3)
	const tunnelID = uint64(0x4100)

	createDone := workers.launch(func() {
		tunnelDataHandler(connection, reverseCreateData(t, tunnelID, authorizationID))
	})
	select {
	case <-dialer.started:
	case <-time.After(time.Second):
		t.Fatal("reverse-tunnel broker dial did not start")
	}

	frameDone := make([]<-chan struct{}, 0, concurrentFrames)
	for sequence := uint64(1); sequence <= concurrentFrames; sequence++ {
		frame := marshalTunnelData(t, &sliverpb.TunnelData{
			TunnelID: tunnelID,
			Sequence: sequence,
			Data:     []byte{byte(sequence)},
		})
		frameDone = append(frameDone, workers.launch(func() {
			tunnelDataHandler(connection, frame)
		}))
	}

	deadline := time.Now().Add(time.Second)
	var opening *reverseTunnelOpening
	for time.Now().Before(deadline) {
		value, ok := reverseTunnelOpenings.Load(tunnelID)
		if ok {
			opening = value.(*reverseTunnelOpening)
			if len(opening.waiters) == concurrentFrames {
				break
			}
		}
		time.Sleep(time.Millisecond)
	}
	admitted := 0
	if opening != nil {
		admitted = len(opening.waiters)
	}
	if admitted != concurrentFrames {
		t.Fatalf("opening admitted %d/%d legitimate concurrent frames", admitted, concurrentFrames)
	}
	if opening.closing.Load() {
		t.Fatal("legitimate concurrent frames canceled the broker dial")
	}
	select {
	case <-connection.Done():
		t.Fatal("legitimate concurrent frames closed the implant connection")
	default:
	}

	dialer.allow()
	waitHandler(t, createDone, "create handler")
	for index, done := range frameDone {
		waitHandler(t, done, fmt.Sprintf("concurrent frame %d", index+1))
	}
	tunnel := rtunnels.GetRTunnel(tunnelID)
	if tunnel == nil {
		t.Fatal("authorized opening did not publish a reverse tunnel")
	}
	if got, want := tunnel.ReadSequence(), uint64(concurrentFrames+1); got != want {
		t.Fatalf("reverse tunnel read sequence = %d, want %d", got, want)
	}
	select {
	case rejection := <-connection.Send:
		t.Fatalf("legitimate concurrent frame emitted rejection type %d", rejection.Type)
	default:
	}
	closeTestReverseTunnel(tunnelID)
	assertReverseTunnelOpeningsEmpty(t)
	assertHandlerAdmissionEmpty(t, attempts)
	assertHandlerAdmissionEmpty(t, waiters)
}

//nolint:gocyclo // The test synchronizes a full data window, terminal ordering, and duplicate close handling.
func TestReverseTunnelOpeningTerminalPreservesFullDataWindow(t *testing.T) {
	const concurrentFrames = maxReverseTunnelOpeningWaiters

	registry := rtunnels.NewRegistry()
	dialer := newGatedHandlerDialer()
	installHandlerBroker(t, registry, dialer)
	attempts := newReverseTunnelAdmission(2, 2)
	waiters := newReverseTunnelAdmission(maxReverseTunnelOpeningWaiters, maxReverseTunnelOpeningWaiters)
	workers := installHandlerAdmissions(t, attempts, waiters)
	connection, session := addBufferedTestSession(t)
	session.Capabilities = sliverpb.CapabilityTunnelTerminalV1
	authorizationID := beginActiveAuthorization(t, registry, session.ID, "127.0.0.1:4451", 4)
	const tunnelID = uint64(0x4101)

	createDone := workers.launch(func() {
		tunnelDataHandler(connection, reverseCreateData(t, tunnelID, authorizationID))
	})
	select {
	case <-dialer.started:
	case <-time.After(time.Second):
		t.Fatal("reverse-tunnel broker dial did not start")
	}

	frameDone := make([]<-chan struct{}, 0, concurrentFrames)
	for sequence := uint64(1); sequence <= concurrentFrames; sequence++ {
		frame := marshalTunnelData(t, &sliverpb.TunnelData{
			TunnelID: tunnelID,
			Sequence: sequence,
			Data:     []byte{byte(sequence)},
		})
		frameDone = append(frameDone, workers.launch(func() {
			tunnelDataHandler(connection, frame)
		}))
	}

	deadline := time.Now().Add(time.Second)
	var opening *reverseTunnelOpening
	for time.Now().Before(deadline) {
		value, ok := reverseTunnelOpenings.Load(tunnelID)
		if ok {
			opening = value.(*reverseTunnelOpening)
			if len(opening.waiters) == concurrentFrames {
				break
			}
		}
		time.Sleep(time.Millisecond)
	}
	if opening == nil || len(opening.waiters) != concurrentFrames {
		t.Fatalf("opening did not fill the %d-frame data window", concurrentFrames)
	}

	terminal := marshalTunnelData(t, &sliverpb.TunnelData{
		TunnelID: tunnelID,
		Sequence: concurrentFrames + 1,
		Closed:   true,
	})
	terminalDone := workers.launch(func() {
		tunnelCloseHandler(connection, terminal)
	})
	deadline = time.Now().Add(time.Second)
	for !opening.terminal.Load() && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if !opening.terminal.Load() {
		t.Fatal("terminal did not claim the opening's reserved slot")
	}
	if opening.closing.Load() {
		t.Fatal("sequenced terminal canceled the authorized broker dial")
	}
	select {
	case <-terminalDone:
		t.Fatal("terminal completed before the broker dial settled")
	default:
	}
	duplicateTerminalDone := make([]<-chan struct{}, 0, 256)
	for range 256 {
		duplicateTerminalDone = append(duplicateTerminalDone, workers.launch(func() {
			tunnelCloseHandler(connection, terminal)
		}))
	}
	for index, done := range duplicateTerminalDone {
		waitHandler(t, done, fmt.Sprintf("duplicate terminal %d", index+1))
	}

	dialer.allow()
	waitHandler(t, createDone, "create handler")
	for index, done := range frameDone {
		waitHandler(t, done, fmt.Sprintf("concurrent frame %d", index+1))
	}
	waitHandler(t, terminalDone, "terminal handler")

	deadline = time.Now().Add(time.Second)
	for dialer.received.Load() != concurrentFrames && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if got := dialer.received.Load(); got != concurrentFrames {
		t.Fatalf("authorized target received %d/%d bytes before terminal close", got, concurrentFrames)
	}

	if tunnel := rtunnels.GetRTunnel(tunnelID); tunnel != nil {
		closeTestReverseTunnel(tunnelID)
		t.Fatalf("sequenced terminal left reverse tunnel %d published", tunnelID)
	}
	select {
	case rejection := <-connection.Send:
		t.Fatalf("sequenced terminal emitted rejection type %d", rejection.Type)
	default:
	}
	select {
	case <-connection.Done():
		t.Fatal("terminal at a full data window closed the implant connection")
	default:
	}
	assertReverseTunnelOpeningsEmpty(t)
	assertHandlerAdmissionEmpty(t, attempts)
	assertHandlerAdmissionEmpty(t, waiters)
}

func TestRejectReverseTunnelDeliversToDelayedReceiver(t *testing.T) {
	connection := core.NewImplantConnection("test", "n/a")
	t.Cleanup(connection.Close)

	rejectReverseTunnel(connection, 0xfeed, errReverseTunnelOpeningLimit)
	time.Sleep(10 * time.Millisecond)
	select {
	case envelope := <-connection.Send:
		if envelope.Type != sliverpb.MsgTunnelClose {
			t.Fatalf("rejection envelope type = %d, want %d", envelope.Type, sliverpb.MsgTunnelClose)
		}
		closed := &sliverpb.TunnelData{}
		if err := proto.Unmarshal(envelope.Data, closed); err != nil {
			t.Fatalf("unmarshal rejection: %v", err)
		}
		if !closed.Closed || closed.TunnelID != 0xfeed {
			t.Fatalf("unexpected rejection payload: %+v", closed)
		}
	case <-time.After(time.Second):
		t.Fatal("delayed rejection receiver did not get tunnel close")
	}
}

func TestTunnelDataHandlerClosesConnectionBeforeOversizedUnmarshal(t *testing.T) {
	connection := core.NewImplantConnection("test", "n/a")
	tunnelDataHandler(connection, make([]byte, maxTunnelDataMessageBytes+1))
	select {
	case <-connection.Done():
	case <-time.After(time.Second):
		t.Fatal("oversized tunnel data did not close the implant connection")
	}
}

func TestTunnelDataHandlerRejectsOversizedGenericTunnelFrame(t *testing.T) {
	connection, session := addTestSession(t)
	tunnel, err := core.Tunnels.Create(session.ID)
	if err != nil {
		t.Fatalf("create generic tunnel: %v", err)
	}
	t.Cleanup(func() { core.Tunnels.CloseIf(tunnel) })
	encoded, err := proto.Marshal(&sliverpb.TunnelData{
		TunnelID: tunnel.ID,
		Sequence: 0,
		Data:     make([]byte, core.MaxTunnelFrameBytes+1),
	})
	if err != nil {
		t.Fatalf("marshal oversized generic tunnel frame: %v", err)
	}
	if len(encoded) > maxTunnelDataMessageBytes {
		t.Fatalf("test frame encoded to %d bytes, raw envelope limit is %d", len(encoded), maxTunnelDataMessageBytes)
	}

	tunnelDataHandler(connection, encoded)
	select {
	case <-connection.Done():
	case <-time.After(time.Second):
		t.Fatal("oversized generic tunnel frame did not fail the implant connection closed")
	}
	select {
	case <-tunnel.Done():
	case <-time.After(time.Second):
		t.Fatal("oversized generic tunnel frame did not close its tunnel generation")
	}
	if got := core.Tunnels.Get(tunnel.ID); got != nil {
		t.Fatalf("oversized generic tunnel frame left generation registered: %p", got)
	}
}

func TestTunnelResourcePressureClassification(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "ingress frame limit", err: core.ErrTunnelIngressLimit, want: true},
		{name: "wrapped byte limit", err: fmt.Errorf("wrapped: %w", core.ErrTunnelPendingBytes), want: true},
		{name: "sequence violation", err: core.ErrTunnelSequenceWindow, want: false},
		{name: "closed tunnel", err: core.ErrTunnelClosed, want: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := isTunnelResourcePressure(test.err); got != test.want {
				t.Fatalf("isTunnelResourcePressure(%v) = %v, want %v", test.err, got, test.want)
			}
		})
	}
}

func TestSocksDataHandlerRelaysCanonicalImplantAcknowledgementOutOfBand(t *testing.T) {
	connection, session := addTestSession(t)
	session.Capabilities = sliverpb.CapabilitySocksFlowControlV1
	tunnel, err := core.SocksTunnels.CreateWithCapabilities(session.ID, sliverpb.CapabilitySocksFlowControlV1)
	if err != nil {
		t.Fatalf("create flow-controlled SOCKS tunnel: %v", err)
	}
	t.Cleanup(func() { core.SocksTunnels.CloseIf(tunnel) })
	client := core.NewSocksClient(discardSocksSender{})
	if owned, newlyBound, bindErr := tunnel.BindClientWithNegotiatedCapabilities(client, "", "", true, sliverpb.CapabilitySocksFlowControlV1); bindErr != nil || !owned || !newlyBound {
		t.Fatalf("bind flow-controlled SOCKS tunnel = owned:%v new:%v err:%v", owned, newlyBound, bindErr)
	}
	atomic.StoreUint64(&tunnel.ToImplantSequence, 3)

	for _, ack := range []uint64{1, 3, 2} {
		socksDataHandler(connection, marshalSessionSocksData(t, &sliverpb.SocksData{TunnelID: tunnel.ID, Ack: ack}))
	}
	select {
	case ack := <-tunnel.AcknowledgementsToClient():
		if ack != 3 {
			t.Fatalf("coalesced implant acknowledgement = %d, want 3", ack)
		}
	default:
		t.Fatal("implant acknowledgement was not relayed to client mailbox")
	}
	select {
	case frame := <-tunnel.FromImplant():
		tunnel.CompleteFromImplant(frame)
		t.Fatalf("implant acknowledgement entered SOCKS payload queue: %+v", frame)
	default:
	}
	if got := core.SocksTunnels.Get(tunnel.ID); got != tunnel {
		t.Fatalf("valid acknowledgement closed SOCKS tunnel: got=%p want=%p", got, tunnel)
	}
	select {
	case <-connection.Done():
		t.Fatal("valid acknowledgement closed implant connection")
	default:
	}
}

func TestSocksDataHandlerRejectsInvalidAcknowledgementOnExactTunnelOnly(t *testing.T) {
	tests := []struct {
		name  string
		frame *sliverpb.SocksData
	}{
		{name: "future acknowledgement", frame: &sliverpb.SocksData{Ack: 2}},
		{name: "mixed acknowledgement payload", frame: &sliverpb.SocksData{Ack: 1, Data: []byte("mixed")}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			connection, session := addTestSession(t)
			session.Capabilities = sliverpb.CapabilitySocksFlowControlV1
			tunnel, err := core.SocksTunnels.CreateWithCapabilities(session.ID, sliverpb.CapabilitySocksFlowControlV1)
			if err != nil {
				t.Fatalf("create flow-controlled SOCKS tunnel: %v", err)
			}
			t.Cleanup(func() { core.SocksTunnels.CloseIf(tunnel) })
			client := core.NewSocksClient(discardSocksSender{})
			if owned, newlyBound, bindErr := tunnel.BindClientWithNegotiatedCapabilities(client, "", "", true, sliverpb.CapabilitySocksFlowControlV1); bindErr != nil || !owned || !newlyBound {
				t.Fatalf("bind flow-controlled SOCKS tunnel = owned:%v new:%v err:%v", owned, newlyBound, bindErr)
			}
			atomic.StoreUint64(&tunnel.ToImplantSequence, 1)
			frame := proto.Clone(test.frame).(*sliverpb.SocksData)
			frame.TunnelID = tunnel.ID

			socksDataHandler(connection, marshalSessionSocksData(t, frame))
			select {
			case <-tunnel.Done():
			case <-time.After(time.Second):
				t.Fatal("invalid acknowledgement did not close exact SOCKS tunnel")
			}
			if got := core.SocksTunnels.Get(tunnel.ID); got != nil {
				t.Fatalf("invalid acknowledgement retained SOCKS tunnel: %p", got)
			}
			select {
			case <-connection.Done():
				t.Fatal("tunnel-scoped acknowledgement violation closed implant connection")
			default:
			}
		})
	}
}

//nolint:gocyclo // The assertions prove failure isolation across exact tunnel generations.
func TestTunnelDataHandlerScopesIngressPressureToExactTunnel(t *testing.T) {
	const receiveWindow = 128

	connection, session := addTestSession(t)
	tunnel, err := core.Tunnels.Create(session.ID)
	if err != nil {
		t.Fatalf("create saturated generic tunnel: %v", err)
	}
	sibling, err := core.Tunnels.Create(session.ID)
	if err != nil {
		core.Tunnels.CloseIf(tunnel)
		t.Fatalf("create sibling generic tunnel: %v", err)
	}
	t.Cleanup(func() {
		core.Tunnels.CloseIf(tunnel)
		core.Tunnels.CloseIf(sibling)
	})

	// Retain all but one receive-window reservation behind a missing sequence
	// zero. Once zero is admitted, its delivery actor blocks on sequence one
	// and holds the serialization lock while a resend consumes the last slot.
	for sequence := 1; sequence < receiveWindow; sequence++ {
		if err := tunnel.ProcessDataFromImplant(&sliverpb.TunnelData{
			TunnelID: tunnel.ID,
			Sequence: uint64(sequence),
			Data:     []byte("queued"),
		}); err != nil {
			t.Fatalf("queue sequence %d: %v", sequence, err)
		}
	}
	zeroResult := make(chan error, 1)
	go func() {
		zeroResult <- tunnel.ProcessDataFromImplant(&sliverpb.TunnelData{
			TunnelID: tunnel.ID,
			Sequence: 0,
			Data:     []byte("zero"),
		})
	}()
	select {
	case frame := <-tunnel.FromImplant:
		if frame.Sequence != 0 {
			t.Fatalf("first delivered sequence = %d, want 0", frame.Sequence)
		}
	case <-time.After(time.Second):
		t.Fatal("sequence zero was not delivered")
	}

	probeResults := make(chan error, 2)
	for range 2 {
		go func() {
			probeResults <- tunnel.ProcessDataFromImplant(&sliverpb.TunnelData{
				TunnelID: tunnel.ID,
				Resend:   true,
				Data:     []byte("probe"),
			})
		}()
	}
	select {
	case err := <-probeResults:
		if !errors.Is(err, core.ErrTunnelIngressLimit) {
			t.Fatalf("saturation probe error = %v, want ErrTunnelIngressLimit", err)
		}
	case <-time.After(time.Second):
		t.Fatal("generic tunnel did not reach its receive admission limit")
	}

	tunnelDataHandler(connection, marshalTunnelData(t, &sliverpb.TunnelData{
		TunnelID: tunnel.ID,
		Resend:   true,
		Data:     []byte("overflow"),
	}))
	select {
	case <-tunnel.Done():
	case <-time.After(time.Second):
		t.Fatal("resource-pressure frame did not close its exact generic tunnel")
	}
	if got := core.Tunnels.Get(tunnel.ID); got != nil {
		t.Fatalf("resource-pressure frame retained generic tunnel: %p", got)
	}
	select {
	case <-connection.Done():
		t.Fatal("generic tunnel resource pressure closed the implant connection")
	default:
	}
	if got := core.Tunnels.Get(sibling.ID); got != sibling {
		t.Fatalf("generic tunnel resource pressure replaced sibling: got=%p want=%p", got, sibling)
	}
	select {
	case <-sibling.Done():
		t.Fatal("generic tunnel resource pressure closed sibling tunnel")
	default:
	}

	select {
	case <-zeroResult:
	case <-time.After(time.Second):
		t.Fatal("sequence-zero worker did not exit after exact tunnel close")
	}
	select {
	case <-probeResults:
	case <-time.After(time.Second):
		t.Fatal("admitted saturation probe did not exit after exact tunnel close")
	}
}

func TestGenericTunnelIngressRequiresExactConnectionGeneration(t *testing.T) {
	ownerConnection, ownerSession := addTestSession(t)
	tunnel, err := core.Tunnels.Create(ownerSession.ID)
	if err != nil {
		t.Fatalf("create generic tunnel: %v", err)
	}
	t.Cleanup(func() { core.Tunnels.CloseIf(tunnel) })

	replacementConnection := core.NewImplantConnection("test", "replacement")
	replacement := core.NewSession(replacementConnection)
	replacement.ID = ownerSession.ID
	core.Sessions.Add(replacement)
	t.Cleanup(func() {
		core.Sessions.RemoveIf(replacement)
		replacementConnection.Close()
	})

	oversized := marshalTunnelData(t, &sliverpb.TunnelData{
		TunnelID: tunnel.ID,
		Sequence: 0,
		Data:     make([]byte, core.MaxTunnelFrameBytes+1),
	})
	if len(oversized) > maxTunnelDataMessageBytes {
		t.Fatalf("test frame encoded to %d bytes, raw limit is %d", len(oversized), maxTunnelDataMessageBytes)
	}
	tunnelDataHandler(replacementConnection, oversized)
	if got := core.Tunnels.Get(tunnel.ID); got != tunnel {
		t.Fatalf("replacement connection mutated generic tunnel: got=%p want=%p", got, tunnel)
	}
	select {
	case <-tunnel.Done():
		t.Fatal("replacement connection closed old-generation generic tunnel")
	default:
	}
	select {
	case <-replacementConnection.Done():
		t.Fatal("wrong-generation generic data closed replacement connection")
	default:
	}

	beforeClose := tunnel.LastFromImplantTime()
	time.Sleep(time.Millisecond)
	tunnelCloseHandler(replacementConnection, marshalTunnelData(t, &sliverpb.TunnelData{
		TunnelID: tunnel.ID,
		Closed:   true,
	}))
	if got := tunnel.LastFromImplantTime(); !got.Equal(beforeClose) {
		t.Fatalf("wrong-generation terminal refreshed old tunnel: got=%v want=%v", got, beforeClose)
	}
	if !tunnel.ClaimFromImplantClose() {
		t.Fatal("wrong-generation terminal claimed old tunnel close ownership")
	}

	// Retire the scheduler claim synchronously during cleanup rather than
	// leaving a timer alive for the remainder of the handler package tests.
	core.Tunnels.CloseIf(tunnel)
	select {
	case <-ownerConnection.Done():
		t.Fatal("wrong-generation ingress closed owner connection")
	default:
	}
}

func TestTunnelCloseHandlerClaimsOneSchedulerPerGeneration(t *testing.T) {
	connection, session := addTestSession(t)
	tunnel, err := core.Tunnels.Create(session.ID)
	if err != nil {
		t.Fatalf("create generic tunnel: %v", err)
	}
	t.Cleanup(func() { core.Tunnels.CloseIf(tunnel) })
	terminal := marshalTunnelData(t, &sliverpb.TunnelData{TunnelID: tunnel.ID, Closed: true})

	tunnelCloseHandler(connection, terminal)
	claimedAt := tunnel.LastFromImplantTime()
	time.Sleep(time.Millisecond)
	for range 100 {
		tunnelCloseHandler(connection, terminal)
	}
	if got := tunnel.LastFromImplantTime(); !got.Equal(claimedAt) {
		t.Fatalf("duplicate terminal refreshed generic close grace: got=%v want=%v", got, claimedAt)
	}
	if tunnel.ClaimFromImplantClose() {
		t.Fatal("handler left implant close ownership unclaimed")
	}
}

//nolint:gocyclo // The assertions cover the full sequenced-terminal lifecycle.
func TestCapableTunnelCloseWaitsForTerminalSequenceThenClosesPromptly(t *testing.T) {
	connection, session := addTestSession(t)
	session.Capabilities = sliverpb.CapabilityTunnelTerminalV1
	tunnel, err := core.Tunnels.Create(session.ID)
	if err != nil {
		t.Fatalf("create capable generic tunnel: %v", err)
	}
	t.Cleanup(func() { core.Tunnels.CloseIf(tunnel) })

	tunnelCloseHandler(connection, marshalTunnelData(t, &sliverpb.TunnelData{
		TunnelID: tunnel.ID,
		Sequence: 2,
		Closed:   true,
	}))
	if got := core.Tunnels.Get(tunnel.ID); got != tunnel {
		t.Fatalf("sequenced terminal closed before preceding data: got=%p want=%p", got, tunnel)
	}
	tunnelDataHandler(connection, marshalTunnelData(t, &sliverpb.TunnelData{
		TunnelID: tunnel.ID,
		Sequence: 1,
		Data:     []byte("second"),
	}))

	received := make(chan []*sliverpb.TunnelData, 1)
	go func() {
		frames := make([]*sliverpb.TunnelData, 0, 2)
		for range 2 {
			frames = append(frames, <-tunnel.FromImplant)
		}
		received <- frames
	}()
	tunnelDataHandler(connection, marshalTunnelData(t, &sliverpb.TunnelData{
		TunnelID: tunnel.ID,
		Sequence: 0,
		Data:     []byte("first"),
	}))
	var frames []*sliverpb.TunnelData
	select {
	case frames = <-received:
		if len(frames) != 2 || frames[0].Sequence != 0 || string(frames[0].Data) != "first" ||
			frames[1].Sequence != 1 || string(frames[1].Data) != "second" {
			t.Fatalf("terminal-ordered frames = %+v", frames)
		}
	case <-time.After(time.Second):
		t.Fatal("terminal-ordered frames did not drain")
	}
	select {
	case <-tunnel.Done():
		t.Fatal("capable tunnel closed before preceding frames were forwarded")
	default:
	}
	for _, frame := range frames {
		if err := tunnel.CompleteDataFromImplantForward(frame.Sequence); err != nil {
			t.Fatalf("complete forwarded frame %d: %v", frame.Sequence, err)
		}
	}
	select {
	case <-tunnel.Done():
	case <-time.After(time.Second):
		t.Fatal("capable tunnel did not close promptly after terminal sequence drained")
	}
	if got := core.Tunnels.Get(tunnel.ID); got != nil {
		t.Fatalf("capable terminal retained drained tunnel: %p", got)
	}
	select {
	case <-connection.Done():
		t.Fatal("valid capable terminal closed implant connection")
	default:
	}
}

func TestCapableTunnelCloseArmsDeadlineBeforeTerminalMark(t *testing.T) {
	connection, session := addTestSession(t)
	session.Capabilities = sliverpb.CapabilityTunnelTerminalV1
	tunnel, err := core.Tunnels.Create(session.ID)
	if err != nil {
		t.Fatalf("create capable generic tunnel: %v", err)
	}
	t.Cleanup(func() { core.Tunnels.CloseIf(tunnel) })

	armEntered := make(chan struct{}, 1)
	releaseArm := make(chan struct{})
	var releaseOnce sync.Once
	release := func() { releaseOnce.Do(func() { close(releaseArm) }) }
	t.Cleanup(release)
	terminal := marshalTunnelData(t, &sliverpb.TunnelData{TunnelID: tunnel.ID, Sequence: 1, Closed: true})
	handlerDone := make(chan struct{})
	go func() {
		tunnelCloseHandlerWithTerminalArm(
			connection,
			terminal,
			func(*core.Tunnel) {
				armEntered <- struct{}{}
				<-releaseArm
			},
		)
		close(handlerDone)
	}()
	select {
	case <-armEntered:
	case <-time.After(time.Second):
		t.Fatal("terminal deadline arm was not entered")
	}

	// While the arm callback is paused, an at-boundary frame is retained. If
	// terminal marking had run first, this frame would already be a protocol
	// violation and would close the connection before the arm is released.
	tunnelDataHandler(connection, marshalTunnelData(t, &sliverpb.TunnelData{
		TunnelID: tunnel.ID,
		Sequence: 1,
		Data:     []byte("at terminal boundary"),
	}))
	select {
	case <-connection.Done():
		t.Fatal("terminal was marked before its deadline actor was armed")
	default:
	}

	release()
	select {
	case <-handlerDone:
	case <-time.After(time.Second):
		t.Fatal("terminal handler did not return after deadline arm was released")
	}
	select {
	case <-connection.Done():
	case <-time.After(time.Second):
		t.Fatal("retained at-boundary frame did not fail terminal validation closed")
	}
}

func TestCapableTunnelCloseRejectsContradictoryTerminalState(t *testing.T) {
	tests := []struct {
		name string
		run  func(*testing.T, *core.ImplantConnection, *core.Tunnel)
	}{
		{
			name: "conflicting terminal",
			run: func(t *testing.T, connection *core.ImplantConnection, tunnel *core.Tunnel) {
				tunnelCloseHandler(connection, marshalTunnelData(t, &sliverpb.TunnelData{TunnelID: tunnel.ID, Sequence: 2, Closed: true}))
				tunnelCloseHandler(connection, marshalTunnelData(t, &sliverpb.TunnelData{TunnelID: tunnel.ID, Sequence: 3, Closed: true}))
			},
		},
		{
			name: "terminal outside receive window",
			run: func(t *testing.T, connection *core.ImplantConnection, tunnel *core.Tunnel) {
				tunnelCloseHandler(connection, marshalTunnelData(t, &sliverpb.TunnelData{TunnelID: tunnel.ID, Sequence: 129, Closed: true}))
			},
		},
		{
			name: "retained data at terminal",
			run: func(t *testing.T, connection *core.ImplantConnection, tunnel *core.Tunnel) {
				tunnelDataHandler(connection, marshalTunnelData(t, &sliverpb.TunnelData{TunnelID: tunnel.ID, Sequence: 2, Data: []byte("at-terminal")}))
				tunnelCloseHandler(connection, marshalTunnelData(t, &sliverpb.TunnelData{TunnelID: tunnel.ID, Sequence: 2, Closed: true}))
			},
		},
		{
			name: "data after accepted terminal",
			run: func(t *testing.T, connection *core.ImplantConnection, tunnel *core.Tunnel) {
				tunnelCloseHandler(connection, marshalTunnelData(t, &sliverpb.TunnelData{TunnelID: tunnel.ID, Sequence: 2, Closed: true}))
				tunnelDataHandler(connection, marshalTunnelData(t, &sliverpb.TunnelData{TunnelID: tunnel.ID, Sequence: 2, Data: []byte("after-terminal")}))
			},
		},
		{
			name: "terminal carries payload",
			run: func(t *testing.T, connection *core.ImplantConnection, tunnel *core.Tunnel) {
				tunnelCloseHandler(connection, marshalTunnelData(t, &sliverpb.TunnelData{TunnelID: tunnel.ID, Sequence: 1, Closed: true, Data: []byte("payload")}))
			},
		},
		{
			name: "conflicting pending data",
			run: func(t *testing.T, connection *core.ImplantConnection, tunnel *core.Tunnel) {
				tunnelDataHandler(connection, marshalTunnelData(t, &sliverpb.TunnelData{TunnelID: tunnel.ID, Sequence: 1, Data: []byte("first")}))
				tunnelDataHandler(connection, marshalTunnelData(t, &sliverpb.TunnelData{TunnelID: tunnel.ID, Sequence: 1, Data: []byte("conflict")}))
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			connection, session := addTestSession(t)
			session.Capabilities = sliverpb.CapabilityTunnelTerminalV1
			tunnel, err := core.Tunnels.Create(session.ID)
			if err != nil {
				t.Fatalf("create capable generic tunnel: %v", err)
			}
			t.Cleanup(func() { core.Tunnels.CloseIf(tunnel) })

			test.run(t, connection, tunnel)
			select {
			case <-connection.Done():
			case <-time.After(time.Second):
				t.Fatal("contradictory terminal state did not close implant connection")
			}
			select {
			case <-tunnel.Done():
			case <-time.After(time.Second):
				t.Fatal("contradictory terminal state did not close exact tunnel")
			}
			if got := core.Tunnels.Get(tunnel.ID); got != nil {
				t.Fatalf("contradictory terminal state retained tunnel: %p", got)
			}
		})
	}
}

func TestTunnelCloseHandlerPreservesReorderedGenericData(t *testing.T) {
	connection, session := addTestSession(t)
	tunnel, err := core.Tunnels.Create(session.ID)
	if err != nil {
		t.Fatalf("create generic tunnel: %v", err)
	}
	t.Cleanup(func() { core.Tunnels.CloseIf(tunnel) })

	// Model independently-dispatched mTLS envelopes where the terminal close
	// overtakes the final two implant-to-client frames.
	tunnelCloseHandler(connection, marshalTunnelData(t, &sliverpb.TunnelData{
		TunnelID: tunnel.ID,
		Sequence: 2,
		Closed:   true,
	}))
	tunnelDataHandler(connection, marshalTunnelData(t, &sliverpb.TunnelData{
		TunnelID: tunnel.ID,
		Sequence: 1,
		Data:     []byte("second"),
	}))

	received := make(chan []*sliverpb.TunnelData, 1)
	go func() {
		frames := make([]*sliverpb.TunnelData, 0, 2)
		for range 2 {
			frames = append(frames, <-tunnel.FromImplant)
		}
		received <- frames
	}()
	tunnelDataHandler(connection, marshalTunnelData(t, &sliverpb.TunnelData{
		TunnelID: tunnel.ID,
		Sequence: 0,
		Data:     []byte("first"),
	}))

	select {
	case frames := <-received:
		if len(frames) != 2 || frames[0].Sequence != 0 || string(frames[0].Data) != "first" ||
			frames[1].Sequence != 1 || string(frames[1].Data) != "second" {
			t.Fatalf("reordered frames = %+v, want sequences 0 then 1", frames)
		}
	case <-time.After(time.Second):
		t.Fatal("implant close grace did not preserve reordered generic data")
	}
	if got := core.Tunnels.Get(tunnel.ID); got != tunnel {
		t.Fatalf("implant close grace detached tunnel before final data drained: got=%p want=%p", got, tunnel)
	}
}

type countingTunnelWriteCloser struct {
	writes atomic.Int32
	closes atomic.Int32
}

func (writer *countingTunnelWriteCloser) Write(data []byte) (int, error) {
	writer.writes.Add(1)
	return len(data), nil
}

func (writer *countingTunnelWriteCloser) Close() error {
	writer.closes.Add(1)
	return nil
}

func TestReverseTunnelRejectsResendControlWithoutRelayingPayload(t *testing.T) {
	connection := core.NewImplantConnection("test", "n/a")
	t.Cleanup(connection.Close)
	tunnelID := core.NewTunnelID()
	writer := &countingTunnelWriteCloser{}
	tunnel := rtunnels.NewRTunnel(tunnelID, "session", writer)
	if !rtunnels.TryAddRTunnel(tunnel) {
		t.Fatalf("failed to register reverse tunnel %d", tunnelID)
	}
	t.Cleanup(func() {
		if rtunnels.RemoveRTunnelIf(tunnelID, tunnel) {
			tunnel.Close()
		}
	})

	RTunnelDataHandler(&sliverpb.TunnelData{
		TunnelID: tunnelID,
		Sequence: 0,
		Resend:   true,
		Data:     []byte("must-not-reach-the-authorized-destination"),
	}, tunnel, connection)

	if got := writer.writes.Load(); got != 0 {
		t.Fatalf("resend control performed %d destination writes", got)
	}
	if got := writer.closes.Load(); got != 1 {
		t.Fatalf("resend control closed destination %d times, want 1", got)
	}
	if got := tunnel.ReadSequence(); got != 0 {
		t.Fatalf("resend control advanced reverse read sequence to %d", got)
	}
	if active := rtunnels.GetRTunnel(tunnelID); active != nil {
		t.Fatalf("resend control left reverse tunnel %d published", tunnelID)
	}
}

type blockedHandlerDial struct {
	address string
	peer    chan net.Conn
}

type blockedHandlerDialer struct {
	started     chan blockedHandlerDial
	calls       atomic.Int32
	lateSuccess bool
	peerMutex   sync.Mutex
	peers       []net.Conn
}

type gatedHandlerDialer struct {
	started     chan struct{}
	release     chan struct{}
	startedOnce sync.Once
	releaseOnce sync.Once
	peerMutex   sync.Mutex
	peers       []net.Conn
	received    atomic.Int64
	drains      sync.WaitGroup
}

func newGatedHandlerDialer() *gatedHandlerDialer {
	return &gatedHandlerDialer{
		started: make(chan struct{}),
		release: make(chan struct{}),
	}
}

func (dialer *gatedHandlerDialer) DialContext(ctx context.Context, _ string, _ string) (net.Conn, error) {
	dialer.startedOnce.Do(func() { close(dialer.started) })
	select {
	case <-dialer.release:
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	connection, peer := net.Pipe()
	dialer.peerMutex.Lock()
	dialer.peers = append(dialer.peers, peer)
	dialer.peerMutex.Unlock()
	dialer.drains.Add(1)
	go func() {
		defer dialer.drains.Done()
		count, _ := io.Copy(io.Discard, peer)
		dialer.received.Add(count)
	}()
	return connection, nil
}

func (dialer *gatedHandlerDialer) allow() {
	dialer.releaseOnce.Do(func() { close(dialer.release) })
}

func (dialer *gatedHandlerDialer) close() {
	dialer.allow()
	dialer.peerMutex.Lock()
	peers := dialer.peers
	dialer.peers = nil
	dialer.peerMutex.Unlock()
	for _, peer := range peers {
		_ = peer.Close()
	}
	dialer.drains.Wait()
}

func newBlockedHandlerDialer(lateSuccess bool) *blockedHandlerDialer {
	return &blockedHandlerDialer{
		started:     make(chan blockedHandlerDial, 16),
		lateSuccess: lateSuccess,
	}
}

func (dialer *blockedHandlerDialer) DialContext(ctx context.Context, _ string, address string) (net.Conn, error) {
	dialer.calls.Add(1)
	attempt := blockedHandlerDial{address: address, peer: make(chan net.Conn, 1)}
	dialer.started <- attempt
	<-ctx.Done()
	if !dialer.lateSuccess {
		return nil, ctx.Err()
	}
	connection, peer := net.Pipe()
	dialer.peerMutex.Lock()
	dialer.peers = append(dialer.peers, peer)
	dialer.peerMutex.Unlock()
	attempt.peer <- peer
	return connection, nil
}

func (dialer *blockedHandlerDialer) close() {
	dialer.peerMutex.Lock()
	peers := dialer.peers
	dialer.peers = nil
	dialer.peerMutex.Unlock()
	for _, peer := range peers {
		_ = peer.Close()
	}
}

func installHandlerBroker(t *testing.T, registry *rtunnels.Registry, dialer rtunnels.ContextDialer) {
	t.Helper()
	previousRegistry := rtunnels.DefaultRegistry
	previousBroker := rtunnels.DefaultBroker
	rtunnels.DefaultRegistry = registry
	rtunnels.DefaultBroker = rtunnels.NewBroker(registry, dialer, time.Second)
	t.Cleanup(func() {
		rtunnels.DefaultBroker = previousBroker
		rtunnels.DefaultRegistry = previousRegistry
	})
	if closer, ok := dialer.(interface{ close() }); ok {
		t.Cleanup(closer.close)
	}
}

type handlerTestWorkers struct {
	waitGroup sync.WaitGroup
}

func (workers *handlerTestWorkers) launch(worker func()) <-chan struct{} {
	done := make(chan struct{})
	workers.waitGroup.Add(1)
	go func() {
		defer workers.waitGroup.Done()
		defer close(done)
		worker()
	}()
	return done
}

func (workers *handlerTestWorkers) wait(t *testing.T) {
	t.Helper()
	finished := make(chan struct{})
	go func() {
		workers.waitGroup.Wait()
		close(finished)
	}()
	deadline := time.NewTimer(2 * time.Second)
	ticker := time.NewTicker(time.Millisecond)
	defer deadline.Stop()
	defer ticker.Stop()
	for {
		select {
		case <-finished:
			return
		case <-ticker.C:
			reverseTunnelOpenings.Range(func(_, value any) bool {
				if opening, ok := value.(*reverseTunnelOpening); ok {
					opening.requestClose()
				}
				return true
			})
		case <-deadline.C:
			t.Error("timed out waiting for reverse-tunnel handler test workers")
			return
		}
	}
}

func installHandlerAdmissions(t *testing.T, attempts *reverseTunnelAdmission, waiters *reverseTunnelAdmission) *handlerTestWorkers {
	t.Helper()
	workers := &handlerTestWorkers{}
	previousAttempts := reverseTunnelOpeningAttempts
	previousWaiters := reverseTunnelOpeningWaiters
	reverseTunnelOpeningAttempts = attempts
	reverseTunnelOpeningWaiters = waiters
	t.Cleanup(func() {
		workers.wait(t)
		deadline := time.Now().Add(2 * time.Second)
		for time.Now().Before(deadline) {
			if reverseTunnelOpeningCount() == 0 && handlerAdmissionEmpty(attempts) && handlerAdmissionEmpty(waiters) {
				break
			}
			time.Sleep(time.Millisecond)
		}
		openingCount := reverseTunnelOpeningCount()
		attemptsEmpty := handlerAdmissionEmpty(attempts)
		waitersEmpty := handlerAdmissionEmpty(waiters)
		reverseTunnelOpeningAttempts = previousAttempts
		reverseTunnelOpeningWaiters = previousWaiters
		if openingCount != 0 || !attemptsEmpty || !waitersEmpty {
			t.Errorf("handler cleanup retained openings=%d attemptsEmpty=%v waitersEmpty=%v", openingCount, attemptsEmpty, waitersEmpty)
		}
	})
	return workers
}

func reverseTunnelOpeningCount() int {
	count := 0
	reverseTunnelOpenings.Range(func(_, _ any) bool {
		count++
		return true
	})
	return count
}

func handlerAdmissionEmpty(admission *reverseTunnelAdmission) bool {
	admission.mutex.Lock()
	defer admission.mutex.Unlock()
	return admission.total == 0 && len(admission.perSession) == 0
}

func assertReverseTunnelOpeningsEmpty(t *testing.T) {
	t.Helper()
	count := 0
	reverseTunnelOpenings.Range(func(_, _ any) bool {
		count++
		return true
	})
	if count != 0 {
		t.Fatalf("reverse tunnel openings retained %d entries", count)
	}
}

func assertHandlerAdmissionEmpty(t *testing.T, admission *reverseTunnelAdmission) {
	t.Helper()
	admission.mutex.Lock()
	defer admission.mutex.Unlock()
	if admission.total != 0 || len(admission.perSession) != 0 {
		t.Fatalf("admission retained total=%d per-session=%v", admission.total, admission.perSession)
	}
}

func marshalTunnelData(t *testing.T, tunnelData *sliverpb.TunnelData) []byte {
	t.Helper()
	data, err := proto.Marshal(tunnelData)
	if err != nil {
		t.Fatalf("marshal tunnel data: %v", err)
	}
	return data
}

func reverseCreateData(t *testing.T, tunnelID uint64, authorizationID rtunnels.AuthorizationID) []byte {
	t.Helper()
	return marshalTunnelData(t, &sliverpb.TunnelData{
		TunnelID:      tunnelID,
		CreateReverse: true,
		Rportfwd: &sliverpb.RPortfwd{
			AuthorizationID: authorizationID.String(),
		},
	})
}

func receiveTunnelRejection(t *testing.T, connection *core.ImplantConnection, tunnelID uint64) {
	t.Helper()
	select {
	case envelope := <-connection.Send:
		if envelope.Type != sliverpb.MsgTunnelClose {
			t.Fatalf("rejection envelope type = %d, want %d", envelope.Type, sliverpb.MsgTunnelClose)
		}
		closed := &sliverpb.TunnelData{}
		if err := proto.Unmarshal(envelope.Data, closed); err != nil {
			t.Fatalf("unmarshal rejection: %v", err)
		}
		if !closed.Closed || closed.TunnelID != tunnelID {
			t.Fatalf("rejection = %+v, want closed tunnel %d", closed, tunnelID)
		}
	case <-time.After(time.Second):
		t.Fatalf("timed out waiting for tunnel %d rejection", tunnelID)
	}
}

func waitHandler(t *testing.T, done <-chan struct{}, name string) {
	t.Helper()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatalf("%s did not finish", name)
	}
}

func receiveBlockedHandlerDial(t *testing.T, dialer *blockedHandlerDialer) blockedHandlerDial {
	t.Helper()
	select {
	case attempt := <-dialer.started:
		return attempt
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for blocked dial")
		return blockedHandlerDial{}
	}
}

func receiveBlockedHandlerPeer(t *testing.T, attempt blockedHandlerDial) net.Conn {
	t.Helper()
	select {
	case peer := <-attempt.peer:
		return peer
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for late dial connection")
		return nil
	}
}

func TestTunnelCloseCancelsOpeningAndRejectsLateDialConnection(t *testing.T) {
	registry := rtunnels.NewRegistry()
	dialer := newBlockedHandlerDialer(true)
	installHandlerBroker(t, registry, dialer)
	attempts := newReverseTunnelAdmission(2, 2)
	waiters := newReverseTunnelAdmission(2, 2)
	workers := installHandlerAdmissions(t, attempts, waiters)
	connection, session := addBufferedTestSession(t)
	authorizationID := beginActiveAuthorization(t, registry, session.ID, "127.0.0.1:4444", 1)
	const tunnelID = uint64(0x4101)
	request := reverseCreateData(t, tunnelID, authorizationID)

	createDone := workers.launch(func() {
		tunnelDataHandler(connection, request)
	})
	attempt := receiveBlockedHandlerDial(t, dialer)
	if attempt.address != "127.0.0.1:4444" {
		t.Fatalf("dial address = %q", attempt.address)
	}
	closeData := marshalTunnelCloseData(t, tunnelID)
	closeDone := workers.launch(func() {
		tunnelCloseHandler(connection, closeData)
	})

	peer := receiveBlockedHandlerPeer(t, attempt)
	t.Cleanup(func() { _ = peer.Close() })
	_ = peer.SetReadDeadline(time.Now().Add(time.Second))
	count, err := peer.Read(make([]byte, 1))
	if networkErr, ok := err.(net.Error); ok && networkErr.Timeout() {
		t.Fatalf("late dial connection remained open until deadline: %v", err)
	}
	if count != 0 || !errors.Is(err, io.EOF) {
		t.Fatalf("late dial read = (%d, %v), want (0, EOF)", count, err)
	}
	waitHandler(t, createDone, "create handler")
	waitHandler(t, closeDone, "close handler")
	receiveTunnelRejection(t, connection, tunnelID)
	if tunnel := rtunnels.GetRTunnel(tunnelID); tunnel != nil {
		closeTestReverseTunnel(tunnelID)
		t.Fatal("canceled opening registered a reverse tunnel")
	}
	assertReverseTunnelOpeningsEmpty(t)
	assertHandlerAdmissionEmpty(t, attempts)
	assertHandlerAdmissionEmpty(t, waiters)
}

func TestDuplicateReverseTunnelCreateReturnsWithoutSecondDial(t *testing.T) {
	registry := rtunnels.NewRegistry()
	dialer := newBlockedHandlerDialer(false)
	installHandlerBroker(t, registry, dialer)
	attempts := newReverseTunnelAdmission(2, 2)
	waiters := newReverseTunnelAdmission(2, 2)
	workers := installHandlerAdmissions(t, attempts, waiters)
	connection, session := addBufferedTestSession(t)
	authorizationID := beginActiveAuthorization(t, registry, session.ID, "127.0.0.1:4445", 2)
	const tunnelID = uint64(0x4102)
	request := reverseCreateData(t, tunnelID, authorizationID)

	firstDone := workers.launch(func() {
		tunnelDataHandler(connection, request)
	})
	receiveBlockedHandlerDial(t, dialer)
	duplicateDone := workers.launch(func() {
		tunnelDataHandler(connection, request)
	})
	waitHandler(t, duplicateDone, "duplicate create handler")
	if got := dialer.calls.Load(); got != 1 {
		t.Fatalf("duplicate create performed %d dials, want 1", got)
	}
	value, ok := reverseTunnelOpenings.Load(tunnelID)
	if !ok {
		t.Fatal("blocked opening disappeared before cancellation")
	}
	if got := len(value.(*reverseTunnelOpening).waiters); got != 0 {
		t.Fatalf("duplicate create added %d opening waiters", got)
	}

	closeData := marshalTunnelCloseData(t, tunnelID)
	closeDone := workers.launch(func() {
		tunnelCloseHandler(connection, closeData)
	})
	waitHandler(t, firstDone, "first create handler")
	waitHandler(t, closeDone, "close handler")
	receiveTunnelRejection(t, connection, tunnelID)
	assertReverseTunnelOpeningsEmpty(t)
	assertHandlerAdmissionEmpty(t, attempts)
	assertHandlerAdmissionEmpty(t, waiters)
}

func TestReverseTunnelOpeningAttemptLimitsProductionHandler(t *testing.T) {
	registry := rtunnels.NewRegistry()
	dialer := newBlockedHandlerDialer(false)
	installHandlerBroker(t, registry, dialer)
	attempts := newReverseTunnelAdmission(1, 2)
	waiters := newReverseTunnelAdmission(4, 8)
	workers := installHandlerAdmissions(t, attempts, waiters)

	connectionA, sessionA := addBufferedTestSession(t)
	connectionB, sessionB := addBufferedTestSession(t)
	connectionC, sessionC := addBufferedTestSession(t)
	authorizationA1 := beginActiveAuthorization(t, registry, sessionA.ID, "127.0.0.1:4501", 11)
	authorizationA2 := beginActiveAuthorization(t, registry, sessionA.ID, "127.0.0.1:4502", 12)
	authorizationB := beginActiveAuthorization(t, registry, sessionB.ID, "127.0.0.1:4503", 13)
	authorizationC := beginActiveAuthorization(t, registry, sessionC.ID, "127.0.0.1:4504", 14)

	type activeOpening struct {
		connection *core.ImplantConnection
		tunnelID   uint64
		done       <-chan struct{}
	}
	active := []activeOpening{
		{connection: connectionA, tunnelID: 0x4201},
		{connection: connectionB, tunnelID: 0x4202},
	}
	requests := [][]byte{
		reverseCreateData(t, active[0].tunnelID, authorizationA1),
		reverseCreateData(t, active[1].tunnelID, authorizationB),
	}
	for index := range active {
		active[index].done = workers.launch(func() {
			tunnelDataHandler(active[index].connection, requests[index])
		})
		receiveBlockedHandlerDial(t, dialer)
	}

	const rejectedPerSession = uint64(0x4203)
	const rejectedGlobal = uint64(0x4204)
	tunnelDataHandler(connectionA, reverseCreateData(t, rejectedPerSession, authorizationA2))
	tunnelDataHandler(connectionC, reverseCreateData(t, rejectedGlobal, authorizationC))
	receiveTunnelRejection(t, connectionA, rejectedPerSession)
	receiveTunnelRejection(t, connectionC, rejectedGlobal)
	if got := dialer.calls.Load(); got != 2 {
		t.Fatalf("opening admission allowed %d dials, want 2", got)
	}
	if _, ok := reverseTunnelOpenings.Load(rejectedPerSession); ok {
		t.Fatal("per-session rejected opening was published")
	}
	if _, ok := reverseTunnelOpenings.Load(rejectedGlobal); ok {
		t.Fatal("globally rejected opening was published")
	}

	for _, opening := range active {
		closeData := marshalTunnelCloseData(t, opening.tunnelID)
		closeDone := workers.launch(func() {
			tunnelCloseHandler(opening.connection, closeData)
		})
		waitHandler(t, opening.done, "admitted create handler")
		waitHandler(t, closeDone, "admitted close handler")
		receiveTunnelRejection(t, opening.connection, opening.tunnelID)
	}
	assertReverseTunnelOpeningsEmpty(t)
	assertHandlerAdmissionEmpty(t, attempts)
	assertHandlerAdmissionEmpty(t, waiters)
}

func TestReverseTunnelOpeningWaitTimeoutReclaimsQuota(t *testing.T) {
	registry := rtunnels.NewRegistry()
	dialer := newBlockedHandlerDialer(false)
	installHandlerBroker(t, registry, dialer)
	attempts := newReverseTunnelAdmission(1, 1)
	waiters := newReverseTunnelAdmission(1, 1)
	workers := installHandlerAdmissions(t, attempts, waiters)
	previousTimeout := reverseTunnelOpeningWaitTimeout
	reverseTunnelOpeningWaitTimeout = 20 * time.Millisecond
	t.Cleanup(func() { reverseTunnelOpeningWaitTimeout = previousTimeout })
	connection, session := addBufferedTestSession(t)
	authorizationID := beginActiveAuthorization(t, registry, session.ID, "127.0.0.1:4601", 21)
	const tunnelID = uint64(0x4301)
	request := reverseCreateData(t, tunnelID, authorizationID)

	createDone := workers.launch(func() {
		tunnelDataHandler(connection, request)
	})
	receiveBlockedHandlerDial(t, dialer)
	tunnelDataHandler(connection, marshalTunnelData(t, &sliverpb.TunnelData{TunnelID: tunnelID, Sequence: 1, Data: []byte("wait")}))
	waitHandler(t, createDone, "timed-out create handler")
	receiveTunnelRejection(t, connection, tunnelID)
	receiveTunnelRejection(t, connection, tunnelID)
	assertReverseTunnelOpeningsEmpty(t)
	assertHandlerAdmissionEmpty(t, attempts)
	assertHandlerAdmissionEmpty(t, waiters)

	opening := newReverseTunnelOpening(session.ID, func() {})
	close(opening.ready)
	if err := opening.wait(time.Second); err != nil {
		t.Fatalf("reclaimed waiter slot was not reusable: %v", err)
	}
	assertHandlerAdmissionEmpty(t, waiters)
}

func TestRejectReverseTunnelSaturationClosesConnectionAndReclaimsSlot(t *testing.T) {
	previousSlots := reverseTunnelRejectionSlots
	reverseTunnelRejectionSlots = make(chan struct{}, 1)
	connection := core.NewImplantConnection("test", "n/a")
	t.Cleanup(connection.Close)
	t.Cleanup(func() {
		connection.Close()
		deadline := time.Now().Add(time.Second)
		for len(reverseTunnelRejectionSlots) != 0 && time.Now().Before(deadline) {
			time.Sleep(time.Millisecond)
		}
		reverseTunnelRejectionSlots = previousSlots
	})

	rejectReverseTunnel(connection, 0x4401, errReverseTunnelOpeningLimit)
	deadline := time.Now().Add(time.Second)
	for len(reverseTunnelRejectionSlots) != 1 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if got := len(reverseTunnelRejectionSlots); got != 1 {
		t.Fatalf("first rejection occupied %d slots, want 1", got)
	}
	rejectReverseTunnel(connection, 0x4402, errReverseTunnelOpeningLimit)
	select {
	case <-connection.Done():
	case <-time.After(time.Second):
		t.Fatal("rejection saturation did not close implant connection")
	}
	deadline = time.Now().Add(time.Second)
	for len(reverseTunnelRejectionSlots) != 0 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if got := len(reverseTunnelRejectionSlots); got != 0 {
		t.Fatalf("rejection close retained %d slots", got)
	}
}
