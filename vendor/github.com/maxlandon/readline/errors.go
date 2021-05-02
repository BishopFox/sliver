package readline

import (
	"errors"
)

const (
	// ErrCtrlC is returned when ctrl+c is pressed.
	// WARNING: this is being deprecated! Please use CtrlC (type error) instead
	ErrCtrlC = "Ctrl+C"

	// ErrEOF is returned when ctrl+d is pressed.
	// WARNING: this is being deprecated! Please use EOF (type error) instead
	ErrEOF = "EOF"
)

var (
	// CtrlC is returned when ctrl+c is pressed
	CtrlC = errors.New("Ctrl+C")

	// EOF is returned when ctrl+d is pressed.
	// (this is actually the same value as io.EOF)
	EOF = errors.New("EOF")
)
