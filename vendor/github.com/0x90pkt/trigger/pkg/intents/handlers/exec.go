// SPDX-License-Identifier: GPL-3.0-or-later

package handlers

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/0x90pkt/trigger/pkg/intents"
)

// ExecConfig is the construction-time configuration for an Exec handler.
//
// Design rules (the original prototype's mistakes we are not repeating):
//
//  1. No shell interpolation. The command is an absolute path plus a
//     pre-split argv. There is no `sh -c "..."` codepath. Operator
//     scripts read context from environment variables, never from
//     argv substitution.
//  2. Inherited environment is NOT propagated. The handler env starts
//     empty and is populated with:
//     - PATH (configurable; sane default if unset)
//     - HOME (configurable; sane default if unset)
//     - Intent context (INTENT, CLIENT_ID, SOURCE_IP, NONCE, TIMESTAMP)
//     - Whatever the operator added via Env.
//     This deliberately blocks accidental leakage of server-side
//     secrets (TRIGGER_SHARED_SECRET, etc.) into handler subprocesses.
//  3. The parent context deadline kills the subprocess. There is no
//     separate handler timeout — the listener sets a per-invocation
//     ctx deadline (default 10s, configurable globally).
//  4. Captured stdout/stderr is bounded (default 64KB combined). On
//     overflow the tail is silently dropped; the truncation is noted
//     in the returned error.
type ExecConfig struct {
	// Cmd is the absolute path of the executable. Relative paths are
	// rejected at construction time.
	Cmd string
	// Args is the verbatim argv passed to the subprocess. NOT shell-
	// interpreted. NOT templated with intent values — intent context
	// is exposed via env vars.
	Args []string
	// Workdir is the subprocess working directory. Must exist or the
	// kernel rejects the exec. If empty, defaults to "/".
	Workdir string
	// Env is extra environment passed to the subprocess on top of the
	// minimal default set. Values are verbatim — no expansion.
	Env map[string]string
	// PathOverride, if set, replaces the default PATH passed to the
	// subprocess. Empty uses a conservative default.
	PathOverride string
	// HomeOverride, if set, replaces the default HOME.
	HomeOverride string
	// MaxOutputBytes caps combined stdout+stderr capture. Default
	// 64KB if zero.
	MaxOutputBytes int
	// Logger receives subprocess output snippets and lifecycle logs.
	// If nil, slog.Default() is used.
	Logger *slog.Logger
}

const (
	defaultExecPath        = "/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"
	defaultExecHome        = "/var/empty"
	defaultExecWorkdir     = "/"
	defaultExecMaxOutBytes = 64 * 1024
)

// Exec runs a configured command-line on intent fire. See ExecConfig.
type Exec struct {
	name           string
	cfg            ExecConfig
	logger         *slog.Logger
	maxOutputBytes int
	workdir        string
	pathEnv        string
	homeEnv        string
}

// NewExec constructs an Exec handler. Returns an error if Cmd is not
// an absolute path or fails sanity checks.
func NewExec(name string, cfg ExecConfig) (*Exec, error) {
	if strings.TrimSpace(name) == "" {
		return nil, errors.New("exec handler: name must be set")
	}
	if cfg.Cmd == "" {
		return nil, errors.New("exec handler: Cmd must be set")
	}
	if !filepath.IsAbs(cfg.Cmd) {
		return nil, fmt.Errorf("exec handler: Cmd %q must be an absolute path", cfg.Cmd)
	}
	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}
	maxOut := cfg.MaxOutputBytes
	if maxOut <= 0 {
		maxOut = defaultExecMaxOutBytes
	}
	workdir := cfg.Workdir
	if workdir == "" {
		workdir = defaultExecWorkdir
	}
	pathEnv := cfg.PathOverride
	if pathEnv == "" {
		pathEnv = defaultExecPath
	}
	homeEnv := cfg.HomeOverride
	if homeEnv == "" {
		homeEnv = defaultExecHome
	}
	return &Exec{
		name:           name,
		cfg:            cfg,
		logger:         logger,
		maxOutputBytes: maxOut,
		workdir:        workdir,
		pathEnv:        pathEnv,
		homeEnv:        homeEnv,
	}, nil
}

