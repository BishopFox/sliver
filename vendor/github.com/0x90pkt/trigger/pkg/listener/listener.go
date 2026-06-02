// SPDX-License-Identifier: GPL-3.0-or-later
//
// Package listener implements the authenticated UDP trigger listener.
// It is the embeddable core that both the standalone `trigger-server`
// binary and the (future) Sliver fork consume as a library.
//
// Lifecycle:
//
//	cfg := listener.Config{ /* ... */ }
//	l, err := listener.New(cfg)
//	if err != nil { return err }
//	if err := l.Start(ctx); err != nil { return err }
//	// Start blocks until ctx cancel or fatal error. To stop:
//	cancel()  // or send a signal that triggers cancel
//
// Validation pipeline (in order):
//  1. global packets-per-second cap (cheap pre-parse reject)
//  2. size cap
//  3. wire decode + structural validation
//  4. source-IP allowlist
//  5. client-ID allowlist
//  6. timestamp clock-skew bound
//  7. HMAC-SHA256 verification
//  8. replay-nonce uniqueness (bounded cache)
//  9. per-(client_id,source_ip) rate limit (post-HMAC: client_id is now authenticated)
//  10. intent registry lookup
//  11. handler dispatch (with panic recovery + ctx deadline)
//
// Each step that rejects emits an AuditEvent with a categorized reason.
// No step emits a response packet — the design is deliberately quiet
// (ACKs would telegraph listener existence and create a timing oracle
// between reject branches). Operator feedback is the audit log.
//
// The pre-HMAC rate limit is global by design: per-source rate limiting
// before HMAC verify enables a spoofed-source-IP DoS that locks out
// legitimate operators. After HMAC, source identity is cryptographic
// (the (client_id, source_ip) tuple can't be spoofed without the key),
// so per-key fairness applies safely.
package listener

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"time"

	"github.com/0x90pkt/trigger/pkg/auth"
	"github.com/0x90pkt/trigger/pkg/intents"
	"github.com/0x90pkt/trigger/pkg/protocol"
)

// Config holds all listener configuration. Validate via New; do not
// mutate after construction.
type Config struct {
	// BindIP is the UDP bind address, e.g. "0.0.0.0" or "127.0.0.1".
	BindIP string
	// BindPort is the UDP bind port. Set to 0 for kernel-assigned
	// ephemeral (test fixtures, transient listeners); production
	// deployments pin a fixed port.
	BindPort int

	// Workers is the number of goroutines dispatching accepted
	// triggers. At least 1.
	Workers int

	// MaxClockSkew is the maximum absolute difference between the
	// timestamp in a trigger and the server's wall clock.
	MaxClockSkew time.Duration

	// ReplayTTL is the duration a nonce stays in the replay cache.
	// Should comfortably exceed MaxClockSkew.
	ReplayTTL time.Duration

	// MaxMessageBytes caps UDP payload size. Packets larger than this
	// are silently dropped pre-parse.
	MaxMessageBytes int

	// GlobalRatePerSecond caps total packets-per-second across all
	// sources, applied BEFORE HMAC verify. Protects the worker pool
	// from a UDP flood. Default 500 if zero.
	GlobalRatePerSecond int

	// PerClientRequestsPerMinute caps requests per (client_id,
	// source_ip) tuple, applied AFTER HMAC verify (so the tuple is
	// authenticated and not spoofable). Default 240 if zero.
	PerClientRequestsPerMinute int

	// MaxRateLimitEntries bounds the post-HMAC rate-limiter map.
	// When full and a new key arrives, stale entries are swept; if
	// still full, the request is rejected. Default 50_000 if zero.
	MaxRateLimitEntries int

	// MaxReplayEntries bounds the replay-nonce cache. Same overflow
	// semantics as MaxRateLimitEntries. Default 50_000 if zero.
	MaxReplayEntries int

	// AllowedClientIDs restricts which client_id values are accepted.
	// Empty map means all signed clients are allowed.
	AllowedClientIDs map[string]struct{}

	// AllowedSources is the source-IP/CIDR allowlist applied before
	// HMAC verify. Nil or empty means allow-all.
	AllowedSources *auth.SourceAllowlist

	// Keyring resolves a client_id to the HMAC secret used to verify
	// its messages. Required. For the simple single-secret case use
	// auth.NewKeyring(auth.Options{DefaultSecret: "..."}).
	Keyring *auth.Keyring

	// ServerID is included in audit events to disambiguate logs from
	// multiple listener instances. Required.
	ServerID string

	// ReadBufferBytes is the UDP read-buffer size. Must be >=
	// MaxMessageBytes.
	ReadBufferBytes int

	// JobQueueBufferSize is the depth of the worker queue.
	JobQueueBufferSize int

	// Sink receives audit events. Required.
	Sink AuditSink

	// Logger is used for non-audit diagnostics (bind failures, panic
	// recoveries, lifecycle). If nil, slog.Default() is used.
	Logger *slog.Logger

	// Handlers dispatches accepted triggers by intent. Required.
	// Empty registry means every authenticated trigger is rejected
	// with reason "no handler registered".
	Handlers *intents.Registry

	// HandlerTimeout bounds each Handler.Execute call. Default 10s if
	// zero.
	HandlerTimeout time.Duration
}

