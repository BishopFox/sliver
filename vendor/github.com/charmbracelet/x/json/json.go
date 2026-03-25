// Package json provides functions to facilitate dealing with JSON.
package json

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// Reader takes an input, marshal it to JSON and returns a io.Reader of it.
func Reader[T any](v T) io.Reader {
	bts, err := json.Marshal(v)
	if err != nil {
		return &ErrorReader{err}
	}
	return bytes.NewReader(bts)
}

// ErrorReader is a reader that always errors with the given error.
type ErrorReader struct {
	err error
}

func (r *ErrorReader) Read(_ []byte) (int, error) {
	return 0, r.err
}

// From parses a io.Reader with JSON.
func From[T any](r io.Reader, t T) (T, error) {
	bts, err := io.ReadAll(r)
	if err != nil {
		return t, fmt.Errorf("failed to read response: %w", err)
	}
	if err := json.Unmarshal(bts, &t); err != nil {
		return t, fmt.Errorf("failed to parse body: %w: %s", err, bts)
	}
	return t, nil
}

// Write writes the given data as JSON.
func Write(w http.ResponseWriter, data any) error {
	bts, err := json.Marshal(data)
	if err != nil {
		return err //nolint:wrapcheck
	}
	w.Header().Add("Content-Type", "application/json")
	_, err = w.Write(bts)
	return err //nolint:wrapcheck
}

// IsValid checks if the given data is valid JSON.
func IsValid[T string | []byte](data T) bool {
	if len(data) == 0 { // hot path
		return false
	}
	var m json.RawMessage
	err := json.Unmarshal([]byte(data), &m)
	return err == nil
}
