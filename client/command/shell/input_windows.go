//go:build windows

package shell

import (
	"errors"
	"fmt"
	"io"
	"os"
	"runtime"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

const shellInputWaitMilliseconds = 100

var (
	cancelSynchronousIO = windows.NewLazySystemDLL("kernel32.dll").NewProc("CancelSynchronousIo")
	peekNamedPipe       = windows.NewLazySystemDLL("kernel32.dll").NewProc("PeekNamedPipe")
)

type cancellableShellInput struct {
	reader    io.Reader
	handle    windows.Handle
	isConsole bool
	isPipe    bool
}

func newCancellableShellInput(file *os.File, reader io.Reader) (*cancellableShellInput, error) {
	handle := windows.Handle(file.Fd())
	fileType, err := windows.GetFileType(handle)
	if err != nil {
		return nil, err
	}
	var consoleMode uint32
	return &cancellableShellInput{
		reader:    reader,
		handle:    handle,
		isConsole: windows.GetConsoleMode(handle, &consoleMode) == nil,
		isPipe:    fileType == windows.FILE_TYPE_PIPE,
	}, nil
}

func (s *cancellableShellInput) Read(data []byte, done <-chan struct{}) (int, bool, error) {
	select {
	case <-done:
		return 0, true, nil
	default:
	}

	if s.isPipe {
		return s.readPipe(data, done)
	}
	if !s.isConsole {
		n, err := s.reader.Read(data)
		return n, false, err
	}

	for {
		select {
		case <-done:
			return 0, true, nil
		default:
		}

		event, err := windows.WaitForSingleObject(s.handle, shellInputWaitMilliseconds)
		if err != nil {
			return 0, false, err
		}
		switch event {
		case windows.WAIT_OBJECT_0:
			select {
			case <-done:
				return 0, true, nil
			default:
			}
			return s.readConsole(data, done)
		case uint32(windows.WAIT_TIMEOUT):
			continue
		default:
			return 0, false, fmt.Errorf("unexpected stdin wait result 0x%x", event)
		}
	}
}

// WaitForSingleObject does not reliably indicate that a pipe read will
// complete. PeekNamedPipe lets redirected stdin remain cancellable without
// starting an overlapped os.File.Read that CancelSynchronousIo cannot stop.
func (s *cancellableShellInput) readPipe(data []byte, done <-chan struct{}) (int, bool, error) {
	ticker := time.NewTicker(shellInputWaitMilliseconds * time.Millisecond)
	defer ticker.Stop()

	for {
		available, err := pipeBytesAvailable(s.handle)
		if errors.Is(err, windows.ERROR_BROKEN_PIPE) ||
			errors.Is(err, windows.ERROR_NO_DATA) ||
			errors.Is(err, windows.ERROR_PIPE_NOT_CONNECTED) {
			return 0, false, io.EOF
		}
		if err != nil {
			return 0, false, err
		}
		if available > 0 {
			n, readErr := s.reader.Read(data)
			return n, false, readErr
		}

		select {
		case <-done:
			return 0, true, nil
		case <-ticker.C:
		}
	}
}

func pipeBytesAvailable(handle windows.Handle) (uint32, error) {
	var available uint32
	result, _, err := peekNamedPipe.Call(
		uintptr(handle),
		0,
		0,
		0,
		uintptr(unsafe.Pointer(&available)),
		0,
	)
	if result == 0 {
		return 0, err
	}
	return available, nil
}

// A console input handle can be signaled by window, focus, mouse, or key-up
// events that do not make ReadFile return. Keep the synchronous read on a
// known OS thread so tunnel shutdown can cancel that exact read without
// leaving a background stdin reader behind to steal input from the REPL.
func (s *cancellableShellInput) readConsole(data []byte, done <-chan struct{}) (int, bool, error) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	thread, err := windows.OpenThread(windows.THREAD_TERMINATE, false, windows.GetCurrentThreadId())
	if err != nil {
		return 0, false, err
	}
	defer func() {
		_ = windows.CloseHandle(thread)
	}()

	readFinished := make(chan struct{})
	cancelFinished := make(chan error, 1)
	go func() {
		select {
		case <-done:
			cancelFinished <- cancelThreadSynchronousIO(thread, readFinished)
		case <-readFinished:
			cancelFinished <- nil
		}
	}()

	n, readErr := s.reader.Read(data)
	close(readFinished)
	cancelErr := <-cancelFinished
	if cancelErr != nil {
		return n, false, cancelErr
	}

	select {
	case <-done:
		if n == 0 && (readErr == nil || errors.Is(readErr, windows.ERROR_OPERATION_ABORTED)) {
			return 0, true, nil
		}
	default:
	}
	return n, false, readErr
}

func cancelThreadSynchronousIO(thread windows.Handle, readFinished <-chan struct{}) error {
	ticker := time.NewTicker(time.Millisecond)
	defer ticker.Stop()

	for {
		result, _, err := cancelSynchronousIO.Call(uintptr(thread))
		if result != 0 {
			return nil
		}
		if !errors.Is(err, windows.ERROR_NOT_FOUND) {
			return err
		}

		// The close signal can race just ahead of the synchronous read. Retry
		// ERROR_NOT_FOUND until the read starts or finishes so no console reader
		// is left behind to steal input from the REPL.
		select {
		case <-readFinished:
			return nil
		case <-ticker.C:
		}
	}
}

func (s *cancellableShellInput) Close() error {
	return nil
}
