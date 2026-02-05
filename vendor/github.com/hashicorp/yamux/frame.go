package yamux

import (
	"encoding/binary"
	"errors"
	"io"
)

const (
	frameTypeOpen  = 0x1
	frameTypeData  = 0x2
	frameTypeClose = 0x3

	frameHeaderLen = 12
)

var errProtocolViolation = errors.New("yamux: protocol violation")

type frame struct {
	typ uint8
	sid uint32
	buf []byte
}

func writeFrame(w io.Writer, f frame) error {
	var hdr [frameHeaderLen]byte
	hdr[0] = f.typ
	binary.BigEndian.PutUint32(hdr[4:8], f.sid)
	binary.BigEndian.PutUint32(hdr[8:12], uint32(len(f.buf)))
	if err := writeAll(w, hdr[:]); err != nil {
		return err
	}
	if len(f.buf) == 0 {
		return nil
	}
	return writeAll(w, f.buf)
}

func readFrame(r io.Reader, maxPayload uint32) (frame, error) {
	var hdr [frameHeaderLen]byte
	if _, err := io.ReadFull(r, hdr[:]); err != nil {
		return frame{}, err
	}
	typ := hdr[0]
	sid := binary.BigEndian.Uint32(hdr[4:8])
	length := binary.BigEndian.Uint32(hdr[8:12])
	if length > maxPayload {
		return frame{}, errProtocolViolation
	}
	var buf []byte
	if length > 0 {
		buf = make([]byte, length)
		if _, err := io.ReadFull(r, buf); err != nil {
			return frame{}, err
		}
	}
	return frame{typ: typ, sid: sid, buf: buf}, nil
}

func writeAll(w io.Writer, p []byte) error {
	for 0 < len(p) {
		n, err := w.Write(p)
		if err != nil {
			return err
		}
		p = p[n:]
	}
	return nil
}
