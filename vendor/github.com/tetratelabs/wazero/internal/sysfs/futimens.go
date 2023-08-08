package sysfs

import (
	"syscall"
	"time"
	"unsafe"

	experimentalsys "github.com/tetratelabs/wazero/experimental/sys"
	"github.com/tetratelabs/wazero/sys"
)

const (
	// UTIME_NOW is a special syscall.Timespec NSec value used to set the
	// file's timestamp to a value close to, but not greater than the current
	// system time.
	UTIME_NOW = _UTIME_NOW

	// UTIME_OMIT is a special syscall.Timespec NSec value used to avoid
	// setting the file's timestamp.
	UTIME_OMIT = _UTIME_OMIT
)

// Utimens set file access and modification times on a path resolved to the
// current working directory, at nanosecond precision.
//
// # Parameters
//
// The `times` parameter includes the access and modification timestamps to
// assign. Special syscall.Timespec NSec values UTIME_NOW and UTIME_OMIT may be
// specified instead of real timestamps. A nil `times` parameter behaves the
// same as if both were set to UTIME_NOW. If the path is a symbolic link, the
// target of expanding that link is updated.
//
// # Errors
//
// A zero sys.Errno is success. The below are expected otherwise:
//   - sys.ENOSYS: the implementation does not support this function.
//   - sys.EINVAL: `path` is invalid.
//   - sys.EEXIST: `path` exists and is a directory.
//   - sys.ENOTDIR: `path` exists and is a file.
//
// # Notes
//
//   - This is like syscall.UtimesNano and `utimensat` with `AT_FDCWD` in
//     POSIX. See https://pubs.opengroup.org/onlinepubs/9699919799/functions/futimens.html
func Utimens(path string, times *[2]syscall.Timespec) experimentalsys.Errno {
	err := utimens(path, times)
	return experimentalsys.UnwrapOSError(err)
}

func timesToPtr(times *[2]syscall.Timespec) unsafe.Pointer { //nolint:unused
	if times != nil {
		return unsafe.Pointer(&times[0])
	}
	return unsafe.Pointer(nil)
}

func utimensPortable(path string, times *[2]syscall.Timespec) error { //nolint:unused
	// Handle when both inputs are current system time.
	if times == nil || times[0].Nsec == UTIME_NOW && times[1].Nsec == UTIME_NOW {
		ts := nowTimespec()
		return syscall.UtimesNano(path, []syscall.Timespec{ts, ts})
	}

	// When both inputs are omitted, there is nothing to change.
	if times[0].Nsec == UTIME_OMIT && times[1].Nsec == UTIME_OMIT {
		return nil
	}

	// Handle when neither input are special values
	if times[0].Nsec != UTIME_NOW && times[1].Nsec != UTIME_NOW &&
		times[0].Nsec != UTIME_OMIT && times[1].Nsec != UTIME_OMIT {
		return syscall.UtimesNano(path, times[:])
	}

	// Now, either atim or mtim is a special value, but not both.

	// Now, either one of the inputs is a special value, or neither. This means
	// we don't have a risk of re-reading the clock or re-doing stat.
	if atim, err := normalizeTimespec(path, times, 0); err != 0 {
		return err
	} else if mtim, err := normalizeTimespec(path, times, 1); err != 0 {
		return err
	} else {
		return syscall.UtimesNano(path, []syscall.Timespec{atim, mtim})
	}
}

func normalizeTimespec(path string, times *[2]syscall.Timespec, i int) (ts syscall.Timespec, err experimentalsys.Errno) { //nolint:unused
	switch times[i].Nsec {
	case UTIME_NOW: // declined in Go per golang/go#31880.
		ts = nowTimespec()
		return
	case UTIME_OMIT:
		// UTIME_OMIT is expensive until progress is made in Go, as it requires a
		// stat to read-back the value to re-apply.
		// - https://github.com/golang/go/issues/32558.
		// - https://go-review.googlesource.com/c/go/+/219638 (unmerged)
		var st sys.Stat_t
		if st, err = stat(path); err != 0 {
			return
		}
		switch i {
		case 0:
			ts = syscall.NsecToTimespec(st.Atim)
		case 1:
			ts = syscall.NsecToTimespec(st.Mtim)
		default:
			panic("BUG")
		}
		return
	default: // not special
		ts = times[i]
		return
	}
}

func nowTimespec() syscall.Timespec { //nolint:unused
	now := time.Now().UnixNano()
	return syscall.NsecToTimespec(now)
}
