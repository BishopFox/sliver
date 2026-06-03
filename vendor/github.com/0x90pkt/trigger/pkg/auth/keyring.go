// SPDX-License-Identifier: GPL-3.0-or-later
//
// Package auth provides HMAC-key resolution for the trigger listener.
// It wraps the mapping from a wire message's client_id to the secret
// the listener should HMAC-verify against.
//
// Two modes:
//
//   - Single shared secret. Every client_id verifies against the same
//     key. Simplest; what the standalone has always done.
//
//   - Per-client keyring. Each known client_id has its own key; an
//     optional default key applies to clients not in the map. Strict
//     mode disables the default, so unknown client_ids reject without
//     a fallback path.
//
// The listener calls Keyring.SecretFor(clientID) AFTER decoding the
// wire message (so it has the client_id) but BEFORE HMAC verify (so
// it knows which key to use). The keyring returns the secret and a
// bool indicating whether a match was found.
package auth

import (
	"errors"
	"fmt"
	"sync"
)

// Keyring resolves a client_id to the HMAC secret bytes that should
// verify its messages. Safe for concurrent use.
type Keyring struct {
	mu      sync.RWMutex
	clients map[string][]byte
	def     []byte
	strict  bool
}

// Options for keyring construction.
type Options struct {
	// DefaultSecret is the fallback HMAC key for client_ids not in the
	// per-client map. Empty disables the fallback.
	DefaultSecret string
	// Strict, if true, rejects any client_id not in the per-client map
	// even when DefaultSecret is set. Useful for hardened deployments
	// where every operator must have an explicit key.
	Strict bool
}

// NewKeyring constructs a Keyring with the given default and strict
// behavior. Use Add to populate per-client keys.
func NewKeyring(opts Options) *Keyring {
	return &Keyring{
		clients: make(map[string][]byte),
		def:     []byte(opts.DefaultSecret),
		strict:  opts.Strict,
	}
}

// Add binds clientID → secret. Returns an error on duplicate; the
// keyring refuses silent overwrites because they hide config bugs.
func (k *Keyring) Add(clientID, secret string) error {
	if clientID == "" {
		return errors.New("client_id must be set")
	}
	if secret == "" {
		return fmt.Errorf("secret for %q is empty", clientID)
	}
	k.mu.Lock()
	defer k.mu.Unlock()
	if _, exists := k.clients[clientID]; exists {
		return fmt.Errorf("client_id %q already has a key", clientID)
	}
	k.clients[clientID] = []byte(secret)
	return nil
}

// SecretFor returns the HMAC secret bytes for clientID. The bool is
// true on match. In strict mode, an unknown client_id returns
// (nil, false) regardless of any configured default.
//
// Caller MUST NOT mutate the returned slice — it is the keyring's
// own storage.
func (k *Keyring) SecretFor(clientID string) ([]byte, bool) {
	k.mu.RLock()
	defer k.mu.RUnlock()
	if secret, ok := k.clients[clientID]; ok {
		return secret, true
	}
	if k.strict {
		return nil, false
	}
	if len(k.def) == 0 {
		return nil, false
	}
	return k.def, true
}

// HasDefault reports whether a fallback default key is configured AND
// strict mode is off. Useful for config validation: "either set a
// default or register every client_id."
func (k *Keyring) HasDefault() bool {
	k.mu.RLock()
	defer k.mu.RUnlock()
	return !k.strict && len(k.def) > 0
}

// Size returns the number of per-client entries (test/diagnostic).
func (k *Keyring) Size() int {
	k.mu.RLock()
	defer k.mu.RUnlock()
	return len(k.clients)
}
