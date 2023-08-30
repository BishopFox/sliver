//go:build unix

package core

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
)

// GetCursorPos returns the current cursor position in the terminal.
// It is safe to call this function even if the shell is reading input.
func (k *Keys) GetCursorPos() (x, y int) {
	disable := func() (int, int) {
		os.Stderr.WriteString("\r\ngetCursorPos() not supported by terminal emulator, disabling....\r\n")
		return -1, -1
	}

	var cursor []byte
	var match [][]string

	// Echo the query and wait for the main key
	// reading routine to send us the response back.
	fmt.Print("\x1b[6n")

	// In order not to get stuck with an input that might be user-one
	// (like when the user typed before the shell is fully started, and yet not having
	// queried cursor yet), we keep reading from stdin until we find the cursor response.
	// Everything else is passed back as user input.
	for {
		switch {
		case k.waiting, k.reading:
			cursor = <-k.cursor
		default:
			buf := make([]byte, keyScanBufSize)

			read, err := os.Stdin.Read(buf)
			if err != nil {
				return disable()
			}

			cursor = buf[:read]
		}

		// We have read (or have been passed) something.
		if len(cursor) == 0 {
			return disable()
		}

		// Attempt to locate cursor response in it.
		match = rxRcvCursorPos.FindAllStringSubmatch(string(cursor), 1)

		// If there is something but not cursor answer, its user input.
		if len(match) == 0 && len(cursor) > 0 {
			k.mutex.RLock()
			k.buf = append(k.buf, cursor...)
			k.mutex.RUnlock()

			continue
		}

		// And if empty, then we should abort.
		if len(match) == 0 {
			return disable()
		}

		break
	}

	// We know that we have a cursor answer, process it.
	y, err := strconv.Atoi(match[0][1])
	if err != nil {
		return disable()
	}

	x, err = strconv.Atoi(match[0][2])
	if err != nil {
		return disable()
	}

	return x, y
}

func (k *Keys) readInputFiltered() (keys []byte, err error) {
	// Start reading from os.Stdin in the background.
	// We will either read keys from user, or an EOF
	// send by ourselves, because we pause reading.
	buf := make([]byte, keyScanBufSize)

	read, err := Stdin.Read(buf)
	if err != nil && errors.Is(err, io.EOF) {
		return
	}

	// Always attempt to extract cursor position info.
	// If found, strip it and keep the remaining keys.
	cursor, keys := k.extractCursorPos(buf[:read])

	if len(cursor) > 0 {
		k.cursor <- cursor
	}

	return keys, nil
}
