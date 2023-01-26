package wazero

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"math"
	"time"

	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/internal/engine/compiler"
	"github.com/tetratelabs/wazero/internal/engine/interpreter"
	"github.com/tetratelabs/wazero/internal/platform"
	internalsys "github.com/tetratelabs/wazero/internal/sys"
	"github.com/tetratelabs/wazero/internal/wasm"
	"github.com/tetratelabs/wazero/sys"
)

// RuntimeConfig controls runtime behavior, with the default implementation as
// NewRuntimeConfig
//
// The example below explicitly limits to Wasm Core 1.0 features as opposed to
// relying on defaults:
//
//	rConfig = wazero.NewRuntimeConfig().WithCoreFeatures(api.CoreFeaturesV1)
//
// Note: RuntimeConfig is immutable. Each WithXXX function returns a new
// instance including the corresponding change.
type RuntimeConfig interface {
	// WithCoreFeatures sets the WebAssembly Core specification features this
	// runtime supports. Defaults to api.CoreFeaturesV2.
	//
	// Example of disabling a specific feature:
	//	features := api.CoreFeaturesV2.SetEnabled(api.CoreFeatureMutableGlobal, false)
	//	rConfig = wazero.NewRuntimeConfig().WithCoreFeatures(features)
	//
	// # Why default to version 2.0?
	//
	// Many compilers that target WebAssembly require features after
	// api.CoreFeaturesV1 by default. For example, TinyGo v0.24+ requires
	// api.CoreFeatureBulkMemoryOperations. To avoid runtime errors, wazero
	// defaults to api.CoreFeaturesV2, even though it is not yet a Web
	// Standard (REC).
	WithCoreFeatures(api.CoreFeatures) RuntimeConfig

	// WithMemoryLimitPages overrides the maximum pages allowed per memory. The
	// default is 65536, allowing 4GB total memory per instance. Setting a
	// value larger than default will panic.
	//
	// This example reduces the largest possible memory size from 4GB to 128KB:
	//	rConfig = wazero.NewRuntimeConfig().WithMemoryLimitPages(2)
	//
	// Note: Wasm has 32-bit memory and each page is 65536 (2^16) bytes. This
	// implies a max of 65536 (2^16) addressable pages.
	// See https://www.w3.org/TR/2019/REC-wasm-core-1-20191205/#grow-mem
	WithMemoryLimitPages(memoryLimitPages uint32) RuntimeConfig

	// WithMemoryCapacityFromMax eagerly allocates max memory, unless max is
	// not defined. The default is false, which means minimum memory is
	// allocated and any call to grow memory results in re-allocations.
	//
	// This example ensures any memory.grow instruction will never re-allocate:
	//	rConfig = wazero.NewRuntimeConfig().WithMemoryCapacityFromMax(true)
	//
	// See https://www.w3.org/TR/2019/REC-wasm-core-1-20191205/#grow-mem
	WithMemoryCapacityFromMax(memoryCapacityFromMax bool) RuntimeConfig

	// WithDebugInfoEnabled toggles DWARF based stack traces in the face of
	// runtime errors. Defaults to true.
	//
	// Those who wish to disable this, can like so:
	//
	//	r := wazero.NewRuntimeWithConfig(wazero.NewRuntimeConfig().WithDebugInfoEnabled(false)
	//
	// When disabled, a stack trace message looks like:
	//
	//	wasm stack trace:
	//		.runtime._panic(i32)
	//		.myFunc()
	//		.main.main()
	//		.runtime.run()
	//		._start()
	//
	// When enabled, the stack trace includes source code information:
	//
	//	wasm stack trace:
	//		.runtime._panic(i32)
	//		  0x16e2: /opt/homebrew/Cellar/tinygo/0.26.0/src/runtime/runtime_tinygowasm.go:73:6
	//		.myFunc()
	//		  0x190b: /Users/XXXXX/wazero/internal/testing/dwarftestdata/testdata/main.go:19:7
	//		.main.main()
	//		  0x18ed: /Users/XXXXX/wazero/internal/testing/dwarftestdata/testdata/main.go:4:3
	//		.runtime.run()
	//		  0x18cc: /opt/homebrew/Cellar/tinygo/0.26.0/src/runtime/scheduler_none.go:26:10
	//		._start()
	//		  0x18b6: /opt/homebrew/Cellar/tinygo/0.26.0/src/runtime/runtime_wasm_wasi.go:22:5
	//
	// Note: This only takes into effect when the original Wasm binary has the
	// DWARF "custom sections" that are often stripped, depending on
	// optimization flags passed to the compiler.
	WithDebugInfoEnabled(bool) RuntimeConfig
}

