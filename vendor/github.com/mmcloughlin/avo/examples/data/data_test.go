package data

import (
	"encoding/binary"
	"math"
	"testing"
)

//go:generate go run asm.go -out data.s -stubs stub.go

func TestDataAt(t *testing.T) {
	order := binary.LittleEndian
	expect := make([]byte, 40)
	order.PutUint64(expect[0:], 0x0011223344556677)         // DATA(0, U64(0x0011223344556677))
	copy(expect[8:], []byte("strconst"))                    // DATA(8, String("strconst"))
	order.PutUint32(expect[16:], math.Float32bits(math.Pi)) // DATA(16, F32(math.Pi))
	order.PutUint64(expect[24:], math.Float64bits(math.Pi)) // DATA(24, F64(math.Pi))
	order.PutUint32(expect[32:], 0x00112233)                // DATA(32, U32(0x00112233))
	order.PutUint16(expect[36:], 0x4455)                    // DATA(36, U16(0x4455))
	expect[38] = 0x66                                       // DATA(38, U8(0x66))
	expect[39] = 0x77                                       // DATA(39, U8(0x77))

	for i, e := range expect {
		b := DataAt(i)
		if b != e {
			t.Errorf("DataAt(%d) = %#02x; expected %#02x", i, b, e)
		}
	}
}
