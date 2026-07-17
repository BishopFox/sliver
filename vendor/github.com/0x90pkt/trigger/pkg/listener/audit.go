// SPDX-License-Identifier: GPL-3.0-or-later

package listener

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"
	"time"
)

// AuditEvent is the structured record the listener emits for every
// packet it processes (accepted or rejected) and for major lifecycle
// transitions. It is JSON-marshaled by the default file sink, but
// custom sinks can route it anywhere (syslog, SIEM, in-memory buffer
// for tests).
//
// Field semantics:
//   - Time      always set, RFC3339Nano UTC.
//   - Event     a short label: "trigger_attempt", "server_started",
//     "server_stopped", "handler_panic", etc.
//   - SourceIP  remote UDP source, empty for lifecycle events.
//   - ClientID  populated once the wire message decodes.
//   - Intent    populated once the wire message decodes.
//   - Nonce     populated once the wire message decodes.
//   - Accepted  true iff the trigger passed every check and an intent
//     handler ran without error.
//   - Reason    short human-readable disposition.
//   - Extra     freeform key/value bag for handler-specific or
//     lifecycle-specific context.
type AuditEvent struct {
	Time     time.Time      `json:"time"`
	Event    string         `json:"event"`
	SourceIP string         `json:"source_ip,omitempty"`
	ClientID string         `json:"client_id,omitempty"`
	Intent   string         `json:"intent,omitempty"`
	Nonce    string         `json:"nonce,omitempty"`
	Accepted bool           `json:"accepted"`
	Reason   string         `json:"reason"`
	ServerID string         `json:"server_id,omitempty"`
	Extra    map[string]any `json:"extra,omitempty"`
}

// AuditSink consumes audit events. Implementations MUST be safe to
// call concurrently — the listener emits from multiple worker
// goroutines without serialization.
type AuditSink interface {
	Emit(evt AuditEvent)
}

// FileAuditSink writes one JSON object per line to an io.Writer.
// Concurrent Emit calls are serialized by an internal mutex so output
// stays line-coherent.
type FileAuditSink struct {
	mu sync.Mutex
	w  io.Writer
}

// NewFileAuditSink wraps any io.Writer as a line-delimited JSON sink.
func NewFileAuditSink(w io.Writer) *FileAuditSink {
	return &FileAuditSink{w: w}
}

// OpenFileAuditSink opens path in append mode (0600) and returns a sink
// plus a closer that callers should invoke on shutdown.
func OpenFileAuditSink(path string) (*FileAuditSink, io.Closer, error) {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return nil, nil, fmt.Errorf("open audit log %q: %w", path, err)
	}
	return NewFileAuditSink(f), f, nil
}

// Emit serializes evt as a single JSON line and writes it. Errors are
// swallowed deliberately: the audit log MUST NOT block or fail the hot
// path. The default sink emits a best-effort fallback line if the
// primary marshal fails.
func (s *FileAuditSink) Emit(evt AuditEvent) {
	s.mu.Lock()
	defer s.mu.Unlock()
	b, err := json.Marshal(evt)
	if err != nil {
		fallback := fmt.Sprintf(
			`{"time":%q,"event":"audit_encode_failure","accepted":false,"reason":%q}`+"\n",
			evt.Time.UTC().Format(time.RFC3339Nano),
			err.Error(),
		)
		_, _ = s.w.Write([]byte(fallback))
		return
	}
	b = append(b, '\n')
	_, _ = s.w.Write(b)
}

// MultiSink fans out emit calls to a fixed set of underlying sinks in
// order. Useful when operators want both a file and stdout.
type MultiSink struct {
	sinks []AuditSink
}

// NewMultiSink returns a sink that calls Emit on each underlying sink
// for every event. Nil sinks are silently dropped.
func NewMultiSink(sinks ...AuditSink) *MultiSink {
	filtered := make([]AuditSink, 0, len(sinks))
	for _, s := range sinks {
		if s != nil {
			filtered = append(filtered, s)
		}
	}
	return &MultiSink{sinks: filtered}
}

// Emit calls Emit on every wrapped sink. Errors are sink-local.
func (m *MultiSink) Emit(evt AuditEvent) {
	for _, s := range m.sinks {
		s.Emit(evt)
	}
}
