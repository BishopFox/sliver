package core

import (
	"bytes"
	"errors"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/bishopfox/sliver/protobuf/sliverpb"
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

func TestTunnelIOAdmitsHTTPTransportBurstAndDrainsInOrder(t *testing.T) {
	const (
		httpBurstFrames = 64
		httpFrameBytes  = 32 * 1024
	)
	tunnel := NewTunnelIO(10, "session")
	if err := tunnel.Open(); err != nil {
		t.Fatalf("open tunnel: %v", err)
	}
	t.Cleanup(func() { _ = tunnel.Close() })

	// HTTP C2 may make this entire response window runnable before the local
	// port-forward reader is scheduled. Admission must remain nonblocking and
	// preserve every frame rather than treating that ordinary burst as overload.
	for sequence := 0; sequence < httpBurstFrames; sequence++ {
		payload := bytes.Repeat([]byte{byte(sequence)}, httpFrameBytes)
		if err := tunnel.RecvData(payload); err != nil {
			t.Fatalf("admit HTTP burst frame %d: %v", sequence, err)
		}
	}
	if !tunnel.IsOpen() {
		t.Fatal("ordinary HTTP response burst closed the tunnel")
	}

	buffer := make([]byte, httpFrameBytes)
	for sequence := 0; sequence < httpBurstFrames; sequence++ {
		n, err := tunnel.Read(buffer)
		if err != nil {
			t.Fatalf("drain HTTP burst frame %d: %v", sequence, err)
		}
		if n != len(buffer) {
			t.Fatalf("drain HTTP burst frame %d length = %d, want %d", sequence, n, len(buffer))
		}
		want := byte(sequence)
		for offset, got := range buffer {
			if got != want {
				t.Fatalf("drain HTTP burst frame %d byte %d = %d, want %d", sequence, offset, got, want)
			}
		}
	}

	tunnel.recvBudgetMu.Lock()
	frames, size := tunnel.recvFrames, tunnel.recvBytes
	tunnel.recvBudgetMu.Unlock()
	if frames != 0 || size != 0 {
		t.Fatalf("receive budget after HTTP burst drain = (%d frames, %d bytes), want zero", frames, size)
	}
}

func TestTunnelIORejectsFullReceiveBufferWithoutBlocking(t *testing.T) {
	tunnel := NewTunnelIO(2, "session")
	if err := tunnel.Open(); err != nil {
		t.Fatalf("open tunnel: %v", err)
	}
	t.Cleanup(func() { _ = tunnel.Close() })

	for i := 0; i < tunnelRecvBufferSize; i++ {
		if err := tunnel.RecvData([]byte{byte(i)}); err != nil {
			t.Fatalf("fill receive buffer frame %d: %v", i, err)
		}
	}

	recvDone := make(chan error, 1)
	go func() {
		recvDone <- tunnel.RecvData([]byte("frame beyond receive window"))
	}()
	select {
	case err := <-recvDone:
		if !errors.Is(err, errTunnelReceiveQueueFull) {
			t.Fatalf("frame beyond bounded receive capacity = %v, want %v", err, errTunnelReceiveQueueFull)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("frame beyond bounded receive capacity blocked")
	}
}

func TestTunnelIOReceiveAdmissionBoundsBytesAndFrameSize(t *testing.T) {
	tunnel := NewTunnelIO(8, "session")
	if err := tunnel.Open(); err != nil {
		t.Fatalf("open tunnel: %v", err)
	}
	t.Cleanup(func() { _ = tunnel.Close() })

	// Use a small budget to exercise byte admission independently of the
	// production frame-count window.
	tunnel.recvByteLimit = 3
	if err := tunnel.RecvData([]byte("abc")); err != nil {
		t.Fatalf("fill receive byte budget: %v", err)
	}
	if err := tunnel.RecvData([]byte("d")); !errors.Is(err, errTunnelReceiveQueueFull) {
		t.Fatalf("frame beyond receive byte budget = %v, want %v", err, errTunnelReceiveQueueFull)
	}

	buffer := make([]byte, 3)
	if n, err := tunnel.Read(buffer); n != 3 || err != nil || string(buffer) != "abc" {
		t.Fatalf("read byte-budget frame = (%d, %q, %v)", n, buffer, err)
	}
	if err := tunnel.RecvData([]byte("d")); err != nil {
		t.Fatalf("admit after byte budget released: %v", err)
	}
	if err := tunnel.RecvData(make([]byte, sliverpb.MaxTunnelFrameBytes+1)); !errors.Is(err, errTunnelReceiveFrameTooLarge) {
		t.Fatalf("oversized receive frame = %v, want %v", err, errTunnelReceiveFrameTooLarge)
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

//nolint:gocyclo // Partial reads, terminal ordering, and EOF form one adapter contract.
func TestTunnelIOReadRetainsShortBufferRemainderAfterClose(t *testing.T) {
	tunnel := NewTunnelIO(7, "session")
	if err := tunnel.Open(); err != nil {
		t.Fatalf("open tunnel: %v", err)
	}
	if n, err := tunnel.Read(make([]byte, 0)); n != 0 || err != nil {
		t.Fatalf("zero-length read = (%d, %v), want (0, nil)", n, err)
	}
	if n, err := tunnel.Write(nil); n != 0 || err != nil {
		t.Fatalf("zero-length write = (%d, %v), want (0, nil)", n, err)
	}
	select {
	case data := <-tunnel.Send:
		t.Fatalf("zero-length write queued data %x", data)
	default:
	}

	if err := tunnel.RecvData([]byte("abcdef")); err != nil {
		t.Fatalf("queue short-buffer payload: %v", err)
	}
	first := make([]byte, 2)
	if n, err := tunnel.Read(first); n != 2 || err != nil || string(first) != "ab" {
		t.Fatalf("first short read = (%d, %q, %v), want (2, ab, nil)", n, first, err)
	}
	if err := tunnel.Close(); err != nil {
		t.Fatalf("close tunnel with remainder: %v", err)
	}
	second := make([]byte, 4)
	if n, err := tunnel.Read(second); n != 4 || err != nil || string(second) != "cdef" {
		t.Fatalf("remainder read = (%d, %q, %v), want (4, cdef, nil)", n, second, err)
	}
	if n, err := tunnel.Read(make([]byte, 1)); n != 0 || !errors.Is(err, io.EOF) {
		t.Fatalf("post-remainder read = (%d, %v), want (0, EOF)", n, err)
	}
}

func TestTunnelIOReadDrainsFrameAfterDoneCaseSelected(t *testing.T) {
	tunnel := NewTunnelIO(9, "session")
	if err := tunnel.Open(); err != nil {
		t.Fatalf("open tunnel: %v", err)
	}
	selectReady := make(chan struct{})
	doneSelected := make(chan struct{})
	releaseDoneBranch := make(chan struct{})
	var readyOnce sync.Once
	var doneOnce sync.Once
	tunnel.readSelectHook = func(selected bool) {
		if !selected {
			readyOnce.Do(func() { close(selectReady) })
			return
		}
		doneOnce.Do(func() { close(doneSelected) })
		<-releaseDoneBranch
	}

	type readResult struct {
		data []byte
		n    int
		err  error
	}
	readDone := make(chan readResult, 1)
	go func() {
		buffer := make([]byte, 32)
		n, err := tunnel.Read(buffer)
		readDone <- readResult{data: buffer[:n], n: n, err: err}
	}()
	waitForTestSignal(t, selectReady, "tunnel Read blocking select")
	if err := tunnel.Close(); err != nil {
		t.Fatalf("close tunnel: %v", err)
	}
	waitForTestSignal(t, doneSelected, "tunnel Read done case")

	payload := []byte("last admitted frame")
	if err := tunnel.queueRecvData(payload); err != nil {
		t.Fatalf("queue final admitted frame: %v", err)
	}
	close(releaseDoneBranch)
	select {
	case result := <-readDone:
		if result.n != len(payload) || result.err != nil || !bytes.Equal(result.data, payload) {
			t.Fatalf("Read after done case = (%d, %q, %v), want (%d, %q, nil)", result.n, result.data, result.err, len(payload), payload)
		}
	case <-time.After(time.Second):
		t.Fatal("Read did not drain the final admitted frame")
	}

	tunnel.readSelectHook = nil
	if n, err := tunnel.Read(make([]byte, 1)); n != 0 || !errors.Is(err, io.EOF) {
		t.Fatalf("Read after final drain = (%d, %v), want (0, EOF)", n, err)
	}
}
