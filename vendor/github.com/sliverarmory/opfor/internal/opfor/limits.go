package opfor

import (
	"context"
	"sync/atomic"
)

type executionMeterKey struct{}
type runtimeResourceAccountKey struct{}

const (
	// LimitResourceInstruction identifies the per-execution VM instruction
	// counter in LimitError.Resource.
	LimitResourceInstruction = "instruction"
	// LimitResourceCollectionEntries identifies the runtime-family collection
	// entry counter in LimitError.Resource.
	LimitResourceCollectionEntries = "collection entries"
	// LimitResourceOutputBytes identifies the runtime-family output byte counter
	// in LimitError.Resource.
	LimitResourceOutputBytes = "output bytes"
	// LimitResourceInputBytes identifies the runtime-family input byte counter
	// in LimitError.Resource.
	LimitResourceInputBytes = "input bytes"
	// LimitResourceDecompressedBytes identifies the runtime-family decompressed
	// byte counter in LimitError.Resource.
	LimitResourceDecompressedBytes = "decompressed bytes"
	// LimitResourceSourceBytes identifies the runtime-family admitted source
	// byte counter in LimitError.Resource.
	LimitResourceSourceBytes = "source bytes"

	resourceInstruction       = LimitResourceInstruction
	resourceCollectionEntries = LimitResourceCollectionEntries
	resourceOutputBytes       = LimitResourceOutputBytes
	resourceInputBytes        = LimitResourceInputBytes
	resourceDecompressedBytes = LimitResourceDecompressedBytes
	resourceSourceBytes       = LimitResourceSourceBytes
)

// Limits configures resource bounds for one Runtime family. Zero disables an
// individual bound. Instruction accounting resets for each top-level execution
// or callback; the other counters are monotonic and shared by root scripts,
// forks, and source-backed ScriptLoader child runtimes for the family's
// lifetime. Monotonic accounting deliberately does not refund on unload or
// close because importers may still retain values and handles.
type Limits struct {
	MaxInstructionsPerExecution    uint64
	MaxCollectionEntriesPerRuntime uint64
	MaxOutputBytesPerRuntime       uint64
	// MaxInputBytesPerRuntime counts bytes delivered through Runtime-owned
	// Sleep I/O reads, including bytes delivered again after mark/reset. It does
	// not count unique transport ingress or bytes merely prefetched into the
	// handle's fixed 8 KiB bufio buffer. Large read/consume/look-ahead result
	// allocations stream through fixed chunks when this limit is enabled; this
	// is an admission bound, not a whole-Go-heap bound.
	MaxInputBytesPerRuntime        uint64
	MaxDecompressedBytesPerRuntime uint64
	MaxSourceBytesPerRuntime       uint64
}

type executionMeter struct {
	limit uint64
	used  atomic.Uint64
}

type runtimeResourceCounter struct {
	limit uint64
	used  atomic.Uint64
}

type runtimeResourceAccount struct {
	limits          Limits
	collections     runtimeResourceCounter
	output          runtimeResourceCounter
	outputViolation atomic.Pointer[LimitError]
	input           runtimeResourceCounter
	decompressed    runtimeResourceCounter
	source          runtimeResourceCounter
}

func cloneLimitError(limitErr *LimitError) *LimitError {
	if limitErr == nil {
		return nil
	}
	return &LimitError{Resource: limitErr.Resource, Limit: limitErr.Limit}
}

// outputLimitError returns a value copy of the immutable internal latch. A
// LimitError has exported fields for ordinary error inspection, so the atomic
// pointee itself must never escape to importer code.
func (account *runtimeResourceAccount) outputLimitError() *LimitError {
	if account == nil {
		return nil
	}
	return cloneLimitError(account.outputViolation.Load())
}

