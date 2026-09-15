package backoff

import (
	"math/rand"
	"time"
)

// This one was ripped from https://github.com/jpillora/backoff/blob/master/backoff.go

// Backoff is a time.Duration counter. It starts at Initial. After every
// call to Duration() it is doubled. It is capped at Max. It returns to
// Initial on every call to Reset(). Used in conjunction with the time
// package.
type Backoff struct {
	attempts int
	// Initial value to scale out
	Initial time.Duration
	// Jitter value randomizes an additional delay between 0 and Jitter
	Jitter time.Duration
	// Max maximum values of the backoff
	Max time.Duration
}

// Duration returns the current value of the counter, then doubles it for the
// next call. Optional jitter is added to the returned value, and the result is
// capped at Max.
func (b *Backoff) Duration() (dur time.Duration) {
	// Zero-values are nonsensical, so we use
	// them to apply defaults
	if b.Max == 0 {
		b.Max = 10 * time.Second
	}

	if b.Initial == 0 {
		b.Initial = 100 * time.Millisecond
	}

	// calculate this duration
	if dur = time.Duration(1 << uint(b.attempts)); dur > 0 {
		dur *= b.Initial
	} else {
		dur = b.Max
	}

	if b.Jitter > 0 {
		dur += time.Duration(rand.Intn(int(b.Jitter)))
	}

	// bump attempts count
	b.attempts++

	return dur
}

// Reset sets the current value of the counter back to Initial
func (b *Backoff) Reset() {
	b.attempts = 0
}

// Attempts returns the number of attempts that we had done so far
func (b *Backoff) Attempts() int {
	return b.attempts
}
