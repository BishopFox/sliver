//go:build aix || darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris

package shell

import (
	"errors"
	"io"
	"os"
	"sync"

	"golang.org/x/sys/unix"
)

const shellInputPollMilliseconds = 100

type cancellableShellInput struct {
	reader        io.Reader
	fd            uintptr
	originalFlags int
	flagsChanged  bool
	restoreOnce   sync.Once
	restoreErr    error
}

func newCancellableShellInput(file *os.File, reader io.Reader) (*cancellableShellInput, error) {
	flags, err := unix.FcntlInt(file.Fd(), unix.F_GETFL, 0)
	if err != nil {
		return nil, err
	}

	input := &cancellableShellInput{
		reader:        reader,
		fd:            file.Fd(),
		originalFlags: flags,
	}
	if flags&unix.O_NONBLOCK == 0 {
		if _, err := unix.FcntlInt(file.Fd(), unix.F_SETFL, flags|unix.O_NONBLOCK); err != nil {
			return nil, err
		}
		input.flagsChanged = true
	}
	return input, nil
}

func (s *cancellableShellInput) Read(data []byte, done <-chan struct{}) (int, bool, error) {
	pollFDs := []unix.PollFd{{Fd: int32(s.fd), Events: unix.POLLIN}}
	for {
		select {
		case <-done:
			return 0, true, nil
		default:
		}

		ready, err := unix.Poll(pollFDs, shellInputPollMilliseconds)
		if errors.Is(err, unix.EINTR) {
			continue
		}
		if err != nil {
			return 0, false, err
		}
		if ready == 0 {
			continue
		}

		n, err := s.reader.Read(data)
		if errors.Is(err, unix.EAGAIN) || errors.Is(err, unix.EWOULDBLOCK) {
			continue
		}
		return n, false, err
	}
}

func (s *cancellableShellInput) Close() error {
	s.restoreOnce.Do(func() {
		if s.flagsChanged {
			_, s.restoreErr = unix.FcntlInt(s.fd, unix.F_SETFL, s.originalFlags)
		}
	})
	return s.restoreErr
}
