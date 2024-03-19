//go:build !(linux || darwin || windows || freebsd || openbsd || netbsd || dragonfly || illumos) || sqlite3_nosys

package vfs

import "os"

// SupportsFileLocking is false on platforms that do not support file locking.
// To open a database file in one such platform,
// you need to use the [nolock] or [immutable] URI parameters.
//
// [nolock]: https://sqlite.org/uri.html#urinolock
// [immutable]: https://sqlite.org/uri.html#uriimmutable
const SupportsFileLocking = false

func osGetSharedLock(_ *os.File) _ErrorCode {
	return _IOERR_RDLOCK
}

func osGetReservedLock(_ *os.File) _ErrorCode {
	return _IOERR_LOCK
}

func osGetPendingLock(_ *os.File, _ bool) _ErrorCode {
	return _IOERR_LOCK
}

func osGetExclusiveLock(_ *os.File, _ bool) _ErrorCode {
	return _IOERR_LOCK
}

func osDowngradeLock(_ *os.File, _ LockLevel) _ErrorCode {
	return _IOERR_RDLOCK
}

func osReleaseLock(_ *os.File, _ LockLevel) _ErrorCode {
	return _IOERR_UNLOCK
}

func osCheckReservedLock(_ *os.File) (bool, _ErrorCode) {
	return false, _IOERR_CHECKRESERVEDLOCK
}
