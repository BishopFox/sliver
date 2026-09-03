package transports

import (
	"context"
	"errors"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/bishopfox/sliver/protobuf/sliverpb"
)

type closeSignalWriteCloser struct {
	closed chan struct{}
	once   sync.Once
}

func newCloseSignalWriteCloser() *closeSignalWriteCloser {
	return &closeSignalWriteCloser{closed: make(chan struct{})}
}

func (*closeSignalWriteCloser) Write(data []byte) (int, error) { return len(data), nil }

func (c *closeSignalWriteCloser) Close() error {
	c.once.Do(func() { close(c.closed) })
	return nil
}

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

func TestConnectionPendingTunnelCloseRejectsLatePublication(t *testing.T) {
	connection := &Connection{Send: make(chan *sliverpb.Envelope, 1)}
	pending, result := connection.BeginTunnel(0x7711, time.Hour)
	if result != TunnelAdded {
		t.Fatalf("begin pending tunnel result = %v, want %v", result, TunnelAdded)
	}
	if !connection.CancelPendingTunnel(0x7711) {
		t.Fatal("pending tunnel close did not cancel the setup")
	}
	select {
	case <-pending.Context().Done():
		if !errors.Is(pending.Context().Err(), context.Canceled) {
			t.Fatalf("pending context error = %v, want %v", pending.Context().Err(), context.Canceled)
		}
	case <-time.After(time.Second):
		t.Fatal("pending tunnel close did not cancel its setup context")
	}

	writer := newCloseSignalWriteCloser()
	late := NewTunnel(0x7711, writer)
	if result := connection.PublishTunnel(pending, late); result != TunnelAddSetupCanceled {
		t.Fatalf("late publication result = %v, want %v", result, TunnelAddSetupCanceled)
	}
	late.Close()
	select {
	case <-writer.closed:
	default:
		t.Fatal("rejected late tunnel resource was not closed by its caller")
	}
	if active := connection.Tunnel(0x7711); active != nil {
		t.Fatalf("late tunnel remained published: %p", active)
	}
	if replacement, result := connection.BeginTunnel(0x7711, time.Hour); result != TunnelAddDuplicate || replacement != nil {
		t.Fatalf("retired pending ID replacement = %p, %v; want nil, %v", replacement, result, TunnelAddDuplicate)
	}
}

func TestConnectionPendingTunnelOwnerDisconnectAndDeadlineCancel(t *testing.T) {
	tests := []struct {
		name    string
		timeout time.Duration
		close   bool
		wantErr error
		wantAdd TunnelAddResult
	}{
		{name: "owner disconnect", timeout: time.Hour, close: true, wantErr: context.Canceled, wantAdd: TunnelAddConnectionClosed},
		{name: "setup deadline", timeout: 20 * time.Millisecond, wantErr: context.DeadlineExceeded, wantAdd: TunnelAddSetupCanceled},
	}
	for index, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			connection := &Connection{Send: make(chan *sliverpb.Envelope, 1)}
			tunnelID := uint64(0x7720 + index)
			pending, result := connection.BeginTunnel(tunnelID, test.timeout)
			if result != TunnelAdded {
				t.Fatalf("begin pending tunnel result = %v, want %v", result, TunnelAdded)
			}
			if test.close {
				connection.Cleanup()
			}
			select {
			case <-pending.Context().Done():
				if !errors.Is(pending.Context().Err(), test.wantErr) {
					t.Fatalf("pending context error = %v, want %v", pending.Context().Err(), test.wantErr)
				}
			case <-time.After(time.Second):
				t.Fatal("pending setup context was not canceled")
			}

			late := NewTunnel(tunnelID, nopWriteCloser{Writer: io.Discard})
			if result := connection.PublishTunnel(pending, late); result != test.wantAdd {
				t.Fatalf("late publication result = %v, want %v", result, test.wantAdd)
			}
			late.Close()
			if active := connection.Tunnel(tunnelID); active != nil {
				t.Fatalf("late tunnel remained published: %p", active)
			}
		})
	}
}

