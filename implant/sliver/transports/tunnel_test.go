package transports

import (
	"bytes"
	"errors"
	"io"
	"sort"
	"sync"
	"sync/atomic"
	"testing"
)

func TestTunnelQueueOutboundSequencesConcurrentWriters(t *testing.T) {
	const writers = 128

	tunnel := NewTunnel(1, nopWriteCloser{Writer: io.Discard})
	sequences := make([]uint64, writers)
	start := make(chan struct{})
	var ready sync.WaitGroup
	var done sync.WaitGroup
	ready.Add(writers)
	done.Add(writers)

	for index := range sequences {
		go func(index int) {
			defer done.Done()
			ready.Done()
			<-start
			if err := tunnel.queueOutbound(func(sequence uint64) error {
				sequences[index] = sequence
				return nil
			}); err != nil {
				t.Errorf("queueOutbound(): %v", err)
			}
		}(index)
	}

	ready.Wait()
	close(start)
	done.Wait()

	sort.Slice(sequences, func(i, j int) bool { return sequences[i] < sequences[j] })
	for index, sequence := range sequences {
		if want := uint64(index); sequence != want {
			t.Fatalf("sequence[%d] = %d, want %d", index, sequence, want)
		}
	}
}

func TestTunnelPeerCloseWaitsForExclusiveTerminalSequence(t *testing.T) {
	tunnel := NewTunnel(1, nopWriteCloser{Writer: io.Discard})
	ready, err := tunnel.MarkPeerClose(2)
	if err != nil || ready {
		t.Fatalf("MarkPeerClose(2) = (%v, %v), want (false, nil)", ready, err)
	}
	if _, err := tunnel.ProcessInbound(1, []byte("second"), func([]byte) error { return nil }); err != nil {
		t.Fatalf("sequence before terminal rejected: %v", err)
	}
	if tunnel.PeerCloseReady() {
		t.Fatal("terminal close became ready before the final frame")
	}
	var output bytes.Buffer
	if _, err := tunnel.ProcessInbound(0, []byte("first"), func(data []byte) error {
		_, writeErr := output.Write(data)
		return writeErr
	}); err != nil {
		t.Fatalf("process final frames: %v", err)
	}
	if !tunnel.PeerCloseReady() {
		t.Fatal("terminal close did not become ready after all frames")
	}
	if _, err := tunnel.ProcessInbound(2, []byte("late"), func([]byte) error { return nil }); !errors.Is(err, ErrTunnelTerminalSequence) {
		t.Fatalf("sequence at terminal error = %v, want %v", err, ErrTunnelTerminalSequence)
	}
}

func TestTunnelAcceptsFullTransportReorderWindowBeforeTerminal(t *testing.T) {
	tunnel := NewTunnel(2, nopWriteCloser{Writer: io.Discard})
	ready, err := tunnel.MarkPeerClose(maxTunnelPendingFrames)
	if err != nil || ready {
		t.Fatalf("MarkPeerClose(window) = (%v, %v), want (false, nil)", ready, err)
	}
	var output bytes.Buffer
	for sequence := maxTunnelPendingFrames; sequence > 0; sequence-- {
		value := byte(sequence - 1)
		if _, err := tunnel.ProcessInbound(uint64(sequence-1), []byte{value}, func(data []byte) error {
			_, writeErr := output.Write(data)
			return writeErr
		}); err != nil {
			t.Fatalf("ProcessInbound(%d): %v", sequence-1, err)
		}
	}
	if !tunnel.PeerCloseReady() {
		t.Fatal("terminal did not become ready after a full reverse-order window")
	}
	if output.Len() != maxTunnelPendingFrames {
		t.Fatalf("output length = %d, want %d", output.Len(), maxTunnelPendingFrames)
	}
	for index, value := range output.Bytes() {
		if value != byte(index) {
			t.Fatalf("output[%d] = %d, want %d", index, value, byte(index))
		}
	}
}

func TestTunnelLegacyZeroCloseIsImmediateWithPendingFrames(t *testing.T) {
	tunnel := NewTunnel(4, nopWriteCloser{Writer: io.Discard})
	if pending, err := tunnel.ProcessInbound(1, []byte("pending"), func([]byte) error { return nil }); err != nil || pending != 1 {
		t.Fatalf("ProcessInbound(1) = (%d, %v), want (1, nil)", pending, err)
	}
	ready, err := tunnel.MarkPeerClose(0)
	if err != nil || !ready {
		t.Fatalf("MarkPeerClose(0) = (%v, %v), want (true, nil)", ready, err)
	}
}

func TestTunnelRemoteTeardownMarkerDoesNotInstallTerminal(t *testing.T) {
	tunnel := NewTunnel(5, nopWriteCloser{Writer: io.Discard})
	// Reproduce the closeRemote linearization window after peer ownership is
	// published but before Done is closed.
	tunnel.peerTeardown.Store(true)
	var output bytes.Buffer
	pending, err := tunnel.ProcessInbound(0, []byte("in-flight"), func(payload []byte) error {
		_, writeErr := output.Write(payload)
		return writeErr
	})
	if err != nil {
		t.Fatalf("ProcessInbound() during remote teardown marker: %v", err)
	}
	if pending != 0 || output.String() != "in-flight" {
		t.Fatalf("ProcessInbound() = pending %d, output %q", pending, output.String())
	}
	if !tunnel.PeerTeardownPending() {
		t.Fatal("remote teardown ownership was not published")
	}
	if tunnel.PeerClosePending() {
		t.Fatal("remote teardown marker installed a terminal sequence")
	}
	tunnel.Close()
}

func TestTunnelCloseLocalSequencesAndNotifiesExactlyOnce(t *testing.T) {
	tunnel := NewTunnel(3, nopWriteCloser{Writer: io.Discard})
	var notifications atomic.Int32
	var terminal atomic.Uint64
	tunnel.setPeerCloseNotifier(func(sequence uint64) error {
		notifications.Add(1)
		terminal.Store(sequence)
		return nil
	})
	if err := tunnel.queueOutbound(func(sequence uint64) error {
		if sequence != 0 {
			t.Fatalf("first sequence = %d, want 0", sequence)
		}
		return nil
	}); err != nil {
		t.Fatalf("queueOutbound(): %v", err)
	}

	var closes sync.WaitGroup
	closes.Add(16)
	for range 16 {
		go func() {
			defer closes.Done()
			_, _ = tunnel.closeLocal()
		}()
	}
	closes.Wait()
	if got := notifications.Load(); got != 1 {
		t.Fatalf("notifications = %d, want 1", got)
	}
	if got := terminal.Load(); got != 1 {
		t.Fatalf("terminal sequence = %d, want 1", got)
	}
}

type nopWriteCloser struct {
	io.Writer
}

func (nopWriteCloser) Close() error { return nil }
