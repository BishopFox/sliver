package opfor

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	osexec "os/exec"
	"path/filepath"
	"reflect"
	goruntime "runtime"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	defaultIOBufferCapacity = 32 * 1024
	defaultIOMarkLimit      = 1024 * 10 * 10
	sleepIOReadBufferSize   = 8192
)

// ioFunctions returns the portable Sleep BasicIO and filesystem functions.
// The behavior follows Sleep 2.1's BasicIO and FileSystemBridge where those
// APIs are meaningful without the JVM. Each function set owns an isolated
// working directory; it never changes the Go process's working directory.
func (r *Runtime) ioFunctions() map[string]NativeFunc {
	workingDirectory, err := os.Getwd()
	if err != nil {
		workingDirectory = "."
	}
	workingDirectory, err = filepath.Abs(workingDirectory)
	if err != nil {
		workingDirectory = filepath.Clean(workingDirectory)
	}

	state := &ioBuiltinState{
		runtime: r,
		cwd:     workingDirectory,
	}
	state.console = r.console
	if state.console == nil {
		state.console = newIOHandle("console", r.stdin, r.stdout, false, false, true).withRuntimeOutputAccount(r.resources)
	}
	return ioFunctionsForState(r, state)
}

// ioFunctionsForState is the single function-name/implementation inventory for
// the portable I/O bridge. Keeping state construction separate lets metadata
// callers enumerate the default functions without consulting the process
// working directory or allocating a live Runtime.
func ioFunctionsForState(r *Runtime, state *ioBuiltinState) map[string]NativeFunc {
	return map[string]NativeFunc{
		"allocate":            state.allocate,
		"fork":                r.fork,
		"wait":                r.wait,
		"getConsole":          state.getConsole,
		"closef":              state.closef,
		"connect":             state.connect,
		"listen":              state.listen,
		"readln":              state.readln,
		"readAll":             state.readAll,
		"readc":               state.readc,
		"read":                state.read,
		"readb":               state.readb,
		"bread":               state.bread,
		"consume":             state.consume,
		"skip":                state.consume,
		"writeb":              state.writeb,
		"bwrite":              state.bwrite,
		"readObject":          state.readObject,
		"writeObject":         state.writeObject,
		"readAsObject":        state.readAsObject,
		"writeAsObject":       state.writeAsObject,
		"available":           state.available,
		"mark":                state.mark,
		"reset":               state.reset,
		"setEncoding":         state.setEncoding,
		"printEOF":            state.printEOF,
		"openf":               state.openf,
		"sleep":               state.sleep,
		"cwd":                 state.currentDirectory,
		"pwd":                 state.currentDirectory,
		"getCurrentDirectory": state.currentDirectory,
		"chdir":               state.chdir,
		"ls":                  state.listFiles,
		"listRoots":           state.listFiles,
		"lof":                 state.lengthOfFile,
		"mkdir":               state.mkdir,
		"createNewFile":       state.createNewFile,
		"deleteFile":          state.deleteFile,
		"move":                state.move,
		"rename":              state.move,
		"copyFile":            state.copyFile,
		"dirname":             state.dirname,
		"getFileParent":       state.dirname,
		"getFileName":         state.getFileName,
		"getFileProper":       state.getFileProper,
		"lastModified":        state.lastModified,
		"setLastModified":     state.setLastModified,
		"setReadOnly":         state.setReadOnly,
		"-canread":            state.filePredicate,
		"-canwrite":           state.filePredicate,
		"-exists":             state.filePredicate,
		"-isDir":              state.filePredicate,
		"-isFile":             state.filePredicate,
		"-isHidden":           state.filePredicate,
		"-eof":                state.endOfFile,
		"-e":                  state.filePredicate,
		"-f":                  state.filePredicate,
		"-d":                  state.filePredicate,
		"__EXEC__":            state.execute,
		"exec":                state.execute,
	}
}

type ioBuiltinState struct {
	runtime *Runtime
	console *sleepIOHandle

	cwdMu sync.RWMutex
	cwd   string
}

type memoryIOPhase uint8

const (
	memoryIOWrite memoryIOPhase = iota
	memoryIORead
	memoryIOClosed
)

// contextReadSource is an optional structural contract for borrowed readers.
// It remains unexported so WithStdin keeps accepting every io.Reader, while an
// importer that already offers ReadContext gains truthful asynchronous-read
// cancellation without transferring Close ownership to OPFOR.
type contextReadSource interface {
	io.Reader
	ReadContext(context.Context, []byte) (int, error)
}

// sleepContextReadAdapter carries the context of one serialized handle read
// through bufio.Reader and any transparent checksum/digest wrappers to the
// original importer reader. readMu serializes calls at the handle level;
// callMu also keeps the adapter correct if a wrapper invokes it directly.
type sleepContextReadAdapter struct {
	source contextReadSource

	callMu sync.Mutex
	mu     sync.RWMutex
	ctx    context.Context
}

func (adapter *sleepContextReadAdapter) Read(data []byte) (int, error) {
	if adapter == nil || adapter.source == nil {
		return 0, io.ErrClosedPipe
	}
	adapter.mu.RLock()
	ctx := adapter.ctx
	adapter.mu.RUnlock()
	if ctx == nil {
		return adapter.source.Read(data)
	}
	return adapter.source.ReadContext(ctx, data)
}

func (adapter *sleepContextReadAdapter) readWithContext(
	ctx context.Context,
	read func() (int, error),
) (int, error) {
	if adapter == nil {
		return read()
	}
	adapter.callMu.Lock()
	defer adapter.callMu.Unlock()
	adapter.mu.Lock()
	adapter.ctx = ctx
	adapter.mu.Unlock()
	defer func() {
		adapter.mu.Lock()
		adapter.ctx = nil
		adapter.mu.Unlock()
	}()
	return read()
}

func (adapter *sleepContextReadAdapter) sleepUnderlyingReader() io.Reader {
	if adapter == nil {
		return nil
	}
	return adapter.source
}

// sleepIOHandle is the opaque object shared by Sleep's text and binary I/O
// functions. It also implements io.Writer so the core print functions can use
// a handle as their first argument.
type sleepIOHandle struct {
	mu          sync.Mutex
	readMu      sync.Mutex
	writeMu     sync.Mutex
	label       string
	reader      *bufio.Reader
	readSource  io.Reader
	contextRead *sleepContextReadAdapter
	writer      io.Writer
	readCloser  io.Closer
	writeClose  io.Closer
	ownRead     bool
	ownWrite    bool
	persistent  bool
	skipLF      bool
	task        *forkTask
	worker      sleepIOWorker
	process     *processObject

	// Sleep exposes the mark/reset support of IOObject's BufferedInputStream.
	// Go's bufio.Reader has no public equivalent, so reads made after a mark
	// are retained here and replayed in front of the unread input on reset.
	// The separate text buffer models InputStreamReader's read-ahead: readln
	// may consume bytes from the binary input buffer that available() can no
	// longer see, while later text reads can still consume the decoded data.
	// All fields below are protected by readMu, except textEncoder, which is
	// protected by writeMu.
	replay       []byte
	markData     []byte
	markLimit    int
	markValid    bool
	textBuffer   []uint16
	textPosition int
	textDecoder  sleepTextDecoder
	textEncoder  sleepTextEncoder
	writeEOFSent bool

	// The resource accounts are immutable after construction. outputAccount
	// charges every raw byte written to an explicit handle, while inputAccount
	// charges bytes materialized through the binary/text read pipelines. They
	// normally point at the same Runtime-family account.
	outputAccount *runtimeResourceAccount
	inputAccount  *runtimeResourceAccount

	memory      *bytes.Buffer
	memoryPhase memoryIOPhase
}

// sleepOnceCloser gives every owned transport one physical close operation.
// Context cancellation deliberately closes a transport without holding the
// handle mutex so it can wake a blocked Read or Write. Normal closef/lifecycle
// teardown may race that path, and a duplex handle may expose the same closer
// through both sides; coordinating at the stored closer keeps arbitrary host
// implementations from receiving concurrent or repeated Close calls.
type sleepOnceCloser struct {
	target io.Closer
	once   sync.Once
	err    error
}

func (closer *sleepOnceCloser) Close() error {
	if closer == nil || closer.target == nil {
		return nil
	}
	closer.once.Do(func() {
		closer.err = closer.target.Close()
	})
	return closer.err
}

func (h *sleepIOHandle) setTask(task *forkTask) {
	if h == nil {
		return
	}
	h.mu.Lock()
	h.task = task
	h.worker = task
	h.mu.Unlock()
}

func (h *sleepIOHandle) getTask() *forkTask {
	if h == nil {
		return nil
	}
	h.mu.Lock()
	task := h.task
	h.mu.Unlock()
	return task
}

// setWorker replaces IOObject's single mutable Thread slot. Starting another
// asynchronous operation changes what wait observes without canceling the
// previously started worker; fork's result token remains stored separately.
func (h *sleepIOHandle) setWorker(worker sleepIOWorker) {
	if h == nil {
		return
	}
	h.mu.Lock()
	h.worker = worker
	h.mu.Unlock()
}

func (h *sleepIOHandle) getWorker() sleepIOWorker {
	if h == nil {
		return nil
	}
	h.mu.Lock()
	worker := h.worker
	h.mu.Unlock()
	return worker
}

func (h *sleepIOHandle) setProcess(process *processObject) {
	if h == nil {
		return
	}
	h.mu.Lock()
	h.process = process
	h.mu.Unlock()
}

func (h *sleepIOHandle) getProcess() *processObject {
	if h == nil {
		return nil
	}
	h.mu.Lock()
	process := h.process
	h.mu.Unlock()
	return process
}