// validate returns an error if any required field is missing or any
// numeric bound is unreasonable.
func (c *Config) validate() error {
	if c.BindPort < 0 || c.BindPort > 65535 {
		return errors.New("BindPort must be 0..65535 (0 = ephemeral, kernel-assigned)")
	}
	if c.Workers < 1 {
		return errors.New("Workers must be >= 1")
	}
	if c.MaxClockSkew <= 0 {
		return errors.New("MaxClockSkew must be positive")
	}
	if c.ReplayTTL <= 0 {
		return errors.New("ReplayTTL must be positive")
	}
	if c.MaxMessageBytes < 128 {
		return errors.New("MaxMessageBytes must be >= 128")
	}
	if c.GlobalRatePerSecond < 0 {
		return errors.New("GlobalRatePerSecond must be >= 0 (0 means use default 500)")
	}
	if c.PerClientRequestsPerMinute < 0 {
		return errors.New("PerClientRequestsPerMinute must be >= 0 (0 means use default 240)")
	}
	if c.MaxRateLimitEntries < 0 {
		return errors.New("MaxRateLimitEntries must be >= 0 (0 means use default 50000)")
	}
	if c.MaxReplayEntries < 0 {
		return errors.New("MaxReplayEntries must be >= 0 (0 means use default 50000)")
	}
	if c.ReadBufferBytes < c.MaxMessageBytes {
		return errors.New("ReadBufferBytes must be >= MaxMessageBytes")
	}
	if c.JobQueueBufferSize < 1 {
		return errors.New("JobQueueBufferSize must be >= 1")
	}
	if c.Keyring == nil {
		return errors.New("Keyring must be set (use auth.NewKeyring(...))")
	}
	if c.ServerID == "" {
		return errors.New("ServerID must be set")
	}
	if c.Sink == nil {
		return errors.New("Sink must be set")
	}
	if c.Handlers == nil {
		return errors.New("Handlers must be set (use intents.NewRegistry() for an empty one)")
	}
	if net.ParseIP(c.BindIP) == nil {
		return fmt.Errorf("invalid BindIP %q", c.BindIP)
	}
	return nil
}

// Stats is a snapshot of the listener's lifetime counters. Read via
// Listener.Stats; all fields are loaded atomically.
type Stats struct {
	PacketsReceived int64
	PacketsAccepted int64
	PacketsRejected int64
	HandlerPanics   int64
	HandlerErrors   int64
}

// Listener is the runtime instance. Construct via New; call Start
// (which blocks). Stop is safe to call multiple times.
type Listener struct {
	cfg            Config
	conn           atomic.Pointer[net.UDPConn]
	replay         *replayGuard
	globalRate     *globalRateLimiter
	keyedRate      *keyedRateLimiter
	logger         *slog.Logger
	handlerTimeout time.Duration

	startOnce sync.Once
	stopOnce  sync.Once
	stopCh    chan struct{}

	packetsReceived atomic.Int64
	packetsAccepted atomic.Int64
	packetsRejected atomic.Int64
	handlerPanics   atomic.Int64
	handlerErrors   atomic.Int64
}

