package handlers

import (
	"bytes"
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
)

const reverseTunnelWriterChunkSize = 32 * 1024

type exactReversePayloadDialer struct {
	expected int
	calls    atomic.Int32
	received chan []byte
	mutex    sync.Mutex
	peers    []net.Conn
	waiters  sync.WaitGroup
}

type blockedReverseWriteConn struct {
	writeStarted chan struct{}
	writeOnce    sync.Once
	closed       chan struct{}
	closeOnce    sync.Once
}

func newBlockedReverseWriteConn() *blockedReverseWriteConn {
	return &blockedReverseWriteConn{writeStarted: make(chan struct{}), closed: make(chan struct{})}
}

func (connection *blockedReverseWriteConn) Read(_ []byte) (int, error) {
	<-connection.closed
	return 0, io.EOF
}

func (connection *blockedReverseWriteConn) Write(_ []byte) (int, error) {
	connection.writeOnce.Do(func() { close(connection.writeStarted) })
	<-connection.closed
	return 0, net.ErrClosed
}

func (connection *blockedReverseWriteConn) Close() error {
	connection.closeOnce.Do(func() { close(connection.closed) })
	return nil
}

func (*blockedReverseWriteConn) LocalAddr() net.Addr              { return reverseTestAddr("local") }
func (*blockedReverseWriteConn) RemoteAddr() net.Addr             { return reverseTestAddr("remote") }
func (*blockedReverseWriteConn) SetDeadline(time.Time) error      { return nil }
func (*blockedReverseWriteConn) SetReadDeadline(time.Time) error  { return nil }
func (*blockedReverseWriteConn) SetWriteDeadline(time.Time) error { return nil }

type reverseTestAddr string

func (address reverseTestAddr) Network() string { return "test" }
func (address reverseTestAddr) String() string  { return string(address) }

type delayedReverseWriteConn struct {
	delay     time.Duration
	started   atomic.Int32
	completed atomic.Int32
	closed    chan struct{}
	closeOnce sync.Once
}

func newDelayedReverseWriteConn(delay time.Duration) *delayedReverseWriteConn {
	return &delayedReverseWriteConn{delay: delay, closed: make(chan struct{})}
}

func (connection *delayedReverseWriteConn) Read(_ []byte) (int, error) {
	<-connection.closed
	return 0, io.EOF
}

func (connection *delayedReverseWriteConn) Write(data []byte) (int, error) {
	connection.started.Add(1)
	timer := time.NewTimer(connection.delay)
	defer timer.Stop()
	select {
	case <-timer.C:
		connection.completed.Add(1)
		return len(data), nil
	case <-connection.closed:
		return 0, net.ErrClosed
	}
}

func (connection *delayedReverseWriteConn) Close() error {
	connection.closeOnce.Do(func() { close(connection.closed) })
	return nil
}

func (*delayedReverseWriteConn) LocalAddr() net.Addr              { return reverseTestAddr("local") }
func (*delayedReverseWriteConn) RemoteAddr() net.Addr             { return reverseTestAddr("remote") }
func (*delayedReverseWriteConn) SetDeadline(time.Time) error      { return nil }
func (*delayedReverseWriteConn) SetReadDeadline(time.Time) error  { return nil }
func (*delayedReverseWriteConn) SetWriteDeadline(time.Time) error { return nil }

type blockedReverseWriteDialer struct {
	calls      atomic.Int32
	connection net.Conn
}

func (dialer *blockedReverseWriteDialer) DialContext(context.Context, string, string) (net.Conn, error) {
	dialer.calls.Add(1)
	return dialer.connection, nil
}

func (dialer *blockedReverseWriteDialer) close() {
	if dialer.connection != nil {
		_ = dialer.connection.Close()
	}
}

func newExactReversePayloadDialer(expected int) *exactReversePayloadDialer {
	return &exactReversePayloadDialer{expected: expected, received: make(chan []byte, 1)}
}

func (dialer *exactReversePayloadDialer) DialContext(_ context.Context, _ string, _ string) (net.Conn, error) {
	dialer.calls.Add(1)
	connection, peer := net.Pipe()
	dialer.mutex.Lock()
	dialer.peers = append(dialer.peers, peer)
	dialer.mutex.Unlock()
	dialer.waiters.Add(1)
	go func() {
		defer dialer.waiters.Done()
		payload := make([]byte, dialer.expected)
		if _, err := io.ReadFull(peer, payload); err != nil {
			dialer.received <- nil
			return
		}
		dialer.received <- payload
		_, _ = io.Copy(io.Discard, peer)
	}()
	return connection, nil
}

func (dialer *exactReversePayloadDialer) close() {
	dialer.mutex.Lock()
	peers := dialer.peers
	dialer.peers = nil
	dialer.mutex.Unlock()
	for _, peer := range peers {
		_ = peer.Close()
	}
	dialer.waiters.Wait()
}

func installReverseTunnelPreCreateRegistry(t *testing.T, ttl time.Duration) *reverseTunnelPreCreateRegistry {
	t.Helper()
	registry := newReverseTunnelPreCreateRegistry(ttl)
	previous := reverseTunnelPreCreates
	reverseTunnelPreCreates = registry
	t.Cleanup(func() {
		deadline := time.Now().Add(time.Second)
		for time.Now().Before(deadline) {
			registry.mutex.Lock()
			empty := len(registry.entries) == 0 && len(registry.perSession) == 0 && registry.total == (reverseTunnelPreCreateUsage{})
			registry.mutex.Unlock()
			if empty {
				break
			}
			time.Sleep(time.Millisecond)
		}
		registry.mutex.Lock()
		entries := len(registry.entries)
		sessions := len(registry.perSession)
		total := registry.total
		registry.mutex.Unlock()
		reverseTunnelPreCreates = previous
		if entries != 0 || sessions != 0 || total != (reverseTunnelPreCreateUsage{}) {
			t.Errorf("pre-create cleanup retained entries=%d sessions=%d total=%+v", entries, sessions, total)
		}
	})
	return registry
}

func reverseTunnelPreCreateSnapshot(registry *reverseTunnelPreCreateRegistry) (int, int, reverseTunnelPreCreateUsage) {
	registry.mutex.Lock()
	defer registry.mutex.Unlock()
	return len(registry.entries), len(registry.perSession), registry.total
}

func waitForReverseTunnelPreCreateEmpty(t *testing.T, registry *reverseTunnelPreCreateRegistry) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		entries, sessions, usage := reverseTunnelPreCreateSnapshot(registry)
		if entries == 0 && sessions == 0 && usage == (reverseTunnelPreCreateUsage{}) {
			return
		}
		time.Sleep(time.Millisecond)
	}
	entries, sessions, usage := reverseTunnelPreCreateSnapshot(registry)
	t.Fatalf("pre-create registry retained entries=%d sessions=%d total=%+v", entries, sessions, usage)
}

func waitForReverseTunnelAdmissionTotal(t *testing.T, admission *reverseTunnelAdmission, want int) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		admission.mutex.Lock()
		total := admission.total
		admission.mutex.Unlock()
		if total == want {
			return
		}
		time.Sleep(time.Millisecond)
	}
	admission.mutex.Lock()
	total := admission.total
	admission.mutex.Unlock()
	t.Fatalf("reverse tunnel opening admission total = %d, want %d", total, want)
}

func deterministicReversePayload(size int) []byte {
	payload := make([]byte, size)
	for index := range payload {
		payload[index] = byte((index*131 + 17) % 251)
	}
	return payload
}

func splitReversePayload(payload []byte) [][]byte {
	chunks := make([][]byte, 0, (len(payload)+reverseTunnelWriterChunkSize-1)/reverseTunnelWriterChunkSize)
	for len(payload) > 0 {
		size := min(len(payload), reverseTunnelWriterChunkSize)
		chunks = append(chunks, payload[:size])
		payload = payload[size:]
	}
	return chunks
}

func reverseFollower(tunnelID uint64, sequence uint64, payload []byte) *sliverpb.TunnelData {
	return &sliverpb.TunnelData{
		TunnelID: tunnelID,
		Sequence: sequence,
		Data:     payload,
		Rportfwd: &sliverpb.RPortfwd{},
	}
}