func newIOHandle(label string, reader io.Reader, writer io.Writer, ownRead, ownWrite, persistent bool) *sleepIOHandle {
	handle := &sleepIOHandle{
		label:      label,
		writer:     writer,
		ownRead:    ownRead,
		ownWrite:   ownWrite,
		persistent: persistent,
	}
	if reader != nil {
		readSource := reader
		if contextual, ok := reader.(contextReadSource); ok {
			handle.contextRead = &sleepContextReadAdapter{source: contextual}
			readSource = handle.contextRead
		}
		handle.readSource = readSource
		handle.reader = bufio.NewReaderSize(readSource, sleepIOReadBufferSize)
		handle.textDecoder.reset(sleepCharsetUTF8)
		if closer, ok := reader.(io.Closer); ok {
			handle.readCloser = closer
		}
	}
	if closer, ok := writer.(io.Closer); ok {
		handle.writeClose = closer
	}
	handle.readCloser, handle.writeClose = coordinateOwnedIOClosers(
		handle.readCloser,
		handle.writeClose,
		ownRead,
		ownWrite,
	)
	if writer != nil {
		handle.textEncoder.reset(sleepCharsetUTF8)
	}
	return handle
}

func coordinateOwnedIOClosers(readCloser, writeCloser io.Closer, ownRead, ownWrite bool) (io.Closer, io.Closer) {
	if ownRead && ownWrite && sameIOCloser(readCloser, writeCloser) {
		coordinator := &sleepOnceCloser{target: readCloser}
		return coordinator, coordinator
	}
	if ownRead && readCloser != nil {
		readCloser = &sleepOnceCloser{target: readCloser}
	}
	if ownWrite && writeCloser != nil {
		writeCloser = &sleepOnceCloser{target: writeCloser}
	}
	return readCloser, writeCloser
}

func (h *sleepIOHandle) withRuntimeOutputAccount(account *runtimeResourceAccount) *sleepIOHandle {
	if h != nil {
		h.outputAccount = account
		h.inputAccount = account
	}
	return h
}

func newMemoryIOHandle(capacity int, accounts ...*runtimeResourceAccount) *sleepIOHandle {
	var outputAccount *runtimeResourceAccount
	if len(accounts) != 0 {
		outputAccount = accounts[0]
	}
	// BufferObject's requested capacity is only a preallocation hint. Under an
	// output quota, grow from accounted writes so allocating many unused
	// buffers cannot reserve unmetered memory.
	if outputAccount != nil && outputAccount.output.limit != 0 {
		capacity = 0
	}
	buffer := bytes.NewBuffer(make([]byte, 0, capacity))
	handle := &sleepIOHandle{
		label:         "memory",
		writer:        buffer,
		ownWrite:      true,
		outputAccount: outputAccount,
		inputAccount:  outputAccount,
		memory:        buffer,
		memoryPhase:   memoryIOWrite,
	}
	handle.textEncoder.reset(sleepCharsetUTF8)
	return handle
}

func (h *sleepIOHandle) String() string {
	if h == nil {
		return "<io:nil>"
	}
	return "<io:" + h.label + ">"
}

func (h *sleepIOHandle) runtimeOutputAccount() *runtimeResourceAccount {
	if h == nil {
		return nil
	}
	return h.outputAccount
}

func (h *sleepIOHandle) Read(data []byte) (int, error) {
	if h == nil {
		return 0, io.ErrClosedPipe
	}
	h.readMu.Lock()
	defer h.readMu.Unlock()
	return h.readBinaryLockedContext(context.Background(), data)
}

func (h *sleepIOHandle) Write(data []byte) (int, error) {
	if h == nil {
		return 0, io.ErrClosedPipe
	}
	h.writeMu.Lock()
	defer h.writeMu.Unlock()
	return h.writeRawLocked(data)
}

// writeRawLocked writes through DataOutputStream's side of the modeled
// IOObject. The caller must hold writeMu. Text output uses a separate encoder
// before reaching this hook; writeb and bwrite deliberately call it directly.
func (h *sleepIOHandle) writeRawLocked(data []byte) (int, error) {
	h.mu.Lock()
	writer := h.writer
	// BufferObject's phase transition copies bytes from this buffer while
	// holding mu. Keep its non-blocking in-memory write under the same lock;
	// every other writer is detached before potentially blocking.
	if h.memory != nil {
		defer h.mu.Unlock()
		if writer == nil {
			return 0, fmt.Errorf("%s is not open for writing", h)
		}
		return writeRuntimeOutput(h.outputAccount, writer, data)
	}
	h.mu.Unlock()
	if writer == nil {
		return 0, fmt.Errorf("%s is not open for writing", h)
	}
	return writeRuntimeOutput(h.outputAccount, writer, data)
}

type sleepIOTextWriter struct {
	handle *sleepIOHandle
}

func (writer sleepIOTextWriter) runtimeOutputAccount() *runtimeResourceAccount {
	if writer.handle == nil {
		return nil
	}
	return writer.handle.outputAccount
}

func (writer sleepIOTextWriter) Write(data []byte) (int, error) {
	if writer.handle == nil {
		return len(data), nil
	}
	failed := false
	var failureErr error
	writer.handle.writeMu.Lock()
	writer.handle.mu.Lock()
	open := writer.handle.writer != nil
	writer.handle.mu.Unlock()
	if open {
		encoded := writer.handle.textEncoder.encode(string(data))
		for len(encoded) != 0 {
			written, err := writer.handle.writeRawLocked(encoded)
			if written > 0 {
				encoded = encoded[written:]
			}
			if err != nil || written == 0 {
				failed = true
				failureErr = err
				break
			}
		}
		if !failed {
			writer.handle.mu.Lock()
			rawWriter := writer.handle.writer
			writer.handle.mu.Unlock()
			if err := flushWriter(rawWriter); err != nil {
				failed = true
				failureErr = err
			}
		}
	}
	writer.handle.writeMu.Unlock()
	if failed {
		// IOObject.print swallows output failures after closing the handle.
		_ = writer.handle.close()
		// Resource quotas are OPFOR safety boundaries rather than a modeled Java
		// stream failure. Preserve the historical swallowing policy for ordinary
		// writers, but let the output LimitError terminate the active execution.
		if errors.Is(failureErr, ErrResourceLimit) {
			return len(data), failureErr
		}
	}
	return len(data), nil
}

// wrapRead installs a transparent reader around the handle's latest raw input
// hook. Sleep's digest/checksum functions rebuild the buffered read pipeline in
// the same way, so bytes consumed after this call pass through the wrapper.
func (h *sleepIOHandle) wrapRead(wrapper func(io.Reader) io.Reader) error {
	if h == nil {
		return io.ErrClosedPipe
	}
	h.readMu.Lock()
	defer h.readMu.Unlock()
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.readSource == nil {
		return fmt.Errorf("%s is not open for reading", h)
	}
	h.readSource = wrapper(h.readSource)
	h.reader = bufio.NewReaderSize(h.readSource, sleepIOReadBufferSize)
	h.skipLF = false
	h.replay = nil
	h.markData = nil
	h.markLimit = 0
	h.markValid = false
	h.textBuffer = nil
	h.textPosition = 0
	h.textDecoder.reset(sleepCharsetUTF8)
	return nil
}

// wrapWrite installs a transparent writer around the handle's latest output
// hook. The original closer remains owned by the handle; wrappers only observe
// bytes and forward flushes to the underlying writer.
func (h *sleepIOHandle) wrapWrite(wrapper func(io.Writer) io.Writer) error {
	if h == nil {
		return io.ErrClosedPipe
	}
	h.writeMu.Lock()
	defer h.writeMu.Unlock()
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.writer == nil {
		return fmt.Errorf("%s is not open for writing", h)
	}
	h.writer = wrapper(h.writer)
	h.textEncoder.reset(sleepCharsetUTF8)
	return nil
}

func (h *sleepIOHandle) setTextEncoding(name string) error {
	if h == nil {
		return io.ErrClosedPipe
	}
	h.readMu.Lock()
	defer h.readMu.Unlock()
	h.writeMu.Lock()
	defer h.writeMu.Unlock()
	h.mu.Lock()
	readerOpen := h.reader != nil
	writerOpen := h.writer != nil
	h.mu.Unlock()
	// IOObject.setEncoding only constructs a charset wrapper for an open
	// binary side. A fully closed object therefore does not validate the name.
	if !readerOpen && !writerOpen {
		return nil
	}
	charset, err := sleepLookupTextCharset(name)
	if err != nil {
		return err
	}
	if writerOpen {
		// Replacing OutputStreamWriter does not flush or close its predecessor;
		// any buffered high surrogate is intentionally discarded.
		h.textEncoder.reset(charset)
	}
	if readerOpen {
		// InputStreamReader may already own up to 8192 bytes read from the raw
		// stream. Replacing it discards those decoded characters and decoder
		// state without rewinding the shared binary side.
		h.textBuffer = nil
		h.textPosition = 0
		h.textDecoder.reset(charset)
	}
	return nil
}

func (h *sleepIOHandle) close() error {
	if h == nil {
		return io.ErrClosedPipe
	}
	h.writeMu.Lock()
	defer h.writeMu.Unlock()
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.closeLocked(true)
}

// closeAfterTextEOF mirrors IOObject.readCharacter and IOObject.readLine
// calling IOObject.close after EOF or a text-read failure.
// A console handle normally treats explicit closef/printEOF as a flush-only
// operation because OPFOR borrows the host streams. Text EOF is different:
// Sleep closes the console IOObject itself, so later implicit and explicit
// prints are silent even though the borrowed Go streams remain usable by the
// host. The caller may hold readMu; close intentionally does not wait for it so
// a duplex handle can close its writer while another raw read is blocked.
func (h *sleepIOHandle) closeAfterTextEOF() error {
	if h == nil {
		return io.ErrClosedPipe
	}
	h.writeMu.Lock()
	defer h.writeMu.Unlock()
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.closeLocked(false)
}

