# Writable memfs WASI fixture

This Go/WASI program is the integration fixture for the extension runtime's
`/memfs` implementation. It creates directories and files, performs sequential
and append I/O, renames entries, reopens persisted data, and removes files and
directories. Its `readonly` mode verifies that seeded data remains readable
while create and truncate operations fail.

Build it outside the Sliver console with the bundled wrapper:

```console
$SLIVER_ROOT_DIR/go/bin/sliver-wasm-go build \
  -o memfs-rw.wasm \
  ./implant/sliver/extension/testdata/memfs-rw
```

The resulting module uses standard Go `os` APIs and runs inside a Sliver Wasm
extension host.
