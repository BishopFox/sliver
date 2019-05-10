# fnv1a

[FNV-1a](https://en.wikipedia.org/wiki/Fowler%E2%80%93Noll%E2%80%93Vo_hash_function#FNV-1a_hash) in `avo`.

[embedmd]:# (asm.go /const/ $)
```go
const (
	OffsetBasis = 0xcbf29ce484222325
	Prime       = 0x100000001b3
)

func main() {
	TEXT("Hash64", NOSPLIT, "func(data []byte) uint64")
	Doc("Hash64 computes the FNV-1a hash of data.")
	ptr := Load(Param("data").Base(), GP64())
	n := Load(Param("data").Len(), GP64())

	h := RAX
	MOVQ(Imm(OffsetBasis), h)
	p := GP64()
	MOVQ(Imm(Prime), p)

	Label("loop")
	CMPQ(n, Imm(0))
	JE(LabelRef("done"))
	b := GP64()
	MOVBQZX(Mem{Base: ptr}, b)
	XORQ(b, h)
	MULQ(p)
	INCQ(ptr)
	DECQ(n)

	JMP(LabelRef("loop"))
	Label("done")
	Store(h, ReturnIndex(0))
	RET()
	Generate()
}
```
