//go:build unix

package dotlk

import (
	"errors"
	"io/fs"
	"os"
	"strconv"

	"golang.org/x/sys/unix"
)

// TryLock returns nil if it acquired the lock,
// fs.ErrExist if another process has the lock.
func TryLock(name string) error {
	for retry := true; retry; retry = false {
		f, err := os.OpenFile(name, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0666)
		if err == nil {
			f.WriteString(strconv.Itoa(os.Getpid()))
			f.Close()
			return nil
		}
		if !errors.Is(err, fs.ErrExist) {
			return err
		}
		if !removeStale(name) {
			break
		}
	}
	return fs.ErrExist
}

func removeStale(name string) bool {
	buf, err := os.ReadFile(name)
	if err != nil {
		return errors.Is(err, fs.ErrNotExist)
	}

	pid, err := strconv.Atoi(string(buf))
	if pid <= 0 || err != nil {
		return false
	}

	if pid == os.Getpid() {
		err = os.Remove(name)
		return err == nil || errors.Is(err, fs.ErrNotExist)
	}

	err = unix.Kill(pid, 0)
	if errors.Is(err, unix.ESRCH) {
		err = os.Remove(name)
		return err == nil || errors.Is(err, fs.ErrNotExist)
	}

	return false
}
