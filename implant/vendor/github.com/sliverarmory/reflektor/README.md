# Reflektor

<img align="right" src=".github/images/reflektor.png" alt="Reflektor" width="300">

Reflektor is a Go library and CLI for loading shared libraries from bytes and invoking exported functions.

It exposes a stable root package (`reflektor`) so other projects can import it directly, while platform-specific loading is handled behind `memmod`.

<br clear="right">

## Platform Support

| OS | Architectures | Shared Library Format | Status | Loader Notes |
| --- | --- | --- | --- | --- |
| Windows | `386`, `amd64`, `arm64` | PE (`.dll`) | Supported | In-memory PE loader |
| Darwin | `amd64`, `arm64` | Mach-O (`.dylib`, bundle) | Supported | Dyld4 root-image loader with system dependencies registered through public dyld; supports no-cgo builds and avoids temp-file legacy NS APIs. |
| Linux | `386`, `amd64`, `arm64` | ELF (`.so`) | Supported | Pure Go in-memory ELF loader (maps PT_LOAD segments, applies relocations, resolves externals from runtime modules/`dlsym`); no `memfd`, no `/dev/shm`, no temp-file disk writes. |
| Other | - | - | Unsupported | Returns an explicit unsupported-platform error. |

## Public API

Import path:

```go
import "github.com/sliverarmory/reflektor"
```

Example:

```go
payload := []byte{}

lib, err := reflektor.LoadLibrary(payload)
if err != nil {
    return err
}
defer lib.Close()

if err := lib.CallExport("StartW"); err != nil {
    return err
}
```

Native C and Rust exports can also receive up to three machine-word arguments
and return a machine-word value:

```go
result, err := lib.CallExportWithArgs(
    "Run",
    uintptr(unsafe.Pointer(unsafe.SliceData(input))),
    uintptr(uint32(len(input))),
    callbackPointer,
)
runtime.KeepAlive(input)
```

This matches extension entry points such as
`Run(char *buffer, uint32_t size, callback_fn callback)`. Convert a Go pointer
to `uintptr` directly in the method call, as above, and keep the pointed-to
object alive until the export returns. CGO-free Darwin and Linux callers can
create C-callable Go callbacks with `purego.NewCallback`; Windows callers can
use `syscall.NewCallback`.

You can also load from a path:

```go
lib, err := reflektor.LoadLibraryFile("./payload.dylib")
```

To read and map a library's non-system dependencies through Reflektor as well,
use recursive mode:

```go
lib, err := reflektor.LoadLibraryFileRecursive("./payload.dylib")
```

The byte-oriented equivalent resolves relative dependency names from the
current working directory:

```go
lib, err := reflektor.LoadLibraryRecursive(payload)
```

Both recursive APIs read the complete custom dependency graph before the root
export is invoked. `LoadLibraryFileRecursive` is preferred for libraries that
use origin-relative names such as `$ORIGIN`, `@loader_path`, or `@rpath`.

## CLI

The CLI is in `reflektor/cli` and uses Cobra.

Build:

```bash
go build -o reflektor ./cli
```

Usage:

```bash
./reflektor <shared-library-path> [--call-export StartW]
```

`--call-export` defaults to `StartW`.

## Behavior Notes

- `CallExport` preserves the original zero-argument API. `CallExportWithArgs`
  accepts zero through three `uintptr` arguments and returns the platform's
  primary machine-word return value.
- `CallExportWithArgs` supports native C and Rust images. Go c-shared images
  return `ErrGoExportArgumentsUnsupported`; their zero-argument exports remain
  available through `CallExport`.
- On Linux, the `CGO_ENABLED=0` argument-call bridge uses purego's runtime
  integration so C-to-Go callbacks are safe. Consequently, hosts importing
  Reflektor are dynamically linked against the platform's glibc loader even if
  they use only the legacy APIs; this is not a fully static or musl-portable
  build mode.
- Reflektor normalizes common symbol naming differences where possible (for example underscore-prefixed forms).
- The root `reflektor.Library` API remains intentionally small:
  `CallExport()`, `CallExportWithArgs()`, and `Close()`.
- Recursive mode maps file-backed application dependencies from their bytes and
  resolves imports within the in-memory graph. Platform runtime libraries remain
  delegated to the native loader: Darwin shared-cache libraries, Windows
  System32/API-set libraries, and Linux libraries in trusted system roots. Those
  libraries require OS-managed TLS, symbol versioning, loader registration, and
  other facilities that cannot be reproduced by simply reading a file—and some
  Darwin shared-cache images do not exist as standalone readable files.
- Linux custom images reject general ELF TLS, IFUNC/IRELATIVE, and RELR with
  explicit errors; those features remain available through the system-library
  carveout. Windows custom dependency cycles and delay-load import tables are
  also rejected explicitly. Darwin and Linux graph cycles are deduplicated.
- After the first export call, Darwin recursive mappings remain process-resident
  because dyld retains their loader records. Reusing a Darwin install-name in a
  later load follows dyld's first-loaded identity semantics. Calls made through
  the same `Library` reuse one mapped root, so an initializer and later
  argument-bearing exports share module state.
- `LoadLibrary` and `LoadLibraryFile` retain their original behavior and API.

## Test Data And Validation

C test shared libraries are generated from:

- `reflektor/testdata/c/args.c`
- `reflektor/testdata/c/basic.c`
- `reflektor/testdata/c/recursive_leaf.c`
- `reflektor/testdata/c/recursive_middle.c`
- `reflektor/testdata/c/recursive_root.c`

The recursive C fixture is a transitive root -> middle -> leaf graph. Its test
checks that the graph is absent from Linux `/proc/self/maps` or the Windows
loader module registry, then renames the dependency directory before calling
`StartW`. On Darwin the rename happens before the lazy dyld transaction. These
checks prove the custom dependencies came from bytes captured by Reflektor.

The Rust HTTPS fixture is built from `reflektor/testdata/rust`. It exports `StartW`, performs a bounded `GET https://example.com/` through libcurl on Darwin/Linux or WinHTTP on Windows, and records `ok:200` after receiving a non-empty successful response. The fixture is dependency-free Rust (`no_std`) so it does not require unsupported thread-local runtime state from the in-memory loaders.

Build test shared libraries for the full matrix:

```bash
./testdata/build_c_shared_libs.sh
```

Run tests:

```bash
go test ./...
```

The Rust fixture test requires Cargo with Rust 1.94.0 and outbound HTTPS access. Linux also requires the libcurl development package so the fixture can link against the system TLS client.

Linux cross-arch Docker harness:

- `reflektor/testdata/docker/linux-memmod.Dockerfile`
- `reflektor/testdata/docker/run-linux-memmod-matrix.sh`

## Repository Layout

- `reflektor/reflektor.go`: root importable package (`reflektor`).
- `reflektor/memmod`: OS-specific loader backends.
- `reflektor/cli`: CLI entrypoint.
- `reflektor/testdata`: portable shared-library fixtures and build/test harnesses.
