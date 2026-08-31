//go:build windows

package opfor

import (
	"context"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

const (
	sleepSocketAcceptWaiting uint32 = iota
	sleepSocketAcceptDelivered
	sleepSocketAcceptCanceled
)

type sleepSocketAcceptResult struct {
	conn net.Conn
	err  error
}

type sleepSocketAcceptWaiter struct {
	state  atomic.Uint32
	result chan sleepSocketAcceptResult
}

// brokeredSleepSocketAcceptor is the Windows counterpart to Unix descriptor
// duplication. TCPListener.File is unsupported on Windows, so one goroutine
// owns AcceptTCP and assigns connections to FIFO waiters. Each waiter retains
// an independent context and timer. A canceled waiter momentarily advances the
// listener deadline only to wake the broker; the broker clears that deadline
// before serving the remaining waiters.
type brokeredSleepSocketAcceptor struct {
	listener *net.TCPListener

	mu        sync.Mutex
	closed    bool
	accepting bool
	waiters   []*sleepSocketAcceptWaiter
}

func newSleepSocketAcceptor(listener *net.TCPListener) sleepSocketAcceptor {
	return &brokeredSleepSocketAcceptor{listener: listener}
}

func (listener *brokeredSleepSocketAcceptor) accept(ctx context.Context, timeout int32) (net.Conn, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	waiter := &sleepSocketAcceptWaiter{result: make(chan sleepSocketAcceptResult, 1)}

	listener.mu.Lock()
	if listener.closed {
		listener.mu.Unlock()
		return nil, net.ErrClosed
	}
	listener.waiters = append(listener.waiters, waiter)
	if !listener.accepting {
		listener.accepting = true
		go listener.acceptLoop()
	}
	listener.mu.Unlock()

	var timer *time.Timer
	var timeoutChannel <-chan time.Time
	if timeout > 0 {
		timer = time.NewTimer(time.Duration(timeout) * time.Millisecond)
		timeoutChannel = timer.C
		defer timer.Stop()
	}

	select {
	case result := <-waiter.result:
		return sleepSocketAcceptResultForContext(ctx, result)
	case <-ctx.Done():
		if waiter.state.CompareAndSwap(sleepSocketAcceptWaiting, sleepSocketAcceptCanceled) {
			listener.wakeAcceptLoop()
			return nil, ctx.Err()
		}
		result := <-waiter.result
		if result.conn != nil {
			_ = result.conn.Close()
		}
		return nil, ctx.Err()
	case <-timeoutChannel:
		if waiter.state.CompareAndSwap(sleepSocketAcceptWaiting, sleepSocketAcceptCanceled) {
			listener.wakeAcceptLoop()
			return nil, sleepSocketAcceptTimeoutError{}
		}
		result := <-waiter.result
		return sleepSocketAcceptResultForContext(ctx, result)
	}
}

func sleepSocketAcceptResultForContext(ctx context.Context, result sleepSocketAcceptResult) (net.Conn, error) {
	if err := ctx.Err(); err != nil {
		if result.conn != nil {
			_ = result.conn.Close()
		}
		return nil, err
	}
	return result.conn, result.err
}

func (listener *brokeredSleepSocketAcceptor) wakeAcceptLoop() {
	listener.mu.Lock()
	defer listener.mu.Unlock()
	if listener.closed || !listener.accepting {
		return
	}
	// acceptLoop resets the deadline while holding the same mutex immediately
	// before AcceptTCP, so a concurrent wake cannot be lost between those two
	// operations.
	_ = listener.listener.SetDeadline(time.Now())
}

func (listener *brokeredSleepSocketAcceptor) acceptLoop() {
	for {
		listener.mu.Lock()
		listener.pruneCanceledWaitersLocked()
		if listener.closed || len(listener.waiters) == 0 {
			listener.accepting = false
			listener.mu.Unlock()
			return
		}
		// Canceling one waiter wakes the shared AcceptTCP with an expired
		// deadline. Clear it under the broker mutex so other waiters continue.
		_ = listener.listener.SetDeadline(time.Time{})
		listener.mu.Unlock()

		conn, err := listener.listener.AcceptTCP()
		if err != nil {
			listener.mu.Lock()
			if listener.closed {
				listener.accepting = false
				listener.mu.Unlock()
				return
			}
			if networkError, ok := err.(net.Error); ok && networkError.Timeout() {
				listener.mu.Unlock()
				continue
			}
			waiter := listener.takeWaiterLocked()
			if waiter == nil {
				listener.accepting = false
			}
			listener.mu.Unlock()
			if waiter == nil {
				return
			}
			waiter.result <- sleepSocketAcceptResult{err: err}
			continue
		}

		listener.mu.Lock()
		if listener.closed {
			listener.accepting = false
			listener.mu.Unlock()
			_ = conn.Close()
			return
		}
		waiter := listener.takeWaiterLocked()
		if waiter == nil {
			listener.accepting = false
		}
		listener.mu.Unlock()
		if waiter == nil {
			// The connection raced with cancellation of the last waiter. It does
			// not belong to a future accept invocation.
			_ = conn.Close()
			return
		}
		waiter.result <- sleepSocketAcceptResult{conn: conn}
	}
}

func (listener *brokeredSleepSocketAcceptor) pruneCanceledWaitersLocked() {
	kept := listener.waiters[:0]
	for _, waiter := range listener.waiters {
		if waiter.state.Load() == sleepSocketAcceptWaiting {
			kept = append(kept, waiter)
		}
	}
	for index := len(kept); index < len(listener.waiters); index++ {
		listener.waiters[index] = nil
	}
	listener.waiters = kept
}

func (listener *brokeredSleepSocketAcceptor) takeWaiterLocked() *sleepSocketAcceptWaiter {
	for len(listener.waiters) > 0 {
		waiter := listener.waiters[0]
		copy(listener.waiters, listener.waiters[1:])
		listener.waiters[len(listener.waiters)-1] = nil
		listener.waiters = listener.waiters[:len(listener.waiters)-1]
		if waiter.state.CompareAndSwap(sleepSocketAcceptWaiting, sleepSocketAcceptDelivered) {
			return waiter
		}
	}
	return nil
}

func (listener *brokeredSleepSocketAcceptor) close() error {
	if listener == nil {
		return nil
	}
	listener.mu.Lock()
	if listener.closed {
		listener.mu.Unlock()
		return nil
	}
	listener.closed = true
	waiters := listener.waiters
	listener.waiters = nil
	closeErr := listener.listener.Close()
	listener.mu.Unlock()

	for _, waiter := range waiters {
		if waiter.state.CompareAndSwap(sleepSocketAcceptWaiting, sleepSocketAcceptDelivered) {
			waiter.result <- sleepSocketAcceptResult{err: net.ErrClosed}
		}
	}
	return closeErr
}

type sleepSocketAcceptTimeoutError struct{}

func (sleepSocketAcceptTimeoutError) Error() string   { return "i/o timeout" }
func (sleepSocketAcceptTimeoutError) Timeout() bool   { return true }
func (sleepSocketAcceptTimeoutError) Temporary() bool { return true }
