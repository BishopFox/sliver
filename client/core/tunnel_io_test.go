package core

import (
	"bytes"
	"errors"
	"io"
	"testing"
	"time"
)

func TestTunnelIOBuffersDataBeforeReaderStarts(t *testing.T) {
	tunnel := NewTunnelIO(1, "session")
	if err := tunnel.Open(); err != nil {
		t.Fatalf("open tunnel: %v", err)
	}
	t.Cleanup(func() {
		_ = tunnel.Close()
	})

	if got := cap(tunnel.Recv); got != tunnelRecvBufferSize {
		t.Fatalf("receive buffer capacity = %d, want %d", got, tunnelRecvBufferSize)
	}

	payload := []byte("early shell output")
	recvDone := make(chan error, 1)
	go func() {
		recvDone <- tunnel.RecvData(payload)
	}()

	select {
	case err := <-recvDone:
		if err != nil {
			t.Fatalf("queue data before reader startup: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("RecvData blocked before the tunnel reader started")
	}

	select {
	case got := <-tunnel.Recv:
		if !bytes.Equal(got, payload) {
			t.Fatalf("queued payload = %q, want %q", got, payload)
		}
	default:
		t.Fatal("early payload was not queued")
	}
}

func TestTunnelIOCloseUnblocksFullReceiveBuffer(t *testing.T) {
	tunnel := NewTunnelIO(2, "session")
	if err := tunnel.Open(); err != nil {
		t.Fatalf("open tunnel: %v", err)
	}

	for i := 0; i < tunnelRecvBufferSize; i++ {
		recvDone := make(chan error, 1)
		go func() {
			recvDone <- tunnel.RecvData([]byte{byte(i)})
		}()
		select {
		case err := <-recvDone:
			if err != nil {
				t.Fatalf("fill receive buffer frame %d: %v", i, err)
			}
		case <-time.After(time.Second):
			t.Fatalf("frame %d blocked before receive buffer was full", i)
		}
	}

	blockedRecv := make(chan error, 1)
	go func() {
		blockedRecv <- tunnel.RecvData([]byte("seventeenth frame"))
	}()
	select {
	case err := <-blockedRecv:
		t.Fatalf("frame beyond bounded receive capacity returned before close: %v", err)
	case <-time.After(50 * time.Millisecond):
	}

	closeDone := make(chan error, 1)
	go func() {
		closeDone <- tunnel.Close()
	}()
	select {
	case err := <-closeDone:
		if err != nil {
			t.Fatalf("close full tunnel: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("Close blocked behind RecvData on a full receive buffer")
	}
	select {
	case err := <-blockedRecv:
		if !errors.Is(err, errClosedTunnel) {
			t.Fatalf("blocked RecvData error = %v, want %v", err, errClosedTunnel)
		}
	case <-time.After(time.Second):
		t.Fatal("blocked RecvData did not wake when tunnel closed")
	}
}

func TestTunnelIOCloseUnblocksRead(t *testing.T) {
	tunnel := NewTunnelIO(3, "session")
	if err := tunnel.Open(); err != nil {
		t.Fatalf("open tunnel: %v", err)
	}

	type readResult struct {
		n   int
		err error
	}
	readDone := make(chan readResult, 1)
	go func() {
		n, err := tunnel.Read(make([]byte, 32))
		readDone <- readResult{n: n, err: err}
	}()

	if err := tunnel.Close(); err != nil {
		t.Fatalf("close tunnel: %v", err)
	}
	select {
	case result := <-readDone:
		if result.n != 0 || !errors.Is(result.err, io.EOF) {
			t.Fatalf("Read after close = (%d, %v), want (0, EOF)", result.n, result.err)
		}
	case <-time.After(time.Second):
		t.Fatal("blocked Read did not wake when tunnel closed")
	}
}

func TestTunnelIOCloseUnblocksWrite(t *testing.T) {
	tunnel := NewTunnelIO(4, "session")
	if err := tunnel.Open(); err != nil {
		t.Fatalf("open tunnel: %v", err)
	}

	type writeResult struct {
		n   int
		err error
	}
	writeDone := make(chan writeResult, 1)
	go func() {
		n, err := tunnel.Write([]byte("blocked write"))
		writeDone <- writeResult{n: n, err: err}
	}()
	select {
	case result := <-writeDone:
		t.Fatalf("Write returned before close without a consumer: (%d, %v)", result.n, result.err)
	case <-time.After(50 * time.Millisecond):
	}

	if err := tunnel.Close(); err != nil {
		t.Fatalf("close tunnel: %v", err)
	}
	select {
	case result := <-writeDone:
		if result.n != 0 || !errors.Is(result.err, io.EOF) {
			t.Fatalf("Write after close = (%d, %v), want (0, EOF)", result.n, result.err)
		}
	case <-time.After(time.Second):
		t.Fatal("blocked Write did not wake when tunnel closed")
	}
}

func TestTunnelIOReadDrainsQueuedDataBeforeEOF(t *testing.T) {
	tunnel := NewTunnelIO(5, "session")
	if err := tunnel.Open(); err != nil {
		t.Fatalf("open tunnel: %v", err)
	}
	payload := []byte("trailing output")
	if err := tunnel.RecvData(payload); err != nil {
		t.Fatalf("queue trailing output: %v", err)
	}
	if err := tunnel.Close(); err != nil {
		t.Fatalf("close tunnel: %v", err)
	}

	buf := make([]byte, 32)
	n, err := tunnel.Read(buf)
	if err != nil || !bytes.Equal(buf[:n], payload) {
		t.Fatalf("first Read after close = (%q, %v), want (%q, nil)", buf[:n], err, payload)
	}
	n, err = tunnel.Read(buf)
	if n != 0 || !errors.Is(err, io.EOF) {
		t.Fatalf("second Read after close = (%d, %v), want (0, EOF)", n, err)
	}
}

func TestTunnelIOCloseBeforeOpenIsTerminal(t *testing.T) {
	tunnel := NewTunnelIO(6, "session")
	if err := tunnel.Close(); err != nil {
		t.Fatalf("close tunnel before open: %v", err)
	}
	select {
	case <-tunnel.Done():
	default:
		t.Fatal("Close before Open did not close Done")
	}
	if err := tunnel.Open(); !errors.Is(err, errClosedTunnel) {
		t.Fatalf("Open after Close error = %v, want %v", err, errClosedTunnel)
	}
	if tunnel.IsOpen() {
		t.Fatal("tunnel reopened after terminal Close")
	}
}
