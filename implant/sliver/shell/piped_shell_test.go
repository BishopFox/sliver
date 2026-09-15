//go:build darwin || linux || freebsd || openbsd || dragonfly

package shell

import (
	"io"
	"testing"
)

func TestPipedShellCombinesOutputAndWaitsOnce(t *testing.T) {
	systemShell, err := StartInteractive(
		1,
		[]string{"/bin/sh", "-c", "printf stdout; printf stderr >&2"},
		false,
		0,
		0,
	)
	if err != nil {
		t.Fatalf("start piped shell: %v", err)
	}
	if systemShell.Stderr != nil {
		t.Fatal("piped shell unexpectedly has a second stderr reader")
	}

	output, err := io.ReadAll(systemShell.Stdout)
	if err != nil {
		t.Fatalf("read combined output: %v", err)
	}
	if err := systemShell.Wait(); err != nil {
		t.Fatalf("wait for piped shell: %v", err)
	}
	if got, want := string(output), "stdoutstderr"; got != want {
		t.Fatalf("combined output = %q, want %q", got, want)
	}
}
