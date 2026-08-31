//go:build !aix && !darwin && !dragonfly && !freebsd && !linux && !netbsd && !openbsd && !solaris && !windows

package shell

import (
	"io"
	"os"
)

// This fallback preserves the legacy blocking behavior on unsupported client
// platforms. Unix and Windows clients use cancellable implementations.
type cancellableShellInput struct {
	reader io.Reader
}

func newCancellableShellInput(_ *os.File, reader io.Reader) (*cancellableShellInput, error) {
	return &cancellableShellInput{reader: reader}, nil
}

func (s *cancellableShellInput) Read(data []byte, done <-chan struct{}) (int, bool, error) {
	select {
	case <-done:
		return 0, true, nil
	default:
	}
	n, err := s.reader.Read(data)
	return n, false, err
}

func (s *cancellableShellInput) Close() error {
	return nil
}
