// +build !windows,!plan9

package readline

import (
	"os"
	"os/exec"
)

const defaultEditor = "vi"

func (rl *Instance) launchEditor(multiline []rune) ([]rune, error) {
	name, err := rl.writeTempFile([]byte(string(multiline)))
	if err != nil {
		return multiline, err
	}

	editor := os.Getenv("EDITOR")
	// default editor if $EDITOR not set
	if editor == "" {
		editor = defaultEditor
	}

	cmd := exec.Command(editor, name)

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return multiline, err
	}

	if err := cmd.Wait(); err != nil {
		return multiline, err
	}

	b, err := readTempFile(name)
	return []rune(string(b)), err
}
