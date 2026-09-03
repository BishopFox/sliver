// Package bof loads and executes native Beacon Object Files from memory.
// Consumers opt in to BOF support by importing this package.
package bof

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"unicode/utf16"

	"github.com/sliverarmory/reflektor/internal/bofloader"
)

const (
	maxObjectSize          = 64 << 20
	maxArgumentBufferSize  = 16 << 20
	maxArgumentPayloadSize = maxArgumentBufferSize - 4
)

// ErrClosed is returned when Execute is called after Close.
var ErrClosed = errors.New("reflektor: BOF is closed")

// Output is one typed record emitted through BeaconOutput or BeaconPrintf.
// Type is the signed Beacon output channel supplied by the object; unknown
// values are preserved without remapping. Data is an owned copy of the raw
// record bytes and may have zero length. The bytes remain valid after later
// Execute calls and after Close.
type Output struct {
	Type int
	Data []byte
}

// Import describes one external symbol referenced by a BOF image. Name is the
// exact object-file symbol spelling. Builtin marks callbacks implemented by
// Reflektor. RequiresHost marks Beacon APIs that Reflektor deliberately does
// not implement and will not search for in system libraries.
type Import struct {
	Name         string
	Weak         bool
	Builtin      bool
	RequiresHost bool
}

// LoadOptions controls entry-point selection and external-symbol policy. Its
// zero value preserves Load's default behavior.
type LoadOptions struct {
	// EntryPoint selects an exact defined executable symbol. When empty,
	// Reflektor searches go, _go, coffee, and _coffee in that order.
	EntryPoint string

	// ValidateImports runs after parsing and host validation, but before image
	// allocation, callback registration, or dynamic-library lookup. The slice
	// is an owned, deterministic snapshot and may be retained by the callback.
	ValidateImports func([]Import) error

	// ResolveSymbol may provide a native address for an import that is not a
	// built-in Reflektor callback. Function addresses must follow the object's
	// platform ABI. Returning handled=false falls back to the normal system
	// resolver, except for RequiresHost imports, which fail explicitly. Any
	// returned error aborts loading. A handled address must be nonzero and
	// remain valid until the loaded BOF is closed. The resolver is called at
	// most once for each exact imported name during each load. Data imports
	// must use the object's native indirection convention when required; for
	// example, Windows data declarations should use __declspec(dllimport).
	ResolveSymbol func(Import) (address uintptr, handled bool, err error)
}

// Beacon output channel values used by Cobalt-compatible BOFs.
const (
	OutputDefault = 0x00
	OutputError   = 0x0d
	OutputOEM     = 0x1e
	OutputUTF8    = 0x20
)

// Arguments builds the length-prefixed argument format consumed by the
// BeaconData* callbacks. Its zero value is ready for use.
type Arguments struct {
	payload []byte
}

// AddInt32 appends a little-endian Beacon "integer" argument.
func (arguments *Arguments) AddInt32(value int32) error {
	var encoded [4]byte
	binary.LittleEndian.PutUint32(encoded[:], uint32(value))
	return arguments.append(encoded[:])
}

// AddInt16 appends a little-endian Beacon "short" argument.
func (arguments *Arguments) AddInt16(value int16) error {
	var encoded [2]byte
	binary.LittleEndian.PutUint16(encoded[:], uint16(value))
	return arguments.append(encoded[:])
}

// AddBytes appends a four-byte length followed by an arbitrary byte string.
func (arguments *Arguments) AddBytes(value []byte) error {
	if uint64(len(arguments.payload))+4+uint64(len(value)) > maxArgumentPayloadSize {
		return fmt.Errorf("reflektor: BOF argument payload exceeds %d bytes", maxArgumentPayloadSize)
	}
	var length [4]byte
	binary.LittleEndian.PutUint32(length[:], uint32(len(value)))
	arguments.payload = append(arguments.payload, length[:]...)
	arguments.payload = append(arguments.payload, value...)
	return nil
}

// AddString appends a NUL-terminated UTF-8 string argument.
func (arguments *Arguments) AddString(value string) error {
	if strings.IndexByte(value, 0) >= 0 {
		return errors.New("reflektor: BOF string argument contains NUL")
	}
	encoded := make([]byte, len(value)+1)
	copy(encoded, value)
	return arguments.AddBytes(encoded)
}

// AddUTF16String appends a NUL-terminated UTF-16LE "wstring" argument.
func (arguments *Arguments) AddUTF16String(value string) error {
	if strings.IndexByte(value, 0) >= 0 {
		return errors.New("reflektor: BOF UTF-16 argument contains NUL")
	}
	codeUnits := utf16.Encode([]rune(value))
	encoded := make([]byte, (len(codeUnits)+1)*2)
	for index, codeUnit := range codeUnits {
		binary.LittleEndian.PutUint16(encoded[index*2:], codeUnit)
	}
	return arguments.AddBytes(encoded)
}

