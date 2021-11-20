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
	toImplantCacheSocks = map[uint64]sync.Map{}

	// SessionID->Tunnels[TunnelID]->Tunnel->Cache
	fromImplantCacheSocks = map[uint64]map[uint64]*sliverpb.SocksData{}
)

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

					recvCache, ok := fromImplantCacheSocks[fromClient.TunnelID]
					if ok {
						recvCache[tunnelData.Sequence] = tunnelData
					}
					for recv, ok := recvCache[socks.FromImplantSequence]; ok; recv, ok = recvCache[socks.FromImplantSequence] {
						rpcLog.Debugf("[socks] agent to (Server To Client)  Data Sequence %d , Data Size %d ,Data %v\n", socks.FromImplantSequence, len(recv.Data), recv.Data)
						socks.Client.Send(&sliverpb.SocksData{
							CloseConn: recv.CloseConn,
							TunnelID:  recv.TunnelID,
							Data:      recv.Data,
						})
						delete(recvCache, socks.FromImplantSequence)
						socks.FromImplantSequence++
					}

				}
			}()
		}

		// Send Agent
		go func() {
			recvCache, ok := toImplantCacheSocks[fromClient.TunnelID]
			if ok {
				recvCache.Store(fromClient.Sequence, &sliverpb.SocksData{
					TunnelID: fromClient.TunnelID,
					Username: fromClient.Username,
					Password: fromClient.Password,
					Data:     fromClient.Data,
					Sequence: fromClient.Sequence,
				})

			}

			for recv, ok := recvCache.Load(socks.ToImplantSequence); ok; recv, ok = recvCache.Load(socks.ToImplantSequence) {
				rpcLog.Debugf("[socks] Client to (Server To Agent) Data Sequence %d ,  Data Size %d \n", socks.ToImplantSequence, len(fromClient.Data))
				data, _ := proto.Marshal(recv.(*sliverpb.SocksData))

				session := core.Sessions.Get(fromClient.Request.SessionID)
				session.Connection.Send <- &sliverpb.Envelope{
					Type: sliverpb.MsgSocksData,
					Data: data,
				}
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
	toImplantCacheSocks[tunnel.ID] = sync.Map{}
	fromImplantCacheSocks[tunnel.ID] = map[uint64]*sliverpb.SocksData{}
	return &sliverpb.Socks{
		SessionID: session.ID,
		TunnelID:  tunnel.ID,
	}, nil
}

// CloseSocks - Client requests we close a Socks
func (s *Server) CloseSocks(ctx context.Context, req *sliverpb.Socks) (*commonpb.Empty, error) {
	err := core.SocksTunnels.Close(req.TunnelID)

	if _, ok := toImplantCacheSocks[req.TunnelID]; ok {
		delete(toImplantCacheSocks, req.TunnelID)
	}

	if _, ok := fromImplantCacheSocks[req.TunnelID]; ok {
		delete(fromImplantCacheSocks, req.TunnelID)
	}

	if err != nil {
		return nil, err
	}
	return &commonpb.Empty{}, nil
}
