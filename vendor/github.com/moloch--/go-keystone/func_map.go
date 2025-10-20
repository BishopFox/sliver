package keystone

// I have tried to adjust the parameters for compiling,
// but I still can't solve the confusion of import and
// export function names, so I have to manually configure
// the mapping table of function names.
//
// Fortunately, the mapping relationship of these function
// names does not seem to change easily.

var importModule = "a"

// imported functions
var (
	___cxa_throw            = "a"
	___syscall_fstat64      = "g"
	___syscall_getcwd       = "o"
	___syscall_lstat64      = "d"
	___syscall_newfstatat   = "e"
	___syscall_openat       = "p"
	___syscall_stat64       = "f"
	__abort_js              = "s"
	__mmap_js               = "k"
	__munmap_js             = "l"
	_emscripten_resize_heap = "t"
	_environ_get            = "m"
	_environ_sizes_get      = "n"
	_exit                   = "h"
	_fd_close               = "c"
	_fd_fdstat_get          = "b"
	_fd_pread               = "i"
	_fd_read                = "r"
	_fd_seek                = "j"
	_fd_write               = "q"
)

// exported functions
var (
	_malloc      = "F"
	_free        = "G"
	_ks_open     = "A"
	_ks_option   = "C"
	_ks_asm      = "E"
	_ks_free     = "D"
	_ks_close    = "B"
	_ks_errno    = "x"
	_ks_strerror = "y"
	_ks_version  = "w"
)
