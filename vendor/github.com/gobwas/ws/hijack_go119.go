//go:build !go1.20
// +build !go1.20

package ws

import (
	"bufio"
	"net"
	"net/http"
)

func hijack(w http.ResponseWriter) (net.Conn, *bufio.ReadWriter, error) {
	hj, ok := w.(http.Hijacker)
	if ok {
		return hj.Hijack()
	}
	return nil, nil, ErrNotHijacker
}
