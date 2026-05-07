// Copyright (c) 2024 Tulir Asokan
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package exbytes

import (
	"errors"
	"fmt"
	"io"
)

// Writer is a simple byte writer that does not allow extending the buffer.
//
// Writes always go after the current len() and will fail if the slice capacity is too low.
//
// The correct way to use this is to create a slice with sufficient capacity and zero length:
//
//	x := make([]byte, 0, 11)
//	w := (*Writer)(&x)
//	w.Write([]byte("hello"))
//	w.WriteByte(' ')
//	w.WriteString("world")
//	fmt.Println(string(x)) // "hello world"
type Writer []byte

var ErrWriterBufferFull = errors.New("exbytes.Writer: buffer full")

var (
	_ io.Writer       = (*Writer)(nil)
	_ io.StringWriter = (*Writer)(nil)
	_ io.ByteWriter   = (*Writer)(nil)
)

func (w *Writer) extendLen(n int) (int, error) {
	ptr := len(*w)
	if ptr+n > cap(*w) {
		return 0, fmt.Errorf("%w (%d + %d > %d)", ErrWriterBufferFull, ptr, n, cap(*w))
	}
	*w = (*w)[:ptr+n]
	return ptr, nil
}

func (w *Writer) Write(b []byte) (n int, err error) {
	ptr, err := w.extendLen(len(b))
	if err != nil {
		return 0, err
	}
	copy((*w)[ptr:], b)
	return len(b), nil
}

func (w *Writer) WriteString(s string) (n int, err error) {
	ptr, err := w.extendLen(len(s))
	if err != nil {
		return 0, err
	}
	copy((*w)[ptr:], s)
	return len(s), nil
}

func (w *Writer) WriteByte(r byte) error {
	ptr, err := w.extendLen(1)
	if err != nil {
		return err
	}
	(*w)[ptr] = r
	return nil
}

func (w *Writer) String() string {
	if w == nil {
		return "<nil>"
	}
	return string(*w)
}
