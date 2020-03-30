package core

import (
	"bytes"
	"io"
	"log"
)

/*
	Sliver Implant Framework
	Copyright (C) 2019  Bishop Fox

	This program is free software: you can redistribute it and/or modify
	it under the terms of the GNU General Public License as published by
	the Free Software Foundation, either version 3 of the License, or
	(at your option) any later version.

	This program is distributed in the hope that it will be useful,
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	GNU General Public License for more details.

	You should have received a copy of the GNU General Public License
	along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

// import (
// 	"bytes"
// 	"fmt"
// 	"io"
// 	"log"
// 	"sync"
// 	"time"

// 	"github.com/bishopfox/sliver/protobuf/rpcpb"
// 	"github.com/bishopfox/sliver/protobuf/sliverpb"
// 	"github.com/golang/protobuf/proto"
// )

// const (
// 	randomIDSize = 16 // 64bits
// )

// type tunnelAddr struct {
// 	network string
// 	addr    string
// }

// func (a *tunnelAddr) Network() string {
// 	return a.network
// }

// func (a *tunnelAddr) String() string {
// 	return fmt.Sprintf("%s://%s", a.network, a.addr)
// }

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
