//go:build aix || darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris || windows

package shell

import (
	"os"
	"testing"
	"time"
)

func TestCancellableShellInputWakesOnTunnelClose(t *testing.T) {
	readFile, writeFile, err := os.Pipe()
	if err != nil {
		t.Fatalf("create stdin pipe: %v", err)
	}
	defer readFile.Close()
	defer writeFile.Close()

	input, err := newCancellableShellInput(readFile, newFilterReader(readFile))
	if err != nil {
		t.Fatalf("prepare cancellable stdin: %v", err)
	}
	defer input.Close()

	done := make(chan struct{})
	type readResult struct {
		n            int
		err          error
		tunnelClosed bool
	}
	readDone := make(chan readResult, 1)
	go func() {
		n, err, tunnelClosed := input.Read(make([]byte, 16), done)
		readDone <- readResult{n: n, err: err, tunnelClosed: tunnelClosed}
	}()

	select {
	case result := <-readDone:
		t.Fatalf("stdin Read returned before data or tunnel close: %+v", result)
	case <-time.After(50 * time.Millisecond):
	}

	close(done)
	select {
	case result := <-readDone:
		if result.n != 0 || result.err != nil || !result.tunnelClosed {
			t.Fatalf("stdin Read after tunnel close = %+v, want n=0 err=nil tunnelClosed=true", result)
		}
	case <-time.After(time.Second):
		t.Fatal("stdin Read did not wake when the tunnel closed")
	}
}
