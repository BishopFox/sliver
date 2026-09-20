package console

import (
	"strings"
	"testing"
)

func TestNonTTYGuardReturnsErrorWithoutRC(t *testing.T) {
	orig := isTerminal
	defer func() { isTerminal = orig }()
	isTerminal = func(fd int) bool { return false }

	err := nonTTYGuard("")
	if err == nil {
		t.Fatal("expected error when stdin is not a TTY and no rc script")
	}
	if !strings.Contains(err.Error(), "TTY") {
		t.Fatalf("expected error to mention TTY, got: %s", err)
	}
}

func TestNonTTYGuardReturnsNilWithRC(t *testing.T) {
	orig := isTerminal
	defer func() { isTerminal = orig }()
	isTerminal = func(fd int) bool { return false }

	err := nonTTYGuard("some-rc-content")
	if err != nil {
		t.Fatalf("expected nil when --rc is provided, got: %s", err)
	}
}

func TestTTYGuardReturnsNil(t *testing.T) {
	orig := isTerminal
	defer func() { isTerminal = orig }()
	isTerminal = func(fd int) bool { return true }

	err := nonTTYGuard("")
	if err != nil {
		t.Fatalf("expected nil on a TTY, got: %s", err)
	}
}