// closeLocked closes both logical pipelines. The caller must hold writeMu and
// mu. respectPersistent retains OPFOR's borrowed-console behavior for explicit
// closef and printEOF; text EOF passes false to match IOObject.readCharacter
// and IOObject.readLine.
func (h *sleepIOHandle) closeLocked(respectPersistent bool) error {
	if respectPersistent && h.persistent {
		return flushWriter(h.writer)
	}

	if h.memory != nil && h.memoryPhase == memoryIOWrite {
		closeErr := h.closeWriteLocked(true)
		data := append([]byte(nil), h.memory.Bytes()...)
		reader := bytes.NewReader(data)
		h.reader = bufio.NewReaderSize(reader, sleepIOReadBufferSize)
		h.readSource = reader
		h.textDecoder.reset(sleepCharsetUTF8)
		h.memoryPhase = memoryIORead
		return closeErr
	}

	sharedCloser := h.ownRead && h.ownWrite && sameIOCloser(h.readCloser, h.writeClose)
	readErr := h.closeReadLocked()
	if sharedCloser {
		// The read side already reported the one shared physical close. Keep the
		// writer's logical flush/teardown, but do not append the coordinator's
		// cached error a second time.
		h.writeClose = nil
	}
	writeErr := h.closeWriteLocked(true)
	if h.memory != nil {
		h.memory = nil
		h.memoryPhase = memoryIOClosed
	}
	return errors.Join(readErr, writeErr)
}

func (h *sleepIOHandle) closeWrite() error {
	if h == nil {
		return io.ErrClosedPipe
	}
	h.writeMu.Lock()
	defer h.writeMu.Unlock()
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.persistent {
		// The host owns the portable console streams. Preserve their physical
		// lifetime even though the JVM bridge closes System.out here.
		_ = flushWriter(h.writer)
		return nil
	}
	// IOObject.sendEOF closes writerb/out but deliberately leaves every
	// writer field installed. This is observable for BufferObject because
	// ByteArrayOutputStream.close is a no-op: later binary/text writes still
	// append, and setEncoding still validates its name.
	_ = flushWriter(h.writer)
	if h.ownWrite && h.writeClose != nil {
		_ = h.writeClose.Close()
	}
	h.writeEOFSent = true
	return nil
}

func (h *sleepIOHandle) closeReadLocked() error {
	var err error
	if h.ownRead && h.readCloser != nil {
		err = h.readCloser.Close()
	}
	h.reader = nil
	h.readSource = nil
	h.contextRead = nil
	h.readCloser = nil
	h.ownRead = false
	h.skipLF = false
	return err
}

func (h *sleepIOHandle) closeWriteLocked(finishText bool) error {
	ignoreErrors := h.writeEOFSent
	var textErr error
	if finishText && h.writer != nil {
		if pending := h.textEncoder.finish(); len(pending) != 0 {
			_, textErr = writeRuntimeOutput(h.outputAccount, h.writer, pending)
		}
	}
	flushErr := flushWriter(h.writer)
	var closeErr error
	if h.ownWrite && h.writeClose != nil {
		closeErr = h.writeClose.Close()
	}
	h.writer = nil
	h.writeClose = nil
	h.ownWrite = false
	h.textEncoder.reset(sleepCharsetUTF8)
	h.writeEOFSent = false
	if ignoreErrors {
		return nil
	}
	return errors.Join(textErr, flushErr, closeErr)
}

func flushWriter(writer io.Writer) error {
	if flusher, ok := writer.(interface{ Flush() error }); ok {
		return flusher.Flush()
	}
	return nil
}

func (h *sleepIOHandle) readLine() (Value, bool, error) {
	return h.readLineContext(context.Background())
}

func (h *sleepIOHandle) readLineContext(ctx context.Context) (Value, bool, error) {
	if h == nil {
		return Null(), false, io.ErrClosedPipe
	}
	h.readMu.Lock()
	defer h.readMu.Unlock()
	h.mu.Lock()
	reader := h.reader
	skipLF := h.skipLF
	h.skipLF = false
	h.mu.Unlock()
	if reader == nil {
		return Null(), false, nil
	}

	if skipLF {
		next, err := h.readTextUnitLocked(ctx)
		switch {
		case err == nil && next != '\n':
			h.textPosition--
		case errors.Is(err, io.EOF):
			return Null(), false, h.closeAfterTextEOF()
		case err != nil:
			closeErr := h.closeAfterTextEOF()
			return Null(), false, errors.Join(err, closeErr)
		}
	}

	line := make([]uint16, 0, 8192)
	for {
		character, err := h.readTextUnitLocked(ctx)
		switch {
		case err == nil:
			switch character {
			case '\n':
				return sleepStringValueFromUnits(line, nil), true, nil
			case '\r':
				h.mu.Lock()
				if h.reader == reader {
					h.skipLF = true
				}
				h.mu.Unlock()
				return sleepStringValueFromUnits(line, nil), true, nil
			default:
				line = append(line, character)
			}
		case errors.Is(err, io.EOF):
			closeErr := h.closeAfterTextEOF()
			if len(line) != 0 {
				return sleepStringValueFromUnits(line, nil), true, closeErr
			}
			return Null(), false, closeErr
		default:
			closeErr := h.closeAfterTextEOF()
			return Null(), false, errors.Join(err, closeErr)
		}
	}
}

// readTextUnitLocked reads through the unicode-aware side of Sleep's input
// pipeline. InputStreamReader reads ahead in 8192-byte chunks from the shared
// binary stream, which is why available() may report zero while readln() can
// still return more lines. The caller must hold readMu.
func (h *sleepIOHandle) readTextUnitLocked(ctx context.Context) (uint16, error) {
	for {
		if h.textPosition < len(h.textBuffer) {
			value := h.textBuffer[h.textPosition]
			h.textPosition++
			return value, nil
		}

		buffer := make([]byte, sleepIOReadBufferSize)
		amount, err := h.readBinaryLockedContext(ctx, buffer)
		// BufferedInputStream satisfies InputStreamReader's 8192-byte request
		// from both its buffered tail and bytes known to be immediately
		// available from the underlying source. bufio.Reader.Read returns after
		// only the buffered tail, so explicitly finish the non-blocking portion
		// of that request. Unknown streaming sources are not probed again.
		for amount < len(buffer) && err == nil {
			available, open, availableErr := h.availableBytesLocked()
			if availableErr != nil || !open || available <= 0 {
				break
			}
			request := len(buffer) - amount
			if available < int64(request) {
				request = int(available)
			}
			more, readErr := h.readBinaryLockedContext(ctx, buffer[amount:amount+request])
			amount += more
			err = readErr
			if more == 0 {
				break
			}
		}
		h.textBuffer = h.textDecoder.decode(buffer[:amount], errors.Is(err, io.EOF))
		h.textPosition = 0
		if len(h.textBuffer) != 0 {
			continue
		}
		if errors.Is(err, io.EOF) {
			return 0, io.EOF
		}
		if err != nil {
			return 0, err
		}
		if amount == 0 {
			return 0, io.ErrNoProgress
		}
	}
}

func (h *sleepIOHandle) readCharacter(ctx context.Context) (uint16, bool, error) {
	if h == nil {
		return 0, false, io.ErrClosedPipe
	}
	h.readMu.Lock()
	defer h.readMu.Unlock()
	h.mu.Lock()
	reader := h.reader
	h.mu.Unlock()
	if reader == nil {
		return 0, false, nil
	}
	unit, err := h.readTextUnitLocked(ctx)
	if err == nil {
		return unit, true, nil
	}
	closeErr := h.closeAfterTextEOF()
	if errors.Is(err, io.EOF) {
		return 0, false, closeErr
	}
	return 0, false, errors.Join(err, closeErr)
}

// readBinaryLocked reads from the DataInputStream side of Sleep's IOObject.
// Replay data logically precedes the current buffered reader. The caller must
// hold readMu.
func (h *sleepIOHandle) readBinaryLocked(data []byte) (int, error) {
	return h.readBinaryLockedContext(context.Background(), data)
}

func (h *sleepIOHandle) readBinaryLockedContext(ctx context.Context, data []byte) (int, error) {
	if len(data) == 0 {
		return 0, nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		h.abortRead()
		return 0, err
	}
	h.mu.Lock()
	reader := h.reader
	contextReader := h.contextRead
	h.mu.Unlock()
	if reader == nil {
		return 0, io.EOF
	}

	replayRead := copy(data, h.replay)
	amount := 0
	var err error
	if replayRead < len(data) {
		stopCancellation := h.cancelReadOnContext(ctx)
		read := func() (int, error) { return reader.Read(data[replayRead:]) }
		amount, err = contextReader.readWithContext(ctx, read)
		stopCancellation()
		if contextErr := ctx.Err(); contextErr != nil {
			h.abortRead()
			return 0, contextErr
		}
	}
	total := replayRead + amount
	if reserveErr := h.inputAccount.reserve(resourceInputBytes, uint64(total)); reserveErr != nil {
		h.abortRead()
		return 0, reserveErr
	}
	if replayRead != 0 {
		h.replay = h.replay[replayRead:]
	}
	if total != 0 {
		h.recordMarkedReadLocked(data[:total])
	}
	return total, err
}

// cancelReadOnContext makes owned blocking inputs observable to importer
// cancellation without leaking a waiter goroutine. Closing an owned file or
// socket, or destroying a managed process, is terminal for the Sleep IOObject
// and wakes the in-flight Read. Borrowed console streams are never closed.
func (h *sleepIOHandle) cancelReadOnContext(ctx context.Context) func() {
	if h == nil || ctx == nil || ctx.Done() == nil {
		return func() {}
	}
	h.mu.Lock()
	process := h.process
	closer := h.readCloser
	owned := h.ownRead
	h.mu.Unlock()
	if process == nil && (!owned || closer == nil) {
		return func() {}
	}
	stop := context.AfterFunc(ctx, func() {
		if process != nil {
			_ = process.close()
			return
		}
		_ = closer.Close()
	})
	return func() { _ = stop() }
}