// Bytes returns an owned argument buffer with its four-byte payload-length
// prefix. Calling Bytes does not consume or alias the builder.
func (arguments *Arguments) Bytes() []byte {
	packed := make([]byte, 4+len(arguments.payload))
	binary.LittleEndian.PutUint32(packed, uint32(len(arguments.payload)))
	copy(packed[4:], arguments.payload)
	return packed
}

// Reset discards all currently packed arguments while retaining capacity.
func (arguments *Arguments) Reset() {
	arguments.payload = arguments.payload[:0]
}

func (arguments *Arguments) append(value []byte) error {
	if uint64(len(arguments.payload))+uint64(len(value)) > maxArgumentPayloadSize {
		return fmt.Errorf("reflektor: BOF argument payload exceeds %d bytes", maxArgumentPayloadSize)
	}
	arguments.payload = append(arguments.payload, value...)
	return nil
}

type objectLoader interface {
	Execute([]byte) ([]bofloader.Output, error)
	Close() error
}

// Object is an in-memory native relocatable Beacon Object File. A loaded
// object may be executed more than once. Close releases its mapped image.
type Object struct {
	mu     sync.RWMutex
	loader objectLoader
	closed bool
}

// Load loads a native relocatable BOF image from memory using default load
// options.
func Load(data []byte) (*Object, error) {
	return LoadWithOptions(data, LoadOptions{})
}

// LoadWithOptions loads a native relocatable BOF image from memory. Windows
// accepts COFF, Linux accepts ELF, and Darwin accepts native Mach-O plus legacy
// ELF relocatable objects.
func LoadWithOptions(data []byte, options LoadOptions) (*Object, error) {
	if len(data) == 0 {
		return nil, errors.New("reflektor: empty BOF image")
	}
	loader, err := loadObject(data, options)
	if err != nil {
		return nil, err
	}
	return &Object{loader: loader}, nil
}

// LoadFile reads and loads a native relocatable BOF image from disk.
func LoadFile(path string) (*Object, error) {
	return LoadFileWithOptions(path, LoadOptions{})
}

// LoadFileWithOptions reads and loads a native relocatable BOF image from disk
// using the supplied load options.
func LoadFileWithOptions(path string, options LoadOptions) (*Object, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("reflektor: read BOF file: %w", err)
	}
	defer file.Close()

	if info, statErr := file.Stat(); statErr == nil && info.Mode().IsRegular() && info.Size() > maxObjectSize {
		return nil, fmt.Errorf("reflektor: BOF file is %d bytes; maximum is %d", info.Size(), maxObjectSize)
	}
	data, err := io.ReadAll(io.LimitReader(file, maxObjectSize+1))
	if err != nil {
		return nil, fmt.Errorf("reflektor: read BOF file: %w", err)
	}
	if len(data) > maxObjectSize {
		return nil, fmt.Errorf("reflektor: BOF file exceeds %d bytes", maxObjectSize)
	}
	return LoadWithOptions(data, options)
}

// Execute invokes the object's go (or coffee) entry point with an encoded
// Beacon argument buffer. It returns one record for every valid BeaconOutput
// or BeaconPrintf call, including zero-length records, in callback-capture
// order. Records captured during an execution that returns a terminal error
// are returned together with that error; callers must process the records even
// when err is non-nil.
func (object *Object) Execute(args []byte) ([]Output, error) {
	if object == nil {
		return nil, ErrClosed
	}
	object.mu.RLock()
	defer object.mu.RUnlock()
	if object.closed || object.loader == nil {
		return nil, ErrClosed
	}
	if len(args) > maxArgumentBufferSize {
		return nil, fmt.Errorf("reflektor: BOF argument buffer is %d bytes; maximum is %d", len(args), maxArgumentBufferSize)
	}
	outputs, err := object.loader.Execute(args)
	converted := make([]Output, len(outputs))
	for index := range outputs {
		converted[index] = Output{Type: outputs[index].Type, Data: outputs[index].Data}
	}
	if errors.Is(err, bofloader.ErrClosed) {
		err = ErrClosed
	}
	return converted, err
}

// Close releases the object's mapped image. It is safe to call more than once.
func (object *Object) Close() error {
	if object == nil {
		return nil
	}
	object.mu.Lock()
	defer object.mu.Unlock()
	if object.closed && object.loader == nil {
		return nil
	}
	object.closed = true
	if object.loader == nil {
		return nil
	}
	if err := object.loader.Close(); err != nil {
		// Keep the loader only so a later Close can retry cleanup. Execute is
		// permanently disabled as soon as the first Close begins.
		return err
	}
	object.loader = nil
	return nil
}