func waitForExactReversePayload(t *testing.T, dialer *exactReversePayloadDialer) []byte {
	t.Helper()
	select {
	case payload := <-dialer.received:
		if payload == nil {
			t.Fatal("reverse target closed before receiving the expected payload")
		}
		return payload
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for reverse target payload")
		return nil
	}
}

func TestReverseTunnelPrecursorsPreserveFollowerFirstWriterBoundaries(t *testing.T) {
	for testIndex, wireBytes := range []int{32769, 65537} {
		t.Run(fmt.Sprintf("wire-bytes-%d", wireBytes), func(t *testing.T) {
			registry := installReverseTunnelPreCreateRegistry(t, time.Second)
			payload := deterministicReversePayload(wireBytes)
			chunks := splitReversePayload(payload)
			dialer := newExactReversePayloadDialer(len(payload))
			brokerRegistry := rtunnels.NewRegistry()
			installHandlerBroker(t, brokerRegistry, dialer)
			connection, session := addBufferedTestSession(t)
			authorizationID := beginActiveAuthorization(t, brokerRegistry, session.ID, "127.0.0.1:4700", uint32(testIndex+1))
			tunnelID := uint64(0x5100 + testIndex)

			// Deliver the exact tails which raced ahead of sequence zero in the
			// native MTLS reproduction, in worst-case reverse sequence order.
			for index := len(chunks) - 1; index >= 1; index-- {
				tunnelDataHandler(connection, marshalTunnelData(t, reverseFollower(tunnelID, uint64(index), chunks[index])))
			}
			entries, sessions, usage := reverseTunnelPreCreateSnapshot(registry)
			if entries != 1 || sessions != 1 || usage.entries != 1 || usage.frames != len(chunks)-1 || usage.bytes != wireBytes-len(chunks[0]) {
				t.Fatalf("staged precursor usage entries=%d sessions=%d total=%+v", entries, sessions, usage)
			}
			if dialer.calls.Load() != 0 {
				t.Fatal("precursor data opened an outbound socket before CreateReverse")
			}

			tunnelDataHandler(connection, marshalTunnelData(t, &sliverpb.TunnelData{
				TunnelID:      tunnelID,
				Sequence:      0,
				Data:          chunks[0],
				CreateReverse: true,
				Rportfwd:      &sliverpb.RPortfwd{AuthorizationID: authorizationID.String()},
			}))

			if got := waitForExactReversePayload(t, dialer); !bytes.Equal(got, payload) {
				t.Fatalf("reverse target payload differs: got %d bytes, want %d", len(got), len(payload))
			}
			if tunnel := rtunnels.GetRTunnel(tunnelID); tunnel == nil {
				t.Fatal("promoted reverse tunnel was not published")
			} else if got := tunnel.ReadSequence(); got != uint64(len(chunks)) {
				t.Fatalf("reverse tunnel read sequence = %d, want %d", got, len(chunks))
			}
			closeTestReverseTunnel(tunnelID)
			entries, sessions, usage = reverseTunnelPreCreateSnapshot(registry)
			if entries != 0 || sessions != 0 || usage != (reverseTunnelPreCreateUsage{}) {
				t.Fatalf("promotion retained precursor quota entries=%d sessions=%d total=%+v", entries, sessions, usage)
			}
		})
	}
}

func TestReverseTunnelPreCreateTerminalDrainsAfterData(t *testing.T) {
	registry := installReverseTunnelPreCreateRegistry(t, time.Second)
	first := []byte("first-")
	second := []byte("second")
	dialer := newExactReversePayloadDialer(len(first) + len(second))
	brokerRegistry := rtunnels.NewRegistry()
	installHandlerBroker(t, brokerRegistry, dialer)
	connection, session := addBufferedTestSession(t)
	session.Capabilities = sliverpb.CapabilityTunnelTerminalV1
	authorizationID := beginActiveAuthorization(t, brokerRegistry, session.ID, "127.0.0.1:4701", 3)
	const tunnelID = uint64(0x5200)

	tunnelDataHandler(connection, marshalTunnelData(t, reverseFollower(tunnelID, 1, second)))
	tunnelCloseHandler(connection, marshalTunnelData(t, &sliverpb.TunnelData{TunnelID: tunnelID, Sequence: 2, Closed: true}))
	tunnelDataHandler(connection, marshalTunnelData(t, &sliverpb.TunnelData{
		TunnelID:      tunnelID,
		Data:          first,
		CreateReverse: true,
		Rportfwd:      &sliverpb.RPortfwd{AuthorizationID: authorizationID.String()},
	}))

	if got := waitForExactReversePayload(t, dialer); !bytes.Equal(got, append(append([]byte(nil), first...), second...)) {
		t.Fatalf("reverse terminal drain payload = %q", got)
	}
	if tunnel := rtunnels.GetRTunnel(tunnelID); tunnel != nil {
		closeTestReverseTunnel(tunnelID)
		t.Fatal("pre-create terminal left the promoted reverse tunnel active")
	}
	entries, sessions, usage := reverseTunnelPreCreateSnapshot(registry)
	if entries != 0 || sessions != 0 || usage != (reverseTunnelPreCreateUsage{}) {
		t.Fatalf("terminal promotion retained precursor quota entries=%d sessions=%d total=%+v", entries, sessions, usage)
	}
}

func TestReverseTunnelPreCreateNeverDialsBeforeAuthorizedCreate(t *testing.T) {
	installReverseTunnelPreCreateRegistry(t, time.Second)
	dialer := newExactReversePayloadDialer(1)
	brokerRegistry := rtunnels.NewRegistry()
	installHandlerBroker(t, brokerRegistry, dialer)
	connection, _ := addBufferedTestSession(t)
	const tunnelID = uint64(0x5300)

	tunnelDataHandler(connection, marshalTunnelData(t, reverseFollower(tunnelID, 1, []byte("x"))))
	if dialer.calls.Load() != 0 {
		t.Fatal("precursor opened an outbound socket")
	}
	tunnelDataHandler(connection, marshalTunnelData(t, &sliverpb.TunnelData{
		TunnelID:      tunnelID,
		Data:          []byte("y"),
		CreateReverse: true,
		Rportfwd:      &sliverpb.RPortfwd{AuthorizationID: "unknown"},
	}))
	if dialer.calls.Load() != 0 {
		t.Fatal("unauthorized create reached the outbound dialer")
	}
	receiveTunnelRejection(t, connection, tunnelID)
	if tunnel := rtunnels.GetRTunnel(tunnelID); tunnel != nil {
		closeTestReverseTunnel(tunnelID)
		t.Fatal("unauthorized precursor promotion published a reverse tunnel")
	}
}

func TestReverseTunnelPreCreateKeyIncludesConnectionGeneration(t *testing.T) {
	registry := newReverseTunnelPreCreateRegistry(time.Second)
	firstConnection, firstSession := addBufferedTestSession(t)
	secondConnection, secondSession := addBufferedTestSession(t)
	const tunnelID = uint64(0x5400)

	if result, err := registry.stageData(firstConnection, firstSession.ID, reverseFollower(tunnelID, 1, []byte("first"))); result != reverseTunnelPreCreateStaged || err != nil {
		t.Fatalf("stage first precursor = (%v, %v)", result, err)
	}
	if result, err := registry.stageData(secondConnection, secondSession.ID, reverseFollower(tunnelID, 1, []byte("second"))); result != reverseTunnelPreCreateStaged || err != nil {
		t.Fatalf("stage second precursor = (%v, %v)", result, err)
	}

	firstOpening := newReverseTunnelOpening(firstSession.ID, firstConnection, func() {})
	first, _, loaded, err := registry.promote(firstConnection, firstSession.ID, tunnelID, firstOpening)
	if err != nil {
		t.Fatalf("first promotion failed: %v", err)
	}
	if loaded || first == nil || len(first.frames) != 1 || !bytes.Equal(first.frames[1].Data, []byte("first")) {
		t.Fatalf("first promotion selected wrong generation: loaded=%v entry=%+v", loaded, first)
	}
	close(firstOpening.ready)
	reverseTunnelOpenings.CompareAndDelete(tunnelID, firstOpening)
	registry.release(first)

	secondOpening := newReverseTunnelOpening(secondSession.ID, secondConnection, func() {})
	second, _, loaded, err := registry.promote(secondConnection, secondSession.ID, tunnelID, secondOpening)
	if err != nil {
		t.Fatalf("second promotion failed: %v", err)
	}
	if loaded || second == nil || len(second.frames) != 1 || !bytes.Equal(second.frames[1].Data, []byte("second")) {
		t.Fatalf("second promotion selected wrong generation: loaded=%v entry=%+v", loaded, second)
	}
	close(secondOpening.ready)
	reverseTunnelOpenings.CompareAndDelete(tunnelID, secondOpening)
	registry.release(second)

	entries, sessions, usage := reverseTunnelPreCreateSnapshot(registry)
	if entries != 0 || sessions != 0 || usage != (reverseTunnelPreCreateUsage{}) {
		t.Fatalf("exact-generation test retained entries=%d sessions=%d total=%+v", entries, sessions, usage)
	}
}

