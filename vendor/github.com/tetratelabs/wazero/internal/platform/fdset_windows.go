package platform

import (
	"syscall"
	"unsafe"
)

var procGetNamedPipeInfo = kernel32.NewProc("GetNamedPipeInfo")

// Maximum number of fds in a WinSockFdSet.
const _FD_SETSIZE = 64

// WinSockFdSet implements the FdSet representation that is used internally by WinSock.
//
// Note: this representation is quite different from the one used in most POSIX implementations
// where a bitfield is usually implemented; instead on Windows we have a simpler array+count pair.
// Notice that because it keeps a count of the inserted handles, the first argument of select
// in WinSock is actually ignored.
//
// The implementation of the Set, Clear, IsSet, Zero, methods follows exactly
// the real implementation found in WinSock2.h, e.g. see:
// https://github.com/microsoft/win32metadata/blob/ef7725c75c6b39adfdc13ba26fb1d89ac954449a/generation/WinSDK/RecompiledIdlHeaders/um/WinSock2.h#L124-L175
type WinSockFdSet struct {
	// count is the number of used slots used in the handles slice.
	count uint64
	// handles is the array of handles. This is called "array" in the WinSock implementation
	// and it has a fixed length of _FD_SETSIZE.
	handles [_FD_SETSIZE]syscall.Handle
}

// FdSet implements the same methods provided on other plaforms.
//
// Note: the implementation is very different from POSIX; Windows provides
// POSIX select only for sockets. We emulate a select for other APIs in the sysfs
// package, but we still want to use the "real" select in the case of sockets.
// So, we keep separate FdSets for sockets, pipes and regular files, so that we can
// handle them separately. For instance sockets can be used directly in winsock select.
type FdSet struct {
	sockets WinSockFdSet
	pipes   WinSockFdSet
	regular WinSockFdSet
}

// SetAll overwrites all the fields in f with the fields in g.
func (f *FdSet) SetAll(g *FdSet) {
	if f == nil {
		return
	}
	f.sockets = g.sockets
	f.pipes = g.pipes
	f.regular = g.regular
}

// Sockets returns a WinSockFdSet containing the handles in this FdSet that are sockets.
func (f *FdSet) Sockets() *WinSockFdSet {
	if f == nil {
		return nil
	}
	return &f.sockets
}

// Regular returns a WinSockFdSet containing the handles in this FdSet that are regular files.
func (f *FdSet) Regular() *WinSockFdSet {
	if f == nil {
		return nil
	}
	return &f.regular
}

// Pipes returns a WinSockFdSet containing the handles in this FdSet that are pipes.
func (f *FdSet) Pipes() *WinSockFdSet {
	if f == nil {
		return nil
	}
	return &f.pipes
}

// getFdSetFor returns a pointer to the right fd set for the given fd.
// It checks the type for fd and returns the field for pipes, regular or sockets
// to simplify code.
//
// For instance, if fd is a socket and it must be set if f.pipes, instead
// of writing:
//
//	 if isSocket(fd) {
//			f.sockets.Set(fd)
//		}
//
// It is possible to write:
//
//	f.getFdSetFor(fd).Set(fd)
func (f *FdSet) getFdSetFor(fd int) *WinSockFdSet {
	h := syscall.Handle(fd)
	t, err := syscall.GetFileType(h)
	if err != nil {
		return nil
	}
	switch t {
	case syscall.FILE_TYPE_CHAR, syscall.FILE_TYPE_DISK:
		return &f.regular
	case syscall.FILE_TYPE_PIPE:
		if isSocket(h) {
			return &f.sockets
		} else {
			return &f.pipes
		}
	default:
		return nil
	}
}

// Set adds the given fd to the set.
func (f *FdSet) Set(fd int) {
	if s := f.getFdSetFor(fd); s != nil {
		s.Set(fd)
	}
}

// Clear removes the given fd from the set.
func (f *FdSet) Clear(fd int) {
	if s := f.getFdSetFor(fd); s != nil {
		s.Clear(fd)
	}
}

// IsSet returns true when fd is in the set.
func (f *FdSet) IsSet(fd int) bool {
	if s := f.getFdSetFor(fd); s != nil {
		return s.IsSet(fd)
	}
	return false
}

// Copy returns a copy of this FdSet. It returns nil, if the FdSet is nil.
func (f *FdSet) Copy() *FdSet {
	if f == nil {
		return nil
	}
	return &FdSet{
		sockets: f.sockets,
		pipes:   f.pipes,
		regular: f.regular,
	}
}

// Zero clears the set. It returns 0 if the FdSet is nil.
func (f *FdSet) Count() int {
	if f == nil {
		return 0
	}
	return f.sockets.Count() + f.regular.Count() + f.pipes.Count()
}

// Zero clears the set.
func (f *FdSet) Zero() {
	if f == nil {
		return
	}
	f.sockets.Zero()
	f.regular.Zero()
	f.pipes.Zero()
}

// Set adds the given fd to the set.
func (f *WinSockFdSet) Set(fd int) {
	if f.count < _FD_SETSIZE {
		f.handles[f.count] = syscall.Handle(fd)
		f.count++
	}
}

// Clear removes the given fd from the set.
func (f *WinSockFdSet) Clear(fd int) {
	h := syscall.Handle(fd)
	for i := uint64(0); i < f.count; i++ {
		if f.handles[i] == h {
			for ; i < f.count-1; i++ {
				f.handles[i] = f.handles[i+1]
			}
			f.count--
			break
		}
	}
}

// IsSet returns true when fd is in the set.
func (f *WinSockFdSet) IsSet(fd int) bool {
	h := syscall.Handle(fd)
	for i := uint64(0); i < f.count; i++ {
		if f.handles[i] == h {
			return true
		}
	}
	return false
}

// Zero clears the set.
func (f *WinSockFdSet) Zero() {
	if f == nil {
		return
	}
	f.handles = [64]syscall.Handle{}
	f.count = 0
}

// Count returns the number of values that are set in the fd set.
func (f *WinSockFdSet) Count() int {
	if f == nil {
		return 0
	}
	return int(f.count)
}

// Copy returns a copy of the fd set or nil if it is nil.
func (f *WinSockFdSet) Copy() *WinSockFdSet {
	if f == nil {
		return nil
	}
	copy := *f
	return &copy
}

// Get returns the handle at the given index.
func (f *WinSockFdSet) Get(index int) syscall.Handle {
	return f.handles[index]
}

// isSocket returns true if the given file handle
// is a pipe.
func isSocket(fd syscall.Handle) bool {
	r, _, errno := syscall.SyscallN(
		procGetNamedPipeInfo.Addr(),
		uintptr(fd),
		uintptr(unsafe.Pointer(nil)),
		uintptr(unsafe.Pointer(nil)),
		uintptr(unsafe.Pointer(nil)),
		uintptr(unsafe.Pointer(nil)))
	return r == 0 || errno != 0
}
