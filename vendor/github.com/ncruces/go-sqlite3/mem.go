package sqlite3

import (
	"bytes"
	"math"

	"github.com/tetratelabs/wazero/api"
)

type memory struct {
	mod api.Module
}

func (m memory) view(ptr uint32, size uint64) []byte {
	if ptr == 0 {
		panic(nilErr)
	}
	if size > math.MaxUint32 {
		panic(rangeErr)
	}
	buf, ok := m.mod.Memory().Read(ptr, uint32(size))
	if !ok {
		panic(rangeErr)
	}
	return buf
}

func (m memory) readUint8(ptr uint32) uint8 {
	if ptr == 0 {
		panic(nilErr)
	}
	v, ok := m.mod.Memory().ReadByte(ptr)
	if !ok {
		panic(rangeErr)
	}
	return v
}

func (m memory) writeUint8(ptr uint32, v uint8) {
	if ptr == 0 {
		panic(nilErr)
	}
	ok := m.mod.Memory().WriteByte(ptr, v)
	if !ok {
		panic(rangeErr)
	}
}

func (m memory) readUint32(ptr uint32) uint32 {
	if ptr == 0 {
		panic(nilErr)
	}
	v, ok := m.mod.Memory().ReadUint32Le(ptr)
	if !ok {
		panic(rangeErr)
	}
	return v
}

func (m memory) writeUint32(ptr uint32, v uint32) {
	if ptr == 0 {
		panic(nilErr)
	}
	ok := m.mod.Memory().WriteUint32Le(ptr, v)
	if !ok {
		panic(rangeErr)
	}
}

func (m memory) readUint64(ptr uint32) uint64 {
	if ptr == 0 {
		panic(nilErr)
	}
	v, ok := m.mod.Memory().ReadUint64Le(ptr)
	if !ok {
		panic(rangeErr)
	}
	return v
}

func (m memory) writeUint64(ptr uint32, v uint64) {
	if ptr == 0 {
		panic(nilErr)
	}
	ok := m.mod.Memory().WriteUint64Le(ptr, v)
	if !ok {
		panic(rangeErr)
	}
}

func (m memory) readFloat64(ptr uint32) float64 {
	return math.Float64frombits(m.readUint64(ptr))
}

func (m memory) writeFloat64(ptr uint32, v float64) {
	m.writeUint64(ptr, math.Float64bits(v))
}

func (m memory) readString(ptr, maxlen uint32) string {
	if ptr == 0 {
		panic(nilErr)
	}
	switch maxlen {
	case 0:
		return ""
	case math.MaxUint32:
		// avoid overflow
	default:
		maxlen = maxlen + 1
	}
	mem := m.mod.Memory()
	buf, ok := mem.Read(ptr, maxlen)
	if !ok {
		buf, ok = mem.Read(ptr, mem.Size()-ptr)
		if !ok {
			panic(rangeErr)
		}
	}
	if i := bytes.IndexByte(buf, 0); i < 0 {
		panic(noNulErr)
	} else {
		return string(buf[:i])
	}
}

func (m memory) writeBytes(ptr uint32, b []byte) {
	buf := m.view(ptr, uint64(len(b)))
	copy(buf, b)
}

func (m memory) writeString(ptr uint32, s string) {
	buf := m.view(ptr, uint64(len(s)+1))
	buf[len(s)] = 0
	copy(buf, s)
}
