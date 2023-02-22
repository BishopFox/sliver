package platform

// Fdatasync is like syscall.Fdatasync except that's only defined in linux.
// This returns syscall.ENOSYS when unimplemented.
func Fdatasync(fd uintptr) (err error) {
	return fdatasync(fd)
}