func TestReverseTunnelPreCreateExpiryTombstonesLateCreate(t *testing.T) {
	registry := installReverseTunnelPreCreateRegistry(t, 20*time.Millisecond)
	dialer := newExactReversePayloadDialer(1)
	brokerRegistry := rtunnels.NewRegistry()
	installHandlerBroker(t, brokerRegistry, dialer)
	connection, session := addBufferedTestSession(t)
	authorizationID := beginActiveAuthorization(t, brokerRegistry, session.ID, "127.0.0.1:4702", 4)
	const tunnelID = uint64(0x5500)

	tunnelDataHandler(connection, marshalTunnelData(t, reverseFollower(tunnelID, 1, []byte("late"))))
	receiveTunnelRejection(t, connection, tunnelID)
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		entries, sessions, usage := reverseTunnelPreCreateSnapshot(registry)
		if entries == 0 && sessions == 0 && usage == (reverseTunnelPreCreateUsage{}) {
			break
		}
		time.Sleep(time.Millisecond)
	}

	tunnelDataHandler(connection, marshalTunnelData(t, &sliverpb.TunnelData{
		TunnelID:      tunnelID,
		Data:          []byte("create"),
		CreateReverse: true,
		Rportfwd:      &sliverpb.RPortfwd{AuthorizationID: authorizationID.String()},
	}))
	receiveTunnelRejection(t, connection, tunnelID)
	if dialer.calls.Load() != 0 {
		t.Fatal("expired precursor allowed a late create to dial")
	}
}

func TestReverseTunnelPreCreateDisconnectReclaimsQuota(t *testing.T) {
	registry := newReverseTunnelPreCreateRegistry(time.Second)
	connection, session := addBufferedTestSession(t)
	if result, err := registry.stageData(connection, session.ID, reverseFollower(0x5600, 1, []byte("secret"))); result != reverseTunnelPreCreateStaged || err != nil {
		t.Fatalf("stage precursor = (%v, %v)", result, err)
	}
	connection.Close()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		entries, sessions, usage := reverseTunnelPreCreateSnapshot(registry)
		if entries == 0 && sessions == 0 && usage == (reverseTunnelPreCreateUsage{}) {
			return
		}
		time.Sleep(time.Millisecond)
	}
	entries, sessions, usage := reverseTunnelPreCreateSnapshot(registry)
	t.Fatalf("disconnect retained entries=%d sessions=%d total=%+v", entries, sessions, usage)
}

func TestReverseTunnelPreCreateRejectsConflictsAndInvalidWindows(t *testing.T) {
	t.Run("duplicate-identical-and-conflicting", func(t *testing.T) {
		registry := newReverseTunnelPreCreateRegistry(time.Second)
		connection, session := addBufferedTestSession(t)
		frame := reverseFollower(0x5700, 1, []byte("same"))
		if result, err := registry.stageData(connection, session.ID, frame); result != reverseTunnelPreCreateStaged || err != nil {
			t.Fatalf("first stage = (%v, %v)", result, err)
		}
		if result, err := registry.stageData(connection, session.ID, frame); result != reverseTunnelPreCreateStaged || err != nil {
			t.Fatalf("identical duplicate = (%v, %v)", result, err)
		}
		_, _, usage := reverseTunnelPreCreateSnapshot(registry)
		if usage.frames != 1 || usage.bytes != len(frame.Data) {
			t.Fatalf("identical duplicate changed usage to %+v", usage)
		}
		result, err := registry.stageData(connection, session.ID, reverseFollower(0x5700, 1, []byte("different")))
		if result != reverseTunnelPreCreateRejected || !errors.Is(err, errReverseTunnelPreCreateConflict) {
			t.Fatalf("conflicting duplicate = (%v, %v)", result, err)
		}
		entries, sessions, usage := reverseTunnelPreCreateSnapshot(registry)
		if entries != 0 || sessions != 0 || usage != (reverseTunnelPreCreateUsage{}) {
			t.Fatalf("conflict retained entries=%d sessions=%d total=%+v", entries, sessions, usage)
		}
	})

	for _, test := range []struct {
		name  string
		frame *sliverpb.TunnelData
	}{
		{name: "data-window", frame: reverseFollower(0x5701, maxReverseTunnelPreCreateFramesPerEntry+1, []byte("x"))},
		{name: "oversized-data", frame: reverseFollower(0x5702, 1, make([]byte, sliverpb.MaxTunnelFrameBytes+1))},
	} {
		t.Run(test.name, func(t *testing.T) {
			registry := newReverseTunnelPreCreateRegistry(time.Second)
			connection, session := addBufferedTestSession(t)
			result, err := registry.stageData(connection, session.ID, test.frame)
			if result != reverseTunnelPreCreateRejected || !errors.Is(err, errReverseTunnelPreCreateInvalid) {
				t.Fatalf("invalid precursor = (%v, %v)", result, err)
			}
			entries, sessions, usage := reverseTunnelPreCreateSnapshot(registry)
			if entries != 0 || sessions != 0 || usage != (reverseTunnelPreCreateUsage{}) {
				t.Fatalf("invalid precursor retained entries=%d sessions=%d total=%+v", entries, sessions, usage)
			}
		})
	}
}

func TestReverseTunnelMarkerCannotEnterGenericTunnel(t *testing.T) {
	installReverseTunnelPreCreateRegistry(t, time.Second)
	connection, session := addBufferedTestSession(t)
	tunnel, err := core.Tunnels.Create(session.ID)
	if err != nil {
		t.Fatalf("create generic tunnel: %v", err)
	}
	t.Cleanup(func() { core.Tunnels.CloseIf(tunnel) })

	tunnelDataHandler(connection, marshalTunnelData(t, reverseFollower(tunnel.ID, 1, []byte("must-not-enter"))))
	receiveTunnelRejection(t, connection, tunnel.ID)
	select {
	case frame := <-tunnel.FromImplant:
		t.Fatalf("reverse precursor entered generic tunnel: %+v", frame)
	default:
	}
}

