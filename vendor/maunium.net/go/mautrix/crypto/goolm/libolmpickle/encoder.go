package libolmpickle

import (
	"bytes"
	"encoding/binary"

	"go.mau.fi/util/exerrors"
)

const (
	PickleBoolLength   = 1
	PickleUInt8Length  = 1
	PickleUInt32Length = 4
)

type Encoder struct {
	bytes.Buffer
}

func NewEncoder() *Encoder { return &Encoder{} }

func (p *Encoder) WriteUInt8(value uint8) {
	exerrors.PanicIfNotNil(p.WriteByte(value))
}

func (p *Encoder) WriteBool(value bool) {
	if value {
		exerrors.PanicIfNotNil(p.WriteByte(0x01))
	} else {
		exerrors.PanicIfNotNil(p.WriteByte(0x00))
	}
}

func (p *Encoder) WriteEmptyBytes(count int) {
	exerrors.Must(p.Write(make([]byte, count)))
}

func (p *Encoder) WriteUInt32(value uint32) {
	exerrors.PanicIfNotNil(binary.Write(&p.Buffer, binary.BigEndian, value))
}
