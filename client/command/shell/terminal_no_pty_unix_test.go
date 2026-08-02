//go:build darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris

package shell

import (
	"os"
	"testing"
)

func TestConfigureNoPTYTerminalIgnoresNonTerminal(t *testing.T) {
	stdin, stdout, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer stdin.Close()
	defer stdout.Close()

	restore, err := configureNoPTYTerminal(int(stdin.Fd()), shellEscapeByte)
	if err != nil {
		t.Fatalf("configureNoPTYTerminal() error = %v", err)
	}
	if err := restore(); err != nil {
		t.Fatalf("restore() error = %v", err)
	}
}