func TestReverseTunnelPromotionDeadlineClosesBlockedDestination(t *testing.T) {
	registry := installReverseTunnelPreCreateRegistry(t, time.Second)
	brokerRegistry := rtunnels.NewRegistry()
	blocked := newBlockedReverseWriteConn()
	dialer := &blockedReverseWriteDialer{connection: blocked}
	installHandlerBroker(t, brokerRegistry, dialer)
	connection, session := addBufferedTestSession(t)
	authorizationID := beginActiveAuthorization(t, brokerRegistry, session.ID, "127.0.0.1:4710", 10)
	const tunnelID = uint64(0x5800)

	previousPromotionTimeout := reverseTunnelPromotionTimeout
	previousWaitTimeout := reverseTunnelOpeningWaitTimeout
	reverseTunnelPromotionTimeout = 25 * time.Millisecond
	reverseTunnelOpeningWaitTimeout = 100 * time.Millisecond
	t.Cleanup(func() {
		reverseTunnelPromotionTimeout = previousPromotionTimeout
		reverseTunnelOpeningWaitTimeout = previousWaitTimeout
	})

	tunnelDataHandler(connection, marshalTunnelData(t, reverseFollower(tunnelID, 1, []byte("cached"))))
	done := make(chan struct{})
	go func() {
		defer close(done)
		tunnelDataHandler(connection, marshalTunnelData(t, &sliverpb.TunnelData{
			TunnelID:      tunnelID,
			Data:          []byte("sequence-zero-blocks"),
			CreateReverse: true,
			Rportfwd:      &sliverpb.RPortfwd{AuthorizationID: authorizationID.String()},
		}))
	}()
	select {
	case <-blocked.writeStarted:
	case <-time.After(time.Second):
		t.Fatal("sequence-zero relay did not reach blocked destination")
	}
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("absolute promotion deadline did not wake blocked destination write")
	}
	select {
	case <-blocked.closed:
	default:
		t.Fatal("promotion cancellation did not close the exact destination")
	}
	if dialer.calls.Load() != 1 {
		t.Fatalf("outbound dial calls = %d, want 1", dialer.calls.Load())
	}
	receiveTunnelRejection(t, connection, tunnelID)
	assertNoReverseTunnelEnvelope(t, connection)
	if tunnel := rtunnels.GetRTunnel(tunnelID); tunnel != nil {
		closeTestReverseTunnel(tunnelID)
		t.Fatal("timed-out promotion retained an active reverse tunnel")
	}
	assertReverseTunnelOpeningsEmpty(t)
	waitForReverseTunnelPreCreateEmpty(t, registry)
	assertHandlerAdmissionEmpty(t, reverseTunnelOpeningAttempts)
}

func TestReverseTunnelPromotionUsesOneCumulativeDrainDeadline(t *testing.T) {
	registry := installReverseTunnelPreCreateRegistry(t, time.Second)
	brokerRegistry := rtunnels.NewRegistry()
	destination := newDelayedReverseWriteConn(20 * time.Millisecond)
	dialer := &blockedReverseWriteDialer{connection: destination}
	installHandlerBroker(t, brokerRegistry, dialer)
	connection, session := addBufferedTestSession(t)
	authorizationID := beginActiveAuthorization(t, brokerRegistry, session.ID, "127.0.0.1:4712", 12)
	const tunnelID = uint64(0x5802)

	previousPromotionTimeout := reverseTunnelPromotionTimeout
	previousWaitTimeout := reverseTunnelOpeningWaitTimeout
	reverseTunnelPromotionTimeout = 30 * time.Millisecond
	reverseTunnelOpeningWaitTimeout = 100 * time.Millisecond
	t.Cleanup(func() {
		reverseTunnelPromotionTimeout = previousPromotionTimeout
		reverseTunnelOpeningWaitTimeout = previousWaitTimeout
	})

	tunnelDataHandler(connection, marshalTunnelData(t, reverseFollower(tunnelID, 1, []byte("second"))))
	tunnelDataHandler(connection, marshalTunnelData(t, &sliverpb.TunnelData{
		TunnelID:      tunnelID,
		Data:          []byte("first"),
		CreateReverse: true,
		Rportfwd:      &sliverpb.RPortfwd{AuthorizationID: authorizationID.String()},
	}))
	if got := destination.started.Load(); got != 2 {
		t.Fatalf("destination writes started = %d, want 2", got)
	}
	if got := destination.completed.Load(); got != 1 {
		t.Fatalf("destination writes completed = %d, want only sequence zero", got)
	}
	receiveTunnelRejection(t, connection, tunnelID)
	assertNoReverseTunnelEnvelope(t, connection)
	if tunnel := rtunnels.GetRTunnel(tunnelID); tunnel != nil {
		closeTestReverseTunnel(tunnelID)
		t.Fatal("cumulative deadline retained active reverse tunnel")
	}
	assertReverseTunnelOpeningsEmpty(t)
	waitForReverseTunnelPreCreateEmpty(t, registry)
}

func TestReverseTunnelLoadedCollisionPoisonsLosingGeneration(t *testing.T) {
	for index, withPrecursor := range []bool{false, true} {
		name := "without-precursor"
		if withPrecursor {
			name = "with-precursor"
		}
		t.Run(name, func(t *testing.T) {
			registry := installReverseTunnelPreCreateRegistry(t, time.Second)
			attempts := newReverseTunnelAdmission(4, 4)
			workers := installHandlerAdmissions(t, attempts, newReverseTunnelAdmission(4, 4))
			brokerRegistry := rtunnels.NewRegistry()
			dialer := newExactReversePayloadDialer(0)
			installHandlerBroker(t, brokerRegistry, dialer)
			loserConnection, loserSession := addBufferedTestSession(t)
			winnerConnection, winnerSession := addBufferedTestSession(t)
			authorizationID := beginActiveAuthorization(t, brokerRegistry, loserSession.ID, "127.0.0.1:4713", uint32(13+index))
			tunnelID := uint64(0x5810 + index)
			if withPrecursor {
				tunnelDataHandler(loserConnection, marshalTunnelData(t, reverseFollower(tunnelID, 1, []byte("must-be-scrubbed"))))
			}

			// Hold the routing linearization point until the losing handler has
			// passed its optimistic opening lookup. Publishing the winner at that
			// instant deterministically exercises promote's LoadOrStore loser path.
			registry.routeMutex.Lock()
			routeLocked := true
			defer func() {
				if routeLocked {
					registry.routeMutex.Unlock()
				}
			}()
			create := reverseCreateData(t, tunnelID, authorizationID)
			loserDone := workers.launch(func() { tunnelDataHandler(loserConnection, create) })
			waitForReverseTunnelAdmissionTotal(t, attempts, 1)

			winner := newReverseTunnelOpening(winnerSession.ID, winnerConnection, func() {})
			if _, loaded := reverseTunnelOpenings.LoadOrStore(tunnelID, winner); loaded {
				t.Fatalf("reverse opening ID %d was already registered", tunnelID)
			}
			t.Cleanup(func() {
				if reverseTunnelOpenings.CompareAndDelete(tunnelID, winner) {
					close(winner.ready)
				}
			})
			registry.routeMutex.Unlock()
			routeLocked = false

			waitHandler(t, loserDone, "losing reverse create")
			receiveTunnelRejection(t, loserConnection, tunnelID)
			assertNoReverseTunnelEnvelope(t, loserConnection)
			waitForReverseTunnelPreCreateEmpty(t, registry)
			if got := dialer.calls.Load(); got != 0 {
				t.Fatalf("losing generation reached outbound dialer %d times", got)
			}

			if !reverseTunnelOpenings.CompareAndDelete(tunnelID, winner) {
				t.Fatal("winning reverse opening disappeared before retry")
			}
			close(winner.ready)
			tunnelDataHandler(loserConnection, create)
			receiveTunnelRejection(t, loserConnection, tunnelID)
			assertNoReverseTunnelEnvelope(t, loserConnection)
			if got := dialer.calls.Load(); got != 0 {
				t.Fatalf("poisoned losing generation retried %d outbound dials", got)
			}
			waitForReverseTunnelPreCreateEmpty(t, registry)
			assertReverseTunnelOpeningsEmpty(t)
			assertHandlerAdmissionEmpty(t, attempts)
		})
	}
}

func TestReverseTunnelOpeningAdmissionRejectionPoisonsGeneration(t *testing.T) {
	for index, withPrecursor := range []bool{false, true} {
		name := "without-precursor"
		if withPrecursor {
			name = "with-precursor"
		}
		t.Run(name, func(t *testing.T) {
			registry := installReverseTunnelPreCreateRegistry(t, time.Second)
			attempts := newReverseTunnelAdmission(1, 1)
			installHandlerAdmissions(t, attempts, newReverseTunnelAdmission(4, 4))
			brokerRegistry := rtunnels.NewRegistry()
			dialer := newExactReversePayloadDialer(0)
			installHandlerBroker(t, brokerRegistry, dialer)
			connection, session := addBufferedTestSession(t)
			authorizationID := beginActiveAuthorization(t, brokerRegistry, session.ID, "127.0.0.1:4714", uint32(15+index))
			tunnelID := uint64(0x5820 + index)
			if withPrecursor {
				tunnelDataHandler(connection, marshalTunnelData(t, reverseFollower(tunnelID, 1, []byte("must-be-scrubbed"))))
			}

			if !attempts.acquire("saturated-opening") {
				t.Fatal("failed to saturate reverse opening admission")
			}
			held := true
			t.Cleanup(func() {
				if held {
					attempts.release("saturated-opening")
				}
			})
			create := reverseCreateData(t, tunnelID, authorizationID)
			tunnelDataHandler(connection, create)
			receiveTunnelRejection(t, connection, tunnelID)
			assertNoReverseTunnelEnvelope(t, connection)
			waitForReverseTunnelPreCreateEmpty(t, registry)
			if got := dialer.calls.Load(); got != 0 {
				t.Fatalf("admission-rejected generation reached outbound dialer %d times", got)
			}

			attempts.release("saturated-opening")
			held = false
			tunnelDataHandler(connection, create)
			receiveTunnelRejection(t, connection, tunnelID)
			assertNoReverseTunnelEnvelope(t, connection)
			if got := dialer.calls.Load(); got != 0 {
				t.Fatalf("poisoned admission generation retried %d outbound dials", got)
			}
			waitForReverseTunnelPreCreateEmpty(t, registry)
			assertReverseTunnelOpeningsEmpty(t)
			assertHandlerAdmissionEmpty(t, attempts)
		})
	}
}

