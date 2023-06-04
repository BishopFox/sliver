package sysfs

import (
	"time"

	"github.com/tetratelabs/wazero/internal/platform"
)

// _select exposes the select(2) syscall. This is named as such to avoid
// colliding with they keyword select while not exporting the function.
//
// # Notes on Parameters
//
//	For convenience, we expose a pointer to a time.Duration instead of a pointer to a syscall.Timeval.
//	It must be a pointer because `nil` means "wait forever".
//
//	However, notice that select(2) may mutate the pointed Timeval on some platforms,
//	for instance if the call returns early.
//
//	This implementation *will not* update the pointed time.Duration value accordingly.
//
//	See also: https://github.com/golang/sys/blob/master/unix/syscall_unix_test.go#L606-L617
//
// # Notes on the Syscall
//
//	Because this is a blocking syscall, it will also block the carrier thread of the goroutine,
//	preventing any means to support context cancellation directly.
//
//	There are ways to obviate this issue. We outline here one idea, that is however not currently implemented.
//	A common approach to support context cancellation is to add a signal file descriptor to the set,
//	e.g. the read-end of a pipe or an eventfd on Linux.
//	When the context is canceled, we may unblock a Select call by writing to the fd, causing it to return immediately.
//	This however requires to do a bit of housekeeping to hide the "special" FD from the end-user.
func _select(n int, r, w, e *platform.FdSet, timeout *time.Duration) (int, error) {
	return syscall_select(n, r, w, e, timeout)
}
