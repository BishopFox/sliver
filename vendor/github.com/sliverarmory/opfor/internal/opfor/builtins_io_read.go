package opfor

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"sync/atomic"
)

// sleepIOWorker models IOObject's single Thread field. Implementations may
// retain richer completion state elsewhere; wait only needs thread liveness.
type sleepIOWorker interface {
	sleepIOWorkerDone() <-chan struct{}
}

// sleepIOWorkerFailure is implemented by workers whose asynchronous execution
// can encounter an OPFOR-authoritative error. IOObject's Java Thread field has
// no result channel, but these failures must not disappear merely because the
// work happened after read returned.
type sleepIOWorkerFailure interface {
	sleepIOWorkerError() error
}

func (task *forkTask) sleepIOWorkerDone() <-chan struct{} {
	if task == nil {
		return nil
	}
	return task.done
}

func (task *sleepSocketTask) sleepIOWorkerDone() <-chan struct{} {
	if task == nil {
		return nil
	}
	return task.done
}

type sleepReadTask struct {
	owner      *Script
	runtime    *Runtime
	handle     *sleepIOHandle
	callback   Callable
	invocation Invocation
	chunkSize  int32

	ctx            context.Context
	cancel         context.CancelCauseFunc
	releaseContext func()
	done           chan struct{}
	once           sync.Once
	// revoked is the logical callback-admission boundary. done closes only when
	// run has actually returned; the two states deliberately remain distinct for
	// an uninterruptible borrowed Read.
	revoked atomic.Bool
	// reading covers both the handle's serialized read lock and the source Read
	// itself. A task queued behind another uninterruptible read is equally unsafe
	// to join during teardown because sync.Mutex acquisition is not cancellable.
	reading atomic.Bool

	errMu       sync.RWMutex
	terminalErr error
}

func (task *sleepReadTask) sleepIOWorkerDone() <-chan struct{} {
	if task == nil {
		return nil
	}
	return task.done
}

func (task *sleepReadTask) sleepIOWorkerError() error {
	if task == nil {
		return nil
	}
	task.errMu.RLock()
	err := task.terminalErr
	task.errMu.RUnlock()
	return err
}

func sleepReadCallback(invocation Invocation, value Value) (Callable, error) {
	// Direct NativeFunc tests and importer calls have no Script lifetime to
	// retain. Runtime ownership still cancels and joins the worker on Close.
	if invocation.Script == 0 {
		callback, ok := value.Function()
		if !ok {
			return nil, ErrInvalidCallable
		}
		return callback, nil
	}
	// BridgeUtilities.getFunction accepts a SleepClosure, not an arbitrary
	// Function object returned by function("&native"). Keep direct importer
	// calls flexible above, while enforcing the source-language boundary for a
	// script-owned invocation.
	if callback, ok := value.Function(); ok {
		if _, closure := callback.(sleepSequenceClosure); !closure {
			return nil, ErrInvalidCallable
		}
	}

	callback, err := retainScriptLifetimeCallback(invocation, value)
	if err == nil {
		return callback, nil
	}
	if !errors.Is(err, ErrInvalidCallable) {
		return nil, err
	}
	// SleepUtils.getFunctionFromScalar accepts an exact named-function
	// reference such as "&handler" in addition to a closure scalar.
	name := value.String()
	if !strings.HasPrefix(name, "&") || invocation.Runtime == nil {
		return nil, ErrInvalidCallable
	}
	owner := invocation.Runtime.script(invocation.Script)
	if owner == nil || !owner.Active() {
		return nil, ErrScriptUnloaded
	}
	closure, ok := owner.resolveFunction(name).(*scriptClosure)
	if !ok || closure == nil {
		return nil, ErrInvalidCallable
	}
	return &scriptLifetimeCallback{owner: owner, callable: closure}, nil
}

