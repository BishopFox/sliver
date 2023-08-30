package shell

import (
	"io"
)

type filterReader struct {
	reader io.Reader
}

func newFilterReader(reader io.Reader) *filterReader {
	return &filterReader{reader}
}

func (r *filterReader) Read(data []byte) (int, error) {
	n, err := r.reader.Read(data)
	if err != nil {
		return n, err
	}

	// Remove Windows new line
	if n >= 2 {
		if data[n-2] == byte('\r') {
			data = data[0 : n-2]
			data = append(data, byte('\n'))
			n -= 1
		}
	}

	return n, nil
}
