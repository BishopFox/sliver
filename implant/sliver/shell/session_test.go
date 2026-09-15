package shell

import (
	"io"
	"sync"
	"testing"
)

func TestStopSessionIsIdempotent(t *testing.T) {
	const tunnelID = 42
	stdin := &countingWriteCloser{}
	var cancelMutex sync.Mutex
	cancelCalls := 0
	session := NewSession(&Shell{
		Stdin: stdin,
		Cancel: func() {
			cancelMutex.Lock()
			cancelCalls++
			cancelMutex.Unlock()
		},
	})
	if !RegisterSession(tunnelID, session) {
		t.Fatal("registered session was rejected")
	}
	t.Cleanup(func() { UnregisterSession(tunnelID) })

	var wait sync.WaitGroup
	for range 32 {
		wait.Add(1)
		go func() {
			defer wait.Done()
			if !StopSession(tunnelID) {
				t.Errorf("registered session was not found")
			}
		}()
	}
	wait.Wait()

	if got := stdin.CloseCalls(); got != 1 {
		t.Fatalf("stdin close calls = %d, want 1", got)
	}
	cancelMutex.Lock()
	gotCancelCalls := cancelCalls
	cancelMutex.Unlock()
	if gotCancelCalls != 1 {
		t.Fatalf("cancel calls = %d, want 1", gotCancelCalls)
	}

	UnregisterSession(tunnelID)
	if StopSession(tunnelID) {
		t.Fatal("unregistered session was still found")
	}
}

func TestCloseBeforeRegisterStopsSession(t *testing.T) {
	const tunnelID = 43
	UnregisterSession(tunnelID)
	t.Cleanup(func() { UnregisterSession(tunnelID) })

	// Tunnel handlers run concurrently, so force the close handler's lookup to
	// happen before the shell handler publishes its newly started process.
	if StopSession(tunnelID) {
		t.Fatal("unregistered session unexpectedly reported as active")
	}

	stdin := &countingWriteCloser{}
	var cancelMutex sync.Mutex
	cancelCalls := 0
	session := NewSession(&Shell{
		Stdin: stdin,
		Cancel: func() {
			cancelMutex.Lock()
			cancelCalls++
			cancelMutex.Unlock()
		},
	})
	if RegisterSession(tunnelID, session) {
		t.Fatal("session registered after its tunnel had already closed")
	}
	if got := stdin.CloseCalls(); got != 1 {
		t.Fatalf("stdin close calls = %d, want 1", got)
	}
	cancelMutex.Lock()
	gotCancelCalls := cancelCalls
	cancelMutex.Unlock()
	if gotCancelCalls != 1 {
		t.Fatalf("cancel calls = %d, want 1", gotCancelCalls)
	}
}

type countingWriteCloser struct {
	mutex sync.Mutex
	calls int
}

func (*countingWriteCloser) Write(data []byte) (int, error) { return len(data), nil }

func (c *countingWriteCloser) Close() error {
	c.mutex.Lock()
	c.calls++
	c.mutex.Unlock()
	return nil
}

func (c *countingWriteCloser) CloseCalls() int {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	return c.calls
}

var _ io.WriteCloser = (*countingWriteCloser)(nil)