// New validates cfg and returns a ready-to-Start Listener. It does NOT
// bind the UDP socket; that happens in Start.
func New(cfg Config) (*Listener, error) {
	if err := cfg.validate(); err != nil {
		return nil, err
	}
	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}
	timeout := cfg.HandlerTimeout
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	globalRate := cfg.GlobalRatePerSecond
	if globalRate == 0 {
		globalRate = 500
	}
	perClientRate := cfg.PerClientRequestsPerMinute
	if perClientRate == 0 {
		perClientRate = 240
	}
	maxRateEntries := cfg.MaxRateLimitEntries
	if maxRateEntries == 0 {
		maxRateEntries = 50_000
	}
	maxReplayEntries := cfg.MaxReplayEntries
	if maxReplayEntries == 0 {
		maxReplayEntries = 50_000
	}
	return &Listener{
		cfg:            cfg,
		replay:         newReplayGuard(cfg.ReplayTTL, maxReplayEntries),
		globalRate:     newGlobalRateLimiter(globalRate),
		keyedRate:      newKeyedRateLimiter(perClientRate, maxRateEntries),
		logger:         logger,
		handlerTimeout: timeout,
		stopCh:         make(chan struct{}),
	}, nil
}

// Stats returns a snapshot of lifetime counters.
func (l *Listener) Stats() Stats {
	return Stats{
		PacketsReceived: l.packetsReceived.Load(),
		PacketsAccepted: l.packetsAccepted.Load(),
		PacketsRejected: l.packetsRejected.Load(),
		HandlerPanics:   l.handlerPanics.Load(),
		HandlerErrors:   l.handlerErrors.Load(),
	}
}

// LocalAddr returns the bound UDP address, or nil if Start has not yet
// succeeded. Useful for tests that bind to port 0. Safe for concurrent
// use with Start.
func (l *Listener) LocalAddr() net.Addr {
	conn := l.conn.Load()
	if conn == nil {
		return nil
	}
	return conn.LocalAddr()
}

// Start binds the UDP socket and runs until ctx is canceled or a fatal
// error occurs. It is an error to call Start more than once.
func (l *Listener) Start(ctx context.Context) error {
	var startErr error
	called := false
	l.startOnce.Do(func() {
		called = true
		startErr = l.run(ctx)
	})
	if !called {
		return errors.New("Listener.Start called more than once")
	}
	return startErr
}

// Stop interrupts the listener if it is currently in Start. Safe to
// call multiple times. Stop does not block on graceful drain; Start
// returns when drain completes.
func (l *Listener) Stop() {
	l.stopOnce.Do(func() {
		close(l.stopCh)
		if conn := l.conn.Load(); conn != nil {
			_ = conn.Close()
		}
	})
}

func (l *Listener) run(ctx context.Context) error {
	udpAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", l.cfg.BindIP, l.cfg.BindPort))
	if err != nil {
		return fmt.Errorf("resolve bind addr: %w", err)
	}
	conn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		return fmt.Errorf("bind udp: %w", err)
	}
	l.conn.Store(conn)

	jobs := make(chan packet, l.cfg.JobQueueBufferSize)

	var wg sync.WaitGroup
	for i := 0; i < l.cfg.Workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			l.workerLoop(ctx, jobs)
		}()
	}

	l.emit(AuditEvent{
		Time:     time.Now().UTC(),
		Event:    "server_started",
		ServerID: l.cfg.ServerID,
		Accepted: true,
		Reason:   "ok",
		Extra: map[string]any{
			"bind":    fmt.Sprintf("%s:%d", l.cfg.BindIP, l.cfg.BindPort),
			"workers": l.cfg.Workers,
		},
	})

	// Cancellation watcher: cancel ctx OR explicit Stop both unblock
	// the read loop by closing the UDP socket.
	go func() {
		select {
		case <-ctx.Done():
		case <-l.stopCh:
		}
		_ = conn.Close()
	}()

	readBuffer := make([]byte, l.cfg.ReadBufferBytes)
	for {
		n, remoteAddr, err := conn.ReadFromUDP(readBuffer)
		if err != nil {
			// Shutdown path: ctx canceled OR Stop called → conn closed.
			select {
			case <-ctx.Done():
			case <-l.stopCh:
			default:
				// Fatal read error unrelated to shutdown.
				close(jobs)
				wg.Wait()
				return fmt.Errorf("udp read: %w", err)
			}
			close(jobs)
			wg.Wait()
			l.emit(AuditEvent{
				Time:     time.Now().UTC(),
				Event:    "server_stopped",
				ServerID: l.cfg.ServerID,
				Accepted: true,
				Reason:   "ok",
			})
			return nil
		}

		payload := make([]byte, n)
		copy(payload, readBuffer[:n])
		remoteCopy := &net.UDPAddr{
			IP:   append(net.IP(nil), remoteAddr.IP...),
			Port: remoteAddr.Port,
			Zone: remoteAddr.Zone,
		}

		l.packetsReceived.Add(1)

		select {
		case jobs <- packet{payload: payload, sourceIP: remoteAddr.IP.String(), remoteAddr: remoteCopy}:
		case <-ctx.Done():
			close(jobs)
			wg.Wait()
			return nil
		case <-l.stopCh:
			close(jobs)
			wg.Wait()
			return nil
		}
	}
}

