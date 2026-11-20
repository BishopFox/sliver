//go:build !unix && !windows

package sqlite3_wrap

import "math"

type Memory struct {
	Buf []byte
	Max int64
}

func (m *Memory) Slice() *[]byte {
	return &m.Buf
}

func (m *Memory) Grow(delta, max int64) int64 {
	len := int64(len(m.Buf))
	old := len >> 16
	if delta == 0 {
		return old
	}
	new := old + delta
	max = min(max, m.Max, int64(math.MaxInt)>>16)
	if new > max || new < old {
		return -1
	}
	m.Buf = append(m.Buf, make([]byte, new<<16-len)...)
	return old
}

func (m *Memory) Close() error {
	m.Buf = nil
	return nil
}
