//go:build solaris

package shell

import "golang.org/x/sys/unix"

func getTermios(fd int) (*unix.Termios, error) {
	return unix.IoctlGetTermios(fd, unix.TCGETA)
}

func setTermios(fd int, state *unix.Termios) error {
	return unix.IoctlSetTermios(fd, unix.TCSETA, state)
}
