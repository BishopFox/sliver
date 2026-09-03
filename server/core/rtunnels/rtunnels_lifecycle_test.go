package rtunnels

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRTunnelCloseIsIdempotentAndNilSafe(t *testing.T) {
	reader := &countReadCloser{}
	writer := &countWriteCloser{}
	tunnel := NewAuthorizedRTunnel(91001, "session", AuthorizationID("authorization"), writer, reader, nil)

	tunnel.Close()
	tunnel.Close()
	assert.Equal(t, int32(1), reader.closes.Load())
	assert.Equal(t, int32(1), writer.closes.Load())
	assert.Equal(t, AuthorizationID("authorization"), tunnel.AuthorizationID())
	select {
	case <-tunnel.Done():
	default:
		t.Fatal("Close() did not signal tunnel completion")
	}
}

func TestRTunnelProcessInboundIsOrderedBoundedAndGenerationOwned(t *testing.T) {
	first := NewRTunnel(92001, "session", &countWriteCloser{})
	second := NewRTunnel(92001, "session", &countWriteCloser{})

	var output bytes.Buffer
	write := func(data []byte) error {
		_, err := output.Write(data)
		return err
	}
	pending, err := first.ProcessInbound(1, []byte("second"), write)
	must.NoError(t, err)
	assert.Equal(t, 1, pending)
	pending, err = first.ProcessInbound(0, []byte("first-"), write)
	must.NoError(t, err)
	assert.Zero(t, pending)
	assert.Equal(t, "first-second", output.String())

	first.Close()
	var secondOutput bytes.Buffer
	pending, err = second.ProcessInbound(0, []byte("new-generation"), func(data []byte) error {
		_, writeErr := secondOutput.Write(data)
		return writeErr
	})
	must.NoError(t, err)
	assert.Zero(t, pending)
	assert.Equal(t, "new-generation", secondOutput.String())
}

func TestRTunnelPeerCloseWaitsForOutOfOrderFinalFrame(t *testing.T) {
	tunnel := NewRTunnel(77, "session", &countWriteCloser{})
	ready, err := tunnel.MarkPeerClose(2)
	assert.NoError(t, err)
	assert.False(t, ready)

	var output bytes.Buffer
	_, err = tunnel.ProcessInbound(1, []byte("second"), func(data []byte) error {
		_, writeErr := output.Write(data)
		return writeErr
	})
	assert.NoError(t, err)
	assert.False(t, tunnel.PeerCloseReady())
	_, err = tunnel.ProcessInbound(0, []byte("first-"), func(data []byte) error {
		_, writeErr := output.Write(data)
		return writeErr
	})
	assert.NoError(t, err)
	assert.Equal(t, "first-second", output.String())
	assert.True(t, tunnel.PeerCloseReady())
	_, err = tunnel.ProcessInbound(2, []byte("after-terminal"), func([]byte) error { return nil })
	assert.ErrorIs(t, err, ErrReverseTunnelTerminal)
}

func TestRTunnelAcceptsFullTransportReorderWindowBeforeTerminal(t *testing.T) {
	tunnel := NewRTunnel(78, "session", &countWriteCloser{})
	ready, err := tunnel.MarkPeerClose(maxReverseTunnelPendingFrames)
	assert.NoError(t, err)
	assert.False(t, ready)

	var output bytes.Buffer
	for sequence := maxReverseTunnelPendingFrames; sequence > 0; sequence-- {
		value := byte(sequence - 1)
		_, err := tunnel.ProcessInbound(uint64(sequence-1), []byte{value}, func(data []byte) error {
			_, writeErr := output.Write(data)
			return writeErr
		})
		if err != nil {
			t.Fatalf("ProcessInbound(%d): %v", sequence-1, err)
		}
	}
	assert.True(t, tunnel.PeerCloseReady())
	assert.Len(t, output.Bytes(), maxReverseTunnelPendingFrames)
	for index, value := range output.Bytes() {
		assert.Equal(t, byte(index), value)
	}
}

