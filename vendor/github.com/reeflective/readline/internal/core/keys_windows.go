//go:build windows
// +build windows

package core

import (
	"errors"
	"io"
	"unsafe"

	"github.com/reeflective/readline/inputrc"
)

// Windows-specific special key codes.
const (
	VK_CANCEL   = 0x03
	VK_BACK     = 0x08
	VK_TAB      = 0x09
	VK_RETURN   = 0x0D
	VK_SHIFT    = 0x10
	VK_CONTROL  = 0x11
	VK_MENU     = 0x12
	VK_ESCAPE   = 0x1B
	VK_LEFT     = 0x25
	VK_UP       = 0x26
	VK_RIGHT    = 0x27
	VK_DOWN     = 0x28
	VK_DELETE   = 0x2E
	VK_LSHIFT   = 0xA0
	VK_RSHIFT   = 0xA1
	VK_LCONTROL = 0xA2
	VK_RCONTROL = 0xA3
	VK_SNAPSHOT = 0x2C
	VK_INSERT   = 0x2D
	VK_HOME     = 0x24
	VK_END      = 0x23
	VK_PRIOR    = 0x21
	VK_NEXT     = 0x22
)

// Use an undefined Virtual Key sequence to pass
// Windows terminal resize events from the reader.
const (
	WINDOWS_RESIZE = 0x07
)

const (
	charTab       = 9
	charCtrlH     = 8
	charBackspace = 127
)

func init() {
	Stdin = newRawReader()
}

// GetTerminalResize sends booleans over a channel to notify resize events on Windows.
// This functions uses the keys reader because on Windows, resize events are sent through
// stdin, not with syscalls like unix's syscall.SIGWINCH.
func GetTerminalResize(keys *Keys) <-chan bool {
	keys.resize = make(chan bool, 1)

	return keys.resize
}

// readInputFiltered on Windows needs to check for terminal resize events.
func (k *Keys) readInputFiltered() (keys []byte, err error) {
	for {
		// Start reading from os.Stdin in the background.
		// We will either read keys from user, or an EOF
		// send by ourselves, because we pause reading.
		buf := make([]byte, keyScanBufSize)

		read, err := Stdin.Read(buf)
		if err != nil && errors.Is(err, io.EOF) {
			return keys, err
		}

		input := buf[:read]

		// On Windows, windows resize events are sent through stdin,
		// so if one is detected, send it back to the display engine.
		if len(input) == 1 && input[0] == WINDOWS_RESIZE {
			k.resize <- true
			continue
		}

		// Always attempt to extract cursor position info.
		// If found, strip it and keep the remaining keys.
		cursor, keys := k.extractCursorPos(input)

		if len(cursor) > 0 {
			k.cursor <- cursor
		}

		return keys, nil
	}
}

// rawReader translates Windows input to ANSI sequences,
// to provide the same behavior as Unix terminals.
type rawReader struct {
	ctrlKey  bool
	altKey   bool
	shiftKey bool
}

// newRawReader returns a new rawReader for Windows.
func newRawReader() *rawReader {
	r := new(rawReader)
	return r
}

