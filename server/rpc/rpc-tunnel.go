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
//
//	no way to associate the tunnel with the correct client, so the client must send
//	a zero-byte message over TunnelData to bind itself to the newly created tunnel.
func (s *Server) CreateTunnel(ctx context.Context, req *sliverpb.Tunnel) (*sliverpb.Tunnel, error) {
	session := core.Sessions.Get(req.SessionID)
	if session == nil {
		return nil, ErrInvalidSessionID
	}
	tunnel, err := core.Tunnels.Create(session.ID)
	if err != nil {
		return nil, rpcError(err)
	}
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
	if tunnel := core.Tunnels.Get(req.TunnelID); tunnel != nil {
		if tunnel.SessionID != req.SessionID {
			return nil, rpcError(core.ErrInvalidTunnelID)
		}
		// Guarantee a close-request grace period even when the tunnel was idle.
		// Client frames update the same activity timestamp if they arrive after
		// this unary RPC overtakes the shared stream.
		tunnel.Touch()
		go core.Tunnels.ScheduleClose(req.TunnelID)
	}
	toImplantCache.DeleteTun(req.TunnelID)
	fromImplantCache.DeleteTun(req.TunnelID)

	return &commonpb.Empty{}, nil
}

// TunnelData - Streams tunnel data back and forth from the client<->server<->implant
func (s *Server) TunnelData(stream rpcpb.SliverRPC_TunnelDataServer) error {
	defer core.Tunnels.CloseForClient(stream)
	var sendMutex sync.Mutex
	sendToClient := func(data *sliverpb.TunnelData) error {
		sendMutex.Lock()
		defer sendMutex.Unlock()
		return stream.Send(data)
	}

	for {
		fromClient, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			rpcLog.Warnf("Error on stream recv %s", err)
			return rpcError(err)
		}
		tunnelLog.Debugf("Tunnel %d: From client %d byte(s)",
			fromClient.TunnelID, len(fromClient.Data))

		tunnel := core.Tunnels.Get(fromClient.TunnelID)
		if tunnel == nil {
			// A CloseTunnel unary RPC can overtake a final frame on this shared
			// stream. A stale tunnel frame must not tear down every active tunnel.
			tunnelLog.Warnf("Dropping data for unknown tunnel %d", fromClient.TunnelID)
			continue
		}
		if tunnel.BindClient(stream) {
			tunnelLog.Debugf("Binding client %v to tunnel id: %d", stream, tunnel.ID)
			if err := sendToClient(&sliverpb.TunnelData{
				TunnelID:  tunnel.ID,
				SessionID: tunnel.SessionID,
				Closed:    false,
			}); err != nil {
				_ = core.Tunnels.Close(tunnel.ID)
				return rpcError(err)
			}
			if !tunnel.MarkClientBound(stream) {
				_ = core.Tunnels.Close(tunnel.ID)
				return rpcError(core.ErrInvalidTunnelID)
			}

			go func() {
				for {
					var tunnelData *sliverpb.TunnelData
					select {
					case tunnelData = <-tunnel.FromImplant:
					case <-tunnel.Done():
						toImplantCache.DeleteTun(tunnel.ID)
						fromImplantCache.DeleteTun(tunnel.ID)
						tunnelLog.Debugf("Closing tunnel %d (To Client)", tunnel.ID)
						_ = sendToClient(&sliverpb.TunnelData{
							TunnelID:  tunnel.ID,
							SessionID: tunnel.SessionID,
							Closed:    true,
						})
						return
					}

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
							if err := sendToClient(&sliverpb.TunnelData{
								TunnelID:  tunnel.ID,
								SessionID: tunnel.SessionID,
								Data:      recv.Data,
								Closed:    false,
							}); err != nil {
								_ = core.Tunnels.Close(tunnel.ID)
								return
							}
							fromImplantCache.DeleteSeq(tunnel.ID, tunnel.FromImplantSequence)
							tunnel.FromImplantSequence++
						}

					} else {

						origtunnelData, ok := toImplantCache.Get(tunnel.ID, tunnelData.Ack)
						if ok {
							tunnelLog.Debugf("Tunnel %d: Resending cached msg: %d", tunnel.ID, tunnelData.Ack)
							session := core.Sessions.Get(tunnel.SessionID)
							if session == nil {
								tunnelLog.Warnf("Tunnel %d: session not found, dropping resend", tunnel.ID)
								continue
							}
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
			}()

			go func() {
				session := core.Sessions.Get(tunnel.SessionID)
				for {
					var data []byte
					select {
					case data = <-tunnel.ToImplant:
					case <-tunnel.Done():
						toImplantCache.DeleteTun(tunnel.ID)
						fromImplantCache.DeleteTun(tunnel.ID)
						tunnelLog.Debugf("Closing tunnel %d (To Implant) ...", tunnel.ID)
						if session != nil {
							closeData, _ := proto.Marshal(&sliverpb.TunnelData{
								TunnelID:  tunnel.ID,
								SessionID: tunnel.SessionID,
								Closed:    true,
							})
							session.Connection.Send <- &sliverpb.Envelope{
								Type: sliverpb.MsgTunnelClose,
								Data: closeData,
							}
						}
						return
					}
					tunnelLog.Debugf("Tunnel %d: To implant %d byte(s), seq: %d", tunnel.ID, len(data), tunnel.ToImplantSequence)
					if session == nil {
						tunnelLog.Warnf("Tunnel %d: session not found, dropping data to implant", tunnel.ID)
						continue
					}
					tunnelData := sliverpb.TunnelData{
						Sequence:  tunnel.ToImplantSequence,
						TunnelID:  tunnel.ID,
						SessionID: tunnel.SessionID,
						Data:      data,
						Closed:    false,
					}
					// Add tunnel data to cache
					toImplantCache.Add(tunnel.ID, tunnelData.Sequence, &tunnelData)

					encoded, _ := proto.Marshal(&tunnelData)
					tunnel.ToImplantSequence++
					session.Connection.Send <- &sliverpb.Envelope{
						Type: sliverpb.MsgTunnelData,
						Data: encoded,
					}

				}
			}()

		} else if tunnel.IsClient(stream) {
			tunnelLog.Debugf("Tunnel %d: From client %d byte(s) to implant...",
				fromClient.TunnelID, len(fromClient.Data))
			if !tunnel.SendDataToImplant(fromClient.GetData()) {
				tunnelLog.Debugf("Tunnel %d closed before client data could be delivered", tunnel.ID)
			}
		}
	}
	return nil
}
