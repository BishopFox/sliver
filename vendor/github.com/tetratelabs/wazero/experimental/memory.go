package experimental

import (
	"context"

	"github.com/tetratelabs/wazero/internal/expctxkeys"
)

// MemoryAllocator is a memory allocation hook,
// invoked to create a LinearMemory.
type MemoryAllocator interface {
	// Allocate should create a new LinearMemory with the given specification:
	// cap is the suggested initial capacity for the backing []byte,
	// and max the maximum length that will ever be requested.
	//
	// Notes:
	//   - To back a shared memory, the address of the backing []byte cannot
	//     change. This is checked at runtime. Implementations should document
	//     if the returned LinearMemory meets this requirement.
	Allocate(cap, max uint64) LinearMemory
}

// MemoryAllocatorFunc is a convenience for defining inlining a MemoryAllocator.
type MemoryAllocatorFunc func(cap, max uint64) LinearMemory

// Allocate implements MemoryAllocator.Allocate.
func (f MemoryAllocatorFunc) Allocate(cap, max uint64) LinearMemory {
	return f(cap, max)
}

// LinearMemory is an expandable []byte that backs a Wasm linear memory.
type LinearMemory interface {
	// Reallocates the linear memory to size bytes in length.
	//
	// Notes:
	//   - To back a shared memory, Reallocate can't change the address of the
	//     backing []byte (only its length/capacity may change).
	Reallocate(size uint64) []byte
	// Free the backing memory buffer.
	Free()
}

// WithMemoryAllocator registers the given MemoryAllocator into the given
// context.Context.
func WithMemoryAllocator(ctx context.Context, allocator MemoryAllocator) context.Context {
	if allocator != nil {
		return context.WithValue(ctx, expctxkeys.MemoryAllocatorKey{}, allocator)
	}
	return ctx
}
