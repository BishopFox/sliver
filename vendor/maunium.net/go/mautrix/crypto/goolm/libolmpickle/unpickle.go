package libolmpickle

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

func isZeroByteSlice(data []byte) bool {
	for _, b := range data {
		if b != 0 {
			return false
		}
	}
	return true
}

type Decoder struct {
	buf bytes.Buffer
}

func NewDecoder(buf []byte) *Decoder {
	return &Decoder{buf: *bytes.NewBuffer(buf)}
}

func (d *Decoder) ReadUInt8() (uint8, error) {
	return d.buf.ReadByte()
}

func (d *Decoder) ReadBool() (bool, error) {
	val, err := d.buf.ReadByte()
	return val != 0x00, err
}

func (d *Decoder) ReadBytes(length int) (data []byte, err error) {
	data = d.buf.Next(length)
	if len(data) != length {
		return nil, fmt.Errorf("only %d in buffer, expected %d", len(data), length)
	} else if isZeroByteSlice(data) {
		return nil, nil
	}
	return
}

func (d *Decoder) ReadUInt32() (uint32, error) {
	data := d.buf.Next(4)
	if len(data) != 4 {
		return 0, fmt.Errorf("only %d bytes is buffer, expected 4 for uint32", len(data))
	} else {
		return binary.BigEndian.Uint32(data), nil
	}
}