func (h *sleepIOHandle) abortRead() {
	if h == nil {
		return
	}
	if process := h.getProcess(); process != nil {
		_ = process.close()
		return
	}
	h.mu.Lock()
	persistent := h.persistent
	h.mu.Unlock()
	if persistent {
		// The host owns console lifetime and buffering. A terminal quota error
		// stops the active execution without closing or flushing borrowed streams.
		return
	}
	// A concurrent duplex write owns writeMu while blocked in the transport.
	// Close the owned transport before close() waits for writeMu; otherwise a
	// quota-failing read can deadlock with a writer that only transport closure
	// can wake. Borrowed persistent console streams are never physically closed.
	h.abortOwnedTransport()
	_ = h.close()
}

func (h *sleepIOHandle) abortOwnedTransport() {
	if h == nil {
		return
	}
	h.mu.Lock()
	if h.persistent {
		h.mu.Unlock()
		return
	}
	readCloser, ownRead := h.readCloser, h.ownRead
	writeCloser, ownWrite := h.writeClose, h.ownWrite
	h.mu.Unlock()

	// Closing the output side first releases a write currently holding writeMu.
	if ownWrite && writeCloser != nil {
		_ = writeCloser.Close()
	}
	if ownRead && readCloser != nil && !sameIOCloser(readCloser, writeCloser) {
		_ = readCloser.Close()
	}
}

func sameIOCloser(left, right io.Closer) bool {
	if left == nil || right == nil {
		return false
	}
	leftType, rightType := reflect.TypeOf(left), reflect.TypeOf(right)
	if leftType != rightType || !leftType.Comparable() {
		return false
	}
	return reflect.ValueOf(left).Interface() == reflect.ValueOf(right).Interface()
}

// recordMarkedReadLocked retains bytes consumed since the latest mark. Java's
// read-limit makes reset unavailable once a reader advances beyond the limit.
// The caller must hold readMu.
func (h *sleepIOHandle) recordMarkedReadLocked(data []byte) {
	if !h.markValid || len(data) == 0 {
		return
	}
	// BufferedInputStream can retain at least its current 8192-byte buffer even
	// when the caller supplies a smaller read limit. It invalidates the mark on
	// the refill that would exceed that capacity; larger requested limits allow
	// the reference buffer to grow up to the requested size.
	retentionLimit := h.markLimit
	if retentionLimit < sleepIOReadBufferSize {
		retentionLimit = sleepIOReadBufferSize
	}
	if retentionLimit < len(h.markData) || len(data) > retentionLimit-len(h.markData) {
		h.markData = nil
		h.markValid = false
		return
	}
	h.markData = append(h.markData, data...)
}

func (h *sleepIOHandle) readBytes(count int) ([]byte, error) {
	return h.readBytesContext(context.Background(), count)
}

func (h *sleepIOHandle) readBytesContext(ctx context.Context, count int) ([]byte, error) {
	if h == nil {
		return nil, io.ErrClosedPipe
	}
	h.readMu.Lock()
	defer h.readMu.Unlock()
	h.mu.Lock()
	reader := h.reader
	h.mu.Unlock()
	if reader == nil || count == 0 {
		return nil, nil
	}
	capacity := sleepIOReadBufferSize
	if count >= 0 && count < capacity {
		capacity = count
	}
	data := make([]byte, 0, capacity)
	buffer := make([]byte, sleepIOReadBufferSize)
	if count >= 0 && count < len(buffer) {
		buffer = buffer[:count]
	}
	read := 0
	for count == -1 || read < count {
		chunk := buffer
		if count >= 0 && len(chunk) > count-read {
			chunk = chunk[:count-read]
		}
		amount, err := h.readBinaryLockedContext(ctx, chunk)
		data = append(data, chunk[:amount]...)
		read += amount
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, err
		}
		if amount == 0 {
			return nil, io.ErrNoProgress
		}
	}
	return data, nil
}

// consumeBytes discards up to count bytes through the same binary pipeline as
// readb. The caller-provided buffer size is observable for short readers and
// mark/reset, so keep it instead of delegating to io.CopyN. Reaching EOF is a
// successful partial consume and deliberately leaves the input pipeline open;
// Sleep's -eof predicate reports a closed pipeline, not a probed stream end.
func (h *sleepIOHandle) consumeBytes(count, bufferSize int) (int, bool, error) {
	return h.consumeBytesContext(context.Background(), count, bufferSize)
}

func (h *sleepIOHandle) consumeBytesContext(ctx context.Context, count, bufferSize int) (int, bool, error) {
	if h == nil {
		return 0, false, io.ErrClosedPipe
	}
	h.readMu.Lock()
	defer h.readMu.Unlock()
	h.mu.Lock()
	readerOpen := h.reader != nil
	h.mu.Unlock()
	if !readerOpen {
		return 0, false, nil
	}
	if bufferSize < 0 {
		return 0, true, fmt.Errorf("%d", bufferSize)
	}
	if count <= 0 {
		return 0, true, nil
	}
	if bufferSize == 0 {
		// DataInputStream.read(byte[0], 0, 0) cannot advance. The pinned
		// bridge loops forever for this input; surface a deterministic I/O
		// failure instead of wedging the runtime.
		return 0, true, io.ErrNoProgress
	}

	allocationSize := bufferSize
	if allocationSize > count {
		allocationSize = count
	}
	// A configured input bound is also a bound on transient consume buffers.
	// Keep the bridge's requested chunk size when input is unlimited; otherwise
	// stream through the same fixed-size chunk used by readb/readln.
	if h.inputAccount != nil && h.inputAccount.input.limit != 0 && allocationSize > sleepIOReadBufferSize {
		allocationSize = sleepIOReadBufferSize
	}
	buffer := make([]byte, allocationSize)
	consumed := 0
	for consumed < count {
		chunk := buffer
		if len(chunk) > count-consumed {
			chunk = chunk[:count-consumed]
		}
		amount, err := h.readBinaryLockedContext(ctx, chunk)
		consumed += amount
		if errors.Is(err, io.EOF) {
			return consumed, true, nil
		}
		if err != nil {
			return consumed, true, err
		}
		if amount == 0 {
			return consumed, true, io.ErrNoProgress
		}
	}
	return consumed, true, nil
}

