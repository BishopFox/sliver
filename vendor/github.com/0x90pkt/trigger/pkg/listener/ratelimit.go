// SPDX-License-Identifier: GPL-3.0-or-later

package listener

import (
	"sync"
	"time"
)

// globalRateLimiter applies a single packets-per-second cap across the
// whole listener. It is the pre-parse cheap reject that protects the
// worker pool from a UDP flood — applied BEFORE HMAC verification so a
// cost-cheap reject path exists for the highest-rate attacker.
//
// Single counter, reset on every second-boundary. No per-source state
// here — that's globalRateLimiter's whole point. The post-HMAC keyed
// limiter (keyedRateLimiter below) handles per-client fairness.
type globalRateLimiter struct {
	mu     sync.Mutex
	limit  int
	bucket int64
	count  int
}

func newGlobalRateLimiter(perSecond int) *globalRateLimiter {
	return &globalRateLimiter{limit: perSecond}
}

// allow returns true if the global packets-per-second budget is not
// yet exhausted; false if exhausted (caller should drop the packet).
func (r *globalRateLimiter) allow(now time.Time) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	bucket := now.Unix()
	if bucket != r.bucket {
		r.bucket = bucket
		r.count = 1
		return true
	}
	if r.count >= r.limit {
		return false
	}
	r.count++
	return true
}

// keyedRateLimiter applies a per-key requests-per-minute cap. Keys are
// opaque strings — used for `client_id:source_ip` composite keys AFTER
// HMAC verification (Wave-2: client_id is now authenticated, so the
// composite can't be spoofed without the key).
//
// Bounded by maxEntries. When the map is full and a new key arrives,
// the limiter first sweeps stale buckets (older than 2 minutes). If
// still full after sweep, the new request is rejected — better to fail
// the new caller than to silently evict a legitimate one that is being
// rate-limited correctly.
type keyedRateLimiter struct {
	mu         sync.Mutex
	limit      int
	maxEntries int
	state      map[string]rateCounter
}

type rateCounter struct {
	minuteBucket int64
	count        int
}

func newKeyedRateLimiter(perMinute, maxEntries int) *keyedRateLimiter {
	return &keyedRateLimiter{
		limit:      perMinute,
		maxEntries: maxEntries,
		state:      make(map[string]rateCounter),
	}
}

// allow returns true if key is under its per-minute limit. If the map
// is at capacity and key is new, sweep first; if still at capacity,
// reject with `false` (treated as rate-limited).
func (r *keyedRateLimiter) allow(key string, now time.Time) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	bucket := now.Unix() / 60
	existing, ok := r.state[key]
	if ok {
		if existing.minuteBucket != bucket {
			r.state[key] = rateCounter{minuteBucket: bucket, count: 1}
			return true
		}
		if existing.count >= r.limit {
			return false
		}
		existing.count++
		r.state[key] = existing
		return true
	}

	// New key. Enforce cap.
	if len(r.state) >= r.maxEntries {
		r.sweepStaleLocked(bucket)
		if len(r.state) >= r.maxEntries {
			return false
		}
	}
	r.state[key] = rateCounter{minuteBucket: bucket, count: 1}
	return true
}

// sweepStaleLocked removes entries whose bucket is more than one
// minute behind `now`. Caller must hold r.mu.
func (r *keyedRateLimiter) sweepStaleLocked(nowBucket int64) {
	for k, v := range r.state {
		if nowBucket-v.minuteBucket > 1 {
			delete(r.state, k)
		}
	}
}

// size returns the number of tracked keys (test/diagnostic).
func (r *keyedRateLimiter) size() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.state)
}
