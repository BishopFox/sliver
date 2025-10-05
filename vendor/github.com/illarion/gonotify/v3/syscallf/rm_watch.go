//go:build linux

package syscallf

import "syscall"

func InotifyRmWatch(fd int, watchdesc int) (int, error) {
	var success int
	var err error

	r0, _, e1 := syscall.RawSyscall(syscall.SYS_INOTIFY_RM_WATCH, uintptr(fd), uintptr(watchdesc), 0)
	success = int(r0)
	if e1 != 0 {
		err = e1
	}
	return success, err
}