func (h *sleepIOHandle) isEOF() bool {
	if h == nil {
		return true
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.reader == nil
}

func (h *sleepIOHandle) availableBytes() (int64, bool, error) {
	if h == nil {
		return 0, false, io.ErrClosedPipe
	}
	h.readMu.Lock()
	defer h.readMu.Unlock()
	return h.availableBytesLocked()
}

func (h *sleepIOHandle) availableBytesLocked() (int64, bool, error) {
	h.mu.Lock()
	reader := h.reader
	source := h.readSource
	h.mu.Unlock()
	if reader == nil {
		return 0, false, nil
	}

	available := int64(len(h.replay) + reader.Buffered())
	for {
		wrapper, ok := source.(interface{ sleepUnderlyingReader() io.Reader })
		if !ok {
			break
		}
		source = wrapper.sleepUnderlyingReader()
	}
	switch source := source.(type) {
	case interface{ Len() int }:
		available += int64(source.Len())
	case *os.File:
		position, err := source.Seek(0, io.SeekCurrent)
		if err != nil {
			return 0, false, err
		}
		info, err := source.Stat()
		if err != nil {
			return 0, false, err
		}
		remaining := info.Size() - position
		if remaining > 0 {
			available += remaining
		}
	}
	return available, true, nil
}

func (h *sleepIOHandle) availableContains(delimiter string) (bool, bool, error) {
	return h.availableContainsContext(context.Background(), delimiter)
}

func (h *sleepIOHandle) availableContainsContext(ctx context.Context, delimiter string) (bool, bool, error) {
	if h == nil {
		return false, false, io.ErrClosedPipe
	}
	h.readMu.Lock()
	defer h.readMu.Unlock()
	available, open, err := h.availableBytesLocked()
	if err != nil || !open {
		return false, open, err
	}
	if available > int64(maximumInt()) {
		return false, true, fmt.Errorf("available byte count %d exceeds platform range", available)
	}
	if err := h.markInputLocked(int(available)); err != nil {
		return false, true, err
	}
	capacity := int(available)
	if h.inputAccount != nil && h.inputAccount.input.limit != 0 && capacity > sleepIOReadBufferSize {
		capacity = sleepIOReadBufferSize
	}
	data := make([]byte, 0, capacity)
	var one [1]byte
	for int64(len(data)) < available {
		amount, readErr := h.readBinaryLockedContext(ctx, one[:])
		if amount != 0 {
			data = append(data, one[0])
		}
		if readErr != nil {
			return false, true, readErr
		}
		if amount == 0 {
			return false, true, io.ErrNoProgress
		}
	}
	if err := h.resetInputLocked(); err != nil {
		return false, true, err
	}
	return strings.Contains(string(data), delimiter), true, nil
}

func (h *sleepIOHandle) markInput(limit int) error {
	if h == nil {
		return io.ErrClosedPipe
	}
	h.readMu.Lock()
	defer h.readMu.Unlock()
	return h.markInputLocked(limit)
}

// markInputLocked updates the shared binary mark while the caller holds
// readMu. Formatted reads use this form so their complete M...R sequence is
// atomic with respect to other readers of the same handle.
func (h *sleepIOHandle) markInputLocked(limit int) error {
	h.mu.Lock()
	reader := h.reader
	h.mu.Unlock()
	if reader == nil {
		return fmt.Errorf("input buffer for %s is closed", h)
	}
	h.markData = nil
	h.markLimit = limit
	h.markValid = true
	return nil
}

// resetInput mirrors BufferedInputStream.reset. A missing or expired mark is
// reported to the caller; BasicIO's reset builtin deliberately swallows that
// error, as the reference bridge does.
func (h *sleepIOHandle) resetInput() error {
	if h == nil {
		return io.ErrClosedPipe
	}
	h.readMu.Lock()
	defer h.readMu.Unlock()
	return h.resetInputLocked()
}

func (h *sleepIOHandle) resetInputLocked() error {
	h.mu.Lock()
	reader := h.reader
	h.mu.Unlock()
	if reader == nil {
		return fmt.Errorf("input buffer for %s is closed", h)
	}
	if !h.markValid {
		return fmt.Errorf("input buffer for %s has no valid mark", h)
	}
	if len(h.markData) != 0 {
		replay := make([]byte, 0, len(h.markData)+len(h.replay))
		replay = append(replay, h.markData...)
		replay = append(replay, h.replay...)
		h.replay = replay
		h.markData = nil
	}
	return nil
}

func (state *ioBuiltinState) allocate(_ context.Context, invocation Invocation) (Value, error) {
	capacity := int64(defaultIOBufferCapacity)
	if len(invocation.Arguments) != 0 {
		capacity = invocation.Arg(0).Int64()
	}
	if capacity < 0 {
		return Null(), fmt.Errorf("&%s: capacity must not be negative", builtinName(invocation.Name))
	}
	if capacity > int64(maximumInt()) {
		return Null(), fmt.Errorf("&%s: capacity %d exceeds platform limit", builtinName(invocation.Name), capacity)
	}
	return ObjectValue(newMemoryIOHandle(int(capacity), state.runtime.resources)), nil
}

func (state *ioBuiltinState) getConsole(_ context.Context, _ Invocation) (Value, error) {
	return ObjectValue(state.console), nil
}

func (state *ioBuiltinState) closef(ctx context.Context, invocation Invocation) (Value, error) {
	if len(invocation.Arguments) != 0 {
		if handle, ok := ioHandleValue(invocation.Arg(0)); ok {
			if state != nil && state.runtime != nil && state.runtime.socketState != nil {
				if socket := state.runtime.socketState.lookup(handle); socket != nil {
					socket.closeHandle()
					return Null(), nil
				}
			}
			var closeErr error
			if process := handle.getProcess(); process != nil {
				closeErr = process.close()
			} else {
				closeErr = handle.close()
			}
			if closeErr != nil {
				return Null(), preserveNativeBoundaryError(ctx,
					fmt.Errorf("&%s: close: %w", builtinName(invocation.Name), closeErr))
			}
			return Null(), nil
		}
	}
	port := int32(80)
	if len(invocation.Arguments) != 0 {
		port = sleepSocketInt32(invocation.Arg(0))
	}
	if state != nil && state.runtime != nil && state.runtime.socketState != nil {
		state.runtime.socketState.release(port)
	}
	return Null(), nil
}

func (state *ioBuiltinState) readln(ctx context.Context, invocation Invocation) (Value, error) {
	handle, _, err := state.chooseHandle(invocation, 1)
	if err != nil {
		return Null(), err
	}
	line, ok, err := handle.readLineContext(ctx)
	if err != nil {
		return Null(), preserveNativeBoundaryError(ctx,
			fmt.Errorf("&%s: read line: %w", builtinName(invocation.Name), err))
	}
	if !ok {
		return Null(), nil
	}
	return line, nil
}

func (state *ioBuiltinState) readc(ctx context.Context, invocation Invocation) (Value, error) {
	handle, _, err := state.chooseHandle(invocation, 1)
	if err != nil {
		return Null(), sleepIOBridgeWarning(ctx, invocation, err)
	}
	unit, ok, readErr := handle.readCharacter(ctx)
	if readErr != nil {
		return Null(), preserveNativeBoundaryError(ctx,
			fmt.Errorf("&%s: read character: %w", builtinName(invocation.Name), readErr))
	}
	if !ok {
		return Null(), nil
	}
	return sleepUTF16CharacterValue(unit), nil
}

func (state *ioBuiltinState) setEncoding(ctx context.Context, invocation Invocation) (Value, error) {
	handle, nameIndex, err := state.chooseHandle(invocation, 1)
	if err != nil {
		return Null(), sleepIOBridgeWarning(ctx, invocation, err)
	}
	name := ""
	if nameIndex < len(invocation.Arguments) {
		name = invocation.Arg(nameIndex).String()
	}
	if err := handle.setTextEncoding(name); err != nil {
		warning := fmt.Errorf("&setEncoding: specified a non-existent encoding '%s'", name)
		return Null(), outputWarning(ctx, warning)
	}
	return Null(), nil
}

func (state *ioBuiltinState) readAll(ctx context.Context, invocation Invocation) (Value, error) {
	handle, _, err := state.chooseHandle(invocation, 1)
	if err != nil {
		return Null(), err
	}
	lines := NewArray()
	for {
		line, ok, readErr := handle.readLineContext(ctx)
		if readErr != nil {
			return Null(), preserveNativeBoundaryError(ctx,
				fmt.Errorf("&%s: read lines: %w", builtinName(invocation.Name), readErr))
		}
		if !ok {
			break
		}
		if err := lines.appendValuesAtExecution(ctx, invocation, line); err != nil {
			return Null(), err
		}
	}
	return ArrayValue(lines), nil
}

// read starts Sleep's CallbackReader worker and returns immediately. The
// worker owns callback dispatch, EOF handling, and the handle's wait slot.
func (state *ioBuiltinState) read(ctx context.Context, invocation Invocation) (Value, error) {
	if len(invocation.Arguments) == 0 {
		if currentFiber(ctx) != nil {
			return Null(), sleepBridgeIllegalArgument("expected &closure--received: " + Null().Describe())
		}
		return Null(), fmt.Errorf("&%s: missing callback argument", builtinName(invocation.Name))
	}

	handle, callbackIndex, err := state.chooseHandle(invocation, 2)
	if err != nil {
		return Null(), err
	}
	if callbackIndex >= len(invocation.Arguments) {
		if currentFiber(ctx) != nil {
			return Null(), sleepBridgeIllegalArgument("expected &closure--received: " + Null().Describe())
		}
		return Null(), fmt.Errorf("&%s: missing callback argument", builtinName(invocation.Name))
	}
	callback, err := sleepReadCallback(invocation, invocation.Arg(callbackIndex))
	if err != nil {
		if currentFiber(ctx) != nil {
			return Null(), sleepBridgeIllegalArgument("expected &closure--received: " + invocation.Arg(callbackIndex).Describe())
		}
		return Null(), fmt.Errorf("&%s: expected callback. received %s", builtinName(invocation.Name), invocation.Arg(callbackIndex).Describe())
	}

	chunkSize := int32(0)
	if callbackIndex+1 < len(invocation.Arguments) {
		chunkSize = sleepInt32(invocation.Arg(callbackIndex + 1))
	}
	if err := state.startSleepReadTask(ctx, invocation, handle, callback, chunkSize); err != nil {
		return Null(), err
	}
	return Null(), nil
}

func (state *ioBuiltinState) readb(ctx context.Context, invocation Invocation) (Value, error) {
	handle, argumentIndex, err := state.chooseHandle(invocation, 2)
	if err != nil {
		return Null(), sleepIOBridgeWarning(ctx, invocation, err)
	}
	count := int32(1)
	if argumentIndex < len(invocation.Arguments) {
		count = sleepInt32(invocation.Arg(argumentIndex))
	}
	if count < -1 {
		handle.mu.Lock()
		inputOpen := handle.reader != nil
		handle.mu.Unlock()
		if !inputOpen {
			return Null(), nil
		}
		// new byte[to] is inside BasicIO.readb's catch block. A negative
		// length closes the handle, populates checkError, and returns the empty
		// scalar without aborting the current Sleep block.
		_ = handle.close()
		negativeSize := fmt.Errorf("java.lang.NegativeArraySizeException: %d", count)
		if state == nil || state.runtime == nil {
			return Null(), negativeSize
		}
		return state.runtime.flagSourceError(invocation, negativeSize)
	}
	data, err := handle.readBytesContext(ctx, int(count))
	if err != nil {
		return Null(), preserveNativeBoundaryError(ctx,
			fmt.Errorf("&%s: read bytes: %w", builtinName(invocation.Name), err))
	}
	if len(data) == 0 {
		return Null(), nil
	}
	return BinaryString(data), nil
}

// consume implements both BasicIO.consume and its skip alias. The pinned Java
// bridge uses an int count and returns only a positive number of discarded
// bytes; zero, negative counts, closed inputs, and immediate EOF return the
// empty scalar. Read failures close the handle and populate checkError while a
// partial count remains the function result.
func (state *ioBuiltinState) consume(ctx context.Context, invocation Invocation) (Value, error) {
	handle, argumentIndex, err := state.chooseHandle(invocation, 2)
	if err != nil {
		return Null(), sleepIOBridgeWarning(ctx, invocation, err)
	}
	count := int32(1)
	if argumentIndex < len(invocation.Arguments) {
		count = sleepInt32(invocation.Arg(argumentIndex))
		argumentIndex++
	}
	bufferSize := int32(defaultIOBufferCapacity)
	if argumentIndex < len(invocation.Arguments) {
		bufferSize = sleepInt32(invocation.Arg(argumentIndex))
	}

	consumed, inputOpen, readErr := handle.consumeBytesContext(ctx, int(count), int(bufferSize))
	result := Null()
	if consumed > 0 {
		result = Int(int32(consumed))
	}
	if readErr == nil {
		return result, nil
	}
	if errors.Is(readErr, ErrResourceLimit) || errors.Is(readErr, context.Canceled) || errors.Is(readErr, context.DeadlineExceeded) {
		return result, readErr
	}
	if inputOpen && bufferSize < 0 {
		// The byte-array allocation is outside BasicIO.consume's catch block,
		// so NegativeArraySizeException follows the bridge-warning path rather
		// than the I/O soft-error slot.
		return Null(), sleepIOBridgeWarning(ctx, invocation, fmt.Errorf("&%s: %d", builtinName(invocation.Name), bufferSize))
	}

	// IOObject.close suppresses close failures. Preserve the read failure as a
	// Java-style soft error and retain any count completed before it occurred.
	_ = handle.close()
	readErr = preserveNativeBoundaryError(ctx, fmt.Errorf("java.io.IOException: %w", readErr))
	if state == nil || state.runtime == nil {
		return result, readErr
	}
	_, flaggedErr := state.runtime.flagSourceError(invocation, readErr)
	return result, flaggedErr
}

func (state *ioBuiltinState) writeb(ctx context.Context, invocation Invocation) (Value, error) {
	handle, dataIndex, err := state.chooseHandle(invocation, 2)
	if err != nil {
		return Null(), sleepIOBridgeWarning(ctx, invocation, err)
	}
	var data []byte
	if dataIndex < len(invocation.Arguments) {
		data = sleepStringLowBytes(invocation.Arg(dataIndex))
	}
	if _, err := handle.Write(data); err != nil {
		if errors.Is(err, ErrResourceLimit) {
			if process := handle.getProcess(); process != nil {
				_ = process.close()
			} else {
				_ = handle.close()
			}
		}
		return Null(), preserveNativeBoundaryError(ctx,
			fmt.Errorf("&%s: write bytes: %w", builtinName(invocation.Name), err))
	}
	return Null(), nil
}

func (state *ioBuiltinState) available(ctx context.Context, invocation Invocation) (Value, error) {
	handle, argumentIndex, err := state.chooseHandle(invocation, 1)
	if err != nil {
		// BasicIO.available catches every exception, including an invalid
		// chooseSource argument.
		return Null(), nil
	}
	if argumentIndex < len(invocation.Arguments) {
		contains, open, err := handle.availableContainsContext(ctx, string(sleepStringLowBytes(invocation.Arg(argumentIndex))))
		if errors.Is(err, ErrResourceLimit) || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return Null(), err
		}
		if err != nil || !open {
			return Null(), nil
		}
		return Bool(contains), nil
	}
	available, open, err := handle.availableBytes()
	if err != nil || !open {
		// BasicIO.available catches every exception from both available() and
		// its delimiter look-ahead path.
		return Null(), nil
	}
	return integerValue(available), nil
}

