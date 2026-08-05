package shell

import (
	"os"

	"golang.org/x/term"
)

func configureShellTerminal(enablePTY bool) (func() error, error) {
	if !enablePTY {
		return configureNoPTYTerminal(int(os.Stdin.Fd()), shellEscapeByte)
	}

	const stdinFD = 0
	oldState, err := term.MakeRaw(stdinFD)
	if err != nil {
		return nil, err
	}
	return func() error {
		return term.Restore(stdinFD, oldState)
	}, nil
}

func replaceControlChar(controlChars []uint8, index int, value uint8) (uint8, bool) {
	if index < 0 || index >= len(controlChars) {
		return 0, false
	}
	original := controlChars[index]
	controlChars[index] = value
	return original, true
}
