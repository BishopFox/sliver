//go:build (linux || darwin || windows || freebsd || openbsd || netbsd || dragonfly || illumos) && !sqlite3_nosys

package vfs

import "github.com/ncruces/go-sqlite3/internal/util"

// SupportsFileLocking is false on platforms that do not support file locking.
// To open a database file on those platforms,
// you need to use the [nolock] or [immutable] URI parameters.
//
// [nolock]: https://sqlite.org/uri.html#urinolock
// [immutable]: https://sqlite.org/uri.html#uriimmutable
const SupportsFileLocking = true

const (
	_PENDING_BYTE  = 0x40000000
	_RESERVED_BYTE = (_PENDING_BYTE + 1)
	_SHARED_FIRST  = (_PENDING_BYTE + 2)
	_SHARED_SIZE   = 510
)

func (f *vfsFile) Lock(lock LockLevel) error {
	// Argument check. SQLite never explicitly requests a pending lock.
	if lock != LOCK_SHARED && lock != LOCK_RESERVED && lock != LOCK_EXCLUSIVE {
		panic(util.AssertErr())
	}

	switch {
	case f.lock < LOCK_NONE || f.lock > LOCK_EXCLUSIVE:
		// Connection state check.
		panic(util.AssertErr())
	case f.lock == LOCK_NONE && lock > LOCK_SHARED:
		// We never move from unlocked to anything higher than a shared lock.
		panic(util.AssertErr())
	case f.lock != LOCK_SHARED && lock == LOCK_RESERVED:
		// A shared lock is always held when a reserved lock is requested.
		panic(util.AssertErr())
	}

	// If we already have an equal or more restrictive lock, do nothing.
	if f.lock >= lock {
		return nil
	}

	// Do not allow any kind of write-lock on a read-only database.
	if f.readOnly && lock >= LOCK_RESERVED {
		return _IOERR_LOCK
	}

	switch lock {
	case LOCK_SHARED:
		// Must be unlocked to get SHARED.
		if f.lock != LOCK_NONE {
			panic(util.AssertErr())
		}
		if rc := osGetSharedLock(f.File); rc != _OK {
			return rc
		}
		f.lock = LOCK_SHARED
		return nil

	case LOCK_RESERVED:
		// Must be SHARED to get RESERVED.
		if f.lock != LOCK_SHARED {
			panic(util.AssertErr())
		}
		if rc := osGetReservedLock(f.File); rc != _OK {
			return rc
		}
		f.lock = LOCK_RESERVED
		return nil

	case LOCK_EXCLUSIVE:
		// Must be SHARED, RESERVED or PENDING to get EXCLUSIVE.
		if f.lock <= LOCK_NONE || f.lock >= LOCK_EXCLUSIVE {
			panic(util.AssertErr())
		}
		reserved := f.lock == LOCK_RESERVED
		// A PENDING lock is needed before acquiring an EXCLUSIVE lock.
		if f.lock < LOCK_PENDING {
			// If we're already RESERVED, we can block indefinitely,
			// since only new readers may briefly hold the PENDING lock.
			if rc := osGetPendingLock(f.File, reserved /* block */); rc != _OK {
				return rc
			}
			f.lock = LOCK_PENDING
		}
		// We already have PENDING, so we're just waiting for readers to leave.
		// If we were RESERVED, we can wait for a little while, before invoking
		// the busy handler; we will only do this once.
		if rc := osGetExclusiveLock(f.File, reserved /* wait */); rc != _OK {
			return rc
		}
		f.lock = LOCK_EXCLUSIVE
		return nil

	default:
		panic(util.AssertErr())
	}
}

func (f *vfsFile) Unlock(lock LockLevel) error {
	// Argument check.
	if lock != LOCK_NONE && lock != LOCK_SHARED {
		panic(util.AssertErr())
	}

	// Connection state check.
	if f.lock < LOCK_NONE || f.lock > LOCK_EXCLUSIVE {
		panic(util.AssertErr())
	}

	// If we don't have a more restrictive lock, do nothing.
	if f.lock <= lock {
		return nil
	}

	switch lock {
	case LOCK_SHARED:
		rc := osDowngradeLock(f.File, f.lock)
		f.lock = LOCK_SHARED
		return rc

	case LOCK_NONE:
		rc := osReleaseLock(f.File, f.lock)
		f.lock = LOCK_NONE
		return rc

	default:
		panic(util.AssertErr())
	}
}

func (f *vfsFile) CheckReservedLock() (bool, error) {
	// Connection state check.
	if f.lock < LOCK_NONE || f.lock > LOCK_EXCLUSIVE {
		panic(util.AssertErr())
	}

	if f.lock >= LOCK_RESERVED {
		return true, nil
	}
	return osCheckReservedLock(f.File)
}
