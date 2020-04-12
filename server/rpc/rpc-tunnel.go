package rpc

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
	"context"
	"io"

	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/rpcpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/bishopfox/sliver/server/core"
)

// CreateTunnel - Create a new tunnel on the server, however based on only this request there's
//                no way to associate the tunnel with the correct client, so the client must send
//                a zero-byte message over TunnelData to bind itself to the newly created tunnel.
func (s *Server) CreateTunnel(ctx context.Context, req *sliverpb.Tunnel) (*sliverpb.Tunnel, error) {
	session := core.Sessions.Get(req.SessionID)
	if session == nil {
		return nil, ErrInvalidSessionID
	}
	tunnel := core.Tunnels.Create(session.ID)
	if tunnel == nil {
		return nil, ErrTunnelInitFailure
	}
	return &sliverpb.Tunnel{
		SessionID: session.ID,
		TunnelID:  tunnel.ID,
	}, nil
}

// CloseTunnel - Client requests we close a tunnel
func (s *Server) CloseTunnel(ctx context.Context, req *sliverpb.Tunnel) (*commonpb.Empty, error) {
	err := core.Tunnels.Close(req.TunnelID)
	if err != nil {
		return nil, err
	}
	return &commonpb.Empty{}, nil
}

// TunnelData - Streams tunnel data back and forth from the client<->server<->implant
func (s *Server) TunnelData(stream rpcpb.SliverRPC_TunnelDataServer) error {
	for {
		fromClient, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			rpcLog.Warn("Error on stream recv %s", err)
			return err
		}

		tunnel := core.Tunnels.Get(fromClient.TunnelID)
		if tunnel == nil {
			return core.ErrInvalidTunnelID
		}
		if tunnel.Client == nil {
			tunnel.Client = stream // Bind client to tunnel
			tunnel.Client.Send(&sliverpb.TunnelData{
				TunnelID:  tunnel.ID,
				SessionID: tunnel.SessionID,
				Closed:    false,
			})

			go func() {
				for data := range tunnel.FromImplant {
					tunnel.Client.Send(&sliverpb.TunnelData{
						TunnelID:  tunnel.ID,
						SessionID: tunnel.SessionID,
						Data:      data,
						Closed:    false,
					})
				}
				tunnel.Client.Send(&sliverpb.TunnelData{
					TunnelID:  tunnel.ID,
					SessionID: tunnel.SessionID,
					Closed:    true,
				})
			}()

		} else if tunnel.Client == stream {
			tunnel.ToImplant <- fromClient.GetData()
		}
	}
	return nil
}