// packet is the internal per-message handoff struct from reader to
// workers. The payload is always a private copy of the UDP buffer.
type packet struct {
	payload    []byte
	sourceIP   string
	remoteAddr *net.UDPAddr
}

func (l *Listener) workerLoop(ctx context.Context, jobs <-chan packet) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-l.stopCh:
			return
		case pkt, ok := <-jobs:
			if !ok {
				return
			}
			l.handlePacket(ctx, pkt)
		}
	}
}

// processResult is the outcome of running pkt through the validation
// pipeline. Audit is always populated; Event is populated iff the
// packet passed every check up through replay (i.e. ready to dispatch).
type processResult struct {
	audit AuditEvent
	event *intents.Event // nil unless audit.Accepted-ready
}

func (l *Listener) handlePacket(ctx context.Context, pkt packet) {
	result := l.processPacket(pkt)
	if result.event == nil {
		// Rejected before reaching dispatch. Audit already final.
		l.packetsRejected.Add(1)
		l.emit(result.audit)
		return
	}

	// Reached dispatch. Resolve handler and run it.
	handler, ok := l.cfg.Handlers.Resolve(result.event.Intent)
	if !ok {
		result.audit.Accepted = false
		result.audit.Reason = "no handler registered"
		l.packetsRejected.Add(1)
		l.emit(result.audit)
		return
	}

	err := l.invokeHandler(ctx, handler, *result.event)
	if err != nil {
		result.audit.Accepted = false
		result.audit.Reason = fmt.Sprintf("handler error: %v", err)
		l.handlerErrors.Add(1)
		l.packetsRejected.Add(1)
		l.emit(result.audit)
		return
	}

	result.audit.Accepted = true
	result.audit.Reason = "accepted"
	l.packetsAccepted.Add(1)
	l.emit(result.audit)
}

// invokeHandler runs handler.Execute with a fresh context deadline and
// a panic recovery wrapper. A handler panic is converted to an error
// and counted; the worker keeps running.
func (l *Listener) invokeHandler(parentCtx context.Context, handler intents.Handler, evt intents.Event) (err error) {
	ctx, cancel := context.WithTimeout(parentCtx, l.handlerTimeout)
	defer cancel()
	defer func() {
		if r := recover(); r != nil {
			l.handlerPanics.Add(1)
			stack := string(debug.Stack())
			l.logger.Error("intent handler panic",
				slog.String("intent", evt.Intent),
				slog.String("client_id", evt.ClientID),
				slog.String("source_ip", evt.SourceIP),
				slog.Any("panic", r),
				slog.String("stack", stack),
			)
			err = fmt.Errorf("handler panic: %v", r)
		}
	}()
	return handler.Execute(ctx, evt)
}

