package sqlite3_wrap

import (
	"bytes"
	"encoding/binary"
	"math"

	"github.com/ncruces/go-sqlite3/internal/errutil"
)

const (
	PtrLen = 4
	IntLen = 4
)

type (
	Ptr_t uint32
	Res_t int32
)

func (mem *Memory) Bytes(ptr Ptr_t, size int64) []byte {
	if ptr == 0 {
		panic(errutil.NilErr)
	}
	return mem.Buf[ptr:][:size:size]
}

func (mem *Memory) Read(ptr Ptr_t) byte {
	if ptr == 0 {
		panic(errutil.NilErr)
	}
	return mem.Buf[ptr]
}

func (mem *Memory) Write(ptr Ptr_t, v byte) {
	if ptr == 0 {
		panic(errutil.NilErr)
	}
	mem.Buf[ptr] = v
}

func (mem *Memory) Read32(ptr Ptr_t) uint32 {
	if ptr == 0 {
		panic(errutil.NilErr)
	}
	return binary.LittleEndian.Uint32(mem.Buf[ptr:])
}

func (mem *Memory) Write32(ptr Ptr_t, v uint32) {
	if ptr == 0 {
		panic(errutil.NilErr)
	}
	binary.LittleEndian.PutUint32(mem.Buf[ptr:], v)
}

func (mem *Memory) Read64(ptr Ptr_t) uint64 {
	if ptr == 0 {
		panic(errutil.NilErr)
	}
	return binary.LittleEndian.Uint64(mem.Buf[ptr:])
}

func (mem *Memory) Write64(ptr Ptr_t, v uint64) {
	if ptr == 0 {
		panic(errutil.NilErr)
	}
	binary.LittleEndian.PutUint64(mem.Buf[ptr:], v)
}

func (mem *Memory) ReadFloat64(ptr Ptr_t) float64 {
	return math.Float64frombits(mem.Read64(ptr))
}

func (mem *Memory) WriteFloat64(ptr Ptr_t, v float64) {
	mem.Write64(ptr, math.Float64bits(v))
}

func (mem *Memory) ReadBool(ptr Ptr_t) bool {
	return mem.Read32(ptr) != 0
}

func (mem *Memory) WriteBool(ptr Ptr_t, v bool) {
	var i uint32
	if v {
		i = 1
	}
	mem.Write32(ptr, i)
}

func (mem *Memory) ReadString(ptr Ptr_t, maxlen int64) string {
	if ptr == 0 {
		panic(errutil.NilErr)
	}
	if maxlen <= 0 {
		return ""
	}
	buf := mem.Buf[ptr:]
	if int64(len(buf)-1) > maxlen {
		buf = buf[:maxlen+1]
	}
	if before, _, ok := bytes.Cut(buf, []byte{0}); ok {
		return string(before)
	}
	panic(errutil.NoNulErr)
}

func (mem *Memory) WriteBytes(ptr Ptr_t, b []byte) {
	buf := mem.Bytes(ptr, int64(len(b)))
	copy(buf, b)
}

func (mem *Memory) WriteString(ptr Ptr_t, s string) {
	buf := mem.Bytes(ptr, int64(len(s))+1)
	buf[len(s)] = 0
	copy(buf, s)
}