func (state *ioBuiltinState) startSleepReadTask(
	ctx context.Context,
	invocation Invocation,
	handle *sleepIOHandle,
	callback Callable,
	chunkSize int32,
) error {
	if state == nil || state.runtime == nil {
		return errors.New("opfor: I/O runtime is unavailable")
	}
	owner := state.runtime.script(invocation.Script)
	if invocation.Script != 0 && (owner == nil || !owner.Active()) {
		return ErrScriptUnloaded
	}
	taskContext, releaseTaskContext, cancel := newAsynchronousExecutionTaskContext(ctx)
	task := &sleepReadTask{
		owner: owner, runtime: state.runtime, handle: handle, callback: callback,
		invocation: invocation, chunkSize: chunkSize,
		ctx: taskContext, cancel: cancel,
		releaseContext: releaseTaskContext,
		done:           make(chan struct{}),
	}
	if owner != nil {
		if !owner.registerReadTask(task) {
			cancel(context.Canceled)
			task.releaseContext()
			return ErrScriptUnloaded
		}
	} else if !state.runtime.registerReadTask(task) {
		cancel(context.Canceled)
		task.releaseContext()
		return ErrRuntimeClosed
	}

	// Match IOObject.setThread immediately before Thread.start. Replacing this
	// slot does not stop the previous worker.
	handle.setWorker(task)
	go task.run()
	return nil
}

func (task *sleepReadTask) run() {
	defer task.complete()
	if task.chunkSize > 0 {
		task.runBinary()
		return
	}
	task.runLines()
}

func (task *sleepReadTask) runLines() {
	for task.ctx.Err() == nil {
		task.reading.Store(true)
		line, present, err := task.handle.readLineContext(task.ctx)
		task.reading.Store(false)
		// IOObject.readLine closes the source and suppresses every read failure.
		// OPFOR resource/runtime boundaries remain authoritative even though the
		// same transport error would ordinarily die with CallbackReader's thread.
		if err != nil && task.authoritativeReadError(err) {
			task.setTerminalError(err)
			return
		}
		if !present || task.ctx.Err() != nil {
			return
		}
		if _, err := task.invoke(line); err != nil {
			// An exception escaping a line callback terminates only CallbackReader's
			// Java thread; it is not returned by read or wait.
			if task.authoritativeCallbackError(err) {
				task.setTerminalError(err)
			}
			return
		}
		// readLineContext retains an unterminated final line even when closing the
		// duplex writer fails. CallbackReader dispatches that line and only then
		// lets the ordinary close failure end its thread.
		if err != nil {
			return
		}
	}
}

func (task *sleepReadTask) runBinary() {
	for task.ctx.Err() == nil && !task.handle.isEOF() {
		task.reading.Store(true)
		data, readErr := task.handle.readFixedBytesContext(task.ctx, int(task.chunkSize))
		task.reading.Store(false)
		if readErr != nil {
			if task.canceled(readErr) {
				return
			}
			if task.authoritativeReadError(readErr) {
				task.setTerminalError(readErr)
				return
			}
			// DataInputStream.readUnsignedByte retains bytes accumulated before
			// EOF and CallbackReader dispatches that final partial chunk once.
			if len(data) != 0 {
				if _, callbackErr := task.invoke(BinaryString(data)); callbackErr != nil {
					// This callback occurs inside CallbackReader's catch block; an
					// exception escaping it skips close and flagError in the JVM too.
					if task.authoritativeCallbackError(callbackErr) {
						task.setTerminalError(callbackErr)
					}
					return
				}
			}
			_ = task.handle.close()
			task.flagError(readErr)
			return
		}

		if _, callbackErr := task.invoke(BinaryString(data)); callbackErr != nil {
			if task.authoritativeCallbackError(callbackErr) {
				task.setTerminalError(callbackErr)
				return
			}
			// CallbackReader's broad binary catch retries the current buffer before
			// closing the source and flagging the original exception.
			if len(data) != 0 {
				if _, retryErr := task.invoke(BinaryString(data)); retryErr != nil {
					if task.authoritativeCallbackError(retryErr) {
						task.setTerminalError(retryErr)
					}
					return
				}
			}
			_ = task.handle.close()
			task.flagError(callbackErr)
			return
		}
	}
}

