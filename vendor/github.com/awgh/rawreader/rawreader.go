package rawreader

import (
	"errors"
	"io"
	"reflect"
	"unsafe"
)

// RawReader implements interfaces implemented by files over raw memory
//		this is completely unsafe by design
// 		you should either know why you want this exactly or stay away!
type RawReader struct {
	sliceHeader *reflect.SliceHeader
	rawPtr      uintptr
	Data        []byte
	Length      int
}

// New returns a new populated RawReader
func New(start uintptr, length int) *RawReader {
	sh := &reflect.SliceHeader{
		Data: start,
		Len:  length,
		Cap:  length,
	}
	data := *(*[]byte)(unsafe.Pointer(sh))
	return &RawReader{sliceHeader: sh, rawPtr: start, Data: data, Length: length}
}

// ReadAt implements io.ReaderAt https://golang.org/pkg/io/#ReaderAt
// ReadAt reads len(p) bytes into p starting at offset off in the underlying input source.
// It returns the number of bytes read (0 <= n <= len(p)) and any error encountered.
//
// When ReadAt returns n < len(p), it returns a non-nil error explaining why more bytes were not returned.
// In this respect, ReadAt is stricter than Read.
//
// Even if ReadAt returns n < len(p), it may use all of p as scratch space during the call.
// If some data is available but not len(p) bytes, ReadAt blocks until either all the data is available or an error occurs.
// In this respect ReadAt is different from Read.
//
// If the n = len(p) bytes returned by ReadAt are at the end of the input source,
// ReadAt may return either err == EOF or err == nil.
//
// If ReadAt is reading from an input source with a seek offset,
// ReadAt should not affect nor be affected by the underlying seek offset.
// Clients of ReadAt can execute parallel ReadAt calls on the same input source.
func (f *RawReader) ReadAt(p []byte, off int64) (n int, err error) {
	if off < 0 {
		return 0, errors.New("RawReader.ReadAt: negative offset")
	}
	reqLen := len(p)
	buffLen := int64(f.Length)
	if off >= buffLen {
		return 0, io.EOF
	}

	n = copy(p, f.Data[off:])
	if n < reqLen {
		err = io.EOF
	}
	return n, err
}
