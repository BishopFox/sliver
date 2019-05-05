# geohash

Integer [geohash](https://en.wikipedia.org/wiki/Geohash) encoding.

Refer to ["Geohash in Golang Assembly"](https://mmcloughlin.com/posts/geohash-assembly) for implementation details.

[embedmd]:# (asm.go /func main/ $)
```go
func main() {
	TEXT("EncodeInt", NOSPLIT, "func(lat, lng float64) uint64")
	Doc("EncodeInt computes the 64-bit integer geohash of (lat, lng).")
	lat := Load(Param("lat"), XMM())
	lng := Load(Param("lng"), XMM())

	MULSD(ConstData("reciprocal180", F64(1/180.0)), lat)
	onepointfive := ConstData("onepointfive", F64(1.5))
	ADDSD(onepointfive, lat)

	MULSD(ConstData("reciprocal360", F64(1/360.0)), lng)
	ADDSD(onepointfive, lng)

	lngi, lati := GP64(), GP64()
	MOVQ(lat, lati)
	SHRQ(U8(20), lati)
	MOVQ(lng, lngi)
	SHRQ(U8(20), lngi)

	mask := ConstData("mask", U64(0x5555555555555555))
	ghsh := GP64()
	PDEPQ(mask, lati, ghsh)
	temp := GP64()
	PDEPQ(mask, lngi, temp)
	SHLQ(U8(1), temp)
	XORQ(temp, ghsh)

	Store(ghsh, ReturnIndex(0))
	RET()

	Generate()
}
```