func newRuntimeResourceAccount(limits Limits) *runtimeResourceAccount {
	return &runtimeResourceAccount{
		limits: limits,
		collections: runtimeResourceCounter{
			limit: limits.MaxCollectionEntriesPerRuntime,
		},
		output: runtimeResourceCounter{
			limit: limits.MaxOutputBytesPerRuntime,
		},
		input: runtimeResourceCounter{
			limit: limits.MaxInputBytesPerRuntime,
		},
		decompressed: runtimeResourceCounter{
			limit: limits.MaxDecompressedBytesPerRuntime,
		},
		source: runtimeResourceCounter{
			limit: limits.MaxSourceBytesPerRuntime,
		},
	}
}

func (account *runtimeResourceAccount) counter(resource string) *runtimeResourceCounter {
	if account == nil {
		return nil
	}
	switch resource {
	case resourceCollectionEntries:
		return &account.collections
	case resourceOutputBytes:
		return &account.output
	case resourceInputBytes:
		return &account.input
	case resourceDecompressedBytes:
		return &account.decompressed
	case resourceSourceBytes:
		return &account.source
	default:
		return nil
	}
}

func (account *runtimeResourceAccount) reserve(resource string, amount uint64) error {
	counter := account.counter(resource)
	if counter == nil || counter.limit == 0 || amount == 0 {
		return nil
	}
	for {
		used := counter.used.Load()
		if used > counter.limit || amount > counter.limit-used {
			return &LimitError{Resource: resource, Limit: counter.limit}
		}
		if counter.used.CompareAndSwap(used, used+amount) {
			return nil
		}
	}
}

func (account *runtimeResourceAccount) used(resource string) uint64 {
	counter := account.counter(resource)
	if counter == nil {
		return 0
	}
	return counter.used.Load()
}

func withExecutionMeter(ctx context.Context, runtime *Runtime) context.Context {
	if runtime == nil {
		return ctx
	}
	if current, _ := ctx.Value(runtimeResourceAccountKey{}).(*runtimeResourceAccount); current != runtime.resources {
		ctx = context.WithValue(ctx, runtimeResourceAccountKey{}, runtime.resources)
	}
	if runtime.maxInstructions == 0 || ctx.Value(executionMeterKey{}) != nil {
		return ctx
	}
	return context.WithValue(ctx, executionMeterKey{}, &executionMeter{limit: runtime.maxInstructions})
}

func consumeInstruction(ctx context.Context) error {
	meter, _ := ctx.Value(executionMeterKey{}).(*executionMeter)
	account, _ := ctx.Value(runtimeResourceAccountKey{}).(*runtimeResourceAccount)
	if account == nil || account.output.limit == 0 {
		account = nil
	}
	return consumeInstructionLimits(meter, account)
}

// vmExecutionLimits resolves execution-local accounting once before entering
// the VM loop. Unlimited output accounts are returned as nil so the default
// loop avoids an atomic output-latch load at every instruction.
func vmExecutionLimits(ctx context.Context) (*executionMeter, *runtimeResourceAccount) {
	meter, _ := ctx.Value(executionMeterKey{}).(*executionMeter)
	account, _ := ctx.Value(runtimeResourceAccountKey{}).(*runtimeResourceAccount)
	if account == nil || account.output.limit == 0 {
		account = nil
	}
	return meter, account
}

func (runtime *Runtime) outputLimitEnabled() bool {
	return runtime != nil && runtime.resources != nil && runtime.resources.output.limit != 0
}

func consumeInstructionLimits(meter *executionMeter, outputAccount *runtimeResourceAccount) error {
	if outputAccount != nil {
		if limitErr := outputAccount.outputLimitError(); limitErr != nil {
			return limitErr
		}
	}
	if meter == nil {
		return nil
	}
	if meter.used.Add(1) <= meter.limit {
		return nil
	}
	return &LimitError{Resource: resourceInstruction, Limit: meter.limit}
}

func reserveRuntimeResource(ctx context.Context, resource string, amount uint64) error {
	account, _ := ctx.Value(runtimeResourceAccountKey{}).(*runtimeResourceAccount)
	return account.reserve(resource, amount)
}

func (runtime *Runtime) reserveResource(resource string, amount uint64) error {
	if runtime == nil {
		return nil
	}
	return runtime.resources.reserve(resource, amount)
}