func TestRTunnelLegacyZeroCloseIsImmediateWithPendingFrames(t *testing.T) {
	tunnel := NewRTunnel(79, "session", &countWriteCloser{})
	pending, err := tunnel.ProcessInbound(1, []byte("pending"), func([]byte) error { return nil })
	assert.NoError(t, err)
	assert.Equal(t, 1, pending)
	ready, err := tunnel.MarkPeerClose(0)
	assert.NoError(t, err)
	assert.True(t, ready)
}

func TestRTunnelLocalCloseNotifiesOnceAfterAcceptedFrames(t *testing.T) {
	tunnel := NewRTunnel(88, "session", &countWriteCloser{})
	var notified []uint64
	tunnel.SetPeerCloseNotifier(func(sequence uint64) error {
		notified = append(notified, sequence)
		return nil
	})
	assert.NoError(t, tunnel.QueueOutbound(func(sequence uint64) error {
		assert.Equal(t, uint64(0), sequence)
		return nil
	}))
	initiated, err := tunnel.closeLocal()
	assert.True(t, initiated)
	assert.NoError(t, err)
	initiated, err = tunnel.closeLocal()
	assert.False(t, initiated)
	assert.NoError(t, err)
	assert.Equal(t, []uint64{1}, notified)
	select {
	case <-tunnel.Done():
	default:
		t.Fatal("local close did not close tunnel resources")
	}
}

func TestRTunnelConcurrentLocalCloseNotifiesOnceOnFailure(t *testing.T) {
	tunnel := NewRTunnel(89, "session", &countWriteCloser{})
	notifyErr := errors.New("notification failed")
	var notifications atomic.Int32
	tunnel.SetPeerCloseNotifier(func(uint64) error {
		notifications.Add(1)
		return notifyErr
	})

	const closers = 32
	type closeResult struct {
		initiated bool
		err       error
	}
	results := make(chan closeResult, closers)
	var closeWait sync.WaitGroup
	closeWait.Add(closers)
	for range closers {
		go func() {
			defer closeWait.Done()
			initiated, err := tunnel.closeLocal()
			results <- closeResult{initiated: initiated, err: err}
		}()
	}
	closeWait.Wait()
	close(results)
	failures := 0
	initiators := 0
	for result := range results {
		if result.initiated {
			initiators++
		}
		if errors.Is(result.err, notifyErr) {
			failures++
		}
	}
	assert.Equal(t, int32(1), notifications.Load())
	assert.Equal(t, 1, initiators)
	assert.Equal(t, 1, failures)
}

func TestRTunnelSimultaneousPeerCloseUnblocksLocalNotifier(t *testing.T) {
	tunnelID := uint64(90)
	tunnel := NewRTunnel(tunnelID, "session", &countWriteCloser{})
	must.True(t, TryAddRTunnel(tunnel))
	t.Cleanup(func() {
		if RemoveRTunnelIf(tunnelID, tunnel) {
			tunnel.Close()
		}
	})

	notifierStarted := make(chan struct{})
	tunnel.SetPeerCloseNotifier(func(uint64) error {
		close(notifierStarted)
		<-tunnel.Done()
		return ErrReverseTunnelClosed
	})
	type localCloseResult struct {
		closed bool
		err    error
	}
	localResult := make(chan localCloseResult, 1)
	go func() {
		closed, err := CloseLocalIfActive(tunnel)
		localResult <- localCloseResult{closed: closed, err: err}
	}()
	select {
	case <-notifierStarted:
	case <-time.After(time.Second):
		t.Fatal("local terminal notifier did not start")
	}

	remoteResult := make(chan bool, 1)
	go func() {
		remoteResult <- CloseRemoteIfActive(tunnel)
	}()
	select {
	case result := <-localResult:
		if result.err != nil {
			assert.ErrorIs(t, result.err, ErrReverseTunnelClosed)
		}
	case <-time.After(time.Second):
		t.Fatal("local close did not finish after peer teardown")
	}
	select {
	case <-remoteResult:
	case <-time.After(time.Second):
		t.Fatal("remote close did not finish")
	}
	assert.Nil(t, GetRTunnel(tunnelID))
	assert.True(t, tunnel.PeerTeardownPending())
	assert.False(t, tunnel.PeerClosePending())
}