// Read reads input record from stdin on Windows.
// It keeps reading until it gets a key event.
func (r *rawReader) Read(buf []byte) (int, error) {
	ir := new(_INPUT_RECORD)
	var read int
	var err error

next:
	// ReadConsoleInputW reads input record from stdin.
	err = kernel.ReadConsoleInputW(stdin,
		uintptr(unsafe.Pointer(ir)),
		1,
		uintptr(unsafe.Pointer(&read)),
	)
	if err != nil {
		return 0, err
	}

	// Keep resize events for the display engine to use.
	if ir.EventType == EVENT_WINDOW_BUFFER_SIZE {
		return r.write(buf, WINDOWS_RESIZE)
	}

	if ir.EventType != EVENT_KEY {
		goto next
	}

	// Reset modifiers if key is released.
	ker := (*_KEY_EVENT_RECORD)(unsafe.Pointer(&ir.Event[0]))
	if ker.bKeyDown == 0 { // keyup
		if r.ctrlKey || r.altKey || r.shiftKey {
			switch ker.wVirtualKeyCode {
			case VK_RCONTROL, VK_LCONTROL, VK_CONTROL:
				r.ctrlKey = false
			case VK_MENU: // alt
				r.altKey = false
			case VK_SHIFT, VK_LSHIFT, VK_RSHIFT:
				r.shiftKey = false
			}
		}
		goto next
	}

	// Keypad, special and arrow keys.
	if ker.unicodeChar == 0 {
		if modifiers, target := r.translateSeq(ker); target != 0 {
			return r.writeEsc(buf, append(modifiers, target)...)
		}
		goto next
	}

	char := rune(ker.unicodeChar)

	// Encode keys with modifiers.
	// Deal with the last (Windows) exceptions to the rule.
	switch {
	case r.shiftKey && char == charTab:
		return r.writeEsc(buf, 91, 90)
	case r.ctrlKey && char == charBackspace:
		char = charCtrlH
	case !r.ctrlKey && char == charCtrlH:
		char = charBackspace
	case r.ctrlKey:
		char = inputrc.Encontrol(char)
	case r.altKey:
		char = inputrc.Enmeta(char)
	}

	// Else, the key is a normal character.
	return r.write(buf, char)
}

// Close is a stub to satisfy io.Closer.
func (r *rawReader) Close() error {
	return nil
}

func (r *rawReader) writeEsc(b []byte, char ...rune) (int, error) {
	b[0] = byte(inputrc.Esc)
	n := copy(b[1:], []byte(string(char)))
	return n + 1, nil
}

func (r *rawReader) write(b []byte, char ...rune) (int, error) {
	n := copy(b, []byte(string(char)))
	return n, nil
}

func (r *rawReader) translateSeq(ker *_KEY_EVENT_RECORD) (modifiers []rune, target rune) {
	// Encode keys with modifiers by default,
	// unless the modifier is pressed alone.
	modifiers = append(modifiers, 91)

	// Modifiers add a default sequence, which is the good sequence for arrow keys by default.
	// The first rune is this sequence might be modified below, if the target is a special key
	// but not an arrow key.
	switch ker.wVirtualKeyCode {
	case VK_RCONTROL, VK_LCONTROL, VK_CONTROL:
		r.ctrlKey = true
	case VK_MENU: // alt
		r.altKey = true
	case VK_SHIFT, VK_LSHIFT, VK_RSHIFT:
		r.shiftKey = true
	}

	switch {
	case r.ctrlKey:
		modifiers = append(modifiers, 49, 59, 53)
	case r.altKey:
		modifiers = append(modifiers, 49, 59, 51)
	case r.shiftKey:
		modifiers = append(modifiers, 49, 59, 50)
	}

	changeModifiers := func(swap rune, pos int) {
		if len(modifiers) > pos-1 && pos > 0 {
			modifiers[pos] = swap
		} else {
			modifiers = append(modifiers, swap)
		}
	}

	// Now we handle the target key.
	switch ker.wVirtualKeyCode {
	// Keypad & arrow keys
	case VK_LEFT:
		target = 68
	case VK_RIGHT:
		target = 67
	case VK_UP:
		target = 65
	case VK_DOWN:
		target = 66
	case VK_HOME:
		target = 72
	case VK_END:
		target = 70

	// Other special keys, with effects on modifiers.
	case VK_SNAPSHOT:
	case VK_INSERT:
		changeModifiers(50, 2)
		target = 126
	case VK_DELETE:
		changeModifiers(51, 2)
		target = 126
	case VK_PRIOR:
		changeModifiers(53, 2)
		target = 126
	case VK_NEXT:
		changeModifiers(54, 2)
		target = 126
	}

	return
}
