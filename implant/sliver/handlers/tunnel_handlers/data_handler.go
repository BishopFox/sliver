package tunnel_handlers

/*
	Sliver Implant Framework
	Copyright (C) 2022  Bishop Fox

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
	"errors"
	"io"

	// {{if .Config.Debug}}
	"log"
	// {{end}}

	"github.com/bishopfox/sliver/implant/sliver/transports"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"google.golang.org/protobuf/proto"
)

func TunnelDataHandler(envelope *sliverpb.Envelope, connection *transports.Connection) {
	tunnelData := &sliverpb.TunnelData{}
	if err := proto.Unmarshal(envelope.Data, tunnelData); err != nil {
		return
	}
	tunnel := connection.Tunnel(tunnelData.TunnelID)
	if tunnel != nil {
		// Resend is a control frame in the legacy generic tunnel protocol, not
		// zero-byte stream data. Reverse tunnels never emit it; reliable C2
		// transports plus the bounded reorder actor handle delayed frames.
		if tunnelData.Resend {
			return
		}
		// {{if .Config.Debug}}
		log.Printf("[tunnel] Process tunnel %d (seq: %d)", tunnel.ID, tunnelData.Sequence)
		// {{end}}
		pending, err := tunnel.ProcessInbound(tunnelData.Sequence, tunnelData.Data, func(payload []byte) error {
			// {{if .Config.Debug}}
			log.Printf("[tunnel] Write %d bytes to tunnel %d", len(payload), tunnel.ID)
			// {{end}}
			written, writeErr := tunnel.Writer.Write(payload)
			if writeErr != nil {
				return writeErr
			}
			if written != len(payload) {
				return io.ErrShortWrite
			}
			return nil
		})
		if err != nil {
			if errors.Is(err, transports.ErrTunnelClosed) {
				return
			}
			protocolFailure := errors.Is(err, transports.ErrTunnelTerminalSequence) ||
				errors.Is(err, transports.ErrTunnelFrameTooLarge) ||
				errors.Is(err, transports.ErrTunnelSequenceWindow) ||
				errors.Is(err, transports.ErrTunnelPendingBytes)
			if protocolFailure {
				if closeTunnelRemote(connection, tunnel) {
					connection.Cleanup()
				}
			} else {
				closeTunnelLocal(connection, tunnel)
			}
			return
		}
		if tunnel.PeerCloseReady() {
			closeTunnelRemote(connection, tunnel)
			return
		}

		//If cache is building up it probably means a msg was lost and the server is currently hung waiting for it.
		//Send a Resend packet to have the msg resent from the cache
		if pending > 3 && !tunnel.IsReverse() {
			err := connection.QueueTunnelControl(tunnel, func(sequence uint64, ack uint64) (*sliverpb.Envelope, error) {
				data, marshalErr := proto.Marshal(&sliverpb.TunnelData{
					Sequence: sequence,
					Ack:      ack,
					Resend:   true,
					TunnelID: tunnel.ID,
					Data:     []byte{},
				})
				if marshalErr != nil {
					return nil, marshalErr
				}
				return &sliverpb.Envelope{Type: sliverpb.MsgTunnelData, Data: data}, nil
			})
			if err != nil {
				if !errors.Is(err, transports.ErrTunnelClosed) {
					// {{if .Config.Debug}}
					log.Printf("[shell] Failed to marshal protobuf %s", err)
					// {{end}}
				}
				closeTunnelLocal(connection, tunnel)
			} else if err == nil {
				// {{if .Config.Debug}}
				log.Printf("[tunnel] Requesting resend of tunnelData seq: %d", tunnel.ReadSequence())
				// {{end}}
			}
		}

	} else {
		// {{if .Config.Debug}}
		log.Printf("[tunnel] Received data for nil tunnel %d", tunnelData.TunnelID)
		log.Printf("[message just transfered] %v", tunnelData)
		// {{end}}
	}
}
