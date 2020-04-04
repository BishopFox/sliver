package core

import (
	"bytes"
	"fmt"
	"io"
	"log"
)

const (
	randomIDSize = 16 // 64bits
)

type tunnelAddr struct {
	network string
	addr    string
}

func (a *tunnelAddr) Network() string {
	return a.network
}

func (a *tunnelAddr) String() string {
	return fmt.Sprintf("%s://%s", a.network, a.addr)
}

// Tunnel - Duplex data tunnel
type Tunnel struct {
	isOpen    bool
	SessionID uint32

	Send chan []byte
	Recv chan []byte
}

func (t *Tunnel) Write(data []byte) (int, error) {
	log.Printf("Sending %d bytes on session %d", len(data), t.SessionID)
	if !t.isOpen {
		return 0, io.EOF
	}
	t.Send <- data
	n := len(data)
	return n, nil
}

func (t *Tunnel) Read(data []byte) (int, error) {
	var buff bytes.Buffer
	if !t.isOpen {
		return 0, io.EOF
	}
	select {
	case msg := <-t.Recv:
		buff.Write(msg)
	default:
		break
	}
	n := copy(data, buff.Bytes())
	return n, nil
}

// Close - Close the tunnel channels
func (t *Tunnel) Close() {
	close(t.Recv)
	close(t.Send)
}
