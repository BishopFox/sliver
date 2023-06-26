package compiler

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"runtime"

	"github.com/tetratelabs/wazero/experimental"
	"github.com/tetratelabs/wazero/internal/platform"
	"github.com/tetratelabs/wazero/internal/u32"
	"github.com/tetratelabs/wazero/internal/u64"
	"github.com/tetratelabs/wazero/internal/wasm"
)

func (e *engine) deleteCompiledModule(module *wasm.Module) {
	e.mux.Lock()
	defer e.mux.Unlock()
	delete(e.codes, module.ID)

	// Note: we do not call e.Cache.Delete, as the lifetime of
	// the content is up to the implementation of extencache.Cache interface.
}

func (e *engine) addCompiledModule(module *wasm.Module, cm *compiledModule, withGoFunc bool) (err error) {
	e.addCompiledModuleToMemory(module, cm)
	if !withGoFunc {
		err = e.addCompiledModuleToCache(module, cm)
	}
	return
}

func (e *engine) getCompiledModule(module *wasm.Module, listeners []experimental.FunctionListener) (cm *compiledModule, ok bool, err error) {
	cm, ok = e.getCompiledModuleFromMemory(module)
	if ok {
		return
	}
	cm, ok, err = e.getCompiledModuleFromCache(module)
	if ok {
		e.addCompiledModuleToMemory(module, cm)
		if len(listeners) > 0 {
			// Files do not contain the actual listener instances (it's impossible to cache them as files!), so assign each here.
			for i := range cm.functions {
				cm.functions[i].listener = listeners[i]
			}
		}
	}
	return
}

func (e *engine) addCompiledModuleToMemory(module *wasm.Module, cm *compiledModule) {
	e.mux.Lock()
	defer e.mux.Unlock()
	e.codes[module.ID] = cm
}

func (e *engine) getCompiledModuleFromMemory(module *wasm.Module) (cm *compiledModule, ok bool) {
	e.mux.RLock()
	defer e.mux.RUnlock()
	cm, ok = e.codes[module.ID]
	return
}

func (e *engine) addCompiledModuleToCache(module *wasm.Module, cm *compiledModule) (err error) {
	if e.fileCache == nil || module.IsHostModule {
		return
	}
	err = e.fileCache.Add(module.ID, serializeCompiledModule(e.wazeroVersion, cm))
	return
}

func (e *engine) getCompiledModuleFromCache(module *wasm.Module) (cm *compiledModule, hit bool, err error) {
	if e.fileCache == nil || module.IsHostModule {
		return
	}

	// Check if the entries exist in the external cache.
	var cached io.ReadCloser
	cached, hit, err = e.fileCache.Get(module.ID)
	if !hit || err != nil {
		return
	}

	// Otherwise, we hit the cache on external cache.
	// We retrieve *code structures from `cached`.
	var staleCache bool
	// Note: cached.Close is ensured to be called in deserializeCodes.
	cm, staleCache, err = deserializeCompiledModule(e.wazeroVersion, cached, module)
	if err != nil {
		hit = false
		return
	} else if staleCache {
		return nil, false, e.fileCache.Delete(module.ID)
	}

	cm.source = module
	return
}

var wazeroMagic = "WAZERO" // version must be synced with the tag of the wazero library.

func serializeCompiledModule(wazeroVersion string, cm *compiledModule) io.Reader {
	buf := bytes.NewBuffer(nil)
	// First 6 byte: WAZERO header.
	buf.WriteString(wazeroMagic)
	// Next 1 byte: length of version:
	buf.WriteByte(byte(len(wazeroVersion)))
	// Version of wazero.
	buf.WriteString(wazeroVersion)
	if cm.ensureTermination {
		buf.WriteByte(1)
	} else {
		buf.WriteByte(0)
	}
	// Number of *code (== locally defined functions in the module): 4 bytes.
	buf.Write(u32.LeBytes(uint32(len(cm.functions))))
	for i := 0; i < len(cm.functions); i++ {
		f := &cm.functions[i]
		// The stack pointer ceil (8 bytes).
		buf.Write(u64.LeBytes(f.stackPointerCeil))
		// The offset of this function in the executable (8 bytes).
		buf.Write(u64.LeBytes(uint64(f.executableOffset)))
	}
	// The length of code segment (8 bytes).
	buf.Write(u64.LeBytes(uint64(cm.executable.Len())))
	// Append the native code.
	buf.Write(cm.executable.Bytes())
	return bytes.NewReader(buf.Bytes())
}

