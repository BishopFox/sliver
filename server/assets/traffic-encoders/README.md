# Traffic Encoders

This directory contains the default traffic encoders.

### Traffic Encoder Specification

Traffic encoders are implemented in WASM, and must export the following functions:

```
decode(ptr, size uint32) (ptrSize uint64)
encode(ptr, size uint32) (ptrSize uint64)
```

The `encode` function takes a pointer to a buffer and the size of the buffer, and returns a pointer to a new buffer containing the encoded data and the size of the new buffer.

The `decode` function takes a pointer to a buffer and the size of the buffer, and returns a pointer to a new buffer containing the decoded data and the size of the new buffer.

#### Debug Log

Optionally, the following import may be used:

```
log(ptr uint32, size uint32)
```

This function takes a pointer to a buffer and the size of the buffer, and logs the contents of the buffer to the console if the implant is in debug mode. On the server will also log the contents if configured to log at the DEBUG level or higher.
