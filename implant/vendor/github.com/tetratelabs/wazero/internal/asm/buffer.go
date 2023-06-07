package asm

import (
	"encoding/binary"
	"fmt"
	"unsafe"

	"github.com/tetratelabs/wazero/internal/platform"
)

var zero [16]byte

// CodeSegment represents a memory mapped segment where native CPU instructions
// are written.
//
// To construct code segments, the program must call Next to obtain a buffer
// view capable of writing data at the end of the segment. Next must be called
// before generating the code of a function because it aligns the next write on
// 16 bytes.
//
// Instances of CodeSegment hold references to memory which is NOT managed by
// the garbage collector and therefore must be released *manually* by calling
// their Unmap method to prevent memory leaks.
//
// The zero value is a valid, empty code segment, equivalent to being
// constructed by calling NewCodeSegment(nil).
type CodeSegment struct {
	code []byte
	size int
}

// NewCodeSegment constructs a CodeSegment value from a byte slice.
//
// No validation is made that the byte slice is a memory mapped region which can
// be unmapped on Close.
func NewCodeSegment(code []byte) *CodeSegment {
	return &CodeSegment{code: code, size: len(code)}
}

// Map allocates a memory mapping of the given size to the code segment.
//
// Note that programs only need to use this method to initialize the code
// segment to a specific content (e.g. when loading pre-compiled code from a
// file), otherwise the backing memory mapping is allocated on demand when code
// is written to the code segment via Buffers returned by calls to Next.
//
// The method errors is the segment is already backed by a memory mapping.
func (seg *CodeSegment) Map(size int) error {
	if seg.code != nil {
		return fmt.Errorf("code segment already initialized to memory mapping of size %d", len(seg.code))
	}
	b, err := platform.MmapCodeSegment(size)
	if err != nil {
		return err
	}
	seg.code = b
	seg.size = size
	return nil
}

// Close unmaps the underlying memory region held by the code segment, clearing
// its state back to an empty code segment.
//
// The value is still usable after unmapping its memory, a new memory area can
// be allocated by calling Map or writing to the segment.
func (seg *CodeSegment) Unmap() error {
	if seg.code != nil {
		if err := platform.MunmapCodeSegment(seg.code[:cap(seg.code)]); err != nil {
			return err
		}
		seg.code = nil
		seg.size = 0
	}
	return nil
}

// Addr returns the address of the beginning of the code segment as a uintptr.
func (seg *CodeSegment) Addr() uintptr {
	if len(seg.code) > 0 {
		return uintptr(unsafe.Pointer(&seg.code[0]))
	}
	return 0
}

// Size returns the size of code segment, which is less or equal to the length
// of the byte slice returned by Len or Bytes.
func (seg *CodeSegment) Size() uintptr {
	return uintptr(seg.size)
}

// Len returns the length of the byte slice referencing the memory mapping of
// the code segment.
func (seg *CodeSegment) Len() int {
	return len(seg.code)
}

// Bytes returns a byte slice to the memory mapping of the code segment.
//
// The returned slice remains valid until more bytes are written to a buffer
// of the code segment, or Unmap is called.
func (seg *CodeSegment) Bytes() []byte {
	return seg.code
}

// Next returns a buffer pointed at the end of the code segment to support
// writing more code instructions to it.
//
// Buffers are passed by value, but they hold a reference to the code segment
// that they were created from.
func (seg *CodeSegment) NextCodeSection() Buffer {
	// Align 16-bytes boundary.
	seg.AppendBytes(zero[:seg.size&15])
	return Buffer{CodeSegment: seg, off: seg.size}
}

// Append appends n bytes to the code segment, returning a slice to the appended
// memory region.
//
// The underlying code segment may be reallocated if it was too short to hold
// n more bytes, which invalidates any addresses previously returned by calls
// to Addr.
func (seg *CodeSegment) Append(n int) []byte {
	seg.size += n
	if seg.size > len(seg.code) {
		seg.growToSize()
	}
	return seg.code[seg.size-n:]
}

// AppendByte appends a single byte to the code segment.
//
// The underlying code segment may be reallocated if it was too short to hold
// one more byte, which invalidates any addresses previously returned by calls
// to Addr.
func (seg *CodeSegment) AppendByte(b byte) {
	seg.size++
	if seg.size > len(seg.code) {
		seg.growToSize()
	}
	seg.code[seg.size-1] = b
}

// AppendBytes appends a copy of b to the code segment.
//
// The underlying code segment may be reallocated if it was too short to hold
// len(b) more bytes, which invalidates any addresses previously returned by
// calls to Addr.
func (seg *CodeSegment) AppendBytes(b []byte) {
	copy(seg.Append(len(b)), b)
}

// AppendUint32 appends a 32 bits integer to the code segment.
//
// The underlying code segment may be reallocated if it was too short to hold
// four more bytes, which invalidates any addresses previously returned by calls
// to Addr.
func (seg *CodeSegment) AppendUint32(u uint32) {
	seg.size += 4
	if seg.size > len(seg.code) {
		seg.growToSize()
	}
	// This can be replaced by an unsafe operation to assign the uint32, which
	// keeps the function cost below the inlining threshold. However, it did not
	// show any improvements in arm64 benchmarks so we retained this safer code.
	binary.LittleEndian.PutUint32(seg.code[seg.size-4:], u)
}

// growMode grows the code segment so that another section can be added to it.
//
// The method is marked go:noinline so that it doesn't get inline in Append,
// and AppendByte, which keeps the inlining score of those methods low enough
// that they can be inlined at the call sites.
//
//go:noinline
func (seg *CodeSegment) growToSize() {
	seg.Grow(0)
}

// Grow ensure that the capacity of the code segment is large enough to hold n
// more bytes.
//
// The underlying code segment may be reallocated if it was too short, which
// invalidates any addresses previously returned by calls to Addr.
func (seg *CodeSegment) Grow(n int) {
	size := len(seg.code)
	want := seg.size + n
	if size >= want {
		return
	}
	if size == 0 {
		size = 65536
	}
	for size < want {
		size *= 2
	}
	b, err := platform.RemapCodeSegment(seg.code, size)
	if err != nil {
		// The only reason for growing the buffer to error is if we run
		// out of memory, so panic for now as it greatly simplifies error
		// handling to assume writing to the buffer would never fail.
		panic(err)
	}
	seg.code = b
}

// Buffer is a reference type representing a section beginning at the end of a
// code segment where new instructions can be written.
type Buffer struct {
	*CodeSegment
	off int
}

func (buf Buffer) Cap() int {
	return len(buf.code) - buf.off
}

func (buf Buffer) Len() int {
	return buf.size - buf.off
}

func (buf Buffer) Bytes() []byte {
	return buf.code[buf.off:buf.size:buf.size]
}

func (buf Buffer) Reset() {
	buf.size = buf.off
}

func (buf Buffer) Truncate(n int) {
	buf.size = buf.off + n
}

func (buf Buffer) Append4Bytes(a, b, c, d byte) {
	buf.AppendUint32(uint32(a) | uint32(b)<<8 | uint32(c)<<16 | uint32(d)<<24)
}
