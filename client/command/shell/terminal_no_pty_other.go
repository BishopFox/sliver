//go:build !darwin && !dragonfly && !freebsd && !linux && !netbsd && !openbsd && !solaris

package shell

func configureNoPTYTerminal(_ int, _ byte) (func() error, error) {
	return func() error { return nil }, nil
}
