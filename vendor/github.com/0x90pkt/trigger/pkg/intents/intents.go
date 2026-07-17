// SPDX-License-Identifier: GPL-3.0-or-later
//
// Package intents defines the handler interface and registry that the
// listener uses to dispatch accepted triggers.
//
// A trigger that passes HMAC verification, replay protection, source
// and client allowlists, and rate limiting is then resolved against
// the registry by its intent label. If a handler is registered, its
// Execute method is called inside a worker goroutine, wrapped with
// panic recovery and a per-handler context deadline.
//
// Built-in handlers (noop, exec, webhook) live in subpackages; this
// file contains only the interface, the registry, and the dispatch
// event type. Anyone embedding the listener (the standalone binary,
// the eventual Sliver fork, third-party tools) implements Handler
// directly to bind intents to custom actions.
package intents

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"
)

// Event carries everything a handler needs about an accepted trigger.
// It is constructed by the listener after all validation passes and
// passed to Handler.Execute. Handlers must treat fields as read-only.
type Event struct {
	Intent    string
	ClientID  string
	SourceIP  string
	Nonce     string
	Timestamp time.Time
}

// Handler is the contract every intent action implements. Name returns
// the intent label this handler answers to. Execute runs the action;
// the listener invokes it in a worker goroutine with panic recovery
// and a per-handler context deadline applied by the registry.
//
// Execute MUST be safe to call concurrently from multiple workers
// against the same Handler instance — the registry does not serialize
// dispatch.
type Handler interface {
	Name() string
	Execute(ctx context.Context, evt Event) error
}

// Registry maps intent names to handlers. A Registry is safe for
// concurrent use. Once the listener is running, callers should treat
// the registry as effectively read-only (registration is cheap, but
// dispatch traffic doesn't expect it).
type Registry struct {
	mu       sync.RWMutex
	handlers map[string]Handler
}

// NewRegistry returns an empty registry. The zero value is not usable
// — always construct via NewRegistry.
func NewRegistry() *Registry {
	return &Registry{handlers: make(map[string]Handler)}
}

// Register adds h under its Name(). Returns an error if the name is
// empty or if a handler with the same name is already registered.
// Registration is not idempotent on purpose: silent overwrites hide
// config bugs.
func (r *Registry) Register(h Handler) error {
	if h == nil {
		return errors.New("handler is nil")
	}
	name := h.Name()
	if name == "" {
		return errors.New("handler Name() returned empty string")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.handlers[name]; exists {
		return fmt.Errorf("intent %q already registered", name)
	}
	r.handlers[name] = h
	return nil
}

// Resolve looks up the handler bound to intent. The bool return is
// false (and the Handler nil) if no handler is registered.
func (r *Registry) Resolve(intent string) (Handler, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	h, ok := r.handlers[intent]
	return h, ok
}

// List returns the registered intent names, sorted, for operator
// introspection. Returns an empty slice if nothing is registered.
func (r *Registry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	names := make([]string, 0, len(r.handlers))
	for name := range r.handlers {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// Len returns the number of registered handlers.
func (r *Registry) Len() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.handlers)
}
