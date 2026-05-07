package pty

import (
	"os"
	"path/filepath"
	"syscall"
	"unsafe"
)

func open() (pty, tty *os.File, err error) {
	/*
	 * from ptm(4):
	 * The PTMGET command allocates a free pseudo terminal, changes its
	 * ownership to the caller, revokes the access privileges for all previous
	 * users, opens the file descriptors for the pty and tty devices and
	 * returns them to the caller in struct ptmget.
	 */

	p, err := os.OpenFile("/dev/ptm", os.O_RDWR|syscall.O_CLOEXEC, 0)
	if err != nil {
		return nil, nil, err
	}
	defer p.Close()

	var ptm ptmget
	if err := ioctl(p.Fd(), uintptr(ioctl_PTMGET), uintptr(unsafe.Pointer(&ptm))); err != nil {
		return nil, nil, err
	}

	cName := cString(ptm.Cn[:])
	sName := cString(ptm.Sn[:])
	ptyPath := "/dev/ptm"
	ttyPath := "/dev/ptm"
	if cName != "" {
		ptyPath = filepath.Join("/dev", cName)
	}
	if sName != "" {
		ttyPath = filepath.Join("/dev", sName)
	}
	pty = os.NewFile(uintptr(ptm.Cfd), ptyPath)
	tty = os.NewFile(uintptr(ptm.Sfd), ttyPath)

	return pty, tty, nil
}

func cString(in []int8) string {
	out := make([]byte, 0, len(in))
	for _, c := range in {
		if c == 0 {
			break
		}
		out = append(out, byte(c))
	}
	return string(out)
}