func (state *ioBuiltinState) endOfFile(ctx context.Context, invocation Invocation) (Value, error) {
	handle, err := state.handleArgument(invocation, 0, false)
	if err != nil {
		return Null(), sleepIOBridgeWarning(ctx, invocation, err)
	}
	return Bool(handle.isEOF()), nil
}

func (state *ioBuiltinState) mark(ctx context.Context, invocation Invocation) (Value, error) {
	handle, argumentIndex, err := state.chooseHandle(invocation, 2)
	if err != nil {
		return Null(), sleepIOBridgeWarning(ctx, invocation, err)
	}
	limit := int32(defaultIOMarkLimit)
	if argumentIndex < len(invocation.Arguments) {
		limit = sleepInt32(invocation.Arg(argumentIndex))
	}
	if err := handle.markInput(int(limit)); err != nil {
		warning := fmt.Errorf("&%s: %w", builtinName(invocation.Name), err)
		if currentFiber(ctx) != nil {
			// Unlike chooseSource's argument error, BasicIO.mark constructs its
			// closed-input exception with the function name already included.
			return Null(), &uncaughtScriptWarning{err: warning}
		}
		return Null(), warning
	}
	return Null(), nil
}

func (state *ioBuiltinState) reset(_ context.Context, invocation Invocation) (Value, error) {
	// BasicIO.reset catches every exception, including a bad handle, a closed
	// input, and an invalidated mark.
	handle, _, err := state.chooseHandle(invocation, 1)
	if err == nil {
		_ = handle.resetInput()
	}
	return Null(), nil
}

func (state *ioBuiltinState) printEOF(ctx context.Context, invocation Invocation) (Value, error) {
	handle, _, err := state.chooseHandle(invocation, 1)
	if err != nil {
		return Null(), err
	}
	if err := handle.closeWrite(); err != nil {
		return Null(), preserveNativeBoundaryError(ctx,
			fmt.Errorf("&%s: close output: %w", builtinName(invocation.Name), err))
	}
	return Null(), nil
}

func (state *ioBuiltinState) openf(ctx context.Context, invocation Invocation) (Value, error) {
	if len(invocation.Arguments) == 0 {
		if currentFiber(ctx) != nil {
			return Null(), sleepBridgeEmptyStack()
		}
		return Null(), fmt.Errorf("&%s: missing file descriptor", builtinName(invocation.Name))
	}
	descriptor := invocation.Arg(0).String()
	// BasicIO.openf constructs its FileObject before FileObject.open attempts
	// to interpret the descriptor. FileObject.open catches every descriptor and
	// filesystem exception through ScriptEnvironment.flagError, so even a
	// failed open returns an inert, non-null IOObject to the script.
	handle := newIOHandle(descriptor, nil, nil, false, false, false).withRuntimeOutputAccount(state.runtime.resources)
	if descriptor == "" {
		return state.flagOpenError(
			invocation,
			handle,
			errors.New("java.lang.StringIndexOutOfBoundsException: Index 0 out of bounds for length 0"),
		)
	}
	// FileObject.open probes descriptor.charAt(1) before its single-'>'
	// branch. Preserve the otherwise surprising failed-handle result for the
	// one-character write descriptor too.
	if descriptor == ">" {
		return state.flagOpenError(
			invocation,
			handle,
			errors.New("java.lang.StringIndexOutOfBoundsException: Index 1 out of bounds for length 1"),
		)
	}
	mode := byte('r')
	pathText := descriptor
	if strings.HasPrefix(descriptor, ">>") {
		mode = 'a'
		pathText = strings.TrimSpace(descriptor[2:])
	} else if strings.HasPrefix(descriptor, ">") {
		mode = 'w'
		pathText = strings.TrimSpace(descriptor[1:])
	}
	path := state.resolvePath(pathText)
	handle.label = path

	if mode == 'r' {
		file, err := os.Open(path)
		if err != nil {
			return state.flagOpenError(invocation, handle, portableFileNotFoundError(path, err))
		}
		info, err := file.Stat()
		if err != nil {
			_ = file.Close()
			return state.flagOpenError(invocation, handle, portableFileNotFoundError(path, err))
		}
		if info.IsDir() {
			// FileInputStream rejects directories even on platforms where
			// os.Open permits reading their directory descriptor. Close that
			// descriptor and preserve Java's portable FileNotFoundException
			// detail while returning the inert handle created above.
			_ = file.Close()
			return state.flagOpenError(
				invocation,
				handle,
				portableFileNotFoundError(path, errors.New("Is a directory")),
			)
		}
		return ObjectValue(newIOHandle(path, file, nil, true, false, false).withRuntimeOutputAccount(state.runtime.resources)), nil
	}

	flags := os.O_WRONLY | os.O_CREATE
	if mode == 'a' {
		flags |= os.O_APPEND
	} else {
		flags |= os.O_TRUNC
	}
	file, err := os.OpenFile(path, flags, 0o666)
	if err != nil {
		return state.flagOpenError(invocation, handle, portableFileNotFoundError(path, err))
	}
	return ObjectValue(newIOHandle(path, nil, file, false, true, false).withRuntimeOutputAccount(state.runtime.resources)), nil
}

func (state *ioBuiltinState) flagOpenError(invocation Invocation, handle *sleepIOHandle, message error) (Value, error) {
	value := ObjectValue(handle)
	if state == nil || state.runtime == nil {
		return value, message
	}
	_, err := state.runtime.flagSourceError(invocation, message)
	return value, err
}

func portableFileNotFoundError(path string, err error) error {
	detail := "File could not be opened"
	if errors.Is(err, os.ErrNotExist) {
		detail = "No such file or directory"
	} else if errors.Is(err, os.ErrPermission) {
		detail = "Permission denied"
	} else {
		var pathError *os.PathError
		if errors.As(err, &pathError) && pathError.Err != nil {
			detail = pathError.Err.Error()
			if detail != "" {
				detail = strings.ToUpper(detail[:1]) + detail[1:]
			}
		} else if err != nil {
			detail = err.Error()
		}
	}
	return fmt.Errorf("java.io.FileNotFoundException: %s (%s)", path, detail)
}

func (state *ioBuiltinState) sleep(ctx context.Context, invocation Invocation) (Value, error) {
	milliseconds := invocation.Arg(0).Int64()
	if milliseconds <= 0 {
		return Null(), nil
	}
	if milliseconds > int64((time.Duration(1<<63-1))/time.Millisecond) {
		return Null(), fmt.Errorf("&%s: duration %dms exceeds platform limit", builtinName(invocation.Name), milliseconds)
	}
	timer := time.NewTimer(time.Duration(milliseconds) * time.Millisecond)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return Null(), ctx.Err()
	case <-timer.C:
		return Null(), nil
	}
}

func (state *ioBuiltinState) currentDirectory(_ context.Context, _ Invocation) (Value, error) {
	return String(state.workingDirectory()), nil
}

func (state *ioBuiltinState) chdir(_ context.Context, invocation Invocation) (Value, error) {
	path := state.pathArgument(invocation, 0)
	absolute, err := sleepFileAbsolute(path)
	if err != nil {
		return Null(), fmt.Errorf("&%s: make %s absolute: %w", builtinName(invocation.Name), path, err)
	}
	state.cwdMu.Lock()
	state.cwd = absolute
	state.cwdMu.Unlock()
	if state.runtime != nil && state.runtime.defaultFileResolver != nil {
		state.runtime.defaultFileResolver.setBaseDirectory(absolute)
	}
	return Null(), nil
}

