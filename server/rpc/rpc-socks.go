package rpc

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
	"context"
	"io"
	"sync"

	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/rpcpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/bishopfox/sliver/server/core"
	"google.golang.org/protobuf/proto"
)

var (
	// SessionID->Tunnels[TunnelID]->Tunnel->Cache map[uint64]*sliverpb.SocksData{}
	toImplantCacheSocks = socksDataCache{mutex: &sync.RWMutex{}, cache: map[uint64]map[uint64]*sliverpb.SocksData{}}

	// SessionID->Tunnels[TunnelID]->Tunnel->Cache
	fromImplantCacheSocks = socksDataCache{mutex: &sync.RWMutex{}, cache: map[uint64]map[uint64]*sliverpb.SocksData{}}
)

type socksDataCache struct {
	mutex *sync.RWMutex
	cache map[uint64]map[uint64]*sliverpb.SocksData
}

func (c *socksDataCache) Add(tunnelID uint64, sequence uint64, tunnelData *sliverpb.SocksData) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if _, ok := c.cache[tunnelID]; !ok {
		c.cache[tunnelID] = map[uint64]*sliverpb.SocksData{}
	}

	c.cache[tunnelID][sequence] = tunnelData
}

func (c *socksDataCache) Get(tunnelID uint64, sequence uint64) (*sliverpb.SocksData, bool) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	if _, ok := c.cache[tunnelID]; !ok {
		return nil, false
	}

	val, ok := c.cache[tunnelID][sequence]

	return val, ok
}

func (c *socksDataCache) DeleteTun(tunnelID uint64) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	delete(c.cache, tunnelID)
}

func (c *socksDataCache) DeleteSeq(tunnelID uint64, sequence uint64) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if _, ok := c.cache[tunnelID]; !ok {
		return
	}

	delete(c.cache[tunnelID], sequence)
}

// Socks - Open an in-band port forward
func (s *Server) SocksProxy(stream rpcpb.SliverRPC_SocksProxyServer) error {
	for {
		fromClient, err := stream.Recv()
		if err == io.EOF {
			break
		}
		//fmt.Println("Send Agent 1 ",fromClient.TunnelID,len(fromClient.Data))
		if err != nil {
			rpcLog.Warnf("Error on stream recv %s", err)
			return err
		}
		tunnelLog.Debugf("Tunnel %d: From client %d byte(s)",
			fromClient.TunnelID, len(fromClient.Data))
		socks := core.SocksTunnels.Get(fromClient.TunnelID)
		if socks == nil {
			return nil
		}
		if socks.Client == nil {
			socks.Client = stream // Bind client to tunnel
			// Send Client
			go func() {
				for tunnelData := range socks.FromImplant {

					fromImplantCacheSocks.Add(fromClient.TunnelID, tunnelData.Sequence, tunnelData)

					for recv, ok := fromImplantCacheSocks.Get(fromClient.TunnelID, socks.FromImplantSequence); ok; recv, ok = fromImplantCacheSocks.Get(fromClient.TunnelID, socks.FromImplantSequence) {
						rpcLog.Debugf("[socks] agent to (Server To Client)  Data Sequence %d , Data Size %d ,Data %v\n", socks.FromImplantSequence, len(recv.Data), recv.Data)
						socks.Client.Send(&sliverpb.SocksData{
							CloseConn: recv.CloseConn,
							TunnelID:  recv.TunnelID,
							Data:      recv.Data,
						})

						fromImplantCacheSocks.DeleteSeq(fromClient.TunnelID, socks.FromImplantSequence)
						socks.FromImplantSequence++
					}

				}
			}()
		}

		// Send Agent
		go func() {
			toImplantCacheSocks.Add(fromClient.TunnelID, fromClient.Sequence, fromClient)

			for recv, ok := toImplantCacheSocks.Get(fromClient.TunnelID, socks.ToImplantSequence); ok; recv, ok = toImplantCacheSocks.Get(fromClient.TunnelID, socks.ToImplantSequence) {
				rpcLog.Debugf("[socks] Client to (Server To Agent) Data Sequence %d ,  Data Size %d \n", socks.ToImplantSequence, len(fromClient.Data))
				data, _ := proto.Marshal(recv)

				session := core.Sessions.Get(fromClient.Request.SessionID)
				session.Connection.Send <- &sliverpb.Envelope{
					Type: sliverpb.MsgSocksData,
					Data: data,
				}

				toImplantCacheSocks.DeleteSeq(fromClient.TunnelID, socks.ToImplantSequence)
				socks.ToImplantSequence++
			}

		}()
	}
	return nil
}

// CreateSocks5 - Create requests we close a Socks
func (s *Server) CreateSocks(ctx context.Context, req *sliverpb.Socks) (*sliverpb.Socks, error) {
	session := core.Sessions.Get(req.SessionID)
	if session == nil {
		return nil, ErrInvalidSessionID
	}
	tunnel := core.SocksTunnels.Create(session.ID)
	if tunnel == nil {
		return nil, ErrTunnelInitFailure
	}

	return &sliverpb.Socks{
		SessionID: session.ID,
		TunnelID:  tunnel.ID,
	}, nil
}

// CloseSocks - Client requests we close a Socks
func (s *Server) CloseSocks(ctx context.Context, req *sliverpb.Socks) (*commonpb.Empty, error) {
	err := core.SocksTunnels.Close(req.TunnelID)
	toImplantCacheSocks.DeleteTun(req.TunnelID)
	fromImplantCacheSocks.DeleteTun(req.TunnelID)
	if err != nil {
		return nil, err
	}
	return &commonpb.Empty{}, nil
}
