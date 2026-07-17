package handlers

/*
	Sliver Implant Framework
	Copyright (C) 2026  Bishop Fox

	This program is free software: you can redistribute it and/or modify
	it under the terms of the GNU General Public License as published by
	the Free Software Foundation, either version 3 of the License, or
	(at your option) any later version.

	This program is distributed in the hope that it will be useful,
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	GNU General Public License for more details.

	You should have received a copy of the GNU General Public License
	along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"os/exec"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/0x90pkt/trigger/pkg/intents"
)

func TestNewReverseShellRejectsBadInputs(t *testing.T) {
	cases := []struct {
		name string
		hand string
		cfg  ReverseShellConfig
		want string
	}{
		{"empty task name", "", ReverseShellConfig{OperatorAddr: "10.0.0.1:4444"}, "task name"},
		{"empty operator addr", "x", ReverseShellConfig{}, "OperatorAddr"},
		{"malformed addr", "x", ReverseShellConfig{OperatorAddr: "not-an-addr"}, "invalid OperatorAddr"},
		{"missing port", "x", ReverseShellConfig{OperatorAddr: "10.0.0.1"}, "invalid OperatorAddr"},
		{"relative shell path", "x", ReverseShellConfig{OperatorAddr: "10.0.0.1:4444", ShellPath: "sh"}, "must be absolute"},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			_, err := NewReverseShell(tc.hand, tc.cfg)
			if err == nil {
				t.Fatalf("expected NewReverseShell to reject %s", tc.name)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error %q did not contain %q", err.Error(), tc.want)
			}
		})
	}
}

func TestNewReverseShellAcceptsValidConfig(t *testing.T) {
	h, err := NewReverseShell("get-shell", ReverseShellConfig{OperatorAddr: "10.0.0.5:4444"})
	if err != nil {
		t.Fatalf("NewReverseShell: %v", err)
	}
	if h.Name() != "get-shell" {
		t.Fatalf("Name = %q want get-shell", h.Name())
	}
}

// installStubs swaps the handler's dial+spawn closures and wires a
// fireDone signal so the test can wait on goroutine completion
// without racing on package-level state.
func installStubs(h *ReverseShell, dial dialerFunc, spawn spawnerFunc) <-chan struct{} {
	done := make(chan struct{})
	h.dialer = dial
	h.spawner = spawn
	h.fireDone = done
	return done
}

func TestReverseShellExecuteReturnsImmediatelyAndFiresAsync(t *testing.T) {
	t.Parallel()
	conn := &fakeConn{}
	dialed := atomic.Int32{}

	h, err := NewReverseShell("get-shell", ReverseShellConfig{
		OperatorAddr: "10.0.0.5:4444",
		ShellPath:    "/bin/true",
		ShellArgs:    []string{},
	})
	if err != nil {
		t.Fatalf("NewReverseShell: %v", err)
	}
	done := installStubs(h,
		func(_, _ string, _ time.Duration, _ bool, _ *tls.Config) (net.Conn, error) {
			dialed.Add(1)
			return conn, nil
		},
		func(ctx context.Context, _ string, _ ...string) *exec.Cmd {
			return exec.CommandContext(ctx, "/bin/true")
		},
	)

	start := time.Now()
	if err := h.Execute(context.Background(), intents.Event{ClientID: "op-jc", SourceIP: "127.0.0.1"}); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if elapsed := time.Since(start); elapsed > 100*time.Millisecond {
		t.Fatalf("Execute did not return immediately: took %v", elapsed)
	}
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatalf("detached shell goroutine never completed")
	}
	if dialed.Load() != 1 {
		t.Fatalf("dialer called %d times, want 1", dialed.Load())
	}
}

func TestReverseShellExecuteSwallowsDialErrors(t *testing.T) {
	t.Parallel()
	spawnerCalled := atomic.Bool{}
	h, _ := NewReverseShell("get-shell", ReverseShellConfig{
		OperatorAddr: "10.0.0.5:4444",
		ShellPath:    "/bin/true",
	})
	done := installStubs(h,
		func(_, _ string, _ time.Duration, _ bool, _ *tls.Config) (net.Conn, error) {
			return nil, errors.New("connection refused")
		},
		func(ctx context.Context, _ string, _ ...string) *exec.Cmd {
			spawnerCalled.Store(true)
			return exec.CommandContext(ctx, "/bin/true")
		},
	)
	if err := h.Execute(context.Background(), intents.Event{}); err != nil {
		t.Fatalf("Execute should swallow dial errors, got %v", err)
	}
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatalf("goroutine never completed")
	}
	if spawnerCalled.Load() {
		t.Fatalf("spawner should not be called when dial fails")
	}
}

func TestReverseShellDefaultsApplied(t *testing.T) {
	h, err := NewReverseShell("x", ReverseShellConfig{OperatorAddr: "1.2.3.4:5"})
	if err != nil {
		t.Fatalf("NewReverseShell: %v", err)
	}
	if h.timeout != 5*time.Second {
		t.Fatalf("default DialTimeout: got %v want 5s", h.timeout)
	}
	if h.maxLife != 30*time.Minute {
		t.Fatalf("default MaxSessionDuration: got %v want 30m", h.maxLife)
	}
	if h.shell == "" {
		t.Fatalf("default shell path empty")
	}
}

// fakeConn is a minimal net.Conn for tests.
type fakeConn struct{}

func (c *fakeConn) Read(b []byte) (int, error)       { return 0, fmt.Errorf("EOF") }
func (c *fakeConn) Write(b []byte) (int, error)      { return len(b), nil }
func (c *fakeConn) Close() error                     { return nil }
func (c *fakeConn) LocalAddr() net.Addr              { return nil }
func (c *fakeConn) RemoteAddr() net.Addr             { return nil }
func (c *fakeConn) SetDeadline(time.Time) error      { return nil }
func (c *fakeConn) SetReadDeadline(time.Time) error  { return nil }
func (c *fakeConn) SetWriteDeadline(time.Time) error { return nil }
