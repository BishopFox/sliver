package lark

import (
	"context"
	"io"
	"net/http"
)

// HTTPWrapper is a wrapper interface, which enables extension on HTTP part.
// Typicall, we do not need this because default client is sufficient.
type HTTPWrapper interface {
	Do(
		ctx context.Context,
		method, url string,
		header http.Header,
		body io.Reader) (io.ReadCloser, error)
}