func TestRTunnelRemoteTeardownMarkerDoesNotInstallTerminal(t *testing.T) {
	tunnel := NewRTunnel(91, "session", &countWriteCloser{})
	// Reproduce the closeRemote linearization window after peer ownership is
	// published but before Done is closed.
	tunnel.peerTeardown.Store(true)
	written := 0
	pending, err := tunnel.ProcessInbound(0, []byte("in-flight"), func(payload []byte) error {
		written += len(payload)
		return nil
	})
	assert.NoError(t, err)
	assert.Zero(t, pending)
	assert.Equal(t, len("in-flight"), written)
	assert.True(t, tunnel.PeerTeardownPending())
	assert.False(t, tunnel.PeerClosePending())
	tunnel.Close()
}

func TestRTunnelProcessInboundRejectsRetainedPointerAfterClose(t *testing.T) {
	tunnel := NewRTunnel(92006, "session", &countWriteCloser{})
	tunnel.Close()

	writeCalled := false
	pending, err := tunnel.ProcessInbound(0, []byte("late-data"), func([]byte) error {
		writeCalled = true
		return nil
	})
	assert.ErrorIs(t, err, ErrReverseTunnelClosed)
	assert.Zero(t, pending)
	assert.False(t, writeCalled)
	assert.NotNil(t, tunnel.pendingInbound)
	assert.Empty(t, tunnel.pendingInbound)
}

func TestRTunnelProcessInboundRejectsResourceExhaustion(t *testing.T) {
	tests := []struct {
		name     string
		sequence uint64
		data     []byte
		want     error
	}{
		{name: "oversized frame", data: make([]byte, maxReverseTunnelFrameBytes+1), want: ErrReverseTunnelFrameTooLarge},
		{name: "sequence window", sequence: maxReverseTunnelPendingFrames, data: []byte("x"), want: ErrReverseTunnelWindow},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tunnel := NewRTunnel(92002, "session", &countWriteCloser{})
			_, err := tunnel.ProcessInbound(test.sequence, test.data, func([]byte) error { return nil })
			if !errors.Is(err, test.want) {
				t.Fatalf("ProcessInbound() error = %v, want %v", err, test.want)
			}
		})
	}
}

func TestRTunnelProcessInboundBoundsIngressBeforeSerialization(t *testing.T) {
	budget := newPendingRelayBudget()
	tunnel := NewAuthorizedRTunnel(92007, "session", AuthorizationID("authorization"), &countWriteCloser{})
	tunnel.pendingBudget = budget

	writeStarted := make(chan struct{})
	releaseWrite := make(chan struct{})
	var blockOnce sync.Once
	write := func([]byte) error {
		blockOnce.Do(func() {
			close(writeStarted)
			<-releaseWrite
		})
		return nil
	}

	results := make(chan error, maxReverseTunnelIngress)
	go func() {
		_, err := tunnel.ProcessInbound(0, []byte("first"), write)
		results <- err
	}()
	select {
	case <-writeStarted:
	case <-time.After(time.Second):
		t.Fatal("first inbound frame did not reach the writer")
	}

	for range maxReverseTunnelIngress - 1 {
		go func() {
			_, err := tunnel.ProcessInbound(1, []byte("queued"), write)
			results <- err
		}()
	}
	deadline := time.Now().Add(time.Second)
	for len(tunnel.inboundAdmission) != maxReverseTunnelIngress && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if got := len(tunnel.inboundAdmission); got != maxReverseTunnelIngress {
		t.Fatalf("admitted inbound handlers = %d, want %d", got, maxReverseTunnelIngress)
	}

	_, err := tunnel.ProcessInbound(1, []byte("rejected"), write)
	assert.ErrorIs(t, err, ErrReverseTunnelIngressLimit)
	close(releaseWrite)
	for range maxReverseTunnelIngress {
		select {
		case err := <-results:
			must.NoError(t, err)
		case <-time.After(time.Second):
			t.Fatal("admitted inbound handler did not finish")
		}
	}
	tunnel.Close()
	assertPendingRelayBudgetEmpty(t, budget)
}

