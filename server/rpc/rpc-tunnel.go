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
	"sync"

	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/rpcpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/bishopfox/sliver/server/core"
	"github.com/bishopfox/sliver/server/log"
	"google.golang.org/protobuf/proto"
)

var (
	tunnelLog = log.NamedLogger("rpc", "tunnel")

	// SessionID->Tunnels[TunnelID]->Tunnel->Cache
	toImplantCache = dataCache{mutex: &sync.RWMutex{}, cache: map[uint64]map[uint64]*sliverpb.TunnelData{}}

	// SessionID->Tunnels[TunnelID]->Tunnel->Cache
	fromImplantCache = dataCache{mutex: &sync.RWMutex{}, cache: map[uint64]map[uint64]*sliverpb.TunnelData{}}
)

type dataCache struct {
	mutex *sync.RWMutex
	cache map[uint64]map[uint64]*sliverpb.TunnelData
}

func (c *dataCache) Add(tunnelID uint64, sequence uint64, tunnelData *sliverpb.TunnelData) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if _, ok := c.cache[tunnelID]; !ok {
		c.cache[tunnelID] = map[uint64]*sliverpb.TunnelData{}
	}

	c.cache[tunnelID][sequence] = tunnelData
}

func (c *dataCache) Get(tunnelID uint64, sequence uint64) (*sliverpb.TunnelData, bool) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	if _, ok := c.cache[tunnelID]; !ok {
		return nil, false
	}

	val, ok := c.cache[tunnelID][sequence]

	return val, ok
}

func (c *dataCache) DeleteTun(tunnelID uint64) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	delete(c.cache, tunnelID)
}

func (c *dataCache) DeleteSeq(tunnelID uint64, sequence uint64) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if _, ok := c.cache[tunnelID]; !ok {
		return
	}

	delete(c.cache[tunnelID], sequence)
}

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
	go core.Tunnels.ScheduleClose(req.TunnelID)
	toImplantCache.DeleteTun(req.TunnelID)
	fromImplantCache.DeleteTun(req.TunnelID)

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
			rpcLog.Warnf("Error on stream recv %s", err)
			return err
		}
		tunnelLog.Debugf("Tunnel %d: From client %d byte(s)",
			fromClient.TunnelID, len(fromClient.Data))

		tunnel := core.Tunnels.Get(fromClient.TunnelID)
		if tunnel == nil {
			return core.ErrInvalidTunnelID
		}
		if tunnel.Client == nil {
			tunnel.Client = stream // Bind client to tunnel
			tunnelLog.Debugf("Binding client %v to tunnel id: %d", stream, tunnel.ID)
			tunnel.Client.Send(&sliverpb.TunnelData{
				TunnelID:  tunnel.ID,
				SessionID: tunnel.SessionID,
				Closed:    false,
			})

			go func() {

				for tunnelData := range tunnel.FromImplant {

					tunnelLog.Debugf("Tunnel %d: From implant %d byte(s), seq: %d ack: %d",
						tunnel.ID, len(tunnelData.Data), tunnelData.Sequence, tunnelData.Ack)

					// Remove tunnel data from send cache if Resend is not set
					if !tunnelData.Resend {

						index := tunnelData.Ack - 1
						for sendMsg, ok := toImplantCache.Get(tunnel.ID, index); ok; sendMsg, ok = toImplantCache.Get(tunnel.ID, index) {
							tunnelLog.Debugf("Tunnel %d: Removing ack: %d from send cache", tunnel.ID, sendMsg.Sequence)
							toImplantCache.DeleteSeq(tunnel.ID, index)
							index = index - 1
						}

						fromImplantCache.Add(tunnel.ID, tunnelData.Sequence, tunnelData)

						for recv, ok := fromImplantCache.Get(tunnel.ID, tunnel.FromImplantSequence); ok; recv, ok = fromImplantCache.Get(tunnel.ID, tunnel.FromImplantSequence) {
							tunnel.Client.Send(&sliverpb.TunnelData{
								TunnelID:  tunnel.ID,
								SessionID: tunnel.SessionID,
								Data:      recv.Data,
								Closed:    false,
							})
							fromImplantCache.DeleteSeq(tunnel.ID, tunnel.FromImplantSequence)
							tunnel.FromImplantSequence++
						}

					} else {

						origtunnelData, ok := toImplantCache.Get(tunnel.ID, tunnelData.Ack)
						if ok {
							tunnelLog.Debugf("Tunnel %d: Resending cached msg: %d", tunnel.ID, tunnelData.Ack)
							session := core.Sessions.Get(tunnel.SessionID)
							data, err := proto.Marshal(origtunnelData)
							if err != nil {
								// {{if .Config.Debug}}
								tunnelLog.Debugf("[shell] Failed to marshal protobuf %s", err)
								// {{end}}
							}
							session.Connection.Send <- &sliverpb.Envelope{
								Type: sliverpb.MsgTunnelData,
								Data: data,
							}
						} else {
							tunnelLog.Debugf("Tunnel %d: Requested msg not in send cache: %d", tunnel.ID, tunnelData.Ack)
						}

					}
				}
				tunnelLog.Debugf("Closing tunnel %d (To Client)", tunnel.ID)
				tunnel.Client.Send(&sliverpb.TunnelData{
					TunnelID:  tunnel.ID,
					SessionID: tunnel.SessionID,
					Closed:    true,
				})
			}()

			go func() {
				session := core.Sessions.Get(tunnel.SessionID)
				for data := range tunnel.ToImplant {
					tunnelLog.Debugf("Tunnel %d: To implant %d byte(s), seq: %d", tunnel.ID, len(data), tunnel.ToImplantSequence)
					tunnelData := sliverpb.TunnelData{
						Sequence:  tunnel.ToImplantSequence,
						TunnelID:  tunnel.ID,
						SessionID: tunnel.SessionID,
						Data:      data,
						Closed:    false,
					}
					// Add tunnel data to cache
					toImplantCache.Add(tunnel.ID, tunnelData.Sequence, &tunnelData)

					data, _ := proto.Marshal(&tunnelData)
					tunnel.ToImplantSequence++
					session.Connection.Send <- &sliverpb.Envelope{
						Type: sliverpb.MsgTunnelData,
						Data: data,
					}

				}
				tunnelLog.Debugf("Closing tunnel %d (To Implant) ...", tunnel.ID)
				data, _ := proto.Marshal(&sliverpb.TunnelData{
					Sequence:  tunnel.ToImplantSequence, // Shouldn't need to increment since this will close the tunnel
					TunnelID:  tunnel.ID,
					SessionID: tunnel.SessionID,
					Data:      make([]byte, 0),
					Closed:    true,
				})
				session.Connection.Send <- &sliverpb.Envelope{
					Type: sliverpb.MsgTunnelData,
					Data: data,
				}
			}()

		} else if tunnel.Client == stream {
			tunnelLog.Debugf("Tunnel %d: From client %d byte(s) to implant...",
				fromClient.TunnelID, len(fromClient.Data))
			tunnel.ToImplant <- fromClient.GetData()
		}
	}
	return nil
}
