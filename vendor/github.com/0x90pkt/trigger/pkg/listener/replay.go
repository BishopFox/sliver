// SPDX-License-Identifier: GPL-3.0-or-later

package listener

import (
	"sync"
	"time"
)

// replayGuard tracks recently-seen nonces to defeat replay attacks.
// Entries auto-expire after a configured TTL. MarkIfNew is the only
// mutator: it returns true on first observation and false on replay.
//
// Bounded by maxEntries. When the map is full and a new nonce arrives,
// the guard first sweeps expired entries. If still full after sweep,
// the new request is rejected — refusing to lose replay protection by
// silent eviction. Operators see "replay cache full" in the audit log
// and can raise the cap or investigate.
//
// Only verified-HMAC packets reach the replay guard (it runs at step
// 8 in the validation pipeline, after HMAC verify at step 7). Filling
// the cache requires the attacker to already hold the shared secret,
// at which point capacity DoS is the least of your problems.
type replayGuard struct {
	mu         sync.Mutex
	ttl        time.Duration
	maxEntries int
	seen       map[string]time.Time
}

func newReplayGuard(ttl time.Duration, maxEntries int) *replayGuard {
	return &replayGuard{
		ttl:        ttl,
		maxEntries: maxEntries,
		seen:       make(map[string]time.Time),
	}
}

// markIfNew records nonce with an expiry of now+ttl if unseen; returns
// false if nonce was already observed within the TTL window OR if the
// cache is full and contains no expired entries to evict.
func (r *replayGuard) markIfNew(nonce string, now time.Time) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Opportunistic sweep of expired entries FIRST so a previously-seen
	// nonce can be re-accepted after its TTL window closes.
	for k, exp := range r.seen {
		if now.After(exp) {
			delete(r.seen, k)
		}
	}

	if _, exists := r.seen[nonce]; exists {
		return false
	}

	if len(r.seen) >= r.maxEntries {
		return false // cap reached with all live entries; refuse rather than silently lose replay protection.
	}

	r.seen[nonce] = now.Add(r.ttl)
	return true
}

// size returns the current number of tracked entries (test/diagnostic).
func (r *replayGuard) size() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.seen)
}
