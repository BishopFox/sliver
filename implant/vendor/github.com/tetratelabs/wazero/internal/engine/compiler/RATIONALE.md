# Compiler engine

This package implements the Compiler engine for WebAssembly *purely written in Go*.
In this README, we describe the background, technical difficulties and some design choices.

## General limitations on pure Go Compiler engines

In Go program, each Goroutine manages its own stack, and each item on Goroutine
stack is managed by Go runtime for garbage collection, etc.

These impose some difficulties on compiler engine purely written in Go because
we *cannot* use native push/pop instructions to save/restore temporary
variables spilling from registers. This results in making it impossible for us
to invoke Go functions from compiled native codes with the native `call`
instruction since it involves stack manipulations.

*TODO: maybe it is possible to hack the runtime to make it possible to achieve
function calls with `call`.*

## How to generate native codes

wazero uses its own assembler, implemented from scratch in the
[`internal/asm`](../../asm/) package. The primary rationale are wazero's zero
dependency policy, and to enable concurrent compilation (a feature the
WebAssembly binary format optimizes for).

Before this, wazero used [`twitchyliquid64/golang-asm`](https://github.com/twitchyliquid64/golang-asm).
However, this was not only a dependency (one of our goals is to have zero
dependencies), but also a large one (several megabytes added to the binary).
Moreover, any copy of golang-asm is not thread-safe, so can't be used for
concurrent compilation (See [#233](https://github.com/tetratelabs/wazero/issues/233)).

The assembled native codes are represented as `[]byte` and the slice region is
marked as executable via mmap system call.

## How to enter native codes

Assuming that we have a native code as `[]byte`, it is straightforward to enter
the native code region via Go assembly code. In this package, we have the
function without body called `nativecall`

```go
func nativecall(codeSegment, engine, memory uintptr)
```

where we pass `codeSegment uintptr` as a first argument. This pointer is to the
first instruction to be executed. The pointer can be easily derived from
`[]byte` via `unsafe.Pointer`:

```go
code := []byte{}
/* ...Compilation ...*/
codeSegment := uintptr(unsafe.Pointer(&code[0]))
nativecall(codeSegment, ...)
```

And `nativecall` is actually implemented in [arch_amd64.s](./arch_amd64.s)
as a convenience layer to comply with the Go's official calling convention.
We delegate the task to jump into the code segment to the Go assembler code.


## Why it's safe to execute runtime-generated machine codes against async Goroutine preemption

Goroutine preemption is the mechanism of the Go runtime to switch goroutines contexts on an OS thread.
There are two types of preemption: cooperative preemption and async preemption. The former happens, for example,
when making a function call, and it is not an issue for our runtime-generated functions as they do not make
direct function calls to Go-implemented functions. On the other hand, the latter, async preemption, can be problematic
since it tries to interrupt the execution of Goroutine at any point of function, and manipulates CPU register states.

Fortunately, our runtime-generated machine codes do not need to take the async preemption into account.
All the assembly codes are entered via the trampoline implemented as Go Assembler Function (e.g. [arch_amd64.s](./arch_amd64.s)),
and as of Go 1.20, these assembler functions are considered as _unsafe_ for async preemption:
- https://github.com/golang/go/blob/go1.20rc1/src/runtime/preempt.go#L406-L407
- https://github.com/golang/go/blob/9f0234214473dfb785a5ad84a8fc62a6a395cbc3/src/runtime/traceback.go#L227

From the Go runtime point of view, the execution of runtime-generated machine codes is considered as a part of
that trampoline function. Therefore, runtime-generated machine code is also correctly considered unsafe for async preemption.

## Why context cancellation is handled in Go code rather than native code

Since [wazero v1.0.0-pre.9](https://github.com/tetratelabs/wazero/releases/tag/v1.0.0-pre.9), the runtime
supports integration with Go contexts to interrupt execution after a timeout, or in response to explicit cancellation.
This support is internally implemented as a special opcode `builtinFunctionCheckExitCode` that triggers the execution of
a Go function (`ModuleInstance.FailIfClosed`) that atomically checks a sentinel value at strategic points in the code
(e.g. [within loops][checkexitcode_loop]).

[It _is indeed_ possible to check the sentinel value directly, without leaving the native world][native_check], thus sparing some cycles;
however, because native code never preempts (see section above), this may lead to a state where the other goroutines
never get the chance to run, and thus never get the chance to set the sentinel value; effectively preventing
cancellation from taking place.

[checkexitcode_loop]: https://github.com/tetratelabs/wazero/blob/86444c67a37dbf9e693ae5b365901f64968d9025/internal/wazeroir/compiler.go#L467-L476
[native_check]: https://github.com/tetratelabs/wazero/issues/1409

## Source Offset Mapping

When translating code from WebAssembly to the wazero IR, and compiling to native
binary, wazero keeps track of two indexes to correlate native program counters
to the original source offset that they were generated from.

Source offset maps are useful for debugging, but holding indexes in memory for
all instructions can have a significant overhead. To reduce the memory footprint
of the compiled modules, wazero uses data structures inspired by
[frame-of-reference and delta encoding][FOR].

Because wazero does not reorder instructions, the source offsets are naturally
sorted during compilation, and the distance between two consecutive offsets is
usually small. Encoding deltas instead of the absolute values allows most of
the indexes to store offsets with an overhead of 8 bits per instruction, instead
of recording 64 bits integers for absolute code positions.

[FOR]: https://lemire.me/blog/2012/02/08/effective-compression-using-frame-of-reference-and-delta-coding/
