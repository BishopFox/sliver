//go:build plan9
// +build plan9

package editor

import "errors"

// EditBuffer is currently not supported on Plan9 operating systems.
func (reg *Buffers) EditBuffer(buf []rune, filename, filetype string, emacs bool) ([]rune, error) {
	return buf, errors.New("Not currently supported on Plan 9")
}
