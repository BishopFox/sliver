// Code taken from github.com/chzyer/readline.

//go:build windows
// +build windows

package core

import (
	"reflect"
	"syscall"
	"unsafe"
)

var (
	kernel = NewKernel()
	stdout = uintptr(syscall.Stdout)
	stdin  = uintptr(syscall.Stdin)
)

// Kernel stores the Windows kernel32.dll functions required for terminal control.
type Kernel struct {
	SetConsoleCursorPosition,
	SetConsoleTextAttribute,
	FillConsoleOutputCharacterW,
	FillConsoleOutputAttribute,
	ReadConsoleInputW,
	GetConsoleScreenBufferInfo,
	GetConsoleCursorInfo,
	GetStdHandle CallFunc
}

type (
	short int16
	word  uint16
	dword uint32
	wchar uint16
)

type _COORD struct {
	x short
	y short
}

func (c *_COORD) ptr() uintptr {
	return uintptr(*(*int32)(unsafe.Pointer(c)))
}

const (
	EVENT_KEY                = 0x0001 // Event for key press/release
	EVENT_MOUSE              = 0x0002 // Event for mouse action
	EVENT_WINDOW_BUFFER_SIZE = 0x0004 // Event for window resize
	EVENT_MENU               = 0x0008 // Event for the menu keys
	EVENT_FOCUS              = 0x0010 // Event for focus change
)

type _KEY_EVENT_RECORD struct {
	bKeyDown          int32
	wRepeatCount      word
	wVirtualKeyCode   word
	wVirtualScanCode  word
	unicodeChar       wchar
	dwControlKeyState dword
}

// KEY_EVENT_RECORD          KeyEvent;
// MOUSE_EVENT_RECORD        MouseEvent;
// WINDOW_BUFFER_SIZE_RECORD WindowBufferSizeEvent;
// MENU_EVENT_RECORD         MenuEvent;
// FOCUS_EVENT_RECORD        FocusEvent;
type _INPUT_RECORD struct {
	EventType word
	Padding   uint16
	Event     [16]byte
}

type _CONSOLE_SCREEN_BUFFER_INFO struct {
	dwSize              _COORD
	dwCursorPosition    _COORD
	wAttributes         word
	srWindow            _SMALL_RECT
	dwMaximumWindowSize _COORD
}

type _SMALL_RECT struct {
	left   short
	top    short
	right  short
	bottom short
}

type _CONSOLE_CURSOR_INFO struct {
	dwSize   dword
	bVisible bool
}

// CallFunc is a function that calls a Windows API function.
type CallFunc func(u ...uintptr) error

// NewKernel returns a new Kernel with all the required Windows API functions.
func NewKernel() *Kernel {
	k := &Kernel{}
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	v := reflect.ValueOf(k).Elem()
	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		name := t.Field(i).Name
		f := kernel32.NewProc(name)
		v.Field(i).Set(reflect.ValueOf(k.Wrap(f)))
	}
	return k
}

// Wrap wraps a Windows API function into a callback function.
func (k *Kernel) Wrap(p *syscall.LazyProc) CallFunc {
	return func(args ...uintptr) error {
		var r0 uintptr
		var e1 syscall.Errno
		size := uintptr(len(args))
		if len(args) <= 3 {
			buf := make([]uintptr, 3)
			copy(buf, args)
			r0, _, e1 = syscall.Syscall(p.Addr(), size,
				buf[0], buf[1], buf[2])
		} else {
			buf := make([]uintptr, 6)
			copy(buf, args)
			r0, _, e1 = syscall.Syscall6(p.Addr(), size,
				buf[0], buf[1], buf[2], buf[3], buf[4], buf[5],
			)
		}

		if int(r0) == 0 {
			if e1 != 0 {
				return error(e1)
			}
			return syscall.EINVAL
		}
		return nil
	}
}

// getConsoleScreenBufferInfo returns the current screen buffer information on Windows.
func getConsoleScreenBufferInfo() (*_CONSOLE_SCREEN_BUFFER_INFO, error) {
	t := new(_CONSOLE_SCREEN_BUFFER_INFO)
	err := kernel.GetConsoleScreenBufferInfo(
		stdout,
		uintptr(unsafe.Pointer(t)),
	)
	return t, err
}

// getConsoleCursorInfo returns the current cursor information on Windows.
func getConsoleCursorInfo() (*_CONSOLE_CURSOR_INFO, error) {
	t := new(_CONSOLE_CURSOR_INFO)
	err := kernel.GetConsoleCursorInfo(stdout, uintptr(unsafe.Pointer(t)))
	return t, err
}

// setConsoleCursorInfo sets the cursor position on Windows.
func setConsoleCursorPosition(c *_COORD) error {
	return kernel.SetConsoleCursorPosition(stdout, c.ptr())
}

// GetCursorPos returns the current cursor position on Windows.
func (k *Keys) GetCursorPos() (x, y int) {
	t := new(_CONSOLE_SCREEN_BUFFER_INFO)
	kernel.GetConsoleScreenBufferInfo(
		stdout,
		uintptr(unsafe.Pointer(t)),
	)

	x = int(t.dwCursorPosition.x) + 1
	y = int(t.dwCursorPosition.y)

	return
}
