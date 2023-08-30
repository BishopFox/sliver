package sysfs

import (
	"context"
	"syscall"
	"time"
	"unsafe"

	"github.com/tetratelabs/wazero/experimental/sys"
	"github.com/tetratelabs/wazero/internal/platform"
)

// pollInterval is the interval between each calls to peekNamedPipe in selectAllHandles
const pollInterval = 100 * time.Millisecond

// zeroDuration is the zero value for time.Duration. It is used in selectAllHandles.
var zeroDuration = time.Duration(0)

// syscall_select emulates the select syscall on Windows, for a subset of cases.
//
// r, w, e may contain any number of file handles, but regular files and pipes are only processed for r (Read).
// Stdin is a pipe, thus it is checked for readiness when present. Pipes are checked using PeekNamedPipe.
// Regular files always immediately report as ready, regardless their actual state and timeouts.
//
// If n==0 it will wait for the given timeout duration, but it will return sys.ENOSYS if timeout is nil,
// i.e. it won't block indefinitely.
//
// Note: ideas taken from https://stackoverflow.com/questions/6839508/test-if-stdin-has-input-for-c-windows-and-or-linux
// PeekNamedPipe: https://learn.microsoft.com/en-us/windows/win32/api/namedpipeapi/nf-namedpipeapi-peeknamedpipe
func syscall_select(n int, r, w, e *platform.FdSet, timeout *time.Duration) (int, error) {
	if n == 0 {
		// Don't block indefinitely.
		if timeout == nil {
			return -1, sys.ENOSYS
		}
		time.Sleep(*timeout)
		return 0, nil
	}

	n, errno := selectAllHandles(context.TODO(), r, w, e, timeout)
	if errno == 0 {
		return n, nil
	}
	return n, errno
}

// selectAllHandles emulates a general-purpose POSIX select on Windows.
//
// The implementation actually polls every 100 milliseconds until it reaches the given duration.
// The duration may be nil, in which case it will wait undefinely. The given ctx is
// used to allow for cancellation, and it is currently used only in tests.
//
// As indicated in the man page for select [1], r, w, e are modified upon completion:
//
// "Upon successful completion, the pselect() or select() function shall modify the objects pointed to by the readfds,
// writefds, and errorfds arguments to indicate which file descriptors are ready for reading, ready for writing,
// or have an error condition pending, respectively, and shall return the total number of ready descriptors in all the output sets"
//
// However, for our purposes, this may be pedantic because currently we do not check the values of r, w, e
// after the invocation of select; thus, this behavior may be subject to change in the future for the sake of simplicity.
//
// [1]: https://linux.die.net/man/3/select
func selectAllHandles(ctx context.Context, r, w, e *platform.FdSet, duration *time.Duration) (n int, errno sys.Errno) {
	r2, w2, e2 := r.Copy(), w.Copy(), e.Copy()
	n, errno = peekAllHandles(r2, w2, e2)
	// Short circuit when there is an error, there is data or the duration is zero.
	if errno != 0 || n > 0 || (duration != nil && *duration == time.Duration(0)) {
		r.SetAll(r2)
		w.SetAll(w2)
		e.SetAll(e2)
		return
	}

	// Ticker that emits at every pollInterval.
	tick := time.NewTicker(pollInterval)
	tickCh := tick.C
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
			r.Zero()
			w.Zero()
			e.Zero()
			return
		case <-afterCh:
			r.Zero()
			w.Zero()
			e.Zero()
			return
		case <-tickCh:
			r2, w2, e2 = r.Copy(), w.Copy(), e.Copy()
			n, errno = peekAllHandles(r2, w2, e2)
			if errno != 0 || n > 0 {
				r.SetAll(r2)
				w.SetAll(w2)
				e.SetAll(e2)
				return
			}
		}
	}
}

func peekAllHandles(r, w, e *platform.FdSet) (int, sys.Errno) {
	// pipes are not checked on w, e
	w.Pipes().Zero()
	e.Pipes().Zero()

	// peek pipes only for reading
	errno := peekAllPipes(r.Pipes())
	if errno != 0 {
		return 0, errno
	}

	_, errno = winsock_select(r.Sockets(), w.Sockets(), e.Sockets(), &zeroDuration)
	if errno != 0 {
		return 0, errno
	}

	return r.Count() + w.Count() + e.Count(), 0
}

func peekAllPipes(pipeHandles *platform.WinSockFdSet) sys.Errno {
	ready := &platform.WinSockFdSet{}
	for i := 0; i < pipeHandles.Count(); i++ {
		h := pipeHandles.Get(i)
		bytes, errno := peekNamedPipe(h)
		if bytes > 0 {
			ready.Set(int(h))
		}
		if errno != 0 {
			return sys.UnwrapOSError(errno)
		}
	}
	*pipeHandles = *ready
	return 0
}

func winsock_select(r, w, e *platform.WinSockFdSet, timeout *time.Duration) (int, sys.Errno) {
	if r.Count() == 0 && w.Count() == 0 && e.Count() == 0 {
		return 0, 0
	}

	var t *syscall.Timeval
	if timeout != nil {
		tv := syscall.NsecToTimeval(timeout.Nanoseconds())
		t = &tv
	}

	rp := unsafe.Pointer(r)
	wp := unsafe.Pointer(w)
	ep := unsafe.Pointer(e)
	tp := unsafe.Pointer(t)

	r0, _, err := syscall.SyscallN(
		procselect.Addr(),
		uintptr(0), // the first argument is ignored and exists only for compat with BSD sockets.
		uintptr(rp),
		uintptr(wp),
		uintptr(ep),
		uintptr(tp))
	return int(r0), sys.UnwrapOSError(err)
}