func TestRTunnelProcessInboundReleasesBudgetOnEveryTerminalPath(t *testing.T) {
	budget := newPendingRelayBudget()
	tunnel := NewAuthorizedRTunnel(92008, "session", AuthorizationID("authorization"), &countWriteCloser{})
	tunnel.pendingBudget = budget

	_, err := tunnel.ProcessInbound(maxReverseTunnelPendingFrames, []byte("window"), func([]byte) error { return nil })
	assert.ErrorIs(t, err, ErrReverseTunnelWindow)
	assertPendingRelayBudgetEmpty(t, budget)

	_, err = tunnel.ProcessInbound(0, []byte("written"), func([]byte) error { return nil })
	must.NoError(t, err)
	_, err = tunnel.ProcessInbound(0, []byte("old"), func([]byte) error { return nil })
	must.NoError(t, err)
	assertPendingRelayBudgetEmpty(t, budget)

	writeErr := errors.New("write failed")
	_, err = tunnel.ProcessInbound(1, []byte("retained"), func([]byte) error { return writeErr })
	assert.ErrorIs(t, err, writeErr)
	budget.mutex.Lock()
	assert.Equal(t, int64(len("retained")), budget.total)
	budget.mutex.Unlock()
	tunnel.Close()
	assertPendingRelayBudgetEmpty(t, budget)
}

func TestRTunnelConcurrentCloseAndInboundFailsClosed(t *testing.T) {
	for iteration := 0; iteration < 100; iteration++ {
		budget := newPendingRelayBudget()
		tunnel := NewAuthorizedRTunnel(uint64(93000+iteration), "session", AuthorizationID("authorization"), &countWriteCloser{})
		tunnel.pendingBudget = budget

		start := make(chan struct{})
		result := make(chan error, 1)
		go func() {
			<-start
			_, err := tunnel.ProcessInbound(0, []byte("data"), func([]byte) error { return nil })
			result <- err
		}()
		close(start)
		tunnel.Close()
		err := <-result
		if err != nil && !errors.Is(err, ErrReverseTunnelClosed) {
			t.Fatalf("ProcessInbound() race error = %v, want nil or closed", err)
		}
		assertPendingRelayBudgetEmpty(t, budget)
	}
}

func TestPendingRelayBudgetEnforcesAggregateLimitsAndSafeRelease(t *testing.T) {
	budget := newPendingRelayBudget()

	must.NoError(t, budget.reserve("auth-session", "auth-a", int(maxPendingBytesPerAuthorization)))
	assert.ErrorIs(t, budget.reserve("auth-session", "auth-a", 1), ErrReverseTunnelAuthBudget)
	budget.release("auth-session", "auth-a", int(maxPendingBytesPerAuthorization))
	assertPendingRelayBudgetEmpty(t, budget)

	for _, authorizationID := range []AuthorizationID{"session-a", "session-b"} {
		must.NoError(t, budget.reserve("session", authorizationID, int(maxPendingBytesPerAuthorization)))
	}
	assert.ErrorIs(t, budget.reserve("session", "session-c", 1), ErrReverseTunnelSessionBudget)
	for _, authorizationID := range []AuthorizationID{"session-a", "session-b"} {
		budget.release("session", authorizationID, int(maxPendingBytesPerAuthorization))
	}
	assertPendingRelayBudgetEmpty(t, budget)

	for sessionIndex := 0; sessionIndex < int(maxPendingBytesGlobal/maxPendingBytesPerSession); sessionIndex++ {
		sessionID := fmt.Sprintf("global-session-%d", sessionIndex)
		for authorizationIndex := 0; authorizationIndex < int(maxPendingBytesPerSession/maxPendingBytesPerAuthorization); authorizationIndex++ {
			authorizationID := AuthorizationID(fmt.Sprintf("auth-%d", authorizationIndex))
			must.NoError(t, budget.reserve(sessionID, authorizationID, int(maxPendingBytesPerAuthorization)))
		}
	}
	assert.ErrorIs(t, budget.reserve("overflow-session", "overflow-auth", 1), ErrReverseTunnelGlobalBudget)

	budget.release("global-session-0", "auth-0", int(maxPendingBytesPerAuthorization*2))
	budget.mutex.Lock()
	wantAfterRelease := maxPendingBytesGlobal - maxPendingBytesPerAuthorization
	assert.Equal(t, wantAfterRelease, budget.total)
	budget.mutex.Unlock()
	budget.release("global-session-0", "auth-0", int(maxPendingBytesPerAuthorization))
	budget.mutex.Lock()
	assert.Equal(t, wantAfterRelease, budget.total, "double release must not consume another authorization's budget")
	budget.mutex.Unlock()
}