// NewRuntimeConfig returns a RuntimeConfig using the compiler if it is supported in this environment,
// or the interpreter otherwise.
func NewRuntimeConfig() RuntimeConfig {
	return newRuntimeConfig()
}

type runtimeConfig struct {
	enabledFeatures       api.CoreFeatures
	memoryLimitPages      uint32
	memoryCapacityFromMax bool
	isInterpreter         bool
	dwarfDisabled         bool // negative as defaults to enabled
	newEngine             func(context.Context, api.CoreFeatures) wasm.Engine
}

// engineLessConfig helps avoid copy/pasting the wrong defaults.
var engineLessConfig = &runtimeConfig{
	enabledFeatures:       api.CoreFeaturesV2,
	memoryLimitPages:      wasm.MemoryLimitPages,
	memoryCapacityFromMax: false,
	dwarfDisabled:         false,
}

// NewRuntimeConfigCompiler compiles WebAssembly modules into
// runtime.GOARCH-specific assembly for optimal performance.
//
// The default implementation is AOT (Ahead of Time) compilation, applied at
// Runtime.CompileModule. This allows consistent runtime performance, as well
// the ability to reduce any first request penalty.
//
// Note: While this is technically AOT, this does not imply any action on your
// part. wazero automatically performs ahead-of-time compilation as needed when
// Runtime.CompileModule is invoked.
//
// Warning: This panics at runtime if the runtime.GOOS or runtime.GOARCH does not
// support Compiler. Use NewRuntimeConfig to safely detect and fallback to
// NewRuntimeConfigInterpreter if needed.
func NewRuntimeConfigCompiler() RuntimeConfig {
	ret := engineLessConfig.clone()
	ret.newEngine = compiler.NewEngine
	return ret
}

// NewRuntimeConfigInterpreter interprets WebAssembly modules instead of compiling them into assembly.
func NewRuntimeConfigInterpreter() RuntimeConfig {
	ret := engineLessConfig.clone()
	ret.isInterpreter = true
	ret.newEngine = interpreter.NewEngine
	return ret
}

// clone makes a deep copy of this runtime config.
func (c *runtimeConfig) clone() *runtimeConfig {
	ret := *c // copy except maps which share a ref
	return &ret
}

// WithCoreFeatures implements RuntimeConfig.WithCoreFeatures
func (c *runtimeConfig) WithCoreFeatures(features api.CoreFeatures) RuntimeConfig {
	ret := c.clone()
	ret.enabledFeatures = features
	return ret
}

// WithMemoryLimitPages implements RuntimeConfig.WithMemoryLimitPages
func (c *runtimeConfig) WithMemoryLimitPages(memoryLimitPages uint32) RuntimeConfig {
	ret := c.clone()
	// This panics instead of returning an error as it is unlikely.
	if memoryLimitPages > wasm.MemoryLimitPages {
		panic(fmt.Errorf("memoryLimitPages invalid: %d > %d", memoryLimitPages, wasm.MemoryLimitPages))
	}
	ret.memoryLimitPages = memoryLimitPages
	return ret
}

// WithMemoryCapacityFromMax implements RuntimeConfig.WithMemoryCapacityFromMax
func (c *runtimeConfig) WithMemoryCapacityFromMax(memoryCapacityFromMax bool) RuntimeConfig {
	ret := c.clone()
	ret.memoryCapacityFromMax = memoryCapacityFromMax
	return ret
}