func TestReverseTunnelCachedGenerationWinsGenericCollision(t *testing.T) {
	registry := installReverseTunnelPreCreateRegistry(t, time.Second)
	brokerRegistry := rtunnels.NewRegistry()
	dialer := newExactReversePayloadDialer(1)
	installHandlerBroker(t, brokerRegistry, dialer)
	connection, session := addBufferedTestSession(t)
	authorizationID := beginActiveAuthorization(t, brokerRegistry, session.ID, "127.0.0.1:4711", 11)
	generic, err := core.Tunnels.Create(session.ID)
	if err != nil {
		t.Fatalf("create generic collision tunnel: %v", err)
	}
	t.Cleanup(func() { core.Tunnels.CloseIf(generic) })
	tunnelID := generic.ID

	// Hide the generic only while staging to deterministically model it being
	// published after the reverse precursor was admitted.
	registry.generic = func(uint64) *core.Tunnel { return nil }
	tunnelDataHandler(connection, marshalTunnelData(t, reverseFollower(tunnelID, 1, []byte("cached"))))
	registry.generic = core.Tunnels.Get
	tunnelCloseHandler(connection, marshalTunnelData(t, &sliverpb.TunnelData{TunnelID: tunnelID, Sequence: 2, Closed: true}))

	registry.mutex.Lock()
	entry := registry.entries[reverseTunnelPreCreateKey{connection: connection, sessionID: session.ID, tunnelID: tunnelID}]
	terminalStaged := entry != nil && entry.terminal != nil && entry.terminal.Sequence == 2
	registry.mutex.Unlock()
	if !terminalStaged {
		t.Fatal("generic collision stole terminal from exact cached reverse generation")
	}

	tunnelDataHandler(connection, marshalTunnelData(t, &sliverpb.TunnelData{
		TunnelID:      tunnelID,
		Data:          []byte("create"),
		CreateReverse: true,
		Rportfwd:      &sliverpb.RPortfwd{AuthorizationID: authorizationID.String()},
	}))
	receiveTunnelRejection(t, connection, tunnelID)
	if dialer.calls.Load() != 0 {
		t.Fatalf("cached generic collision reached outbound dialer %d times", dialer.calls.Load())
	}
	select {
	case <-generic.Done():
		t.Fatal("reverse collision closed the generic tunnel")
	default:
	}
	waitForReverseTunnelPreCreateEmpty(t, registry)

	// The reverse marker remains authoritative after rejection and must keep a
	// late terminal away from the still-live generic tunnel.
	tunnelCloseHandler(connection, marshalTunnelData(t, &sliverpb.TunnelData{
		TunnelID: tunnelID,
		Sequence: 2,
		Closed:   true,
		Rportfwd: &sliverpb.RPortfwd{},
	}))
	select {
	case <-generic.Done():
		t.Fatal("post-collision reverse terminal closed the generic tunnel")
	default:
	}
}

func TestReverseTunnelOpeningNeverFallsThroughToGeneric(t *testing.T) {
	installReverseTunnelPreCreateRegistry(t, time.Second)
	connection, session := addBufferedTestSession(t)
	tunnel, err := core.Tunnels.Create(session.ID)
	if err != nil {
		t.Fatalf("create generic tunnel: %v", err)
	}
	t.Cleanup(func() { core.Tunnels.CloseIf(tunnel) })

	opening := newReverseTunnelOpening(session.ID, connection, func() {})
	if _, loaded := reverseTunnelOpenings.LoadOrStore(tunnel.ID, opening); loaded {
		t.Fatalf("reverse opening ID %d was already registered", tunnel.ID)
	}
	close(opening.ready)
	t.Cleanup(func() { reverseTunnelOpenings.CompareAndDelete(tunnel.ID, opening) })

	tunnelDataHandler(connection, marshalTunnelData(t, reverseFollower(tunnel.ID, 1, []byte("must-not-enter"))))
	select {
	case frame := <-tunnel.FromImplant:
		t.Fatalf("failed reverse opening injected data into generic tunnel: %+v", frame)
	default:
	}

	tunnelCloseHandler(connection, marshalTunnelData(t, &sliverpb.TunnelData{TunnelID: tunnel.ID, Sequence: 2, Closed: true}))
	if core.Tunnels.Get(tunnel.ID) != tunnel {
		t.Fatal("failed reverse opening terminal closed generic tunnel")
	}
}

func assertNoReverseTunnelEnvelope(t *testing.T, connection *core.ImplantConnection) {
	t.Helper()
	select {
	case envelope := <-connection.Send:
		t.Fatalf("unexpected tunnel envelope type=%d data-bytes=%d", envelope.Type, len(envelope.Data))
	case <-time.After(25 * time.Millisecond):
	}
}

func TestReverseTunnelMalformedCachedGenerationIsPoisoned(t *testing.T) {
	tests := []struct {
		name string
		send func(*testing.T, *core.ImplantConnection, uint64, rtunnels.AuthorizationID)
	}{
		{
			name: "data",
			send: func(t *testing.T, connection *core.ImplantConnection, tunnelID uint64, _ rtunnels.AuthorizationID) {
				tunnelDataHandler(connection, marshalTunnelData(t, reverseFollower(tunnelID, maxReverseTunnelPreCreateFramesPerEntry+1, []byte("invalid"))))
			},
		},
		{
			name: "terminal",
			send: func(t *testing.T, connection *core.ImplantConnection, tunnelID uint64, _ rtunnels.AuthorizationID) {
				tunnelCloseHandler(connection, marshalTunnelData(t, &sliverpb.TunnelData{
					TunnelID: tunnelID,
					Sequence: maxReverseTunnelPreCreateTerminalSequence + 1,
					Closed:   true,
					Rportfwd: &sliverpb.RPortfwd{},
				}))
			},
		},
		{
			name: "create",
			send: func(t *testing.T, connection *core.ImplantConnection, tunnelID uint64, authorizationID rtunnels.AuthorizationID) {
				tunnelDataHandler(connection, marshalTunnelData(t, &sliverpb.TunnelData{
					TunnelID:      tunnelID,
					Sequence:      1,
					CreateReverse: true,
					Rportfwd:      &sliverpb.RPortfwd{AuthorizationID: authorizationID.String()},
				}))
			},
		},
	}

	for index, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			registry := installReverseTunnelPreCreateRegistry(t, time.Second)
			brokerRegistry := rtunnels.NewRegistry()
			dialer := newExactReversePayloadDialer(1)
			installHandlerBroker(t, brokerRegistry, dialer)
			connection, session := addBufferedTestSession(t)
			authorizationID := beginActiveAuthorization(t, brokerRegistry, session.ID, "127.0.0.1:4720", uint32(20+index))
			tunnelID := uint64(0x5900 + index)

			tunnelDataHandler(connection, marshalTunnelData(t, reverseFollower(tunnelID, 1, []byte("valid"))))
			test.send(t, connection, tunnelID, authorizationID)
			receiveTunnelRejection(t, connection, tunnelID)
			waitForReverseTunnelPreCreateEmpty(t, registry)

			// The permanent claim makes a duplicate malformed frame idempotent and
			// prevents unbounded rejection work.
			test.send(t, connection, tunnelID, authorizationID)
			assertNoReverseTunnelEnvelope(t, connection)

			tunnelDataHandler(connection, reverseCreateData(t, tunnelID, authorizationID))
			receiveTunnelRejection(t, connection, tunnelID)
			if got := dialer.calls.Load(); got != 0 {
				t.Fatalf("poisoned generation reached outbound dialer %d times", got)
			}
		})
	}
}

