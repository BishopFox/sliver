//go:build !windows
// +build !windows

package editor

import (
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

var (
	// ErrNoTempDirectory indicates that Go's standard os.TempDir() did not return a directory.
	ErrNoTempDirectory = errors.New("could not identify the temp directory on this system")
	// ErrWrite indicates that we failed to write the buffer to the file.
	ErrWrite = errors.New("failed to write buffer to file")
	// ErrCreate indicates that we failed to create the temp buffer file.
	ErrCreate = errors.New("failed to create buffer file")
	// ErrOpen indicates that we failed to open the buffer file.
	ErrOpen = errors.New("failed to open buffer file")
	// ErrRemove indicates that we failed to delete the buffer file.
	ErrRemove = errors.New("failed to remove buffer file")
	// ErrRead indicates that we failed to read the buffer file's content.
	ErrRead = errors.New("failed to read buffer file")
)

func writeToFile(buf []byte, filename string) (string, error) {
	var path string

	// Get the temp directory, or fail.
	tmp := os.TempDir()
	if tmp == "" {
		return "", ErrNoTempDirectory
	}

	// If the user has not provided any filename (including an extension)
	// we generate a random filename with no extension.
	if filename == "" {
		fileID := strconv.Itoa(time.Now().Nanosecond()) + ":" + string(buf)

		h := md5.New()

		_, err := h.Write([]byte(fileID))
		if err != nil {
			return "", err
		}

		name := "readline-" + hex.EncodeToString(h.Sum(nil)) + "-" + strconv.Itoa(os.Getpid())
		path = filepath.Join(tmp, name)
	} else {
		// Else, still use the temp/ dir, but with the provided filename
		path = filepath.Join(tmp, filename)
	}

	file, err := os.Create(path)
	if err != nil {
		return "", fmt.Errorf("%w: %s", ErrCreate, err.Error())
	}

	defer file.Close()

	_, err = file.Write(buf)
	if err != nil {
		return "", fmt.Errorf("%w: %s", ErrWrite, err.Error())
	}

	return path, nil
}

func readTempFile(name string) ([]byte, error) {
	file, err := os.Open(name)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrOpen, err.Error())
	}

	buf, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrRead, err.Error())
	}

	if len(buf) > 0 && buf[len(buf)-1] == '\n' {
		buf = buf[:len(buf)-1]
	}

	if len(buf) > 0 && buf[len(buf)-1] == '\r' {
		buf = buf[:len(buf)-1]
	}

	if len(buf) > 0 && buf[len(buf)-1] == '\n' {
		buf = buf[:len(buf)-1]
	}

	if len(buf) > 0 && buf[len(buf)-1] == '\r' {
		buf = buf[:len(buf)-1]
	}

	if err = os.Remove(name); err != nil {
		return nil, fmt.Errorf("%w: %s", ErrRemove, err.Error())
	}

	return buf, nil
}

func getSystemEditor(emacsDefault bool) (editor string) {
	editor = os.Getenv("VISUAL")
	if editor == "" {
		return
	}

	editor = os.Getenv("EDITOR")
	if editor == "" {
		return
	}

	if emacsDefault {
		return "emacs"
	}

	return "vi"
}
