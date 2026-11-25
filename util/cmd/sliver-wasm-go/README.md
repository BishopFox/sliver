# sliver-wasm-go

`sliver-wasm-go` is installed beside Sliver's bundled `go` and `garble`
binaries in `$SLIVER_ROOT_DIR/go/bin`. It invokes that exact Go toolchain with
`GOOS=wasip1`, `GOARCH=wasm`, `CGO_ENABLED=0`, and an embedded `-overlay`
that supplies TCP, UDP, and DNS through Sliver's `sliver_wasi_net_v1` host.

The wrapper is standalone after asset installation, so modules can be built
outside the Sliver server or client console:

```text
$SLIVER_ROOT_DIR/go/bin/sliver-wasm-go build -o module.wasm .
```

Use `-buildmode=c-shared` for a traffic encoder. This creates a WASI reactor,
which both traffic runtimes initialize through its `_initialize` export before
calling the encoder exports.

The overlay is pinned to the bundled Go source hashes. The wrapper rejects a
different toolchain or a second `-overlay` instead of silently compiling an
incompatible module.