func TestMalformedCreateDoesNotRejectKnownReverseGeneration(t *testing.T) {
	for _, state := range []string{"active", "opening"} {
		t.Run(state, func(t *testing.T) {
			installReverseTunnelPreCreateRegistry(t, time.Second)
			connection, session := addBufferedTestSession(t)
			tunnelID := uint64(0x5a00)
			var active *rtunnels.RTunnel
			var opening *reverseTunnelOpening
			if state == "active" {
				active = rtunnels.NewOwnedRTunnel(tunnelID, session.ID, connection.ID, &countingTunnelWriteCloser{})
				if !rtunnels.TryAddRTunnel(active) {
					t.Fatal("failed to add active reverse tunnel")
				}
				t.Cleanup(func() {
					if rtunnels.RemoveRTunnelIf(tunnelID, active) {
						active.Close()
					}
				})
			} else {
				opening = newReverseTunnelOpening(session.ID, connection, func() {})
				if _, loaded := reverseTunnelOpenings.LoadOrStore(tunnelID, opening); loaded {
					t.Fatal("failed to add reverse opening")
				}
				t.Cleanup(func() {
					reverseTunnelOpenings.CompareAndDelete(tunnelID, opening)
					close(opening.ready)
				})
			}

			tunnelDataHandler(connection, marshalTunnelData(t, &sliverpb.TunnelData{
				TunnelID:      tunnelID,
				Sequence:      1,
				CreateReverse: true,
				Rportfwd:      &sliverpb.RPortfwd{},
			}))
			assertNoReverseTunnelEnvelope(t, connection)
			if active != nil && rtunnels.GetRTunnel(tunnelID) != active {
				t.Fatal("malformed retransmit removed active reverse generation")
			}
			if opening != nil {
				if got, ok := reverseTunnelOpenings.Load(tunnelID); !ok || got != opening {
					t.Fatal("malformed retransmit removed opening reverse generation")
				}
			}
		})
	}
}

func TestMarkedMalformedTerminalCannotCloseGenericTunnel(t *testing.T) {
	installReverseTunnelPreCreateRegistry(t, time.Second)
	connection, session := addBufferedTestSession(t)
	generic, err := core.Tunnels.Create(session.ID)
	if err != nil {
		t.Fatalf("create generic tunnel: %v", err)
	}
	t.Cleanup(func() { core.Tunnels.CloseIf(generic) })

	tunnelCloseHandler(connection, marshalTunnelData(t, &sliverpb.TunnelData{
		TunnelID: generic.ID,
		Sequence: maxReverseTunnelPreCreateTerminalSequence + 1,
		Closed:   true,
		Rportfwd: &sliverpb.RPortfwd{},
	}))
	receiveTunnelRejection(t, connection, generic.ID)
	if core.Tunnels.Get(generic.ID) != generic {
		t.Fatal("malformed marked reverse terminal closed generic tunnel")
	}
}

func TestUnknownTerminalRequiresReverseMarkerToClaimPreCreateState(t *testing.T) {
	t.Run("unmarked remains unclassified", func(t *testing.T) {
		registry := installReverseTunnelPreCreateRegistry(t, time.Second)
		brokerRegistry := rtunnels.NewRegistry()
		dialer := newExactReversePayloadDialer(0)
		installHandlerBroker(t, brokerRegistry, dialer)
		connection, session := addBufferedTestSession(t)
		authorizationID := beginActiveAuthorization(t, brokerRegistry, session.ID, "127.0.0.1:4730", 30)
		const tunnelID = uint64(0x5b00)

		tunnelCloseHandler(connection, marshalTunnelData(t, &sliverpb.TunnelData{TunnelID: tunnelID, Closed: true}))
		entries, sessions, usage := reverseTunnelPreCreateSnapshot(registry)
		if entries != 0 || sessions != 0 || usage != (reverseTunnelPreCreateUsage{}) {
			t.Fatalf("unknown unmarked terminal allocated reverse state: entries=%d sessions=%d total=%+v", entries, sessions, usage)
		}
		assertNoReverseTunnelEnvelope(t, connection)

		tunnelDataHandler(connection, reverseCreateData(t, tunnelID, authorizationID))
		if got := dialer.calls.Load(); got != 1 {
			t.Fatalf("unmarked terminal claimed reverse ID; dial calls=%d, want 1", got)
		}
		closeTestReverseTunnel(tunnelID)
	})

	t.Run("marked tombstones overtaking create", func(t *testing.T) {
		registry := installReverseTunnelPreCreateRegistry(t, time.Second)
		brokerRegistry := rtunnels.NewRegistry()
		dialer := newExactReversePayloadDialer(0)
		installHandlerBroker(t, brokerRegistry, dialer)
		connection, session := addBufferedTestSession(t)
		authorizationID := beginActiveAuthorization(t, brokerRegistry, session.ID, "127.0.0.1:4731", 31)
		const tunnelID = uint64(0x5b01)

		tunnelCloseHandler(connection, marshalTunnelData(t, &sliverpb.TunnelData{
			TunnelID: tunnelID,
			Closed:   true,
			Rportfwd: &sliverpb.RPortfwd{},
		}))
		entries, _, usage := reverseTunnelPreCreateSnapshot(registry)
		if entries != 1 || usage.entries != 1 {
			t.Fatalf("marked terminal was not staged: entries=%d total=%+v", entries, usage)
		}
		tunnelDataHandler(connection, reverseCreateData(t, tunnelID, authorizationID))
		receiveTunnelRejection(t, connection, tunnelID)
		if got := dialer.calls.Load(); got != 0 {
			t.Fatalf("marked overtaking terminal allowed %d dials", got)
		}
		waitForReverseTunnelPreCreateEmpty(t, registry)
	})
}

