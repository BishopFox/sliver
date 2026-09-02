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
	"errors"
	"io"
	"sync"
	"time"

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

type tunnelDataReceiveResult struct {
	data *sliverpb.TunnelData
	err  error
}

// receiveTunnelDataFrames keeps the handler responsive to relay-worker
// failures while the transport receive is blocked. The gRPC runtime cancels a
// blocked Recv after TunnelData returns, so this pump must not be joined by the
// handler's relay-worker barrier.
func receiveTunnelDataFrames(ctx context.Context, receiver rpcpb.SliverRPC_TunnelDataServer) <-chan tunnelDataReceiveResult {
	results := make(chan tunnelDataReceiveResult, 1)
	go func() {
		defer close(results)
		for {
			data, err := receiver.Recv()
			select {
			case results <- tunnelDataReceiveResult{data: data, err: err}:
			case <-ctx.Done():
				return
			}
			if err != nil {
				return
			}
		}
	}()
	return results
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
		// Guarantee one close-request grace period even when the tunnel was idle.
		// Only client-to-implant frames may extend it if this unary RPC overtakes
		// their shared stream; duplicates must not spawn more schedulers.
		if tunnel.ClaimToImplantClose() {
			go core.Tunnels.ScheduleCloseTunnelToImplant(tunnel)
		}
	}

	return &commonpb.Empty{}, nil
}

