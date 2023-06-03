//go:build windows
// +build windows

package editor

import "errors"

// EditBuffer is currently not supported on Windows operating systems.
func (reg *Buffers) EditBuffer(buf []rune, filename, filetype string, emacs bool) ([]rune, error) {
	return buf, errors.New("Not currently supported on Windows")
}
