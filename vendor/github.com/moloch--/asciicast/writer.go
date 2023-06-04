package asciicast

import (
	"encoding/json"
	"io"
	"time"
)

type Encoder struct {
	// Header contains recording meta-data.
	Header

	w          io.Writer
	start      time.Time
	header     Header
	headerSent bool
}

// NewEncoder can emit an asciicast v2 stream.
func NewEncoder(w io.Writer, width, height int) *Encoder {
	return NewEncoderEx(w, Header{
		Width:  width,
		Height: height,
	})
}

// NewEncoderEx can emit an asciicast v2 stream.
func NewEncoderEx(w io.Writer, header Header) *Encoder {
	header.Version = 2
	if header.Timestamp == 0 {
		header.Timestamp = time.Now().Unix()
	}
	return &Encoder{
		Header: header,
		w:      w,
	}
}

func (e *Encoder) writeJSON(v interface{}) error {
	/*
		var (
			buf bytes.Buffer
			err = json.NewEncoder(&buf).Encode(v)
		)
		if err == nil {
			_, err = buf.WriteTo(e.w)
		}
		return err
	*/
	return json.NewEncoder(e.w).Encode(v)
}

// WriteHeader writes the recording meta-data.
func (e *Encoder) WriteHeader() error {
	if e.headerSent {
		return nil
	}
	e.headerSent = true
	e.start = time.Now()
	return e.writeJSON(&e.Header)
}

// Write writes output. If WriteHeader wasn't called, it will be called first.
func (e *Encoder) Write(p []byte) (int, error) {
	return len(p), e.WriteEvent(Output, string(p))
}

// WriteInput writes input. If WriteHeader wasn't called, it will be called first.
func (e *Encoder) WriteInput(p []byte) (int, error) {
	return len(p), e.WriteEvent(Input, string(p))
}

// WriteEvent writes an event. If WriteHeader wasn't called, it will be called first.
func (e *Encoder) WriteEvent(kind EventType, data string) error {
	if !e.headerSent {
		if err := e.WriteHeader(); err != nil {
			return err
		}
	}
	return e.writeJSON(&Event{
		Time: time.Since(e.start).Seconds(),
		Type: kind,
		Data: data,
	})
}

// WriteRawEvent writes a raw event.
func (e *Encoder) WriteRawEvent(event Event) error {
	return e.writeJSON(&event)
}
