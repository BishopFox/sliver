// +build !windows

package readline

import (
	"crypto/md5"
	"encoding/hex"
	"io/ioutil"
	"os"
	"strconv"
	"time"
)

func (rl *Instance) writeTempFile(content []byte) (string, error) {
	fileID := strconv.Itoa(time.Now().Nanosecond()) + ":" + string(rl.line)

	h := md5.New()
	_, err := h.Write([]byte(fileID))
	if err != nil {
		return "", err
	}

	name := rl.TempDirectory + "readline-" + hex.EncodeToString(h.Sum(nil)) + "-" + strconv.Itoa(os.Getpid())

	file, err := os.Create(name)
	if err != nil {
		return "", err
	}

	defer file.Close()

	_, err = file.Write(content)
	return name, err
}

func readTempFile(name string) ([]byte, error) {
	file, err := os.Open(name)
	if err != nil {
		return nil, err
	}

	b, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, err
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
