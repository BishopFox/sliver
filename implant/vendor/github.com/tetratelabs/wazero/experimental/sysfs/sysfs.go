// Package sysfs includes a low-level filesystem interface and utilities needed
// for WebAssembly host functions (ABI) such as WASI.
//
// The name sysfs was chosen because wazero's public API has a "sys" package,
// which was named after https://github.com/golang/sys.
//
// This tracked in https://github.com/tetratelabs/wazero/issues/1013
package sysfs

import (
	experimentalsys "github.com/tetratelabs/wazero/experimental/sys"
	"github.com/tetratelabs/wazero/internal/sysfs"
)

// AdaptFS adapts the input to sys.FS. Use DirFS instead of adapting an
// os.DirFS as it handles interop issues such as windows support.
//
// Note: This performs no flag verification on OpenFile. sys.FS cannot read
// flags as there is no parameter to pass them through with. Moreover, sys.FS
// documentation does not require the file to be present. In summary, we can't
// enforce flag behavior.
type AdaptFS = sysfs.AdaptFS

// DirFS is like os.DirFS except it returns sys.FS, which has more features.
func DirFS(dir string) experimentalsys.FS {
	return sysfs.DirFS(dir)
}

// ReadFS is used to mask an existing sys.FS for reads. Notably, this allows
// the CLI to do read-only mounts of directories the host user can write, but
// doesn't want the guest wasm to. For example, Python libraries shouldn't be
// written to at runtime by the python wasm file.
//
// Note: This implements read-only by returning sys.EROFS or sys.EBADF,
// depending on the operation that require write access.
type ReadFS = sysfs.ReadFS