// WithDebugInfoEnabled implements RuntimeConfig.WithDebugInfoEnabled
func (c *runtimeConfig) WithDebugInfoEnabled(dwarfEnabled bool) RuntimeConfig {
	ret := c.clone()
	ret.dwarfDisabled = !dwarfEnabled
	return ret
}

// CompiledModule is a WebAssembly module ready to be instantiated (Runtime.InstantiateModule) as an api.Module.
//
// In WebAssembly terminology, this is a decoded, validated, and possibly also compiled module. wazero avoids using
// the name "Module" for both before and after instantiation as the name conflation has caused confusion.
// See https://www.w3.org/TR/2019/REC-wasm-core-1-20191205/#semantic-phases%E2%91%A0
//
// Note: Closing the wazero.Runtime closes any CompiledModule it compiled.
type CompiledModule interface {
	// Name returns the module name encoded into the binary or empty if not.
	Name() string

	// ImportedFunctions returns all the imported functions
	// (api.FunctionDefinition) in this module or nil if there are none.
	//
	// Note: Unlike ExportedFunctions, there is no unique constraint on
	// imports.
	ImportedFunctions() []api.FunctionDefinition

	// ExportedFunctions returns all the exported functions
	// (api.FunctionDefinition) in this module keyed on export name.
	ExportedFunctions() map[string]api.FunctionDefinition

	// ImportedMemories returns all the imported memories
	// (api.MemoryDefinition) in this module or nil if there are none.
	//
	// ## Notes
	//   - As of WebAssembly Core Specification 2.0, there can be at most one
	//     memory.
	//   - Unlike ExportedMemories, there is no unique constraint on imports.
	ImportedMemories() []api.MemoryDefinition

	// ExportedMemories returns all the exported memories
	// (api.MemoryDefinition) in this module keyed on export name.
	//
	// Note: As of WebAssembly Core Specification 2.0, there can be at most one
	// memory.
	ExportedMemories() map[string]api.MemoryDefinition

	// Close releases all the allocated resources for this CompiledModule.
	//
	// Note: It is safe to call Close while having outstanding calls from an
	// api.Module instantiated from this.
	Close(context.Context) error
}

// compile-time check to ensure compiledModule implements CompiledModule
var _ CompiledModule = &compiledModule{}

type compiledModule struct {
	module *wasm.Module
	// compiledEngine holds an engine on which `module` is compiled.
	compiledEngine wasm.Engine
	// closeWithModule prevents leaking compiled code when a module is compiled implicitly.
	closeWithModule bool
}

// Name implements CompiledModule.Name
func (c *compiledModule) Name() (moduleName string) {
	if ns := c.module.NameSection; ns != nil {
		moduleName = ns.ModuleName
	}
	return
}

// Close implements CompiledModule.Close
func (c *compiledModule) Close(context.Context) error {
	c.compiledEngine.DeleteCompiledModule(c.module)
	// It is possible the underlying may need to return an error later, but in any case this matches api.Module.Close.
	return nil
}

// ImportedFunctions implements CompiledModule.ImportedFunctions
func (c *compiledModule) ImportedFunctions() []api.FunctionDefinition {
	return c.module.ImportedFunctions()
}

// ExportedFunctions implements CompiledModule.ExportedFunctions
func (c *compiledModule) ExportedFunctions() map[string]api.FunctionDefinition {
	return c.module.ExportedFunctions()
}

// ImportedMemories implements CompiledModule.ImportedMemories
func (c *compiledModule) ImportedMemories() []api.MemoryDefinition {
	return c.module.ImportedMemories()
}

// ExportedMemories implements CompiledModule.ExportedMemories
func (c *compiledModule) ExportedMemories() map[string]api.MemoryDefinition {
	return c.module.ExportedMemories()
}

