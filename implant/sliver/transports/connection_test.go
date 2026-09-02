package transports

import (
	"errors"
	"io"
	"testing"
	"time"

	"github.com/bishopfox/sliver/protobuf/sliverpb"
)

func TestNextSendEnvelopeSkipsNilAndStopsWithConnection(t *testing.T) {
	t.Run("skips nil envelope", func(t *testing.T) {
		connection := &Connection{}
		send := make(chan *sliverpb.Envelope, 2)
		send <- nil
		want := &sliverpb.Envelope{Type: sliverpb.MsgTunnelData}
		send <- want

		got, ok := nextSendEnvelope(connection, send)
		if !ok {
			t.Fatal("nextSendEnvelope stopped while the connection was live")
		}
		if got != want {
			t.Fatalf("nextSendEnvelope() = %p, want %p", got, want)
		}
	})

	t.Run("stops on connection cleanup", func(t *testing.T) {
		connection := &Connection{}
		send := make(chan *sliverpb.Envelope)
		result := make(chan bool, 1)
		go func() {
			_, ok := nextSendEnvelope(connection, send)
			result <- ok
		}()

		connection.Cleanup()
		select {
		case ok := <-result:
			if ok {
				t.Fatal("nextSendEnvelope accepted an envelope after cleanup")
			}
		case <-time.After(time.Second):
			t.Fatal("nextSendEnvelope remained blocked after cleanup")
		}
	})
}

func TestConnectionRejectsRetiredTunnelID(t *testing.T) {
	connection := &Connection{Send: make(chan *sliverpb.Envelope, 1)}
	first := NewTunnel(77, nopWriteCloser{Writer: io.Discard})
	if result := connection.TryAddTunnel(first); result != TunnelAdded {
		t.Fatalf("first tunnel generation result = %v, want %v", result, TunnelAdded)
	}
	if !connection.CloseTunnelRemote(first) {
		t.Fatal("failed to close first tunnel generation")
	}
	second := NewTunnel(77, nopWriteCloser{Writer: io.Discard})
	if result := connection.TryAddTunnel(second); result != TunnelAddDuplicate {
		t.Fatalf("retired tunnel ID result = %v, want %v", result, TunnelAddDuplicate)
	}
	second.Close()
}

func TestConnectionClaimedTunnelIDCapacityFailsClosed(t *testing.T) {
	cleanupCalled := make(chan struct{})
	connection := &Connection{
		Send: make(chan *sliverpb.Envelope, 1),
		cleanup: func() {
			close(cleanupCalled)
		},
	}

	for tunnelID := uint64(0); tunnelID < maxClaimedTunnelIDsPerConnection; tunnelID++ {
		tunnel := NewTunnel(tunnelID, nopWriteCloser{Writer: io.Discard})
		if result := connection.TryAddTunnel(tunnel); result != TunnelAdded {
			t.Fatalf("add tunnel %d result = %v, want %v", tunnelID, result, TunnelAdded)
		}
		if !connection.CloseTunnelRemote(tunnel) {
			t.Fatalf("failed to retire tunnel %d", tunnelID)
		}
	}

	duplicate := NewTunnel(0, nopWriteCloser{Writer: io.Discard})
	if result := connection.TryAddTunnel(duplicate); result != TunnelAddDuplicate {
		t.Fatalf("duplicate-at-capacity result = %v, want %v", result, TunnelAddDuplicate)
	}
	duplicate.Close()
	select {
	case <-connection.Done():
		t.Fatal("duplicate-at-capacity closed the connection")
	default:
	}

	overflow := NewTunnel(maxClaimedTunnelIDsPerConnection, nopWriteCloser{Writer: io.Discard})
	result := make(chan TunnelAddResult, 1)
	go func() {
		result <- connection.TryAddTunnel(overflow)
	}()
	select {
	case addResult := <-result:
		if addResult != TunnelAddCapacityExhausted {
			t.Fatalf("capacity exhaustion result = %v, want %v", addResult, TunnelAddCapacityExhausted)
		}
	case <-time.After(time.Second):
		t.Fatal("capacity exhaustion deadlocked")
	}
	overflow.Close()
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

func TestConnectionTunnelCloseUnblocksSaturatedDataSend(t *testing.T) {
	connection := &Connection{Send: make(chan *sliverpb.Envelope)}
	tunnel := NewTunnel(78, nopWriteCloser{Writer: io.Discard})
	if !connection.AddTunnel(tunnel) {
		t.Fatal("failed to add tunnel")
	}
	result := make(chan error, 1)
	go func() {
		result <- connection.QueueTunnelData(tunnel, func(uint64, uint64) (*sliverpb.Envelope, error) {
			return &sliverpb.Envelope{Type: sliverpb.MsgTunnelData}, nil
		})
	}()

	select {
	case err := <-result:
		t.Fatalf("saturated send returned before close: %v", err)
	case <-time.After(25 * time.Millisecond):
	}
	if !connection.CloseTunnelRemote(tunnel) {
		t.Fatal("failed to close saturated tunnel")
	}
	select {
	case err := <-result:
		if !errors.Is(err, ErrTunnelClosed) {
			t.Fatalf("QueueTunnelData() error = %v, want %v", err, ErrTunnelClosed)
		}
	case <-time.After(time.Second):
		t.Fatal("tunnel close did not unblock saturated send")
	}
}

func TestConnectionSimultaneousPeerCloseDoesNotFailC2(t *testing.T) {
	connection := &Connection{Send: make(chan *sliverpb.Envelope)}
	tunnel := NewTunnel(79, nopWriteCloser{Writer: io.Discard})
	if !connection.AddTunnel(tunnel) {
		t.Fatal("failed to add tunnel")
	}

	notifierStarted := make(chan struct{})
	tunnel.setPeerCloseNotifier(func(uint64) error {
		close(notifierStarted)
		<-tunnel.Done()
		return ErrTunnelClosed
	})
	localResult := make(chan bool, 1)
	go func() {
		localResult <- connection.CloseTunnelLocal(tunnel)
	}()
	select {
	case <-notifierStarted:
	case <-time.After(time.Second):
		t.Fatal("local terminal notifier did not start")
	}

	remoteResult := make(chan bool, 1)
	go func() {
		remoteResult <- connection.CloseTunnelRemote(tunnel)
	}()
	select {
	case <-localResult:
	case <-time.After(time.Second):
		t.Fatal("local close did not finish after peer teardown")
	}
	select {
	case <-remoteResult:
	case <-time.After(time.Second):
		t.Fatal("remote close did not finish")
	}

	if active := connection.Tunnel(tunnel.ID); active != nil {
		t.Fatalf("simultaneously closed tunnel remained published: %p", active)
	}
	if !tunnel.PeerTeardownPending() {
		t.Fatal("remote close did not publish peer ownership")
	}
	if tunnel.PeerClosePending() {
		t.Fatal("unsequenced remote close installed a terminal sequence")
	}
	select {
	case <-connection.Done():
		t.Fatal("normal simultaneous peer close failed the C2 connection")
	default:
	}
}
