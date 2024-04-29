package util

import (
	"bytes"
	"math"

	"github.com/tetratelabs/wazero/api"
)

func View(mod api.Module, ptr uint32, size uint64) []byte {
	if ptr == 0 {
		panic(NilErr)
	}
	if size > math.MaxUint32 {
		panic(RangeErr)
	}
	if size == 0 {
		return nil
	}
	buf, ok := mod.Memory().Read(ptr, uint32(size))
	if !ok {
		panic(RangeErr)
	}
	return buf
}

func ReadUint8(mod api.Module, ptr uint32) uint8 {
	if ptr == 0 {
		panic(NilErr)
	}
	v, ok := mod.Memory().ReadByte(ptr)
	if !ok {
		panic(RangeErr)
	}
	return v
}

func ReadUint32(mod api.Module, ptr uint32) uint32 {
	if ptr == 0 {
		panic(NilErr)
	}
	v, ok := mod.Memory().ReadUint32Le(ptr)
	if !ok {
		panic(RangeErr)
	}
	return v
}

func WriteUint8(mod api.Module, ptr uint32, v uint8) {
	if ptr == 0 {
		panic(NilErr)
	}
	ok := mod.Memory().WriteByte(ptr, v)
	if !ok {
		panic(RangeErr)
	}
}

func WriteUint32(mod api.Module, ptr uint32, v uint32) {
	if ptr == 0 {
		panic(NilErr)
	}
	ok := mod.Memory().WriteUint32Le(ptr, v)
	if !ok {
		panic(RangeErr)
	}
}

func ReadUint64(mod api.Module, ptr uint32) uint64 {
	if ptr == 0 {
		panic(NilErr)
	}
	v, ok := mod.Memory().ReadUint64Le(ptr)
	if !ok {
		panic(RangeErr)
	}
	return v
}

func WriteUint64(mod api.Module, ptr uint32, v uint64) {
	if ptr == 0 {
		panic(NilErr)
	}
	ok := mod.Memory().WriteUint64Le(ptr, v)
	if !ok {
		panic(RangeErr)
	}
}

func ReadFloat64(mod api.Module, ptr uint32) float64 {
	return math.Float64frombits(ReadUint64(mod, ptr))
}

func WriteFloat64(mod api.Module, ptr uint32, v float64) {
	WriteUint64(mod, ptr, math.Float64bits(v))
}

func ReadString(mod api.Module, ptr, maxlen uint32) string {
	if ptr == 0 {
		panic(NilErr)
	}
	switch maxlen {
	case 0:
		return ""
	case math.MaxUint32:
		// avoid overflow
	default:
		maxlen = maxlen + 1
	}
	mem := mod.Memory()
	buf, ok := mem.Read(ptr, maxlen)
	if !ok {
		buf, ok = mem.Read(ptr, mem.Size()-ptr)
		if !ok {
			panic(RangeErr)
		}
	}
	if i := bytes.IndexByte(buf, 0); i < 0 {
		panic(NoNulErr)
	} else {
		return string(buf[:i])
	}
}

func WriteBytes(mod api.Module, ptr uint32, b []byte) {
	buf := View(mod, ptr, uint64(len(b)))
	copy(buf, b)
}

func WriteString(mod api.Module, ptr uint32, s string) {
	buf := View(mod, ptr, uint64(len(s)+1))
	buf[len(s)] = 0
	copy(buf, s)
}
