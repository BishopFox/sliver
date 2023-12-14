//go:build !(linux || darwin || windows || freebsd || openbsd || netbsd || dragonfly || illumos) || sqlite3_nosys

package vfs

import "os"

func osGetSharedLock(file *os.File) _ErrorCode {
	return _IOERR_RDLOCK
}

func osGetReservedLock(file *os.File) _ErrorCode {
	return _IOERR_LOCK
}

func osGetPendingLock(file *os.File) _ErrorCode {
	return _IOERR_LOCK
}

func osGetExclusiveLock(file *os.File) _ErrorCode {
	return _IOERR_LOCK
}

func osDowngradeLock(file *os.File, state LockLevel) _ErrorCode {
	return _IOERR_RDLOCK
}

func osReleaseLock(file *os.File, _ LockLevel) _ErrorCode {
	return _IOERR_UNLOCK
}

func osCheckReservedLock(file *os.File) (bool, _ErrorCode) {
	return false, _IOERR_CHECKRESERVEDLOCK
}
