package fsapi

import "syscall"

// Windows does not have these constants, we declare placeholders which should
// not conflict with other open flags. These placeholders are not declared as
// value zero so code written in a way which expects them to be bit flags still
// works as expected.
//
// Since those placeholder are not interpreted by the open function, the unix
// features they represent are also not implemented on windows:
//
//   - O_DIRECTORY allows programs to ensure that the opened file is a directory.
//     This could be emulated by doing a stat call on the file after opening it
//     to verify that it is in fact a directory, then closing it and returning an
//     error if it is not.
//
//   - O_NOFOLLOW allows programs to ensure that if the opened file is a symbolic
//     link, the link itself is opened instead of its target.
const (
	O_DIRECTORY = 1 << 29
	O_NOFOLLOW  = 1 << 30
	O_NONBLOCK  = syscall.O_NONBLOCK
)
