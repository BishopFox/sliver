# malasada

[Donut](https://github.com/sliverarmory/wasm-donut) for Linux, converts a Linux ELF shared object (`.so`) into a position-independent `.bin` blob that can be executed directly from memory (for example, by copying it into an `mmap`'d region and jumping to it as a function pointer).

This project intentionally avoids `memfd` (`memfd_create`, `execveat` on memfd, etc).

## Supported Platforms

- Linux `amd64`
- Linux `arm64`
- Linux `386`

## Build

Requirements:

- Go 1.25+
- Zig 0.15+ (required for `go generate` and for tests)

Build the CLI:

```bash
make
```

## Quick Start (Local)

Build a hello-world `.so`, convert it to a `.bin`, and run it with the included runner:

```bash
# Build payload .so
go build -buildmode=c-shared -o /tmp/hello.so ./testdata/hello

# Convert .so -> .bin (call the exported symbol "Hello")
./malasada --call-export Hello -o /tmp/hello.bin /tmp/hello.so

# Optional: compress the embedded payload (stage0 will depack before loading)
./malasada --compress --call-export Hello -o /tmp/hello.compressed.bin /tmp/hello.so

# Build the runner (PIC shellcode executor) with zig cc
zig cc -O2 -o /tmp/runner ./testdata/runner/runner.c

# Run it (stage0 hands off to ld-linux; runner will not return)
/tmp/runner /tmp/hello.bin
```

Expected output contains:

```
hello from go
```

## go generate (Prebuild + Embed stage0)

The repo embeds prebuilt `stage0` blobs:

- `internal/stage0/stage0_linux_amd64.bin`
- `internal/stage0/stage0_linux_arm64.bin`
- `internal/stage0/stage0_linux_386.bin`

If you edit `internal/stage0/stage0.c` or `internal/stage0/linker.ld`, regenerate them:

```bash
go generate ./...
```

The CLI always uses the embedded `stage0` blobs (no Zig needed at runtime). To change `stage0`, edit `internal/stage0/stage0.c` and re-run:

```bash
go generate ./...
```

## Docker Test Harness

`testdata/Dockerfile` builds the CLI, builds the hello `.so`, converts it to a `.bin`, builds the runner with Zig, and runs the end-to-end test in a Linux container.

Examples:

```bash
docker buildx build --platform linux/amd64 -f testdata/Dockerfile .
docker buildx build --platform linux/arm64 -f testdata/Dockerfile .
docker buildx build --platform linux/386 -f testdata/Dockerfile .

# Or via Makefile:
make docker-test-386
```