// ModuleConfig configures resources needed by functions that have low-level interactions with the host operating
// system. Using this, resources such as STDIN can be isolated, so that the same module can be safely instantiated
// multiple times.
//
// Here's an example:
//
//	// Initialize base configuration:
//	config := wazero.NewModuleConfig().WithStdout(buf).WithSysNanotime()
//
//	// Assign different configuration on each instantiation
//	module, _ := r.InstantiateModule(ctx, compiled, config.WithName("rotate").WithArgs("rotate", "angle=90", "dir=cw"))
//
// While wazero supports Windows as a platform, host functions using ModuleConfig follow a UNIX dialect.
// See RATIONALE.md for design background and relationship to WebAssembly System Interfaces (WASI).
//
// Note: ModuleConfig is immutable. Each WithXXX function returns a new instance including the corresponding change.
type ModuleConfig interface {
	// WithArgs assigns command-line arguments visible to an imported function that reads an arg vector (argv). Defaults to
	// none. Runtime.InstantiateModule errs if any arg is empty.
	//
	// These values are commonly read by the functions like "args_get" in "wasi_snapshot_preview1" although they could be
	// read by functions imported from other modules.
	//
	// Similar to os.Args and exec.Cmd Env, many implementations would expect a program name to be argv[0]. However, neither
	// WebAssembly nor WebAssembly System Interfaces (WASI) define this. Regardless, you may choose to set the first
	// argument to the same value set via WithName.
	//
	// Note: This does not default to os.Args as that violates sandboxing.
	//
	// See https://linux.die.net/man/3/argv and https://en.wikipedia.org/wiki/Null-terminated_string
	WithArgs(...string) ModuleConfig

	// WithEnv sets an environment variable visible to a Module that imports functions. Defaults to none.
	// Runtime.InstantiateModule errs if the key is empty or contains a NULL(0) or equals("") character.
	//
	// Validation is the same as os.Setenv on Linux and replaces any existing value. Unlike exec.Cmd Env, this does not
	// default to the current process environment as that would violate sandboxing. This also does not preserve order.
	//
	// Environment variables are commonly read by the functions like "environ_get" in "wasi_snapshot_preview1" although
	// they could be read by functions imported from other modules.
	//
	// While similar to process configuration, there are no assumptions that can be made about anything OS-specific. For
	// example, neither WebAssembly nor WebAssembly System Interfaces (WASI) define concerns processes have, such as
	// case-sensitivity on environment keys. For portability, define entries with case-insensitively unique keys.
	//
	// See https://linux.die.net/man/3/environ and https://en.wikipedia.org/wiki/Null-terminated_string
	WithEnv(key, value string) ModuleConfig

	// WithFS assigns the file system to use for any paths beginning at "/".
	// Defaults return fs.ErrNotExist.
	//
	// This example sets a read-only, embedded file-system:
	//
	//	//go:embed testdata/index.html
	//	var testdataIndex embed.FS
	//
	//	rooted, err := fs.Sub(testdataIndex, "testdata")
	//	require.NoError(t, err)
	//
	//	// "index.html" is accessible as "/index.html".
	//	config := wazero.NewModuleConfig().WithFS(rooted)
	//
	// This example sets a mutable file-system:
	//
	//	// Files relative to "/work/appA" are accessible as "/".
	//	config := wazero.NewModuleConfig().WithFS(os.DirFS("/work/appA"))
	//
	// Isolation
	//
	// os.DirFS documentation includes important notes about isolation, which
	// also applies to fs.Sub. As of Go 1.19, the built-in file-systems are not
	// jailed (chroot). See https://github.com/golang/go/issues/42322
	//
	// Working Directory "."
	//
	// Relative path resolution, such as "./config.yml" to "/config.yml" or
	// otherwise, is compiler-specific. See /RATIONALE.md for notes.
	WithFS(fs.FS) ModuleConfig

	// WithName configures the module name. Defaults to what was decoded from the name section.
	WithName(string) ModuleConfig

	// WithStartFunctions configures the functions to call after the module is
	// instantiated. Defaults to "_start".
	//
	// # Notes
	//
	//   - If any function doesn't exist, it is skipped. However, all functions
	//	  that do exist are called in order.
	//   - Some start functions may exit the module during instantiate with a
	//	  sys.ExitError (e.g. emscripten), preventing use of exported functions.
	WithStartFunctions(...string) ModuleConfig

	// WithStderr configures where standard error (file descriptor 2) is written. Defaults to io.Discard.
	//
	// This writer is most commonly used by the functions like "fd_write" in "wasi_snapshot_preview1" although it could
	// be used by functions imported from other modules.
	//
	// # Notes
	//
	//   - The caller is responsible to close any io.Writer they supply: It is not closed on api.Module Close.
	//   - This does not default to os.Stderr as that both violates sandboxing and prevents concurrent modules.
	//
	// See https://linux.die.net/man/3/stderr
	WithStderr(io.Writer) ModuleConfig

	// WithStdin configures where standard input (file descriptor 0) is read. Defaults to return io.EOF.
	//
	// This reader is most commonly used by the functions like "fd_read" in "wasi_snapshot_preview1" although it could
	// be used by functions imported from other modules.
	//
	// # Notes
	//
	//   - The caller is responsible to close any io.Reader they supply: It is not closed on api.Module Close.
	//   - This does not default to os.Stdin as that both violates sandboxing and prevents concurrent modules.
	//
	// See https://linux.die.net/man/3/stdin
	WithStdin(io.Reader) ModuleConfig

	// WithStdout configures where standard output (file descriptor 1) is written. Defaults to io.Discard.
	//
	// This writer is most commonly used by the functions like "fd_write" in "wasi_snapshot_preview1" although it could
	// be used by functions imported from other modules.
	//
	// # Notes
	//
	//   - The caller is responsible to close any io.Writer they supply: It is not closed on api.Module Close.
	//   - This does not default to os.Stdout as that both violates sandboxing and prevents concurrent modules.
	//
	// See https://linux.die.net/man/3/stdout
	WithStdout(io.Writer) ModuleConfig

	// WithWalltime configures the wall clock, sometimes referred to as the
	// real time clock. Defaults to a fake result that increases by 1ms on
	// each reading.
	//
	// Here's an example that uses a custom clock:
	//	moduleConfig = moduleConfig.
	//		WithWalltime(func(context.Context) (sec int64, nsec int32) {
	//			return clock.walltime()
	//		}, sys.ClockResolution(time.Microsecond.Nanoseconds()))
	//
	// Note: This does not default to time.Now as that violates sandboxing. Use
	// WithSysWalltime for a usable implementation.
	WithWalltime(sys.Walltime, sys.ClockResolution) ModuleConfig

	// WithSysWalltime uses time.Now for sys.Walltime with a resolution of 1us
	// (1000ns).
	//
	// See WithWalltime
	WithSysWalltime() ModuleConfig

	// WithNanotime configures the monotonic clock, used to measure elapsed
	// time in nanoseconds. Defaults to a fake result that increases by 1ms
	// on each reading.
	//
	// Here's an example that uses a custom clock:
	//	moduleConfig = moduleConfig.
	//		WithNanotime(func(context.Context) int64 {
	//			return clock.nanotime()
	//		}, sys.ClockResolution(time.Microsecond.Nanoseconds()))
	//
	// # Notes:
	//   - This does not default to time.Since as that violates sandboxing.
	//   - Some compilers implement sleep by looping on sys.Nanotime (e.g. Go).
	//   - If you set this, you should probably set WithNanosleep also.
	//   - Use WithSysNanotime for a usable implementation.
	WithNanotime(sys.Nanotime, sys.ClockResolution) ModuleConfig

	// WithSysNanotime uses time.Now for sys.Nanotime with a resolution of 1us.
	//
	// See WithNanotime
	WithSysNanotime() ModuleConfig

	// WithNanosleep configures the how to pause the current goroutine for at
	// least the configured nanoseconds. Defaults to return immediately.
	//
	// This example uses a custom sleep function:
	//	moduleConfig = moduleConfig.
	//		WithNanosleep(func(ctx context.Context, ns int64) {
	//			rel := unix.NsecToTimespec(ns)
	//			remain := unix.Timespec{}
	//			for { // loop until no more time remaining
	//				err := unix.ClockNanosleep(unix.CLOCK_MONOTONIC, 0, &rel, &remain)
	//			--snip--
	//
	// # Notes:
	//   - This primarily supports `poll_oneoff` for relative clock events.
	//   - This does not default to time.Sleep as that violates sandboxing.
	//   - Some compilers implement sleep by looping on sys.Nanotime (e.g. Go).
	//   - If you set this, you should probably set WithNanotime also.
	//   - Use WithSysNanosleep for a usable implementation.
	WithNanosleep(sys.Nanosleep) ModuleConfig

	// WithSysNanosleep uses time.Sleep for sys.Nanosleep.
	//
	// See WithNanosleep
	WithSysNanosleep() ModuleConfig

	// WithRandSource configures a source of random bytes. Defaults to return a
	// deterministic source. You might override this with crypto/rand.Reader
	//
	// This reader is most commonly used by the functions like "random_get" in
	// "wasi_snapshot_preview1", "seed" in AssemblyScript standard "env", and
	// "getRandomData" when runtime.GOOS is "js".
	//
	// Note: The caller is responsible to close any io.Reader they supply: It
	// is not closed on api.Module Close.
	WithRandSource(io.Reader) ModuleConfig
}

