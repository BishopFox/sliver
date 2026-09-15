package core

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/bishopfox/sliver/protobuf/sliverpb"
)

func TestImplantConnectionSendEnvelopeTimesOutWhenTransportStalls(t *testing.T) {
	connection := NewImplantConnection("test", "test")
	err := connection.SendEnvelope(&sliverpb.Envelope{Type: sliverpb.MsgPing}, 20*time.Millisecond)
	if !errors.Is(err, ErrImplantSendTimeout) {
		t.Fatalf("SendEnvelope error = %v, want %v", err, ErrImplantSendTimeout)
	}
}

func TestImplantConnectionSendEnvelopeUnblocksOnDisconnect(t *testing.T) {
	connection := NewImplantConnection("test", "test")
	connection.Send = make(chan *sliverpb.Envelope, 1)
	queued := &sliverpb.Envelope{Type: sliverpb.MsgPing, Data: []byte("queued")}
	connection.Send <- queued
	result := make(chan error, 1)
	started := make(chan struct{})
	go func() {
		close(started)
		result <- connection.SendEnvelope(&sliverpb.Envelope{Type: sliverpb.MsgPing, Data: []byte("blocked")}, time.Hour)
	}()
	<-started
	connection.Close()

	select {
	case err := <-result:
		if !errors.Is(err, ErrImplantConnectionClosed) {
			t.Fatalf("SendEnvelope error = %v, want %v", err, ErrImplantConnectionClosed)
		}
	case <-time.After(time.Second):
		t.Fatal("SendEnvelope remained blocked after connection close")
	}
	if got := <-connection.Send; got != queued {
		t.Fatalf("full queue was overwritten after disconnect: got %+v, want %+v", got, queued)
	}
	select {
	case unexpected := <-connection.Send:
		t.Fatalf("blocked envelope was queued after disconnect: %+v", unexpected)
	default:
	}
}

func TestImplantConnectionSendEnvelopeUnblocksOnOwnerClose(t *testing.T) {
	connection := NewImplantConnection("test", "test")
	ownerDone := make(chan struct{})
	result := make(chan error, 1)
	go func() {
		result <- connection.SendEnvelopeUntil(&sliverpb.Envelope{Type: sliverpb.MsgPing}, ownerDone, time.Hour)
	}()
	close(ownerDone)

	select {
	case err := <-result:
		if !errors.Is(err, ErrImplantConnectionClosed) {
			t.Fatalf("SendEnvelopeUntil error = %v, want %v", err, ErrImplantConnectionClosed)
		}
	case <-time.After(time.Second):
		t.Fatal("SendEnvelopeUntil remained blocked after owner close")
	}
}

func TestImplantConnectionSendEnvelopeRejectsClosedConnection(t *testing.T) {
	connection := NewImplantConnection("test", "test")
	connection.Send = make(chan *sliverpb.Envelope, 1)
	connection.Close()

	err := connection.SendEnvelope(&sliverpb.Envelope{Type: sliverpb.MsgPing}, time.Second)
	if !errors.Is(err, ErrImplantConnectionClosed) {
		t.Fatalf("SendEnvelope error = %v, want %v", err, ErrImplantConnectionClosed)
	}
	select {
	case envelope := <-connection.Send:
		t.Fatalf("closed connection accepted envelope: %+v", envelope)
	default:
	}
}

func TestImplantConnectionDeliverResponseNeverBlocks(t *testing.T) {
	connection := NewImplantConnection("test", "test")
	response := make(chan *sliverpb.Envelope, 1)
	connection.Resp[7] = response

	if !connection.DeliverResponse(&sliverpb.Envelope{ID: 7, Data: []byte("first")}) {
		t.Fatal("first response was not delivered")
	}
	if connection.DeliverResponse(&sliverpb.Envelope{ID: 7, Data: []byte("duplicate")}) {
		t.Fatal("duplicate response was delivered to a full waiter")
	}
	if connection.DeliverResponse(&sliverpb.Envelope{ID: 8}) {
		t.Fatal("response was delivered to an unknown waiter")
	}
}

func TestImplantConnectionCloseIsIdempotent(t *testing.T) {
	connection := NewImplantConnection("test", "test")
	var cleanupCalls atomic.Int32
	if !connection.SetCleanup(func() {
		cleanupCalls.Add(1)
	}) {
		t.Fatal("failed to install cleanup")
	}

	select {
	case <-connection.Done():
		t.Fatal("new connection is already closed")
	default:
	}

	const closers = 32
	var waitGroup sync.WaitGroup
	waitGroup.Add(closers)
	for range closers {
		go func() {
			defer waitGroup.Done()
			connection.Close()
		}()
	}
	waitGroup.Wait()

	select {
	case <-connection.Done():
	case <-time.After(time.Second):
		t.Fatal("connection Done was not closed")
	}
	if got := cleanupCalls.Load(); got != 1 {
		t.Fatalf("cleanup called %d times, want 1", got)
	}

	connection.Close()
	if got := cleanupCalls.Load(); got != 1 {
		t.Fatalf("cleanup called %d times after repeated Close, want 1", got)
	}
}

