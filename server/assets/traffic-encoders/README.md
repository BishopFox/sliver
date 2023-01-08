# Traffic Encoders

This directory contains the default traffic encoders, which are currently built using TinyGo.

### Traffic Encoder Specification

Traffic encoders are implemented in WASM, and must export the following functions:

```go
decode(ptr, size uint32) (ptrSize uint64)
encode(ptr, size uint32) (ptrSize uint64)

// These will typically be exported for you by the language runtime
malloc(size uint64)
free(ptr uint64)
```

The `encode` function takes a `uint32` pointer to a buffer and the `uint32` size of the buffer, and returns a `uint64` value where the upper 32 bits are a pointer to the address of the buffer and the lower 32 bits are the size of the buffer.

The `decode` function takes a `uint32` pointer to a buffer and the `uint32` size of the buffer, and returns a `uint64` value where the upper 32 bits are a pointer to the address of the buffer and the lower 32 bits are the size of the buffer.

For example, a return value in Go may look like:

```go
return (uint64(ptr) << uint64(32)) | uint64(size)
```

Where `ptr` is a pointer to the buffer and `size` is the size of the buffer.

#### Debug Log

Optionally, the following import may be used:

```
log(ptr uint32, size uint32)
```

This function takes a pointer to a buffer and the size of the buffer, and logs the contents of the buffer to the console if the implant is in debug mode. On the server will also log the contents if configured to log at the DEBUG level or higher.
