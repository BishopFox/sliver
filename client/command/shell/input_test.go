//go:build aix || darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris || windows

package shell

import (
	"errors"
	"io"
	"os"
	"testing"
	"time"
)

func closeShellInputTestFile(t *testing.T, file *os.File) {
	t.Helper()
	if err := file.Close(); err != nil {
		t.Errorf("close %s: %v", file.Name(), err)
	}
}

func TestCancellableShellInputWakesOnTunnelClose(t *testing.T) {
	readFile, writeFile, err := os.Pipe()
	if err != nil {
		t.Fatalf("create stdin pipe: %v", err)
	}
	t.Cleanup(func() { closeShellInputTestFile(t, readFile) })
	t.Cleanup(func() { closeShellInputTestFile(t, writeFile) })

	input, err := newCancellableShellInput(readFile, newFilterReader(readFile))
	if err != nil {
		t.Fatalf("prepare cancellable stdin: %v", err)
	}
	t.Cleanup(func() {
		if err := input.Close(); err != nil {
			t.Errorf("close cancellable stdin: %v", err)
		}
	})

	done := make(chan struct{})
	type readResult struct {
		n            int
		err          error
		tunnelClosed bool
	}
	readDone := make(chan readResult, 1)
	go func() {
		n, tunnelClosed, err := input.Read(make([]byte, 16), done)
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

func TestCancellableShellInputReportsPipeEOF(t *testing.T) {
	readFile, writeFile, err := os.Pipe()
	if err != nil {
		t.Fatalf("create stdin pipe: %v", err)
	}
	t.Cleanup(func() { closeShellInputTestFile(t, readFile) })
	if err := writeFile.Close(); err != nil {
		t.Fatalf("close stdin pipe writer: %v", err)
	}

	input, err := newCancellableShellInput(readFile, newFilterReader(readFile))
	if err != nil {
		t.Fatalf("prepare cancellable stdin: %v", err)
	}
	t.Cleanup(func() {
		if err := input.Close(); err != nil {
			t.Errorf("close cancellable stdin: %v", err)
		}
	})

	n, tunnelClosed, err := input.Read(make([]byte, 16), make(chan struct{}))
	if n != 0 || tunnelClosed || !errors.Is(err, io.EOF) {
		t.Fatalf("stdin Read after pipe EOF = n=%d tunnelClosed=%v err=%v, want n=0 tunnelClosed=false err=EOF", n, tunnelClosed, err)
	}
}

func TestCancellableShellInputReadsRegularFile(t *testing.T) {
	file, err := os.CreateTemp(t.TempDir(), "shell-stdin-*")
	if err != nil {
		t.Fatalf("create stdin file: %v", err)
	}
	t.Cleanup(func() { closeShellInputTestFile(t, file) })

	const inputData = "whoami\n"
	if _, err := file.WriteString(inputData); err != nil {
		t.Fatalf("write stdin file: %v", err)
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		t.Fatalf("rewind stdin file: %v", err)
	}

	input, err := newCancellableShellInput(file, newFilterReader(file))
	if err != nil {
		t.Fatalf("prepare cancellable stdin: %v", err)
	}
	t.Cleanup(func() {
		if err := input.Close(); err != nil {
			t.Errorf("close cancellable stdin: %v", err)
		}
	})

	data := make([]byte, len(inputData))
	n, tunnelClosed, err := input.Read(data, make(chan struct{}))
	if err != nil {
		t.Fatalf("read stdin file: %v", err)
	}
	if tunnelClosed {
		t.Fatal("stdin file read reported a closed tunnel")
	}
	if got := string(data[:n]); got != inputData {
		t.Fatalf("stdin file data = %q, want %q", got, inputData)
	}
}
