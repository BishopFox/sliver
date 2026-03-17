package console

import (
	"bytes"
	"os"
	"testing"
	"time"
)

func TestDrainStdoutPipeBufferStripsSyncFrames(t *testing.T) {
	marker := []byte{0x01, 0x02, 0x03, 0x04}
	frame := buildStdoutSyncFrame(marker, 7)

	var got bytes.Buffer
	var acked []uint64
	pending := drainStdoutPipeBuffer(&got, append(append([]byte("before"), frame...), []byte("after")...), marker, func(seq uint64) {
		acked = append(acked, seq)
	})
	if len(pending) > 0 {
		_, _ = got.Write(pending)
	}

	if got.String() != "beforeafter" {
		t.Fatalf("unexpected output %q", got.String())
	}
	if len(acked) != 1 || acked[0] != 7 {
		t.Fatalf("unexpected sync acknowledgements: %v", acked)
	}
}

func TestDrainStdoutPipeBufferHandlesSplitFrame(t *testing.T) {
	marker := []byte{0xaa, 0xbb, 0xcc, 0xdd}
	frame := buildStdoutSyncFrame(marker, 11)

	var got bytes.Buffer
	var acked []uint64

	chunk1 := append([]byte("left"), frame[:5]...)
	pending := drainStdoutPipeBuffer(&got, chunk1, marker, func(seq uint64) {
		acked = append(acked, seq)
	})
	if got.String() != "left" {
		t.Fatalf("expected prefix output to flush, got %q", got.String())
	}

	chunk2 := append(append([]byte{}, pending...), frame[5:]...)
	chunk2 = append(chunk2, []byte("right")...)
	pending = drainStdoutPipeBuffer(&got, chunk2, marker, func(seq uint64) {
		acked = append(acked, seq)
	})
	if len(pending) > 0 {
		_, _ = got.Write(pending)
	}

	if got.String() != "leftright" {
		t.Fatalf("unexpected split-frame output %q", got.String())
	}
	if len(acked) != 1 || acked[0] != 11 {
		t.Fatalf("unexpected split-frame acknowledgements: %v", acked)
	}
}

func TestSyncOutputWaitsForPipeDrain(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}

	done := make(chan struct{})
	con := &SliverClient{
		stdoutPipeWriter: w,
		stdoutPipeDone:   done,
		stdoutSyncMarker: []byte("0123456789abcdef"),
		stdoutSyncAcks:   map[uint64]chan struct{}{},
	}

	var got bytes.Buffer
	go con.copyStdoutPipe(r, &got, done)

	if _, err := w.Write([]byte("hello")); err != nil {
		t.Fatalf("write prefix: %v", err)
	}
	con.syncOutput()
	if got.String() != "hello" {
		t.Fatalf("syncOutput returned before draining stdout: %q", got.String())
	}

	if _, err := w.Write([]byte(" world")); err != nil {
		t.Fatalf("write suffix: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("close writer: %v", err)
	}

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for stdout copier to finish")
	}

	if got.String() != "hello world" {
		t.Fatalf("unexpected final output %q", got.String())
	}
}
