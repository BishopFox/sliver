package core

/*
	Sliver Implant Framework
	Copyright (C) 2021  Bishop Fox

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
	"sync"

	"github.com/bishopfox/sliver/protobuf/rpcpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
)

var (
	// TunSocksTunnels - Interacting with duplex SocksTunnels
	SocksTunnels = tcpTunnel{
		tunnels: map[uint64]*TcpTunnel{},
		mutex:   &sync.Mutex{},
	}
)

type TcpTunnel struct {
	ID                uint64
	SessionID         string
	ToImplantSequence uint64
	ToImplantMux      sync.Mutex

	FromImplant         chan *sliverpb.SocksData
	FromImplantSequence uint64
	Client              rpcpb.SliverRPC_SocksProxyServer
}

type tcpTunnel struct {
	tunnels map[uint64]*TcpTunnel
	mutex   *sync.Mutex
}

func (t *tcpTunnel) Create(sessionID string) *TcpTunnel {
	tunnelID := NewTunnelID()
	session := Sessions.Get(sessionID)
	tunnel := &TcpTunnel{
		ID:        tunnelID,
		SessionID: session.ID,
		//ToImplant:   make(chan []byte),
		FromImplant: make(chan *sliverpb.SocksData),
	}
	t.mutex.Lock()
	defer t.mutex.Unlock()
	t.tunnels[tunnel.ID] = tunnel

	return tunnel
}

func (t *tcpTunnel) Close(tunnelID uint64) error {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	tunnel := t.tunnels[tunnelID]
	if tunnel == nil {
		return ErrInvalidTunnelID
	}
	delete(t.tunnels, tunnelID)
	//close(tunnel.ToImplant)
	close(tunnel.FromImplant)
	return nil
}

func (t *tcpTunnel) Get(tunnelID uint64) *TcpTunnel {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	return t.tunnels[tunnelID]
}
