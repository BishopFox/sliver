//go:build aix || darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris

package shell

import (
	"os"
	"testing"
	"time"

	"golang.org/x/sys/unix"
)

func TestCancellableShellInputRestoresFileFlags(t *testing.T) {
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("create input pipe: %v", err)
	}
	t.Cleanup(func() { closeShellInputTestFile(t, reader) })
	t.Cleanup(func() { closeShellInputTestFile(t, writer) })

	originalFlags, err := unix.FcntlInt(reader.Fd(), unix.F_GETFL, 0)
	if err != nil {
		t.Fatalf("read original input flags: %v", err)
	}
	input, err := newCancellableShellInput(reader, reader)
	if err != nil {
		t.Fatalf("prepare cancellable input: %v", err)
	}

	type readResult struct {
		err    error
		closed bool
	}
	done := make(chan struct{})
	result := make(chan readResult, 1)
	go func() {
		_, closed, err := input.Read(make([]byte, 32), done)
		result <- readResult{err: err, closed: closed}
	}()

	select {
	case got := <-result:
		t.Fatalf("input read returned before data or close: %+v", got)
	case <-time.After(50 * time.Millisecond):
	}
	close(done)
	select {
	case got := <-result:
		if got.err != nil || !got.closed {
			t.Fatalf("input read after close = %+v, want closed without error", got)
		}
	case <-time.After(time.Second):
		t.Fatal("input read did not wake after tunnel close")
	}

	if err := input.Close(); err != nil {
		t.Fatalf("restore input flags: %v", err)
	}
	restoredFlags, err := unix.FcntlInt(reader.Fd(), unix.F_GETFL, 0)
	if err != nil {
		t.Fatalf("read restored input flags: %v", err)
	}
	if restoredFlags != originalFlags {
		t.Fatalf("restored input flags = %#x, want %#x", restoredFlags, originalFlags)
	}
}