// TunnelData - Streams tunnel data back and forth from the client<->server<->implant
//
//nolint:gocyclo // This is the explicit bidirectional tunnel lifecycle state machine.
func (s *Server) TunnelData(stream rpcpb.SliverRPC_TunnelDataServer) (resultErr error) {
	ctx, cancel := context.WithCancel(stream.Context())
	var workers sync.WaitGroup
	var workerErr error
	var workerErrMutex sync.Mutex
	reportWorkerError := func(err error) {
		if err == nil {
			return
		}
		workerErrMutex.Lock()
		if workerErr == nil {
			workerErr = err
		}
		workerErrMutex.Unlock()
		cancel()
	}
	getWorkerError := func() error {
		workerErrMutex.Lock()
		defer workerErrMutex.Unlock()
		return workerErr
	}
	defer func() {
		cancel()
		core.Tunnels.CloseForClient(stream)
		workers.Wait()
		if resultErr == nil {
			resultErr = rpcError(getWorkerError())
		}
	}()

	var sendMutex sync.Mutex
	var sendErr error
	sendToClient := func(data *sliverpb.TunnelData) error {
		sendMutex.Lock()
		defer sendMutex.Unlock()
		if sendErr != nil {
			return sendErr
		}
		sendErr = stream.Send(data)
		return sendErr
	}

	receiveResults := receiveTunnelDataFrames(ctx, stream)
	for {
		var fromClient *sliverpb.TunnelData
		select {
		case <-ctx.Done():
			if err := getWorkerError(); err != nil {
				return rpcError(err)
			}
			return rpcError(ctx.Err())
		case received, ok := <-receiveResults:
			if !ok {
				if err := getWorkerError(); err != nil {
					return rpcError(err)
				}
				if err := ctx.Err(); err != nil {
					return rpcError(err)
				}
				return nil
			}
			if received.err == io.EOF {
				return nil
			}
			if received.err != nil {
				rpcLog.Warnf("Error on stream recv %s", received.err)
				return rpcError(received.err)
			}
			fromClient = received.data
		}
		if fromClient == nil {
			continue
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
		if fromClient.SessionID != tunnel.SessionID {
			// Tunnel identifiers are random, but they are not authorization. A
			// frame must name the exact owning session before it can reserve or
			// write to a tunnel generation.
			tunnelLog.Warnf("Dropping tunnel %d frame for mismatched session", fromClient.TunnelID)
			continue
		}
		if !tunnel.IsClient(stream) && !isCanonicalTunnelBindFrame(tunnel, fromClient) {
			// The first frame is only a bind control. Ignoring malformed bind
			// attempts leaves the legitimate stream able to claim the tunnel and
			// avoids turning a guessed ID into a tunnel-close primitive.
			tunnelLog.Warnf("Dropping malformed bind frame for tunnel %d", fromClient.TunnelID)
			continue
		}
		implantConnection := tunnel.ImplantConnection()
		if s.tunnelDataBeforeBindClient != nil {
			s.tunnelDataBeforeBindClient(tunnel, fromClient)
		}
		if tunnel.BindClient(stream) {
			tunnelLog.Debugf("Binding client %v to tunnel id: %d", stream, tunnel.ID)
			if err := sendToClient(&sliverpb.TunnelData{
				TunnelID:  tunnel.ID,
				SessionID: tunnel.SessionID,
				Closed:    false,
			}); err != nil {
				core.Tunnels.CloseIf(tunnel)
				_ = sendTunnelCloseToImplant(tunnel, core.DefaultImplantSendTimeout)
				return rpcError(err)
			}
			if s.tunnelDataAfterBindAcknowledgment != nil {
				s.tunnelDataAfterBindAcknowledgment(tunnel, fromClient)
			}
			if !tunnel.MarkClientBound(stream) {
				if tunnel.ClientBindLeaseExpired() {
					// Bind-lease expiry is published under the same lock as bind
					// acknowledgement. Detach the exact generation before sending
					// terminals even if the lease worker has not reached CloseIf yet.
					core.Tunnels.CloseIf(tunnel)
				}
				if tunnelIsClosed(tunnel) {
					// A close may win after the bind acknowledgement but before
					// the exact client generation is marked bound. This tunnel is
					// already terminal; notify this client without terminating the
					// shared stream and its unrelated tunnels.
					if err := notifyTunnelClosedToClient(tunnel, stream, sendToClient); err != nil {
						return rpcError(err)
					}
					_ = sendTunnelCloseToImplant(tunnel, core.DefaultImplantSendTimeout)
					continue
				}
				core.Tunnels.CloseIf(tunnel)
				_ = sendTunnelCloseToImplant(tunnel, core.DefaultImplantSendTimeout)
				return rpcError(core.ErrInvalidTunnelID)
			}

			workers.Add(1)
			go func() {
				defer workers.Done()
				// Every exit caused by a closed bound tunnel must publish the
				// terminal frame. In particular, Done can race a ready implant
				// frame: processing that frame then observes ErrTunnelClosed and
				// returns without taking the Done select branch.
				defer func() {
					reportWorkerError(notifyTunnelClosedToClient(tunnel, stream, sendToClient))
				}()
				for {
					var tunnelData *sliverpb.TunnelData
					select {
					case tunnelData = <-tunnel.FromImplant:
					case <-tunnel.Done():
						return
					}

					tunnelLog.Debugf("Tunnel %d: From implant %d byte(s), seq: %d ack: %d",
						tunnel.ID, len(tunnelData.Data), tunnelData.Sequence, tunnelData.Ack)

					if s.tunnelDataBeforeImplantControl != nil {
						s.tunnelDataBeforeImplantControl(tunnel, tunnelData)
					}
					if tunnelData.Resend {
						originalTunnelData, ok, resendErr := tunnel.ResendDataToImplant(tunnelData.Ack)
						if resendErr != nil {
							// A close may win after this worker receives a ready resend
							// control but before it reads the cached frame. The exact
							// tunnel is already terminal, so the implant connection is
							// still healthy.
							if errors.Is(resendErr, core.ErrTunnelClosed) {
								return
							}
							if implantConnection != nil {
								implantConnection.Close()
							}
							core.Tunnels.CloseIf(tunnel)
							return
						}
						if ok {
							tunnelLog.Debugf("Tunnel %d: Resending cached msg: %d", tunnel.ID, tunnelData.Ack)
							if implantConnection == nil {
								tunnelLog.Warnf("Tunnel %d: implant connection not found, dropping resend", tunnel.ID)
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
							if err := implantConnection.SendEnvelopeUntil(&sliverpb.Envelope{
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
						// A close may win after this worker receives a ready frame but
						// before it acknowledges that frame. The exact generation is
						// already terminal; do not turn that benign lifecycle race into
						// a failure of the otherwise healthy implant connection.
						if errors.Is(err, core.ErrTunnelClosed) {
							return
						}
						if implantConnection != nil {
							implantConnection.Close()
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
						reportWorkerError(err)
						return
					}
					if err := tunnel.CompleteDataFromImplantForward(tunnelData.Sequence); err != nil {
						// A concurrent exact close after a successful stream Send is
						// benign. Any other completion mismatch violates the tunnel's
						// ordered forwarding invariant and fails its creating transport.
						if errors.Is(err, core.ErrTunnelClosed) {
							return
						}
						if implantConnection != nil {
							implantConnection.Close()
						}
						core.Tunnels.CloseIf(tunnel)
						return
					}
				}
			}()

			workers.Add(1)
			go func() {
				defer workers.Done()
				for {
					var data []byte
					select {
					case data = <-tunnel.ToImplant:
					case <-tunnel.Done():
						tunnelLog.Debugf("Closing tunnel %d (To Implant) ...", tunnel.ID)
						_ = sendTunnelCloseToImplant(tunnel, core.DefaultImplantSendTimeout)
						return
					}
					tunnelLog.Debugf("Tunnel %d: To implant %d byte(s), seq: %d", tunnel.ID, len(data), tunnel.ToImplantSequence)
					if implantConnection == nil {
						tunnelLog.Warnf("Tunnel %d: implant connection not found, dropping data to implant", tunnel.ID)
						tunnel.CompleteDataToImplant()
						continue
					}
					if s.tunnelDataBeforeNextToImplant != nil {
						s.tunnelDataBeforeNextToImplant(tunnel, data)
					}
					err := forEachTunnelPayloadFrame(data, func(frame []byte) error {
						tunnelData, frameErr := tunnel.NextDataToImplant(frame)
						if frameErr != nil {
							return frameErr
						}
						encoded, frameErr := proto.Marshal(tunnelData)
						if frameErr != nil {
							return frameErr
						}
						envelope := &sliverpb.Envelope{
							Type: sliverpb.MsgTunnelData,
							Data: encoded,
						}
						if s.tunnelDataSendToImplant != nil {
							frameErr = s.tunnelDataSendToImplant(implantConnection, envelope, tunnel.Done(), core.DefaultImplantSendTimeout)
						} else {
							frameErr = implantConnection.SendEnvelopeUntil(envelope, tunnel.Done(), core.DefaultImplantSendTimeout)
						}
						if frameErr != nil {
							return frameErr
						}
						if s.tunnelDataAfterImplantSend != nil {
							s.tunnelDataAfterImplantSend(tunnel, tunnelData)
						}
						return tunnel.CompleteDataToImplantForward(tunnelData.Sequence)
					})
					tunnel.CompleteDataToImplant()
					if err != nil {
						core.Tunnels.CloseIf(tunnel)
						_ = sendTunnelCloseToImplant(tunnel, core.DefaultImplantSendTimeout)
						return
					}

				}
			}()

		} else if tunnelIsClosed(tunnel) || tunnel.ClientBindLeaseExpired() {
			// The exact tunnel may close after registry lookup but before the bind
			// attempt. A bind lease can likewise publish expiry just before its
			// cleanup worker detaches the generation. Neither lifecycle race may
			// terminate the shared stream and its unrelated tunnels.
			if tunnel.ClientBindLeaseExpired() {
				core.Tunnels.CloseIf(tunnel)
			}
			if err := notifyTunnelClosedToClient(tunnel, stream, sendToClient); err != nil {
				return rpcError(err)
			}
			_ = sendTunnelCloseToImplant(tunnel, core.DefaultImplantSendTimeout)
		} else if tunnel.IsClient(stream) {
			tunnelLog.Debugf("Tunnel %d: From client %d byte(s) to implant...",
				fromClient.TunnelID, len(fromClient.Data))
			if !tunnel.SendDataToImplant(fromClient.GetData()) {
				tunnelLog.Debugf("Tunnel %d closed before client data could be delivered", tunnel.ID)
			}
		}
	}
}

func isCanonicalTunnelBindFrame(tunnel *core.Tunnel, frame *sliverpb.TunnelData) bool {
	return tunnel != nil && frame != nil &&
		frame.TunnelID == tunnel.ID && frame.SessionID == tunnel.SessionID &&
		len(frame.Data) == 0 && !frame.Closed && frame.Sequence == 0 &&
		frame.Ack == 0 && !frame.Resend && !frame.CreateReverse && frame.Rportfwd == nil
}

func notifyTunnelClosedToClient(tunnel *core.Tunnel, client rpcpb.SliverRPC_TunnelDataServer, send func(*sliverpb.TunnelData) error) error {
	if tunnel == nil || client == nil || send == nil {
		return nil
	}
	select {
	case <-tunnel.Done():
		if !tunnel.ClaimClientTerminalDelivery(client) {
			return nil
		}
		tunnelLog.Debugf("Closing tunnel %d (To Client)", tunnel.ID)
		return send(&sliverpb.TunnelData{
			TunnelID:  tunnel.ID,
			SessionID: tunnel.SessionID,
			Closed:    true,
		})
	default:
		return nil
	}
}

func tunnelIsClosed(tunnel *core.Tunnel) bool {
	if tunnel == nil {
		return true
	}
	select {
	case <-tunnel.Done():
		return true
	default:
		return false
	}
}

// sendTunnelCloseToImplant delivers the one terminal that retires the remote
// generic tunnel. If an otherwise-live connection cannot accept it, failing the
// connection closed ensures implant cleanup cannot leave the target relay
// stranded.
func sendTunnelCloseToImplant(tunnel *core.Tunnel, timeout time.Duration) error {
	if tunnel == nil {
		return nil
	}
	connection := tunnel.ImplantConnection()
	if connection == nil {
		return nil
	}
	if !tunnel.ClaimImplantTerminalDelivery() {
		return nil
	}
	// Terminal publication joins any payload handed to the forwarding worker,
	// including the narrow interval after a successful transport enqueue but
	// before its contiguous prefix is recorded. Done cancels a blocked failed
	// send, while a successful send advances the terminal boundary first.
	tunnel.QuiesceDataToImplant()
	closeData, err := proto.Marshal(&sliverpb.TunnelData{
		TunnelID:  tunnel.ID,
		SessionID: tunnel.SessionID,
		Closed:    true,
		Sequence:  tunnel.ToImplantTerminalSequence(),
	})
	if err != nil {
		connection.Close()
		return err
	}
	err = connection.SendEnvelope(&sliverpb.Envelope{
		Type: sliverpb.MsgTunnelClose,
		Data: closeData,
	}, timeout)
	if err != nil && !errors.Is(err, core.ErrImplantConnectionClosed) {
		connection.Close()
	}
	return err
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
