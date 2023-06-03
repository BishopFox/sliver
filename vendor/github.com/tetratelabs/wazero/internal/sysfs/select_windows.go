package sysfs

import (
	"context"
	"syscall"
	"time"
	"unsafe"

	"github.com/tetratelabs/wazero/internal/platform"
)

// wasiFdStdin is the constant value for stdin on Wasi.
// We need this constant because on Windows os.Stdin.Fd() != 0.
const wasiFdStdin = 0

// pollInterval is the interval between each calls to peekNamedPipe in pollNamedPipe
const pollInterval = 100 * time.Millisecond

var kernel32 = syscall.NewLazyDLL("kernel32.dll")

// procPeekNamedPipe is the syscall.LazyProc in kernel32 for PeekNamedPipe
var procPeekNamedPipe = kernel32.NewProc("PeekNamedPipe")

// syscall_select emulates the select syscall on Windows for two, well-known cases, returns syscall.ENOSYS for all others.
// If r contains fd 0, and it is a regular file, then it immediately returns 1 (data ready on stdin)
// and r will have the fd 0 bit set.
// If r contains fd 0, and it is a FILE_TYPE_CHAR, then it invokes PeekNamedPipe to check the buffer for input;
// if there is data ready, then it returns 1 and r will have fd 0 bit set.
// If n==0 it will wait for the given timeout duration, but it will return syscall.ENOSYS if timeout is nil,
// i.e. it won't block indefinitely.
//
// Note: idea taken from https://stackoverflow.com/questions/6839508/test-if-stdin-has-input-for-c-windows-and-or-linux
// PeekNamedPipe: https://learn.microsoft.com/en-us/windows/win32/api/namedpipeapi/nf-namedpipeapi-peeknamedpipe
// "GetFileType can assist in determining what device type the handle refers to. A console handle presents as FILE_TYPE_CHAR."
// https://learn.microsoft.com/en-us/windows/console/console-handles
func syscall_select(n int, r, w, e *platform.FdSet, timeout *time.Duration) (int, error) {
	if n == 0 {
		// Don't block indefinitely.
		if timeout == nil {
			return -1, syscall.ENOSYS
		}
		time.Sleep(*timeout)
		return 0, nil
	}
	if r.IsSet(wasiFdStdin) {
		fileType, err := syscall.GetFileType(syscall.Stdin)
		if err != nil {
			return 0, err
		}
		if fileType&syscall.FILE_TYPE_CHAR != 0 {
			res, err := pollNamedPipe(context.TODO(), syscall.Stdin, timeout)
			if err != nil {
				return -1, err
			}
			if !res {
				r.Zero()
				return 0, nil
			}
		}
		r.Zero()
		r.Set(wasiFdStdin)
		return 1, nil
	}
	return -1, syscall.ENOSYS
}

// pollNamedPipe polls the given named pipe handle for the given duration.
//
// The implementation actually polls every 100 milliseconds until it reaches the given duration.
// The duration may be nil, in which case it will wait undefinely. The given ctx is
// used to allow for cancellation. Currently used only in tests.
func pollNamedPipe(ctx context.Context, pipeHandle syscall.Handle, duration *time.Duration) (bool, error) {
	// Short circuit when the duration is zero.
	if duration != nil && *duration == time.Duration(0) {
		return peekNamedPipe(pipeHandle)
	}

	// Ticker that emits at every pollInterval.
	tick := time.NewTicker(pollInterval)
	tichCh := tick.C
	defer tick.Stop()

	// Timer that expires after the given duration.
	// Initialize afterCh as nil: the select below will wait forever.
	var afterCh <-chan time.Time
	if duration != nil {
		// If duration is not nil, instantiate the timer.
		after := time.NewTimer(*duration)
		defer after.Stop()
		afterCh = after.C
	}

	for {
		select {
		case <-ctx.Done():
			return false, nil
		case <-afterCh:
			return false, nil
		case <-tichCh:
			res, err := peekNamedPipe(pipeHandle)
			if err != nil {
				return false, err
			}
			if res {
				return true, nil
			}
		}
	}
}

// peekNamedPipe partially exposes PeekNamedPipe from the Win32 API
// see https://learn.microsoft.com/en-us/windows/win32/api/namedpipeapi/nf-namedpipeapi-peeknamedpipe
func peekNamedPipe(handle syscall.Handle) (bool, error) {
	var totalBytesAvail uint32
	totalBytesPtr := unsafe.Pointer(&totalBytesAvail)
	_, _, err := procPeekNamedPipe.Call(
		uintptr(handle),        // [in]            HANDLE  hNamedPipe,
		0,                      // [out, optional] LPVOID  lpBuffer,
		0,                      // [in]            DWORD   nBufferSize,
		0,                      // [out, optional] LPDWORD lpBytesRead
		uintptr(totalBytesPtr), // [out, optional] LPDWORD lpTotalBytesAvail,
		0)                      // [out, optional] LPDWORD lpBytesLeftThisMessage
	if err == syscall.Errno(0) {
		return totalBytesAvail > 0, nil
	}
	return totalBytesAvail > 0, err
}
