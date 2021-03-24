// +build !windows

package readline

import (
	"crypto/md5"
	"encoding/hex"
	"io/ioutil"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

func (rl *Instance) launchEditor(multiline []rune) ([]rune, error) {
	name, err := rl.writeTempFile([]byte(string(multiline)))
	if err != nil {
		return multiline, err
	}

	editor := os.Getenv("EDITOR")
	// default editor is $EDITOR not set
	if editor == "" {
		editor = "vim"
	}

	cmd := exec.Command(editor, name)

	//cmd.SysProcAttr = &syscall.SysProcAttr{
	//	Ctty: int(os.Stdout.Fd()),
	//}

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return multiline, err
	}

	if err := cmd.Wait(); err != nil {
		return multiline, err
	}

	b, err := readTempFile(name, true)
	return []rune(string(b)), err
}

func (rl *Instance) writeTempFile(content []byte) (string, error) {
	fileID := strconv.Itoa(time.Now().Nanosecond()) + ":" + string(rl.line)

	h := md5.New()
	_, err := h.Write([]byte(fileID))
	if err != nil {
		return "", err
	}

	name := rl.TempDirectory + "/" + "readline-" + hex.EncodeToString(h.Sum(nil)) + "-" + strconv.Itoa(os.Getpid())

	file, err := os.Create(name)
	if err != nil {
		return "", err
	}

	defer file.Close()

	_, err = file.Write(content)
	if err != nil {

	}
	return name, err
}

// readTempFile - this function is only meant to be called by the shell.
// We don't use it, in Wiregost, for reading entire files that we then output
// to the console. Other wise we need to add an explicit return escape, so that
// it doesn't screw up the terminal.
func readTempFile(name string, forShell bool) ([]byte, error) {
	file, err := os.Open(name)
	if err != nil {
		return nil, err
	}

	b, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, err
	}

	// We parse all occurences of "\n" for not screwing the shell here
	if forShell {
		b = []byte(strings.Replace(string(b), "\n", " ", -1))
	}

	if len(b) > 0 && b[len(b)-1] == '\n' {
		b = b[:len(b)-1]
	}

	if len(b) > 0 && b[len(b)-1] == '\r' {
		b = b[:len(b)-1]
	}

	if len(b) > 0 && b[len(b)-1] == '\n' {
		b = b[:len(b)-1]
	}

	if len(b) > 0 && b[len(b)-1] == '\r' {
		b = b[:len(b)-1]
	}

	err = os.Remove(name)
	return b, err
}