func TestReverseTunnelPreCreateQuotaSaturationReclaimsExactly(t *testing.T) {
	t.Run("per-session entries", func(t *testing.T) {
		registry := newReverseTunnelPreCreateRegistry(time.Hour)
		connection := core.NewImplantConnection("test", "quota")
		connection.Send = make(chan *sliverpb.Envelope, 8)
		const sessionID = "entry-quota-session"
		for index := 0; index < maxReverseTunnelPreCreateEntriesPerSession; index++ {
			result, err := registry.stageData(connection, sessionID, reverseFollower(uint64(0x6000+index), 1, []byte{byte(index)}))
			if result != reverseTunnelPreCreateStaged || err != nil {
				t.Fatalf("stage entry %d = (%v, %v)", index, result, err)
			}
		}
		overflowID := uint64(0x60ff)
		result, err := registry.stageData(connection, sessionID, reverseFollower(overflowID, 1, []byte("overflow")))
		if result != reverseTunnelPreCreateRejected || !errors.Is(err, errReverseTunnelPreCreateEntryLimit) {
			t.Fatalf("entry quota overflow = (%v, %v)", result, err)
		}
		select {
		case <-connection.Done():
			t.Fatal("ordinary per-session precursor quota closed the C2 connection")
		default:
		}
		if result, _ := registry.stageData(connection, sessionID, reverseFollower(overflowID, 1, []byte("overflow"))); result != reverseTunnelPreCreateTombstoned {
			t.Fatalf("entry quota replay result = %v, want tombstoned", result)
		}
		entries, sessions, usage := reverseTunnelPreCreateSnapshot(registry)
		if entries != maxReverseTunnelPreCreateEntriesPerSession || sessions != 1 || usage.entries != maxReverseTunnelPreCreateEntriesPerSession {
			t.Fatalf("entry quota snapshot entries=%d sessions=%d total=%+v", entries, sessions, usage)
		}
		connection.Close()
		waitForReverseTunnelPreCreateEmpty(t, registry)
	})

	t.Run("per-session frames", func(t *testing.T) {
		registry := newReverseTunnelPreCreateRegistry(time.Hour)
		connection := core.NewImplantConnection("test", "quota")
		connection.Send = make(chan *sliverpb.Envelope, 8)
		const sessionID = "frame-quota-session"
		const tunnelID = uint64(0x6100)
		for sequence := uint64(1); sequence <= maxReverseTunnelPreCreateFramesPerSession; sequence++ {
			result, err := registry.stageData(connection, sessionID, reverseFollower(tunnelID, sequence, []byte{byte(sequence)}))
			if result != reverseTunnelPreCreateStaged || err != nil {
				t.Fatalf("stage frame %d = (%v, %v)", sequence, result, err)
			}
		}
		result, err := registry.stageData(connection, sessionID, reverseFollower(0x6101, 1, []byte("overflow")))
		if result != reverseTunnelPreCreateRejected || !errors.Is(err, errReverseTunnelPreCreateFrameLimit) {
			t.Fatalf("frame quota overflow = (%v, %v)", result, err)
		}
		entries, sessions, usage := reverseTunnelPreCreateSnapshot(registry)
		if entries != 1 || sessions != 1 || usage.entries != 1 || usage.frames != maxReverseTunnelPreCreateFramesPerSession {
			t.Fatalf("frame quota snapshot entries=%d sessions=%d total=%+v", entries, sessions, usage)
		}
		connection.Close()
		waitForReverseTunnelPreCreateEmpty(t, registry)
	})

	t.Run("global entries", func(t *testing.T) {
		registry := newReverseTunnelPreCreateRegistry(time.Hour)
		connections := make([]*core.ImplantConnection, 0, 5)
		for sessionIndex := 0; sessionIndex < 4; sessionIndex++ {
			connection := core.NewImplantConnection("test", "quota")
			connection.Send = make(chan *sliverpb.Envelope, 8)
			connections = append(connections, connection)
			sessionID := fmt.Sprintf("global-quota-%d", sessionIndex)
			for entryIndex := 0; entryIndex < maxReverseTunnelPreCreateEntriesPerSession; entryIndex++ {
				tunnelID := uint64(0x6200 + sessionIndex*maxReverseTunnelPreCreateEntriesPerSession + entryIndex)
				result, err := registry.stageData(connection, sessionID, reverseFollower(tunnelID, 1, []byte("x")))
				if result != reverseTunnelPreCreateStaged || err != nil {
					t.Fatalf("stage global entry %d/%d = (%v, %v)", sessionIndex, entryIndex, result, err)
				}
			}
		}
		overflow := core.NewImplantConnection("test", "quota-overflow")
		overflow.Send = make(chan *sliverpb.Envelope, 8)
		connections = append(connections, overflow)
		result, err := registry.stageData(overflow, "global-overflow", reverseFollower(0x62ff, 1, []byte("overflow")))
		if result != reverseTunnelPreCreateRejected || !errors.Is(err, errReverseTunnelPreCreateEntryLimit) {
			t.Fatalf("global entry overflow = (%v, %v)", result, err)
		}
		select {
		case <-overflow.Done():
			t.Fatal("global precursor saturation closed unrelated C2 connection")
		default:
		}
		entries, sessions, usage := reverseTunnelPreCreateSnapshot(registry)
		if entries != maxReverseTunnelPreCreateEntriesGlobal || sessions != 4 || usage.entries != maxReverseTunnelPreCreateEntriesGlobal {
			t.Fatalf("global quota snapshot entries=%d sessions=%d total=%+v", entries, sessions, usage)
		}
		for _, connection := range connections {
			connection.Close()
		}
		waitForReverseTunnelPreCreateEmpty(t, registry)
	})
}

func TestActiveReverseTunnelRejectsSameSessionReplacementGeneration(t *testing.T) {
	installReverseTunnelPreCreateRegistry(t, time.Second)
	brokerRegistry := rtunnels.NewRegistry()
	dialer := newExactReversePayloadDialer(0)
	installHandlerBroker(t, brokerRegistry, dialer)
	ownerConnection, ownerSession := addBufferedTestSession(t)
	authorizationID := beginActiveAuthorization(t, brokerRegistry, ownerSession.ID, "127.0.0.1:4740", 40)
	const tunnelID = uint64(0x6300)

	tunnelDataHandler(ownerConnection, reverseCreateData(t, tunnelID, authorizationID))
	tunnel := rtunnels.GetRTunnel(tunnelID)
	if tunnel == nil || !tunnel.OwnedBy(ownerSession.ID, ownerConnection.ID) {
		t.Fatal("reverse tunnel was not bound to its exact owner generation")
	}

	replacementConnection := core.NewImplantConnection("test", "replacement")
	replacementConnection.Send = make(chan *sliverpb.Envelope, 8)
	replacement := core.NewSession(replacementConnection)
	replacement.ID = ownerSession.ID
	core.Sessions.Add(replacement)
	t.Cleanup(func() {
		core.Sessions.RemoveIf(replacement)
		replacementConnection.Close()
	})

	tunnelDataHandler(replacementConnection, marshalTunnelData(t, reverseFollower(tunnelID, 1, []byte("must-not-relay"))))
	receiveTunnelRejection(t, replacementConnection, tunnelID)
	tunnelCloseHandler(replacementConnection, marshalTunnelData(t, &sliverpb.TunnelData{
		TunnelID: tunnelID,
		Sequence: 2,
		Closed:   true,
		Rportfwd: &sliverpb.RPortfwd{},
	}))
	if got := tunnel.ReadSequence(); got != 1 {
		t.Fatalf("replacement generation advanced reverse read sequence to %d", got)
	}
	if rtunnels.GetRTunnel(tunnelID) != tunnel {
		t.Fatal("replacement generation closed stale owner's reverse tunnel")
	}

	ownerConnection.Close()
	deadline := time.Now().Add(time.Second)
	for rtunnels.GetRTunnel(tunnelID) == tunnel && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if rtunnels.GetRTunnel(tunnelID) == tunnel {
		closeTestReverseTunnel(tunnelID)
		t.Fatal("exact owner disconnect did not close its reverse tunnel after same-ID session replacement")
	}

	tunnelDataHandler(replacementConnection, reverseCreateData(t, tunnelID, authorizationID))
	receiveTunnelRejection(t, replacementConnection, tunnelID)
	if got := dialer.calls.Load(); got != 1 {
		t.Fatalf("replacement reopened a stream missing its earlier frame; dial calls=%d, want 1", got)
	}
}

