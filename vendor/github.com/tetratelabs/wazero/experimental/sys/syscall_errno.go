package sys

import "syscall"

func syscallToErrno(errno syscall.Errno) Errno {
	switch errno {
	case 0:
		return 0
	case syscall.EACCES:
		return EACCES
	case syscall.EAGAIN:
		return EAGAIN
	case syscall.EBADF:
		return EBADF
	case syscall.EEXIST:
		return EEXIST
	case syscall.EFAULT:
		return EFAULT
	case syscall.EINTR:
		return EINTR
	case syscall.EINVAL:
		return EINVAL
	case syscall.EIO:
		return EIO
	case syscall.EISDIR:
		return EISDIR
	case syscall.ELOOP:
		return ELOOP
	case syscall.ENAMETOOLONG:
		return ENAMETOOLONG
	case syscall.ENOENT:
		return ENOENT
	case syscall.ENOSYS:
		return ENOSYS
	case syscall.ENOTDIR:
		return ENOTDIR
	case syscall.ENOTEMPTY:
		return ENOTEMPTY
	case syscall.ENOTSOCK:
		return ENOTSOCK
	case syscall.ENOTSUP:
		return ENOTSUP
	case syscall.EPERM:
		return EPERM
	case syscall.EROFS:
		return EROFS
	default:
		return EIO
	}
}

// Unwrap is a convenience for runtime.GOOS which define syscall.Errno.
func (e Errno) Unwrap() error {
	switch e {
	case 0:
		return nil
	case EACCES:
		return syscall.EACCES
	case EAGAIN:
		return syscall.EAGAIN
	case EBADF:
		return syscall.EBADF
	case EEXIST:
		return syscall.EEXIST
	case EFAULT:
		return syscall.EFAULT
	case EINTR:
		return syscall.EINTR
	case EINVAL:
		return syscall.EINVAL
	case EIO:
		return syscall.EIO
	case EISDIR:
		return syscall.EISDIR
	case ELOOP:
		return syscall.ELOOP
	case ENAMETOOLONG:
		return syscall.ENAMETOOLONG
	case ENOENT:
		return syscall.ENOENT
	case ENOSYS:
		return syscall.ENOSYS
	case ENOTDIR:
		return syscall.ENOTDIR
	case ENOTEMPTY:
		return syscall.ENOTEMPTY
	case ENOTSOCK:
		return syscall.ENOTSOCK
	case ENOTSUP:
		return syscall.ENOTSUP
	case EPERM:
		return syscall.EPERM
	case EROFS:
		return syscall.EROFS
	default:
		return syscall.EIO
	}
}
