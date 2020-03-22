package core

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

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"sync"
	"time"

	"github.com/bishopfox/sliver/protobuf/rpcpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/golang/protobuf/proto"
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
	rpc       *rpcpb.SliverRPCClient
	SessionID uint32
	ID        uint64
	isOpen    bool
}

func (t *tunnel) Write(data []byte) (n int, err error) {
	log.Printf("Sending %d bytes on tunnel %d (sliver %d)", len(data), t.ID, t.SessionID)
	if !t.isOpen {
		return 0, io.EOF
	}
	rpc.Tunnel(context.Background()).Send(&sliverpb.TunnelData{
		SessionID: t.SessionID,
		TunnelID:  t.ID,
		Data:      data,
	})

	n = len(data)
	return
}

func (t *tunnel) Read(data []byte) (n int, err error) {
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
	n = copy(data, buff.Bytes())
	return
}

func (t *tunnel) Close() error {
	tunnelClose, err := proto.Marshal(&sliverpb.ShellReq{
		TunnelID: t.ID,
	})
	t.server.RPC(&sliverpb.Envelope{
		Type: sliverpb.MsgTunnelClose,
		Data: tunnelClose,
	}, 30*time.Second)
	close(t.Recv)
	return err
}
