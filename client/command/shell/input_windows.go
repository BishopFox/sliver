//go:build windows

package shell

import (
	"errors"
	"fmt"
	"io"
	"os"
	"runtime"

	"golang.org/x/sys/windows"
)

const shellInputWaitMilliseconds = 100

var cancelSynchronousIO = windows.NewLazySystemDLL("kernel32.dll").NewProc("CancelSynchronousIo")

type cancellableShellInput struct {
	reader    io.Reader
	handle    windows.Handle
	isConsole bool
}

func newCancellableShellInput(file *os.File, reader io.Reader) (*cancellableShellInput, error) {
	handle := windows.Handle(file.Fd())
	var consoleMode uint32
	return &cancellableShellInput{
		reader:    reader,
		handle:    handle,
		isConsole: windows.GetConsoleMode(handle, &consoleMode) == nil,
	}, nil
}

func (s *cancellableShellInput) Read(data []byte, done <-chan struct{}) (int, error, bool) {
	for {
		select {
		case <-done:
			return 0, nil, true
		default:
		}

		event, err := windows.WaitForSingleObject(s.handle, shellInputWaitMilliseconds)
		if err != nil {
			return 0, err, false
		}
		switch event {
		case windows.WAIT_OBJECT_0:
			select {
			case <-done:
				return 0, nil, true
			default:
			}
			if s.isConsole {
				return s.readConsole(data, done)
			}
			return s.read(data)
		case uint32(windows.WAIT_TIMEOUT):
			continue
		default:
			return 0, fmt.Errorf("unexpected stdin wait result 0x%x", event), false
		}
	}
}

// A console input handle can be signaled by window, focus, mouse, or key-up
// events that do not make ReadFile return. Keep the synchronous read on a
// known OS thread so tunnel shutdown can cancel that exact read without
// leaving a background stdin reader behind to steal input from the REPL.
func (s *cancellableShellInput) readConsole(data []byte, done <-chan struct{}) (int, error, bool) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	thread, err := windows.OpenThread(windows.THREAD_TERMINATE, false, windows.GetCurrentThreadId())
	if err != nil {
		return 0, err, false
	}
	defer windows.CloseHandle(thread)

	readFinished := make(chan struct{})
	cancelFinished := make(chan error, 1)
	go func() {
		select {
		case <-done:
			cancelFinished <- cancelThreadSynchronousIO(thread)
		case <-readFinished:
			cancelFinished <- nil
		}
	}()

	n, readErr, _ := s.read(data)
	close(readFinished)
	cancelErr := <-cancelFinished
	if cancelErr != nil {
		return n, cancelErr, false
	}

	select {
	case <-done:
		if n == 0 && (readErr == nil || errors.Is(readErr, windows.ERROR_OPERATION_ABORTED)) {
			return 0, nil, true
		}
	default:
	}
	return n, readErr, false
}

func cancelThreadSynchronousIO(thread windows.Handle) error {
	result, _, err := cancelSynchronousIO.Call(uintptr(thread))
	if result != 0 || errors.Is(err, windows.ERROR_NOT_FOUND) {
		return nil
	}
	return err
}

func (s *cancellableShellInput) read(data []byte) (int, error, bool) {
	n, err := s.reader.Read(data)
	return n, err, false
}

func (s *cancellableShellInput) Close() error {
	return nil
}