type moduleConfig struct {
	name               string
	startFunctions     []string
	stdin              io.Reader
	stdout             io.Writer
	stderr             io.Writer
	randSource         io.Reader
	walltime           *sys.Walltime
	walltimeResolution sys.ClockResolution
	nanotime           *sys.Nanotime
	nanotimeResolution sys.ClockResolution
	nanosleep          *sys.Nanosleep
	args               [][]byte
	// environ is pair-indexed to retain order similar to os.Environ.
	environ [][]byte
	// environKeys allow overwriting of existing values.
	environKeys map[string]int
	// fs is the file system to open files with
	fs fs.FS
}

// NewModuleConfig returns a ModuleConfig that can be used for configuring module instantiation.
func NewModuleConfig() ModuleConfig {
	return &moduleConfig{
		startFunctions: []string{"_start"},
		environKeys:    map[string]int{},
	}
}

// clone makes a deep copy of this module config.
func (c *moduleConfig) clone() *moduleConfig {
	ret := *c // copy except maps which share a ref
	ret.environKeys = make(map[string]int, len(c.environKeys))
	for key, value := range c.environKeys {
		ret.environKeys[key] = value
	}
	return &ret
}

// WithArgs implements ModuleConfig.WithArgs
func (c *moduleConfig) WithArgs(args ...string) ModuleConfig {
	ret := c.clone()
	ret.args = toByteSlices(args)
	return ret
}

