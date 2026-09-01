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
)

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
		go core.Tunnels.ScheduleCloseTunnel(tunnel)
	}

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
				core.Tunnels.CloseIf(tunnel)
				return rpcError(err)
			}
			if !tunnel.MarkClientBound(stream) {
				core.Tunnels.CloseIf(tunnel)
				return rpcError(core.ErrInvalidTunnelID)
			}

			go func() {
				for {
					var tunnelData *sliverpb.TunnelData
					select {
					case tunnelData = <-tunnel.FromImplant:
					case <-tunnel.Done():
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

					if tunnelData.Resend {
						originalTunnelData, ok, resendErr := tunnel.ResendDataToImplant(tunnelData.Ack)
						if resendErr != nil {
							if session := core.Sessions.Get(tunnel.SessionID); session != nil {
								session.Connection.Close()
							}
							core.Tunnels.CloseIf(tunnel)
							return
						}
						if ok {
							tunnelLog.Debugf("Tunnel %d: Resending cached msg: %d", tunnel.ID, tunnelData.Ack)
							session := core.Sessions.Get(tunnel.SessionID)
							if session == nil {
								tunnelLog.Warnf("Tunnel %d: session not found, dropping resend", tunnel.ID)
								continue
							}
							data, err := proto.Marshal(originalTunnelData)
							if err != nil {
								// {{if .Config.Debug}}
								tunnelLog.Debugf("[shell] Failed to marshal protobuf %s", err)
								// {{end}}
								core.Tunnels.CloseIf(tunnel)
								return
							}
							if err := session.Connection.SendEnvelopeUntil(&sliverpb.Envelope{
								Type: sliverpb.MsgTunnelData,
								Data: data,
							}, tunnel.Done(), core.DefaultImplantSendTimeout); err != nil {
								core.Tunnels.CloseIf(tunnel)
								return
							}
						} else {
							tunnelLog.Debugf("Tunnel %d: Requested msg not in send cache: %d", tunnel.ID, tunnelData.Ack)
						}

						continue
					}

					if err := tunnel.AcknowledgeDataToImplant(tunnelData.Ack); err != nil {
						if session := core.Sessions.Get(tunnel.SessionID); session != nil {
							session.Connection.Close()
						}
						core.Tunnels.CloseIf(tunnel)
						return
					}
					if err := sendToClient(&sliverpb.TunnelData{
						TunnelID:  tunnel.ID,
						SessionID: tunnel.SessionID,
						Data:      tunnelData.Data,
						Closed:    false,
					}); err != nil {
						core.Tunnels.CloseIf(tunnel)
						return
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
						tunnelLog.Debugf("Closing tunnel %d (To Implant) ...", tunnel.ID)
						if session != nil {
							closeData, _ := proto.Marshal(&sliverpb.TunnelData{
								TunnelID:  tunnel.ID,
								SessionID: tunnel.SessionID,
								Closed:    true,
							})
							_ = session.Connection.SendEnvelope(&sliverpb.Envelope{
								Type: sliverpb.MsgTunnelClose,
								Data: closeData,
							}, core.DefaultImplantSendTimeout)
						}
						return
					}
					tunnelLog.Debugf("Tunnel %d: To implant %d byte(s), seq: %d", tunnel.ID, len(data), tunnel.ToImplantSequence)
					if session == nil {
						tunnelLog.Warnf("Tunnel %d: session not found, dropping data to implant", tunnel.ID)
						continue
					}
					if err := forEachTunnelPayloadFrame(data, func(frame []byte) error {
						tunnelData, frameErr := tunnel.NextDataToImplant(frame)
						if frameErr != nil {
							return frameErr
						}
						encoded, frameErr := proto.Marshal(tunnelData)
						if frameErr != nil {
							return frameErr
						}
						return session.Connection.SendEnvelopeUntil(&sliverpb.Envelope{
							Type: sliverpb.MsgTunnelData,
							Data: encoded,
						}, tunnel.Done(), core.DefaultImplantSendTimeout)
					}); err != nil {
						core.Tunnels.CloseIf(tunnel)
						return
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

func forEachTunnelPayloadFrame(payload []byte, yield func([]byte) error) error {
	if len(payload) == 0 {
		return yield(nil)
	}
	for len(payload) > 0 {
		frameSize := min(len(payload), core.MaxTunnelFrameBytes)
		if err := yield(payload[:frameSize]); err != nil {
			return err
		}
		payload = payload[frameSize:]
	}
	return nil
}
