package opfor

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
)

// getAggressorClientType identifies OPFOR's stock non-graphical client. A
// Cobalt-aware importer may replace this function through WithFunction when it
// exposes another client surface.
func (*Runtime) getAggressorClientType(_ context.Context, invocation Invocation) (Value, error) {
	if err := requireExactAggressorClientArguments(invocation, 0); err != nil {
		return Null(), err
	}
	return String("headless"), nil
}

// dispatchEvent implements Aggressor's dispatch_event(callback) scheduling
// boundary. Invocation.Callback makes the callable safe to retain after this
// native call returns while keeping it tied to the owning Script's lifetime.
func (r *Runtime) dispatchEvent(ctx context.Context, invocation Invocation) (Value, error) {
	if err := requireExactAggressorClientArguments(invocation, 1); err != nil {
		return Null(), err
	}
	if r == nil {
		return Null(), errors.New("opfor: runtime is nil")
	}
	if isNilInterface(r.eventDispatcher) {
		return Null(), errors.New("opfor: Aggressor event dispatcher is nil")
	}
	if err := executionContextError(ctx); err != nil {
		return Null(), err
	}
	if ctx == nil {
		ctx = context.Background()
	}

	callback, err := invocation.Callback(0)
	if err != nil {
		if errors.Is(err, ErrInvalidCallable) {
			return Null(), fmt.Errorf("&%s: argument 1 is not callable: %w", builtinName(invocation.Name), err)
		}
		return Null(), err
	}
	dispatchContext := ctx
	dispatchCallback := callback
	var importerCallback *importerDispatchEventCallback
	_, synchronous := r.eventDispatcher.(synchronousAggressorEventDispatcherMarker)
	if !synchronous {
		var meter *executionMeter
		dispatchContext, meter = captureCallbackSchedulingContext(ctx)
		importerCallback = &importerDispatchEventCallback{
			callback: callback,
			meter:    meter,
		}
		importerCallback.dispatchActive.Store(true)
		defer importerCallback.dispatchActive.Store(false)
		dispatchCallback = importerCallback
	}
	if err := dispatchContext.Err(); err != nil {
		return Null(), err
	}
	dispatchErr := r.eventDispatcher.DispatchAggressorEvent(dispatchContext, dispatchCallback)
	if importerCallback != nil {
		// Invocation snapshots taken before this store keep the meter pointer they
		// already selected. Later retained invocations see no originating meter and
		// let scriptClosure install a fresh top-level budget.
		importerCallback.dispatchActive.Store(false)
	}
	if dispatchErr != nil {
		if !synchronous {
			dispatchErr = preserveNativeBoundaryError(ctx, dispatchErr)
		}
		return Null(), dispatchErr
	}
	if err := dispatchContext.Err(); err != nil {
		return Null(), err
	}
	return Null(), nil
}

// synchronousAggressorEventDispatcherMarker is private so only OPFOR's stock,
// non-retaining dispatcher can opt into the live evaluator context. Importer
// dispatchers always cross the detached boundary below.
type synchronousAggressorEventDispatcherMarker interface {
	synchronousAggressorEventDispatcher()
}

// importerDispatchEventCallback binds context selection to each callback
// invocation. It never exposes a fiber or lifecycle token to importer code.
// An invocation which begins before DispatchAggressorEvent returns gets a
// stable pointer to the originating instruction meter for its entire call;
// later retained invocations start with no meter and receive a fresh one from
// scriptClosure.Invoke.
type importerDispatchEventCallback struct {
	callback Callable
	meter    *executionMeter

	dispatchActive atomic.Bool
}

func (callback *importerDispatchEventCallback) Invoke(ctx context.Context, arguments ...Value) (Value, error) {
	if callback == nil || callback.callback == nil {
		return Null(), ErrInvalidCallable
	}
	var meter *executionMeter
	if callback.dispatchActive.Load() {
		meter = callback.meter
	}
	return callback.callback.Invoke(
		snapshotCallbackInvocationContext(ctx, meter),
		arguments...,
	)
}

func requireExactAggressorClientArguments(invocation Invocation, count int) error {
	if len(invocation.Arguments) == count {
		return nil
	}
	return fmt.Errorf("&%s: expected exactly %d argument(s), received %d",
		builtinName(invocation.Name), count, len(invocation.Arguments))
}
