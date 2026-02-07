# Beignet

[Donut](github.com/sliverarmory/wasm-donut) for MacOS, converts .dylib files into MacOS PIC shellcode, can be used as a CLI or imported as a golang library.

### CLI

Convert a dylib to a raw shellcode buffer:

`./beignet --out payload.bin ./payload.dylib`

Optionally compress the staged dylib with aPLib (AP32):

`./beignet --compress --out payload.bin ./payload.dylib`

### Comple from Source

`make`

### Regenerating the embedded loader (darwin/arm64)

`go generate ./internal/stager`