func TestReverseTunnelClaimCapacityCleanupDoesNotBlockUnrelatedPromotion(t *testing.T) {
	registry := newReverseTunnelPreCreateRegistry(time.Hour)
	blockedConnection := core.NewImplantConnection("test", "blocked-cleanup")
	cleanupStarted := make(chan struct{})
	releaseCleanup := make(chan struct{})
	var releaseOnce sync.Once
	release := func() { releaseOnce.Do(func() { close(releaseCleanup) }) }
	t.Cleanup(release)
	if !blockedConnection.SetCleanup(func() {
		close(cleanupStarted)
		<-releaseCleanup
	}) {
		t.Fatal("failed to install blocked cleanup")
	}
	const reverseTunnelIDHistoryLimit = 4096
	for tunnelID := uint64(0); tunnelID < reverseTunnelIDHistoryLimit; tunnelID++ {
		if result := blockedConnection.TryClaimReverseTunnelID(tunnelID); result != core.ReverseTunnelIDClaimed {
			t.Fatalf("prefill claim %d = %v", tunnelID, result)
		}
	}
	blockedResult := make(chan reverseTunnelPreCreateStageResult, 1)
	go func() {
		result, _ := registry.stageData(blockedConnection, "blocked", reverseFollower(reverseTunnelIDHistoryLimit, 1, []byte("x")))
		blockedResult <- result
	}()
	select {
	case <-cleanupStarted:
	case <-time.After(time.Second):
		t.Fatal("capacity exhaustion did not enter deferred cleanup")
	}

	unrelatedConnection := core.NewImplantConnection("test", "unrelated")
	unrelatedConnection.Send = make(chan *sliverpb.Envelope, 1)
	const unrelatedTunnelID = uint64(0x6400)
	unrelatedStage := make(chan reverseTunnelPreCreateStageResult, 1)
	go func() {
		result, _ := registry.stageData(unrelatedConnection, "unrelated", reverseFollower(unrelatedTunnelID, 1, []byte("ok")))
		unrelatedStage <- result
	}()
	select {
	case result := <-unrelatedStage:
		if result != reverseTunnelPreCreateStaged {
			t.Fatalf("unrelated stage result = %v", result)
		}
	case <-time.After(250 * time.Millisecond):
		t.Fatal("blocked C2 cleanup retained the global pre-create route lock")
	}

	opening := newReverseTunnelOpening("unrelated", unrelatedConnection, func() {})
	type promotionResult struct {
		entry  *reverseTunnelPreCreateEntry
		loaded bool
		err    error
	}
	promoted := make(chan promotionResult, 1)
	go func() {
		entry, _, loaded, err := registry.promote(unrelatedConnection, "unrelated", unrelatedTunnelID, opening)
		promoted <- promotionResult{entry: entry, loaded: loaded, err: err}
	}()
	var result promotionResult
	select {
	case result = <-promoted:
	case <-time.After(250 * time.Millisecond):
		t.Fatal("blocked C2 cleanup prevented unrelated reverse promotion")
	}
	if result.loaded || result.err != nil || result.entry == nil {
		t.Fatalf("unrelated promotion = entry:%p loaded:%v err:%v", result.entry, result.loaded, result.err)
	}
	<-result.entry.claimReady
	registry.release(result.entry)
	reverseTunnelOpenings.CompareAndDelete(unrelatedTunnelID, opening)
	close(opening.ready)
	unrelatedConnection.Close()

	release()
	select {
	case result := <-blockedResult:
		if result != reverseTunnelPreCreateTombstoned {
			t.Fatalf("capacity-exhausted stage result = %v", result)
		}
	case <-time.After(time.Second):
		t.Fatal("capacity-exhausted stage did not finish after cleanup release")
	}
	waitForReverseTunnelPreCreateEmpty(t, registry)
}

func TestReverseTunnelPreCreateConcurrentLifecycleRaces(t *testing.T) {
	registry := newReverseTunnelPreCreateRegistry(time.Millisecond)
	const iterations = 64
	for iteration := 0; iteration < iterations; iteration++ {
		connection := core.NewImplantConnection("test", "race")
		connection.Send = make(chan *sliverpb.Envelope, 4)
		sessionID := fmt.Sprintf("race-session-%d", iteration)
		tunnelID := uint64(0x7000 + iteration)
		opening := newReverseTunnelOpening(sessionID, connection, func() {})
		start := make(chan struct{})
		stageDone := make(chan struct{})
		go func() {
			defer close(stageDone)
			<-start
			_, _ = registry.stageData(connection, sessionID, reverseFollower(tunnelID, 1, []byte("race")))
		}()
		type promotionResult struct {
			entry  *reverseTunnelPreCreateEntry
			loaded bool
			err    error
		}
		promotionDone := make(chan promotionResult, 1)
		go func() {
			<-start
			time.Sleep(time.Duration(iteration%3) * time.Millisecond)
			entry, _, loaded, err := registry.promote(connection, sessionID, tunnelID, opening)
			promotionDone <- promotionResult{entry: entry, loaded: loaded, err: err}
		}()
		closeDone := make(chan struct{})
		go func() {
			defer close(closeDone)
			<-start
			time.Sleep(time.Duration((iteration+1)%3) * time.Millisecond)
			connection.Close()
		}()
		close(start)

		select {
		case <-stageDone:
		case <-time.After(time.Second):
			t.Fatalf("iteration %d stage did not finish", iteration)
		}
		var promoted promotionResult
		select {
		case promoted = <-promotionDone:
		case <-time.After(time.Second):
			t.Fatalf("iteration %d promotion did not finish", iteration)
		}
		if promoted.loaded || promoted.err != nil {
			t.Fatalf("iteration %d promotion loaded=%v err=%v", iteration, promoted.loaded, promoted.err)
		}
		if promoted.entry != nil {
			<-promoted.entry.claimReady
			registry.release(promoted.entry)
		}
		reverseTunnelOpenings.CompareAndDelete(tunnelID, opening)
		close(opening.ready)
		select {
		case <-closeDone:
		case <-time.After(time.Second):
			t.Fatalf("iteration %d disconnect did not finish", iteration)
		}
		waitForReverseTunnelPreCreateEmpty(t, registry)
	}
}

func TestMalformedMarkedFramesFailKnownReverseGenerationOnce(t *testing.T) {
	for _, state := range []string{"active", "opening"} {
		for _, frameKind := range []string{"data", "terminal"} {
			t.Run(state+"-"+frameKind, func(t *testing.T) {
				installReverseTunnelPreCreateRegistry(t, time.Second)
				connection, session := addBufferedTestSession(t)
				generic, err := core.Tunnels.Create(session.ID)
				if err != nil {
					t.Fatalf("create generic collision: %v", err)
				}
				t.Cleanup(func() { core.Tunnels.CloseIf(generic) })
				tunnelID := generic.ID
				if result := connection.TryClaimReverseTunnelID(tunnelID); result != core.ReverseTunnelIDClaimed {
					t.Fatalf("claim reverse ID result = %v", result)
				}

				writer := &countingTunnelWriteCloser{}
				var active *rtunnels.RTunnel
				var opening *reverseTunnelOpening
				var canceled atomic.Int32
				if state == "active" {
					active = rtunnels.NewOwnedRTunnel(tunnelID, session.ID, connection.ID, writer)
					if !rtunnels.TryAddRTunnel(active) {
						t.Fatal("failed to add active reverse tunnel")
					}
					t.Cleanup(func() {
						if rtunnels.RemoveRTunnelIf(tunnelID, active) {
							active.Close()
						}
					})
				} else {
					opening = newReverseTunnelOpening(session.ID, connection, func() { canceled.Add(1) })
					if _, loaded := reverseTunnelOpenings.LoadOrStore(tunnelID, opening); loaded {
						t.Fatal("failed to add reverse opening")
					}
					t.Cleanup(func() {
						reverseTunnelOpenings.CompareAndDelete(tunnelID, opening)
						close(opening.ready)
					})
				}

				sendMalformed := func() {
					if frameKind == "data" {
						tunnelDataHandler(connection, marshalTunnelData(t, &sliverpb.TunnelData{
							TunnelID: tunnelID,
							Sequence: 1,
							Data:     []byte("must-not-relay"),
							Rportfwd: &sliverpb.RPortfwd{Host: "non-empty-marker"},
						}))
						return
					}
					tunnelCloseHandler(connection, marshalTunnelData(t, &sliverpb.TunnelData{
						TunnelID: tunnelID,
						Sequence: 1,
						Closed:   true,
						Data:     []byte("invalid-terminal-payload"),
						Rportfwd: &sliverpb.RPortfwd{},
					}))
				}
				sendMalformed()
				receiveTunnelRejection(t, connection, tunnelID)
				sendMalformed()
				assertNoReverseTunnelEnvelope(t, connection)

				if got := writer.writes.Load(); got != 0 {
					t.Fatalf("malformed %s performed %d destination writes", frameKind, got)
				}
				if state == "active" && rtunnels.GetRTunnel(tunnelID) == active {
					t.Fatal("malformed marked frame retained active reverse relay")
				}
				if state == "opening" {
					if !opening.closing.Load() || !opening.failure.Load() {
						t.Fatal("malformed marked frame did not poison reverse opening")
					}
					if got := canceled.Load(); got == 0 {
						t.Fatal("malformed marked frame did not cancel reverse opening")
					}
				}
				if core.Tunnels.Get(tunnelID) != generic {
					t.Fatal("malformed marked reverse frame affected generic collision tunnel")
				}
				select {
				case frame := <-generic.FromImplant:
					t.Fatalf("malformed marked reverse frame entered generic tunnel: %+v", frame)
				default:
				}
				select {
				case <-connection.Done():
					t.Fatal("exact reverse protocol failure closed the whole C2 connection")
				default:
				}
			})
		}
	}
}