func toByteSlices(strings []string) (result [][]byte) {
	if len(strings) == 0 {
		return
	}
	result = make([][]byte, len(strings))
	for i, a := range strings {
		result[i] = []byte(a)
	}
	return
}

// WithEnv implements ModuleConfig.WithEnv
func (c *moduleConfig) WithEnv(key, value string) ModuleConfig {
	ret := c.clone()
	// Check to see if this key already exists and update it.
	if i, ok := ret.environKeys[key]; ok {
		ret.environ[i+1] = []byte(value) // environ is pair-indexed, so the value is 1 after the key.
	} else {
		ret.environKeys[key] = len(ret.environ)
		ret.environ = append(ret.environ, []byte(key), []byte(value))
	}
	return ret
}

// WithFS implements ModuleConfig.WithFS
func (c *moduleConfig) WithFS(fs fs.FS) ModuleConfig {
	ret := c.clone()
	ret.fs = fs
	return ret
}

// WithName implements ModuleConfig.WithName
func (c *moduleConfig) WithName(name string) ModuleConfig {
	ret := c.clone()
	ret.name = name
	return ret
}

// WithStartFunctions implements ModuleConfig.WithStartFunctions
func (c *moduleConfig) WithStartFunctions(startFunctions ...string) ModuleConfig {
	ret := c.clone()
	ret.startFunctions = startFunctions
	return ret
}