// processPacket runs the validation pipeline up to (but not including)
// handler dispatch. It returns a ready-to-emit audit envelope and, if
// the packet passed every check, an intents.Event for dispatch.
//
// This function is intentionally pure (no I/O, no logging) so it can
// be unit-tested without standing up the full Listener.
func (l *Listener) processPacket(pkt packet) processResult {
	audit := AuditEvent{
		Time:     time.Now().UTC(),
		Event:    "trigger_attempt",
		SourceIP: pkt.sourceIP,
		ServerID: l.cfg.ServerID,
		Accepted: false,
		Reason:   "unknown",
	}

	now := time.Now().UTC()

	// 1. Global packets-per-second cap — pre-parse, source-IP agnostic.
	//    UDP source IPs are spoofable; per-source rate-limit here would
	//    be a free DoS lever for any attacker. The global cap protects
	//    the worker pool from sheer volume without distinguishing.
	if !l.globalRate.allow(now) {
		audit.Reason = "global rate limit exceeded"
		return processResult{audit: audit}
	}

	// 2. Size cap.
	if len(pkt.payload) > l.cfg.MaxMessageBytes {
		audit.Reason = "message exceeds max size"
		return processResult{audit: audit}
	}

	// 3. Wire decode + structural validation.
	msg, err := protocol.DecodeWire(pkt.payload)
	if err != nil {
		audit.Reason = fmt.Sprintf("protocol error: %v", err)
		return processResult{audit: audit}
	}

	audit.ClientID = msg.ClientID
	audit.Intent = msg.Intent
	audit.Nonce = msg.Nonce

	// 4-5. Allowlists (source IP/CIDR, client ID).
	if l.cfg.AllowedSources != nil && !l.cfg.AllowedSources.Contains(pkt.sourceIP) {
		audit.Reason = "source IP not allowed"
		return processResult{audit: audit}
	}
	if !setMember(msg.ClientID, l.cfg.AllowedClientIDs) {
		audit.Reason = "client ID not allowed"
		return processResult{audit: audit}
	}

	// 6. Timestamp skew.
	msgTime, err := protocol.ParseTimestamp(msg.Timestamp)
	if err != nil {
		audit.Reason = fmt.Sprintf("timestamp parse failed: %v", err)
		return processResult{audit: audit}
	}
	skew := now.Sub(msgTime)
	if skew < 0 {
		skew = -skew
	}
	if skew > l.cfg.MaxClockSkew {
		audit.Reason = fmt.Sprintf("clock skew too large: %s", skew)
		return processResult{audit: audit}
	}

	// 7. HMAC verify. After this, client_id is authenticated.
	//    Per-client key resolution: keyring lookup → strict-mode reject
	//    if client_id has no registered key (no fallback default).
	secret, ok := l.cfg.Keyring.SecretFor(msg.ClientID)
	if !ok {
		audit.Reason = "no key registered for client"
		return processResult{audit: audit}
	}
	verified, err := protocol.Verify(msg, string(secret))
	if err != nil {
		audit.Reason = fmt.Sprintf("signature verification error: %v", err)
		return processResult{audit: audit}
	}
	if !verified {
		audit.Reason = "signature verification failed"
		return processResult{audit: audit}
	}

	// 8. Replay uniqueness.
	if !l.replay.markIfNew(msg.Nonce, now) {
		audit.Reason = "replay detected or replay cache full"
		return processResult{audit: audit}
	}

	// 9. Per-(client_id, source_ip) rate limit. AFTER HMAC: the tuple
	//    is cryptographically authenticated, so an attacker can't
	//    target a specific operator by spoofing.
	rateKey := msg.ClientID + "@" + pkt.sourceIP
	if !l.keyedRate.allow(rateKey, now) {
		audit.Reason = "per-client rate limit exceeded"
		return processResult{audit: audit}
	}

	return processResult{
		audit: audit,
		event: &intents.Event{
			Intent:    msg.Intent,
			ClientID:  msg.ClientID,
			SourceIP:  pkt.sourceIP,
			Nonce:     msg.Nonce,
			Timestamp: msgTime,
		},
	}
}

// setMember returns true if allowlist is empty (allow-all) or if key
// is present. Matches the original main.go semantics.
func setMember(key string, allowlist map[string]struct{}) bool {
	if len(allowlist) == 0 {
		return true
	}
	_, ok := allowlist[key]
	return ok
}

// emit fans the event out to the configured sink. Centralizes the
// nil-check so callers don't have to.
func (l *Listener) emit(evt AuditEvent) {
	if l.cfg.Sink == nil {
		return
	}
	l.cfg.Sink.Emit(evt)
}