func TestImplantConnectionDonePrecedesCleanup(t *testing.T) {
	connection := NewImplantConnection("test", "test")
	cleanupStarted := make(chan struct{})
	releaseCleanup := make(chan struct{})
	if !connection.SetCleanup(func() {
		close(cleanupStarted)
		<-releaseCleanup
	}) {
		t.Fatal("failed to install cleanup")
	}

	go connection.Close()
	select {
	case <-cleanupStarted:
	case <-time.After(time.Second):
		t.Fatal("cleanup did not start")
	}
	select {
	case <-connection.Done():
	case <-time.After(time.Second):
		t.Fatal("Done did not close while cleanup was blocked")
	}
	close(releaseCleanup)
}

func TestImplantConnectionZeroValueCloseWithNilCleanup(t *testing.T) {
	connection := &ImplantConnection{}
	connection.Close()
	select {
	case <-connection.Done():
	case <-time.After(time.Second):
		t.Fatal("zero-value connection Done was not closed")
	}
	connection.Close()
}

func TestImplantConnectionCleanupRegistrationIsOneShotAndClosedAware(t *testing.T) {
	connection := NewImplantConnection("test", "test")
	var firstCalls atomic.Int32
	var duplicateCalls atomic.Int32
	if !connection.SetCleanup(func() { firstCalls.Add(1) }) {
		t.Fatal("first cleanup registration was rejected")
	}
	if connection.SetCleanup(func() { duplicateCalls.Add(1) }) {
		t.Fatal("duplicate cleanup registration was accepted")
	}
	connection.Close()
	if got := firstCalls.Load(); got != 1 {
		t.Fatalf("first cleanup called %d times, want 1", got)
	}
	if got := duplicateCalls.Load(); got != 0 {
		t.Fatalf("duplicate cleanup called %d times, want 0", got)
	}

	closed := NewImplantConnection("test", "test")
	closed.Close()
	if closed.SetCleanup(func() { t.Error("cleanup ran after closed registration") }) {
		t.Fatal("cleanup registration succeeded after Close")
	}
}

func TestImplantConnectionNeverReusesReverseTunnelID(t *testing.T) {
	connection := NewImplantConnection("test", "test")
	if result := connection.TryClaimReverseTunnelID(42); result != ReverseTunnelIDClaimed {
		t.Fatalf("first tunnel ID claim result = %v, want %v", result, ReverseTunnelIDClaimed)
	}
	if result := connection.TryClaimReverseTunnelID(42); result != ReverseTunnelIDDuplicate {
		t.Fatalf("reused tunnel ID claim result = %v, want %v", result, ReverseTunnelIDDuplicate)
	}
	connection.Close()
	if result := connection.TryClaimReverseTunnelID(43); result != ReverseTunnelIDConnectionClosed {
		t.Fatalf("closed connection claim result = %v, want %v", result, ReverseTunnelIDConnectionClosed)
	}
}

func TestImplantConnectionClaimedReverseTunnelIDCapacityFailsClosed(t *testing.T) {
	connection := NewImplantConnection("test", "test")
	cleanupCalled := make(chan struct{})
	if !connection.SetCleanup(func() {
		// Re-entering a lifecycle method proves the exhaustion path does not
		// invoke Close while holding lifecycleMutex.
		if result := connection.TryClaimReverseTunnelID(0); result != ReverseTunnelIDConnectionClosed {
			t.Errorf("cleanup reentrant claim result = %v, want %v", result, ReverseTunnelIDConnectionClosed)
		}
		close(cleanupCalled)
	}) {
		t.Fatal("failed to install cleanup")
	}

	for tunnelID := uint64(0); tunnelID < maxClaimedReverseTunnelIDsPerConnection; tunnelID++ {
		if result := connection.TryClaimReverseTunnelID(tunnelID); result != ReverseTunnelIDClaimed {
			t.Fatalf("claim %d result = %v, want %v", tunnelID, result, ReverseTunnelIDClaimed)
		}
	}

	// Duplicate detection takes precedence over capacity, so an ordinary replay
	// remains distinguishable and does not close the connection.
	if result := connection.TryClaimReverseTunnelID(0); result != ReverseTunnelIDDuplicate {
		t.Fatalf("duplicate-at-capacity result = %v, want %v", result, ReverseTunnelIDDuplicate)
	}
	select {
	case <-connection.Done():
		t.Fatal("duplicate-at-capacity closed the connection")
	default:
	}

	result := make(chan ReverseTunnelIDClaimResult, 1)
	go func() {
		result <- connection.TryClaimReverseTunnelID(maxClaimedReverseTunnelIDsPerConnection)
	}()
	select {
	case claimResult := <-result:
		if claimResult != ReverseTunnelIDCapacityExhausted {
			t.Fatalf("capacity exhaustion result = %v, want %v", claimResult, ReverseTunnelIDCapacityExhausted)
		}
	case <-time.After(time.Second):
		t.Fatal("capacity exhaustion deadlocked")
	}
	select {
	case <-cleanupCalled:
	case <-time.After(time.Second):
		t.Fatal("capacity exhaustion deadlocked before cleanup completed")
	}
	select {
	case <-connection.Done():
	default:
		t.Fatal("capacity exhaustion did not fail the connection closed")
	}
}