// This lifecycle test intentionally exercises admission, retirement, replay,
// generation identity, and connection survival in one ordered scenario.
//
//nolint:gocyclo
func TestConnectionRetiredTunnelWindowRotatesWithoutClosingConnection(t *testing.T) {
	cleanupCalled := make(chan struct{})
	connection := &Connection{
		Send: make(chan *sliverpb.Envelope, 1),
		cleanup: func() {
			close(cleanupCalled)
		},
	}

	var first *Tunnel
	total := maxRetiredTunnelIDsPerConnection + 64
	for tunnelID := uint64(0); tunnelID < uint64(total); tunnelID++ {
		tunnel := NewTunnel(tunnelID, nopWriteCloser{Writer: io.Discard})
		if tunnelID == 0 {
			first = tunnel
		}
		if result := connection.TryAddTunnel(tunnel); result != TunnelAdded {
			t.Fatalf("add tunnel %d result = %v, want %v", tunnelID, result, TunnelAdded)
		}
		if !connection.CloseTunnelRemote(tunnel) {
			t.Fatalf("failed to retire tunnel %d", tunnelID)
		}
	}

	select {
	case <-cleanupCalled:
		t.Fatal("rotating retired tunnel IDs cleaned up the connection")
	case <-connection.Done():
		t.Fatal("rotating retired tunnel IDs closed the connection")
	default:
	}
	connection.mutex.RLock()
	retiredCount := len(connection.retiredTunnels)
	retiredOrderCount := len(connection.retiredOrder)
	connection.mutex.RUnlock()
	if retiredCount != maxRetiredTunnelIDsPerConnection || retiredOrderCount != maxRetiredTunnelIDsPerConnection {
		t.Fatalf(
			"retired replay window = map:%d order:%d, want %d each",
			retiredCount,
			retiredOrderCount,
			maxRetiredTunnelIDsPerConnection,
		)
	}

	newestID := uint64(total - 1)
	duplicate := NewTunnel(newestID, nopWriteCloser{Writer: io.Discard})
	if result := connection.TryAddTunnel(duplicate); result != TunnelAddDuplicate {
		t.Fatalf("recent retired tunnel ID result = %v, want %v", result, TunnelAddDuplicate)
	}
	duplicate.Close()

	// ID zero has rotated out of the bounded wire replay window. A fresh random
	// collision may reuse it, but stale in-process work still carries the exact
	// old pointer and cannot remove the replacement generation.
	replacement := NewTunnel(0, nopWriteCloser{Writer: io.Discard})
	if result := connection.TryAddTunnel(replacement); result != TunnelAdded {
		t.Fatalf("post-window replacement result = %v, want %v", result, TunnelAdded)
	}
	if connection.CloseTunnelRemote(first) {
		t.Fatal("stale exact tunnel generation closed its replacement")
	}
	if active := connection.Tunnel(replacement.ID); active != replacement {
		t.Fatalf("replacement tunnel = %p, want %p", active, replacement)
	}
	if !connection.CloseTunnelRemote(replacement) {
		t.Fatal("failed to close replacement tunnel")
	}

	select {
	case <-connection.Done():
		t.Fatal("replacement lifecycle closed the connection")
	case <-cleanupCalled:
		t.Fatal("replacement lifecycle cleaned up the connection")
	default:
	}
}

func TestConnectionLiveTunnelCapacityRejectsOnlyNewGeneration(t *testing.T) {
	cleanupCalled := make(chan struct{})
	connection := &Connection{
		Send: make(chan *sliverpb.Envelope, 1),
		cleanup: func() {
			close(cleanupCalled)
		},
	}
	tunnels := make([]*Tunnel, 0, maxLiveTunnelIDsPerConnection)
	for tunnelID := uint64(0); tunnelID < uint64(maxLiveTunnelIDsPerConnection); tunnelID++ {
		tunnel := NewTunnel(tunnelID, nopWriteCloser{Writer: io.Discard})
		if result := connection.TryAddTunnel(tunnel); result != TunnelAdded {
			t.Fatalf("add live tunnel %d result = %v, want %v", tunnelID, result, TunnelAdded)
		}
		tunnels = append(tunnels, tunnel)
	}

	overflow := NewTunnel(uint64(maxLiveTunnelIDsPerConnection), nopWriteCloser{Writer: io.Discard})
	if result := connection.TryAddTunnel(overflow); result != TunnelAddCapacityExhausted {
		t.Fatalf("live capacity result = %v, want %v", result, TunnelAddCapacityExhausted)
	}
	overflow.Close()
	select {
	case <-connection.Done():
		t.Fatal("live capacity rejection closed the connection")
	case <-cleanupCalled:
		t.Fatal("live capacity rejection cleaned up the connection")
	default:
	}

	if !connection.CloseTunnelRemote(tunnels[0]) {
		t.Fatal("failed to retire one live tunnel")
	}
	replacement := NewTunnel(uint64(maxLiveTunnelIDsPerConnection), nopWriteCloser{Writer: io.Discard})
	if result := connection.TryAddTunnel(replacement); result != TunnelAdded {
		t.Fatalf("add after live capacity release = %v, want %v", result, TunnelAdded)
	}
	connection.Cleanup()
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