func (task *sleepReadTask) invoke(data Value) (result Value, resultErr error) {
	if task == nil || task.callback == nil {
		return Null(), ErrInvalidCallable
	}
	// This fast-path keeps a returned read from approaching callback admission
	// after explicit task revocation. The Script/Runtime execution lease below
	// remains the authoritative race-free lifecycle boundary.
	if task.revoked.Load() {
		return Null(), context.Canceled
	}
	callbackContext := withCurrentFiber(task.ctx, nil)
	callbackContext = withExecutionMeter(callbackContext, task.runtime)
	if task.owner == nil {
		var release func() error
		var err error
		callbackContext, release, err = task.runtime.acquireRuntimeExecution(callbackContext)
		if err != nil {
			return Null(), err
		}
		defer func() { resultErr = joinExecutionError(resultErr, release) }()
	}
	defer func() {
		resultErr = preserveNativeBoundaryError(callbackContext, resultErr)
		resultErr = joinExecutionContextError(callbackContext, resultErr)
	}()
	if named, ok := task.callback.(interface {
		invokeNamed(context.Context, string, ...Value) (Value, error)
	}); ok {
		result, resultErr = named.invokeNamed(callbackContext, "&read", ObjectValue(task.handle), data)
		return result, resultErr
	}
	result, resultErr = task.callback.Invoke(callbackContext, ObjectValue(task.handle), data)
	return result, resultErr
}

// authoritativeCallbackError distinguishes OPFOR execution failures from the
// two exception classes which Sleep lets terminate CallbackReader's Java
// thread. Importer/provider failures are authoritative by default; limits are
// therefore never retried by the binary callback compatibility path.
func (task *sleepReadTask) authoritativeCallbackError(err error) bool {
	if err == nil || task.canceled(err) {
		return false
	}
	return !sleepReadCallbackThreadException(err)
}

func sleepReadCallbackThreadException(err error) bool {
	if err == nil {
		return true
	}
	switch err.(type) {
	case *scriptThrow, *uncaughtScriptWarning:
		return true
	case *RuntimeError, *nativeBoundaryError:
		return false
	}
	switch wrapped := err.(type) {
	case interface{ Unwrap() []error }:
		children := wrapped.Unwrap()
		if len(children) == 0 {
			return false
		}
		for _, child := range children {
			if !sleepReadCallbackThreadException(child) {
				return false
			}
		}
		return true
	case interface{ Unwrap() error }:
		return sleepReadCallbackThreadException(wrapped.Unwrap())
	default:
		return false
	}
}

// authoritativeReadError preserves failures produced by OPFOR's execution
// and native boundaries while retaining IOObject.readLine/CallbackReader's
// ordinary transport-error compatibility.
func (task *sleepReadTask) authoritativeReadError(err error) bool {
	if err == nil || task.canceled(err) {
		return false
	}
	if errors.Is(err, ErrResourceLimit) {
		return true
	}
	var runtimeErr *RuntimeError
	if errors.As(err, &runtimeErr) {
		return true
	}
	var boundaryErr *nativeBoundaryError
	return errors.As(err, &boundaryErr)
}

func (task *sleepReadTask) setTerminalError(err error) {
	if task == nil || err == nil {
		return
	}
	task.errMu.Lock()
	if task.terminalErr == nil {
		task.terminalErr = err
	}
	task.errMu.Unlock()
}

func (task *sleepReadTask) canceled(err error) bool {
	return task == nil || task.ctx.Err() != nil || errors.Is(err, context.Canceled) ||
		errors.Is(err, context.DeadlineExceeded) || errors.Is(err, ErrScriptUnloaded) ||
		errors.Is(err, ErrRuntimeClosed)
}

func (task *sleepReadTask) flagError(err error) {
	if task == nil || task.runtime == nil || err == nil || task.canceled(err) {
		return
	}
	switch {
	case errors.Is(err, io.EOF), errors.Is(err, io.ErrUnexpectedEOF):
		err = errors.New("java.io.EOFException")
	case !strings.HasPrefix(err.Error(), "java."):
		err = fmt.Errorf("java.io.IOException: %w", err)
	}
	_, _ = task.runtime.flagSourceError(task.invocation, err)
}

func (task *sleepReadTask) complete() {
	if task == nil {
		return
	}
	task.once.Do(func() {
		if task.releaseContext != nil {
			task.releaseContext()
		}
		task.unregisterOwner()
		close(task.done)
	})
}

