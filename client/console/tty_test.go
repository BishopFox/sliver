package console

import (
	"strings"
	"testing"
)

func TestNonTTYGuardReturnsErrorWithoutRC(t *testing.T) {
	orig := isTerminal
	defer func() { isTerminal = orig }()
	isTerminal = func(_ int) bool { return false }

	startInteractive, err := nonTTYGuard("")
	if err == nil {
		t.Fatal("expected error when stdin is not a TTY and no rc script")
	}
	if startInteractive {
		t.Fatal("expected interactive console not to start without a TTY")
	}
	if !strings.Contains(err.Error(), "TTY") {
		t.Fatalf("expected error to mention TTY, got: %s", err)
	}
}

func TestNonTTYGuardReturnsNilWithRC(t *testing.T) {
	orig := isTerminal
	defer func() { isTerminal = orig }()
	isTerminal = func(_ int) bool { return false }

	startInteractive, err := nonTTYGuard("some-rc-content")
	if err != nil {
		t.Fatalf("expected nil when --rc is provided, got: %s", err)
	}
	if startInteractive {
		t.Fatal("expected interactive console not to start after running --rc without a TTY")
	}
}

func TestTTYGuardReturnsNil(t *testing.T) {
	orig := isTerminal
	defer func() { isTerminal = orig }()
	isTerminal = func(_ int) bool { return true }

	startInteractive, err := nonTTYGuard("")
	if err != nil {
		t.Fatalf("expected nil on a TTY, got: %s", err)
	}
	if !startInteractive {
		t.Fatal("expected interactive console to start on a TTY")
	}
}
