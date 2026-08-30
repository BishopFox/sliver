package opfor

import (
	"context"
	"fmt"
	"strings"
	"sync"
)

// sleepSemaphore is the pure-Go counterpart of sleep.bridges.Semaphore. The
// notification channel is replaced whenever the count changes from a release,
// giving every blocked waiter the notifyAll behavior used by Sleep 2.1 while
// still allowing an OPFOR execution context to cancel a blocked acquire.
type sleepSemaphore struct {
	mu     sync.Mutex
	count  int64
	notify chan struct{}
}

func newSleepSemaphore(initial int64) *sleepSemaphore {
	return &sleepSemaphore{count: initial, notify: make(chan struct{})}
}

func (semaphore *sleepSemaphore) String() string {
	if semaphore == nil {
		return "[Semaphore: 0]"
	}
	semaphore.mu.Lock()
	count := semaphore.count
	semaphore.mu.Unlock()
	return fmt.Sprintf("[Semaphore: %d]", count)
}

func (semaphore *sleepSemaphore) getCount() int64 {
	if semaphore == nil {
		return 0
	}
	semaphore.mu.Lock()
	count := semaphore.count
	semaphore.mu.Unlock()
	return count
}

func (semaphore *sleepSemaphore) acquire(ctx context.Context) error {
	for {
		semaphore.mu.Lock()
		if semaphore.count > 0 {
			semaphore.count--
			semaphore.mu.Unlock()
			return nil
		}
		notify := semaphore.notify
		semaphore.mu.Unlock()

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-notify:
		}
	}
}

func (semaphore *sleepSemaphore) release() {
	semaphore.mu.Lock()
	semaphore.count++
	close(semaphore.notify)
	semaphore.notify = make(chan struct{})
	semaphore.mu.Unlock()
}

// invoke exposes the complete public surface of Sleep's own
// sleep.bridges.Semaphore class. This is part of the Sleep bridge contract,
// not a general JVM reflection implementation.
func (semaphore *sleepSemaphore) invoke(ctx context.Context, invocation ObjectInvocation) (Value, bool, error) {
	if invocation.Op != ObjectInvoke || len(invocation.Arguments) != 0 {
		return Null(), false, nil
	}
	switch invocation.Message {
	case "P":
		if err := semaphore.acquire(ctx); err != nil {
			return Null(), true, err
		}
		return Null(), true, nil
	case "V":
		semaphore.release()
		return Null(), true, nil
	case "getCount":
		return Long(semaphore.getCount()), true, nil
	case "toString":
		return String(semaphore.String()), true, nil
	default:
		return Null(), false, nil
	}
}

func (r *Runtime) concurrencyFunctions() map[string]NativeFunc {
	return map[string]NativeFunc{
		"semaphore": newSemaphore,
		"acquire":   acquireSemaphore,
		"release":   releaseSemaphore,
	}
}

func newSemaphore(_ context.Context, invocation Invocation) (Value, error) {
	initial := int64(1)
	if len(invocation.Arguments) != 0 {
		// BasicUtilities uses BridgeUtilities.getInt, so an explicit initial
		// value has Sleep's signed 32-bit coercion even though Semaphore stores
		// its counter as a Java long.
		initial = int64(sleepInt32(invocation.Arg(0)))
	}
	return ObjectValue(newSleepSemaphore(initial)), nil
}

func acquireSemaphore(ctx context.Context, invocation Invocation) (Value, error) {
	semaphore, err := semaphoreArgument(invocation)
	if err != nil {
		return Null(), err
	}
	if err := semaphore.acquire(ctx); err != nil {
		return Null(), err
	}
	return Null(), nil
}

func releaseSemaphore(_ context.Context, invocation Invocation) (Value, error) {
	semaphore, err := semaphoreArgument(invocation)
	if err != nil {
		return Null(), err
	}
	semaphore.release()
	return Null(), nil
}

func semaphoreArgument(invocation Invocation) (*sleepSemaphore, error) {
	value := invocation.Arg(0)
	// BridgeUtilities.getObject returns null for both an absent argument and a
	// $null scalar. SyncPrimitives then dereferences that null Semaphore, which
	// Sleep's Block reports using its stable null-value warning.
	if value.IsNull() {
		return nil, sleepBridgeNullValue()
	}
	object, ok := value.Object()
	if !ok {
		return nil, sleepSemaphoreCastWarning(value)
	}
	semaphore, ok := object.(*sleepSemaphore)
	if !ok || semaphore == nil {
		return nil, sleepSemaphoreCastWarning(value)
	}
	return semaphore, nil
}

func sleepSemaphoreCastWarning(value Value) error {
	actual, ok := portableObjectClass(value)
	if !ok || actual == "" {
		actual = "java.lang.Object"
	}
	target := "sleep.bridges.Semaphore"
	message := fmt.Sprintf("attempted an invalid cast: class %s cannot be cast to class %s", actual, target)
	switch {
	case strings.HasPrefix(actual, "java."):
		message += fmt.Sprintf(" (%s is in module java.base of loader 'bootstrap'; %s is in unnamed module of loader 'app')", actual, target)
	case strings.HasPrefix(actual, "sleep."):
		message += fmt.Sprintf(" (%s and %s are in unnamed module of loader 'app')", actual, target)
	}
	return sleepBridgeIllegalArgument(message)
}