// cancelAndClose revokes future callbacks and attempts to interrupt a blocked
// source read. Its result says whether join is safe: false means an arbitrary
// borrowed reader may still be executing Read and must remain host-owned.
func (task *sleepReadTask) cancelAndClose() bool {
	if task == nil {
		return true
	}
	task.revoked.Store(true)
	if task.cancel != nil {
		task.cancel(context.Canceled)
	}
	if task.releaseContext != nil {
		task.releaseContext()
	}
	if channelClosed(task.done) {
		return true
	}
	if !task.reading.Load() {
		// No source read or uncancellable read-lock acquisition is active, and
		// revocation prevents the loop from starting another one.
		return true
	}
	if process := task.handle.getProcess(); process != nil {
		_ = process.close()
		return true
	}
	if task.handle.hasOwnedReadCloser() {
		// Closing owned descriptors breaks a goroutine blocked in Read. Borrowed
		// console streams remain host-owned and are never physically closed.
		task.handle.abortOwnedTransport()
		return true
	}
	if task.handle.hasContextReader() || !task.handle.hasOpenReader() {
		return true
	}
	// Pure Go has no operation which can interrupt an arbitrary borrowed
	// io.Reader. Keep done truthful and let the host decide when its Read may
	// return; lifecycle teardown reports the incomplete quiescence instead of
	// hanging or claiming that this goroutine exited.
	return false
}

func (h *sleepIOHandle) hasOwnedReadCloser() bool {
	if h == nil {
		return false
	}
	h.mu.Lock()
	owned := h.ownRead && h.readCloser != nil
	h.mu.Unlock()
	return owned
}

func (h *sleepIOHandle) hasContextReader() bool {
	if h == nil {
		return false
	}
	h.mu.Lock()
	contextual := h.contextRead != nil
	h.mu.Unlock()
	return contextual
}

func (h *sleepIOHandle) hasOpenReader() bool {
	if h == nil {
		return false
	}
	h.mu.Lock()
	open := h.reader != nil
	h.mu.Unlock()
	return open
}

func revokeSleepReadTasks(tasks []*sleepReadTask) (joinable []*sleepReadTask, incomplete bool) {
	joinable = make([]*sleepReadTask, 0, len(tasks))
	for _, task := range tasks {
		if task.cancelAndClose() {
			joinable = append(joinable, task)
			continue
		}
		incomplete = true
	}
	return joinable, incomplete
}

func (task *sleepReadTask) join(ctx context.Context) error {
	if task == nil {
		return nil
	}
	select {
	case <-task.done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (script *Script) registerReadTask(task *sleepReadTask) bool {
	if script == nil || task == nil {
		return false
	}
	script.mu.Lock()
	defer script.mu.Unlock()
	if !script.active {
		return false
	}
	if script.readTasks == nil {
		script.readTasks = make(map[*sleepReadTask]struct{})
	}
	script.readTasks[task] = struct{}{}
	return true
}

func (runtime *Runtime) registerReadTask(task *sleepReadTask) bool {
	if runtime == nil || task == nil {
		return false
	}
	runtime.mu.Lock()
	defer runtime.mu.Unlock()
	if runtime.closing || runtime.closed {
		return false
	}
	if runtime.readTasks == nil {
		runtime.readTasks = make(map[*sleepReadTask]struct{})
	}
	runtime.readTasks[task] = struct{}{}
	return true
}

func (task *sleepReadTask) unregisterOwner() {
	if task == nil {
		return
	}
	if task.owner != nil {
		task.owner.mu.Lock()
		delete(task.owner.readTasks, task)
		task.owner.mu.Unlock()
		return
	}
	if task.runtime != nil {
		task.runtime.mu.Lock()
		delete(task.runtime.readTasks, task)
		task.runtime.mu.Unlock()
	}
}

// readFixedBytesContext mirrors repeated DataInputStream.readUnsignedByte
// calls: it returns every byte accumulated before the first exception.
func (h *sleepIOHandle) readFixedBytesContext(ctx context.Context, count int) ([]byte, error) {
	if h == nil {
		return nil, io.ErrClosedPipe
	}
	h.readMu.Lock()
	defer h.readMu.Unlock()
	capacity := count
	if capacity > sleepIOReadBufferSize {
		capacity = sleepIOReadBufferSize
	}
	data := make([]byte, 0, capacity)
	buffer := make([]byte, capacity)
	for len(data) < count {
		chunk := buffer
		if remaining := count - len(data); remaining < len(chunk) {
			chunk = chunk[:remaining]
		}
		amount, err := h.readBinaryLockedContext(ctx, chunk)
		data = append(data, chunk[:amount]...)
		if err != nil {
			return data, err
		}
		if amount == 0 {
			return data, io.ErrNoProgress
		}
	}
	return data, nil
}
