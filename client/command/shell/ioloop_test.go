package shell

import (
	"bytes"
	"os"
	"testing"
	"time"

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/client/core"
)

func TestIsShellExitCommand(t *testing.T) {
	tests := []struct {
		name    string
		line    string
		windows bool
		want    bool
	}{
		{name: "exit", line: "exit", want: true},
		{name: "unix uppercase is not exit", line: " EXIT ", want: false},
		{name: "windows case insensitive exit", line: " EXIT ", windows: true, want: true},
		{name: "exit status", line: "exit 42", want: true},
		{name: "cmd exit arguments", line: "Exit /B 1", windows: true, want: true},
		{name: "logout", line: "logout", want: true},
		{name: "windows logout arguments", line: "LoGoUt --force", windows: true, want: true},
		{name: "empty", line: "  ", want: false},
		{name: "exit prefix", line: "exiting", want: false},
		{name: "logout prefix", line: "logout-now", want: false},
		{name: "exit later in line", line: "echo exit", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isShellExitCommand(tt.line, tt.windows); got != tt.want {
				t.Fatalf("isShellExitCommand(%q, %v) = %v, want %v", tt.line, tt.windows, got, tt.want)
			}
		})
	}
}

func TestRunAttachedIODeliversExitBeforeReturning(t *testing.T) {
	readFile, writeFile, err := os.Pipe()
	if err != nil {
		t.Fatalf("create stdin pipe: %v", err)
	}
	defer readFile.Close()
	defer writeFile.Close()

	originalStdin := os.Stdin
	os.Stdin = readFile
	defer func() { os.Stdin = originalStdin }()

	tunnel := core.NewTunnelIO(91, "session")
	if err := tunnel.Open(); err != nil {
		t.Fatalf("open tunnel: %v", err)
	}
	defer tunnel.Close()

	con := &console.SliverClient{}
	received := make(chan []byte, 1)
	go func() {
		select {
		case data := <-tunnel.Send:
			received <- append([]byte(nil), data...)
		case <-tunnel.Done():
		}
	}()

	type ioResult struct {
		detached bool
		closed   bool
	}
	result := make(chan ioResult, 1)
	go func() {
		detached, closed := runAttachedIO(tunnel, con, "linux")
		result <- ioResult{detached: detached, closed: closed}
	}()
	if _, err := writeFile.Write([]byte("exit\n")); err != nil {
		t.Fatalf("write test stdin: %v", err)
	}

	select {
	case data := <-received:
		if !bytes.Equal(data, []byte("exit\n")) {
			t.Fatalf("delivered input = %q, want exit command", data)
		}
	case <-time.After(time.Second):
		t.Fatal("exit command was not delivered before attached I/O returned")
	}
	select {
	case got := <-result:
		if got.detached || !got.closed {
			t.Fatalf("runAttachedIO result = %+v, want close requested", got)
		}
	case <-time.After(time.Second):
		t.Fatal("runAttachedIO did not return after delivering exit")
	}
}
