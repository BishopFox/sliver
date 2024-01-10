//go:build go1.20
// +build go1.20

package ws

import (
	"bufio"
	"errors"
	"net"
	"net/http"
)

func hijack(w http.ResponseWriter) (net.Conn, *bufio.ReadWriter, error) {
	conn, rw, err := http.NewResponseController(w).Hijack()
	if errors.Is(err, http.ErrNotSupported) {
		return nil, nil, ErrNotHijacker
	}
	return conn, rw, err
}
