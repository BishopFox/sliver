Sliver v1.6 supports user-defined "Traffic Encoders," which can be used to arbitrarily modify the Sliver implant's network communication over HTTP(S). Traffic Encoders are [Wasm-based](https://webassembly.org/) callback functions that encode/decode network traffic between the implant and the server. Traffic Encoders can be written in any language that compiles to [Wasm](https://webassembly.org/). Traffic Encoders are supported on all platforms and CPU architectures that Sliver can target, though performance may vary significantly.

[Examples](https://github.com/BishopFox/sliver/tree/v1.6.0/server/assets/traffic-encoders) are provided in this repository of a [Rust](https://www.rust-lang.org/)-based and a [TinyGo](https://tinygo.org/)-based encoder. For performance reasons we recommend implementing Traffic Encoders in Rust.

For performance reasons, by default C2 messages over 2 MB in size are NOT passed through user-defined Traffic Encoders, but instead always use a native built-in encoder; this limit can be configured at implant generation-time.

## Traffic Encoder Specification

### Exports

Traffic encoders are implemented in Wasm, and must export the following functions:

```go
decode(ptr, size uint32) (ptrSize uint64)
encode(ptr, size uint32) (ptrSize uint64)

malloc(size uint32) uint32
free(ptr uint32, size uint32)
```

The `encode` function takes a `uint32` pointer to a buffer and the `uint32` size of the buffer, and returns a `uint64` value where the upper 32 bits are a pointer to the address of the buffer and the lower 32 bits are the size of the buffer.

The `decode` function takes a `uint32` pointer to a buffer and the `uint32` size of the buffer, and returns a `uint64` value where the upper 32 bits are a pointer to the address of the buffer and the lower 32 bits are the size of the buffer.

The `malloc` function takes a `uint32` size. The function should allocate a buffer of the given size in bytes and return its `uint32` linear-memory pointer. For compatibility with older modules and documentation, Sliver also accepts an `i64` parameter or result when the value fits in 32-bit Wasm linear memory.

The `free` function takes a pointer to memory which should be freed and a size.

For example, a return value in Go may look like:

```go
return (uint64(ptr) << uint64(32)) | uint64(size)
```

Where `ptr` is a pointer to the buffer and `size` is the size of the buffer.

### Imports

Optionally, the following imports may be used:

```go
log(ptr uint32, size uint32)
rand() uint64
time() int64
```

The `log` function takes a pointer to a buffer and the size of the buffer, and logs the contents of the buffer to the console if the implant is in debug mode. The server will also log the contents if configured to log at the DEBUG level or higher.

The `rand()` function returns 64-bits of cryptographically secure random data.

The `time()` function returns the current Unix Nano time as an `int64`.

### Go WASI networking

Sliver's server and implant traffic-encoder runtimes both provide the
versioned `sliver_wasi_net_v1` host. Go modules built with the standalone
`$SLIVER_ROOT_DIR/go/bin/sliver-wasm-go` wrapper can therefore use standard
library TCP, UDP, DNS, and HTTP APIs on both sides:

```console
$SLIVER_ROOT_DIR/go/bin/sliver-wasm-go build -buildmode=c-shared -o encoder.wasm .
```

The shared build mode creates a WASI reactor. Sliver detects and invokes its
`_initialize` export before calling `malloc`, `encode`, or `decode`.
