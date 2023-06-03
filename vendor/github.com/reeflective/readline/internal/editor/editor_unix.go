//go:build !windows && !plan9

package editor

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
)

// ErrStart indicates that the command to start the editor failed.
var ErrStart = errors.New("failed to start editor")

// EditBuffer starts the system editor and opens the given buffer in it.
// If the filename is specified, the file will be created in the system
// temp directory under this name.
// If the filetype is not empty and if the system editor supports it, the
// file will be opened with the specified filetype passed to the editor.
func (reg *Buffers) EditBuffer(buf []rune, filename, filetype string, emacs bool) ([]rune, error) {
	name, err := writeToFile([]byte(string(buf)), filename)
	if err != nil {
		return buf, err
	}

	editor := getSystemEditor(emacs)

	args := []string{}
	if filetype != "" {
		args = append(args, fmt.Sprintf("-c 'set filetype=%s", filetype))
	}

	args = append(args, name)

	cmd := exec.Command(editor, args...)

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err = cmd.Start(); err != nil {
		return buf, fmt.Errorf("%w: %s", ErrStart, err.Error())
	}

	if err = cmd.Wait(); err != nil {
		return buf, fmt.Errorf("%w: %s", ErrStart, err.Error())
	}

	b, err := readTempFile(name)

	return []rune(string(b)), err
}
