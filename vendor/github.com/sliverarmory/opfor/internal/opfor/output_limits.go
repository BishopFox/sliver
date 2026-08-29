package opfor

import (
	"context"
	"errors"
	"io"
	"sync"
)

// runtimeOutputAccountWriter identifies a writer that already charges bytes to
// one runtime-family output account. Wrappers which preserve that property
// expose the same account so console routing and stream decorators do not
// charge one byte more than once.
type runtimeOutputAccountWriter interface {
	runtimeOutputAccount() *runtimeResourceAccount
}

// runtimeOutputWriter enforces the family-wide output quota immediately before
// bytes cross an OPFOR-owned output or buffering boundary. A write which spans
// the remaining budget commits the allowed prefix and returns a LimitError for
// the rejected suffix. Reservations are monotonic even when the downstream
// writer subsequently reports a short write or error.
type runtimeOutputWriter struct {
	account *runtimeResourceAccount
	writer  io.Writer

	errMu sync.Mutex
	err   *LimitError
}

func newRuntimeOutputWriter(account *runtimeResourceAccount, writer io.Writer) *runtimeOutputWriter {
	return &runtimeOutputWriter{account: account, writer: writer}
}

func runtimeOutputWriterFor(account *runtimeResourceAccount, writer io.Writer) io.Writer {
	if writer == nil || runtimeOutputAccountOf(writer) == account {
		return writer
	}
	return newRuntimeOutputWriter(account, writer)
}

func runtimeOutputAccountFor(ctx context.Context, runtime *Runtime) *runtimeResourceAccount {
	if ctx != nil {
		if account, _ := ctx.Value(runtimeResourceAccountKey{}).(*runtimeResourceAccount); account != nil {
			return account
		}
	}
	if runtime == nil {
		return nil
	}
	return runtime.resources
}

func runtimeOutputAccountOf(writer io.Writer) *runtimeResourceAccount {
	if accounted, ok := writer.(runtimeOutputAccountWriter); ok && accounted != nil {
		return accounted.runtimeOutputAccount()
	}
	return nil
}

func (writer *runtimeOutputWriter) runtimeOutputAccount() *runtimeResourceAccount {
	if writer == nil {
		return nil
	}
	return writer.account
}

func (writer *runtimeOutputWriter) Write(data []byte) (int, error) {
	if writer == nil || writer.writer == nil {
		return 0, io.ErrClosedPipe
	}
	written, err := writeRuntimeOutput(writer.account, writer.writer, data)
	writer.recordLimitError()
	return written, err
}

func writeRuntimeOutput(account *runtimeResourceAccount, destination io.Writer, data []byte) (int, error) {
	if destination == nil {
		return 0, io.ErrClosedPipe
	}
	// A router or decorated handle may already own this exact account. Let the
	// downstream boundary make the one authoritative reservation.
	if account != nil && runtimeOutputAccountOf(destination) == account {
		return destination.Write(data)
	}

	allowed, limitErr := reserveRuntimeOutputPrefix(account, len(data))
	// Publish a rejected suffix before entering an importer-controlled sink. The
	// sink may block or panic, but later runtime-family work must already observe
	// the non-consuming fatal condition created by the successful reservation.
	recordRuntimeOutputLimit(account, limitErr)
	if allowed == 0 && limitErr != nil {
		return 0, limitErr
	}
	written, writeErr := destination.Write(data[:allowed])
	if limitErr == nil {
		return written, writeErr
	}
	// Put the output violation first so an unrelated typed downstream error
	// cannot mask it from errors.As callers which need the exact resource.
	return written, errors.Join(limitErr, writeErr)
}

func (writer *runtimeOutputWriter) Flush() error {
	if writer == nil || writer.writer == nil {
		return nil
	}
	if flusher, ok := writer.writer.(interface{ Flush() error }); ok {
		return flusher.Flush()
	}
	return nil
}

func (writer *runtimeOutputWriter) LimitError() error {
	if writer == nil {
		return nil
	}
	writer.errMu.Lock()
	defer writer.errMu.Unlock()
	if writer.err == nil {
		return nil
	}
	return cloneLimitError(writer.err)
}

func (runtime *Runtime) outputLimitError() error {
	if runtime == nil || runtime.resources == nil {
		return nil
	}
	limitErr := runtime.resources.outputLimitError()
	if limitErr == nil {
		return nil
	}
	return limitErr
}

// recordRuntimeOutputLimit makes a rejected output attempt a non-consuming,
// family-wide fatal condition. Internal diagnostics historically ignore their
// writer error, and asynchronous producers may fail after their initiating
// call has returned. Retaining the first typed failure on the shared account
// ensures either case is observed deterministically at execution boundaries;
// one concurrent execution cannot steal the failure from another.
func recordRuntimeOutputLimit(account *runtimeResourceAccount, limitErr *LimitError) {
	if account == nil || limitErr == nil || limitErr.Resource != resourceOutputBytes {
		return
	}
	account.outputViolation.CompareAndSwap(nil, &LimitError{
		Resource: limitErr.Resource,
		Limit:    limitErr.Limit,
	})
}

func (writer *runtimeOutputWriter) recordLimitError() {
	if writer == nil || writer.account == nil {
		return
	}
	limitErr := writer.account.outputLimitError()
	if limitErr == nil {
		return
	}
	writer.errMu.Lock()
	if writer.err == nil {
		writer.err = limitErr
	}
	writer.errMu.Unlock()
}

func reserveRuntimeOutputPrefix(account *runtimeResourceAccount, requested int) (int, *LimitError) {
	if requested <= 0 || account == nil || account.output.limit == 0 {
		return requested, nil
	}
	counter := &account.output
	amount := uint64(requested)
	for {
		used := counter.used.Load()
		if used >= counter.limit {
			return 0, &LimitError{Resource: resourceOutputBytes, Limit: counter.limit}
		}
		remaining := counter.limit - used
		reserved := amount
		if reserved > remaining {
			reserved = remaining
		}
		if !counter.used.CompareAndSwap(used, used+reserved) {
			continue
		}
		if reserved < amount {
			return int(reserved), &LimitError{Resource: resourceOutputBytes, Limit: counter.limit}
		}
		return requested, nil
	}
}

// synchronizedWriter retains the marker through Runtime's shared stdout/stderr
// serialization layer.
func (writer synchronizedWriter) runtimeOutputAccount() *runtimeResourceAccount {
	return runtimeOutputAccountOf(writer.writer)
}
