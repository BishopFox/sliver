# ext

Demonstrates how to use external types in an `avo` function signature.

In this case, you will need to write the function stub yourself.

[embedmd]:# (stub.go /package/ $)
```go
package ext

import "github.com/mmcloughlin/avo/examples/ext/ext"

// StructFieldB returns field B.
func StructFieldB(e ext.Struct) byte
```

Then in place of the usual `TEXT` declaration we use `Implement` to state that we are beginning the definition of a function already declared in the package stub file.

[embedmd]:# (asm.go go /.*Package.*/ /RET.*/)
```go
	Package("github.com/mmcloughlin/avo/examples/ext")
	Implement("StructFieldB")
	b := Load(Param("e").Field("B"), GP8())
	Store(b, ReturnIndex(0))
	RET()
```

Finally, in this case the `go:generate` line is different since we do not need to generate the stub file.

[embedmd]:# (ext_test.go go /.*go:generate.*/)
```go
//go:generate go run asm.go -out ext.s
```