func (state *ioBuiltinState) listFiles(ctx context.Context, invocation Invocation) (Value, error) {
	if invocation.Name == "listRoots" && len(invocation.Arguments) == 0 {
		roots := sleepFilesystemRoots()
		if err := reserveCollectionEntries(invocation.Runtime, len(roots)); err != nil {
			return Null(), err
		}
		values := make([]Value, len(roots))
		for index, root := range roots {
			values[index] = String(root)
		}
		return ArrayValue(NewReadOnlyArray(values...)), nil
	}
	path := state.pathArgument(invocation, 0)
	directory, err := os.Open(path)
	if err != nil {
		// File.listFiles returns null for an inaccessible, missing, or non-directory
		// path; FileSystemBridge converts that into an empty CollectionWrapper.
		return ArrayValue(NewReadOnlyArray()), nil
	}
	defer directory.Close()
	var values []Value
	for {
		entries, readErr := directory.ReadDir(128)
		if err := executionContextError(ctx); err != nil {
			return Null(), err
		}
		if err := reserveCollectionEntries(invocation.Runtime, len(entries)); err != nil {
			return Null(), err
		}
		for _, entry := range entries {
			child := portableJavaFileResolve(path, portableJavaFileNormalize(entry.Name()))
			absolute, absoluteErr := sleepFileAbsolute(child)
			if absoluteErr != nil {
				return ArrayValue(NewReadOnlyArray()), nil
			}
			values = append(values, String(absolute))
		}
		if errors.Is(readErr, io.EOF) {
			break
		}
		if readErr != nil {
			return ArrayValue(NewReadOnlyArray()), nil
		}
	}
	// os.ReadDir(path) sorted the previous implementation by filename. Preserve
	// that observable order after bounded streaming admission.
	sort.Slice(values, func(left, right int) bool { return values[left].String() < values[right].String() })
	return ArrayValue(NewReadOnlyArray(values...)), nil
}

func (state *ioBuiltinState) lengthOfFile(_ context.Context, invocation Invocation) (Value, error) {
	path := state.pathArgument(invocation, 0)
	info, err := os.Stat(path)
	if err != nil {
		return Long(0), nil
	}
	return Long(info.Size()), nil
}

// createNewFile mirrors java.io.File.createNewFile: creation is atomic, an
// existing path returns the empty scalar, and other I/O failures populate
// Sleep's soft-error slot.
func (state *ioBuiltinState) createNewFile(_ context.Context, invocation Invocation) (Value, error) {
	path := state.pathArgument(invocation, 0)
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o666)
	if err == nil {
		if closeErr := file.Close(); closeErr != nil {
			return state.flagFilesystemError(invocation, closeErr)
		}
		return Bool(true), nil
	}
	if errors.Is(err, os.ErrExist) {
		return Null(), nil
	}
	return state.flagFilesystemError(invocation, err)
}

func (state *ioBuiltinState) flagFilesystemError(invocation Invocation, err error) (Value, error) {
	if state == nil || state.runtime == nil {
		return Null(), err
	}
	return state.runtime.flagSourceError(invocation, err)
}

func (state *ioBuiltinState) mkdir(_ context.Context, invocation Invocation) (Value, error) {
	path := state.pathArgument(invocation, 0)
	if _, err := os.Stat(path); err == nil {
		return Null(), nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return Null(), nil
	}
	if err := os.MkdirAll(path, 0o777); err != nil {
		return Null(), nil
	}
	return Bool(true), nil
}

func (state *ioBuiltinState) deleteFile(_ context.Context, invocation Invocation) (Value, error) {
	path := state.pathArgument(invocation, 0)
	if err := os.Remove(path); err != nil {
		return Null(), nil
	}
	return Bool(true), nil
}

func (state *ioBuiltinState) move(_ context.Context, invocation Invocation) (Value, error) {
	source := state.pathArgument(invocation, 0)
	destination := state.pathArgument(invocation, 1)
	if err := os.Rename(source, destination); err != nil {
		return Null(), nil
	}
	return Bool(true), nil
}

func (state *ioBuiltinState) copyFile(ctx context.Context, invocation Invocation) (Value, error) {
	if len(invocation.Arguments) < 2 {
		return Null(), fmt.Errorf("&%s: expected source and destination paths", builtinName(invocation.Name))
	}
	source := state.pathArgument(invocation, 0)
	destination := state.pathArgument(invocation, 1)
	if err := copyFileContents(ctx, source, destination, state.runtime.resources); err != nil {
		return Null(), fmt.Errorf("&%s: copy %s to %s: %w", builtinName(invocation.Name), source, destination, err)
	}
	return Bool(true), nil
}

func (state *ioBuiltinState) dirname(_ context.Context, invocation Invocation) (Value, error) {
	parent, ok := portableJavaFileParent(state.pathArgument(invocation, 0))
	if !ok {
		return Null(), nil
	}
	return String(parent), nil
}

func (state *ioBuiltinState) getFileName(_ context.Context, invocation Invocation) (Value, error) {
	return String(portableJavaFileName(state.pathArgument(invocation, 0))), nil
}

// getFileProper is Sleep's portable path constructor. FileSystemBridge starts
// with BridgeUtilities.getFile (which resolves a relative first component
// against the script cwd), appends every remaining component, and returns the
// resulting absolute path without requiring it to exist.
func (state *ioBuiltinState) getFileProper(_ context.Context, invocation Invocation) (Value, error) {
	path := state.workingDirectory()
	if len(invocation.Arguments) != 0 {
		first := portableJavaFileNormalize(invocation.Arg(0).String())
		if first == "" {
			// Preserve Java File's empty abstract pathname. If a child follows,
			// the File(parent, child) constructor resolves it from the platform's
			// default parent rather than from the ScriptInstance cwd.
			path = ""
		} else if filepath.IsAbs(first) {
			path = first
		} else {
			path = sleepFileResolve(state.workingDirectory(), first)
		}
	}
	for index := 1; index < len(invocation.Arguments); index++ {
		path = sleepFileResolve(path, portableJavaFileNormalize(invocation.Arg(index).String()))
	}
	absolute, err := sleepFileAbsolute(path)
	if err != nil {
		return Null(), fmt.Errorf("&%s: make %s absolute: %w", builtinName(invocation.Name), path, err)
	}
	return String(absolute), nil
}

