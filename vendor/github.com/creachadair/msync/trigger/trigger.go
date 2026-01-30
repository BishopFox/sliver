// Package trigger implements channel-based condition variable.
package trigger

import "sync"

// A Cond is a condition shared by multiple goroutines.  The [Cond.Ready]
// method returns a channel that is closed when the condition is activated.
//
// When a condition is first created it is inactive. While inactive, reads on
// the ready channel will block.  The condition remains inactive until
// [Cond.Set] or [Cond.Signal] is called, either of which causes the current
// ready channel to be closed (and thus deliver a zero value). Once a condition
// has been activated, it remains active until it is reset. Use [Cond.Reset] to
// make it inactive again.
//
// The [Cond.Signal] method activates and then immediately resets the
// condition, acting as Set and Reset done in a single step.
//
// A zero Cond is ready for use, and is inactive, but must not be copied after
// any of its methods have been called.
type Cond struct {
	μ      sync.Mutex
	ch     chan struct{}
	closed bool

	// The signal channel is lazily initialized by the first waiter.
}

// New constructs a new inactive [Cond].
func New() *Cond { return new(Cond) }

// Signal activates and immediately resets the condition.  If the condition was
// already active, this is equivalent to [Cond.Reset].
func (c *Cond) Signal() {
	c.μ.Lock()
	defer c.μ.Unlock()

	if c.ch != nil && !c.closed {
		close(c.ch) // wake any pending waiters
	}
	c.ch = nil
	c.closed = false
}

// Set activates the condition. If it was already active, Set has no effect.
func (c *Cond) Set() {
	c.μ.Lock()
	defer c.μ.Unlock()

	if c.ch == nil {
		c.ch = make(chan struct{})
	}
	if !c.closed {
		close(c.ch)
		c.closed = true
	}
}

// Reset resets the condition. If it was already inactive, Reset has no effect.
func (c *Cond) Reset() {
	c.μ.Lock()
	defer c.μ.Unlock()

	if c.closed {
		c.ch = nil
		c.closed = false
	}
}

// Ready returns a channel that is closed when c is activated.  If c is active
// when Ready is called, the returned channel will already be closed.
func (c *Cond) Ready() <-chan struct{} {
	c.μ.Lock()
	defer c.μ.Unlock()

	if c.ch == nil {
		c.ch = make(chan struct{})
		c.closed = false
	}
	return c.ch
}
