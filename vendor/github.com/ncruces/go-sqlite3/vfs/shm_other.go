//go:build !(darwin || linux || illumos) || !(amd64 || arm64 || riscv64) || sqlite3_flock || sqlite3_noshm || sqlite3_nosys

package vfs

// SupportsSharedMemory is true on platforms that support shared memory.
// To enable shared memory support on those platforms,
// you need to set the appropriate [wazero.RuntimeConfig];
// otherwise, [EXCLUSIVE locking mode] is activated automatically
// to use [WAL without shared-memory].
//
// [WAL without shared-memory]: https://sqlite.org/wal.html#noshm
// [EXCLUSIVE locking mode]: https://sqlite.org/pragma.html#pragma_locking_mode
const SupportsSharedMemory = false

type vfsShm struct{}

func (vfsShm) Close() error { return nil }