// sleepFileResolve follows java.io.File(File, String) instead of filepath.Join.
// The Java constructor deliberately retains dot segments and, on Unix, joins
// a leading-slash child to a non-root parent rather than replacing the parent.
func sleepFileResolve(parent, child string) string {
	if goruntime.GOOS == "windows" {
		if parent == "" {
			return string(filepath.Separator) + strings.TrimLeft(child, `/\`)
		}
		// filepath.Join captures Windows drive and UNC rules. Its cleaning of
		// dot segments is the only residual representational difference here;
		// all filesystem effects resolve those segments identically.
		return filepath.Join(parent, child)
	}
	if child == "" {
		return parent
	}
	if parent == "" {
		parent = string(filepath.Separator)
	}
	if parent == string(filepath.Separator) {
		return parent + strings.TrimPrefix(child, string(filepath.Separator))
	}
	return strings.TrimSuffix(parent, string(filepath.Separator)) + string(filepath.Separator) +
		strings.TrimPrefix(child, string(filepath.Separator))
}

func sleepFileAbsolute(path string) (string, error) {
	if goruntime.GOOS == "windows" {
		return filepath.Abs(path)
	}
	if path == "" {
		return os.Getwd()
	}
	if filepath.IsAbs(path) {
		return path, nil
	}
	workingDirectory, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return sleepFileResolve(workingDirectory, path), nil
}

func (state *ioBuiltinState) lastModified(_ context.Context, invocation Invocation) (Value, error) {
	info, err := os.Stat(state.pathArgument(invocation, 0))
	if err != nil {
		// java.io.File.lastModified reports zero for a missing or inaccessible
		// path rather than raising an I/O exception.
		return Long(0), nil
	}
	return Long(info.ModTime().UnixMilli()), nil
}

func (state *ioBuiltinState) setLastModified(_ context.Context, invocation Invocation) (Value, error) {
	path := state.pathArgument(invocation, 0)
	milliseconds := invocation.Arg(1).Int64()
	if milliseconds < 0 {
		return Null(), errors.New("java.lang.IllegalArgumentException: Negative time")
	}
	info, err := os.Stat(path)
	if err != nil {
		return Null(), nil
	}
	if err := os.Chtimes(path, info.ModTime(), time.UnixMilli(milliseconds)); err != nil {
		return Null(), nil
	}
	return Bool(true), nil
}

func (state *ioBuiltinState) setReadOnly(_ context.Context, invocation Invocation) (Value, error) {
	path := state.pathArgument(invocation, 0)
	info, err := os.Stat(path)
	if err != nil {
		return Null(), nil
	}
	mode := info.Mode() & (os.ModePerm | os.ModeSetuid | os.ModeSetgid | os.ModeSticky)
	if err := os.Chmod(path, mode&^0o222); err != nil {
		return Null(), nil
	}
	return Bool(true), nil
}

func sleepFilesystemRoots() []string {
	if goruntime.GOOS != "windows" {
		return []string{string(filepath.Separator)}
	}
	roots := make([]string, 0, 26)
	for drive := 'A'; drive <= 'Z'; drive++ {
		root := fmt.Sprintf("%c:%c", drive, filepath.Separator)
		if _, err := os.Stat(root); err == nil {
			roots = append(roots, root)
		}
	}
	return roots
}

func (state *ioBuiltinState) filePredicate(_ context.Context, invocation Invocation) (Value, error) {
	// BridgeUtilities.toSleepFile treats an explicit empty string as new
	// File("") rather than the script cwd. File("").exists() is false; only a
	// missing argument defaults to cwd.
	if len(invocation.Arguments) != 0 && invocation.Arg(0).String() == "" {
		return Bool(false), nil
	}
	path := state.pathArgument(invocation, 0)
	if strings.EqualFold(invocation.Name, "-isHidden") {
		name := portableJavaFileName(path)
		if goruntime.GOOS != "windows" {
			return Bool(strings.HasPrefix(name, ".")), nil
		}
		_, err := os.Stat(path)
		return Bool(err == nil && strings.HasPrefix(name, ".")), nil
	}
	info, err := os.Stat(path)
	if err != nil {
		return Bool(false), nil
	}
	switch strings.ToLower(invocation.Name) {
	case "-exists", "-e":
		return Bool(true), nil
	case "-isdir", "-d":
		return Bool(info.IsDir()), nil
	case "-isfile", "-f":
		return Bool(info.Mode().IsRegular()), nil
	case "-canread":
		return Bool(info.Mode().Perm()&0o444 != 0), nil
	case "-canwrite":
		return Bool(info.Mode().Perm()&0o222 != 0), nil
	default:
		return Bool(false), nil
	}
}

func (state *ioBuiltinState) execute(ctx context.Context, invocation Invocation) (Value, error) {
	if invocation.Name == "exec" {
		return state.executeProcess(ctx, invocation)
	}
	command, shell, err := commandArguments(invocation)
	if err != nil {
		return Null(), err
	}

	var process *osexec.Cmd
	if shell {
		process = shellCommand(ctx, command[0])
	} else {
		process = osexec.CommandContext(ctx, command[0], command[1:]...)
	}
	process.Dir = state.workingDirectory()
	// BasicIO.__EXEC__ never connects the script console to the child process's
	// input stream. Keep backticks from consuming importer-supplied stdin.
	var stdout, stderr bytes.Buffer
	account := runtimeOutputAccountFor(ctx, state.runtime)
	stdoutWriter := newRuntimeOutputWriter(account, &stdout)
	stderrWriter := newRuntimeOutputWriter(account, &stderr)
	process.Stdout = stdoutWriter
	process.Stderr = stderrWriter
	runErr := process.Run()
	outputErr := errors.Join(stdoutWriter.LimitError(), stderrWriter.LimitError())
	lineCount := outputLineCount(stdout.Bytes())
	if arrayErr := reserveCollectionEntries(invocation.Runtime, lineCount); arrayErr != nil {
		return Null(), errors.Join(outputErr, arrayErr)
	}
	output := outputLines(stdout.Bytes())
	array := NewArray(output...)
	result := ArrayValue(array)
	if outputErr != nil {
		return result, outputErr
	}
	if runErr == nil {
		return result, nil
	}
	if ctxErr := ctx.Err(); ctxErr != nil {
		return result, ctxErr
	}

	name := builtinName(invocation.Name)
	var exitError *osexec.ExitError
	if errors.As(runErr, &exitError) {
		if invocation.Name == "__EXEC__" {
			return state.flagBacktickError(invocation, result, fmt.Errorf("abnormal termination: %d", exitError.ExitCode()))
		}
		detail := strings.TrimSpace(stderr.String())
		if detail == "" {
			detail = strings.TrimSpace(stdout.String())
		}
		if detail != "" {
			return result, fmt.Errorf("&%s: command exited with status %d: %s", name, exitError.ExitCode(), detail)
		}
		return result, fmt.Errorf("&%s: command exited with status %d", name, exitError.ExitCode())
	}
	if invocation.Name == "__EXEC__" {
		return state.flagBacktickError(invocation, result, runErr)
	}
	return result, fmt.Errorf("&%s: start command: %w", name, runErr)
}

func (state *ioBuiltinState) flagBacktickError(invocation Invocation, result Value, err error) (Value, error) {
	if state == nil || state.runtime == nil {
		return result, err
	}
	_, flaggedErr := state.runtime.flagSourceError(invocation, err)
	return result, flaggedErr
}

func (state *ioBuiltinState) handleArgument(invocation Invocation, index int, defaultConsole bool) (*sleepIOHandle, error) {
	if index >= len(invocation.Arguments) {
		if defaultConsole {
			return state.console, nil
		}
		return nil, fmt.Errorf("&%s: missing I/O handle argument", builtinName(invocation.Name))
	}
	handle, ok := ioHandleValue(invocation.Arg(index))
	if !ok {
		return nil, fmt.Errorf("&%s: expected I/O handle argument, received: %s", builtinName(invocation.Name), invocation.Arg(index).Describe())
	}
	return handle, nil
}

// sleepIOBridgeWarning mirrors the evaluator's handling of an exception thrown
// directly by a Sleep bridge: script execution emits a warning and aborts only
// the active block, while direct Go invocation receives an ordinary error.
func sleepIOBridgeWarning(ctx context.Context, invocation Invocation, err error) error {
	if err == nil || currentFiber(ctx) == nil {
		return err
	}
	prefix := "&" + builtinName(invocation.Name) + ": "
	message := strings.TrimPrefix(err.Error(), prefix)
	return &uncaughtScriptWarning{err: errors.New(message)}
}

// chooseHandle mirrors Sleep BasicIO's chooseSource convention. Calls with at
// least functionArity arguments require an explicit leading handle. Shorter
// calls use a leading handle when present and otherwise fall back to console.
// The returned index identifies the first argument after the selected handle.
func (state *ioBuiltinState) chooseHandle(invocation Invocation, functionArity int) (*sleepIOHandle, int, error) {
	if len(invocation.Arguments) >= functionArity {
		handle, err := state.handleArgument(invocation, 0, false)
		return handle, 1, err
	}
	if len(invocation.Arguments) != 0 {
		if handle, ok := ioHandleValue(invocation.Arg(0)); ok {
			return handle, 1, nil
		}
	}
	return state.console, 0, nil
}

func ioHandleValue(value Value) (*sleepIOHandle, bool) {
	object, ok := value.Object()
	if !ok {
		return nil, false
	}
	handle, ok := object.(*sleepIOHandle)
	return handle, ok && handle != nil
}

func (state *ioBuiltinState) workingDirectory() string {
	state.cwdMu.RLock()
	defer state.cwdMu.RUnlock()
	return state.cwd
}

func (state *ioBuiltinState) pathArgument(invocation Invocation, index int) string {
	if index >= len(invocation.Arguments) {
		return state.workingDirectory()
	}
	path := portableJavaFileNormalize(invocation.Arg(index).String())
	if path == "" || filepath.IsAbs(path) {
		return path
	}
	return portableJavaFileResolve(state.workingDirectory(), path)
}

func (state *ioBuiltinState) resolvePath(path string) string {
	path = filepath.FromSlash(path)
	if path == "" {
		return state.workingDirectory()
	}
	if filepath.IsAbs(path) {
		return filepath.Clean(path)
	}
	return filepath.Clean(filepath.Join(state.workingDirectory(), path))
}

func copyFileContents(ctx context.Context, source, destination string, outputAccount *runtimeResourceAccount) error {
	sourceInfo, err := os.Stat(source)
	if err != nil {
		return err
	}
	if !sourceInfo.Mode().IsRegular() {
		return fmt.Errorf("source is not a regular file")
	}
	if destinationInfo, statErr := os.Stat(destination); statErr == nil {
		if os.SameFile(sourceInfo, destinationInfo) {
			return fmt.Errorf("source and destination are the same file")
		}
	} else if !errors.Is(statErr, os.ErrNotExist) {
		return statErr
	}

	input, err := os.Open(source)
	if err != nil {
		return err
	}
	output, err := os.OpenFile(destination, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, sourceInfo.Mode().Perm())
	if err != nil {
		return errors.Join(err, input.Close())
	}
	_, copyErr := io.Copy(newRuntimeOutputWriter(outputAccount, output), contextCheckingReader{ctx: ctx, reader: input})
	return errors.Join(copyErr, output.Close(), input.Close())
}

type contextCheckingReader struct {
	ctx    context.Context
	reader io.Reader
}

func (reader contextCheckingReader) Read(data []byte) (int, error) {
	if reader.ctx != nil {
		if err := reader.ctx.Err(); err != nil {
			return 0, err
		}
	}
	amount, err := reader.reader.Read(data)
	if reader.ctx != nil {
		if contextErr := reader.ctx.Err(); contextErr != nil {
			return amount, contextErr
		}
	}
	return amount, err
}

func commandArguments(invocation Invocation) ([]string, bool, error) {
	name := builtinName(invocation.Name)
	if len(invocation.Arguments) == 0 {
		return nil, false, fmt.Errorf("&%s: missing command", name)
	}

	command := invocation.Arg(0).String()
	if command == "" {
		return nil, false, fmt.Errorf("&%s: command is empty", name)
	}
	return []string{command}, true, nil
}

func shellCommand(ctx context.Context, command string) *osexec.Cmd {
	if goruntime.GOOS == "windows" {
		shell := os.Getenv("COMSPEC")
		if shell == "" {
			shell = "cmd.exe"
		}
		return osexec.CommandContext(ctx, shell, "/C", command)
	}
	shell, err := osexec.LookPath("sh")
	if err != nil {
		shell = "/bin/sh"
	}
	return osexec.CommandContext(ctx, shell, "-c", command)
}

func outputLines(output []byte) []Value {
	if len(output) == 0 {
		return nil
	}
	normalized := strings.ReplaceAll(string(output), "\r\n", "\n")
	normalized = strings.ReplaceAll(normalized, "\r", "\n")
	lines := strings.Split(normalized, "\n")
	if len(lines) != 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	values := make([]Value, len(lines))
	for index, line := range lines {
		values[index] = String(line)
	}
	return values
}

// outputLineCount mirrors outputLines' CRLF/bare-CR normalization and trailing
// empty removal without materializing a normalized string, split slice, or
// Values. Backtick execution can therefore reserve its returned collection
// entries before those proportional allocations.
func outputLineCount(output []byte) int {
	if len(output) == 0 {
		return 0
	}
	count := 1
	endsWithBreak := false
	for index := 0; index < len(output); index++ {
		switch output[index] {
		case '\r':
			count++
			endsWithBreak = true
			if index+1 < len(output) && output[index+1] == '\n' {
				index++
			}
		case '\n':
			count++
			endsWithBreak = true
		default:
			endsWithBreak = false
		}
	}
	if endsWithBreak {
		count--
	}
	return count
}

func integerValue(value int64) Value {
	if value >= -1<<31 && value <= 1<<31-1 {
		return Int(int32(value))
	}
	return Long(value)
}

func maximumInt() int {
	return int(^uint(0) >> 1)
}