// WithStderr implements ModuleConfig.WithStderr
func (c *moduleConfig) WithStderr(stderr io.Writer) ModuleConfig {
	ret := c.clone()
	ret.stderr = stderr
	return ret
}

// WithStdin implements ModuleConfig.WithStdin
func (c *moduleConfig) WithStdin(stdin io.Reader) ModuleConfig {
	ret := c.clone()
	ret.stdin = stdin
	return ret
}

// WithStdout implements ModuleConfig.WithStdout
func (c *moduleConfig) WithStdout(stdout io.Writer) ModuleConfig {
	ret := c.clone()
	ret.stdout = stdout
	return ret
}

// WithWalltime implements ModuleConfig.WithWalltime
func (c *moduleConfig) WithWalltime(walltime sys.Walltime, resolution sys.ClockResolution) ModuleConfig {
	ret := c.clone()
	ret.walltime = &walltime
	ret.walltimeResolution = resolution
	return ret
}

// We choose arbitrary resolutions here because there's no perfect alternative. For example, according to the
// source in time.go, windows monotonic resolution can be 15ms. This chooses arbitrarily 1us for wall time and
// 1ns for monotonic. See RATIONALE.md for more context.

// WithSysWalltime implements ModuleConfig.WithSysWalltime
func (c *moduleConfig) WithSysWalltime() ModuleConfig {
	return c.WithWalltime(platform.Walltime, sys.ClockResolution(time.Microsecond.Nanoseconds()))
}

// WithNanotime implements ModuleConfig.WithNanotime
func (c *moduleConfig) WithNanotime(nanotime sys.Nanotime, resolution sys.ClockResolution) ModuleConfig {
	ret := c.clone()
	ret.nanotime = &nanotime
	ret.nanotimeResolution = resolution
	return ret
}

// WithSysNanotime implements ModuleConfig.WithSysNanotime
func (c *moduleConfig) WithSysNanotime() ModuleConfig {
	return c.WithNanotime(platform.Nanotime, sys.ClockResolution(1))
}

// WithNanosleep implements ModuleConfig.WithNanosleep
func (c *moduleConfig) WithNanosleep(nanosleep sys.Nanosleep) ModuleConfig {
	ret := *c // copy
	ret.nanosleep = &nanosleep
	return &ret
}

// WithSysNanosleep implements ModuleConfig.WithSysNanosleep
func (c *moduleConfig) WithSysNanosleep() ModuleConfig {
	return c.WithNanosleep(platform.Nanosleep)
}

// WithRandSource implements ModuleConfig.WithRandSource
func (c *moduleConfig) WithRandSource(source io.Reader) ModuleConfig {
	ret := c.clone()
	ret.randSource = source
	return ret
}

// toSysContext creates a baseline wasm.Context configured by ModuleConfig.
func (c *moduleConfig) toSysContext() (sysCtx *internalsys.Context, err error) {
	var environ [][]byte // Intentionally doesn't pre-allocate to reduce logic to default to nil.
	// Same validation as syscall.Setenv for Linux
	for i := 0; i < len(c.environ); i += 2 {
		key, value := c.environ[i], c.environ[i+1]
		keyLen := len(key)
		if keyLen == 0 {
			err = errors.New("environ invalid: empty key")
			return
		}
		valueLen := len(value)
		result := make([]byte, keyLen+valueLen+1)
		j := 0
		for ; j < keyLen; j++ {
			if k := key[j]; k == '=' { // NUL enforced in NewContext
				err = errors.New("environ invalid: key contains '=' character")
				return
			} else {
				result[j] = k
			}
		}
		result[j] = '='
		copy(result[j+1:], value)
		environ = append(environ, result)
	}

	return internalsys.NewContext(
		math.MaxUint32,
		c.args,
		environ,
		c.stdin,
		c.stdout,
		c.stderr,
		c.randSource,
		c.walltime, c.walltimeResolution,
		c.nanotime, c.nanotimeResolution,
		c.nanosleep,
		c.fs,
	)
}
