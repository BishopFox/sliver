# implant/sliver/wasmnet

This package is the single implementation of Sliver's versioned
`sliver_wasi_net_v1` host module. Implant extensions, implant traffic
encoders, and server traffic encoders each create a private `Host` for their
wazero runtime.

The ABI exposes asynchronous TCP, UDP, and DNS operations plus polling,
cancellation, deadlines, shutdown, address lookup, and handle cleanup. Keeping
the host here makes generated implants and the server compile the same source,
while keeping socket and operation handles isolated between Wasm runtimes.

The module exports `dial_start`, `listen`, `accept_start`, `read_start`,
`write_start`, `recv_from_start`, `send_to_start`, `lookup_start`, `op_poll`,
`op_cancel`, `shutdown`, `close`, `get_addr`, and `set_deadline`. Guest code
normally reaches these through the `sliver-wasm-go` standard-library overlay
instead of importing them directly.

Unsupported Sliver targets keep a no-op stub so non-networked traffic encoders
remain buildable. A module that imports `sliver_wasi_net_v1` fails
instantiation on those targets instead of receiving partial networking
behavior.
