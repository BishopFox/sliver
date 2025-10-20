# go-keystone
WASM based bindings for the [Keystone](https://github.com/For-ACGN/keystone) assembler.
## Features
Since Keystone is compiled into a wasm module and a pure go-implemented wasm runtime [wazero](https://github.com/tetratelabs/wazero) is used, calling the C program is implemented while retaining cross-compilation.
## Usage
```bash
keystone -arch x86 -mode 32 -src hello.asm -out hello.bin
```
## Development
```go
package main

import (
    "fmt"
    "os"

    "github.com/For-ACGN/go-keystone"
)

func main() {
    engine, err := keystone.NewEngine(keystone.ARCH_X86, keystone.MODE_64)
    checkError(err)
    defer func() { _ = engine.Close() }()

    err = engine.Option(keystone.OPT_SYNTAX, keystone.OPT_SYNTAX_INTEL)
    checkError(err)

    src := ".code64\n"
    src += "xor rax, rax\n"
    src += "ret\n"
    inst, err := engine.Assemble(src, 0)
    checkError(err)

    // [0x48, 0x31, 0xC0, 0xC3]
    fmt.Println(inst)
}

func checkError(err error) {
    if err != nil {
        fmt.Println(err)
        os.Exit(1)
    }
}
```