func deserializeCompiledModule(wazeroVersion string, reader io.ReadCloser, module *wasm.Module) (cm *compiledModule, staleCache bool, err error) {
	defer reader.Close()
	cacheHeaderSize := len(wazeroMagic) + 1 /* version size */ + len(wazeroVersion) + 1 /* ensure termination */ + 4 /* number of functions */

	// Read the header before the native code.
	header := make([]byte, cacheHeaderSize)
	n, err := reader.Read(header)
	if err != nil {
		return nil, false, fmt.Errorf("compilationcache: error reading header: %v", err)
	}

	if n != cacheHeaderSize {
		return nil, false, fmt.Errorf("compilationcache: invalid header length: %d", n)
	}

	// Check the version compatibility.
	versionSize := int(header[len(wazeroMagic)])

	cachedVersionBegin, cachedVersionEnd := len(wazeroMagic)+1, len(wazeroMagic)+1+versionSize
	if cachedVersionEnd >= len(header) {
		staleCache = true
		return
	} else if cachedVersion := string(header[cachedVersionBegin:cachedVersionEnd]); cachedVersion != wazeroVersion {
		staleCache = true
		return
	}

	ensureTermination := header[cachedVersionEnd] != 0
	functionsNum := binary.LittleEndian.Uint32(header[len(header)-4:])
	cm = &compiledModule{functions: make([]compiledFunction, functionsNum), ensureTermination: ensureTermination}

	imported := module.ImportFunctionCount

	var eightBytes [8]byte
	for i := uint32(0); i < functionsNum; i++ {
		f := &cm.functions[i]
		f.parent = cm

		// Read the stack pointer ceil.
		if f.stackPointerCeil, err = readUint64(reader, &eightBytes); err != nil {
			err = fmt.Errorf("compilationcache: error reading func[%d] stack pointer ceil: %v", i, err)
			return
		}

		// Read the offset of each function in the executable.
		var offset uint64
		if offset, err = readUint64(reader, &eightBytes); err != nil {
			err = fmt.Errorf("compilationcache: error reading func[%d] executable offset: %v", i, err)
			return
		}
		f.executableOffset = uintptr(offset)
		f.index = imported + i
	}

	executableLen, err := readUint64(reader, &eightBytes)
	if err != nil {
		err = fmt.Errorf("compilationcache: error reading executable size: %v", err)
		return
	}

	if executableLen > 0 {
		if err = cm.executable.Map(int(executableLen)); err != nil {
			err = fmt.Errorf("compilationcache: error mmapping executable (len=%d): %v", executableLen, err)
			return
		}

		_, err = io.ReadFull(reader, cm.executable.Bytes())
		if err != nil {
			err = fmt.Errorf("compilationcache: error reading executable (len=%d): %v", executableLen, err)
			return
		}

		if runtime.GOARCH == "arm64" {
			// On arm64, we cannot give all of rwx at the same time, so we change it to exec.
			if err = platform.MprotectRX(cm.executable.Bytes()); err != nil {
				return
			}
		}
	}
	return
}

// readUint64 strictly reads an uint64 in little-endian byte order, using the
// given array as a buffer. This returns io.EOF if less than 8 bytes were read.
func readUint64(reader io.Reader, b *[8]byte) (uint64, error) {
	s := b[0:8]
	n, err := reader.Read(s)
	if err != nil {
		return 0, err
	} else if n < 8 { // more strict than reader.Read
		return 0, io.EOF
	}

	// Read the u64 from the underlying buffer.
	ret := binary.LittleEndian.Uint64(s)

	// Clear the underlying array.
	for i := 0; i < 8; i++ {
		b[i] = 0
	}
	return ret, nil
}