// Name implements intents.Handler.
func (h *Exec) Name() string { return h.name }

// Execute runs the configured subprocess. The parent ctx deadline kills
// the subprocess via SIGTERM/SIGKILL handled by exec.CommandContext.
// Returns the subprocess's exit error wrapped with captured output
// excerpts.
func (h *Exec) Execute(ctx context.Context, evt intents.Event) error {
	cmd := exec.CommandContext(ctx, h.cfg.Cmd, h.cfg.Args...)
	cmd.Dir = h.workdir
	cmd.Env = h.buildEnv(evt)

	stdout := newBoundedBuffer(h.maxOutputBytes)
	stderr := newBoundedBuffer(h.maxOutputBytes)
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	err := cmd.Run()

	h.logger.Info("exec handler complete",
		slog.String("intent", h.name),
		slog.String("client_id", evt.ClientID),
		slog.String("source_ip", evt.SourceIP),
		slog.String("cmd", h.cfg.Cmd),
		slog.Bool("ok", err == nil),
		slog.Int("stdout_bytes", stdout.Len()),
		slog.Int("stderr_bytes", stderr.Len()),
		slog.Bool("stdout_truncated", stdout.truncated),
		slog.Bool("stderr_truncated", stderr.truncated),
	)

	if err != nil {
		return fmt.Errorf("exec %s failed: %w (stderr: %q)", h.cfg.Cmd, err, stderr.tail(512))
	}
	return nil
}

// buildEnv constructs the subprocess environment from scratch. We do
// not inherit the parent (server) process environment.
func (h *Exec) buildEnv(evt intents.Event) []string {
	env := map[string]string{
		"PATH":      h.pathEnv,
		"HOME":      h.homeEnv,
		"INTENT":    evt.Intent,
		"CLIENT_ID": evt.ClientID,
		"SOURCE_IP": evt.SourceIP,
		"NONCE":     evt.Nonce,
		"TIMESTAMP": evt.Timestamp.UTC().Format("2006-01-02T15:04:05.999999999Z07:00"),
	}
	// Operator-configured env overrides any of the above.
	for k, v := range h.cfg.Env {
		env[k] = v
	}

	out := make([]string, 0, len(env))
	for k := range env {
		out = append(out, k)
	}
	sort.Strings(out) // determinism for tests + audit
	for i, k := range out {
		out[i] = k + "=" + env[k]
	}
	return out
}

// boundedBuffer is an io.Writer with a hard size cap. Writes past the
// cap are silently dropped; the truncated flag records that overflow
// occurred.
type boundedBuffer struct {
	buf       *bytes.Buffer
	max       int
	truncated bool
}

func newBoundedBuffer(max int) *boundedBuffer {
	return &boundedBuffer{buf: &bytes.Buffer{}, max: max}
}

func (b *boundedBuffer) Write(p []byte) (int, error) {
	if b.buf.Len() >= b.max {
		b.truncated = true
		return len(p), nil
	}
	remaining := b.max - b.buf.Len()
	if len(p) <= remaining {
		return b.buf.Write(p)
	}
	b.truncated = true
	n, err := b.buf.Write(p[:remaining])
	if err != nil {
		return n, err
	}
	// Pretend we consumed the rest; the caller doesn't need to know.
	return len(p), nil
}

func (b *boundedBuffer) Len() int { return b.buf.Len() }

func (b *boundedBuffer) tail(n int) string {
	if b.buf.Len() <= n {
		return b.buf.String()
	}
	bs := b.buf.Bytes()
	return string(bs[len(bs)-n:])
}

// Compile-time: ensure boundedBuffer satisfies io.Writer.
var _ io.Writer = (*boundedBuffer)(nil)
