//go:build !windows

package shell

import "io"

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

	return n, nil
}