func assertPendingRelayBudgetEmpty(t *testing.T, budget *pendingRelayBudget) {
	t.Helper()
	budget.mutex.Lock()
	defer budget.mutex.Unlock()
	assert.Zero(t, budget.total)
	assert.Empty(t, budget.bySession)
	assert.Empty(t, budget.byAuthorization)
}

func TestTryAddRTunnelAndConditionalRemovalRejectStaleCleanup(t *testing.T) {
	const tunnelID = uint64(91002)

	first := NewRTunnel(tunnelID, "first", &countWriteCloser{})
	duplicate := NewRTunnel(tunnelID, "second", &countWriteCloser{})
	t.Cleanup(func() {
		RemoveRTunnelIf(tunnelID, first)
	})
	must.True(t, TryAddRTunnel(first))
	assert.False(t, TryAddRTunnel(duplicate))
	assert.Same(t, first, GetRTunnel(tunnelID))
	assert.False(t, RemoveRTunnelIf(tunnelID, duplicate))
	assert.Same(t, first, GetRTunnel(tunnelID))
	assert.True(t, RemoveRTunnelIf(tunnelID, first))
	assert.Nil(t, GetRTunnel(tunnelID))
	assert.False(t, RemoveRTunnelIf(tunnelID, first))
	assert.False(t, TryAddRTunnel(nil))
}

func TestCloseAuthorizationAndCloseSessionAreScopedAndIdempotent(t *testing.T) {
	const (
		matchingID  = uint64(91003)
		otherAuthID = uint64(91004)
		otherSID    = uint64(91005)
	)
	t.Cleanup(func() {
		CloseSession("session")
		CloseSession("other-session")
	})

	matchingWriter := &countWriteCloser{}
	otherAuthWriter := &countWriteCloser{}
	otherSessionWriter := &countWriteCloser{}
	matching := NewAuthorizedRTunnel(matchingID, "session", AuthorizationID("auth-a"), matchingWriter)
	otherAuth := NewAuthorizedRTunnel(otherAuthID, "session", AuthorizationID("auth-b"), otherAuthWriter)
	otherSession := NewAuthorizedRTunnel(otherSID, "other-session", AuthorizationID("auth-a"), otherSessionWriter)
	must.True(t, TryAddRTunnel(matching))
	must.True(t, TryAddRTunnel(otherAuth))
	must.True(t, TryAddRTunnel(otherSession))

	assert.Equal(t, 1, CloseAuthorization("session", AuthorizationID("auth-a")))
	assert.Equal(t, int32(1), matchingWriter.closes.Load())
	assert.Zero(t, otherAuthWriter.closes.Load())
	assert.Zero(t, otherSessionWriter.closes.Load())
	assert.Equal(t, 0, CloseAuthorization("session", AuthorizationID("auth-a")))
	assert.Equal(t, 0, CloseAuthorization("session", ""))

	assert.Equal(t, 1, CloseSession("session"))
	assert.Equal(t, int32(1), otherAuthWriter.closes.Load())
	assert.Zero(t, otherSessionWriter.closes.Load())
	assert.Equal(t, 0, CloseSession("session"))
	assert.Equal(t, 1, CloseSession("other-session"))
	assert.Equal(t, int32(1), otherSessionWriter.closes.Load())
}

type countReadCloser struct {
	closes atomic.Int32
}

func (*countReadCloser) Read(_ []byte) (int, error) {
	return 0, io.EOF
}

func (closer *countReadCloser) Close() error {
	closer.closes.Add(1)
	return nil
}

type countWriteCloser struct {
	closes atomic.Int32
}

func (*countWriteCloser) Write(buffer []byte) (int, error) {
	return len(buffer), nil
}

func (closer *countWriteCloser) Close() error {
	closer.closes.Add(1)
	return nil
}
