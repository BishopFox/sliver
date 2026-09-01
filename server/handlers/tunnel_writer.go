package handlers

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
	"time"

	// {{if .Config.Debug}}
	"log"
	// {{end}}
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/bishopfox/sliver/server/core"
	"github.com/bishopfox/sliver/server/core/rtunnels"
	"google.golang.org/protobuf/proto"
)

const reverseTunnelSendTimeout = 10 * time.Second

var (
	errReverseTunnelClosed    = errors.New("reverse tunnel closed")
	errReverseTunnelSendQueue = errors.New("reverse tunnel implant send queue timed out")
)

// tunnelWriter - Sends data back to the server based on data read()
// I know the reader/writer stuff is a little hard to keep track of
type tunnelWriter struct {
	tun  *rtunnels.RTunnel
	conn *core.ImplantConnection
}

func (tw tunnelWriter) Write(data []byte) (int, error) {
	n := len(data)
	err := tw.tun.QueueOutbound(func(sequence uint64) error {
		marshaled, err := proto.Marshal(&sliverpb.TunnelData{
			Sequence: sequence,
			Ack:      tw.tun.ReadSequence(),
			TunnelID: tw.tun.ID,
			Data:     data,
		})
		if err != nil {
			return err
		}
		// {{if .Config.Debug}}
		log.Printf("[tunnelWriter] Write %d bytes (write seq: %d) ack: %d", n, sequence, tw.tun.ReadSequence())
		// {{end}}
		return queueTunnelEnvelope(tw.conn, tw.tun, &sliverpb.Envelope{
			Type: sliverpb.MsgTunnelData,
			Data: marshaled,
		})
	})
	if err != nil {
		if errors.Is(err, rtunnels.ErrReverseTunnelClosed) {
			return 0, errReverseTunnelClosed
		}
		return 0, err
	}
	return n, nil
}

func queueTunnelEnvelope(connection *core.ImplantConnection, tunnel *rtunnels.RTunnel, envelope *sliverpb.Envelope) error {
	if connection == nil || tunnel == nil || envelope == nil {
		return errReverseTunnelClosed
	}
	err := connection.SendEnvelopeUntil(envelope, tunnel.Done(), reverseTunnelSendTimeout)
	if err == nil {
		return nil
	}
	if errors.Is(err, core.ErrImplantConnectionClosed) {
		return errReverseTunnelClosed
	}
	return errReverseTunnelSendQueue
}
