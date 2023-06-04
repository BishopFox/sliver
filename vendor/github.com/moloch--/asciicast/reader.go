package asciicast

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"io"
)

// ErrVersion is returned by Decoder/Reader if the version in the header is not version 2.
var ErrVersion = errors.New("asciicast: not a version 2 header")

// Decoder for asciicast v2 recordings.
type Decoder struct {
	r          *bufio.Reader
	headerRead bool
	headerErr  error
	header     Header
}

//  NewDecoder returns a new Decoder for the provided Reader.
func NewDecoder(r io.Reader) *Decoder {
	return &Decoder{
		r: bufio.NewReader(r),
	}
}

func (d *Decoder) readJSON(v interface{}) error {
	b, err := d.r.ReadBytes('\n')
	if err != nil {
		return err
	}
	return json.Unmarshal(b, v)
}

// DecodeHeader decodes the recording meta-data. DecodeHeader can be called multiple times and will only read the
// header from the Reader once.
func (d *Decoder) DecodeHeader() (*Header, error) {
	if !d.headerRead {
		if d.headerErr = d.readJSON(&d.header); d.headerErr == nil {
			if d.header.Version != Version {
				return nil, ErrVersion
			}
		}
		d.headerRead = true
	}
	return &d.header, d.headerErr
}

// DecodeEvent decodes the next Event.
func (d *Decoder) DecodeEvent() (*Event, error) {
	if _, err := d.DecodeHeader(); err != nil {
		return nil, err
	}
	e := new(Event)
	if err := d.readJSON(e); err != nil {
		return nil, err
	}
	return e, nil
}

// Reader for one event type.
type Reader struct {
	h         *Header
	d         *Decoder
	eventType EventType
	buf       bytes.Buffer
	err       error
}

const eventsChannelSize = 16

// NewReader returns a new reader for the selected EventType.
func NewReader(r io.Reader, eventType EventType) (*Reader, error) {
	d := &Reader{
		d:         NewDecoder(r),
		eventType: eventType,
	}
	h, err := d.d.DecodeHeader()
	if err != nil {
		return nil, err
	}
	d.h = h
	return d, nil
}

func (r *Reader) Read(p []byte) (n int, err error) {
	if r.err != nil {
		return 0, r.err
	}

	if r.buf.Len() > 0 {
		n, _ = r.buf.Read(p)
	}

	l := len(p)
	for n < l {
		var e *Event
		if e, r.err = r.d.DecodeEvent(); r.err != nil {
			return n, r.err
		} else if e.Type != r.eventType {
			continue
		}

		r.buf.WriteString(e.Data)
		m, _ := r.buf.Read(p[n:])
		n += m
	}
	return
}
